package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v52/github"
)

func (m *Manager) ListRepos() ([]*github.Repository, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var repos []*github.Repository
	for {
		installations, resp, err := m.EMASSClient.Apps.ListRepos(m.Context, opts)
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

func (m *Manager) listCodeQLDatabases(org, repo string) ([]codeQLDatabase, error) {
	databaseAPIEndpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/code-scanning/codeql/databases", org, repo)
	apiURL, err := url.Parse(databaseAPIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %v", err)
	}

	var databases []codeQLDatabase
	request := &http.Request{
		Method: http.MethodGet,
		URL:    apiURL,
	}
	response, err := m.EMASSClient.Do(m.Context, request, &databases)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get databases: %v", err)
	}

	return databases, nil
}

func (m *Manager) ListCodeQLAnalyses(owner, repo, branch string) ([]analysisResult, error) {
	page := 0
	var results []analysisResult
	endpoint := "https://api.github.com/repos/%s/%s/code-scanning/analyses?per_page=100&page=%d"
	for {
		page++
		analysesAPIEndpoint := fmt.Sprintf(endpoint, owner, repo, page)
		apiURL, err := url.Parse(analysesAPIEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url: %v", err)
		}

		requestBody := &analysisRequest{
			ToolName:  "CodeQL",
			Ref:       fmt.Sprintf("refs/heads/%s", branch),
			Direction: "desc",
			Sort:      "created",
		}
		requestJSON, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}

		var analysesResults []analysisResult
		request := &http.Request{
			Method: http.MethodGet,
			URL:    apiURL,
			Body:   io.NopCloser(bytes.NewReader(requestJSON)),
		}
		response, err := m.EMASSClient.Do(m.Context, request, &analysesResults)
		if err != nil {
			if response.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("failed to get analyses, repository not found: %v", err)
			}
			return nil, fmt.Errorf("failed to make request: %v", err)
		}
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get databases: %v", err)
		}

		if len(analysesResults) == 0 {
			break
		}

		done := false
		for _, analysis := range analysesResults {
			if !isInDayRange(analysis.CreatedAt, m.Config.DaysToScan) {
				done = true
			}
			if strings.Contains(analysis.Category, "ois-") {
				results = append(results, analysis)
			}
		}
		if done {
			break
		}
	}

	latestAnalyses := make(map[string]analysisResult)
	for _, analysis := range results {
		if _, ok := latestAnalyses[analysis.Language]; !ok {
			latestAnalyses[analysis.Language] = analysis
		}
	}

	var analyses []analysisResult
	for _, analysis := range latestAnalyses {
		analyses = append(analyses, analysis)
	}

	return analyses, nil
}
