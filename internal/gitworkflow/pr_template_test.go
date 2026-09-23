package gitworkflow

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
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
		return
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

func TestGeneratePRTemplate_EcosystemChecklist(t *testing.T) {
	tests := []struct {
		name      string
		languages []types.LanguageChoice
		want      []string
	}{
		{
			name:      "go",
			languages: []types.LanguageChoice{{Name: "go", Version: "1.22"}},
			want:      []string{"- [ ] `go vet ./...` passes", "- [ ] `go test ./...` passes"},
		},
		{
			name:      "javascript follows the configured package manager",
			languages: []types.LanguageChoice{{Name: "javascript", PackageManager: "pnpm"}},
			want:      []string{"- [ ] `pnpm test` passes", "- [ ] `pnpm run lint` passes"},
		},
		{
			name:      "multiple ecosystems",
			languages: []types.LanguageChoice{{Name: "go"}, {Name: "python"}, {Name: "rust"}},
			want:      []string{"`go test ./...`", "`cargo test`"},
		},
		{
			name:      "unknown language contributes nothing",
			languages: []types.LanguageChoice{{Name: "not-a-language"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := GeneratePRTemplate(types.WizardAnswers{Languages: tt.languages})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			content := string(f.Content)
			for _, w := range tt.want {
				if !strings.Contains(content, w) {
					t.Errorf("content missing %q:\n%s", w, content)
				}
			}
		})
	}
}

// TestGeneratePRTemplate_EveryCatalogLanguage proves the checklist covers every
// registered ecosystem module — the hardcoded table covered only six languages
// and carried a dead "typescript" case.
func TestGeneratePRTemplate_EveryCatalogLanguage(t *testing.T) {
	for _, mod := range ecosystem.DefaultRegistry().All() {
		cmds := mod.VerificationCommands(ecosystem.ModuleConfig{}).All()
		if len(cmds) == 0 {
			continue
		}
		t.Run(mod.Name(), func(t *testing.T) {
			f, err := GeneratePRTemplate(types.WizardAnswers{Languages: []types.LanguageChoice{{Name: mod.Name()}}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, c := range cmds {
				if !strings.Contains(string(f.Content), "`"+c+"`") {
					t.Errorf("checklist missing %s command %q", mod.Name(), c)
				}
			}
		})
	}
}

func TestVerificationCommands_Dedup(t *testing.T) {
	got := verificationCommands(types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go"}, {Name: "go"}}}, ecosystem.DefaultRegistry())
	seen := map[string]bool{}
	for _, c := range got {
		if seen[c] {
			t.Errorf("duplicate command %q in %v", c, got)
		}
		seen[c] = true
	}
	if len(got) == 0 {
		t.Error("no commands for go")
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
	if !strings.Contains(content, "Container image builds") {
		t.Error("content missing container build checklist item")
	}
	if !strings.Contains(content, "Image scanned for vulnerabilities") {
		t.Error("content missing Docker scan checklist item")
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
