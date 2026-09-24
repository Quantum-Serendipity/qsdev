package devinit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// executeContainerCmd creates and runs the container command in the given
// directory with the provided args.
func executeContainerCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}

	cmd := containerCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestContainerCmd_NoComposeFiles(t *testing.T) {
	dir := t.TempDir()

	output, err := executeContainerCmd(t, dir, "migrate")
	if err == nil {
		t.Fatal("expected error for no compose files")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		// The error may be wrapped in cobra output.
		t.Logf("error type: %T, output: %s", err, output)
		return
	}
	if exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.Code)
	}
	if !strings.Contains(output, "No compose files") {
		t.Errorf("output missing 'No compose files': %s", output)
	}
}

func TestContainerCmd_DryRunDefault(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(`
services:
  web:
    image: nginx
`), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}

	output, err := executeContainerCmd(t, dir, "migrate")
	// Ignore exit errors since there may be findings.
	if err != nil {
		if _, ok := err.(*ExitError); !ok {
			t.Fatalf("migrate error = %v\nOutput: %s", err, output)
		}
	}

	// File should NOT be modified (dry-run default).
	content, _ := os.ReadFile(composePath)
	if strings.Contains(string(content), "docker.io") {
		t.Error("compose file was modified in dry-run mode")
	}
}

func TestContainerCmd_AutoFix(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(`services:
  web:
    image: nginx
`), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}

	output, err := executeContainerCmd(t, dir, "migrate", "--auto-fix")
	if err != nil {
		if _, ok := err.(*ExitError); !ok {
			t.Fatalf("auto-fix error = %v\nOutput: %s", err, output)
		}
	}

	// File should be modified.
	content, _ := os.ReadFile(composePath)
	if !strings.Contains(string(content), "docker.io/library/nginx") {
		t.Errorf("compose file was not updated with auto-fix:\n%s", content)
	}
}

func TestContainerCmd_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`
services:
  web:
    image: nginx
`), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}

	output, err := executeContainerCmd(t, dir, "migrate", "--json")
	if err != nil {
		if _, ok := err.(*ExitError); !ok {
			t.Fatalf("json output error = %v\nOutput: %s", err, output)
		}
	}

	if !strings.Contains(output, `"source_runtime"`) {
		t.Errorf("JSON output missing source_runtime:\n%s", output)
	}
	if !strings.Contains(output, `"issues"`) {
		t.Errorf("JSON output missing issues:\n%s", output)
	}
}

func TestContainerCmd_DetectSubcommand(t *testing.T) {
	dir := t.TempDir()

	output, err := executeContainerCmd(t, dir, "detect")
	if err != nil {
		t.Fatalf("detect error = %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Active runtime:") {
		t.Errorf("detect output missing runtime info:\n%s", output)
	}
	if !strings.Contains(output, "Capabilities:") {
		t.Errorf("detect output missing capabilities:\n%s", output)
	}
}

func TestMigrateAppliesFixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		autoFix   bool
		dryRun    bool
		dryRunSet bool
		want      bool
	}{
		{name: "default is a dry run", dryRun: true, want: false},
		{name: "auto-fix applies", autoFix: true, dryRun: true, want: true},
		{name: "explicit dry-run wins over auto-fix", autoFix: true, dryRun: true, dryRunSet: true, want: false},
		{name: "auto-fix with dry-run=false applies", autoFix: true, dryRun: false, dryRunSet: true, want: true},
		{name: "dry-run=false alone applies", dryRun: false, dryRunSet: true, want: true},
		{name: "explicit dry-run alone previews", dryRun: true, dryRunSet: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := migrateAppliesFixes(tt.autoFix, tt.dryRun, tt.dryRunSet); got != tt.want {
				t.Errorf("migrateAppliesFixes(%v, %v, %v) = %v, want %v",
					tt.autoFix, tt.dryRun, tt.dryRunSet, got, tt.want)
			}
		})
	}
}

// fixableCompose uses a short image name, which --auto-fix qualifies.
const fixableCompose = "services:\n  web:\n    image: nginx\n"

func TestContainerCmd_DryRunFlagCombinations(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantChanged bool
		wantOutput  string
	}{
		{
			name:        "auto-fix with explicit dry-run only previews",
			args:        []string{"migrate", "--auto-fix", "--dry-run"},
			wantChanged: false,
			wantOutput:  "would modify",
		},
		{
			name:        "dry-run=false alone applies fixes",
			args:        []string{"migrate", "--dry-run=false"},
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			composePath := filepath.Join(dir, "docker-compose.yml")
			if err := os.WriteFile(composePath, []byte(fixableCompose), 0o644); err != nil {
				t.Fatalf("writing compose: %v", err)
			}

			output, err := executeContainerCmd(t, dir, tt.args...)
			if err != nil {
				if _, ok := err.(*ExitError); !ok {
					t.Fatalf("migrate error = %v\nOutput: %s", err, output)
				}
			}

			content, _ := os.ReadFile(composePath)
			if changed := string(content) != fixableCompose; changed != tt.wantChanged {
				t.Errorf("compose changed = %v, want %v:\n%s", changed, tt.wantChanged, content)
			}
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output missing %q:\n%s", tt.wantOutput, output)
			}
		})
	}
}

func TestContainerCmd_AutoFixPreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(fixableCompose), 0o600); err != nil {
		t.Fatalf("writing compose: %v", err)
	}

	output, err := executeContainerCmd(t, dir, "migrate", "--auto-fix")
	if err != nil {
		if _, ok := err.(*ExitError); !ok {
			t.Fatalf("auto-fix error = %v\nOutput: %s", err, output)
		}
	}

	info, err := os.Stat(composePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("compose mode after auto-fix = %o, want 600", got)
	}
	content, _ := os.ReadFile(composePath)
	if string(content) == fixableCompose {
		t.Error("compose file was not fixed")
	}
}

func TestContainerCmd_OutputFileHoldsFullReport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(fixableCompose), 0o644); err != nil {
		t.Fatalf("writing compose: %v", err)
	}
	reportPath := filepath.Join(dir, "report.json")

	output, err := executeContainerCmd(t, dir, "migrate", "--json", "--output", reportPath)
	if err != nil {
		if _, ok := err.(*ExitError); !ok {
			t.Fatalf("migrate error = %v\nOutput: %s", err, output)
		}
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("reading report: %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("report file is not valid JSON:\n%s", data)
	}
	if strings.Contains(output, `"issues"`) {
		t.Errorf("report was also written to stdout:\n%s", output)
	}
}
