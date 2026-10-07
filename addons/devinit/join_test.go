package devinit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newJoinTestCmd returns a cobra command with a live context and buffered
// output, suitable for driving runJoin/buildJoinAnswers in tests.
func newJoinTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "init"}
	cmd.SetContext(context.Background())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

// TestConfigToAnswers_MapsConfigFields locks the config->answers half of the
// join flow: every field the create path reads from WizardAnswers must be
// populated from the parsed .qsdev.yaml. Regression test for BL-P1-17.
func TestConfigToAnswers_MapsConfigFields(t *testing.T) {
	t.Parallel()
	enabled := true
	cfg := &types.QsdevConfig{
		Version: types.ConfigVersionCurrent,
		Tier:    "full",
		Profile: "go-web",
		Languages: []types.LanguageConfig{
			{Name: "go", Version: "1.24", PackageManager: "go"},
		},
		Services: []types.ServiceConfig{
			{Name: "postgres", Version: "16"},
		},
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "standard",
			Skills:          []string{"deploy"},
			MCPServers:      []string{"context7"},
		},
		Tools: types.ToolsConfig{
			Enabled: []string{"ripsecrets"},
		},
		Infrastructure: types.InfraConfig{
			RegistryProxy: "https://proxy.example.com",
		},
	}

	answers := qsdevconfig.ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/proj")

	if len(answers.Languages) != 1 || answers.Languages[0].Name != "go" || answers.Languages[0].Version != "1.24" {
		t.Errorf("languages not mapped: %+v", answers.Languages)
	}
	if len(answers.Services) != 1 || answers.Services[0].Name != "postgres" {
		t.Errorf("services not mapped: %+v", answers.Services)
	}
	if !answers.ClaudeCode {
		t.Error("ClaudeCode should be enabled")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want standard", answers.PermissionLevel)
	}
	if len(answers.Skills) != 1 || answers.Skills[0] != "deploy" {
		t.Errorf("Skills not mapped: %+v", answers.Skills)
	}
	if len(answers.MCPServers) != 1 || answers.MCPServers[0] != "context7" {
		t.Errorf("MCPServers not mapped: %+v", answers.MCPServers)
	}
	if !answers.EnabledTools["ripsecrets"] {
		t.Errorf("EnabledTools not mapped: %+v", answers.EnabledTools)
	}
	if answers.Tier != "full" {
		t.Errorf("Tier = %q, want full", answers.Tier)
	}
	if answers.ProjectTypeProfile != "go-web" {
		t.Errorf("ProjectTypeProfile = %q, want go-web", answers.ProjectTypeProfile)
	}
	if answers.Infrastructure.RegistryProxy != "https://proxy.example.com" {
		t.Errorf("Infrastructure not mapped: %+v", answers.Infrastructure)
	}
	if !answers.Confirmed {
		t.Error("Confirmed should be true for the non-interactive join path")
	}
}

// TestRunJoin_WritesGeneratedFilesFromConfig exercises the full join flow
// (buildJoinAnswers -> configToAnswers -> runAccumulator -> writeJoinResults)
// and asserts it writes exactly the files the shared accumulator (also used by
// the create path) produces, plus state and answers. Regression test for
// BL-P1-17: runJoin previously had no test coverage.
func TestRunJoin_WritesGeneratedFilesFromConfig(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")

	dir := t.TempDir()
	configYAML := "" +
		"version: 1\n" +
		"languages:\n" +
		"  - name: go\n" +
		"    version: \"1.24\"\n" +
		"claude_code:\n" +
		"  enabled: true\n" +
		"  permission_level: standard\n"
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := InitOptions{Quiet: true}

	// Compute the file set the shared accumulator (the create path's engine)
	// produces for the join answers, so we can assert parity with what gets
	// written to disk.
	cmd, _ := newJoinTestCmd()
	answers, err := buildJoinAnswers(cmd, opts, dir)
	if err != nil {
		t.Fatalf("buildJoinAnswers: %v", err)
	}
	acc, err := runAccumulator(answers, generationScope{})
	if err != nil {
		t.Fatalf("runAccumulator: %v", err)
	}
	if len(acc.allFiles) == 0 {
		t.Fatal("accumulator produced no files")
	}

	// Run the real join flow.
	if err := runJoin(cmd, opts, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}

	// Every accumulator file (the same set create writes) must be on disk.
	for _, f := range acc.allFiles {
		if _, err := os.Stat(filepath.Join(dir, f.Path)); err != nil {
			t.Errorf("expected generated file %q on disk after join: %v", f.Path, err)
		}
	}

	// Core create-path artifacts and join bookkeeping.
	for _, rel := range []string{"devenv.yaml", "devenv.nix", stateFilePath()} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %q after join: %v", rel, err)
		}
	}
	if _, err := os.Stat(answersPath(dir)); err != nil {
		t.Errorf("expected saved answers file after join: %v", err)
	}

	// The saved answers must reflect the config that drove the join.
	answersContent, err := os.ReadFile(answersPath(dir))
	if err != nil {
		t.Fatalf("reading saved answers: %v", err)
	}
	if !strings.Contains(string(answersContent), "name: go") {
		t.Error("saved join answers should contain the go language from the config")
	}
}

