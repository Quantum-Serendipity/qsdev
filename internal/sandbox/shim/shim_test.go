package shim

import (
	"bytes"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestMain makes the test binary the shim, the way instance.Main makes qsdev
// one: a test re-executes os.Executable() with Argv and gets the real code
// path, ready byte and exec included.
func TestMain(m *testing.M) {
	if Invoked(os.Args) {
		os.Exit(Main(os.Args, os.Stderr))
	}
	os.Exit(m.Run())
}

func TestSandboxPath(t *testing.T) {
	t.Parallel()
	if got, want := SandboxPath(), "/.qsdev/bin/qsdev"; got != want {
		t.Errorf("SandboxPath = %q, want %q", got, want)
	}
	if got, want := SandboxRoot(), "/.qsdev"; got != want {
		t.Errorf("SandboxRoot = %q, want %q", got, want)
	}
}

func TestArgv(t *testing.T) {
	t.Parallel()
	got := Argv("/x/qsdev", 4)
	want := []string{"/x/qsdev", "sandbox", "shim", "--ready-fd", "4", "--"}
	if !slices.Equal(got, want) {
		t.Errorf("Argv = %q, want %q", got, want)
	}
	if !Invoked(append(got, "sh")) {
		t.Error("Invoked does not recognise Argv's own output")
	}
}

func TestShimInvoked(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"shim argv", []string{"qsdev", "sandbox", "shim", "--ready-fd", "3", "--", "sh"}, true},
		{"shim without arguments", []string{"qsdev", "sandbox", "shim"}, true},
		{"sandbox exec", []string{"qsdev", "sandbox", "exec", "--", "sh"}, false},
		{"shim not at position one", []string{"qsdev", "--debug", "sandbox", "shim"}, false},
		{"bare sandbox", []string{"qsdev", "sandbox"}, false},
		{"no arguments", []string{"qsdev"}, false},
		{"empty", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Invoked(tt.args); got != tt.want {
				t.Errorf("Invoked(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

// readyPipe returns a pipe whose write end stands in for the ready fd.
func readyPipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	return r, w
}

// drain closes the parent's write end and returns everything written to r.
func drain(t *testing.T, r, w *os.File) []byte {
	t.Helper()
	_ = w.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// TestShimMain_BadArgs runs Main in-process: a malformed argv fails before
// the write, so it neither writes the ready byte nor execs anything.
func TestShimMain_BadArgs(t *testing.T) {
	tests := []struct {
		name string
		args func(fd string) []string
	}{
		{"no ready-fd flag", func(string) []string { return []string{"q", "sandbox", "shim", "--", "sh"} }},
		{"wrong flag", func(fd string) []string { return []string{"q", "sandbox", "shim", "--fd", fd, "--", "sh"} }},
		{"non-numeric fd", func(string) []string { return []string{"q", "sandbox", "shim", "--ready-fd", "x", "--", "sh"} }},
		{"stdio fd", func(string) []string { return []string{"q", "sandbox", "shim", "--ready-fd", "2", "--", "sh"} }},
		{"missing separator", func(fd string) []string { return []string{"q", "sandbox", "shim", "--ready-fd", fd, "sh"} }},
		{"no command", func(fd string) []string { return []string{"q", "sandbox", "shim", "--ready-fd", fd, "--"} }},
		{"truncated", func(string) []string { return []string{"q", "sandbox", "shim", "--ready-fd"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w := readyPipe(t)
			var stderr bytes.Buffer
			code := Main(tt.args(strconv.Itoa(int(w.Fd()))), &stderr)
			if code != NotStartedExit {
				t.Errorf("Main exit = %d, want %d", code, NotStartedExit)
			}
			if !strings.Contains(stderr.String(), "sandbox shim") {
				t.Errorf("stderr %q does not name the shim", stderr.String())
			}
			if got := drain(t, r, w); len(got) != 0 {
				t.Errorf("ready fd received %q, want nothing", got)
			}
		})
	}
}

// TestShimMain_UnresolvableCommandWritesNothing: a command PATH cannot
// resolve is a launch failure, reported as not-started.
func TestShimMain_UnresolvableCommandWritesNothing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	r, w := readyPipe(t)
	var stderr bytes.Buffer
	args := append(Argv("q", int(w.Fd())), "qsdev-shim-absent-command")
	if code := Main(args, &stderr); code != NotStartedExit {
		t.Errorf("Main exit = %d, want %d", code, NotStartedExit)
	}
	if !strings.Contains(stderr.String(), "qsdev-shim-absent-command") {
		t.Errorf("stderr %q does not name the command", stderr.String())
	}
	if got := drain(t, r, w); len(got) != 0 {
		t.Errorf("ready fd received %q, want nothing", got)
	}
}
