package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/google/go-github/v52/github"
	core "github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"
)

var (
	ConfigFailed             map[string]error
	InstalledVerifyScansApp  []string
	SkippedArchived          []string
	SkippedAlreadyConfigured []string
)

const (
	PullRequestTitle = "Action Required: Configure CodeQL"
	SourceBranchName = "ghas-enforcement-codeql"
	SourceRepo       = "department-of-veterans-affairs/codeql-tools"
)

type manager struct {
	adminGitHubClient                 *github.Client
	configureCodeQLClient             *github.Client
	configureCodeQLInstallationClient *github.Client
	verifyScansGithubClient           *github.Client
	verifyScansInstallationClient     *github.Client

	config  *input
	context context.Context
}

func init() {
	SkippedArchived = []string{}
	SkippedAlreadyConfigured = []string{}
	ConfigFailed = make(map[string]error)
}

func main() {
	config := parseInput()
	adminClient := utils.NewGitHubClient(config.adminToken)
	configureCodeQLClient, err := utils.NewGitHubAppClient(config.configureCodeQLAppID, config.configureCodeQLPrivateKey)
	if err != nil {
		core.Fatalf("failed to create EMASS GitHub App client: %v", err)
	}
	configureCodeQLInstallationClient, err := utils.NewGitHubInstallationClient(config.configureCodeQLAppID, config.configureCodeQLPrivateKey, config.configureCodeQLInstallationID)
	if err != nil {
		core.Fatalf("failed to create EMASS GitHub App client: %v", err)
	}
	verifyScansClient, err := utils.NewGitHubAppClient(config.verifyScansAppID, config.verifyScansPrivateKey)
	if err != nil {
		core.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	verifyScansInstallationClient, err := utils.NewGitHubInstallationClient(config.verifyScansAppID, config.verifyScansPrivateKey, config.verifyScansInstallationID)
	if err != nil {
		core.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}

	m := &manager{
		adminGitHubClient:                 adminClient,
		configureCodeQLClient:             configureCodeQLClient,
		configureCodeQLInstallationClient: configureCodeQLInstallationClient,
		verifyScansGithubClient:           verifyScansClient,
		verifyScansInstallationClient:     verifyScansInstallationClient,

		config:  config,
		context: context.Background(),
	}

	core.Infof("querying verify-scans app for all installed repos")
	installedRepos, err := m.listInstalledRepos()
	if err != nil {
		core.Fatalf("failed to list installed repos: %v", err)
	}

	opts := &github.ListOptions{
		PerPage: 100,
	}
	for {
		results, resp, err := m.configureCodeQLInstallationClient.Apps.ListRepos(m.context, opts)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				core.Fatalf("failed to list repositories, unknown organization: %v", err)
			}
			core.Fatalf("failed to list repositories: %v", err)
		}

		for _, repo := range results.Repositories {
			core.Infof("[%s] processing repository", repo.GetName())
			if repo.Owner.GetLogin() != config.org {
				core.Infof("skipping repository %s, not in organization %s", repo.GetFullName(), config.org)
				continue
			}

			name := repo.GetName()
			if repo.GetArchived() {
				core.Infof("[%s] skipping archived repo %s", name, name)
				SkippedArchived = append(SkippedArchived, name)
				continue
			}

			if contains(installedRepos, repo.GetName()) {
				core.Infof("[%s] skipping repo, already configured", name)
				SkippedAlreadyConfigured = append(SkippedAlreadyConfigured, name)
				continue
			}

			core.Infof("[%s] checking if repository has Code Scanning enabled", name)
			enabled, workflowPath, err := m.codeScanningEnabled(config.org, name)
			if err != nil {
				core.Errorf("[%s] failed to check if repository has Code Scanning enabled: %v", name, err)
				ConfigFailed[name] = err
				continue
			}

			if enabled {
				core.Infof("[%s] code scanning enabled, validating repository in not using default code scanning", name)
				defaultScanningEnabled, err := m.defaultCodeScanningEnabled(config.org, name)
				if err != nil {
					core.Errorf("[%s] failed to check if repository is using default code scanning: %v", name, err)
					ConfigFailed[name] = err
					continue
				}

				if defaultScanningEnabled {
					core.Infof("[%s] default code scanning enabled, configuring repository", name)
				} else {
					core.Infof("[%s] default code scanning disabled, validating reusable workflow in use", name)
					reusableWorkflowInUse, err := m.reusableWorkflowInUse(config.org, name, repo.GetDefaultBranch(), workflowPath)
					if err != nil {
						core.Errorf("[%s] failed to check if reusable workflow is in use: %v", name, err)
						ConfigFailed[name] = err
						continue
					}

					if reusableWorkflowInUse {
						core.Infof("[%s] reusable workflow in use, installing verify-scans app", name)
						_, _, err = m.adminGitHubClient.Apps.AddRepository(m.context, config.verifyScansInstallationID, repo.GetID())
						if err != nil {
							core.Errorf("[%s] failed to install verify-scans app: %v", name, err)
							ConfigFailed[name] = err
							continue
						}
						InstalledVerifyScansApp = append(InstalledVerifyScansApp, name)
						continue
					}

					core.Infof("[%s] reusable workflow not in use, configuring repository", name)
				}

				core.Infof("[%s] retrieving supported languages", name)
				languages, err := m.listSupportedLanguages(repo.Owner.GetLogin(), repo.GetName())
				if err != nil {
					core.Errorf("[%s] failed to retrieve supported languages: %v", name, err)
					ConfigFailed[name] = err
					continue
				}

				core.Infof("[%s] generating CodeQL workflow", name)
				workflow, err := generateCodeQLWorkflow(languages, repo.GetDefaultBranch())
				if err != nil {
					core.Errorf("[%s] failed to generate CodeQL workflow: %v", name, err)
					ConfigFailed[name] = err
					continue
				}

				fmt.Println(workflow)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
}

func (m *manager) listInstalledRepos() ([]string, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}
	var repos []string
	for {
		results, resp, err := m.verifyScansInstallationClient.Apps.ListRepos(m.context, opts)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("failed to list installations, unknown organization: %w", err)
			}
			return nil, fmt.Errorf("failed to list installations: %w", err)
		}

		for _, repo := range results.Repositories {
			if repo.Owner.GetLogin() == m.config.org {
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

func (m *manager) codeScanningEnabled(org, repo string) (bool, string, error) {
	alerts, resp, err := m.adminGitHubClient.CodeScanning.ListAnalysesForRepo(m.context, org, repo, &github.AnalysesListOptions{
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

func (m *manager) defaultCodeScanningEnabled(org, repo string) (bool, error) {
	url := fmt.Sprintf("/repos/%s/%s/code-scanning/default-setup", org, repo)
	req, err := m.adminGitHubClient.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	var result defaultCodeScanning
	resp, err := m.adminGitHubClient.Do(m.context, req, &result)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return result.State == "configured", nil
}

func (m *manager) reusableWorkflowInUse(org, repo, branch, path string) (bool, error) {
	results, _, resp, err := m.verifyScansInstallationClient.Repositories.GetContents(m.context, org, repo, path, &github.RepositoryContentGetOptions{
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

func (m *manager) listSupportedLanguages(org, repo string) ([]string, error) {
	languages, resp, err := m.configureCodeQLInstallationClient.Repositories.ListLanguages(m.context, org, repo)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("failed to list languages, unknown repository: %w", err)
		}

		return nil, fmt.Errorf("failed to list languages: %w", err)
	}

	var supportedLanguages []string
	for language, _ := range languages {
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

func generateCodeQLWorkflow(languages []string, defaultBranch string) (string, error) {
	workflow := analysisTemplate{
		Name: "CodeQL",
		On: on{
			Push: branch{
				Branches: []string{defaultBranch},
			},
			PullRequest: branch{
				Branches: []string{defaultBranch},
			},
			Schedule: []cron{
				{
					Cron: generateRandomWeeklyCron(),
				},
			},
			WorkflowDispatch: nil,
		},
		Jobs: jobs{
			Analyze: job{
				Name:        "Analyze",
				RunsOn:      "ubuntu-latest",
				Concurrency: "${{ github.workflow }}-${{ github.ref }}",
				Permissions: map[string]string{
					"actions":         "read",
					"contents":        "read",
					"security-events": "write",
				},
				Strategy: strategy{
					FailFast: false,
					Matrix: matrix{
						Language: languages,
					},
				},
				Steps: []step{
					{
						Name: "Run Code Scanning",
						Uses: "department-of-veterans-affairs/codeql-tools/codeql-analysis@main",
						With: map[string]string{
							"languages": "${{ matrix.Language }}",
						},
					},
				},
			},
		},
	}

	workflowBytes, err := yaml.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow: %w", err)
	}

	return string(workflowBytes), nil
}

func generateRandomWeeklyCron() string {
	minute := rand.Intn(60)
	hour := rand.Intn(24)
	dayOfWeek := rand.Intn(7)

	return fmt.Sprintf("%d %d * * %d", minute, hour, dayOfWeek)
}

func contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}
