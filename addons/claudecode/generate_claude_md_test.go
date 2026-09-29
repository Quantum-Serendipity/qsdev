package claudecode_test

import (
	"slices"
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

// TestGenerateClaudeMd_HasGdevReference verifies the @-import of the qsdev
// reference appears only at the Full tier, the only tier that generates
// .claude/qsdev-reference.md.
func TestGenerateClaudeMd_HasGdevReference(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		tier string
		want bool
	}{
		{"standard", false},
		{"full", true},
	} {
		t.Run(tc.tier, func(t *testing.T) {
			t.Parallel()
			reg := newTestRegistry(t, goMock())
			answers := types.WizardAnswers{
				ProjectName: "test",
				Tier:        tc.tier,
				Languages:   []types.LanguageChoice{{Name: "go"}},
			}
			got, err := claudecode.GenerateClaudeMd(answers, reg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if has := strings.Contains(string(got.Content), "@.claude/qsdev-reference.md"); has != tc.want {
				t.Errorf("reference import present = %v, want %v", has, tc.want)
			}
		})
	}
}

// TestGenerate_ClaudeMdAdvertisesOnlyGeneratedArtifacts verifies that every
// skill, agent and @-import CLAUDE.md advertises exists in the file set
// Generate emits, at every tier and for both legacy (nil) and explicit
// EnabledTools.
func TestGenerate_ClaudeMdAdvertisesOnlyGeneratedArtifacts(t *testing.T) {
	t.Parallel()
	explicit := map[string]bool{
		"qsdev-doctor":                           true,
		"qsdev-add-dep":                          true,
		"consulting-agent-security-reviewer":     true,
		"consulting-workflow-write-adr":          true,
		"consulting-workflow-onboard-me":         true,
		"consulting-agent-codebase-explorer":     true,
		"consulting-agent-test-gap-analyzer":     false,
		"consulting-workflow-incident-debug":     false,
		"consulting-agent-handoff-doc-generator": false,
	}
	for _, tierName := range []string{"supply-chain-only", "standard", "full"} {
		for _, enabled := range []map[string]bool{nil, explicit} {
			name := tierName + "/nil-enabled"
			if enabled != nil {
				name = tierName + "/explicit-enabled"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				reg := newTestRegistry(t, goMock())
				answers := types.WizardAnswers{
					ProjectName:  "adv",
					Tier:         tierName,
					Languages:    []types.LanguageChoice{{Name: "go"}},
					EnabledTools: enabled,
				}
				files, err := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{}).Generate(answers)
				if err != nil {
					t.Fatalf("Generate: %v", err)
				}
				paths := make(map[string]bool, len(files))
				var claudeMd string
				for _, f := range files {
					paths[f.Path] = true
					if f.Path == "CLAUDE.md" {
						claudeMd = string(f.Content)
					}
				}
				for _, line := range strings.Split(claudeMd, "\n") {
					switch {
					case strings.HasPrefix(line, "- `/"):
						skill := strings.TrimPrefix(line, "- `/")
						skill = skill[:strings.Index(skill, "`")]
						if p := ".claude/skills/" + skill + "/SKILL.md"; !paths[p] {
							t.Errorf("CLAUDE.md advertises /%s but %s is not generated", skill, p)
						}
					case strings.HasPrefix(line, "- `@"):
						agent := strings.TrimPrefix(line, "- `@")
						agent = agent[:strings.Index(agent, "`")]
						if p := ".claude/agents/" + agent + ".md"; !paths[p] {
							t.Errorf("CLAUDE.md advertises @%s but %s is not generated", agent, p)
						}
					case strings.HasPrefix(line, "@"):
						if p := strings.TrimSpace(strings.TrimPrefix(line, "@")); !paths[p] {
							t.Errorf("CLAUDE.md imports @%s but it is not generated", p)
						}
					}
				}
			})
		}
	}
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

// TestGenerateClaudeMd_BuildAndTestWithoutBuildCommands verifies the Build &
// Test section renders for ecosystems that have test/lint commands but no
// build command (Python, Terraform), instead of being dropped entirely.
func TestGenerateClaudeMd_BuildAndTestWithoutBuildCommands(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t, pythonMock())
	answers := types.WizardAnswers{
		ProjectName: "py",
		Languages:   []types.LanguageChoice{{Name: "python"}},
	}
	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got.Content)
	requireContains(t, content, "## Build & Test\n\n```bash\npython -m pytest\nruff check .\n```")
}

// TestGenerateClaudeMd_ToolSectionOnlyWhenToolEmitsFiles verifies a tool's
// CLAUDE.md section is advertised only when the tool actually emits files for
// the configuration: lookup-docs' skill is a Full-tier artifact, so a
// Standard-tier CLAUDE.md must not point at it.
func TestGenerateClaudeMd_ToolSectionOnlyWhenToolEmitsFiles(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		tier string
		want bool
	}{
		{"standard", false},
		{"full", true},
	} {
		t.Run(tc.tier, func(t *testing.T) {
			t.Parallel()
			reg := newTestRegistry(t, goMock())
			answers := types.WizardAnswers{
				ProjectName:  "docs",
				Tier:         tc.tier,
				Languages:    []types.LanguageChoice{{Name: "go"}},
				EnabledTools: map[string]bool{"lookup-docs": true},
			}
			got, err := claudecode.GenerateClaudeMd(answers, reg)
			if err != nil {
				t.Fatal(err)
			}
			if has := strings.Contains(string(got.Content), "<!-- qsdev:lookup-docs -->"); has != tc.want {
				t.Errorf("lookup-docs section present = %v, want %v", has, tc.want)
			}
		})
	}
}

// TestBuildClaudeMdData_ConfiguredPackageManager lists only the package
// manager a language is configured with (W079): a uv project must not be told
// that pip and poetry apply too.
func TestBuildClaudeMdData_ConfiguredPackageManager(t *testing.T) {
	t.Parallel()
	py := &ecosystem.MockModule{
		NameVal:        "python",
		DisplayNameVal: "Python",
		TierVal:        1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{
			{Name: "pip"}, {Name: "uv"}, {Name: "poetry"},
		},
	}
	java := &ecosystem.MockModule{
		NameVal:            "java",
		DisplayNameVal:     "Java",
		TierVal:            1,
		PackageManagersVal: []ecosystem.PackageManagerInfo{{Name: "maven"}, {Name: "gradle"}},
	}
	reg := newTestRegistry(t, py, java)

	tests := []struct {
		name string
		lang types.LanguageChoice
		want []string
	}{
		{"configured uv", types.LanguageChoice{Name: "python", PackageManager: "uv"}, []string{"uv"}},
		{"configured poetry", types.LanguageChoice{Name: "python", PackageManager: "poetry"}, []string{"poetry"}},
		{"unconfigured uses module default", types.LanguageChoice{Name: "python"}, []string{"pip"}},
		{"detected build tool", types.LanguageChoice{Name: "java", Extras: []string{"build_tool=gradle"}}, []string{"gradle"}},
		{"both build tools", types.LanguageChoice{Name: "java", Extras: []string{"build_tool=both"}}, []string{"maven", "gradle"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := claudecode.BuildClaudeMdData(types.WizardAnswers{Languages: []types.LanguageChoice{tt.lang}}, reg)
			if !slices.Equal(data.PackageManagers, tt.want) {
				t.Errorf("PackageManagers = %v, want %v", data.PackageManagers, tt.want)
			}
		})
	}
}
