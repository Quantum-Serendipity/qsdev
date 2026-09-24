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

	bash := Bash(t)

	stubDir := t.TempDir()
	logPath := filepath.Join(stubDir, "calls.log")
	for name, stub := range stubs {
		writeStub(t, bash, stubDir, logPath, name, stub)
	}

	cmd := exec.Command(bash.Path, "-c", command) //nolint:gosec // test-controlled command line
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

// BashShell is the bash a test runs scripts with.
type BashShell struct {
	// Path is the executable to pass to exec.Command.
	Path string
	// Interpreter is bash's own name for itself ($BASH): the path a stub's
	// "#!" line must name. It differs from Path where bash is not a native
	// program: Git for Windows' bash is C:\Program Files\Git\usr\bin\bash.exe
	// to Windows but /usr/bin/bash inside its own POSIX layer, and a "#!"
	// line holding the Windows path breaks at the space and is never found.
	Interpreter string
}

// Shebang returns the "#!" line, with its newline, that runs a stub script
// under this bash.
func (b BashShell) Shebang() string {
	return "#!" + b.Interpreter + "\n"
}

// Bash locates bash for running generated scripts and stub executables,
// skipping the test when bash is unavailable.
func Bash(t testing.TB) BashShell {
	t.Helper()

	path, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	out, err := exec.Command(path, "-c", `printf '%s' "$BASH"`).Output() //nolint:gosec // fixed command line
	if err != nil {
		t.Fatalf("asking %s for its own path: %v", path, err)
	}
	interp := string(out)
	if interp == "" || strings.ContainsAny(interp, " \t") {
		t.Fatalf("bash reports its path as %q, which a \"#!\" line cannot name", interp)
	}
	return BashShell{Path: path, Interpreter: interp}
}

// Quote returns s as a single bash word, single-quoted.
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// QuotePath returns the host path p as a single bash word that names the
// same file inside bash. Separators become forward slashes, which Git for
// Windows' bash resolves from a drive-letter path ("C:/Users/...") just as it
// does from a native one, without backslashes that bash would treat as
// escapes. Elsewhere the path is unchanged.
func QuotePath(p string) string {
	return Quote(filepath.ToSlash(p))
}

// writeStub writes the executable stub for name into dir.
func writeStub(t testing.TB, bash BashShell, dir, logPath, name string, stub Stub) {
	t.Helper()

	stdoutPath := filepath.Join(dir, name+".stdout")
	if err := os.WriteFile(stdoutPath, []byte(stub.Stdout), 0o600); err != nil {
		t.Fatalf("writing stub output: %v", err)
	}
	script := bash.Shebang() +
		"printf '%s\\n' " + Quote(name) + "\" $*\" >> " + QuotePath(logPath) + "\n" +
		"cat " + QuotePath(stdoutPath) + "\n" +
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
