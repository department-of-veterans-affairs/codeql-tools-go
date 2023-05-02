package main

type input struct {
	adminToken                    string
	configureCodeQLAppID          int64
	configureCodeQLPrivateKey     []byte
	configureCodeQLInstallationID int64
	org                           string
	pullRequestBody               string
	verifyScansAppID              int64
	verifyScansPrivateKey         []byte
	verifyScansInstallationID     int64
}

type defaultCodeScanning struct {
	State string `json:"state"`
}

type analysisTemplate struct {
	Name string `yaml:"name"`
	On   on     `yaml:"on"`
	Jobs jobs   `yaml:"jobs"`
}

type on struct {
	Push             branch      `yaml:"push"`
	PullRequest      branch      `yaml:"pull_request"`
	Schedule         []cron      `yaml:"schedule"`
	WorkflowDispatch interface{} `yaml:"workflow_dispatch"`
}

type branch struct {
	Branches []string `yaml:"branches"`
}

type cron struct {
	Cron string `yaml:"cron"`
}

type jobs struct {
	Analyze job `yaml:"analyze"`
}

type job struct {
	Name        string            `yaml:"name"`
	RunsOn      string            `yaml:"runs-on"`
	Concurrency string            `yaml:"concurrency"`
	Permissions map[string]string `yaml:"permissions"`
	Strategy    strategy          `yaml:"strategy"`
	Steps       []step            `yaml:"steps"`
}

type strategy struct {
	FailFast bool   `yaml:"fail-fast"`
	Matrix   matrix `yaml:"matrix"`
}

type matrix struct {
	Language []string `yaml:"language"`
}

type step struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
}
