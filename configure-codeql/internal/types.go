package internal

type Input struct {
	AdminToken                    string
	ConfigureCodeQLAppID          int64
	ConfigureCodeQLPrivateKey     []byte
	ConfigureCodeQLInstallationID int64
	Org                           string
	PullRequestBody               string
	Repo                          string
	VerifyScansAppID              int64
	VerifyScansPrivateKey         []byte
	VerifyScansInstallationID     int64
}

type DefaultCodeScanning struct {
	State string `json:"state"`
}

type AnalysisTemplate struct {
	Name string `yaml:"name"`
	On   On     `yaml:"on"`
	Jobs Jobs   `yaml:"jobs"`
}

type On struct {
	Push             Branch      `yaml:"push"`
	PullRequest      Branch      `yaml:"pull_request"`
	Schedule         []Cron      `yaml:"schedule"`
	WorkflowDispatch interface{} `yaml:"workflow_dispatch"`
}

type Branch struct {
	Branches []string `yaml:"branches"`
}

type Cron struct {
	Cron string `yaml:"cron"`
}

type Jobs struct {
	Analyze Job `yaml:"analyze"`
}

type Job struct {
	Name        string            `yaml:"name"`
	RunsOn      string            `yaml:"runs-on"`
	Concurrency string            `yaml:"concurrency"`
	Permissions map[string]string `yaml:"permissions"`
	Strategy    Strategy          `yaml:"strategy"`
	Steps       []Step            `yaml:"steps"`
}

type Strategy struct {
	FailFast bool   `yaml:"fail-fast"`
	Matrix   Matrix `yaml:"matrix"`
}

type Matrix struct {
	Language []string `yaml:"language"`
}

type Step struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}

type EMASS struct {
	SystemID         int64  `json:"systemID"`
	SystemName       string `json:"systemName"`
	SystemOwnerName  string `json:"systemOwnerName"`
	SystemOwnerEmail string `json:"systemOwnerEmail"`
}
