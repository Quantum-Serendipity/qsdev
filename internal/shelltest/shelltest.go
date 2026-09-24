// Package shelltest runs generated shell command lines under bash in tests,
// with stub executables standing in for the tools they invoke, so a test can
// check how a command behaves when a tool fails, rather than only how it is
// spelled.
package shelltest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Stub is a stand-in executable: it records its invocation, prints Stdout,
// runs Script (bash, with the invocation's arguments in "$@") and exits with
// Exit.
type Stub struct {
	Stdout string
	Script string
	Exit   int
}

// Result is the outcome of Run.
type Result struct {
	Exit   int
	Output string   // combined stdout and stderr
	Calls  []string // one "name arg..." line per stub invocation, in order
}

// Run runs command with `bash -c` in dir. The stubs are placed ahead of the
// real PATH, so they shadow the tools they name while standard utilities
// (find, grep, mktemp, ...) stay available. bash runs without -e or pipefail:
// a command must enforce its own failure semantics, since the shell a CI
// system runs it in may not. Run skips the test when bash is unavailable.
func Run(t testing.TB, dir, command string, stubs map[string]Stub) Result {
	t.Helper()

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}

	stubDir := t.TempDir()
	logPath := filepath.Join(stubDir, "calls.log")
	for name, stub := range stubs {
		writeStub(t, bash, stubDir, logPath, name, stub)
	}

	cmd := exec.Command(bash, "-c", command) //nolint:gosec // test-controlled command line
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()

	res := Result{Output: string(out)}
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		res.Exit = exitErr.ExitCode()
	case err != nil:
		t.Fatalf("running %q: %v", command, err)
	}

	if data, err := os.ReadFile(logPath); err == nil { //nolint:gosec // test temp file
		res.Calls = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	}
	return res
}

// writeStub writes the executable stub for name into dir.
func writeStub(t testing.TB, bash, dir, logPath, name string, stub Stub) {
	t.Helper()

	stdoutPath := filepath.Join(dir, name+".stdout")
	if err := os.WriteFile(stdoutPath, []byte(stub.Stdout), 0o600); err != nil {
		t.Fatalf("writing stub output: %v", err)
	}
	script := "#!" + bash + "\n" +
		"printf '%s\\n' " + strconv.Quote(name) + "\" $*\" >> " + strconv.Quote(logPath) + "\n" +
		"cat " + strconv.Quote(stdoutPath) + "\n" +
		stub.Script + "\n" +
		"exit " + strconv.Itoa(stub.Exit) + "\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o700); err != nil { //nolint:gosec // executable test stub
		t.Fatalf("writing stub %s: %v", name, err)
	}
}

// WriteTree creates dir/name with the given content for every entry of
// files, creating parent directories as needed. It returns dir.
func WriteTree(t testing.TB, dir string, files map[string]string) string {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return dir
}
