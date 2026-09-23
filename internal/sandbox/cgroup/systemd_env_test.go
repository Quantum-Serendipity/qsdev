//go:build linux

package cgroup

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestHookEnvironment pins that the systemd-run backend always filters the
// hook's environment, including when the caller supplies none and the process
// environment is the source, while still giving systemd-run the user-bus
// variables it needs to reach the user manager. It uses t.Setenv, so it is not
// parallel.
func TestHookEnvironment(t *testing.T) {
	t.Setenv("AWS_SECRET_ACCESS_KEY", "leaked-secret")
	t.Setenv("GITHUB_TOKEN", "leaked-token")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/4242")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/4242/bus")

	tests := []struct {
		name    string
		env     map[string]string
		want    []string
		notWant []string
	}{
		{
			name:    "process environment is filtered",
			env:     nil,
			want:    []string{"PATH=/usr/bin", "XDG_RUNTIME_DIR=/run/user/4242", "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/4242/bus"},
			notWant: []string{"AWS_SECRET_ACCESS_KEY=leaked-secret", "GITHUB_TOKEN=leaked-token"},
		},
		{
			name:    "supplied environment is filtered",
			env:     map[string]string{"HOME": "/home/u", "NPM_TOKEN": "leaked"},
			want:    []string{"HOME=/home/u", "XDG_RUNTIME_DIR=/run/user/4242"},
			notWant: []string{"NPM_TOKEN=leaked", "PATH=/usr/bin"},
		},
		{
			name:    "empty environment never inherits",
			env:     map[string]string{},
			want:    []string{"XDG_RUNTIME_DIR=/run/user/4242"},
			notWant: []string{"AWS_SECRET_ACCESS_KEY=leaked-secret", "PATH=/usr/bin"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hookEnvironment(&sandbox.SandboxConfig{Environment: tt.env, HookCategory: sandbox.CategoryLinter})

			if got == nil {
				t.Fatal("hookEnvironment returned nil, which os/exec treats as inherit-everything")
			}
			for _, w := range tt.want {
				if !slices.Contains(got, w) {
					t.Errorf("env missing %q: %v", w, got)
				}
			}
			for _, nw := range tt.notWant {
				if slices.Contains(got, nw) {
					t.Errorf("env must not contain %q: %v", nw, got)
				}
			}
		})
	}
}

// TestSystemdRunBackend_AvailableRequiresUserSession pins that an existing
// systemd-run binary is not enough: without a user bus every `systemd-run
// --user` fails, so the backend must report itself unavailable. It uses
// t.Setenv, so it is not parallel.
func TestSystemdRunBackend_AvailableRequiresUserSession(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "systemd-run")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // test stub must be executable
		t.Fatalf("writing stub: %v", err)
	}
	runtimeWithBus := t.TempDir()
	if err := os.WriteFile(filepath.Join(runtimeWithBus, "bus"), nil, 0o600); err != nil {
		t.Fatalf("writing bus stub: %v", err)
	}

	tests := []struct {
		name       string
		dbus       string
		runtimeDir string
		wantErr    bool
	}{
		{name: "no session variables", wantErr: true},
		{name: "runtime dir without bus socket", runtimeDir: t.TempDir(), wantErr: true},
		{name: "runtime dir with bus socket", runtimeDir: runtimeWithBus, wantErr: false},
		{name: "explicit bus address", dbus: "unix:path=/run/user/1/bus", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DBUS_SESSION_BUS_ADDRESS", tt.dbus)
			t.Setenv("XDG_RUNTIME_DIR", tt.runtimeDir)

			err := NewSystemdRunBackend(bin).Available()
			if (err != nil) != tt.wantErr {
				t.Errorf("Available() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSystemdRunBackend_RunHook_ForwardsStdin runs a real transient scope and
// pins that the hook receives its stdin payload byte for byte. It is skipped on
// hosts without a usable systemd user session.
func TestSystemdRunBackend_RunHook_ForwardsStdin(t *testing.T) {
	t.Parallel()

	bin, err := exec.LookPath("systemd-run")
	if err != nil {
		t.Skip("systemd-run not installed")
	}
	if err := sandbox.UserScopeUsable(bin); err != nil {
		t.Skipf("no systemd user session: %v", err)
	}
	cat, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not found")
	}

	payload := `{"tool_name":"Bash","tool_input":{"command":"ls"}}`
	cfg := &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		HookCommand:  []string{cat},
		Resources:    sandbox.DefaultResourceLimits(),
		ExecOpts:     sandbox.ExecOpts{Stdin: strings.NewReader(payload)},
	}

	res, err := NewSystemdRunBackend(bin).RunHook(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunHook: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr %q", res.ExitCode, res.Stderr)
	}
	if got := string(res.Stdout); got != payload {
		t.Errorf("hook stdout = %q, want the stdin payload %q", got, payload)
	}
}
