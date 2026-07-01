package claudecode_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	refcc "github.com/Quantum-Serendipity/qsdev/pkg/aiframework/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework/contracttest"
)

func newAdapter() *refcc.Adapter {
	cfg := ccaddon.Config{DefaultPermissions: ccaddon.PermissionPresetStandard}
	return refcc.New(cfg, nil)
}

// presentRoot returns a temp dir populated with Claude Code markers.
func presentRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatalf("creating .claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("writing CLAUDE.md: %v", err)
	}
	return root
}

func TestFrameworkID(t *testing.T) {
	t.Parallel()
	if id := newAdapter().FrameworkID(); id != aiframework.ClaudeCode {
		t.Errorf("FrameworkID() = %q, want %q", id, aiframework.ClaudeCode)
	}
}

func TestEnforcementTier(t *testing.T) {
	t.Parallel()
	if tier := newAdapter().EnforcementTier(); tier != aiframework.TierHook {
		t.Errorf("EnforcementTier() = %v, want TierHook", tier)
	}
}

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		root     func(t *testing.T) string
		detected bool
	}{
		{"present", presentRoot, true},
		{"absent", func(t *testing.T) string { t.Helper(); return t.TempDir() }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			det, err := newAdapter().Detect(tt.root(t))
			if err != nil {
				t.Fatalf("Detect() error: %v", err)
			}
			if det.Detected != tt.detected {
				t.Errorf("Detected = %v, want %v", det.Detected, tt.detected)
			}
			if tt.detected && len(det.Evidence) == 0 {
				t.Error("expected evidence for detected project")
			}
		})
	}
}

func TestMarkers(t *testing.T) {
	t.Parallel()
	if len(newAdapter().Markers()) == 0 {
		t.Error("Markers() returned empty slice")
	}
}

// TestTranslatePermissions checks that the policy's explicit allow/deny/ask
// rules are merged into the real rendered settings.json at the right keys.
func TestTranslatePermissions(t *testing.T) {
	t.Parallel()

	policy := &aiframework.PermissionPolicy{
		Preset:     "standard",
		AllowRules: []aiframework.PermissionRule{{Pattern: "Bash(make build)"}},
		DenyRules:  []aiframework.PermissionRule{{Pattern: "Bash(curl *)", Reason: "no remote fetch"}},
		AskRules:   []aiframework.PermissionRule{{Pattern: "Bash(docker *)"}},
	}

	arts, err := newAdapter().TranslatePermissions(context.Background(), policy)
	if err != nil {
		t.Fatalf("TranslatePermissions() error: %v", err)
	}
	if arts.ActiveTier != aiframework.TierHook {
		t.Errorf("ActiveTier = %v, want TierHook", arts.ActiveTier)
	}
	if len(arts.GeneratedFiles) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(arts.GeneratedFiles))
	}
	f := arts.GeneratedFiles[0]
	if f.Path != ".claude/settings.json" {
		t.Errorf("path = %q, want .claude/settings.json", f.Path)
	}

	var settings ccaddon.SettingsJSON
	if err := json.Unmarshal(f.Content, &settings); err != nil {
		t.Fatalf("rendered settings.json is not valid JSON: %v", err)
	}
	if !slices.Contains(settings.Permissions.Allow, "Bash(make build)") {
		t.Errorf("allow rules missing custom pattern: %v", settings.Permissions.Allow)
	}
	if !slices.Contains(settings.Permissions.Deny, "Bash(curl *)") {
		t.Errorf("deny rules missing custom pattern: %v", settings.Permissions.Deny)
	}
	if !slices.Contains(settings.Permissions.Ask, "Bash(docker *)") {
		t.Errorf("ask rules missing custom pattern: %v", settings.Permissions.Ask)
	}
}

