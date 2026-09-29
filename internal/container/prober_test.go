package container

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestExecProber_RunsOutsideWorkingDirectory pins that host probes do not
// inherit the caller's working directory: on Windows a probe grandchild that
// outlives its timeout would otherwise keep the project directory open and
// block deleting or renaming it.
func TestExecProber_RunsOutsideWorkingDirectory(t *testing.T) {
	// Print the probe's working directory in the platform's own path form:
	// Git Bash's pwd on Windows reports MSYS paths (/tmp), which never equal
	// the Windows form of os.TempDir().
	name, args := "pwd", []string{"-P"}
	if runtime.GOOS == "windows" {
		name, args = "cmd", []string{"/c", "cd"}
	}
	bin, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not available", name)
	}
	t.Chdir(t.TempDir())

	out, err := (&ExecProber{}).Output(context.Background(), bin, args...)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	got := strings.TrimSpace(string(out))
	// Compare by file identity: symlinks (/tmp on macOS) and 8.3 short names
	// (C:\Users\RUNNER~1 on Windows) name the same directory differently.
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("probe reported working directory %q: %v", got, err)
	}
	wantInfo, err := os.Stat(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Errorf("probe ran in %q, want %q", got, os.TempDir())
	}
}
