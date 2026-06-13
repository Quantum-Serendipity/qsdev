package cigeneration

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GitLabCIGenerator produces .gitlab-ci.yml.
type GitLabCIGenerator struct{}

// gitlabJobMeta maps CIJobID to GitLab stage and display name.
type gitlabJobMeta struct {
	ID    CIJobID
	Stage string
	Name  string
	Rules string // optional rules: clause
}

var gitlabJobs = []gitlabJobMeta{
	{ID: JobLintSAST, Stage: "test", Name: "lint-sast"},
	{ID: JobSecretScan, Stage: "test", Name: "secret-scan"},
	{ID: JobVulnerabilityScan, Stage: "test", Name: "vulnerability-scan"},
	{ID: JobContainerSecurity, Stage: "build", Name: "container-security"},
	{ID: JobSecurityReview, Stage: "review", Name: "security-review", Rules: "merge_requests"},
	{ID: JobLicenseCompliance, Stage: "compliance", Name: "license-compliance", Rules: "schedules"},
}

// Generate builds the GitLab CI YAML via string builder.
func (g *GitLabCIGenerator) Generate(cfg GenerateConfig, steps map[CIJobID][]CIStep) ([]types.GeneratedFile, error) {
	var b strings.Builder

	b.WriteString("# Security scanning pipeline\n")
	fmt.Fprintf(&b, "# %s — do not edit manually.\n\n", branding.GeneratedBy())

	// Collect active stages in order.
	activeStages := collectActiveStages(steps)
	b.WriteString("stages:\n")
	for _, s := range activeStages {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	b.WriteString("\n")

	// Jobs
	for _, jm := range gitlabJobs {
		jobSteps, ok := steps[jm.ID]
		if !ok {
			continue
		}
		writeGitLabJob(&b, jm, jobSteps)
	}

	content := b.String()
	return []types.GeneratedFile{
		{
			Path:     ".gitlab-ci.yml",
			Content:  []byte(content),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
			Owner:    "ci-generation",
		},
	}, nil
}

// writeGitLabJob emits a single GitLab CI job.
func writeGitLabJob(b *strings.Builder, jm gitlabJobMeta, steps []CIStep) {
	fmt.Fprintf(b, "%s:\n", jm.Name)
	fmt.Fprintf(b, "  stage: %s\n", jm.Stage)
	b.WriteString("  image: ubuntu:latest\n")

	if jm.Rules != "" {
		b.WriteString("  rules:\n")
		switch jm.Rules {
		case "merge_requests":
			b.WriteString("    - if: $CI_PIPELINE_SOURCE == \"merge_request_event\"\n")
		case "schedules":
			b.WriteString("    - if: $CI_PIPELINE_SOURCE == \"schedule\"\n")
		}
	}

	// Build script from steps (skip harden-runner and checkout — GitLab handles these)
	var scripts []string
	for _, step := range steps {
		if step.ToolName == "harden-runner" || step.ToolName == "checkout" {
			continue
		}
		if step.Run != "" {
			scripts = append(scripts, step.Run)
		} else if step.Uses != "" {
			// Convert GitHub Actions usage to a comment + equivalent command hint
			scripts = append(scripts, fmt.Sprintf("echo 'Running %s (GitHub Action: %s)'", step.Name, step.Uses))
		}
	}

	if len(scripts) > 0 {
		b.WriteString("  script:\n")
		for _, s := range scripts {
			fmt.Fprintf(b, "    - %s\n", s)
		}
	}

	// Artifacts for SARIF/report files
	hasArtifacts := false
	for _, step := range steps {
		if step.With != nil {
			if _, ok := step.With["path"]; ok {
				hasArtifacts = true
				break
			}
		}
	}
	if hasArtifacts {
		b.WriteString("  artifacts:\n")
		b.WriteString("    paths:\n")
		for _, step := range steps {
			if step.With != nil {
				if path, ok := step.With["path"]; ok {
					fmt.Fprintf(b, "      - %s\n", path)
				}
			}
		}
		b.WriteString("    when: always\n")
	}

	b.WriteString("\n")
}

// collectActiveStages returns stages that have at least one active job, in order.
func collectActiveStages(steps map[CIJobID][]CIStep) []string {
	stageOrder := []string{"test", "build", "review", "compliance"}
	stageSet := make(map[string]bool)

	for _, jm := range gitlabJobs {
		if _, ok := steps[jm.ID]; ok {
			stageSet[jm.Stage] = true
		}
	}

	var result []string
	for _, s := range stageOrder {
		if stageSet[s] {
			result = append(result, s)
		}
	}
	return result
}
