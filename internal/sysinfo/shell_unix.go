//go:build !windows

package sysinfo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// detectCurrentShell detects the current shell on Unix systems.
// It returns the shell name and full path.
func detectCurrentShell() (name, path string) {
	// 1. Try /proc/{PPID}/comm (Linux-only, very fast).
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", os.Getppid()))
	if err == nil {
		candidate := cleanShellName(strings.TrimSpace(string(raw)))
		if isKnownShell(candidate) {
			name = candidate
		}
	}

	// 2. If that didn't yield a shell (e.g. macOS, or parent is go test), try ps.
	if name == "" {
		out, err := exec.Command("ps", "-p", fmt.Sprint(os.Getppid()), "-o", "comm=").Output()
		if err == nil {
			candidate := cleanShellName(strings.TrimSpace(string(out)))
			if isKnownShell(candidate) {
				name = candidate
			}
		}
	}

	// 3. Fall back to $SHELL (login shell).
	if name == "" {
		if shellEnv := os.Getenv("SHELL"); shellEnv != "" {
			name = cleanShellName(filepath.Base(shellEnv))
		}
	}

	if name == "" {
		return "unknown", ""
	}

	resolved, err := exec.LookPath(name)
	if err == nil {
		path = resolved
	}

	return name, path
}

// cleanShellName strips the leading dash from login shells (e.g. "-bash" → "bash")
// and takes the basename in case a full path was returned.
func cleanShellName(s string) string {
	s = filepath.Base(s)
	s = strings.TrimPrefix(s, "-")
	return s
}
