package main

import (
	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/google/go-github/v52/github"
	"github.com/sethvargo/go-githubactions"
)

type manager struct {
	adminGitHubClient       *github.Client
	emassGithubClient       *github.Client
	verifyScansGithubClient *github.Client

	config *input
}

func main() {
	config := parseInput()
	adminClient := utils.NewGitHubClient(config.adminToken)
	emassClient, err := utils.NewGitHubAppClient(config.emassPromotionAppID, config.emassPromotionPrivateKey)
	if err != nil {
		githubactions.Fatalf("failed to create EMASS GitHub App client: %v", err)
	}
	verifyScansClient, err := utils.NewGitHubAppClient(config.verifyScansAppID, config.verifyScansPrivateKey)
	if err != nil {
		githubactions.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}

	m := &manager{
		adminGitHubClient:       adminClient,
		emassGithubClient:       emassClient,
		verifyScansGithubClient: verifyScansClient,
		config:                  config,
	}

	m.getFile("", "", "")
}

func (m *manager) getFile(owner, repo, path string) {

}