// TestTranslateIgnorePatterns checks each ignore pattern becomes a Read() deny.
func TestTranslateIgnorePatterns(t *testing.T) {
	t.Parallel()

	files, err := newAdapter().TranslateIgnorePatterns(context.Background(), []aiframework.IgnorePattern{
		{Pattern: "./build/**", Category: aiframework.CategoryBinary},
		{Pattern: "./vendor/**", Category: aiframework.CategoryVendor},
	})
	if err != nil {
		t.Fatalf("TranslateIgnorePatterns() error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	var settings ccaddon.SettingsJSON
	if err := json.Unmarshal(files[0].Content, &settings); err != nil {
		t.Fatalf("invalid settings.json: %v", err)
	}
	for _, want := range []string{"Read(./build/**)", "Read(./vendor/**)"} {
		if !slices.Contains(settings.Permissions.Deny, want) {
			t.Errorf("deny rules missing %q: %v", want, settings.Permissions.Deny)
		}
	}
}

// TestInjectCredentials checks that real exclusion data is produced, that
// required keys become passthrough references, and that no artifact leaks a
// sandbox filter glob or a secret value.
func TestInjectCredentials(t *testing.T) {
	t.Parallel()

	scope := &aiframework.CredentialScope{
		AWSProfile:     "default",
		APIKeys:        []aiframework.APIKeyRequirement{{Provider: "openai", EnvVar: "OPENAI_API_KEY", Required: true}},
		SandboxFilters: aiframework.DefaultSandboxFilters(),
	}

	arts, err := newAdapter().InjectCredentials(context.Background(), scope)
	if err != nil {
		t.Fatalf("InjectCredentials() error: %v", err)
	}
	if len(arts.ExcludePaths) == 0 {
		t.Error("expected non-empty ExcludePaths")
	}
	if !slices.Contains(arts.ExcludePaths, "~/.aws/**") {
		t.Errorf("AWS scope did not add ~/.aws/** : %v", arts.ExcludePaths)
	}
	if got := arts.EnvVars["OPENAI_API_KEY"]; got != "${OPENAI_API_KEY}" {
		t.Errorf("EnvVars[OPENAI_API_KEY] = %q, want ${OPENAI_API_KEY}", got)
	}
	for _, f := range arts.GeneratedFiles {
		for _, pattern := range scope.SandboxFilters {
			if strings.Contains(string(f.Content), pattern) {
				t.Errorf("generated file %q leaked filter %q", f.Path, pattern)
			}
		}
	}
}

func TestReportGaps(t *testing.T) {
	t.Parallel()

	policy := &aiframework.PermissionPolicy{
		DenyRules: []aiframework.PermissionRule{{Pattern: "Bash(rm -rf /)", Reason: "wipe"}},
	}
	gaps := newAdapter().ReportGaps(context.Background(), policy)
	if len(gaps) == 0 {
		t.Fatal("expected at least one gap")
	}
	for _, g := range gaps {
		if g.Description == "" || g.Mitigation == "" {
			t.Errorf("gap missing description/mitigation: %+v", g)
		}
	}
}

// TestRenderContextBudget exercises the CalculateContextBudget delegation: an
// empty project under a declared model is within budget and renders cleanly.
func TestRenderContextBudget(t *testing.T) {
	t.Parallel()

	input := &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		Permissions: &aiframework.PermissionPolicy{Preset: "standard"},
		Model:       &aiframework.ModelPreferences{PreferredModel: "sonnet"},
	}
	files, err := newAdapter().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("Render() produced no files")
	}
}

func TestContractSuite(t *testing.T) {
	t.Parallel()

	a := newAdapter()
	contracttest.RunAllContractTests(t, contracttest.ContractAdapters{
		Detection: a,
		Config:    a,
		Tools:     a,
	}, contracttest.ContractFixtures{
		PresentRoot: presentRoot(t),
		AbsentRoot:  t.TempDir(),
		PolicyInput: &aiframework.PolicyInput{
			ProjectRoot: t.TempDir(),
			Permissions: &aiframework.PermissionPolicy{
				Preset:    "standard",
				DenyRules: []aiframework.PermissionRule{{Pattern: "Bash(rm -rf *)", Reason: "destructive"}},
			},
		},
	})
}
