package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/department-of-veterans-affairs/codeql-tools/verify-scans/internal"
	log "github.com/sirupsen/logrus"
)

const (
	NonCompliantLabel = "ghas-non-compliant"
)

func init() {
	log.SetLevel(log.InfoLevel)
	debug := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG"))) == "true"
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	config := internal.ParseInput()

	rootLogger := log.WithField("app", "verify-scans")
	adminClient := utils.NewGitHubClient(config.AdminToken)

	rootLogger.Infof("Creating eMASS Promotion GitHub App client")
	emassClient, err := utils.NewGitHubAppClient(config.EMASSPromotionAppID, config.EMASSPromotionPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create eMASS Promotion GitHub App client: %v", err)
	}
	rootLogger.Debugf("eMASS Promotion GitHub App client created")

	rootLogger.Infof("Creating Verify Scans GitHub App Installation client")
	verifyScansClient, err := utils.NewGitHubInstallationClient(config.VerifyScansAppID, config.VerifyScansInstallationID, config.VerifyScansPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	rootLogger.Debugf("Verify Scans GitHub App client created")

	m := &internal.Manager{
		Context: context.Background(),

		AdminGitHubClient:       adminClient,
		EMASSGithubClient:       emassClient,
		VerifyScansGithubClient: verifyScansClient,

		Config: config,
	}

	rootLogger.Infof("Retrieving repositories")
	repos, err := m.ListRepos()
	if err != nil {
		rootLogger.Fatalf("failed to list repositories: %v", err)
	}
	rootLogger.Debugf("Retrieved %d repositories", len(repos))

	rootLogger.Infof("Retrieving latest CodeQL versions")
	latestCodeQLVersions, err := m.GetLatestCodeQLVersions()
	if err != nil {
		rootLogger.Fatalf("failed to get latest CodeQL versions: %v", err)
	}
	rootLogger.Debugf("Retrieved latest CodeQL versions")

	rootLogger.Infof("Retrieving eMASS system list")
	emassSystemIDs, err := m.GetEMASSSystemList(m.Config.Org, m.Config.EMASSSystemListRepo, m.Config.EMASSSystemListPath)
	if err != nil {
		rootLogger.Fatalf("failed to get eMASS system list: %v", err)
	}
	rootLogger.Debugf("Retrieved %d eMASS system IDs", len(emassSystemIDs))

	for _, repo := range repos {
		logger := rootLogger.WithField("repo", repo.GetName())
		m.Logger = logger

		org := repo.GetOwner().GetLogin()
		name := repo.GetName()
		defaultBranch := repo.GetDefaultBranch()

		logger.Info("Checking if repository is ignored")
		repoIgnored, err := m.FileExists(org, name, ".github/.emass-repo-ignore")
		if err != nil {
			logger.Fatalf("failed to check if repository is ignored: %v", err)
		}
		if repoIgnored {
			logger.Infof("[skipped-ignore] Repository is ignored, skipping")
			continue
		}

		logger.Infof("Retrieving open '%s' issues", NonCompliantLabel)
		issues, err := m.ListOpenIssues(org, name, NonCompliantLabel)
		if err != nil {
			logger.Warnf("Failed to retrieve open issues, skipping closing issues: %v", err)
		} else {
			logger.Infof("Closing %d open issues", len(issues))
			m.CloseIssues(org, name, issues)
		}
		logger.Debugf("Open issues retrieved")

		logger.Infof("Retrieving CodeQL Configuration File")
		codeqlConfig, err := m.GetCodeQLConfig(org, name, defaultBranch)
		if err != nil {
			logger.Errorf("failed to retrieve CodeQL Configuration File, skipping repo: %v", err)
			continue
		}
		logger.Debugf("CodeQL Configuration File retrieved")

		logger.Infof("Retrieving eMASS configuration file")
		emassConfig, err := m.GetEMASSConfig(org, name, ".github/emass.json")
		if err != nil {
			logger.Errorf("failed to retrieve eMASS Configuration File, skipping repo: %v", err)
			continue
		}
		if emassConfig == nil {
			logger.WithField("event", "missing-configuration").Warnf(".github/emass.json not found")
		}
		if emassConfig.SystemID == 0 {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System ID not found")
		}
		if emassConfig.SystemName == "" {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System Name not found")
		}
		if emassConfig.SystemOwnerName == "" {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System Owner not found")
		}
		if emassConfig.SystemOwnerEmail == "" || !strings.Contains(emassConfig.SystemOwnerEmail, "@") {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System Owner Email not found or invalid")
		}
		if emassConfig != nil && emassConfig.SystemID != 0 && !internal.IncludesInt64(emassSystemIDs, emassConfig.SystemID) {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System ID not found in eMASS system list")
		}
		if emassConfig == nil || emassConfig.SystemID == 0 || emassConfig.SystemName == "" || emassConfig.SystemOwnerName == "" || emassConfig.SystemOwnerEmail == "" {
			logger.WithField("event", "generating-email").Warnf("Sending 'Error: GitHub Repository Not Mapped To eMASS System' email to OIS and system owner")
			body := internal.GenerateMissingEMASSEmailBody(config.MissingInfoEmailTemplate, repo.GetHTMLURL())
			err = m.SendEmail("", "Error: GitHub Repository Not Mapped To eMASS System", body)
			if err != nil {
				logger.Errorf("failed to send email, skipping repository: %v", err)
				continue
			}
			logger.Debugf("Email sent")
			logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")

			issueBody := internal.GenerateMissingEMASSIssueBody(config.MissingInfoIssueTemplate, repo.GetHTMLURL())
			err = m.CreateIssue(org, name, "Error: GitHub Repository Not Mapped To eMASS System", issueBody, []string{NonCompliantLabel})
			if err != nil {
				logger.Errorf("failed to create issue, skipping repository: %v", err)
				continue
			}
			logger.Debugf("Issue created")

			logger.Infof("eMASS configuration file missing or invalid, skipping repo")
			continue
		}
		logger.Debugf("eMASS configuration file processed")

		logger.Infof("Retrieving supported CodeQL languages")
		expectedLanguages, err := m.ListExpectedCodeQLLanguages(org, name, codeqlConfig.ExcludedLanguages)
		if err != nil {
			logger.Errorf("failed to retrieve supported CodeQL languages, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Supported CodeQL languages retrieved")

		logger.Info("Retrieving recent CodeQL analyses")
		recentAnalyses, err := m.ListCodeQLAnalyses(org, name, defaultBranch, expectedLanguages)
		if err != nil {
			logger.Errorf("failed to retrieve recent CodeQL analyses, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Recent CodeQL analyses retrieved")

		if len(recentAnalyses.Languages) > 0 {
			logger.Infof("Analyses found, validating 'eMASS-Promotion' app is installed on repository")
			installed, err := m.EMASSAppInstalled(org, name)
			if err != nil {
				logger.Errorf("failed to validate 'eMASS-Promotion' app is installed on repository, skipping repo: %v", err)
				continue
			}
			if !installed {
				logger.Infof("'eMASS-Promotion' app not installed, installing now")
				err = m.InstallEMASSApp(repo.GetID())
				if err != nil {
					logger.Errorf("failed to install 'eMASS-Promotion' app, skipping repo: %v", err)
					continue
				}
			}
			logger.Debugf("'eMASS-Promotion' app installed")
		}

		logger.Info("Validating scans performed with latest CodeQL version")
		if len(recentAnalyses.Versions) > 0 {
			for _, version := range recentAnalyses.Versions {
				if !internal.Includes(latestCodeQLVersions, version) {
					logger.WithField("event", "out-of-date-cli").Warnf("Outdated CodeQL CLI version found: %s", version)
					logger.WithField("event", "generating-email").Warnf("Sending 'GitHub Repository Code Scanning Software Is Out Of Date' email to OIS and System Owner")
					body := internal.GenerateOutOfComplianceCLIEmailBody(m.Config.OutOfComplianceCLIEmailTemplate, name, repo.GetHTMLURL(), version)
					err = m.SendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Software Is Out Of Date", body)
					if err != nil {
						logger.Errorf("failed to send email, skipping repository: %v", err)
						continue
					}
					logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
					logger.Debugf("Email sent")

					err = m.CreateIssue(org, name, "GitHub Repository Code Scanning Software Is Out Of Date", body, []string{NonCompliantLabel})
					if err != nil {
						logger.Errorf("failed to create issue, skipping repository: %v", err)
						continue
					}
					logger.Debugf("Issue created")
				}
			}
		}
		logger.Debugf("CodeQL CLI versions validated")

		logger.Infof("Retrieving missing CodeQL languages")
		missingLanguages := internal.CalculateMissingLanguages(expectedLanguages, recentAnalyses.Languages)
		logger.Debugf("Missing CodeQL languages retrieved: %v", missingLanguages)

		logger.Infof("Retrieving support CodeQL database languags")
		databaseLanguages, err := m.ListCodeQLDatabaseLanguages(org, name)
		if err != nil {
			logger.Errorf("failed to retrieve supported CodeQL database languages, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Supported CodeQL database languages retrieved")

		logger.Infof("Calculating missing CodeQL database languages")
		missingDatabaseLanguages := internal.CalculateMissingLanguages(expectedLanguages, databaseLanguages)
		logger.Debugf("Missing CodeQL database languages calculated: %v", missingDatabaseLanguages)

		if len(missingLanguages) == 0 && len(missingDatabaseLanguages) == 0 {
			logger.Infof("No missing analyses or databases found")
			logger.WithField("event", "successfully-processed").Infof("Successfully processed repository")
			continue
		}

		var missingData struct {
			MissingAnalyses  []string `json:"missing_analyses"`
			MissingDatabases []string `json:"missing_databases"`
		}
		missingData.MissingAnalyses = missingLanguages
		if missingLanguages == nil {
			missingData.MissingAnalyses = []string{}
		}
		missingData.MissingDatabases = missingDatabaseLanguages
		if missingDatabaseLanguages == nil {
			missingData.MissingDatabases = []string{}
		}

		missingDataJSON, err := json.Marshal(missingData)
		if err != nil {
			logger.Errorf("failed to marshal missing data, skipping repository: %v", err)
			continue
		}

		logger.WithField("event", "missing-data").Warnf("Missing analyses or databases identified: %s", string(missingDataJSON))
		logger.WithField("event", "generating-email").Warnf("Sending 'GitHub Repository Code Scanning Not Enabled' email to OIS and system owner")
		missingLanguages = internal.Unique(missingData.MissingAnalyses, missingData.MissingDatabases)
		body := internal.GenerateNonCompliantEmailBody(m.Config.NonCompliantEmailTemplate, repo.GetName(), emassConfig.SystemName, emassConfig.SystemID, missingLanguages)
		err = m.SendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Not Enabled", body)
		if err != nil {
			logger.Errorf("failed to send email, skipping repository: %v", err)
			continue
		}
		logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
		logger.Debugf("Email sent")
	}

}
