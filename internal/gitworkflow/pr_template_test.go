package gitworkflow

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGeneratePRTemplate_EmptyAnswers(t *testing.T) {
	answers := types.WizardAnswers{}
	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}
	if f.Path != ".github/pull_request_template.md" {
		t.Errorf("path = %q, want .github/pull_request_template.md", f.Path)
	}
	if f.Mode != 0o644 {
		t.Errorf("mode = %o, want 644", f.Mode)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("strategy = %v, want Overwrite", f.Strategy)
	}

	content := string(f.Content)

	// Base sections must be present.
	for _, section := range []string{
		"## Summary",
		"## Type of Change",
		"## Testing",
		"## Breaking Changes",
		"## Reviewer Notes",
	} {
		if !strings.Contains(content, section) {
			t.Errorf("content missing section %q", section)
		}
	}

	// Security checklist should NOT be present with empty answers.
	if strings.Contains(content, "## Security Checklist") {
		t.Error("security checklist should not appear with empty answers")
	}
}

func TestGeneratePRTemplate_GoEcosystem(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.22"},
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "`go vet ./...`") {
		t.Error("content missing go vet checklist item")
	}
	if !strings.Contains(content, "`go test ./...`") {
		t.Error("content missing go test checklist item")
	}
}

func TestGeneratePRTemplate_MultipleEcosystems(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
			{Name: "rust"},
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)

	expectations := []struct {
		lang  string
		check string
	}{
		{"go", "`go vet ./...`"},
		{"go", "`go test ./...`"},
		{"python", "Type hints added"},
		{"python", "Linter passes"},
		{"rust", "`cargo clippy`"},
		{"rust", "`cargo test`"},
	}

	for _, exp := range expectations {
		if !strings.Contains(content, exp.check) {
			t.Errorf("content missing %s checklist item: %q", exp.lang, exp.check)
		}
	}
}

func TestGeneratePRTemplate_SecurityToolsEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		EnabledTools: map[string]bool{
			"semgrep": true,
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "## Security Checklist") {
		t.Error("security checklist should appear when security tools are enabled")
	}
	if !strings.Contains(content, "No secrets or credentials") {
		t.Error("security checklist missing secrets check")
	}
}

func TestGeneratePRTemplate_ComplianceLevelTriggersSecurityChecklist(t *testing.T) {
	answers := types.WizardAnswers{
		ComplianceLevel: "soc2",
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "## Security Checklist") {
		t.Error("security checklist should appear when compliance level is set")
	}
}

func TestGeneratePRTemplate_DockerfileDetected(t *testing.T) {
	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			HasDockerfile: true,
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "Docker image builds") {
		t.Error("content missing Docker build checklist item")
	}
	if !strings.Contains(content, "Image scanned for vulnerabilities") {
		t.Error("content missing Docker scan checklist item")
	}
}

func TestGeneratePRTemplate_JavascriptEcosystem(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "javascript"},
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "TypeScript types exported correctly") {
		t.Error("content missing TypeScript types checklist item")
	}
	if !strings.Contains(content, "`npm audit`") {
		t.Error("content missing npm audit checklist item")
	}
}

func TestGeneratePRTemplate_JavaEcosystem(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "java"},
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "Build passes") {
		t.Error("content missing Java build checklist item")
	}
	if !strings.Contains(content, "Static analysis clean") {
		t.Error("content missing Java static analysis checklist item")
	}
}

func TestGeneratePRTemplate_DotnetEcosystem(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "dotnet"},
		},
	}

	f, err := GeneratePRTemplate(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(f.Content)
	if !strings.Contains(content, "`dotnet build`") {
		t.Error("content missing dotnet build checklist item")
	}
	if !strings.Contains(content, "`dotnet test`") {
		t.Error("content missing dotnet test checklist item")
	}
}

func TestHasSecurityTools(t *testing.T) {
	tests := []struct {
		name   string
		tools  map[string]bool
		expect bool
	}{
		{"nil map", nil, false},
		{"empty map", map[string]bool{}, false},
		{"semgrep", map[string]bool{"semgrep": true}, true},
		{"gitleaks", map[string]bool{"gitleaks": true}, true},
		{"container-security", map[string]bool{"container-security": true}, true},
		{"unrelated tool", map[string]bool{"commitlint": true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answers := types.WizardAnswers{EnabledTools: tt.tools}
			got := hasSecurityTools(answers)
			if got != tt.expect {
				t.Errorf("hasSecurityTools() = %v, want %v", got, tt.expect)
			}
		})
	}
}
