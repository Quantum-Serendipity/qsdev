// Package shim is the first program a sandbox backend starts in place of a
// hook: `qsdev sandbox shim --ready-fd N -- <hook argv…>`. It resolves the
// hook, writes one byte to fd N, closes it and execs the hook. The parent
// holds the read end of that fd (sandbox.LaunchGuard), so a byte proves the
// sandbox started and reached the hook, and its absence proves the sandbox's
// own setup failed, whatever exit code that failure produced.
//
// The shim does no policy, config, logging or update-check work: anything
// that could print or fail would blur that signal. It is dispatched before
// any of qsdev's startup (instance.Main), and its argv is an ABI between the
// parent and the child, which are always the same binary.
package shim

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// SandboxPath returns where a namespace backend mounts the host's qsdev
// binary inside the sandbox: /.<app>/bin/<app>, /.qsdev/bin/qsdev under the
// default branding. It derives from the branding so a rebranded binary is
// mounted under its own name, and so the dot-directory name keeps one owner.
func SandboxPath() string {
	return path.Join(SandboxRoot(), "bin", branding.Get().AppName)
}

// SandboxRoot returns the in-sandbox directory that holds SandboxPath:
// /.<app>, /.qsdev under the default branding. It is reserved for the
// backend's own mounts; no project or policy mount may land on or under it.
func SandboxRoot() string {
	return "/." + branding.Get().AppName
}

const readyFDFlag = "--ready-fd"

// NotStartedExit is the shim's exit code for any failure before the ready
// byte is written: malformed arguments, an unresolvable command or an
// unusable ready fd. The parent reads the missing byte, not this code.
const NotStartedExit = 127

// execFailedExit is the exit code when exec fails after the ready byte was
// written: the hook was reached but could not run, which blocks.
const execFailedExit = 2

// minReadyFD is the lowest ready fd accepted: 0-2 are stdio.
const minReadyFD = 3

// Argv returns the argv prefix that runs exe as the shim with readyFD as the
// ready fd; the hook's argv follows it.
func Argv(exe string, readyFD int) []string {
	return []string{exe, "sandbox", "shim", readyFDFlag, strconv.Itoa(readyFD), "--"}
}

// Invoked reports whether args (os.Args) invokes the shim.
func Invoked(args []string) bool {
	return len(args) >= 3 && args[1] == "sandbox" && args[2] == "shim"
}

// Main runs the shim on args (os.Args) and returns the exit code when it does
// not exec. Failures before the ready byte return NotStartedExit without
// writing it; an exec failure after it returns 2.
func Main(args []string, stderr io.Writer) int {
	fd, argv, err := parse(args)
	if err != nil {
		return fail(stderr, NotStartedExit, err)
	}
	path, err := procexec.LookPath(argv[0])
	if err != nil {
		return fail(stderr, NotStartedExit, fmt.Errorf("resolving %q: %w", argv[0], err))
	}
	if err := signalReady(fd); err != nil {
		return fail(stderr, NotStartedExit, err)
	}
	err = syscall.Exec(path, argv, os.Environ()) //nolint:gosec // argv is the hook the parent was asked to run
	return fail(stderr, execFailedExit, fmt.Errorf("exec %s: %w", path, err))
}

// parse splits args into the ready fd and the hook's argv.
func parse(args []string) (int, []string, error) {
	if len(args) < 7 || args[3] != readyFDFlag || args[5] != "--" {
		return 0, nil, fmt.Errorf("usage: sandbox shim %s N -- command [args...]", readyFDFlag)
	}
	fd, err := strconv.Atoi(args[4])
	if err != nil || fd < minReadyFD {
		return 0, nil, fmt.Errorf("invalid %s %q", readyFDFlag, args[4])
	}
	return fd, args[6:], nil
}

// signalReady writes the ready byte to fd and closes it, so the hook does
// not inherit it.
func signalReady(fd int) error {
	f := os.NewFile(uintptr(fd), "ready-fd")
	if f == nil {
		return fmt.Errorf("invalid ready fd %d", fd)
	}
	if _, err := f.Write([]byte{'R'}); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing ready fd %d: %w", fd, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing ready fd %d: %w", fd, err)
	}
	return nil
}

func fail(stderr io.Writer, code int, err error) int {
	_, _ = fmt.Fprintf(stderr, "qsdev sandbox shim: %v\n", err)
	return code
}

// HostExecutable returns the absolute, symlink-free path of the running
// binary: the file a backend mounts at SandboxPath or runs as the shim.
func HostExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating qsdev executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving qsdev executable %s: %w", exe, err)
	}
	return resolved, nil
}
