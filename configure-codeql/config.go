package main

import (
	"strconv"

	"github.com/sethvargo/go-githubactions"
)

func parseInput() *input {
	adminToken := githubactions.GetInput("admin_token")
	if adminToken == "" {
		githubactions.Fatalf("admin_token input is required")
	}

	configureCodeQLAppID := githubactions.GetInput("configure_codeql_app_id")
	if configureCodeQLAppID == "" {
		githubactions.Fatalf("configure_codeql_app_id input is required")
	}

	configureCodeQLPrivateKey := githubactions.GetInput("configure_codeql_private_key")
	if configureCodeQLPrivateKey == "" {
		githubactions.Fatalf("configure_codeql_private_key input is required")
	}

	configureCodeQLInstallationID := githubactions.GetInput("configure_codeql_installation_id")
	if configureCodeQLInstallationID == "" {
		githubactions.Fatalf("configure_codeql_installation_id input is required")
	}

	org := githubactions.GetInput("org")
	if org == "" {
		githubactions.Fatalf("org input is required")
	}

	pullRequestBody := githubactions.GetInput("pull_request_body")
	if pullRequestBody == "" {
		githubactions.Fatalf("pull_request_body input is required")
	}

	verifyScansAppID := githubactions.GetInput("verify_scans_app_id")
	if verifyScansAppID == "" {
		githubactions.Fatalf("verify_scans_app_id input is required")
	}

	verifyScansPrivateKey := githubactions.GetInput("verify_scans_private_key")
	if verifyScansPrivateKey == "" {
		githubactions.Fatalf("verify_scans_private_key input is required")
	}

	verifyScansInstallationID := githubactions.GetInput("verify_scans_installation_id")
	if verifyScansInstallationID == "" {
		githubactions.Fatalf("verify_scans_installation_id input is required")
	}

	configureCodeQLAppIDInt64, err := strconv.ParseInt(configureCodeQLAppID, 10, 64)
	if err != nil {
		githubactions.Fatalf("configure_codeql_app_id input must be an integer: %v", err)
	}

	configureCodeQLInstallationIDInt64, err := strconv.ParseInt(configureCodeQLInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("configure_codeql_installation_id input must be an integer: %v", err)
	}

	verifyScansAppIDInt64, err := strconv.ParseInt(verifyScansAppID, 10, 64)
	if err != nil {
		githubactions.Fatalf("verify_scans_app_id input must be an integer: %v", err)
	}

	verifyScansInstallationIDInt64, err := strconv.ParseInt(verifyScansInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("verify_scans_installation_id input must be an integer: %v", err)
	}

	return &input{
		adminToken:                    adminToken,
		configureCodeQLAppID:          configureCodeQLAppIDInt64,
		configureCodeQLPrivateKey:     []byte(configureCodeQLPrivateKey),
		configureCodeQLInstallationID: configureCodeQLInstallationIDInt64,
		org:                           org,
		pullRequestBody:               pullRequestBody,
		verifyScansAppID:              verifyScansAppIDInt64,
		verifyScansPrivateKey:         []byte(verifyScansPrivateKey),
		verifyScansInstallationID:     verifyScansInstallationIDInt64,
	}
}
