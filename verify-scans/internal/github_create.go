package internal

import (
	"fmt"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) CreateIssue(owner, repo, title, body string, labels []string) error {
	if DisableNotifications {
		m.Logger.Warnf("notifications are disabled, skipping creating issue")
		return nil
	}
	request := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}
	_, _, err := m.VerifyScansGithubClient.Issues.Create(m.Context, owner, repo, request)
	if err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	return nil
}
