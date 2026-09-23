package devinit

import (
	"bytes"
	"context"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newInitTestCmd returns an init command with every init flag registered and
// args parsed, plus the parsed options, like cobra hands them to RunE.
func newInitTestCmd(t *testing.T, args ...string) (*cobra.Command, InitOptions, *bytes.Buffer) {
	t.Helper()
	cmd, buf := newJoinTestCmd()
	var opts InitOptions
	RegisterInitFlags(cmd, &opts)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parsing flags %v: %v", args, err)
	}
	return cmd, opts, buf
}

// createAnswers builds answers exactly as the non-interactive create path
// (`qsdev init --yes ...`) does.
func createAnswers(t *testing.T, dir string, args ...string) types.WizardAnswers {
	t.Helper()
	cmd, opts, _ := newInitTestCmd(t, append([]string{"--yes"}, args...)...)
	answers, err := buildAnswersFromInputs(cmd, opts, dir, detect.Detect(context.Background(), dir), NewFlagSet(cmd))
	if err != nil {
		t.Fatalf("building create answers: %v", err)
	}
	return answers
}

// commitConfig writes the .qsdev.yaml the create path would commit.
func commitConfig(t *testing.T, dir string, answers types.WizardAnswers) {
	t.Helper()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	if err := writeQsdevConfig(path, buildQsdevConfig(answers, "test")); err != nil {
		t.Fatalf("writing config: %v", err)
	}
}

// generatedContent renders answers through the shared accumulator and
// returns the content of each generated file by path.
func generatedContent(t *testing.T, answers types.WizardAnswers) map[string]string {
	t.Helper()
	acc, err := runAccumulator(answers, generationScope{})
	if err != nil {
		t.Fatalf("runAccumulator: %v", err)
	}
	out := make(map[string]string, len(acc.allFiles))
	for _, f := range acc.allFiles {
		out[f.Path] = string(f.Content)
	}
	return out
}

// hooksMissing lists the hook choices enabled in want but not in got.
func hooksMissing(got, want types.HookChoices) []string {
	var missing []string
	g, w := reflect.ValueOf(got), reflect.ValueOf(want)
	for i := range w.NumField() {
		if w.Field(i).Bool() && !g.Field(i).Bool() {
			missing = append(missing, w.Type().Field(i).Name)
		}
	}
	return missing
}

func enabledToolSet(m map[string]bool) []string {
	var names []string
	for _, name := range slices.Sorted(maps.Keys(m)) {
		if m[name] {
			names = append(names, name)
		}
	}
	return names
}

func newGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestJoin_ParityWithCreate is the regression test for join dropping the
// hook, agent-tool and tier-derived tool choices: a teammate joining from the
// committed .qsdev.yaml must get the same security hooks, tools and Claude
// Code configuration the creator generated.
func TestJoin_ParityWithCreate(t *testing.T) {
	for _, tierArg := range []string{"", "standard", "full", "supply-chain-only"} {
		t.Run("tier="+tierArg, func(t *testing.T) {
			dir := newGoProject(t)
			args := []string{"--lang", "go"}
			if tierArg != "" {
				args = append(args, "--tier", tierArg)
			}
			created := createAnswers(t, dir, args...)
			commitConfig(t, dir, created)

			cmd, _ := newJoinTestCmd()
			joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
			if err != nil {
				t.Fatalf("buildJoinAnswers: %v", err)
			}

			// The persisted security level may imply extra hooks on join
			// (config.ConfigToAnswers), but join must never drop one create chose.
			if missing := hooksMissing(joined.Hooks, created.Hooks); len(missing) > 0 {
				t.Errorf("join dropped hooks %v: join %+v, create %+v", missing, joined.Hooks, created.Hooks)
			}
			if got, want := enabledToolSet(joined.EnabledTools), enabledToolSet(created.EnabledTools); !slices.Equal(got, want) {
				t.Errorf("enabled tools: join %v, create %v", got, want)
			}
			ja, ca := joined.AgentTools, created.AgentTools
			if ja.PostmortemEnabled != ca.PostmortemEnabled || ja.VersionSentinel != ca.VersionSentinel || ja.SembleEnabled != ca.SembleEnabled {
				t.Errorf("agent tools: join %+v, create %+v", ja, ca)
			}
			if joined.ComplianceLevel != created.ComplianceLevel {
				t.Errorf("compliance: join %q, create %q", joined.ComplianceLevel, created.ComplianceLevel)
			}
			if !slices.Equal(joined.MCPServers, created.MCPServers) {
				t.Errorf("MCP servers: join %v, create %v", joined.MCPServers, created.MCPServers)
			}

			joinFiles, createFiles := generatedContent(t, joined), generatedContent(t, created)
			if joinFiles[".mcp.json"] != createFiles[".mcp.json"] {
				t.Errorf(".mcp.json differs:\njoin:\n%s\ncreate:\n%s", joinFiles[".mcp.json"], createFiles[".mcp.json"])
			}
			joinSettings, createSettings := joinFiles[".claude/settings.json"], createFiles[".claude/settings.json"]
			for _, line := range strings.Split(createSettings, "\n") {
				if strings.Contains(line, `"command":`) && !strings.Contains(joinSettings, line) {
					t.Errorf("join settings.json lacks create's hook %s", strings.TrimSpace(line))
				}
			}
			// Only the strict level adds a generated hook (audit-log); below it
			// the settings must match exactly.
			if created.ComplianceLevel != "strict" && joinSettings != createSettings {
				t.Errorf("settings.json differs:\njoin:\n%s\ncreate:\n%s", joinSettings, createSettings)
			}
		})
	}
}

