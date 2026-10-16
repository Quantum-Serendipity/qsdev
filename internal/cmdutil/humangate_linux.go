package cmdutil

import (
	"fmt"
	"os"
	"strings"
)

// parentPID returns pid's parent from /proc/<pid>/stat.
func parentPID(pid int) (int, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, fmt.Errorf("reading the parent of process %d: %w", pid, err)
	}
	return parseStatPPID(data)
}

// processEnviron returns the environment pid started with, from
// /proc/<pid>/environ: later setenv/unsetenv calls in the process do not
// change it.
func processEnviron(pid int) ([]string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return nil, fmt.Errorf("reading the environment of process %d: %w", pid, err)
	}
	return splitNUL(data), nil
}

// processProgram returns the executable pid runs, from /proc/<pid>/exe ("",
// when it is another user's), and its argv, from /proc/<pid>/cmdline.
func processProgram(pid int) (string, []string, error) {
	exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil && exe == "" {
		return "", nil, fmt.Errorf("reading the program of process %d: %w", pid, err)
	}
	return exe, splitNUL(data), nil
}

// splitNUL splits a NUL-separated /proc/<pid>/environ or cmdline, dropping
// empty entries.
func splitNUL(data []byte) []string {
	return strings.FieldsFunc(string(data), func(r rune) bool { return r == 0 })
}
