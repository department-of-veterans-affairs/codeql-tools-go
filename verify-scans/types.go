package main

type input struct {
	adminToken                      string
	daysToScan                      string
	emassPromotionAppID             int64
	emassPromotionPrivateKey        []byte
	emassPromotionInstallationID    int64
	gmailFrom                       string
	gmailUser                       string
	gmailPassword                   string
	missingInfoEmailTemplate        string
	missingInfoIssueTemplate        string
	nonCompliantEmailTemplate       string
	org                             string
	outOfComplianceCLIEmailTemplate string
	verifyScansAppID                int64
	verifyScansPrivateKey           []byte
	verifyScansInstallationID       int64
}
