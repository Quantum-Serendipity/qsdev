package devinit

import (
	"bytes"
	"cmp"
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
	if err := qsdevconfig.WriteProjectConfig(path, qsdevconfig.AnswersToConfig(answers, "test")); err != nil {
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

			// The tier is always persisted (the catalog default without
			// --tier), so neither create nor join has to infer it.
			wantTier := cmp.Or(tierArg, "standard")
			cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
			if err != nil {
				t.Fatalf("parsing committed config: %v", err)
			}
			if created.Tier != wantTier || cfg.Tier != wantTier {
				t.Errorf("tier: create %q, persisted %q, want %q", created.Tier, cfg.Tier, wantTier)
			}

			cmd, _ := newJoinTestCmd()
			joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
			if err != nil {
				t.Fatalf("buildJoinAnswers: %v", err)
			}
			if joined.Tier != created.Tier {
				t.Errorf("tier: join %q, create %q", joined.Tier, created.Tier)
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
	cfg := qsdevconfig.AnswersToConfig(in, "test")
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

// TestJoin_InfraProfileRoundTrips is the regression test for the infra
// profile being persisted into the project-type `profile` key: `qsdev check`
// then rejected it as an unknown project-type profile, and a teammate joining
// lost it and got the consulting-default CI/Renovate files. The infra profile
// must land in `infra_profile`, the project-type --profile in `profile`, and
// join must restore both and generate the same profile-driven files.
func TestJoin_InfraProfileRoundTrips(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantInfra   string
		wantProject string
	}{
		{"infra profile only", append([]string{"--infra-profile", "enterprise"}, infraEndpointFlags...), "enterprise", ""},
		{"infra profile opting out of its components", []string{"--infra-profile", "startup-github", "--registry-proxy", "none", "--nix-cache", "none"}, "startup-github", ""},
		{"default infra profile", append([]string{"--infra-profile", "consulting-default"}, infraEndpointFlags...), "consulting-default", ""},
		{"both profiles", append([]string{"--infra-profile", "startup-github", "--profile", "go-web"}, infraEndpointFlags...), "startup-github", "go-web"},
		{"project-type profile only", []string{"--profile", "go-web"}, "", "go-web"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QSDEV_SKIP_SETUP", "1")
			dir := newGoProject(t)
			created := createAnswers(t, dir, append([]string{"--lang", "go"}, tt.args...)...)
			commitConfig(t, dir, created)

			cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
			if err != nil {
				t.Fatalf("parsing committed config: %v", err)
			}
			if cfg.InfraProfile != tt.wantInfra || cfg.Profile != tt.wantProject {
				t.Errorf("persisted infra_profile=%q profile=%q, want %q %q", cfg.InfraProfile, cfg.Profile, tt.wantInfra, tt.wantProject)
			}
			if cfg.Infrastructure.NixCache != created.Infrastructure.NixCache || cfg.Infrastructure.RegistryProxy != created.Infrastructure.RegistryProxy {
				t.Errorf("persisted infrastructure %+v, want %+v", cfg.Infrastructure, created.Infrastructure)
			}
			// `qsdev check` validates `profile` against the project-type registry.
			opts := qsdevconfig.ValidateOptions{ProfileNames: ensureProfileRegistry().Names()}
			if errs := qsdevconfig.ValidateQsdevConfig(cfg, opts); len(errs) > 0 {
				t.Errorf("committed config fails validation: %v", errs)
			}

			cmd, _ := newJoinTestCmd()
			joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
			if err != nil {
				t.Fatalf("buildJoinAnswers: %v", err)
			}
			if joined.ProfileName != tt.wantInfra || joined.ProjectTypeProfile != tt.wantProject {
				t.Errorf("join ProfileName=%q ProjectTypeProfile=%q, want %q %q", joined.ProfileName, joined.ProjectTypeProfile, tt.wantInfra, tt.wantProject)
			}

			joinFiles, createFiles := generatedContent(t, joined), generatedContent(t, created)
			if _, ok := createFiles["docs/security-overview.md"]; !ok {
				t.Fatal("create generated no infra-profile files; the comparison below would be vacuous")
			}
			for _, path := range []string{".github/workflows/security-scan.yml", "renovate.json", ".github/dependabot.yml", "docs/security-overview.md"} {
				if joinFiles[path] != createFiles[path] {
					t.Errorf("%s differs between join and create:\njoin:\n%s\ncreate:\n%s", path, joinFiles[path], createFiles[path])
				}
			}
		})
	}
}

// infraEndpointFlags supply the real endpoints an explicit infra profile
// requires (the built-in profiles carry none).
var infraEndpointFlags = []string{
	"--registry-proxy", "https://nexus.corp.internal",
	"--nix-cache", "corp",
	"--nix-cache-public-key", "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=",
}

// TestInitFlags_InfraEndpoints checks the endpoint flags reach the answers,
// persist in .qsdev.yaml and come back on join, and that an explicit
// profile without them fails generation with a message naming them.
func TestInitFlags_InfraEndpoints(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := newGoProject(t)
	created := createAnswers(t, dir, append([]string{"--lang", "go", "--infra-profile", "enterprise"}, infraEndpointFlags...)...)
	want := types.InfraConfig{
		RegistryProxy:     "https://nexus.corp.internal",
		NixCache:          "corp",
		NixCachePublicKey: "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=",
	}
	if created.Infrastructure.RegistryProxy != want.RegistryProxy || created.Infrastructure.NixCache != want.NixCache ||
		created.Infrastructure.NixCachePublicKey != want.NixCachePublicKey {
		t.Fatalf("flag answers Infrastructure = %+v, want %+v", created.Infrastructure, want)
	}
	commitConfig(t, dir, created)
	cmd, _ := newJoinTestCmd()
	joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}
	if joined.Infrastructure.NixCachePublicKey != want.NixCachePublicKey || joined.Infrastructure.RegistryProxy != want.RegistryProxy {
		t.Errorf("join Infrastructure = %+v, want %+v", joined.Infrastructure, want)
	}

	bare := createAnswers(t, newGoProject(t), "--lang", "go", "--infra-profile", "enterprise")
	if _, err := runAccumulator(bare, generationScope{}); err == nil || !strings.Contains(err.Error(), "--registry-proxy") {
		t.Errorf("explicit profile without endpoints: err = %v, want one naming --registry-proxy", err)
	}
}

