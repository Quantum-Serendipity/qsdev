//go:build linux

package bwrap

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// These tests compile nix/ll-restrict/ll-restrict.c and run it for real, so
// they need a C compiler and a kernel with Landlock enabled; otherwise they
// skip. The restricted command is this test binary re-executed in helper mode
// (llHelperEnv), which performs one probe and reports it via its exit status.

const llHelperEnv = "QSDEV_LLRESTRICT_HELPER"

// TestLLRestrictHelperProcess is the probe run inside the Landlock domain. It
// is a no-op unless llHelperEnv is set.
func TestLLRestrictHelperProcess(t *testing.T) {
	mode := os.Getenv(llHelperEnv)
	if mode == "" {
		return
	}
	arg := os.Getenv(llHelperEnv + "_ARG")
	var err error
	switch mode {
	case "write":
		err = os.WriteFile(arg, []byte("written"), 0o600)
	case "connect":
		var c net.Conn
		c, err = net.Dial("unix", "@"+arg)
		if err == nil {
			_ = c.Close()
		}
	case "signal":
		pid, _ := strconv.Atoi(arg)
		err = syscall.Kill(pid, 0)
	default:
		err = fmt.Errorf("unknown helper mode %q", mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// buildLLRestrict compiles the helper and returns its path and the Landlock
// ABI the running kernel enforces, skipping when either is unavailable.
func buildLLRestrict(t *testing.T) (string, int) {
	t.Helper()
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler on PATH")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "nix", "ll-restrict", "ll-restrict.c")
	bin := filepath.Join(t.TempDir(), "ll-restrict")
	if out, err := exec.Command(cc, "-O2", "-Wall", "-Wextra", "-Werror", "-o", bin, src).CombinedOutput(); err != nil {
		t.Fatalf("compiling ll-restrict: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("ll-restrict --version: %v", err)
	}
	abi, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(string(out)), "landlock-abi:"))
	if err != nil {
		t.Fatalf("parsing %q: %v", out, err)
	}
	if abi < 1 {
		t.Skip("Landlock is not enabled on this kernel")
	}
	return bin, abi
}

// runRestricted runs the helper probe under ll-restrict with the given path
// flags and returns the exit code and combined output.
func runRestricted(t *testing.T, bin string, flags []string, mode, arg string) (int, string) {
	t.Helper()
	args := append(append([]string{}, flags...), "--", os.Args[0], "-test.run=^TestLLRestrictHelperProcess$")
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), llHelperEnv+"="+mode, llHelperEnv+"_ARG="+arg)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, string(out)
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), string(out)
	default:
		t.Fatalf("running ll-restrict: %v", err)
		return -1, ""
	}
}

// TestLLRestrict_FileRules is a regression test: a --ro/--rw rule on a regular
// file used to request directory-only rights, which Landlock rejects with
// EINVAL, and the failure was only logged, so the file was unusable inside the
// sandbox.
func TestLLRestrict_FileRules(t *testing.T) {
	bin, _ := buildLLRestrict(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "config")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	// "/" read-only lets the probe binary load; the file rule alone decides
	// whether it may be written.
	code, out := runRestricted(t, bin, []string{"--ro", "/", "--rw", file}, "write", file)
	if code != 0 {
		t.Fatalf("writing a --rw file: exit %d, want 0\n%s", code, out)
	}
	if strings.Contains(out, "ll-restrict:") {
		t.Errorf("unexpected diagnostic: %s", out)
	}

	code, out = runRestricted(t, bin, []string{"--ro", "/", "--ro", file}, "write", file)
	if code != 1 {
		t.Errorf("writing a --ro file: exit %d, want 1 (denied)\n%s", code, out)
	}
}

// TestLLRestrict_ScopesIPC is a regression test: on Landlock ABI 6+ the
// restricted command must not reach an abstract UNIX socket (X11, D-Bus) or
// signal a process outside its domain. Abstract sockets are per network
// namespace, so without scoping a network-allowed hook could reach them.
func TestLLRestrict_ScopesIPC(t *testing.T) {
	bin, abi := buildLLRestrict(t)
	if abi < 6 {
		t.Skipf("Landlock ABI %d has no IPC scoping (needs 6)", abi)
	}

	name := fmt.Sprintf("qsdev-llrestrict-test-%d", os.Getpid())
	ln, err := net.Listen("unix", "@"+name)
	if err != nil {
		t.Fatalf("listening on abstract socket: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	tests := []struct {
		mode, arg string
	}{
		{"connect", name},
		{"signal", strconv.Itoa(os.Getpid())},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			// Control: the probe succeeds without ll-restrict.
			ctl := exec.Command(os.Args[0], "-test.run=^TestLLRestrictHelperProcess$")
			ctl.Env = append(os.Environ(), llHelperEnv+"="+tt.mode, llHelperEnv+"_ARG="+tt.arg)
			if out, err := ctl.CombinedOutput(); err != nil {
				t.Fatalf("unrestricted probe failed: %v\n%s", err, out)
			}
			if code, out := runRestricted(t, bin, []string{"--ro", "/"}, tt.mode, tt.arg); code != 1 {
				t.Errorf("restricted %s probe: exit %d, want 1 (denied)\n%s", tt.mode, code, out)
			}
		})
	}
}
