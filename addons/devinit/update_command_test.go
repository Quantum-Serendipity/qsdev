package devinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestStageStatus_String(t *testing.T) {
	tests := []struct {
		status   StageStatus
		expected string
	}{
		{StageSuccess, "updated"},
		{StageSkipped, "skipped"},
		{StageFailed, "failed"},
		{StageUpToDate, "up-to-date"},
		{StageStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expected {
				t.Errorf("StageStatus(%d).String() = %q, want %q", int(tt.status), got, tt.expected)
			}
		})
	}
}

func newTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

func TestRunSelfUpdateStage_DevBuild(t *testing.T) {
	cmd, _ := newTestCmd()
	result := runSelfUpdateStage(cmd, FullUpdateOptions{})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped for dev build, got %v", result.Status)
	}
	if result.Name != "Self-update" {
		t.Errorf("expected name 'Self-update', got %q", result.Name)
	}
	if !strings.Contains(result.Message, "dev build") {
		t.Errorf("expected message to mention 'dev build', got %q", result.Message)
	}
}

func TestRunDevenvInputStage_NotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cmd, _ := newTestCmd()
	result := runDevenvInputStage(cmd, FullUpdateOptions{})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped when devenv not on PATH, got %v", result.Status)
	}
	if result.Name != "Devenv inputs" {
		t.Errorf("expected name 'Devenv inputs', got %q", result.Name)
	}
	if !strings.Contains(result.Message, "devenv not installed") {
		t.Errorf("expected message about devenv not installed, got %q", result.Message)
	}
}

func TestRunDevenvInputStage_DryRun(t *testing.T) {
	cmd, _ := newTestCmd()
	result := runDevenvInputStage(cmd, FullUpdateOptions{DryRun: true})

	if result.Status != StageSkipped {
		t.Errorf("expected StageSkipped for dry-run, got %v", result.Status)
	}
	if strings.Contains(result.Message, "devenv not installed") {
		return
	}
	if !strings.Contains(result.Message, "dry-run") {
		t.Errorf("expected message to mention 'dry-run', got %q", result.Message)
	}
	if !strings.Contains(result.Message, "devenv update") {
		t.Errorf("expected message to mention 'devenv update', got %q", result.Message)
	}
}

func TestRunFullUpdate_SelectiveStages(t *testing.T) {
	tests := []struct {
		name          string
		opts          FullUpdateOptions
		expectStages  []string
		excludeStages []string
	}{
		{
			name:          "self-only runs only self-update",
			opts:          FullUpdateOptions{SelfOnly: true},
			expectStages:  []string{"Self-update"},
			excludeStages: []string{"Config regeneration", "Devenv inputs"},
		},
		{
			name:          "configs-only runs only config regeneration",
			opts:          FullUpdateOptions{ConfigsOnly: true},
			expectStages:  []string{"Config regeneration"},
			excludeStages: []string{"Self-update", "Devenv inputs"},
		},
		{
			name:          "deps-only runs only devenv inputs",
			opts:          FullUpdateOptions{DepsOnly: true},
			expectStages:  []string{"Devenv inputs"},
			excludeStages: []string{"Self-update", "Config regeneration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())

			cmd, buf := newTestCmd()
			_ = runFullUpdate(cmd, tt.opts)

			output := buf.String()

			for _, stage := range tt.expectStages {
				if !strings.Contains(output, stage) {
					t.Errorf("expected output to contain stage %q, got:\n%s", stage, output)
				}
			}

			for _, stage := range tt.excludeStages {
				summaryIdx := strings.Index(output, "Update Summary:")
				if summaryIdx >= 0 {
					summary := output[summaryIdx:]
					if strings.Contains(summary, stage) {
						t.Errorf("expected summary NOT to contain stage %q, got:\n%s", stage, summary)
					}
				}
			}
		})
	}
}

