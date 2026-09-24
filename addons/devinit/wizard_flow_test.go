package devinit

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// parseInitFlags parses args with the init flag definitions and returns the
// pre-wizard answers and the FlagSet, exactly as runCreate derives them.
func parseInitFlags(t *testing.T, args ...string) (types.WizardAnswers, *FlagSet) {
	t.Helper()
	var opts InitOptions
	cmd := &cobra.Command{Use: "init"}
	RegisterInitFlags(cmd, &opts)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parsing flags %v: %v", args, err)
	}
	partial, err := AnswersFromFlags(opts, "/tmp/project")
	if err != nil {
		t.Fatalf("AnswersFromFlags(%v): %v", args, err)
	}
	return partial, NewFlagSet(cmd)
}

// lineReader returns one line per Read call, like a terminal in canonical
// mode, so each accessible prompt consumes exactly one answer.
type lineReader struct{ lines []string }

func (l *lineReader) Read(p []byte) (int, error) {
	if len(l.lines) == 0 {
		return 0, io.EOF
	}
	n := copy(p, l.lines[0]+"\n")
	l.lines = l.lines[1:]
	return n, nil
}

func languageNames(langs []types.LanguageChoice) []string {
	names := make([]string, len(langs))
	for i, l := range langs {
		names[i] = l.Name
	}
	return names
}

func TestInit_SelfProtectionHookOnEveryCreatePath(t *testing.T) {
	answersYAML := "languages:\n  - name: go\ndirenv: true\nclaude_code: true\npermission_level: standard\n"

	tests := []struct {
		name string
		args func(dir string) []string
	}{
		{"profile without --yes", func(string) []string { return []string{"--profile", "go-web"} }},
		{"answers file without --yes", func(dir string) []string {
			return []string{"--answers-file", filepath.Join(dir, "answers.yaml")}
		}},
		{"flags with --yes", func(string) []string { return []string{"--lang", "go", "--yes"} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "answers.yaml"), []byte(answersYAML), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, err := executeInitCmd(t, dir, tt.args(dir)...); err != nil {
				t.Fatalf("init failed: %v\n%s", err, out)
			}
			settings, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("reading settings.json: %v", err)
			}
			for _, want := range []string{" selfprotect", "package-guard"} {
				if !strings.Contains(string(settings), want) {
					t.Errorf("settings.json lacks %q hook:\n%s", want, settings)
				}
			}
		})
	}
}

func TestMapFormToAnswers_CustomizeEnablesSelfProtection(t *testing.T) {
	t.Parallel()
	fs := &formState{quickChoice: "customize", selectedLanguages: []string{"go"}, claudeCode: true, permissionLevel: "standard"}
	got := mapFormToAnswers(fs, "/tmp/project", "project", types.DetectedProject{})
	if !got.Hooks.SelfProtection {
		t.Error("customize path with Claude Code enabled must enable the self-protection hook")
	}
}

// TestMapFormToAnswers_TierFollowsFormPermissionLevel verifies that without
// --tier the customize path records the tier implied by the permission level
// the form chose, and that an explicit --tier is kept.
func TestMapFormToAnswers_TierFollowsFormPermissionLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		partialTier     string
		permissionLevel string
		wantTier        string
		wantCompliance  string
	}{
		{"standard level gets the default tier", "", "standard", "standard", "enhanced"},
		{"supply-chain-only level gets its tier", "", "supply-chain-only", "supply-chain-only", "baseline"},
		{"explicit tier is kept", "full", "supply-chain-only", "full", "strict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := &formState{
				quickChoice:       "customize",
				selectedLanguages: []string{"go"},
				claudeCode:        true,
				permissionLevel:   tt.permissionLevel,
				partial:           types.WizardAnswers{Tier: tt.partialTier},
			}
			got := mapFormToAnswers(fs, "/tmp/project", "project", types.DetectedProject{})
			if got.Tier != tt.wantTier {
				t.Errorf("Tier = %q, want %q", got.Tier, tt.wantTier)
			}
			if got.ComplianceLevel != tt.wantCompliance {
				t.Errorf("ComplianceLevel = %q, want %q", got.ComplianceLevel, tt.wantCompliance)
			}
			if got.PermissionLevel != tt.permissionLevel {
				t.Errorf("PermissionLevel = %q, want %q", got.PermissionLevel, tt.permissionLevel)
			}
		})
	}
}