// TestBuildJoinAnswers_InfraProfileFromConfig checks join restores the
// committed infra profile, including from a version 1 file that held it under
// `profile` (projects generated before the key split), and that an explicit
// --infra-profile on join replaces the committed value.
func TestBuildJoinAnswers_InfraProfileFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		args        []string
		wantInfra   string
		wantProject string
	}{
		{"committed value kept", "version: 2\ninfra_profile: enterprise\n", nil, "enterprise", ""},
		{"flag overrides", "version: 2\ninfra_profile: enterprise\n", []string{"--infra-profile", "startup-github"}, "startup-github", ""},
		{"v1 infra profile under profile", "version: 1\nprofile: enterprise\n", nil, "enterprise", ""},
		{"v1 project-type profile under profile", "version: 1\nprofile: go-web\n", nil, "", "go-web"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := newGoProject(t)
			config := tt.config + "languages:\n  - name: go\n"
			if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte(config), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd, opts, _ := newInitTestCmd(t, tt.args...)
			answers, err := buildJoinAnswers(cmd, opts, dir)
			if err != nil {
				t.Fatalf("buildJoinAnswers: %v", err)
			}
			if answers.ProfileName != tt.wantInfra || answers.ProjectTypeProfile != tt.wantProject {
				t.Errorf("ProfileName=%q ProjectTypeProfile=%q, want %q %q", answers.ProfileName, answers.ProjectTypeProfile, tt.wantInfra, tt.wantProject)
			}
		})
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
	// The remedy names --yes --force; only the ignored-flag list must not.
	listed, _, _ := strings.Cut(out, " because")
	for _, unwanted := range []string{"--yes", "--quiet"} {
		if strings.Contains(listed, unwanted) {
			t.Errorf("warning %q lists run-control flag %s", out, unwanted)
		}
	}

	quiet, _, quietBuf := newInitTestCmd(t, "--yes")
	warnIgnoredInitFlags(quiet)
	if quietBuf.Len() != 0 {
		t.Errorf("unexpected warning with only run-control flags: %q", quietBuf.String())
	}
}

// TestJoin_JavaScriptSubprojectParity is the regression test for join
// dropping detected module extras: .qsdev.yaml records no JavaScript project
// directory, so a joiner of a Go service with its UI in frontend/ must
// re-detect it and generate the same devenv.nix and frontend/.npmrc the
// creator did, not a root .npmrc npm never reads there.
func TestJoin_JavaScriptSubprojectParity(t *testing.T) {
	dir := newGoProject(t)
	for name, content := range map[string]string{
		"frontend/package.json":      `{"dependencies":{"react":"19.0.0"}}`,
		"frontend/package-lock.json": "{}",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	created := createAnswers(t, dir, "--lang", "go,javascript")
	commitConfig(t, dir, created)

	cmd, _ := newJoinTestCmd()
	joined, err := buildJoinAnswers(cmd, InitOptions{Quiet: true}, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}

	joinFiles, createFiles := generatedContent(t, joined), generatedContent(t, created)
	if _, ok := createFiles["frontend/.npmrc"]; !ok {
		t.Fatalf("create generated no frontend/.npmrc; files: %v", slices.Sorted(maps.Keys(createFiles)))
	}
	for _, path := range []string{"devenv.nix", "frontend/.npmrc"} {
		if joinFiles[path] != createFiles[path] {
			t.Errorf("%s differs:\njoin:\n%s\ncreate:\n%s", path, joinFiles[path], createFiles[path])
		}
	}
	if _, ok := joinFiles[".npmrc"]; ok {
		t.Error("join generated a root .npmrc for a frontend/ subproject")
	}
}
