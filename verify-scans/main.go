package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

const (
	NonCompliantLabel = "ghas-non-compliant"
)

var (
	DisableNotifications = os.Getenv("DRY_RUN") == "true"
)

type manager struct {
	ctx context.Context

	adminGitHubClient       *github.Client
	emassGithubClient       *github.Client
	verifyScansGithubClient *github.Client

	config *input
	logger *log.Entry
}

func main() {
	config := parseInput()

	rootLogger := log.WithField("app", "verify-scans")
	adminClient := utils.NewGitHubClient(config.adminToken)

	rootLogger.Infof("Creating eMASS Promotion GitHub App client")
	emassClient, err := utils.NewGitHubAppClient(config.emassPromotionAppID, config.emassPromotionPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create eMASS Promotion GitHub App client: %v", err)
	}
	rootLogger.Debugf("eMASS Promotion GitHub App client created")

	rootLogger.Infof("Creating Verify Scans GitHub App Installation client")
	verifyScansClient, err := utils.NewGitHubInstallationClient(config.verifyScansAppID, config.verifyScansInstallationID, config.verifyScansPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	rootLogger.Debugf("Verify Scans GitHub App client created")

	m := &manager{
		ctx: context.Background(),

		adminGitHubClient:       adminClient,
		emassGithubClient:       emassClient,
		verifyScansGithubClient: verifyScansClient,

		config: config,
	}

	rootLogger.Infof("Retrieving repositories")
	repos, err := m.listRepos()
	if err != nil {
		rootLogger.Fatalf("failed to list repositories: %v", err)
	}
	rootLogger.Debugf("Retrieved %d repositories", len(repos))

	rootLogger.Infof("Retrieving latest CodeQL versions")
	latestCodeQLVersions, err := m.getLatestCodeQLVersions()
	if err != nil {
		rootLogger.Fatalf("failed to get latest CodeQL versions: %v", err)
	}
	rootLogger.Debugf("Retrieved latest CodeQL versions")

	rootLogger.Infof("Retrieving eMASS system list")
	emassSystemIDs, err := m.getEMASSSystemList(m.config.org, m.config.emassSystemListRepo, m.config.emassSystemListPath)
	if err != nil {
		rootLogger.Fatalf("failed to get eMASS system list: %v", err)
	}
	rootLogger.Debugf("Retrieved %d eMASS system IDs", len(emassSystemIDs))

	for _, repo := range repos {
		logger := rootLogger.WithField("repo", repo.GetName())
		m.logger = logger

		org := repo.GetOwner().GetLogin()
		name := repo.GetName()
		defaultBranch := repo.GetDefaultBranch()

		logger.Info("Checking if repository is ignored")
		repoIgnored, err := m.fileExists(org, name, ".github/.emass-repo-ignore")
		if err != nil {
			logger.Fatalf("failed to check if repository is ignored: %v", err)
		}

		if repoIgnored {
			logger.Infof("[skipped-ignore] Repository is ignored, skipping")
			continue
		}

		logger.Infof("Retrieving open '%s' issues", NonCompliantLabel)
		issues, err := m.listOpenIssues(org, name, NonCompliantLabel)
		if err != nil {
			logger.Warnf("Failed to retrieve open issues, skipping closing issues: %v", err)
		} else {
			logger.Infof("Closing %d open issues", len(issues))
			m.closeIssues(org, name, issues)
		}
		logger.Debugf("Open issues retrieved")

		logger.Infof("Retrieving CodeQL Configuration File")
		codeqlConfig, err := m.getCodeQLConfig(org, name, defaultBranch)
		if err != nil {
			logger.Errorf("failed to retrieve CodeQL Configuration File, skipping repo: %v", err)
			continue
		}
		logger.Debugf("CodeQL Configuration File retrieved")

		logger.Infof("Retrieving eMASS configuration file")
		emassConfig, err := m.getEMASSConfig(org, name, ".github/emass.json")
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
		if emassConfig != nil && !includesInt64(emassSystemIDs, emassConfig.SystemID) {
			logger.WithField("event", "missing-configuration").Warnf("eMASS System ID not found in eMASS system list")
		}
		if emassConfig == nil || emassConfig.SystemID == 0 || emassConfig.SystemName == "" || emassConfig.SystemOwnerName == "" || emassConfig.SystemOwnerEmail == "" {
			logger.WithField("event", "generating-email").Warnf("Sending 'Error: GitHub Repository Not Mapped To eMASS System' email to OIS and system owner")
			body := generateMissingEMASSEmailBody(config.missingInfoEmailTemplate, repo.GetHTMLURL())
			err = m.sendEmail("", "Error: GitHub Repository Not Mapped To eMASS System", body)
			if err != nil {
				logger.Errorf("failed to send email, skipping repository: %v", err)
				continue
			}
			logger.Debugf("Email sent")
			logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")

			issueBody := generateMissingEMASSIssueBody(config.missingInfoIssueTemplate, repo.GetHTMLURL())
			err = m.createIssue(org, name, "Error: GitHub Repository Not Mapped To eMASS System", issueBody, []string{NonCompliantLabel})
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
		expectedLanguages, err := m.listExpectedCodeQLLanguages(org, name, codeqlConfig.ExcludedLanguages)
		if err != nil {
			logger.Errorf("failed to retrieve supported CodeQL languages, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Supported CodeQL languages retrieved")

		logger.Info("Retrieving recent CodeQL analyses")
		recentAnalyses, err := m.listCodeQLAnalyses(org, name, defaultBranch, expectedLanguages)
		if err != nil {
			logger.Errorf("failed to retrieve recent CodeQL analyses, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Recent CodeQL analyses retrieved")

		if len(recentAnalyses.Languages) > 0 {
			logger.Infof("Analyses found, validating 'eMASS-Promotion' app is installed on repository")
			installed, err := m.eMASSAppInstalled(org, name)
			if err != nil {
				logger.Errorf("failed to validate 'eMASS-Promotion' app is installed on repository, skipping repo: %v", err)
				continue
			}
			if !installed {
				logger.Infof("'eMASS-Promotion' app not installed, installing now")
				err = m.installEMASSApp(repo.GetID())
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
				if !includes(latestCodeQLVersions, version) {
					logger.WithField("event", "out-of-date-cli").Warnf("Outdated CodeQL CLI version found: %s", version)
					logger.WithField("event", "generating-email").Warnf("Sending 'GitHub Repository Code Scanning Software Is Out Of Date' email to OIS and System Owner")
					body := generateOutOfComplianceCLIEmailBody(m.config.outOfComplianceCLIEmailTemplate, name, repo.GetHTMLURL(), version)
					err = m.sendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Software Is Out Of Date", body)
					if err != nil {
						logger.Errorf("failed to send email, skipping repository: %v", err)
						continue
					}
					logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
					logger.Debugf("Email sent")

					err = m.createIssue(org, name, "GitHub Repository Code Scanning Software Is Out Of Date", body, []string{NonCompliantLabel})
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
		missingLanguages := m.calculateMissingLanguages(expectedLanguages, recentAnalyses.Languages)
		logger.Debugf("Missing CodeQL languages retrieved: %v", missingLanguages)

		logger.Infof("Retrieving support CodeQL database languags")
		databaseLanguages, err := m.listCodeQLDatabaseLanguages(org, name)
		if err != nil {
			logger.Errorf("failed to retrieve supported CodeQL database languages, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Supported CodeQL database languages retrieved")

		logger.Infof("Calculating missing CodeQL database languages")
		missingDatabaseLanguages := m.calculateMissingLanguages(expectedLanguages, databaseLanguages)
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
		missingLanguages = unique(missingData.MissingAnalyses, missingData.MissingDatabases)
		body := generateNonCompliantEmailBody(m.config.nonCompliantEmailTemplate, repo.GetName(), emassConfig.SystemName, emassConfig.SystemID, missingLanguages)
		err = m.sendEmail(emassConfig.SystemOwnerEmail, "GitHub Repository Code Scanning Not Enabled", body)
		if err != nil {
			logger.Errorf("failed to send email, skipping repository: %v", err)
			continue
		}
		logger.WithField("event", "system-owner-notified").Infof("Sent email to system owner")
		logger.Debugf("Email sent")
	}

}

func (m *manager) listRepos() ([]*github.Repository, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var repos []*github.Repository
	for {
		installations, resp, err := m.verifyScansGithubClient.Apps.ListRepos(m.ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %v", err)
		}
		repos = append(repos, installations.Repositories...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repos, nil
}

func (m *manager) getEMASSConfig(owner, repo, path string) (*eMASSConfig, error) {
	content, _, resp, err := m.verifyScansGithubClient.Repositories.GetContents(m.ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get file: %v", err)
	}

	decodedContent, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %v", err)
	}

	var config eMASSConfig
	err = json.Unmarshal([]byte(decodedContent), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file content: %v", err)
	}

	return &config, nil
}

func (m *manager) getCodeQLConfig(owner, repo, path string) (*codeQLConfig, error) {
	content, _, resp, err := m.verifyScansGithubClient.Repositories.GetContents(m.ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return &codeQLConfig{
				BuildCommands:     map[string]string{},
				ExcludedLanguages: []string{},
			}, nil
		}

		return nil, fmt.Errorf("failed to get file: %v", err)
	}

	decodedContent, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %v", err)
	}

	var config codeQLConfig
	err = json.Unmarshal([]byte(decodedContent), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file content: %v", err)
	}

	return &config, nil
}

func (m *manager) getEMASSSystemList(owner, repo, path string) ([]int64, error) {
	content, _, resp, err := m.adminGitHubClient.Repositories.GetContents(m.ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("file not found")
		}

		return nil, fmt.Errorf("failed to get file: %v", err)
	}

	decodedContent, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %v", err)
	}

	var ids []int64
	trimmedContent := strings.TrimSpace(decodedContent)
	lines := strings.Split(trimmedContent, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if !strings.Contains(trimmedLine, "#") {
			id, err := strconv.ParseInt(trimmedLine, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse system ID: %v", err)
			}
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func (m *manager) fileExists(owner, repo, path string) (bool, error) {
	_, _, resp, err := m.verifyScansGithubClient.Repositories.GetContents(m.ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get file: %v", err)
	}

	return true, nil
}

func (m *manager) calculateMissingLanguages(expectedLanguages, actualLanguages []string) []string {
	var missingLanguages []string
	for _, language := range expectedLanguages {
		if !includes(actualLanguages, language) {
			missingLanguages = append(missingLanguages, language)
		}
	}

	return missingLanguages
}

func (m *manager) listExpectedCodeQLLanguages(owner, repo string, ignoredLanguages []string) ([]string, error) {
	languages, _, err := m.verifyScansGithubClient.Repositories.ListLanguages(m.ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list languages: %v", err)
	}

	var supportedLanguages []string
	mappedLanguages := mapLanguages(languages)
	for _, language := range mappedLanguages {
		if !includes(ignoredLanguages, language) {
			if includes(SupportedCodeQLLanguages, language) {
				supportedLanguages = append(supportedLanguages, language)
			}
		}
	}

	return supportedLanguages, nil
}

func mapLanguages(languages map[string]int) []string {
	mappedLanauges := make([]string, len(languages))
	for language := range languages {
		switch language {
		case "kotlin":
			mappedLanauges = append(mappedLanauges, "java")
		default:
			mappedLanauges = append(mappedLanauges, strings.ToLower(language))
		}
	}

	return mappedLanauges
}

func includes(a []string, s string) bool {
	for _, value := range a {
		if value == s {
			return true
		}
	}

	return false
}

func includesInt64(a []int64, s int64) bool {
	for _, value := range a {
		if value == s {
			return true
		}
	}

	return false
}

func unique(a, b []string) []string {
	var unique []string
	for _, value := range a {
		if !includes(b, value) {
			unique = append(unique, value)
		}
	}

	return unique
}

func (m *manager) listCodeQLDatabaseLanguages(owner, repo string) ([]string, error) {
	databaseAPIEndpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/code-scanning/codeql/databases", owner, repo)
	apiURL, err := url.Parse(databaseAPIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %v", err)
	}

	var databases []codeQLDatabase
	request := &http.Request{
		Method: http.MethodGet,
		URL:    apiURL,
	}
	response, err := m.verifyScansGithubClient.Do(m.ctx, request, &databases)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get databases: %v", err)
	}

	var languages []string
	for _, database := range databases {
		if m.isDateInRange(database.CreatedAt) {
			languages = append(languages, database.Language)
		}
	}

	return languages, nil
}

func (m *manager) isDateInRange(createdAt time.Time) bool {
	currentDate := time.Now()
	diff := currentDate.Sub(createdAt)
	diffDays := int(diff.Hours() / 24)

	return diffDays <= m.config.daysToScan
}

func (m *manager) listCodeQLAnalyses(owner, repo, branch string, requiredLanguages []string) (*analyses, error) {
	page := 0
	results := &analyses{}
	endpoint := "https://api.github.com/repos/%s/%s/code-scanning/analyses?per_page=100&page=%d"
	for {
		page++
		analysesAPIEndpoint := fmt.Sprintf(endpoint, owner, repo, page)
		apiURL, err := url.Parse(analysesAPIEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url: %v", err)
		}

		requestBody := &analysisRequest{
			ToolName:  "CodeQL",
			Ref:       fmt.Sprintf("refs/heads/%s", branch),
			Direction: "desc",
			Sort:      "created",
		}
		requestJSON, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}

		var analysesResults []analysisResult
		request := &http.Request{
			Method: http.MethodGet,
			URL:    apiURL,
			Body:   io.NopCloser(bytes.NewReader(requestJSON)),
		}
		response, err := m.verifyScansGithubClient.Do(m.ctx, request, &analysesResults)
		if err != nil {
			if response.StatusCode == http.StatusNotFound {
				return &analyses{
					Languages: []string{},
					Versions:  []string{},
				}, nil
			}
			return nil, fmt.Errorf("failed to make request: %v", err)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get databases: %v", err)
		}

		if len(analysesResults) == 0 {
			break
		}

		done := false
		for _, analysis := range analysesResults {
			if !m.isDateInRange(analysis.CreatedAt) {
				done = true
			}
			if strings.HasPrefix(analysis.Category, "ois-") {
				language := strings.TrimPrefix(analysis.Category, "ois-")
				analysis.Language = strings.ToLower(language)
				if !includes(results.Languages, analysis.Language) {
					results.Languages = append(results.Languages, analysis.Language)
					results.Versions = append(results.Versions, analysis.Tool.Version)
				}
				complete := allRequiredAnalysesFound(results.Languages, requiredLanguages)
				if complete {
					m.logger.Infof("found all required analyses, stopping search")
					return results, nil
				}
			}
		}

		if done {
			break
		}

	}

	return results, nil
}

func allRequiredAnalysesFound(languages, requiredLanguages []string) bool {
	for _, language := range requiredLanguages {
		if !includes(languages, language) {
			return false
		}
	}

	return true
}

func filterLatestAnalysisPerLanguage(analyses *[]analysisResult, languages []string) {
	var latestAnalyses []analysisResult
	for _, language := range languages {
		for _, analysis := range *analyses {
			if analysis.Language == language {
				latestAnalyses = append(latestAnalyses, analysis)
				break
			}
		}
	}
}

func (m *manager) eMASSAppInstalled(owner, repo string) (bool, error) {
	_, resp, err := m.emassGithubClient.Apps.FindRepositoryInstallation(m.ctx, owner, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to find repository installation: %v", err)
	}

	return true, nil
}

func (m *manager) sendEmail(emailAddress, subjectContent, body string) error {
	if DisableNotifications {
		m.logger.Warnf("notifications are disabled, skipping sending email")
		return nil
	}

	emails := []string{m.config.secondaryEmail}
	if emailAddress != "" && !includes(emails, emailAddress) {
		emails = append(emails, emailAddress)
	}
	from := fmt.Sprintf("From: %s", m.config.gmailFrom)
	to := fmt.Sprintf("To: %s", strings.Join(emails, ","))
	replyTo := fmt.Sprintf("Reply-To: %s", m.config.secondaryEmail)
	subject := fmt.Sprintf("Subject: %s", subjectContent)

	addr := "smtp.gmail.com:587"
	msg := fmt.Sprintf("%s\n%s\n%s\n%s\n\n%s", from, to, replyTo, subject, body)
	auth := smtp.PlainAuth("", m.config.gmailFrom, m.config.gmailPassword, "smtp.gmail.com")
	err := smtp.SendMail(addr, auth, from, emails, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func generateMissingEMASSEmailBody(template, repo string /*languages []string*/) string {
	//languageTemplate := ""
	//for _, language := range languages {
	//	languageTemplate += fmt.Sprintf("<li>%s</li>", language)
	//}

	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repo)
	//template = strings.ReplaceAll(template, "<LANGUAGES_PLACEHOLDER>", languageTemplate)

	return template
}

func generateMissingEMASSIssueBody(template, repo string /*languages []string*/) string {
	//languageTemplate := ""
	//for _, language := range languages {
	//	languageTemplate += fmt.Sprintf("- `%s`\n", language)
	//}

	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repo)
	//template = strings.ReplaceAll(template, "<LANGUAGES_PLACEHOLDER>", languageTemplate)

	return template
}

func generateNonCompliantEmailBody(template, repo, systemName string, systemID int64, languages []string) string {
	languageTemplate := ""
	for _, language := range languages {
		languageTemplate += fmt.Sprintf("<li>%s</li>\n", language)
	}

	template = strings.ReplaceAll(template, "<SYSTEM_ID_PLACEHOLDER>", strconv.FormatInt(systemID, 10))
	template = strings.ReplaceAll(template, "<SYSTEM_NAME_PLACEHOLDER>", systemName)
	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repo)
	template = strings.ReplaceAll(template, "<LANGUAGES_PLACEHOLDER>", languageTemplate)

	return template
}

func generateOutOfComplianceCLIEmailBody(template, repo, repoURL, version string) string {
	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repoURL)
	template = strings.ReplaceAll(template, "<REPOSITORY_NAME_PLACEHOLDER>", repo)
	template = strings.ReplaceAll(template, "<CODEQL_VERSION_PLACEHOLDER>", version)

	return template
}

func (m *manager) installEMASSApp(repositoryID int64) error {
	_, _, err := m.adminGitHubClient.Apps.AddRepository(m.ctx, m.config.emassPromotionInstallationID, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to add repository: %v", err)
	}

	return nil
}

func (m *manager) issuesExists(owner, repo, label string) (bool, error) {
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{label},
	}
	issues, _, err := m.verifyScansGithubClient.Issues.ListByRepo(m.ctx, owner, repo, opts)
	if err != nil {
		return false, fmt.Errorf("failed to list issues: %v", err)
	}

	return len(issues) > 0, nil
}

func (m *manager) createIssue(owner, repo, title, body string, labels []string) error {
	if DisableNotifications {
		m.logger.Warnf("notifications are disabled, skipping creating issue")
		return nil
	}
	request := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}
	_, _, err := m.verifyScansGithubClient.Issues.Create(m.ctx, owner, repo, request)
	if err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	return nil
}

func (m *manager) getLatestCodeQLVersions() ([]string, error) {
	opts := &github.ListOptions{
		PerPage: 5,
	}
	versions, _, err := m.verifyScansGithubClient.Repositories.ListReleases(m.ctx, "github", "codeql-cli-binaries", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %v", err)
	}

	var tags []string
	for _, version := range versions {
		tag := version.GetTagName()
		if strings.HasPrefix(tag, "v") {
			sanitizedVersion := strings.Split(tag, "v")[1]
			tags = append(tags, sanitizedVersion)
		}
	}

	return tags, nil
}

func (m *manager) listOpenIssues(owner, repo, label string) ([]int, error) {
	listOpts := github.ListOptions{
		PerPage: 100,
	}
	listIssuesOpts := &github.IssueListByRepoOptions{
		State: "open",
		Labels: []string{
			label,
		},
		ListOptions: listOpts,
	}
	issues, _, err := m.verifyScansGithubClient.Issues.ListByRepo(m.ctx, owner, repo, listIssuesOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %v", err)
	}

	var issueNumbers []int
	for _, issue := range issues {
		issueNumbers = append(issueNumbers, int(issue.GetNumber()))
	}

	return issueNumbers, nil
}

func (m *manager) closeIssues(owner, repo string, issueNumbers []int) {
	for _, number := range issueNumbers {
		_, _, err := m.verifyScansGithubClient.Issues.Edit(m.ctx, owner, repo, number, &github.IssueRequest{
			State: github.String("closed"),
		})
		if err != nil {
			m.logger.Errorf("failed to close issue: %v", err)
		}
	}
}
