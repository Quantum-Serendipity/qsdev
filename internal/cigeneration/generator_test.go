package cigeneration

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateWorkflow_AllTools(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     true,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)

	// Verify valid YAML structure.
	if !strings.Contains(content, "name: Security") {
		t.Error("missing workflow name")
	}
	if !strings.Contains(content, "on:") {
		t.Error("missing on: trigger")
	}
	if !strings.Contains(content, "jobs:") {
		t.Error("missing jobs: block")
	}

	// All six jobs should be present.
	expectedJobs := []string{
		"lint-sast:",
		"secret-scan:",
		"vulnerability-scan:",
		"container-security:",
		"security-review:",
		"license-compliance:",
	}
	for _, job := range expectedJobs {
		if !strings.Contains(content, job) {
			t.Errorf("expected job %q in output", job)
		}
	}

	// File path.
	if files[0].Path != ".github/workflows/security.yml" {
		t.Errorf("expected path .github/workflows/security.yml, got %q", files[0].Path)
	}
}

func TestGenerateWorkflow_MinimalTools(t *testing.T) {
	// Use an empty registry — only harden-runner and checkout registered.
	registry := NewStepRegistry()
	registry.Register("harden-runner", &HardenRunnerContributor{})
	registry.Register("checkout", &CheckoutContributor{})

	cfg := GenerateConfig{
		Platform: PlatformGitHubActions,
	}

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	// With only infrastructure steps and no tool steps, output should be nil.
	if files != nil {
		t.Errorf("expected nil output for minimal (no tool) registry, got %d files", len(files))
	}
}

func TestGenerateWorkflow_NoDocker(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     false,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)

	if strings.Contains(content, "container-security:") {
		t.Error("container-security job should not be present when HasDocker=false")
	}

	// Other jobs should still be present.
	if !strings.Contains(content, "lint-sast:") {
		t.Error("lint-sast job should be present")
	}
}

func TestGenerateWorkflow_NoClaude(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     true,
		HasClaudeCode: false,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)

	if strings.Contains(content, "security-review:") {
		t.Error("security-review job should not be present when HasClaudeCode=false")
	}

	// pull-requests permission should not be present.
	if strings.Contains(content, "pull-requests:") {
		t.Error("pull-requests permission should not be present without Claude Code")
	}

	// Other jobs should still be present.
	if !strings.Contains(content, "container-security:") {
		t.Error("container-security job should be present when HasDocker=true")
	}
}

func TestGenerateWorkflow_SHAPinned(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     true,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	content := string(files[0].Content)

	// Find all uses: lines and verify they have SHA pins.
	shaPattern := regexp.MustCompile(`uses:\s+\S+/\S+@([a-f0-9]{40})`)
	usesPattern := regexp.MustCompile(`uses:\s+(\S+)`)

	usesMatches := usesPattern.FindAllStringSubmatch(content, -1)
	if len(usesMatches) == 0 {
		t.Fatal("no uses: lines found in output")
	}

	for _, match := range usesMatches {
		ref := match[1]
		if !shaPattern.MatchString("uses: " + ref) {
			t.Errorf("uses: reference %q is not SHA-pinned (expected owner/repo@<40-hex-chars>)", ref)
		}
	}
}

func TestGenerateWorkflow_HardenRunnerFirst(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     true,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	content := string(files[0].Content)

	// For each job block, verify Harden Runner is the first step.
	jobs := strings.Split(content, "    steps:\n")
	for i, block := range jobs {
		if i == 0 {
			// Header, not a job block.
			continue
		}
		// Find first step name.
		lines := strings.Split(block, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- name: ") {
				stepName := strings.TrimPrefix(trimmed, "- name: ")
				if stepName != "Harden Runner" {
					t.Errorf("job block %d: first step is %q, expected 'Harden Runner'", i, stepName)
				}
				break
			}
		}
	}
}

func TestGenerateWorkflow_PlatformNone(t *testing.T) {
	cfg := GenerateConfig{
		Platform: PlatformNone,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil for PlatformNone, got %d files", len(files))
	}
}

func TestGenerateWorkflow_GitLab(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitLabCI,
		HasDocker:     true,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Path != ".gitlab-ci.yml" {
		t.Errorf("expected path .gitlab-ci.yml, got %q", files[0].Path)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "stages:") {
		t.Error("missing stages: block")
	}
	if !strings.Contains(content, "lint-sast:") {
		t.Error("missing lint-sast job")
	}
	if !strings.Contains(content, "script:") {
		t.Error("missing script: blocks")
	}
	if !strings.Contains(content, "stage: test") {
		t.Error("missing test stage assignment")
	}
}