func TestWizard_ExplicitFlagsSurvive(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}

	for _, quickChoice := range []string{"yes", "customize"} {
		t.Run(quickChoice, func(t *testing.T) {
			t.Parallel()
			partial, flags := parseInitFlags(t,
				"--lang", "python", "--service", "redis", "--tier", "full",
				"--env", "API_BASE=https://example.test", "--claude-hooks", "safety-block,pre-commit")
			partial.Detected = detected
			fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)
			if fs.quickChoice != "customize" {
				t.Errorf("explicit flags: quickChoice = %q, want customize", fs.quickChoice)
			}
			fs.quickChoice = quickChoice
			fs.confirmed = true

			got := mapFormToAnswers(fs, "/tmp/project", "project", detected)

			if names := languageNames(got.Languages); !slices.Equal(names, []string{"python"}) {
				t.Errorf("languages = %v, want [python]", names)
			}
			if len(got.Services) != 1 || got.Services[0].Name != "redis" {
				t.Errorf("services = %v, want [redis]", got.Services)
			}
			if got.Tier != "full" {
				t.Errorf("tier = %q, want full", got.Tier)
			}
			if got.EnvVars["API_BASE"] != "https://example.test" {
				t.Errorf("env vars = %v, want API_BASE kept", got.EnvVars)
			}
			if !got.Hooks.PreCommit {
				t.Error("pre-commit hook from --claude-hooks was dropped")
			}
			preview := renderPlanPreview(got)
			if !strings.Contains(preview, "Python, Redis") || strings.Contains(preview, "Go") {
				t.Errorf("preview does not match generated answers:\n%s", preview)
			}
		})
	}
}

func TestWizard_CustomizeChoicesDriveEnabledTools(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	partial, flags := parseInitFlags(t, "--lang", "go", "--tier", "full")
	partial.Detected = detected
	fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)
	fs.agentPostmortem = false
	fs.confirmed = true

	got := mapFormToAnswers(fs, "/tmp/project", "project", detected)
	toolreg.MergeInferredTools(&got, toolreg.DefaultRegistry())

	if got.EnabledTools[toolreg.ToolAgentPostmortem] {
		t.Error("agent-postmortem is enabled although the wizard turned it off")
	}
	if !got.EnabledTools["secretspec"] {
		t.Errorf("tier-derived tools missing from EnabledTools: %v", got.EnabledTools)
	}
	if got.ComplianceLevel == "" {
		t.Error("tier-derived compliance level missing on the customize path")
	}
}

func TestNewFormState_SeedsOnlyFromExplicitFlags(t *testing.T) {
	t.Parallel()
	goOnly := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	python := types.DetectedProject{HasPyProject: true, PythonVersion: "3.12"}

	tests := []struct {
		name     string
		args     []string
		detected types.DetectedProject
		check    func(t *testing.T, fs *formState)
	}{
		{"no flags keeps safe seeds", nil, goOnly, func(t *testing.T, fs *formState) {
			if !fs.safetyBlock || !fs.direnv || !fs.claudeCode || !fs.agentPostmortem || fs.autoFormat {
				t.Errorf("seeds overwritten by unset flags: %+v", *fs)
			}
			if fs.agentVersionSentinel {
				t.Error("version-sentinel seed should follow language support (Go is not covered)")
			}
			if fs.quickChoice != "yes" {
				t.Errorf("quickChoice = %q, want yes", fs.quickChoice)
			}
		}},
		{"--direnv=false", []string{"--direnv=false"}, goOnly, func(t *testing.T, fs *formState) {
			if fs.direnv {
				t.Error("--direnv=false ignored")
			}
		}},
		{"--agent-postmortem=false", []string{"--agent-postmortem=false"}, goOnly, func(t *testing.T, fs *formState) {
			if fs.agentPostmortem {
				t.Error("--agent-postmortem=false ignored")
			}
		}},
		{"--lang selects the version-sentinel seed", []string{"--lang", "python"}, goOnly, func(t *testing.T, fs *formState) {
			if !fs.agentVersionSentinel {
				t.Error("version-sentinel seed should follow the explicit --lang (Python is covered), not detection")
			}
		}},
		{"--agent-version-sentinel=false", []string{"--agent-version-sentinel=false"}, python, func(t *testing.T, fs *formState) {
			if fs.agentVersionSentinel {
				t.Error("--agent-version-sentinel=false ignored")
			}
		}},
		{"--claude-hooks auto-format", []string{"--claude-hooks", "auto-format"}, goOnly, func(t *testing.T, fs *formState) {
			if !fs.autoFormat || fs.safetyBlock {
				t.Errorf("autoFormat=%v safetyBlock=%v, want true/false", fs.autoFormat, fs.safetyBlock)
			}
		}},
		{"--tier selects its preset", []string{"--tier", "supply-chain-only"}, goOnly, func(t *testing.T, fs *formState) {
			if fs.permissionLevel != "supply-chain-only" {
				t.Errorf("permissionLevel = %q, want the tier preset", fs.permissionLevel)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			partial, flags := parseInitFlags(t, tt.args...)
			fs := newFormState(tt.detected, MapDetectionToDefaults(tt.detected, "/tmp/project"), partial, flags)
			tt.check(t, fs)
		})
	}
}

func TestAnswersFromFlags_PermissionLevelFollowsTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"tier only", []string{"--lang", "go", "--tier", "supply-chain-only"}, ""},
		{"explicit preset", []string{"--lang", "go", "--claude-permissions", "minimal"}, "minimal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			partial, _ := parseInitFlags(t, tt.args...)
			partial.FillDefaults(types.DetectedProject{}, catalog.MustDefault())
			if partial.PermissionLevel != tt.want {
				t.Errorf("PermissionLevel = %q, want %q", partial.PermissionLevel, tt.want)
			}
		})
	}
}

