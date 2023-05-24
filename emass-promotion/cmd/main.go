package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/emass-promotion/internal"
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

	globalLogger.Infof("Creating eMASS Promotion GitHub App client")
	emassAppClient, err := utils.NewGitHubAppClient(config.EMASSPromotionAppID, config.EMASSPromotionPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create eMASS Promotion GitHub App client: %v", err)
	}
	globalLogger.Debugf("eMASS Promotion GitHub App client created")

	globalLogger.Infof("Creating eMASS Org GitHub App Installation client")
	emassOrgClient, err := utils.NewGitHubInstallationClient(config.EMASSPromotionAppID, config.EMASSOrgInstallationID, config.EMASSPromotionPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create eMASS Promotion GitHub App client: %v", err)
	}
	globalLogger.Debugf("eMASS Promotion GitHub App client created")

	globalLogger.Infof("Creating eMASS Org GitHub App Installation client")
	emassClient, err := utils.NewGitHubInstallationClient(config.EMASSPromotionAppID, config.EMASSPromotionInstallationID, config.EMASSPromotionPrivateKey)
	if err != nil {
		globalLogger.Fatalf("failed to create eMASS Promotion GitHub App client: %v", err)
	}
	globalLogger.Debugf("eMASS Promotion GitHub App client created")

	m := &internal.Manager{
		Context: context.Background(),

		AdminGitHubClient: adminClient,
		EMASSAppClient:    emassAppClient,
		EMASSOrgClient:    emassOrgClient,
		EMASSClient:       emassClient,

		Config:       config,
		GlobalLogger: globalLogger,
	}

	globalLogger.Infof("Retrieving repositories")
	repos, err := m.ListRepos()
	if err != nil {
		globalLogger.Fatalf("failed to list repositories: %v", err)
	}
	globalLogger.Debugf("Retrieved %d repositories", len(repos))

	globalLogger.Infof("Retrieving eMASS system list")
	emassSystemIDs, err := m.GetEMASSSystemList(m.Config.EMASSSystemListOrg, m.Config.EMASSSystemListRepo, m.Config.EMASSSystemListPath)
	if err != nil {
		globalLogger.Fatalf("failed to get eMASS system list: %v", err)
	}
	m.EMASSSystemIDs = emassSystemIDs
	globalLogger.Debugf("Retrieved %d eMASS system IDs", len(emassSystemIDs))

	if config.Repo == "" {
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