// TestJoin_LegacyConfigKeepsSecurityHooks covers configs that record no tool
// decisions (every config written before tools were persisted): the joiner
// must still get the self-protection and package-guard hooks and the default
// agent tools.
func TestJoin_LegacyConfigKeepsSecurityHooks(t *testing.T) {
	dir := newGoProject(t)
	legacy := "version: 1\nlanguages:\n  - name: go\nclaude_code:\n  enabled: true\n  permission_level: standard\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newJoinTestCmd()
	answers, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}
	if !answers.Hooks.SelfProtection || !answers.Hooks.SafetyBlock {
		t.Errorf("hooks = %+v, want self-protection and safety-block on", answers.Hooks)
	}
	if !answers.AgentTools.PostmortemEnabled || !answers.AgentTools.VersionSentinel {
		t.Errorf("agent tools = %+v, want catalog defaults", answers.AgentTools)
	}

	settings := generatedContent(t, answers)[".claude/settings.json"]
	for _, hook := range []string{branding.Get().AppName + " selfprotect", "package-guard.py"} {
		if !strings.Contains(settings, hook) {
			t.Errorf("settings.json lacks the %q hook:\n%s", hook, settings)
		}
	}
}

// TestJoin_ClaudeCodeOptOutRoundTrips is the regression test for a
// `--claude-code=false` project flipping back to Claude-enabled on join.
func TestJoin_ClaudeCodeOptOutRoundTrips(t *testing.T) {
	for _, optOut := range []string{"--claude-code=false", "--devenv-only"} {
		t.Run(optOut, func(t *testing.T) {
			dir := newGoProject(t)
			created := createAnswers(t, dir, "--lang", "go", optOut)
			commitConfig(t, dir, created)

			cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, ".qsdev.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ClaudeCode.Enabled == nil || *cfg.ClaudeCode.Enabled {
				t.Fatalf("claude_code.enabled = %v, want an explicit false", cfg.ClaudeCode.Enabled)
			}

			cmd, _ := newJoinTestCmd()
			joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
			if err != nil {
				t.Fatalf("buildJoinAnswers: %v", err)
			}
			if joined.ClaudeCode {
				t.Fatal("join re-enabled Claude Code for an opted-out project")
			}
			if _, ok := generatedContent(t, joined)[".claude/settings.json"]; ok {
				t.Error("join generated .claude/settings.json for an opted-out project")
			}
		})
	}
}

// TestBuildQsdevConfig_RoundTripsThroughJoin locks the fields buildQsdevConfig
// persists against what the join converter reads back.
func TestBuildQsdevConfig_RoundTripsThroughJoin(t *testing.T) {
	t.Parallel()
	in := types.WizardAnswers{
		Languages:       []types.LanguageChoice{{Name: "go", Version: "1.24"}},
		Services:        []types.ServiceChoice{{Name: "postgres", Version: "16", Settings: map[string]string{"initial_db": "app"}}},
		ClaudeCode:      true,
		PermissionLevel: "minimal",
		Tier:            "full",
		ComplianceLevel: "strict",
		EnabledTools:    map[string]bool{"gitleaks": true, "semble": false},
		Infrastructure: types.InfraConfig{
			RegistryProxy:      "https://proxy.example.com",
			RegistryProxyPaths: map[string]string{"npm": "/npm/"},
		},
	}
	cfg := buildQsdevConfig(in, "test")
	out := qsdevconfig.ConfigToAnswers(&cfg, types.DetectedProject{}, "/tmp/proj")

	if out.Services[0].Settings["initial_db"] != "app" {
		t.Errorf("service settings lost: %+v", out.Services)
	}
	if !maps.Equal(out.EnabledTools, in.EnabledTools) {
		t.Errorf("tools: got %v, want %v", out.EnabledTools, in.EnabledTools)
	}
	if out.ComplianceLevel != "strict" || out.Tier != "full" || out.PermissionLevel != "minimal" {
		t.Errorf("security fields lost: compliance=%q tier=%q perm=%q", out.ComplianceLevel, out.Tier, out.PermissionLevel)
	}
	if out.Infrastructure.RegistryProxyPaths["npm"] != "/npm/" {
		t.Errorf("registry_proxy_paths lost: %+v", out.Infrastructure)
	}
}

