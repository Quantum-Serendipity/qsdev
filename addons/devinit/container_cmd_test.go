package devinit

import (
	"bytes"
	"os"
	"path/filepath"
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