func TestGenerateWorkflow_Permissions(t *testing.T) {
	t.Run("with docker has id-token", func(t *testing.T) {
		cfg := GenerateConfig{
			Platform:      PlatformGitHubActions,
			HasDocker:     true,
			HasClaudeCode: false,
		}
		registry := DefaultStepRegistry()

		files, err := GenerateWorkflow(cfg, registry)
		if err != nil {
			t.Fatalf("GenerateWorkflow returned error: %v", err)
		}

		content := string(files[0].Content)
		if !strings.Contains(content, "id-token: write") {
			t.Error("expected id-token: write permission when Docker is enabled (cosign)")
		}
	})

	t.Run("with claude has pull-requests", func(t *testing.T) {
		cfg := GenerateConfig{
			Platform:      PlatformGitHubActions,
			HasDocker:     false,
			HasClaudeCode: true,
		}
		registry := DefaultStepRegistry()

		files, err := GenerateWorkflow(cfg, registry)
		if err != nil {
			t.Fatalf("GenerateWorkflow returned error: %v", err)
		}

		content := string(files[0].Content)
		if !strings.Contains(content, "pull-requests: write") {
			t.Error("expected pull-requests: write permission for Claude Code review")
		}
	})
}

func TestGenerateWorkflow_ScheduleTrigger(t *testing.T) {
	cfg := GenerateConfig{
		Platform: PlatformGitHubActions,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	content := string(files[0].Content)

	// License compliance job is always present, so schedule trigger should be too.
	if !strings.Contains(content, "schedule:") {
		t.Error("expected schedule: trigger when license-compliance job exists")
	}
	if !strings.Contains(content, "cron: '0 6 * * 1'") {
		t.Error("expected weekly cron schedule")
	}
}

func TestGenerateWorkflow_JobConditions(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitHubActions,
		HasDocker:     false,
		HasClaudeCode: true,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "if: github.event_name == 'pull_request'") {
		t.Error("security-review job should have PR-only condition")
	}
	if !strings.Contains(content, "if: github.event.schedule != ''") {
		t.Error("license-compliance job should have schedule-only condition")
	}
}

func TestStepRegistry_CollectSteps_Sorted(t *testing.T) {
	registry := NewStepRegistry()

	// Add a contributor with steps at different orders.
	registry.Register("test", stepContributorFunc(func(cfg GenerateConfig) []CIStep {
		return []CIStep{
			{ToolName: "b", JobID: JobLintSAST, Name: "B", Order: 20},
			{ToolName: "a", JobID: JobLintSAST, Name: "A", Order: 5},
			{ToolName: "c", JobID: JobLintSAST, Name: "C", Order: 10},
		}
	}))

	cfg := GenerateConfig{Platform: PlatformGitHubActions}
	steps := registry.CollectSteps(cfg)

	jobSteps := steps[JobLintSAST]
	if len(jobSteps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(jobSteps))
	}
	if jobSteps[0].Name != "A" || jobSteps[1].Name != "C" || jobSteps[2].Name != "B" {
		t.Errorf("steps not sorted by order: got %s, %s, %s", jobSteps[0].Name, jobSteps[1].Name, jobSteps[2].Name)
	}
}

func TestActionRef_String(t *testing.T) {
	ref := ActionRef{Owner: "actions", Repo: "checkout", SHA: "abc123def456", Tag: "v4"}
	got := ref.String()
	want := "actions/checkout@abc123def456"
	if got != want {
		t.Errorf("ActionRef.String() = %q, want %q", got, want)
	}
}

func TestActionRef_Comment(t *testing.T) {
	ref := ActionRef{Owner: "actions", Repo: "checkout", SHA: "abc123", Tag: "v4.2.2"}
	got := ref.Comment()
	want := "# v4.2.2"
	if got != want {
		t.Errorf("ActionRef.Comment() = %q, want %q", got, want)
	}
}

func TestGenerateWorkflow_GitLabNoDocker(t *testing.T) {
	cfg := GenerateConfig{
		Platform:      PlatformGitLabCI,
		HasDocker:     false,
		HasClaudeCode: false,
	}
	registry := DefaultStepRegistry()

	files, err := GenerateWorkflow(cfg, registry)
	if err != nil {
		t.Fatalf("GenerateWorkflow returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := string(files[0].Content)
	if strings.Contains(content, "container-security:") {
		t.Error("container-security should not appear without Docker")
	}
	if strings.Contains(content, "security-review:") {
		t.Error("security-review should not appear without Claude Code")
	}
}

// stepContributorFunc adapts a function to StepContributor for test convenience.
type stepContributorFunc func(GenerateConfig) []CIStep

func (f stepContributorFunc) CISteps(cfg GenerateConfig) []CIStep { return f(cfg) }
