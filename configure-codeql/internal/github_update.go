package internal

import (
	"encoding/base64"
	"fmt"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) UpdateFile(owner, repo, branch, sha, path, message, content string) error {
	content = base64.StdEncoding.EncodeToString([]byte(content))
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: []byte(content),
		Branch:  github.String(branch),
		SHA:     github.String(sha),
	}
	_, _, err := m.AdminGitHubClient.Repositories.UpdateFile(m.Context, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("failed to update file: %v", err)
	}

	return nil
}
