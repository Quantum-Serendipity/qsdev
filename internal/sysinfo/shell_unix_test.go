//go:build !windows

package sysinfo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanShellName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain name", "bash", "bash"},
		{"login dash prefix", "-bash", "bash"},
		{"full path", "/bin/bash", "bash"},
		{"full path with dash", "/usr/bin/-zsh", "zsh"},
		{"nix store path", "/nix/store/abc123-bash-5.2/bin/bash", "bash"},
		{"empty string", "", "."},
		{"just dash", "-", ""},
		{"double dash prefix", "--fish", "-fish"},
		{"name with dots", "bash.exe", "bash.exe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cleanShellName(tc.input)
			if got != tc.want {
				t.Errorf("cleanShellName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// shellPath prefers an absolute $SHELL naming the detected shell, so a login
// shell that is not on PATH still resolves, and otherwise looks the name up.
func TestShellPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	zsh := filepath.Join(dir, "zsh")
	if err := os.WriteFile(zsh, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	notExec := filepath.Join(dir, "bash")
	if err := os.WriteFile(notExec, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	onPath := func(name string) (string, error) { return "/path/" + name, nil }
	notOnPath := func(string) (string, error) { return "", errors.New("not found") }

	tests := []struct {
		name, shell, env string
		lookPath         func(string) (string, error)
		want             string
	}{
		{"absolute $SHELL off PATH", "zsh", zsh, notOnPath, zsh},
		{"absolute $SHELL wins over PATH", "zsh", zsh, onPath, zsh},
		{"$SHELL names another shell", "bash", zsh, onPath, "/path/bash"},
		{"relative $SHELL", "zsh", "zsh", onPath, "/path/zsh"},
		{"$SHELL not executable", "bash", notExec, onPath, "/path/bash"},
		{"$SHELL missing", "zsh", filepath.Join(dir, "none", "zsh"), onPath, "/path/zsh"},
		{"nothing resolves", "zsh", "", notOnPath, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(k string) string {
				if k == "SHELL" {
					return tt.env
				}
				return ""
			}
			if got := shellPath(tt.shell, getenv, tt.lookPath); got != tt.want {
				t.Errorf("shellPath(%q) with SHELL=%q = %q, want %q", tt.shell, tt.env, got, tt.want)
			}
		})
	}
}
