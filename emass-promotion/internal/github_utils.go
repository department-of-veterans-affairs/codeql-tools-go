package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) FileExists(owner, repo, path string) (bool, error) {
	_, _, resp, err := m.EMASSClient.Repositories.GetContents(m.Context, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get file: %v", err)
	}

	return true, nil
}

func (m *Manager) repoExists(owner, repo string) (bool, error) {
	_, resp, err := m.EMASSOrgClient.Repositories.Get(m.Context, owner, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get repository: %v", err)
	}

	return true, nil
}

func (m *Manager) createRepo(org, repo string) error {
	_, _, err := m.EMASSOrgClient.Repositories.Create(m.Context, org, &github.Repository{
		Name:        &repo,
		Private:     github.Bool(true),
		HasIssues:   github.Bool(false),
		HasProjects: github.Bool(false),
		HasWiki:     github.Bool(false),
		AutoInit:    github.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create repository: %v", err)
	}

	return nil
}

func (m *Manager) downloadFileToDisk(url, path string) error {
	installation, _, err := m.EMASSAppClient.Apps.CreateInstallationToken(m.Context, m.Config.EMASSPromotionInstallationID, nil)
	if err != nil {
		return fmt.Errorf("failed to create token: %v", err)
	}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	request.Header.Set("Accept", "application/zip")
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", installation.GetToken()))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return fmt.Errorf("failed to copy content to file: %v", err)
	}

	return nil
}

func (m *Manager) UploadFile(owner, repo, language, path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	uploadURL := fmt.Sprintf("https://uploads.github.com/repos/%s/%s/code-scanning/codeql/databases/%s?name=%s", owner, repo, language, name)
	request, err := m.EMASSOrgClient.NewUploadRequest(uploadURL, file, fileInfo.Size(), "application/zip")
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}

	_, err = m.EMASSOrgClient.Do(m.Context, request, nil)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}

	return nil
}

func (m *Manager) RefExists(owner, repo, branch string) (bool, error) {
	ref := fmt.Sprintf("heads/%s", branch)
	_, resp, err := m.EMASSOrgClient.Git.GetRef(m.Context, owner, repo, ref)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, fmt.Errorf("failed to get ref: %v", err)
	}

	return true, nil
}

func (m *Manager) setDefaultBranch(owner, repo, branch string) error {
	_, _, err := m.EMASSOrgClient.Repositories.Edit(m.Context, owner, repo, &github.Repository{
		DefaultBranch: &branch,
	})
	if err != nil {
		return fmt.Errorf("failed to set default branch: %v", err)
	}

	return nil
}

func (m *Manager) downloadSARIF(org, repo string, id int64) ([]byte, error) {
	installation, _, err := m.EMASSAppClient.Apps.CreateInstallationToken(m.Context, m.Config.EMASSPromotionInstallationID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/%s/code-scanning/analyses/%d", org, repo, id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	request.Header.Set("Accept", "application/sarif+json")
	request.Header.Set("Authorization", fmt.Sprintf("bearer %s", installation.GetToken()))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %v", err)
	}

	return body, nil
}

func (m *Manager) uploadSarif(owner, repo, sha, ref, sarif string) error {
	_, resp, err := m.EMASSOrgClient.CodeScanning.UploadSarif(m.Context, owner, repo, &github.SarifAnalysis{
		CommitSHA: &sha,
		Ref:       &ref,
		Sarif:     &sarif,
	})
	if err != nil {
		if resp.StatusCode == http.StatusAccepted {
			return nil
		}
		return fmt.Errorf("failed to upload sarif: %v", err)
	}

	return nil
}
