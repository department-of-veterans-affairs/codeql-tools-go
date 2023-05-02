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

	daysToScan := githubactions.GetInput("days_to_scan")
	if daysToScan == "" {
		githubactions.Fatalf("days_to_scan input is required")
	}

	emassPromotionAppID := githubactions.GetInput("emass_promotion_app_id")
	if emassPromotionAppID == "" {
		githubactions.Fatalf("emass_promotion_app_id input is required")
	}

	emassPromotionPrivateKey := githubactions.GetInput("emass_promotion_private_key")
	if emassPromotionPrivateKey == "" {
		githubactions.Fatalf("emass_promotion_private_key input is required")
	}

	emassPromotionInstallationID := githubactions.GetInput("emass_promotion_installation_id")
	if emassPromotionInstallationID == "" {
		githubactions.Fatalf("emass_promotion_installation_id input is required")
	}

	gmailFrom := githubactions.GetInput("gmail_from")
	if gmailFrom == "" {
		githubactions.Fatalf("gmail_from input is required")
	}

	gmailUser := githubactions.GetInput("gmail_user")
	if gmailUser == "" {
		githubactions.Fatalf("gmail_user input is required")
	}

	gmailPassword := githubactions.GetInput("gmail_password")
	if gmailPassword == "" {
		githubactions.Fatalf("gmail_password input is required")
	}

	missingInfoEmailTemplate := githubactions.GetInput("missing_info_email_template")
	if missingInfoEmailTemplate == "" {
		githubactions.Fatalf("missing_info_email_template input is required")
	}

	missingInfoIssueTemplate := githubactions.GetInput("missing_info_issue_template")
	if missingInfoIssueTemplate == "" {
		githubactions.Fatalf("missing_info_issue_template input is required")
	}

	nonCompliantEmailTemplate := githubactions.GetInput("non_compliant_email_template")
	if nonCompliantEmailTemplate == "" {
		githubactions.Fatalf("non_compliant_email_template input is required")
	}

	org := githubactions.GetInput("org")
	if org == "" {
		githubactions.Fatalf("org input is required")
	}

	outOfComplianceCLIEmailTemplate := githubactions.GetInput("out_of_compliance_cli_email_template")
	if outOfComplianceCLIEmailTemplate == "" {
		githubactions.Fatalf("out_of_compliance_cli_email_template input is required")
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

	emassPromotionAppIDInt64, err := strconv.ParseInt(emassPromotionAppID, 10, 64)
	if err != nil {
		githubactions.Fatalf("emass_promotion_app_id input must be an integer")
	}

	emassPromotionInstallationIDInt64, err := strconv.ParseInt(emassPromotionInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("emass_promotion_installation_id input must be an integer")
	}

	verifyScansAppIDInt64, err := strconv.ParseInt(verifyScansAppID, 10, 64)
	if err != nil {
		githubactions.Fatalf("verify_scans_app_id input must be an integer")
	}

	verifyScansInstallationIDInt64, err := strconv.ParseInt(verifyScansInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("verify_scans_installation_id input must be an integer")
	}

	return &input{
		adminToken:                      adminToken,
		daysToScan:                      daysToScan,
		emassPromotionAppID:             emassPromotionAppIDInt64,
		emassPromotionPrivateKey:        []byte(emassPromotionPrivateKey),
		emassPromotionInstallationID:    emassPromotionInstallationIDInt64,
		gmailFrom:                       gmailFrom,
		gmailUser:                       gmailUser,
		gmailPassword:                   gmailPassword,
		missingInfoEmailTemplate:        missingInfoEmailTemplate,
		missingInfoIssueTemplate:        missingInfoIssueTemplate,
		nonCompliantEmailTemplate:       nonCompliantEmailTemplate,
		org:                             org,
		outOfComplianceCLIEmailTemplate: outOfComplianceCLIEmailTemplate,
		verifyScansAppID:                verifyScansAppIDInt64,
		verifyScansPrivateKey:           []byte(verifyScansPrivateKey),
		verifyScansInstallationID:       verifyScansInstallationIDInt64,
	}
}
