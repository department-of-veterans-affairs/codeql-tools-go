package internal

import (
	"context"
	"time"

	"github.com/google/go-github/v52/github"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	Context context.Context

	AdminGitHubClient       *github.Client
	EMASSGithubClient       *github.Client
	VerifyScansGithubClient *github.Client

	Config *Input
	Logger *log.Entry
}

type Input struct {
	AdminToken                      string
	DaysToScan                      int
	EMASSPromotionAppID             int64
	EMASSPromotionPrivateKey        []byte
	EMASSPromotionInstallationID    int64
	EMASSSystemListPath             string
	EMASSSystemListRepo             string
	GmailFrom                       string
	GmailUser                       string
	GmailPassword                   string
	MissingInfoEmailTemplate        string
	MissingInfoIssueTemplate        string
	NonCompliantEmailTemplate       string
	Org                             string
	OutOfComplianceCLIEmailTemplate string
	SecondaryEmail                  string
	VerifyScansAppID                int64
	VerifyScansPrivateKey           []byte
	VerifyScansInstallationID       int64
}

type EMASSConfig struct {
	SystemID         int64  `json:"systemID"`
	SystemName       string `json:"systemName"`
	SystemOwnerEmail string `json:"systemOwnerEmail"`
	SystemOwnerName  string `json:"systemOwnerName"`
}

type CodeQLConfig struct {
	ExcludedLanguages []string          `yaml:"excluded_languages"`
	BuildCommands     map[string]string `yaml:"build_commands"`
}

type codeQLDatabase struct {
	Language  string    `json:"language"`
	CreatedAt time.Time `json:"created_at"`
}
type Analyses struct {
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