func TestAnswersFromFlags_EnvWithoutEquals(t *testing.T) {
	t.Parallel()
	_, err := AnswersFromFlags(InitOptions{Env: []string{"API_BASE"}}, "/tmp/project")
	if err == nil || !strings.Contains(err.Error(), "KEY=VALUE") {
		t.Errorf("err = %v, want a KEY=VALUE error", err)
	}
}

func TestPreviewBindings_CoverEveryFormField(t *testing.T) {
	t.Parallel()
	fs := &formState{moduleFields: newModuleFields(ecosystem.DefaultRegistry(), nil, types.DetectedProject{})}
	if len(fs.moduleFields) == 0 {
		t.Fatal("no language contributes module wizard fields")
	}
	bound := make(map[uintptr]bool)
	for _, b := range fs.previewBindings() {
		bound[reflect.ValueOf(b).Pointer()] = true
	}
	v := reflect.ValueOf(fs).Elem()
	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		if name == "partial" || name == "confirmed" || name == "moduleFields" {
			continue // moduleFields are bound answer by answer, checked below
		}
		if !bound[v.Field(i).UnsafeAddr()] {
			t.Errorf("formState.%s is missing from previewBindings; the Plan Preview would not refresh when it changes", name)
		}
	}
	for _, lf := range fs.moduleFields {
		for _, f := range lf.fields {
			if !bound[reflect.ValueOf(f.binding()).Pointer()] {
				t.Errorf("module field %s/%s is missing from previewBindings", lf.lang, f.spec.Key)
			}
		}
	}
}

func TestQuickPath_DetailsMatchGeneratedAnswers(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	partial, flags := parseInitFlags(t)
	partial.Detected = detected
	fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)

	generated := mapFormToAnswers(fs, "/tmp/project", "project", detected)
	if !generated.Direnv || !generated.ClaudeCode {
		t.Fatalf("quick path should generate direnv and Claude Code, got direnv=%v claude=%v", generated.Direnv, generated.ClaudeCode)
	}

	quick := quickPathAnswers(partial, detected)
	details := buildDetailedDefaults(quick)
	for _, want := range []string{"direnv: enabled", "Claude Code: enabled (standard permissions)", "self-protection"} {
		if !strings.Contains(details, want) {
			t.Errorf("details missing %q:\n%s", want, details)
		}
	}
	if summary := QuickPathSummary(quick); !strings.Contains(summary, "direnv") || !strings.Contains(summary, "Claude Code") {
		t.Errorf("summary %q omits generated components", summary)
	}
}

func TestRunAccessibleSteps_HonorsHiddenSteps(t *testing.T) {
	t.Parallel()
	enable := true
	var hidden, final string
	steps := []wizardStep{
		{fields: func() []huh.Field { return []huh.Field{huh.NewConfirm().Title("Enable extra?").Value(&enable)} }},
		{
			fields: func() []huh.Field { return []huh.Field{huh.NewInput().Title("Hidden detail").Value(&hidden)} },
			hidden: func() bool { return !enable },
		},
		{fields: func() []huh.Field { return []huh.Field{huh.NewInput().Title("Final answer").Value(&final)} }},
	}

	var out bytes.Buffer
	if err := runAccessibleSteps(steps, huh.ThemeBase16(), &out, &lineReader{lines: []string{"n", "hello"}}); err != nil {
		t.Fatalf("runAccessibleSteps: %v", err)
	}
	if strings.Contains(out.String(), "Hidden detail") {
		t.Errorf("hidden step was prompted:\n%s", out.String())
	}
	if final != "hello" {
		t.Errorf("final = %q, want hello (answer went to the wrong prompt)", final)
	}
}

func TestRunAccessibleSteps_EndOfInputAborts(t *testing.T) {
	t.Parallel()
	var a, b string
	steps := []wizardStep{
		{fields: func() []huh.Field { return []huh.Field{huh.NewInput().Title("First").Value(&a)} }},
		{fields: func() []huh.Field { return []huh.Field{huh.NewInput().Title("Second").Value(&b)} }},
	}
	err := runAccessibleSteps(steps, huh.ThemeBase16(), io.Discard, &lineReader{lines: []string{"x"}})
	if !errors.Is(err, huh.ErrUserAborted) {
		t.Errorf("err = %v, want ErrUserAborted", err)
	}
}

