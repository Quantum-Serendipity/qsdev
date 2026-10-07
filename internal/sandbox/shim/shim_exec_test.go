//go:build !windows

package shim

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runShim re-executes the test binary as the shim with the ready pipe at fd
// 3 and returns the exit code and what the pipe received.
func runShim(t *testing.T, env []string, argv ...string) (int, []byte) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r, w := readyPipe(t)
	cmd := exec.Command(exe, append(Argv(exe, 3), argv...)[1:]...) //nolint:gosec // the test binary itself
	cmd.Env = append(os.Environ(), env...)
	cmd.ExtraFiles = []*os.File{w}
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	case err != nil:
		t.Fatal(err)
	}
	return code, drain(t, r, w)
}

// TestShimMain_WritesReadyThenExecs: the shim writes exactly one byte, closes
// the ready fd before exec (the target's write to fd 3 fails), and the
// target's exit code is the process's.
func TestShimMain_WritesReadyThenExecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		argv []string
		want int
	}{
		{"exit 7", []string{"sh", "-c", "printf X >&3 2>/dev/null; exit 7"}, 7},
		{"exit 0", []string{"sh", "-c", "printf X >&3 2>/dev/null; exit 0"}, 0},
		{"exit 1", []string{"sh", "-c", "exit 1"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, ready := runShim(t, nil, tt.argv...)
			if code != tt.want {
				t.Errorf("exit = %d, want %d", code, tt.want)
			}
			if len(ready) != 1 {
				t.Errorf("ready fd received %q, want exactly one byte", ready)
			}
		})
	}
}

// TestShimMain_ResolvesBareCommand: a bare argv[0] is resolved through PATH
// before the ready byte is written.
func TestShimMain_ResolvesBareCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	script := filepath.Join(dir, "qsdev-shim-probe")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 9\n"), 0o755); err != nil { //nolint:gosec // executable test fixture
		t.Fatal(err)
	}
	code, ready := runShim(t, []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")}, "qsdev-shim-probe")
	if code != 9 {
		t.Errorf("exit = %d, want 9", code)
	}
	if len(ready) != 1 {
		t.Errorf("ready fd received %q, want exactly one byte", ready)
	}
}

// TestShimMain_UnresolvableCommandExits127 runs the not-started path in a
// real child: exit 127 and an empty ready pipe.
func TestShimMain_UnresolvableCommandExits127(t *testing.T) {
	t.Parallel()
	code, ready := runShim(t, nil, "qsdev-shim-absent-command")
	if code != NotStartedExit {
		t.Errorf("exit = %d, want %d", code, NotStartedExit)
	}
	if len(ready) != 0 {
		t.Errorf("ready fd received %q, want nothing", ready)
	}
}
