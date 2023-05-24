package internal

import (
	"os"
	"strconv"
	"strings"

	"github.com/sethvargo/go-githubactions"
)

var (
	DisableNotifications = os.Getenv("DRY_RUN") == "true"
)

func ParseInput() *Input {
	adminToken := githubactions.GetInput("admin_token")
	if adminToken == "" {
		githubactions.Fatalf("admin_token input is required")
	}

	daysToScanString := githubactions.GetInput("days_to_scan")
	if daysToScanString == "" {
		githubactions.Fatalf("days_to_scan input is required")
	}
	daysToScan, err := strconv.Atoi(daysToScanString)
	if err != nil {
		githubactions.Fatalf("days_to_scan input must be an integer")
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

	emassSystemListRepo := githubactions.GetInput("emass_system_list_repo")
	if emassSystemListRepo == "" {
		githubactions.Fatalf("emass_system_list_repo input is required")
	}

	emassSystemListPath := githubactions.GetInput("emass_system_list_path")
	if emassSystemListPath == "" {
		githubactions.Fatalf("emass_system_list_path input is required")
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

	repo := githubactions.GetInput("repo")

	secondaryEmail := githubactions.GetInput("secondary_email")
	if secondaryEmail == "" {
		githubactions.Fatalf("secondary_email input is required")
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

	return &Input{
		AdminToken:                      adminToken,
		DaysToScan:                      daysToScan,
		EMASSPromotionAppID:             emassPromotionAppIDInt64,
		EMASSPromotionPrivateKey:        []byte(emassPromotionPrivateKey),
		EMASSPromotionInstallationID:    emassPromotionInstallationIDInt64,
		EMASSSystemListPath:             emassSystemListPath,
		EMASSSystemListRepo:             strings.ToLower(emassSystemListRepo),
		GmailFrom:                       gmailFrom,
		GmailUser:                       gmailUser,
		GmailPassword:                   gmailPassword,
		MissingInfoEmailTemplate:        missingInfoEmailTemplate,
		MissingInfoIssueTemplate:        missingInfoIssueTemplate,
		NonCompliantEmailTemplate:       nonCompliantEmailTemplate,
		Org:                             strings.ToLower(org),
		OutOfComplianceCLIEmailTemplate: outOfComplianceCLIEmailTemplate,
		Repo:                            strings.ToLower(repo),
		SecondaryEmail:                  secondaryEmail,
		VerifyScansAppID:                verifyScansAppIDInt64,
		VerifyScansPrivateKey:           []byte(verifyScansPrivateKey),
		VerifyScansInstallationID:       verifyScansInstallationIDInt64,
	}
}
