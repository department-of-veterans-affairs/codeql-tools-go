package internal

import (
	"fmt"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) GetDefaultRefSHA(owner, repo, branch string) (string, error) {
	refSpec := fmt.Sprintf("heads/%s", branch)
	ref, _, err := m.AdminGitHubClient.Git.GetRef(m.Context, owner, repo, refSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get ref: %v", err)
	}

	return ref.GetObject().GetSHA(), nil
}

func (m *Manager) GetFileSHA(owner, repo, branch, path string) (string, error) {
	fileContent, _, _, err := m.ConfigureCodeQLInstallationClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: fmt.Sprintf("refs/heads/%s", branch),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get file: %v", err)
	}

	return fileContent.GetSHA(), nil
}
