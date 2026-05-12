package claudecode_test

import (
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// newTestRegistry creates a registry and registers the given mock modules.
func newTestRegistry(t *testing.T, mocks ...*ecosystem.MockModule) *ecosystem.Registry {
	t.Helper()
	reg := ecosystem.NewRegistry()
	for _, m := range mocks {
		if err := reg.Register(m); err != nil {
			t.Fatalf("registering mock %q: %v", m.NameVal, err)
		}
	}
	return reg
}

func goMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "go modules"},
		},
	}
}

func pythonMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "pip"},
		},
	}
}

func jsMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "javascript",
		DisplayNameVal: "JavaScript/TypeScript",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "npm"},
		},
	}
}

func rustMock() *ecosystem.MockModule {
	return &ecosystem.MockModule{
		NameVal:        "rust",
		DisplayNameVal: "Rust",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "cargo"},
		},
	}
}

// requireContains asserts that s contains the substring sub.
func requireContains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("output does not contain %q\n\nFull output:\n%s", sub, s)
	}
}

// requireNotContains asserts that s does not contain the substring sub.
func requireNotContains(t *testing.T, s, sub string) {
	t.Helper()
	if strings.Contains(s, sub) {
		t.Errorf("output unexpectedly contains %q", sub)
	}
}

func TestGenerateClaudeMd_MarkersPresent(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "myproject",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "<!-- BEGIN GENERATED SECTION")
	requireContains(t, content, "<!-- END GENERATED SECTION -->")

	// BEGIN must come before END.
	beginIdx := strings.Index(content, "<!-- BEGIN GENERATED SECTION")
	endIdx := strings.Index(content, "<!-- END GENERATED SECTION -->")
	if beginIdx >= endIdx {
		t.Errorf("BEGIN marker (at %d) should appear before END marker (at %d)", beginIdx, endIdx)
	}
}

func TestGenerateClaudeMd_GoConventions(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "myproject",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "### Go")
	requireContains(t, content, "fmt.Errorf")
	requireContains(t, content, "table-driven tests")
	requireContains(t, content, "internal/")
	requireContains(t, content, "context.Context")
}

func TestGenerateClaudeMd_MultiLanguage(t *testing.T) {
	reg := newTestRegistry(t, goMock(), pythonMock())
	answers := types.WizardAnswers{
		ProjectName: "polyglot",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "### Go")
	requireContains(t, content, "### Python")
	requireContains(t, content, "type hints")
	requireContains(t, content, "pathlib.Path")
}

func TestGenerateClaudeMd_SecurityAlwaysPresent(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "## Security")
	requireContains(t, content, "devenv.nix")
	requireContains(t, content, "ripsecrets")
	requireContains(t, content, "Lock files")
}

func TestGenerateClaudeMd_BuildTestLintCommands(t *testing.T) {
	reg := newTestRegistry(t, goMock(), pythonMock())
	answers := types.WizardAnswers{
		ProjectName: "cmdtest",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	// Build commands.
	requireContains(t, content, "go build ./...")

	// Test commands.
	requireContains(t, content, "go test ./...")
	requireContains(t, content, "python -m pytest")

	// Lint commands.
	requireContains(t, content, "golangci-lint run")
	requireContains(t, content, "ruff check .")
}

func TestGenerateClaudeMd_EmptyFieldsNoError(t *testing.T) {
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error with empty answers: %v", err)
	}

	content := string(got.Content)

	// Should still have the basic structure.
	requireContains(t, content, "# CLAUDE.md")
	requireContains(t, content, "## Security")

	// Should not have empty command sections.
	requireNotContains(t, content, "## Build Commands")
	requireNotContains(t, content, "## Test Commands")
	requireNotContains(t, content, "## Lint Commands")
}

func TestGenerateClaudeMd_CustomInstructionsBelowMarkers(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "myproject",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	endIdx := strings.Index(content, "<!-- END GENERATED SECTION -->")
	customIdx := strings.Index(content, "## Custom Instructions")

	if endIdx < 0 {
		t.Fatal("END marker not found")
	}
	if customIdx < 0 {
		t.Fatal("Custom Instructions section not found")
	}
	if customIdx <= endIdx {
		t.Errorf("Custom Instructions (at %d) should appear after END marker (at %d)", customIdx, endIdx)
	}
}

func TestGenerateClaudeMd_FileMetadata(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "metacheck",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Path != "CLAUDE.md" {
		t.Errorf("Path = %q, want %q", got.Path, "CLAUDE.md")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.SectionMarker {
		t.Errorf("Strategy = %v, want SectionMarker", got.Strategy)
	}
}

func TestGenerateClaudeMd_JavaScriptConventions(t *testing.T) {
	reg := newTestRegistry(t, jsMock())
	answers := types.WizardAnswers{
		ProjectName: "frontend",
		Languages:   []types.LanguageChoice{{Name: "javascript"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "### JavaScript / TypeScript")
	requireContains(t, content, "strict TypeScript")
	requireContains(t, content, "unknown")
}

func TestGenerateClaudeMd_RustConventions(t *testing.T) {
	reg := newTestRegistry(t, rustMock())
	answers := types.WizardAnswers{
		ProjectName: "syslib",
		Languages:   []types.LanguageChoice{{Name: "rust"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "### Rust")
	requireContains(t, content, "cargo clippy")
	requireContains(t, content, "&str")
	requireContains(t, content, "Debug")
}

func TestGenerateClaudeMd_PackageManagersInSecurity(t *testing.T) {
	reg := newTestRegistry(t, goMock(), jsMock())
	answers := types.WizardAnswers{
		ProjectName: "fullstack",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
		},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "go modules")
	requireContains(t, content, "npm")
}

func TestGenerateClaudeMd_SecurityHooksEnabled(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "secure",
		Languages:   []types.LanguageChoice{{Name: "go"}},
		Hooks:       types.HookChoices{SafetyBlock: true},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "Safety-block hooks are enabled")
}

func TestGenerateClaudeMd_DefaultDescription(t *testing.T) {
	reg := newTestRegistry(t, goMock(), pythonMock())
	answers := types.WizardAnswers{
		ProjectName: "myapp",
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}

	data := claudecode.BuildClaudeMdData(answers, reg)

	requireContains(t, data.ProjectDescription, "myapp")
	requireContains(t, data.ProjectDescription, "Go")
	requireContains(t, data.ProjectDescription, "Python")
}
