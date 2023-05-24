package internal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	Context context.Context

	AdminGitHubClient *github.Client
	EMASSAppClient    *github.Client
	EMASSOrgClient    *github.Client
	EMASSClient       *github.Client

	Config       *Input
	Logger       *log.Entry
	GlobalLogger *log.Logger

	EMASSSystemIDs []int64
}

func (m *Manager) ProcessRepository(repo *github.Repository) {
	logger := m.GlobalLogger.WithField("repo", repo.GetName())
	m.Logger = logger

	org := strings.ToLower(repo.GetOwner().GetLogin())
	name := strings.ToLower(repo.GetName())
	defaultBranch := repo.GetDefaultBranch()

	if org == m.Config.EMASSOrg {
		logger.Debugf("Skipping repository in EMASS org")
		return
	}

	logger.Info("Checking if repository is ignored")
	repoIgnored, err := m.FileExists(org, name, ".github/.emass-repo-ignore")
	if err != nil {
		logger.Fatalf("failed to check if repository is ignored: %v", err)
	}
	if repoIgnored {
		logger.WithField("event", "skipped-ignored").Infof("Found .emass-repo-ignore file, skipping repository")
		return
	}

	logger.Infof("Retrieving eMASS configuration file")
	emassConfig, err := m.GetEMASSConfig(org, name, ".github/emass.json")
	if err != nil {
		logger.Errorf("failed to retrieve eMASS Configuration File, skipping repo: %v", err)
		return
	}
	if emassConfig == nil {
		logger.WithField("event", "emass-json-not-found").Warnf("Skipping repository as it does not contain an emass.json file")
		return
	}
	if !includesInt64(m.EMASSSystemIDs, emassConfig.SystemID) {
		logger.WithField("event", "invalid-system-id").Warnf("Skipping repository as it contains an invalid System ID")
	}
	logger.Debugf("eMASS configuration file processed")

	emassRepoName := fmt.Sprintf("%d-%s", emassConfig.SystemID, name)
	logger.Infof("Checking if repository exists in eMASS org")
	repoExists, err := m.repoExists(m.Config.EMASSOrg, emassRepoName)
	if err != nil {
		logger.Errorf("failed to check if repository exists in eMASS org, skipping repo: %v", err)
		return
	}
	if !repoExists {
		logger.Infof("Repository does not exist in eMASS org, creating eMASS repository")
		err = m.createRepo(m.Config.EMASSOrg, emassRepoName)
		if err != nil {
			logger.Errorf("failed to create eMASS repository, skipping repo: %v", err)
			return
		}
	}

	logger.Infof("Retrieving CodeQL databases")
	databases, err := m.listCodeQLDatabases(org, name)
	if err != nil {
		logger.Errorf("failed to list CodeQL databases, skipping repo: %v", err)
		return
	}
	if len(databases) == 0 {
		logger.WithField("event", "skipped-database-not-found").Infof("Skipping repository as it does not contain any new CodeQL databases")
		return
	}

	for _, database := range databases {
		if !isInDayRange(database.CreatedAt, m.Config.DaysToScan) {
			continue
		}

		path := fmt.Sprintf("%s-database.zip", database.Language)
		logger.Infof("Downloading CodeQL database")
		err = m.downloadFileToDisk(database.URL, path)
		if err != nil {
			logger.Errorf("failed to download CodeQL database, skipping: %v", err)
			continue
		}
		logger.Debugf("CodeQL database downloaded")

		logger.Infof("Uploading CodeQL database to eMASS repository")
		err = m.UploadFile(m.Config.EMASSOrg, emassRepoName, database.Language, path, database.Name)
		if err != nil {
			logger.Errorf("failed to upload CodeQL database to eMASS repository: %v", err)
			continue
		}
		logger.Debugf("CodeQL database uploaded")

		logger.Infof("Deleting local CodeQL database")
		err = os.Remove(path)
		if err != nil {
			logger.Errorf("failed to delete local CodeQL database: %v", err)
			continue
		}
		logger.Debugf("CodeQL database deleted")
	}

	logger.Infof("Retrieving recent CodeQL analyses")
	analyses, err := m.ListCodeQLAnalyses(org, name, defaultBranch)
	if err != nil {
		logger.Errorf("failed to list CodeQL analyses, skipping repo: %v", err)
		return
	}
	if len(analyses) == 0 {
		logger.WithField("event", "skipped-sarif-not-found").Infof("Skipping repository as it does not contain any new SARIF analyses")
		return
	}
	logger.Debugf("CodeQL analyses retrieved")

	logger.Infof("Retrieving default branch SHA")
	sha, err := m.GetDefaultRefSHA(m.Config.EMASSOrg, emassRepoName)
	if err != nil {
		logger.Errorf("failed to retrieve default branch SHA, skipping repo: %v", err)
		return
	}
	logger.Debugf("Default branch SHA retrieved")

	logger.Infof("Checking if ref exists")
	refExists, err := m.RefExists(m.Config.EMASSOrg, emassRepoName, defaultBranch)
	if err != nil {
		logger.Errorf("failed to check if ref exists, skipping repo: %v", err)
		return
	}
	if !refExists {
		logger.Infof("Ref does not exist, creating ref")
		err = m.CreateRef(m.Config.EMASSOrg, emassRepoName, defaultBranch, sha)
		if err != nil {
			logger.Errorf("failed to create ref, skipping repo: %v", err)
			return
		}
		logger.Debugf("Ref created")
	}
	logger.Debugf("Finished evaluating ref")

	logger.Infof("Setting branch %s as default branch", defaultBranch)
	err = m.setDefaultBranch(m.Config.EMASSOrg, emassRepoName, defaultBranch)
	if err != nil {
		logger.Errorf("failed to set default branch, skipping repo: %v", err)
		return
	}
	logger.Debugf("Branch %s set as default branch", defaultBranch)

	for _, analysis := range analyses {
		logger.Infof("Downloading SARIF file")
		sarif, err := m.downloadSARIF(org, name, analysis.ID)
		if err != nil {
			logger.Errorf("failed to download SARIF file, skipping analysis: %v", err)
			continue
		}
		logger.Debugf("SARIF file downloaded")

		logger.Infof("Encoding and GZipping SARIF")
		encodedSarif, err := gZip(sarif)
		if err != nil {
			logger.Errorf("failed to encode and gzip SARIF file, skipping analysis: %v", err)
			continue
		}
		logger.Debugf("SARIF file encoded and gzipped")

		logger.Infof("Uploading SARIF file to eMASS repository")
		err = m.uploadSarif(m.Config.EMASSOrg, emassRepoName, sha, analysis.Ref, encodedSarif)
		if err != nil {
			logger.Errorf("failed to upload SARIF file to eMASS repository: %v", err)
			continue
		}
		logger.WithField("event", "successful-upload").Infof("Successfully promoted %s artifacts to eMASS repository", analysis.Language)
	}
	logger.WithField("event", "finished-processing").Infof("Finished processed repository")
}
