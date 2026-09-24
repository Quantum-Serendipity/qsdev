package container

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecProber_RunsOutsideWorkingDirectory pins that host probes do not
// inherit the caller's working directory: on Windows a probe grandchild that
// outlives its timeout would otherwise keep the project directory open and
// block deleting or renaming it.
func TestExecProber_RunsOutsideWorkingDirectory(t *testing.T) {
	pwd, err := exec.LookPath("pwd")
	if err != nil {
		t.Skip("pwd not available")
	}
	t.Chdir(t.TempDir())

	out, err := (&ExecProber{}).Output(context.Background(), pwd, "-P")
	if err != nil {
		t.Fatalf("pwd: %v", err)
	}
	want, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != want {
		t.Errorf("probe ran in %q, want %q", got, want)
	}
}
