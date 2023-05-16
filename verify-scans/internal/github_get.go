package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) GetEMASSConfig(owner, repo, path string) (*EMASSConfig, error) {
	content, _, resp, err := m.VerifyScansGithubClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
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

	var config EMASSConfig
	err = json.Unmarshal([]byte(decodedContent), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file content: %v", err)
	}

	return &config, nil
}

func (m *Manager) GetCodeQLConfig(owner, repo, path string) (*CodeQLConfig, error) {
	content, _, resp, err := m.VerifyScansGithubClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return &CodeQLConfig{
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

	var config CodeQLConfig
	err = json.Unmarshal([]byte(decodedContent), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file content: %v", err)
	}

	return &config, nil
}

func (m *Manager) GetEMASSSystemList(owner, repo, path string) ([]int64, error) {
	content, _, resp, err := m.AdminGitHubClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
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

func (m *Manager) GetLatestCodeQLVersions() ([]string, error) {
	opts := &github.ListOptions{
		PerPage: 5,
	}
	versions, _, err := m.VerifyScansGithubClient.Repositories.ListReleases(m.Context, "github", "codeql-cli-binaries", opts)
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