// TestRunJoin_DryRunLeavesWorkingTreeUntouched is the regression test for a
// join --dry-run writing .gitignore.
func TestRunJoin_DryRunLeavesWorkingTreeUntouched(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := t.TempDir()
	config := "version: 1\nlanguages:\n  - name: go\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, _ := newJoinTestCmd()
	if err := runJoin(cmd, InitOptions{DryRun: true, Yes: true, Quiet: true}, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != ".qsdev.yaml" {
			t.Errorf("dry-run created %q", e.Name())
		}
	}
}

// TestJoinPrerequisites_SkippedForDryRunAndClaudeOnly checks the prerequisite
// step never runs (and so never auto-installs) for a preview or a
// Claude-only join, like the create path.
func TestJoinPrerequisites_SkippedForDryRunAndClaudeOnly(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // every prerequisite is missing

	tests := []struct {
		name        string
		opts        InitOptions
		wantMissing bool
	}{
		{"dry run with --yes", InitOptions{DryRun: true, Yes: true}, false},
		{"claude only", InitOptions{ClaudeOnly: true}, false},
		{"normal run warns", InitOptions{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, buf := newJoinTestCmd()
			if got := joinPrerequisites(cmd, tt.opts); got != tt.wantMissing {
				t.Errorf("joinPrerequisites = %v, want %v", got, tt.wantMissing)
			}
			if !tt.wantMissing && buf.Len() != 0 {
				t.Errorf("unexpected output: %s", buf.String())
			}
		})
	}
}

// TestBuildJoinAnswers_OverridesLayerOverConfig is the regression test for
// join discarding the committed config under --answers-file and ignoring
// explicit flags.
func TestBuildJoinAnswers_OverridesLayerOverConfig(t *testing.T) {
	dir := newGoProject(t)
	config := "version: 1\ntier: full\nlanguages:\n  - name: go\n    version: \"1.24\"\n" +
		"claude_code:\n  enabled: true\n  permission_level: standard\n" +
		"infrastructure:\n  registry_proxy: https://proxy.example.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	answersFile := filepath.Join(t.TempDir(), "team.yaml")
	if err := os.WriteFile(answersFile, []byte("languages:\n  - name: python\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("answers file overlays config", func(t *testing.T) {
		cmd, opts, _ := newInitTestCmd(t, "--answers-file", answersFile)
		answers, err := buildJoinAnswers(cmd, opts, dir)
		if err != nil {
			t.Fatalf("buildJoinAnswers: %v", err)
		}
		if len(answers.Languages) != 1 || answers.Languages[0].Name != "python" {
			t.Errorf("languages = %+v, want the answers file's python", answers.Languages)
		}
		if answers.Tier != "full" || answers.Infrastructure.RegistryProxy != "https://proxy.example.com" {
			t.Errorf("config values lost: tier=%q infra=%+v", answers.Tier, answers.Infrastructure)
		}
	})

	t.Run("explicit flags override config", func(t *testing.T) {
		cmd, opts, _ := newInitTestCmd(t, "--tier", "standard")
		answers, err := buildJoinAnswers(cmd, opts, dir)
		if err != nil {
			t.Fatalf("buildJoinAnswers: %v", err)
		}
		if answers.Tier != "standard" {
			t.Errorf("tier = %q, want the --tier flag's standard", answers.Tier)
		}
		if answers.PermissionLevel != "standard" || len(answers.Languages) != 1 || answers.Languages[0].Name != "go" {
			t.Errorf("unset flags must not override config: %+v", answers)
		}
	})
}

// TestRunJoin_HonoursDevenvOnly is the regression test for --devenv-only being
// ignored in join mode.
func TestRunJoin_HonoursDevenvOnly(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := newGoProject(t)
	config := "version: 1\nlanguages:\n  - name: go\nclaude_code:\n  enabled: true\n  permission_level: standard\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, opts, _ := newInitTestCmd(t, "--devenv-only", "--quiet")
	if err := runJoin(cmd, opts, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "devenv.nix")); err != nil {
		t.Errorf("devenv.nix not generated: %v", err)
	}
	for _, rel := range []string{".claude/settings.json", "CLAUDE.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			t.Errorf("--devenv-only join generated %s", rel)
		}
	}
}

func TestWarnIgnoredInitFlags(t *testing.T) {
	cmd, _, buf := newInitTestCmd(t, "--lang", "go", "--tier", "full", "--yes", "--quiet")
	warnIgnoredInitFlags(cmd)
	out := buf.String()
	for _, want := range []string{"--lang", "--tier"} {
		if !strings.Contains(out, want) {
			t.Errorf("warning %q does not mention %s", out, want)
		}
	}
	for _, unwanted := range []string{"--yes", "--quiet"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("warning %q lists run-control flag %s", out, unwanted)
		}
	}

	quiet, _, quietBuf := newInitTestCmd(t, "--yes")
	warnIgnoredInitFlags(quiet)
	if quietBuf.Len() != 0 {
		t.Errorf("unexpected warning with only run-control flags: %q", quietBuf.String())
	}
}
