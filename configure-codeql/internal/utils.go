package internal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"gopkg.in/yaml.v3"
)

func GenerateRandomSuffix(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

func GeneratePullRequestBody(template, owner, repo, branch string, languages []string) string {
	return fmt.Sprintf("This pull request adds a CodeQL workflow to your repository. This workflow will analyze your code for vulnerabilities in the following languages: %s", strings.Join(languages, ", "))
}

func GenerateCodeQLWorkflow(languages []string, defaultBranch string) (string, error) {
	workflow := AnalysisTemplate{
		Name: "CodeQL",
		On: On{
			Push: Branch{
				Branches: []string{defaultBranch},
			},
			PullRequest: Branch{
				Branches: []string{defaultBranch},
			},
			Schedule: []Cron{
				{
					Cron: GenerateRandomWeeklyCron(),
				},
			},
			WorkflowDispatch: nil,
		},
		Jobs: Jobs{
			Analyze: Job{
				Name:        "Analyze",
				RunsOn:      "ubuntu-latest",
				Concurrency: "${{ github.workflow }}-${{ github.ref }}",
				Permissions: map[string]string{
					"actions":         "read",
					"contents":        "read",
					"security-events": "write",
				},
				Strategy: Strategy{
					FailFast: false,
					Matrix: Matrix{
						Language: languages,
					},
				},
				Steps: []Step{
					{
						Name: "Run Code Scanning",
						Uses: "department-of-veterans-affairs/codeql-tools/codeql-analysis@main",
						With: map[string]string{
							"languages": "${{ matrix.Language }}",
						},
					},
				},
			},
		},
	}

	workflowBytes, err := yaml.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow: %w", err)
	}

	return string(workflowBytes), nil
}

func GenerateEMASSJSON() (string, error) {
	emassJSON := &EMASS{
		SystemID:         0,
		SystemName:       "<system_name>",
		SystemOwnerName:  "<full_name>",
		SystemOwnerEmail: "<email>",
	}

	emassBytes, err := json.Marshal(emassJSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal emass json: %w", err)
	}

	return string(emassBytes), nil
}

func GenerateRandomWeeklyCron() string {
	minute := rand.Intn(60)
	hour := rand.Intn(24)
	dayOfWeek := rand.Intn(7)

	return fmt.Sprintf("%d %d * * %d", minute, hour, dayOfWeek)
}

func Contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}

	return false
}