func TestRunAccessibleSteps_WizardQuickPathSkipsCustomizeScreens(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{HasGoMod: true, GoVersion: "1.24"}
	partial, flags := parseInitFlags(t)
	partial.Detected = detected
	fs := newFormState(detected, MapDetectionToDefaults(detected, "/tmp/project"), partial, flags)

	var out bytes.Buffer
	steps := buildWizardSteps(detected, fs, flags)
	if err := runAccessibleSteps(steps, huh.ThemeBase16(), &out, &lineReader{lines: []string{"1", "y"}}); err != nil {
		t.Fatalf("runAccessibleSteps: %v\n%s", err, out.String())
	}
	for _, unwanted := range []string{"Languages & Runtimes", "Go version", "Enable Claude Code?", "Semble mode"} {
		if strings.Contains(out.String(), unwanted) {
			t.Errorf("quick path prompted %q:\n%s", unwanted, out.String())
		}
	}
	if !strings.Contains(out.String(), "Plan Preview") || !fs.confirmed {
		t.Errorf("confirm screen not answered (confirmed=%v):\n%s", fs.confirmed, out.String())
	}
}

func TestWizardOptions_DerivedFromCatalog(t *testing.T) {
	t.Parallel()
	cat := catalog.MustDefault()
	values := func(opts []huh.Option[string]) []string {
		out := make([]string, len(opts))
		for i, o := range opts {
			out[i] = o.Value
		}
		return out
	}

	if got, want := values(serviceOptions()), cat.Services(); !slices.Equal(got, want) {
		t.Errorf("service options = %v, want catalog services %v", got, want)
	}
	if got, want := values(permissionOptions()), cat.PermissionPresets(); !slices.Equal(got, want) {
		t.Errorf("permission options = %v, want catalog presets %v", got, want)
	}
	mcp := values(mcpServerOptions())
	for _, name := range cat.DefaultMCPServers() {
		if !slices.Contains(mcp, name) {
			t.Errorf("default MCP server %q is not selectable (options %v)", name, mcp)
		}
	}

	partial, _ := parseInitFlags(t)
	if partial.AgentTools.VersionSentinelHours != cat.DefaultVersionSentinelHours() {
		t.Errorf("VersionSentinelHours = %d, want catalog default %d",
			partial.AgentTools.VersionSentinelHours, cat.DefaultVersionSentinelHours())
	}
}

func TestMapDetectionToDefaults_EcosystemsFromRegistry(t *testing.T) {
	t.Parallel()
	detected := types.DetectedProject{
		HasPackageJSON: true,
		Ecosystems:     map[string]bool{"javascript": true, "node": true, "aws": true, "ruby": true},
	}
	got := languageNames(MapDetectionToDefaults(detected, "/tmp/project").Languages)
	if want := []string{"javascript", "aws", "ruby"}; !slices.Equal(got, want) {
		t.Errorf("languages = %v, want %v", got, want)
	}
}

func TestRegisterProfiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{"new profile registers", "embedder-custom", false},
		{"collision with built-in fails", "go-web", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := DefaultProjectProfileRegistry()
			err := registerProfiles(reg, map[string]Profile{tt.profile: {Description: "custom"}})
			if (err != nil) != tt.wantErr {
				t.Fatalf("registerProfiles err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.profile) {
				t.Errorf("error %q does not name the profile", err)
			}
		})
	}
}

func TestIsDevBuildVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		version string
		want    bool
	}{
		{"", true},
		{"dev", true},
		{"(devel)", true},
		{"v0.8.1-0.20260101120000-abcdef123456", true},
		{"v0.8.1-0.20260101120000-abcdef123456+dirty", true},
		{"v0.8.0+dirty", true},
		{"v0.8.0", false},
		{"0.8.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			if got := isDevBuildVersion(tt.version); got != tt.want {
				t.Errorf("isDevBuildVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

// TestHookNames_MarksHooksWithoutPolicy guards W046: the init preview must
// not present tool-gates without a policy as an active control.
func TestHookNames_MarksHooksWithoutPolicy(t *testing.T) {
	t.Parallel()
	gates := types.HookChoices{SafetyBlock: true, ToolGates: true}
	tests := []struct {
		name   string
		policy types.HooksConfig
		want   []string
	}{
		{name: "no policy", want: []string{"safety-block", "tool-gates (no policy)"}},
		{
			name:   "deny list",
			policy: types.HooksConfig{ToolGates: types.ToolGatesConfig{Denied: []string{"WebFetch"}}},
			want:   []string{"safety-block", "tool-gates"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hookNames(types.WizardAnswers{Hooks: gates, HookPolicy: tt.policy})
			if !slices.Equal(got, tt.want) {
				t.Errorf("hookNames = %v, want %v", got, tt.want)
			}
		})
	}
}
