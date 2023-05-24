package internal

import (
	"fmt"

	"github.com/google/go-github/v52/github"
)

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
