package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	Context context.Context

	AdminGitHubClient                 *github.Client
	ConfigureCodeQLInstallationClient *github.Client
	VerifyScansGithubClient           *github.Client
	VerifyScansInstallationClient     *github.Client

	Config       *Input
	Logger       *log.Entry
	GlobalLogger *log.Logger

	VerifiedScansAppInstalledRepos []string
}

func (m *Manager) ProcessRepository(repo *github.Repository) {
	logger := m.GlobalLogger.WithField("repo", repo.GetName())
	m.Logger = logger

	org := repo.GetOwner().GetLogin()
	name := repo.GetName()
	defaultBranch := repo.GetDefaultBranch()

	if org != m.Config.Org {
		logger.Debugf("skipping repo %s, not in org %s", repo.GetFullName(), m.Config.Org)
		return
	}

	logger.Info("Checking if repository is ignored")
	repoIgnored, err := m.FileExists(org, name, ".github/.emass-repo-ignore")
	if err != nil {
		logger.Errorf("failed to check if repository is ignored, skipping repo: %v", err)
		return
	}
	if repoIgnored {
		logger.WithField("event", "skipped-ignored").Infof("Found .emass-repo-ignore file, skipping repository")
		return
	}

	logger.Infof("Checking if repository is already configured")
	if Contains(m.VerifiedScansAppInstalledRepos, name) {
		logger.WithField("event", "skipped-already-configured").Infof("Skipping repository as it is has already been configured via the Configure CodeQL GitHub App Pull Request")
		return
	}

	logger.Infof("Checking if repository is archived")
	if repo.GetArchived() {
		logger.WithField("event", "skipped-archived").Infof("Repository is archived, skipping")
		return
	}

	logger.Infof("Checking if repository has Code Scanning enabled")
	enabled, workflowPath, err := m.CodeScanningEnabled(m.Config.Org, name)
	if err != nil {
		logger.Errorf("failed to check if repository has Code Scanning enabled, skipping repo: %v", err)
		return
	}
	if enabled {
		logger.Infof("Code scanning enabled, validating repository in not using default code scanning")
		defaultScanningEnabled, err := m.DefaultCodeScanningEnabled(m.Config.Org, name)
		if err != nil {
			logger.Errorf("failed to check if repository is using default code scanning, skipping repo: %v", err)
			return
		}
		if defaultScanningEnabled {
			logger.Infof("Default code scanning enabled, configuring repository")
		} else {
			logger.Infof("Default code scanning disabled, validating reusable workflow in use")
			reusableWorkflowInUse, err := m.ReusableWorkflowInUse(m.Config.Org, name, repo.GetDefaultBranch(), workflowPath)
			if err != nil {
				logger.Errorf("Failed to check if reusable workflow is in use, skipping repo: %v", err)
				return
			}

			if reusableWorkflowInUse {
				logger.Infof("Reusable workflow in use, installing Verify Scans app")
				err = m.InstallVerifyScansApp(repo.GetID())
				if err != nil {
					logger.Errorf("failed to install verify-scans app, skipping repo: %v", err)
					return
				}
				logger.WithField("event", "repo-already-configured").Infof("Verify Scans app installed")
				return
			}
			logger.Infof("Reusable workflow not in use, configuring repository")
		}
	}
	logger.Debugf("Finished checking if repository has Code Scanning enabled")

	logger.Infof("Retrieving supported languages")
	languages, err := m.ListSupportedLanguages(repo.Owner.GetLogin(), repo.GetName())
	if err != nil {
		logger.Errorf("failed to retrieve supported languages, skipping repo: %v", err)
		return
	}
	logger.Debugf("Retrieved %d supported languages", len(languages))

	if len(languages) == 0 {
		logger.WithField("event", "skipped-no-supported-languages").Infof("Skipping repository as it does not contain any supported languages")
		return
	}

	logger.Infof("Generating CodeQL workflow for supported languages: [%s]", strings.Join(languages, ", "))
	workflow, err := GenerateCodeQLWorkflow(languages, repo.GetDefaultBranch())
	if err != nil {
		logger.Errorf("failed to generate CodeQL workflow, skipping repo: %v", err)
		return
	}
	logger.Debugf("Generated CodeQL workflow")

	logger.Infof("Generating emass.json contents")
	emassJSON, err := GenerateEMASSJSON()
	if err != nil {
		logger.Errorf("failed to generate emass.json, skipping repo: %v", err)
		return
	}
	logger.Debugf("Generated emass.json")

	logger.Infof("Retrieving SHA for branch %s", defaultBranch)
	sha, err := m.GetDefaultRefSHA(org, name, defaultBranch)
	if err != nil {
		logger.Errorf("failed to retrieve SHA for branch %s, skipping repo: %v", defaultBranch, err)
		return
	}
	logger.Debugf("Retrieved SHA %s for branch %s", sha, defaultBranch)

	ghasBranch := SourceBranchName
	logger.Infof("Checking if branch %s exists", ghasBranch)
	branchExists, err := m.RefExists(org, name, ghasBranch)
	if err != nil {
		logger.Errorf("failed to check if branch %s exists, skipping repo: %v", ghasBranch, err)
		return
	}
	if branchExists {
		logger.Infof("Branch %s exists, appending random suffix", ghasBranch)
		ghasBranch = fmt.Sprintf("%s-%s", ghasBranch, GenerateRandomSuffix(5))
	}
	logger.Debugf("Branch name is %s", ghasBranch)

	logger.Infof("Creating branch %s", ghasBranch)
	err = m.CreateRef(org, name, ghasBranch, sha)
	if err != nil {
		logger.Errorf("failed to create branch %s, skipping repo: %v", ghasBranch, err)
		return
	}
	logger.Debugf("Created branch %s", ghasBranch)

	logger.Infof("Checking if '.github/workflows/codeql-analysis.yml' exists")
	workflowExists, err := m.FileExists(org, name, ".github/workflows/codeql-analysis.yml")
	if err != nil {
		logger.Errorf("failed to check if workflow file exists, skipping repo: %v", err)
		return
	}
	if !workflowExists {
		logger.Infof("Workflow file does not exist, creating file")
		err = m.CreateFile(org, name, ghasBranch, ".github/workflows/codeql-analysis.yml", "Create CodeQL workflow", workflow)
		if err != nil {
			logger.Errorf("failed to create workflow file, skipping repo: %v", err)
			return
		}
		logger.Debugf("Created workflow file")
	} else {
		logger.Infof("Workflow file exists, retrieving SHA for file")
		workflowSHA, err := m.GetFileSHA(org, name, ".github/workflows/codeql-analysis.yml", ghasBranch)
		if err != nil {
			logger.Errorf("failed to retrieve SHA for workflow file, skipping repo: %v", err)
			return
		}
		logger.Debugf("Retrieved SHA %s for workflow file", sha)

		logger.Infof("Updating workflow file")
		err = m.UpdateFile(org, name, ghasBranch, workflowSHA, ".github/workflows/codeql-analysis.yml", "Update CodeQL workflow", workflow)
		if err != nil {
			logger.Errorf("failed to update workflow file, skipping repo: %v", err)
			return
		}
		logger.Debugf("Updated workflow file")
	}

	logger.Infof("Checking if '.github/emass.json' exists")
	emassExists, err := m.FileExists(org, name, ".github/emass.json")
	if err != nil {
		logger.Errorf("failed to check if emass.json exists, skipping repo: %v", err)
		return
	}
	if !emassExists {
		logger.Infof("emass.json does not exist, creating file")
		err = m.CreateFile(org, name, ghasBranch, ".github/emass.json", "Create emass.json", emassJSON)
		if err != nil {
			logger.Errorf("failed to create emass.json, skipping repo: %v", err)
			return
		}
		logger.Debugf("Created emass.json")
	} else {
		logger.Infof("emass.json exists, retrieving SHA for file")
	}

	logger.Infof("Generating pull request body with supported languages: [%s]", strings.Join(languages, ", "))
	body := GeneratePullRequestBody(m.Config.PullRequestBody, org, name, ghasBranch, languages)
	logger.Debugf("Pull request body: %s", body)

	logger.Infof("Creating pull request")
	err = m.CreatePullRequest(org, name, ghasBranch, defaultBranch, PullRequestTitle, body)
	if err != nil {
		logger.Errorf("failed to create pull request, skipping repo: %v", err)
		return
	}
	logger.Debugf("Created pull request")

	logger.Infof("Installing Verify Scans app")
	err = m.InstallVerifyScansApp(repo.GetID())
	if err != nil {
		logger.Errorf("failed to install Verify Scans app, skipping repo: %v", err)
		return
	}
	logger.WithField("event", "installed-verify-scans-application").Infof("Successfully installed 'verify-scans' app on repository")
	logger.WithField("event", "successfully-configured").Infof("Repository successfully configured")
}
