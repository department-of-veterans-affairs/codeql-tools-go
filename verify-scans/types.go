package main

import "time"

var SupportedCodeQLLanguages = []string{
	"c",
	"cpp",
	"csharp",
	"go",
	"java",
	"kotlin",
	"javascript",
	"python",
	"ruby",
	"typescript",
	"swift",
}

type input struct {
	adminToken                      string
	daysToScan                      int
	emassPromotionAppID             int64
	emassPromotionPrivateKey        []byte
	emassPromotionInstallationID    int64
	emassSystemListPath             string
	emassSystemListRepo             string
	gmailFrom                       string
	gmailUser                       string
	gmailPassword                   string
	missingInfoEmailTemplate        string
	missingInfoIssueTemplate        string
	nonCompliantEmailTemplate       string
	org                             string
	outOfComplianceCLIEmailTemplate string
	secondaryEmail                  string
	verifyScansAppID                int64
	verifyScansPrivateKey           []byte
	verifyScansInstallationID       int64
}

type eMASSConfig struct {
	SystemID         int64  `json:"systemID"`
	SystemName       string `json:"systemName"`
	SystemOwnerEmail string `json:"systemOwnerEmail"`
	SystemOwnerName  string `json:"systemOwnerName"`
}

type codeQLConfig struct {
	ExcludedLanguages []string          `yaml:"excluded_languages"`
	BuildCommands     map[string]string `yaml:"build_commands"`
}

type codeQLDatabase struct {
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
}
type analyses struct {
	Languages []string `json:"languages"`
	Versions  []string `json:"versions"`
}

type analysisResult struct {
	CreatedAt time.Time `json:"created_at"`
	Language  string    `json:"language"`
	Category  string    `json:"category"`
	Tool      struct {
		Version string `json:"version"`
	} `json:"tool"`
}

type analysisRequest struct {
	Ref       string `json:"ref"`
	ToolName  string `json:"tool_name"`
	Direction string `json:"direction"`
	Sort      string `json:"sort"`
}
