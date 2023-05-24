// TODO: Close existing PR's and cleanup branches in case of failure

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/configure-codeql/internal"
	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
	debug := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG"))) == "true"
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if repoValue, ok := entry.Data["repo"]; ok {
		repo := fmt.Sprint(repoValue)
		if eventValue, ok := entry.Data["event"]; ok {
			event := fmt.Sprint(eventValue)
			return []byte(fmt.Sprintf("[%s]: [%s] %s\n", repo, event, entry.Message)), nil
		}
		return []byte(fmt.Sprintf("[%s]: %s\n", repo, entry.Message)), nil
	}

	return []byte(fmt.Sprintf("%s\n", entry.Message)), nil
}

func main() {
	config := internal.ParseInput()

	globalLogger := log.New()
	globalLogger.SetFormatter(&CustomFormatter{})

	globalLogger.Infof("Creating admin GitHub client")
	adminClient := utils.NewGitHubClient(config.AdminToken)

	globalLogger.Infof("Creating Configure CodeQL GitHub Installation client")
	configureCodeQLInstallationClient, err := utils.NewGitHubInstallationClient(config.ConfigureCodeQLAppID, config.ConfigureCodeQLInstallationID, config.ConfigureCodeQLPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create EMASS GitHub App client: %v", err)
	}

	globalLogger.Infof("Creating Verify Scans GitHub App client")
	verifyScansClient, err := utils.NewGitHubAppClient(config.VerifyScansAppID, config.VerifyScansPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	globalLogger.Debugf("Verify Scans GitHub App client created")

	globalLogger.Infof("Creating Verify Scans GitHub App Installation client")
	verifyScansInstallationClient, err := utils.NewGitHubInstallationClient(config.VerifyScansAppID, config.VerifyScansInstallationID, config.VerifyScansPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	globalLogger.Debugf("Verify Scans GitHub App client created")

	m := &internal.Manager{
		AdminGitHubClient:                 adminClient,
		ConfigureCodeQLInstallationClient: configureCodeQLInstallationClient,
		VerifyScansGithubClient:           verifyScansClient,
		VerifyScansInstallationClient:     verifyScansInstallationClient,

		Config:       config,
		Context:      context.Background(),
		GlobalLogger: globalLogger,
	}

	globalLogger.Infof("Querying verify-scans app for all installed repos")
	installedRepos, err := m.ListVerifyScansInstalledRepos()
	if err != nil {
		globalLogger.Fatalf("failed to list installed repos: %v", err)
	}
	m.VerifiedScansAppInstalledRepos = installedRepos
	globalLogger.Debugf("found %d installed repos", len(installedRepos))

	globalLogger.Infof("Retrieving repositories")
	repos, err := m.ListRepos()
	if err != nil {
		globalLogger.Fatalf("failed to list repositories: %v", err)
	}
	globalLogger.Debugf("Retrieved %d repositories", len(repos))

	if config.Repo == "" {
		globalLogger.Infof("Processing all repos")
		for _, repo := range repos {
			m.ProcessRepository(repo)
		}
	} else {
		globalLogger.WithField("repo", config.Repo).Infof("Processing single repo")
		repo, resp, err := m.AdminGitHubClient.Repositories.Get(m.Context, config.Org, config.Repo)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				globalLogger.Fatalf("repo does not exist: %v", err)
			}
			globalLogger.Fatalf("failed to retrieve repository: %v", err)
		}
		m.ProcessRepository(repo)
	}
}
