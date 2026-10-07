package devinit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
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

func TestSelfUpdateConfig_Strictness(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		opts       FullUpdateOptions
		wantStrict bool
	}{
		{name: "signature required by default", opts: FullUpdateOptions{}, wantStrict: true},
		{name: "--no-strict relaxes it", opts: FullUpdateOptions{NoStrict: true}, wantStrict: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := selfUpdateConfig(tt.opts).Strict; got != tt.wantStrict {
				t.Errorf("Strict = %v, want %v", got, tt.wantStrict)
			}
		})
	}
}

func TestUpdateCmd_NoStrictFlag(t *testing.T) {
	t.Parallel()
	if updateCmd().Flags().Lookup("no-strict") == nil {
		t.Fatal("qsdev update must expose --no-strict, the escape hatch the strict-mode error points to")
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
			// Run outside any project so a config stage cannot touch the repo.
			t.Chdir(testutil.IsolatedDir(t))

			cmd, buf := newTestCmd()
			err := runFullUpdate(cmd, tt.opts)

			output := buf.String()

			// A stage may fail in this sandbox, but only as a reported stage
			// failure; any other error means runFullUpdate bailed out early.
			if err != nil && !strings.Contains(err.Error(), "one or more update stages failed") {
				t.Fatalf("runFullUpdate returned an unexpected error: %v\noutput:\n%s", err, output)
			}

			summaryIdx := strings.Index(output, "Update Summary:")
			if summaryIdx < 0 {
				t.Fatalf("expected 'Update Summary:' in output, got:\n%s", output)
			}
			summary := output[summaryIdx:]

			for _, stage := range tt.expectStages {
				if !strings.Contains(summary, stage) {
					t.Errorf("expected summary to contain stage %q, got:\n%s", stage, summary)
				}
			}

			for _, stage := range tt.excludeStages {
				if strings.Contains(summary, stage) {
					t.Errorf("expected summary NOT to contain stage %q, got:\n%s", stage, summary)
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
			// A teammate's clone has the shared config but no local state;
			// it is still a project, and both project stages must refuse
			// with the join hint rather than regenerate or bump devenv.lock.
			name: "project config without local state asks to join",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:      true,
			wantInOutput: []string{"✗ Config regeneration", "✗ Devenv inputs", "init --yes"},
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
			dir := testutil.IsolatedDir(t)
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
			t.Chdir(testutil.IsolatedDir(t))

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

	dir := testutil.IsolatedDir(t)
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
			dir := testutil.IsolatedDir(t)
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

// codedErr is a stage error that carries a process exit code.
type codedErr struct{ code int }

func (e codedErr) Error() string { return fmt.Sprintf("coded %d", e.code) }
func (e codedErr) ExitCode() int { return e.code }

func TestPrintStageSummary_PropagatesStageExitCode(t *testing.T) {
	t.Parallel()
	plain := errors.New("plain failure")
	tests := []struct {
		name     string
		results  []StageResult
		wantErr  bool
		wantCode int // 0: the error must not carry an exit code
	}{
		{name: "all succeeded", results: []StageResult{{Name: "a", Status: StageSuccess}}},
		{
			name:    "failure without exit code",
			results: []StageResult{{Name: "a", Status: StageFailed, Err: plain}},
			wantErr: true,
		},
		{
			name:     "coded failure",
			results:  []StageResult{{Name: "a", Status: StageFailed, Err: fmt.Errorf("wrapped: %w", codedErr{3})}},
			wantErr:  true,
			wantCode: 3,
		},
		{
			name: "first coded failure wins",
			results: []StageResult{
				{Name: "a", Status: StageFailed, Err: plain},
				{Name: "b", Status: StageFailed, Err: codedErr{3}},
				{Name: "c", Status: StageFailed, Err: codedErr{4}},
			},
			wantErr:  true,
			wantCode: 3,
		},
		{
			name:    "coded error on a skipped stage is ignored",
			results: []StageResult{{Name: "a", Status: StageSkipped, Err: codedErr{3}}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, _ := newTestCmd()
			err := printStageSummary(cmd, tc.results)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			var coded interface{ ExitCode() int }
			hasCode := errors.As(err, &coded)
			switch {
			case tc.wantCode == 0 && hasCode:
				t.Errorf("err %v carries exit code %d, want none", err, coded.ExitCode())
			case tc.wantCode != 0 && (!hasCode || coded.ExitCode() != tc.wantCode):
				t.Errorf("err %v does not carry exit code %d", err, tc.wantCode)
			}
		})
	}
}

// TestUpdateDryRunNoLockChange is the U14 regression test: `update --dry-run`
// in an initialized project previews the devenv-input stage without running
// `devenv update`, so devenv.lock stays byte-identical and devenv is never
// started, even though it is on PATH.
func TestUpdateDryRunNoLockChange(t *testing.T) {
	dir := testutil.IsolatedDir(t)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/dry\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "devenv-ran")
	writeFakeExecutable(t, binDir, "devenv", fmt.Sprintf("echo \"$@\" > %q\necho bumped > devenv.lock\n", marker))
	t.Setenv("PATH", binDir)
	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	// init may probe `devenv version`; only what update does is under test.
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, "devenv.lock")
	lock := []byte(`{"nodes":{"root":{}},"root":"root","version":7}` + "\n")
	if err := os.WriteFile(lockPath, lock, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)
	t.Setenv(procexec.ForbidExecEnv, "1")
	cmd := updateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update --dry-run: %v\n%s", err, out.String())
	}

	if _, err := os.Stat(marker); err == nil {
		ran, _ := os.ReadFile(marker)
		t.Errorf("update --dry-run started devenv %s", ran)
	}
	got, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, lock) {
		t.Errorf("devenv.lock changed by update --dry-run:\n got %q\nwant %q", got, lock)
	}
	if !strings.Contains(out.String(), "would run: devenv update") {
		t.Errorf("dry-run did not preview the devenv stage:\n%s", out.String())
	}
}

// TestUpdateDryRun_ReleaseBuildQueriesNothing pins the read-only contract
// for `update --dry-run` on a release build: the preview names the binary
// stage but makes no release metadata query (no network), download, install
// or exec. `update --check` is the explicit network query.
func TestUpdateDryRun_ReleaseBuildQueriesNothing(t *testing.T) {
	t.Chdir(testutil.IsolatedDir(t))
	t.Setenv(procexec.ForbidExecEnv, "1")

	origVersion, origCheck, origDo := binaryVersion, checkForUpdate, doSelfUpdate
	t.Cleanup(func() { binaryVersion, checkForUpdate, doSelfUpdate = origVersion, origCheck, origDo })
	binaryVersion = func() string { return "v0.7.9" }
	checks := 0
	checkForUpdate = func(context.Context, selfupdate.Config, string) (*selfupdate.Release, error) {
		checks++
		return &selfupdate.Release{Version: "0.8.0"}, nil
	}
	doSelfUpdate = func(context.Context, selfupdate.Config, *selfupdate.Release) error {
		t.Error("update --dry-run installed a binary")
		return nil
	}

	for _, args := range [][]string{
		{"--dry-run", "--self-only"},
		{"--dry-run", "--self-only", "--force"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			checks = 0
			cmd := updateCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("update %v: %v\n%s", args, err, out.String())
			}
			if checks != 0 {
				t.Errorf("release metadata queried %d times, want 0", checks)
			}
			if !strings.Contains(out.String(), stageSelfUpdate) || !strings.Contains(out.String(), "qsdev update --check") {
				t.Errorf("dry-run did not preview the binary stage:\n%s", out.String())
			}
		})
	}
}
