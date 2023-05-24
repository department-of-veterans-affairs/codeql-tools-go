package internal

import (
	"strconv"
	"strings"

	"github.com/sethvargo/go-githubactions"
)

func ParseInput() *Input {
	adminToken := githubactions.GetInput("admin_token")
	if adminToken == "" {
		githubactions.Fatalf("admin_token input is required")
	}

	daysToScan := githubactions.GetInput("days_to_scan")
	if daysToScan == "" {
		githubactions.Fatalf("days_to_scan input is required")
	}

	emassOrg := githubactions.GetInput("emass_org")
	if emassOrg == "" {
		githubactions.Fatalf("emass_org input is required")
	}

	emassOrganizationInstallationID := githubactions.GetInput("emass_organization_installation_id")
	if emassOrganizationInstallationID == "" {
		githubactions.Fatalf("emass_organization_installation_id input is required")
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

	emassSystemListOrg := githubactions.GetInput("emass_system_list_org")
	if emassSystemListOrg == "" {
		githubactions.Fatalf("emass_system_list_path input is required")
	}

	emassSystemListPath := githubactions.GetInput("emass_system_list_path")
	if emassSystemListPath == "" {
		githubactions.Fatalf("emass_system_list_path input is required")
	}

	emassSystemListRepo := githubactions.GetInput("emass_system_list_repo")
	if emassSystemListRepo == "" {
		githubactions.Fatalf("emass_system_list_repo input is required")
	}

	org := githubactions.GetInput("org")

	repo := githubactions.GetInput("repo")

	daysToScanInt, err := strconv.Atoi(daysToScan)
	if err != nil {
		githubactions.Fatalf("days_to_scan input must be an integer")
	}

	emassOrganizationInstallationIDInt64, err := strconv.ParseInt(emassOrganizationInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("emass_organization_installation_id input must be an integer")
	}

	emassPromotionAppIDInt64, err := strconv.ParseInt(emassPromotionAppID, 10, 64)
	if err != nil {
		githubactions.Fatalf("emass_promotion_app_id input must be an integer")
	}

	emassPromotionInstallationIDInt64, err := strconv.ParseInt(emassPromotionInstallationID, 10, 64)
	if err != nil {
		githubactions.Fatalf("emass_promotion_installation_id input must be an integer")
	}

	return &Input{
		AdminToken:                   adminToken,
		DaysToScan:                   daysToScanInt,
		EMASSOrg:                     strings.ToLower(emassOrg),
		EMASSOrgInstallationID:       emassOrganizationInstallationIDInt64,
		EMASSPromotionAppID:          emassPromotionAppIDInt64,
		EMASSPromotionPrivateKey:     []byte(emassPromotionPrivateKey),
		EMASSPromotionInstallationID: emassPromotionInstallationIDInt64,
		EMASSSystemListOrg:           strings.ToLower(emassSystemListOrg),
		EMASSSystemListPath:          strings.ToLower(emassSystemListPath),
		EMASSSystemListRepo:          strings.ToLower(emassSystemListRepo),
		Org:                          strings.ToLower(org),
		Repo:                         strings.ToLower(repo),
	}
}
