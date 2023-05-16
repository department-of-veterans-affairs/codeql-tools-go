// TODO: Close existing PR's and cleanup branches in case of failure

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/department-of-veterans-affairs/codeql-tools/configure-codeql/internal"
	"github.com/department-of-veterans-affairs/codeql-tools/utils"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
	debug := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG"))) == "true"
	if debug {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	config := internal.ParseInput()

	rootLogger := log.WithField("app", "configure-codeql")

	adminClient := utils.NewGitHubClient(config.AdminToken)
	//configureCodeQLClient, err := utils.NewGitHubAppClient(config.configureCodeQLAppID, config.configureCodeQLPrivateKey)
	//if err != nil {
	//	core.Fatalf("failed to create EMASS GitHub App client: %v", err)
	//}

	configureCodeQLInstallationClient, err := utils.NewGitHubInstallationClient(config.ConfigureCodeQLAppID, config.ConfigureCodeQLInstallationID, config.ConfigureCodeQLPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create EMASS GitHub App client: %v", err)
	}

	rootLogger.Infof("Creating Verify Scans GitHub App client")
	verifyScansClient, err := utils.NewGitHubAppClient(config.VerifyScansAppID, config.VerifyScansPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	rootLogger.Debugf("Verify Scans GitHub App client created")

	rootLogger.Infof("Creating Verify Scans GitHub App Installation client")
	verifyScansInstallationClient, err := utils.NewGitHubInstallationClient(config.VerifyScansAppID, config.VerifyScansInstallationID, config.VerifyScansPrivateKey)
	if err != nil {
		rootLogger.Fatalf("failed to create Verify Scans GitHub App client: %v", err)
	}
	rootLogger.Debugf("Verify Scans GitHub App client created")

	m := &internal.Manager{
		AdminGitHubClient: adminClient,
		//configureCodeQLClient:             configureCodeQLClient,
		ConfigureCodeQLInstallationClient: configureCodeQLInstallationClient,
		VerifyScansGithubClient:           verifyScansClient,
		VerifyScansInstallationClient:     verifyScansInstallationClient,

		Config:  config,
		Context: context.Background(),
	}

	rootLogger.Infof("Querying verify-scans app for all installed repos")
	installedRepos, err := m.ListVerifyScansInstalledRepos()
	if err != nil {
		rootLogger.Fatalf("failed to list installed repos: %v", err)
	}
	rootLogger.Debugf("found %d installed repos", len(installedRepos))

	rootLogger.Infof("Retrieving repositories")
	repos, err := m.ListRepos()
	if err != nil {
		rootLogger.Fatalf("failed to list repositories: %v", err)
	}
	rootLogger.Debugf("Retrieved %d repositories", len(repos))

	for _, repo := range repos {
		logger := rootLogger.WithField("repo", repo.GetName())
		m.Logger = logger

		org := repo.GetOwner().GetLogin()
		name := repo.GetName()
		defaultBranch := repo.GetDefaultBranch()

		if org != config.Org {
			logger.Debugf("skipping repo %s, not in org %s", repo.GetFullName(), config.Org)
			continue
		}

		logger.Info("Checking if repository is ignored")
		repoIgnored, err := m.FileExists(org, name, ".github/.emass-repo-ignore")
		if err != nil {
			logger.Errorf("failed to check if repository is ignored, skipping repo: %v", err)
			continue
		}
		if repoIgnored {
			logger.Infof("[skipped-ignore] Repository is ignored, skipping")
			continue
		}

		if repo.GetArchived() {
			logger.Infof("Repository is archived, skipping")
			continue
		}

		if internal.Contains(installedRepos, name) {
			logger.Infof("Repository is already configured, skipping")
			continue
		}

		logger.Infof("Checking if repository has Code Scanning enabled")
		enabled, workflowPath, err := m.CodeScanningEnabled(config.Org, name)
		if err != nil {
			logger.Errorf("failed to check if repository has Code Scanning enabled, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Finished checking if repository has Code Scanning enabled")

		if enabled {
			logger.Infof("Code scanning enabled, validating repository in not using default code scanning")
			defaultScanningEnabled, err := m.DefaultCodeScanningEnabled(config.Org, name)
			if err != nil {
				logger.Errorf("failed to check if repository is using default code scanning, skipping repo: %v", err)
				continue
			}

			if defaultScanningEnabled {
				logger.Infof("Default code scanning enabled, configuring repository")
			} else {
				logger.Infof("Default code scanning disabled, validating reusable workflow in use")
				reusableWorkflowInUse, err := m.ReusableWorkflowInUse(config.Org, name, repo.GetDefaultBranch(), workflowPath)
				if err != nil {
					logger.Errorf("Failed to check if reusable workflow is in use, skipping repo: %v", err)
					continue
				}

				if reusableWorkflowInUse {
					logger.Infof("Reusable workflow in use, installing Verify Scans app")
					err = m.InstallVerifyScansApp(repo.GetID())
					if err != nil {
						logger.Errorf("failed to install verify-scans app: %v", err)
						continue
					}
					logger.Debugf("Verify Scans app installed")
					continue
				}
				logger.Infof("Reusable workflow not in use, configuring repository")
			}
		}

		logger.Infof("Retrieving supported languages")
		languages, err := m.ListSupportedLanguages(repo.Owner.GetLogin(), repo.GetName())
		if err != nil {
			logger.Errorf("failed to retrieve supported languages, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Retrieved %d supported languages", len(languages))

		logger.Infof("Generating CodeQL workflow")
		workflow, err := internal.GenerateCodeQLWorkflow(languages, repo.GetDefaultBranch())
		if err != nil {
			logger.Errorf("failed to generate CodeQL workflow, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Generated CodeQL workflow")

		logger.Infof("Generating emass.json contents")
		emassJSON, err := internal.GenerateEMASSJSON()
		if err != nil {
			logger.Errorf("failed to generate emass.json, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Generated emass.json")

		logger.Infof("Retrieving SHA for branch %s", defaultBranch)
		sha, err := m.GetDefaultRefSHA(org, name, defaultBranch)
		if err != nil {
			logger.Errorf("failed to retrieve SHA for branch %s, skipping repo: %v", defaultBranch, err)
			continue
		}
		logger.Debugf("Retrieved SHA %s for branch %s", sha, defaultBranch)

		ghasBranch := internal.SourceBranchName
		logger.Infof("Checking if branch %s exists", ghasBranch)
		branchExists, err := m.RefExists(org, name, ghasBranch)
		if err != nil {
			logger.Errorf("failed to check if branch %s exists, skipping repo: %v", ghasBranch, err)
			continue
		}
		if branchExists {
			logger.Infof("Branch %s exists, appending random suffix", ghasBranch)
			ghasBranch = fmt.Sprintf("%s-%s", ghasBranch, internal.GenerateRandomSuffix(5))
		}
		logger.Debugf("Branch name is %s", ghasBranch)

		logger.Infof("Creating branch %s", ghasBranch)
		err = m.CreateRef(org, name, ghasBranch, sha)
		if err != nil {
			logger.Errorf("failed to create branch %s, skipping repo: %v", ghasBranch, err)
			continue
		}
		logger.Debugf("Created branch %s", ghasBranch)

		logger.Infof("Checking if '.github/workflows/codeql-analysis.yml' exists")
		workflowExists, err := m.FileExists(org, name, ".github/workflows/codeql-analysis.yml")
		if err != nil {
			logger.Errorf("failed to check if workflow file exists, skipping repo: %v", err)
			continue
		}
		if !workflowExists {
			logger.Infof("Workflow file does not exist, creating file")
			err = m.CreateFile(org, name, ghasBranch, ".github/workflows/codeql-analysis.yml", "Create CodeQL workflow", workflow)
			if err != nil {
				logger.Errorf("failed to create workflow file, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Created workflow file")
		} else {
			logger.Infof("Workflow file exists, retrieving SHA for file")
			workflowSHA, err := m.GetFileSHA(org, name, ".github/workflows/codeql-analysis.yml", ghasBranch)
			if err != nil {
				logger.Errorf("failed to retrieve SHA for workflow file, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Retrieved SHA %s for workflow file", sha)

			logger.Infof("Updating workflow file")
			err = m.UpdateFile(org, name, ghasBranch, workflowSHA, ".github/workflows/codeql-analysis.yml", "Update CodeQL workflow", workflow)
			if err != nil {
				logger.Errorf("failed to update workflow file, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Updated workflow file")
		}

		logger.Infof("Checking if '.github/emass.json' exists")
		emassExists, err := m.FileExists(org, name, ".github/emass.json")
		if err != nil {
			logger.Errorf("failed to check if emass.json exists, skipping repo: %v", err)
			continue
		}

		if !emassExists {
			logger.Infof("emass.json does not exist, creating file")
			err = m.CreateFile(org, name, ghasBranch, ".github/emass.json", "Create emass.json", emassJSON)
			if err != nil {
				logger.Errorf("failed to create emass.json, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Created emass.json")
		} else {
			logger.Infof("emass.json exists, retrieving SHA for file")
			emassSHA, err := m.GetFileSHA(org, name, ghasBranch, ".github/emass.json")
			if err != nil {
				logger.Errorf("failed to retrieve SHA for emass.json, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Retrieved SHA %s for emass.json", sha)

			logger.Infof("Updating emass.json")
			err = m.UpdateFile(org, name, ghasBranch, emassSHA, ".github/emass.json", "Update emass.json", emassJSON)
			if err != nil {
				logger.Errorf("failed to update emass.json, skipping repo: %v", err)
				continue
			}
			logger.Debugf("Updated emass.json")
		}

		logger.Infof("Generating pull request body with supported languages: [%s]", strings.Join(languages, ", "))
		body := internal.GeneratePullRequestBody(m.Config.PullRequestBody, org, name, ghasBranch, languages)
		logger.Debugf("Pull request body: %s", body)

		logger.Infof("Creating pull request")
		err = m.CreatePullRequest(org, name, ghasBranch, defaultBranch, internal.PullRequestTitle, body)
		if err != nil {
			logger.Errorf("failed to create pull request, skipping repo: %v", err)
			continue
		}
		logger.Debugf("Created pull request")

		logger.Infof("Installing Verify Scans app")
		err = m.InstallVerifyScansApp(repo.GetID())
		if err != nil {
			logger.Errorf("failed to install Verify Scans app, skipping repo: %v", err)
			continue
		}
		logger.Infof("[installed-verify-scans-application] Installed 'verify-scans' app")
		logger.Infof("[successfully-configured] Repository successfully configured")
	}
}
