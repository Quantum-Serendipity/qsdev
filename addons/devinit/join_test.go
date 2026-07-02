package devinit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

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

	answers := configToAnswers(cfg, types.DetectedProject{}, "/tmp/proj")

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
	acc, err := runAccumulator(answers, struct {
		ClaudeOnly bool
		DevenvOnly bool
	}{})
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
