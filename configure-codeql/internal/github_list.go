package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/google/go-github/v52/github"
)

func (m *Manager) ListRepos() ([]*github.Repository, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var repos []*github.Repository
	for {
		installations, resp, err := m.ConfigureCodeQLInstallationClient.Apps.ListRepos(m.Context, opts)
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

func (m *Manager) ListSupportedLanguages(org, repo string) ([]string, error) {
	languages, resp, err := m.ConfigureCodeQLInstallationClient.Repositories.ListLanguages(m.Context, org, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("failed to list languages, unknown repository: %w", err)
		}

		return nil, fmt.Errorf("failed to list languages: %w", err)
	}

	var supportedLanguages []string
	for language := range languages {
		language = strings.ToLower(language)
		if language == "kotlin" {
			language = "java"
		}
		if utils.IsSupportedCodeQLLanguage(language) {
			supportedLanguages = append(supportedLanguages, language)
		}
	}

	return supportedLanguages, nil
}

func (m *Manager) ListVerifyScansInstalledRepos() ([]string, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}
	var repos []string
	for {
		results, resp, err := m.VerifyScansInstallationClient.Apps.ListRepos(m.Context, opts)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("failed to list installations, unknown organization: %w", err)
			}
			return nil, fmt.Errorf("failed to list installations: %w", err)
		}

		for _, repo := range results.Repositories {
			if repo.Owner.GetLogin() == m.Config.Org {
				repos = append(repos, repo.GetName())
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return repos, nil
}
