package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
		VerificationCommandsVal: ecosystem.VerificationCommands{
			Build:  []string{"go build ./..."},
			Test:   []string{"go test ./..."},
			Lint:   []string{"go vet ./...", "golangci-lint run"},
			Format: []string{"gofmt -l ."},
		},
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "go.mod", Ecosystem: "go", VSSupported: false, LockFile: "go.sum", LockFilePolicy: ecosystem.LockFilePolicyRecommended},
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
		VerificationCommandsVal: ecosystem.VerificationCommands{
			Test:      []string{"python -m pytest"},
			Lint:      []string{"ruff check ."},
			TypeCheck: []string{"mypy ."},
			Format:    []string{"ruff format --check ."},
		},
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "requirements.txt", Ecosystem: "pip", VSSupported: true, LockFilePolicy: ecosystem.LockFilePolicyNone},
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
		VerificationCommandsVal: ecosystem.VerificationCommands{
			Build:  []string{"npm run build"},
			Test:   []string{"npm test"},
			Lint:   []string{"npm run lint"},
			Format: []string{"prettier --check ."},
		},
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "package.json", Ecosystem: "npm", VSSupported: true, LockFile: "package-lock.json", LockFilePolicy: ecosystem.LockFilePolicyRequired},
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
		VerificationCommandsVal: ecosystem.VerificationCommands{
			Build:  []string{"cargo build"},
			Test:   []string{"cargo test"},
			Lint:   []string{"cargo clippy -- -D warnings"},
			Format: []string{"cargo fmt -- --check"},
		},
		ManifestFilesVal: []ecosystem.ManifestFileInfo{
			{Path: "Cargo.toml", Ecosystem: "cargo", VSSupported: true, LockFile: "Cargo.lock", LockFilePolicy: ecosystem.LockFilePolicyRecommended},
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

func TestGenerateClaudeMd_GoProject(t *testing.T) {
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
	requireContains(t, content, "go build")
	requireContains(t, content, "qsdev init")
	requireContains(t, content, "qsdev-reference.md")
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
	requireContains(t, content, "go build")
	requireContains(t, content, "pytest")
	requireContains(t, content, "qsdev Commands")
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
	requireContains(t, content, "package guard hook")
	requireContains(t, content, "qsdev enable")
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

func TestGenerateClaudeMd_NoDefaultContentAfterEndMarker(t *testing.T) {
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
	if endIdx < 0 {
		t.Fatal("END marker not found")
	}

	afterMarker := strings.TrimSpace(content[endIdx+len("<!-- END GENERATED SECTION -->"):])
	if afterMarker != "" {
		t.Errorf("expected no content after END marker, got: %q", afterMarker)
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

func TestGenerateClaudeMd_JavaScriptProject(t *testing.T) {
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
	requireContains(t, content, "npm")
	requireContains(t, content, "qsdev Commands")
}

func TestGenerateClaudeMd_RustProject(t *testing.T) {
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
	requireContains(t, content, "cargo")
	requireContains(t, content, "qsdev Commands")
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
		ProjectName:  "secure",
		Languages:    []types.LanguageChoice{{Name: "go"}},
		Hooks:        types.HookChoices{SafetyBlock: true},
		EnabledTools: map[string]bool{"attach-guard": true},
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

func TestBuildClaudeMdData_GdevCommands(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	data := claudecode.BuildClaudeMdData(answers, reg)

	if len(data.GdevCommands) == 0 {
		t.Fatal("expected GdevCommands to be populated")
	}

	cmdNames := make(map[string]bool)
	for _, c := range data.GdevCommands {
		cmdNames[c.Command] = true
	}

	for _, expected := range []string{"qsdev init", "qsdev devenv doctor", "qsdev status", "qsdev check"} {
		if !cmdNames[expected] {
			t.Errorf("expected command %q in GdevCommands", expected)
		}
	}
}

func TestGenerateClaudeMd_SectionMarkers(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
		Hooks:       types.HookChoices{SafetyBlock: true},
		AgentTools: types.AgentToolsAnswers{
			PostmortemEnabled: true,
			VersionSentinel:   true,
			SembleEnabled:     true,
			SembleMode:        "mcp",
		},
		Skills: []string{"security-review"},
		EnabledTools: map[string]bool{
			"attach-guard":         true,
			"agent-postmortem":     true,
			"version-sentinel":     true,
			"semble":               true,
			"trail-of-bits-skills": true,
		},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	markers := []string{
		"qsdev:attach-guard",
		"qsdev:agent-postmortem",
		"qsdev:version-sentinel",
		"qsdev:semble",
		"qsdev:trail-of-bits-skills",
		"qsdev:skills",
		"qsdev:commands",
	}

	for _, marker := range markers {
		openTag := "<!-- " + marker + " -->"
		closeTag := "<!-- /" + marker + " -->"
		openCount := strings.Count(content, openTag)
		closeCount := strings.Count(content, closeTag)
		if openCount == 0 {
			continue
		}
		if openCount != closeCount {
			t.Errorf("unbalanced section marker %q: open=%d close=%d", marker, openCount, closeCount)
		}
	}
}

func TestGenerateClaudeMd_HasGdevReference(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requireContains(t, string(got.Content), "@.claude/qsdev-reference.md")
}

func TestGenerateClaudeMd_QsdevCommandsSection(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "## qsdev Commands")
	requireContains(t, content, "<!-- qsdev:commands -->")
	requireContains(t, content, "<!-- /qsdev:commands -->")
	requireContains(t, content, "qsdev init")
	requireContains(t, content, "qsdev check")
}

func TestGenerateClaudeMd_CatalogDrivenToolSections(t *testing.T) {
	reg := newTestRegistry(t, goMock())

	t.Run("enabled tools get markers from catalog", func(t *testing.T) {
		answers := types.WizardAnswers{
			ProjectName: "test",
			Languages:   []types.LanguageChoice{{Name: "go"}},
			EnabledTools: map[string]bool{
				"semgrep":  true,
				"gitleaks": true,
			},
		}
		got, err := claudecode.GenerateClaudeMd(answers, reg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content := string(got.Content)
		requireContains(t, content, "<!-- qsdev:semgrep -->")
		requireContains(t, content, "<!-- /qsdev:semgrep -->")
		requireContains(t, content, "<!-- qsdev:gitleaks -->")
		requireContains(t, content, "<!-- /qsdev:gitleaks -->")
		requireContains(t, content, "Semgrep SAST")
		requireContains(t, content, "Gitleaks")
	})

	t.Run("disabled tools get no markers", func(t *testing.T) {
		answers := types.WizardAnswers{
			ProjectName:  "test",
			Languages:    []types.LanguageChoice{{Name: "go"}},
			EnabledTools: map[string]bool{},
		}
		got, err := claudecode.GenerateClaudeMd(answers, reg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content := string(got.Content)
		requireNotContains(t, content, "<!-- qsdev:semgrep -->")
		requireNotContains(t, content, "<!-- qsdev:gitleaks -->")
	})

	t.Run("tools without CLAUDE.md section_id get no markers", func(t *testing.T) {
		answers := types.WizardAnswers{
			ProjectName: "test",
			Languages:   []types.LanguageChoice{{Name: "go"}},
			EnabledTools: map[string]bool{
				"branch-naming": true,
			},
		}
		got, err := claudecode.GenerateClaudeMd(answers, reg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content := string(got.Content)
		requireNotContains(t, content, "<!-- qsdev:branch-naming -->")
	})
}

func TestGenerateClaudeMd_NoLanguageConventions(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "test",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireNotContains(t, content, "Language Conventions")
	requireNotContains(t, content, "### Go")
	requireNotContains(t, content, "fmt.Errorf")
}
