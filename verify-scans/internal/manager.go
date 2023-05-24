package internal

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

const (
	NonCompliantLabel = "ghas-non-compliant"
)

type Manager struct {
	Context context.Context

	AdminGitHubClient       *github.Client
	EMASSGithubClient       *github.Client
	VerifyScansGithubClient *github.Client

	Config       *Input
	Logger       *log.Entry
	GlobalLogger *log.Logger

	EMASSSystemIDs       []int64
	LatestCodeQLVersions []string
}

func (m *Manager) ProcessRepository(repo *github.Repository) {
	logger := m.GlobalLogger.WithField("repo", repo.GetName())
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
		logger.WithField("event", "skipped-ignored").Infof("Found .emass-repo-ignore file, skipping repository")
		return
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
		return
	}
	logger.Debugf("CodeQL Configuration File retrieved")

	logger.Infof("Retrieving eMASS configuration file")
	emassConfig, err := m.GetEMASSConfig(org, name, ".github/emass.json")
	if err != nil {
		logger.Errorf("failed to retrieve eMASS Configuration File, skipping repo: %v", err)
		return
	}
	if emassConfig == nil || emassConfig.SystemID == 0 || emassConfig.SystemName == "" || emassConfig.SystemOwnerName == "" || emassConfig.SystemOwnerEmail == "" {
		logger.WithField("event", "missing-configuration").Warnf(".github/emass.json not found, or missing/incorrect eMASS data")
		logger.WithField("event", "generating-email").Infof("Sending 'Error: GitHub Repository Not Mapped To eMASS System' email to OIS and system owner")
		body := GenerateMissingEMASSEmailBody(m.Config.MissingInfoEmailTemplate, repo.GetHTMLURL())
		err = m.SendEmail("", "Error: GitHub Repository Not Mapped To eMASS System", body)
		if err != nil {
			logger.Errorf("failed to send email, skipping repository: %v", err)
			return
		}
		logger.Debugf("Email sent")
		logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")

		issueBody := GenerateMissingEMASSIssueBody(m.Config.MissingInfoIssueTemplate, repo.GetHTMLURL())
		err = m.CreateIssue(org, name, "Error: GitHub Repository Not Mapped To eMASS System", issueBody, []string{NonCompliantLabel})
		if err != nil {
			logger.Errorf("failed to create issue, skipping repository: %v", err)
			return
		}
		logger.Debugf("Issue created")

		logger.Infof("eMASS configuration file missing or invalid, skipping repo")
		return
	}
	logger.Debugf("eMASS configuration file processed")

	logger.Infof("Retrieving supported CodeQL languages")
	expectedLanguages, err := m.ListExpectedCodeQLLanguages(org, name, codeqlConfig.ExcludedLanguages)
	if err != nil {
		logger.Errorf("failed to retrieve supported CodeQL languages, skipping repo: %v", err)
		return
	}
	logger.Debugf("Supported CodeQL languages retrieved")

	logger.Info("Retrieving recent CodeQL analyses")
	recentAnalyses, err := m.ListCodeQLAnalyses(org, name, defaultBranch, expectedLanguages)
	if err != nil {
		logger.Errorf("failed to retrieve recent CodeQL analyses, skipping repo: %v", err)
		return
	}
	logger.Debugf("Recent CodeQL analyses retrieved")

	if len(recentAnalyses.Languages) > 0 {
		logger.Infof("Analyses found, validating 'eMASS-Promotion' app is installed on repository")
		installed, err := m.EMASSAppInstalled(org, name)
		if err != nil {
			logger.Errorf("failed to validate 'eMASS-Promotion' app is installed on repository, skipping repo: %v", err)
			return
		}
		if !installed {
			logger.Infof("'eMASS-Promotion' app not installed, installing now")
			err = m.InstallEMASSApp(repo.GetID())
			if err != nil {
				logger.Errorf("failed to install 'eMASS-Promotion' app, skipping repo: %v", err)
				return
			}
		}
		logger.Debugf("'eMASS-Promotion' app installed")
	}

	logger.Info("Validating scans performed with latest CodeQL version")
	if len(recentAnalyses.Versions) > 0 {
		for _, version := range recentAnalyses.Versions {
			if !Includes(m.LatestCodeQLVersions, version) {
				logger.WithField("event", "out-of-date-cli").Warnf("Outdated CodeQL CLI version found: %s", version)
				logger.WithField("event", "generating-email").Warnf("Sending 'GitHub Repository Code Scanning Software Is Out Of Date' email to OIS and System Owner")
				body := GenerateOutOfComplianceCLIEmailBody(m.Config.OutOfComplianceCLIEmailTemplate, name, repo.GetHTMLURL(), version)
				err = m.SendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Software Is Out Of Date", body)
				if err != nil {
					logger.Errorf("failed to send email, skipping repository: %v", err)
					return
				}
				logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
				logger.Debugf("Email sent")

				err = m.CreateIssue(org, name, "GitHub Repository Code Scanning Software Is Out Of Date", body, []string{NonCompliantLabel})
				if err != nil {
					logger.Errorf("failed to create issue, skipping repository: %v", err)
					return
				}
				logger.Debugf("Issue created")
			}
		}
	}
	logger.Debugf("CodeQL CLI versions validated")

	logger.Infof("Retrieving missing CodeQL languages")
	missingLanguages := CalculateMissingLanguages(expectedLanguages, recentAnalyses.Languages)
	logger.Debugf("Missing CodeQL languages retrieved: %v", missingLanguages)

	logger.Infof("Retrieving support CodeQL database languags")
	databaseLanguages, err := m.ListCodeQLDatabaseLanguages(org, name)
	if err != nil {
		logger.Errorf("failed to retrieve supported CodeQL database languages, skipping repo: %v", err)
		return
	}
	logger.Debugf("Supported CodeQL database languages retrieved")

	logger.Infof("Calculating missing CodeQL database languages")
	missingDatabaseLanguages := CalculateMissingLanguages(expectedLanguages, databaseLanguages)
	logger.Debugf("Missing CodeQL database languages calculated: %v", missingDatabaseLanguages)

	if len(missingLanguages) == 0 && len(missingDatabaseLanguages) == 0 {
		logger.Infof("No missing analyses or databases found")
		logger.WithField("event", "successfully-processed").Infof("Successfully processed repository")
		return
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
		return
	}

	logger.WithField("event", "missing-data").Warnf("Missing analyses or databases identified: %s", string(missingDataJSON))
	logger.WithField("event", "generating-email").Warnf("Sending 'GitHub Repository Code Scanning Not Enabled' email to OIS and system owner")
	missingLanguages = Unique(missingData.MissingAnalyses, missingData.MissingDatabases)
	body := GenerateNonCompliantEmailBody(m.Config.NonCompliantEmailTemplate, repo.GetName(), emassConfig.SystemName, emassConfig.SystemID, missingLanguages)
	err = m.SendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Not Enabled", body)
	if err != nil {
		logger.Errorf("failed to send email, skipping repository: %v", err)
		return
	}
	logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
	logger.Debugf("Email sent")
}