func TestRunFullUpdate_AllStages(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, dir string)
		wantErr      bool
		wantInOutput []string
	}{
		{
			// A bare `update` outside a project is how the binary is upgraded;
			// the project stages must be skipped, not failed.
			name:         "outside a project skips project stages",
			setup:        func(*testing.T, string) {},
			wantErr:      false,
			wantInOutput: []string{"Config regeneration: not a qsdev project", "Devenv inputs: not a qsdev project"},
		},
		{
			// A teammate's clone has the shared config but no local answers;
			// it is still a project, so the real error must surface.
			name: "project config without answers is still a project",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:      true,
			wantInOutput: []string{"✗ Config regeneration", "no saved answers"},
		},
		{
			name: "inside a project a config failure fails the run",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeAnswersFile(t, dir, "languages: [unterminated\n")
			},
			wantErr:      true,
			wantInOutput: []string{"✗ Config regeneration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			dir := t.TempDir()
			tt.setup(t, dir)
			t.Chdir(dir)

			cmd, buf := newTestCmd()
			err := runFullUpdate(cmd, FullUpdateOptions{})
			output := buf.String()

			want := append([]string{"Self-update", "[1/3]", "[2/3]", "[3/3]", "Update Summary:"}, tt.wantInOutput...)
			for _, w := range want {
				if !strings.Contains(output, w) {
					t.Errorf("expected output to contain %q, got:\n%s", w, output)
				}
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("runFullUpdate error = %v, wantErr %v\n%s", err, tt.wantErr, output)
			}
			if err != nil && !strings.Contains(err.Error(), "one or more update stages failed") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

// writeAnswersFile writes raw content to the project's saved answers file.
func writeAnswersFile(t *testing.T, dir, content string) {
	t.Helper()
	path := answersPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeFakeExecutable writes a POSIX shell script to dir/name and returns its
// path. Tests using it are skipped on Windows.
func writeFakeExecutable(t *testing.T, dir, name, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake executables are not supported on Windows")
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRunFullUpdate_ForceWithoutSelfUpdateWarns verifies that --force, which
// no longer overwrites edited configs, tells the user when it has no effect.
func TestRunFullUpdate_ForceWithoutSelfUpdateWarns(t *testing.T) {
	tests := []struct {
		name     string
		opts     FullUpdateOptions
		wantNote bool
	}{
		{name: "configs-only with force", opts: FullUpdateOptions{ConfigsOnly: true, Force: true}, wantNote: true},
		{name: "configs-only without force", opts: FullUpdateOptions{ConfigsOnly: true}, wantNote: false},
		{name: "self-only with force", opts: FullUpdateOptions{SelfOnly: true, Force: true}, wantNote: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			t.Chdir(t.TempDir())

			cmd, buf := newTestCmd()
			_ = runFullUpdate(cmd, tt.opts)
			if got := strings.Contains(buf.String(), "--force only reinstalls the binary"); got != tt.wantNote {
				t.Errorf("note printed = %v, want %v\n%s", got, tt.wantNote, buf.String())
			}
		})
	}
}

func TestConfigStageArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts FullUpdateOptions
		want string
	}{
		{"defaults", FullUpdateOptions{}, "update --configs-only"},
		{"binary-only flags are not forwarded", FullUpdateOptions{Force: true}, "update --configs-only"},
		{
			"config flags are forwarded",
			FullUpdateOptions{OverwriteModified: true, AllowDowngrade: true, SkipContainer: true},
			"update --configs-only --overwrite-modified --allow-downgrade --skip-container",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := strings.Join(configStageArgs(tt.opts), " "); got != tt.want {
				t.Errorf("configStageArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunConfigStageInBinary verifies that after a self-update the config
// stage runs in the freshly installed binary rather than in this process,
// whose embedded templates are the old release's.
func TestRunConfigStageInBinary(t *testing.T) {
	tests := []struct {
		name       string
		exitCode   int
		missing    bool
		wantStatus StageStatus
		wantMsg    string
	}{
		{name: "success", exitCode: 0, wantStatus: StageSuccess, wantMsg: "updated binary"},
		{name: "child failure fails the stage", exitCode: 3, wantStatus: StageFailed, wantMsg: "exit status 3"},
		{name: "unrunnable binary asks for a re-run", missing: true, wantStatus: StageSkipped, wantMsg: "update --configs-only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			argsFile := filepath.Join(dir, "args")
			exe := writeFakeExecutable(t, dir, "qsdev-new",
				fmt.Sprintf("echo \"$@\" > %q\nexit %d\n", argsFile, tt.exitCode))
			if tt.missing {
				exe = filepath.Join(dir, "does-not-exist")
			}

			cmd, _ := newTestCmd()
			res := runConfigStageInBinary(cmd, exe, FullUpdateOptions{OverwriteModified: true})

			if res.Status != tt.wantStatus {
				t.Fatalf("status = %v, want %v (message %q)", res.Status, tt.wantStatus, res.Message)
			}
			if !strings.Contains(res.Message, tt.wantMsg) {
				t.Errorf("message %q does not contain %q", res.Message, tt.wantMsg)
			}
			if tt.missing {
				return
			}
			got, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatalf("fake binary was not run: %v", err)
			}
			if want := "update --configs-only --overwrite-modified\n"; string(got) != want {
				t.Errorf("child args = %q, want %q", got, want)
			}
		})
	}

	t.Run("no executable path", func(t *testing.T) {
		cmd, _ := newTestCmd()
		res := runConfigStageInBinary(cmd, "", FullUpdateOptions{})
		if res.Status != StageSkipped || !strings.Contains(res.Message, "update --configs-only") {
			t.Errorf("got %v %q, want skipped with re-run hint", res.Status, res.Message)
		}
	})
}

// TestRunDevenvInputStage_ClaudeOnlySkipped verifies `devenv update` never
// runs against a claude-only project's user-owned devenv environment.
func TestRunDevenvInputStage_ClaudeOnlySkipped(t *testing.T) {
	binDir := t.TempDir()
	marker := filepath.Join(binDir, "ran")
	writeFakeExecutable(t, binDir, "devenv", fmt.Sprintf("touch %q\n", marker))
	t.Setenv("PATH", binDir)

	dir := t.TempDir()
	if err := saveAnswers(dir, types.WizardAnswers{ClaudeCode: true, MergeMode: mergeModeClaudeOnly}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cmd, _ := newTestCmd()
	res := runDevenvInputStage(cmd, FullUpdateOptions{})
	if res.Status != StageSkipped || !strings.Contains(res.Message, "claude-only") {
		t.Errorf("got %v %q, want skipped for claude-only project", res.Status, res.Message)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("devenv update ran for a claude-only project")
	}
}

// TestRunConfigUpdateStage_ForceOnlyAffectsBinary verifies that `update
// --force` (reinstall the binary) no longer discards user edits to managed
// configs; only --overwrite-modified does.
func TestRunConfigUpdateStage_ForceOnlyAffectsBinary(t *testing.T) {
	tests := []struct {
		name         string
		opts         FullUpdateOptions
		wantPreserve bool
	}{
		{name: "force keeps user edits", opts: FullUpdateOptions{Force: true}, wantPreserve: true},
		{name: "overwrite-modified discards user edits", opts: FullUpdateOptions{OverwriteModified: true}, wantPreserve: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := executeInitCmd(t, dir, "--lang", "go", "--yes"); err != nil {
				t.Fatalf("init failed: %v", err)
			}
			yamlPath := filepath.Join(dir, "devenv.yaml")
			orig, err := os.ReadFile(yamlPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(yamlPath, append(orig, []byte("\n# user customization\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			cmd, buf := newTestCmd()
			tt.opts.SkipContainer = true
			if res := runConfigUpdateStage(cmd, tt.opts); res.Status != StageSuccess {
				t.Fatalf("config stage = %v %q\n%s", res.Status, res.Message, buf.String())
			}

			preserved := strings.Contains(readFileContent(t, dir, "devenv.yaml"), "# user customization")
			if preserved != tt.wantPreserve {
				t.Errorf("user edit preserved = %v, want %v", preserved, tt.wantPreserve)
			}
		})
	}
}

func TestUpdateCmd_CheckFlag(t *testing.T) {
	cmd := updateCmd()
	if err := cmd.Flags().Set("check", "true"); err != nil {
		t.Fatalf("failed to set --check flag: %v", err)
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// In test context, version is "dev", so check-only prints skip message.
	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Dev build") {
		t.Errorf("expected dev build skip message, got: %s", output)
	}
}

func TestUpdateCmd_HasExpectedFlags(t *testing.T) {
	cmd := updateCmd()

	expectedFlags := []string{
		"dry-run", "force", "overwrite-modified", "allow-downgrade",
		"self-only", "configs-only", "deps-only", "check", "changelog",
	}

	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to be registered", name)
		}
	}
}
