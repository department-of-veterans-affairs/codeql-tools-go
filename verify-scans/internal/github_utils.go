package internal

import (
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) FileExists(owner, repo, path string) (bool, error) {
	_, _, resp, err := m.VerifyScansGithubClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get file: %v", err)
	}

	return true, nil
}

func CalculateMissingLanguages(expectedLanguages, actualLanguages []string) []string {
	var missingLanguages []string
	for _, language := range expectedLanguages {
		if !Includes(actualLanguages, language) {
			missingLanguages = append(missingLanguages, language)
		}
	}

	return missingLanguages
}

func MapLanguages(languages map[string]int) []string {
	mappedLanguages := make([]string, len(languages))
	for language := range languages {
		switch language {
		case "kotlin":
			mappedLanguages = append(mappedLanguages, "java")
		default:
			mappedLanguages = append(mappedLanguages, strings.ToLower(language))
		}
	}

	return mappedLanguages
}

func Includes(a []string, s string) bool {
	for _, value := range a {
		if value == s {
			return true
		}
	}

	return false
}

func IncludesInt64(a []int64, s int64) bool {
	for _, value := range a {
		if value == s {
			return true
		}
	}

	return false
}

func Unique(a, b []string) []string {
	var unique []string
	for _, value := range a {
		if !Includes(b, value) {
			unique = append(unique, value)
		}
	}

	return unique
}

func (m *Manager) IsDateInRange(createdAt time.Time) bool {
	currentDate := time.Now()
	diff := currentDate.Sub(createdAt)
	diffDays := int(diff.Hours() / 24)

	return diffDays <= m.Config.DaysToScan
}

func AllRequiredAnalysesFound(languages, requiredLanguages []string) bool {
	for _, language := range requiredLanguages {
		if !Includes(languages, language) {
			return false
		}
	}

	return true
}

func (m *Manager) EMASSAppInstalled(owner, repo string) (bool, error) {
	_, resp, err := m.EMASSGithubClient.Apps.FindRepositoryInstallation(m.Context, owner, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to find repository installation: %v", err)
	}

	return true, nil
}

func (m *Manager) SendEmail(emailAddress, subjectContent, body string) error {
	if DisableNotifications {
		m.Logger.Warnf("notifications are disabled, skipping sending email")
		return nil
	}

	emails := []string{m.Config.SecondaryEmail}
	if emailAddress != "" && !Includes(emails, emailAddress) {
		emails = append(emails, emailAddress)
	}
	from := fmt.Sprintf("From: %s", m.Config.GmailFrom)
	to := fmt.Sprintf("To: %s", strings.Join(emails, ","))
	replyTo := fmt.Sprintf("Reply-To: %s", m.Config.SecondaryEmail)
	subject := fmt.Sprintf("Subject: %s", subjectContent)

	addr := "smtp.gmail.com:587"
	msg := fmt.Sprintf("%s\n%s\n%s\n%s\n\n%s", from, to, replyTo, subject, body)
	auth := smtp.PlainAuth("", m.Config.GmailFrom, m.Config.GmailPassword, "smtp.gmail.com")
	err := smtp.SendMail(addr, auth, from, emails, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func GenerateMissingEMASSEmailBody(template, repo string /*languages []string*/) string {
	//languageTemplate := ""
	//for _, language := range languages {
	//	languageTemplate += fmt.Sprintf("<li>%s</li>", language)
	//}

	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repo)
	//template = strings.ReplaceAll(template, "<LANGUAGES_PLACEHOLDER>", languageTemplate)

	return template
}

func GenerateMissingEMASSIssueBody(template, repo string /*languages []string*/) string {
	//languageTemplate := ""
	//for _, language := range languages {
	//	languageTemplate += fmt.Sprintf("- `%s`\n", language)
	//}

	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repo)
	//template = strings.ReplaceAll(template, "<LANGUAGES_PLACEHOLDER>", languageTemplate)

	return template
}

func GenerateNonCompliantEmailBody(template, repo, systemName string, systemID int64, languages []string) string {
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

func GenerateOutOfComplianceCLIEmailBody(template, repo, repoURL, version string) string {
	template = strings.ReplaceAll(template, "<REPOSITORY_URL_PLACEHOLDER>", repoURL)
	template = strings.ReplaceAll(template, "<REPOSITORY_NAME_PLACEHOLDER>", repo)
	template = strings.ReplaceAll(template, "<CODEQL_VERSION_PLACEHOLDER>", version)

	return template
}

func (m *Manager) InstallEMASSApp(repositoryID int64) error {
	_, _, err := m.AdminGitHubClient.Apps.AddRepository(m.Context, m.Config.EMASSPromotionInstallationID, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to add repository: %v", err)
	}

	return nil
}

func (m *Manager) IssuesExists(owner, repo, label string) (bool, error) {
	opts := &github.IssueListByRepoOptions{
		State:  "open",
		Labels: []string{label},
	}
	issues, _, err := m.VerifyScansGithubClient.Issues.ListByRepo(m.Context, owner, repo, opts)
	if err != nil {
		return false, fmt.Errorf("failed to list issues: %v", err)
	}

	return len(issues) > 0, nil
}

func (m *Manager) CloseIssues(owner, repo string, issueNumbers []int) {
	for _, number := range issueNumbers {
		_, _, err := m.VerifyScansGithubClient.Issues.Edit(m.Context, owner, repo, number, &github.IssueRequest{
			State: github.String("closed"),
		})
		if err != nil {
			m.Logger.Errorf("failed to close issue: %v", err)
		}
	}
}
