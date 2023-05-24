package internal

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) CreateFile(owner, repo, branch, path, message, content string) error {
	content = base64.StdEncoding.EncodeToString([]byte(content))
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: []byte(content),
		Branch:  github.String(branch),
	}
	_, _, err := m.AdminGitHubClient.Repositories.CreateFile(m.Context, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}

	return nil
}

func (m *Manager) CreatePullRequest(owner, repo, head, base, title, body string) error {
	pr := &github.NewPullRequest{
		Title:               github.String(title),
		Head:                github.String(head),
		Base:                github.String(base),
		Body:                github.String(body),
		MaintainerCanModify: github.Bool(true),
	}
	_, _, err := m.ConfigureCodeQLInstallationClient.PullRequests.Create(context.Background(), owner, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return nil
}

func (m *Manager) CreateRef(owner, repo, branch, sha string) error {
	ref := &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/heads/%s", branch)),
		Object: &github.GitObject{
			SHA: github.String(sha),
		},
	}
	_, _, err := m.AdminGitHubClient.Git.CreateRef(m.Context, owner, repo, ref)
	if err != nil {
		return fmt.Errorf("failed to create ref: %v", err)
	}

	return nil
}