// writeJoinConfig writes a minimal committed .qsdev.yaml for lang to dir, as
// a teammate's fresh clone has it.
func writeJoinConfig(t *testing.T, dir, lang string) {
	t.Helper()
	configYAML := "version: 1\nlanguages:\n  - name: " + lang + "\nclaude_code:\n  enabled: true\n"
	if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rawStateFiles returns the file keys recorded in dir's init state file as
// written to disk, bypassing LoadStateFromFile's legacy migration so a
// producer that still records an entry is caught.
func rawStateFiles(t *testing.T, dir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, stateFilePath()))
	if err != nil {
		t.Fatalf("reading state: %v", err)
	}
	var raw struct {
		Files map[string]any `yaml:"files"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing state: %v", err)
	}
	return raw.Files
}

// rawManifestPaths returns the paths listed in dir's committed manifest as
// written to disk, bypassing LoadManifest's legacy filtering.
func rawManifestPaths(t *testing.T, dir string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, state.ManifestFile()))
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	paths := map[string]bool{}
	for line := range strings.Lines(string(data)) {
		if _, p, ok := strings.Cut(strings.TrimRight(line, "\r\n"), "  "); ok {
			paths[p] = true
		}
	}
	return paths
}

// TestJoin_LocalConfigNotTracked verifies join writes the developer's local
// config as an untracked, gitignored file: neither the committed manifest
// nor the init state records it (U02-01).
func TestJoin_LocalConfigNotTracked(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := t.TempDir()
	writeJoinConfig(t, dir, "go")

	cmd, _ := newJoinTestCmd()
	if err := runJoin(cmd, InitOptions{Quiet: true}, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}

	localCfg := branding.Get().LocalConfig
	requireFileExists(t, dir, localCfg)
	if _, tracked := rawStateFiles(t, dir)[localCfg]; tracked {
		t.Errorf("init state records %s", localCfg)
	}
	if rawManifestPaths(t, dir)[localCfg] {
		t.Errorf("committed manifest lists %s", localCfg)
	}
	if !slices.Contains(strings.Split(readFileContent(t, dir, ".gitignore"), "\n"), localCfg) {
		t.Errorf(".gitignore does not list %s", localCfg)
	}
}

// TestJoin_LocalConfigEditIsNotDrift verifies editing the local config after
// join leaves the project "already set up" instead of reporting drift.
func TestJoin_LocalConfigEditIsNotDrift(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := t.TempDir()
	writeJoinConfig(t, dir, "go")

	cmd, _ := newJoinTestCmd()
	if err := runJoin(cmd, InitOptions{Quiet: true}, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}
	localCfg := filepath.Join(dir, branding.Get().LocalConfig)
	if err := os.WriteFile(localCfg, []byte("extra_packages:\n  - jq\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectOnboardingMode(dir)
	if err != nil {
		t.Fatalf("DetectOnboardingMode: %v", err)
	}
	if !got.AlreadySetUp {
		t.Errorf("mode = %v (%s), want already set up", got.Mode, got.Explanation)
	}
}

// TestJoin_KeepsExistingLocalConfig verifies join never rewrites a local
// config the developer already has, and that a dry run previews the template
// without writing it.
func TestJoin_KeepsExistingLocalConfig(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	localCfg := branding.Get().LocalConfig

	t.Run("existing file kept", func(t *testing.T) {
		dir := t.TempDir()
		writeJoinConfig(t, dir, "go")
		const mine = "# mine\nextra_packages:\n  - jq\n"
		if err := os.WriteFile(filepath.Join(dir, localCfg), []byte(mine), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd, _ := newJoinTestCmd()
		if err := runJoin(cmd, InitOptions{Quiet: true}, dir); err != nil {
			t.Fatalf("runJoin: %v", err)
		}
		if got := readFileContent(t, dir, localCfg); got != mine {
			t.Errorf("join rewrote %s:\n%s", localCfg, got)
		}
	})

	// A local config symlinked from elsewhere (for example a dotfiles
	// repository) is the developer's, so join keeps the link and its target.
	for _, tc := range []struct {
		name    string
		outside bool
	}{
		{name: "symlink to existing file in root kept"},
		{name: "symlink to existing file outside root kept", outside: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeJoinConfig(t, dir, "go")
			const mine = "# mine\nextra_packages:\n  - jq\n"
			target, linkTarget := filepath.Join(dir, "my-local.yaml"), "my-local.yaml"
			if tc.outside {
				target = filepath.Join(t.TempDir(), "dot-local.yaml")
				linkTarget = target
			}
			if err := os.WriteFile(target, []byte(mine), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(linkTarget, filepath.Join(dir, localCfg)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			cmd, _ := newJoinTestCmd()
			if err := runJoin(cmd, InitOptions{Quiet: true}, dir); err != nil {
				t.Fatalf("runJoin: %v", err)
			}
			assertSymlinkKept(t, filepath.Join(dir, localCfg), target, mine)
		})
	}

	t.Run("created file is announced", func(t *testing.T) {
		dir := t.TempDir()
		writeJoinConfig(t, dir, "go")
		cmd, out := newJoinTestCmd()
		if err := runJoin(cmd, InitOptions{}, dir); err != nil {
			t.Fatalf("runJoin: %v", err)
		}
		if want := "+ " + localCfg + " (local, untracked)"; !strings.Contains(out.String(), want) {
			t.Errorf("join output lacks %q:\n%s", want, out)
		}
		if _, err := os.Stat(filepath.Join(dir, localCfg)); err != nil {
			t.Errorf("join did not create %s: %v", localCfg, err)
		}
	})

	t.Run("dry run previews only", func(t *testing.T) {
		dir := t.TempDir()
		writeJoinConfig(t, dir, "go")
		cmd, out := newJoinTestCmd()
		if err := runJoin(cmd, InitOptions{DryRun: true}, dir); err != nil {
			t.Fatalf("runJoin --dry-run: %v", err)
		}
		if want := "+ " + localCfg + " (local, untracked)"; !strings.Contains(out.String(), want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, out)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Errorf("dry run wrote files: %v", names)
		}
	})
}

// TestJoin_PHPManifestOmitsGitignoredComposerConfig verifies the Composer
// global config, written under the gitignored project dot-directory, stays in
// local state (so drift and repair cover it) but is not committed to the
// manifest a fresh clone is checked against (U09-02).
func TestJoin_PHPManifestOmitsGitignoredComposerConfig(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := t.TempDir()
	writeJoinConfig(t, dir, "php")

	cmd, _ := newJoinTestCmd()
	if err := runJoin(cmd, InitOptions{Quiet: true}, dir); err != nil {
		t.Fatalf("runJoin: %v", err)
	}

	composerCfg := "." + branding.Get().AppName + "/composer/config.json"
	if _, tracked := rawStateFiles(t, dir)[composerCfg]; !tracked {
		t.Errorf("init state does not record %s", composerCfg)
	}
	if rawManifestPaths(t, dir)[composerCfg] {
		t.Errorf("committed manifest lists gitignored %s", composerCfg)
	}
}
