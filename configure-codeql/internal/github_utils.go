package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) CodeScanningEnabled(org, repo string) (bool, string, error) {
	alerts, resp, err := m.AdminGitHubClient.CodeScanning.ListAnalysesForRepo(m.Context, org, repo, &github.AnalysesListOptions{
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, "", nil
		}

		return false, "", fmt.Errorf("failed to list alerts: %w", err)
	}
	workflow := strings.Split(alerts[0].GetAnalysisKey(), ":")[0]

	return true, workflow, nil
}

func (m *Manager) DefaultCodeScanningEnabled(org, repo string) (bool, error) {
	url := fmt.Sprintf("/repos/%s/%s/code-scanning/default-setup", org, repo)
	req, err := m.AdminGitHubClient.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	var result DefaultCodeScanning
	resp, err := m.AdminGitHubClient.Do(m.Context, req, &result)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return result.State == "configured", nil
}

func (m *Manager) FileExists(owner, repo, path string) (bool, error) {
	_, _, resp, err := m.ConfigureCodeQLInstallationClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get file: %v", err)
	}

	return true, nil
}

func (m *Manager) InstallVerifyScansApp(repositoryID int64) error {
	_, _, err := m.AdminGitHubClient.Apps.AddRepository(m.Context, m.Config.VerifyScansInstallationID, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to add repository: %v", err)
	}

	return nil
}

func (m *Manager) RefExists(owner, repo, branch string) (bool, error) {
	ref := fmt.Sprintf("heads/%s", branch)
	_, resp, err := m.AdminGitHubClient.Git.GetRef(m.Context, owner, repo, ref)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get ref: %v", err)
	}

	return true, nil
}

func (m *Manager) ReusableWorkflowInUse(org, repo, branch, path string) (bool, error) {
	results, _, resp, err := m.VerifyScansInstallationClient.Repositories.GetContents(m.Context, org, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get contents: %w", err)
	}

	content, err := results.GetContent()
	if err != nil {
		return false, fmt.Errorf("failed to get content: %w", err)
	}

	return strings.Contains(strings.ToLower(content), SourceRepo), nil
}

func (m *Manager) VerifyScansAppInstalled(owner, repo string) (bool, error) {
	_, resp, err := m.VerifyScansGithubClient.Apps.FindRepositoryInstallation(m.Context, owner, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to find repository installation: %v", err)
	}

	return true, nil
}
