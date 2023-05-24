package internal

import "time"

type Input struct {
	AdminToken                   string
	DaysToScan                   int
	EMASSOrg                     string
	EMASSOrgInstallationID       int64
	EMASSPromotionAppID          int64
	EMASSPromotionPrivateKey     []byte
	EMASSPromotionInstallationID int64
	EMASSSystemListOrg           string
	EMASSSystemListPath          string
	EMASSSystemListRepo          string
	Org                          string
	Repo                         string
}

type EMASSConfig struct {
	SystemID         int64  `json:"systemID"`
	SystemName       string `json:"systemName"`
	SystemOwnerEmail string `json:"systemOwnerEmail"`
	SystemOwnerName  string `json:"systemOwnerName"`
}

type codeQLDatabase struct {
	CreatedAt time.Time `json:"created_at"`
	Language  string    `json:"language"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
}

type analysisResult struct {
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	ID        int64     `json:"id"`
	Language  string    `json:"language"`
	Ref       string    `json:"ref"`
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
