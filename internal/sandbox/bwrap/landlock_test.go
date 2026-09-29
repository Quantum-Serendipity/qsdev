//go:build !windows

package bwrap

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// TestLandlockFlags exercises the flag builder directly, so the Landlock policy
// is verified on every host, not only where ll-restrict is installed. The
// assertions pin exact flag/path pairs: a bare "--ro" or "--rw" is always
// present for /nix/store and /tmp and proves nothing about the worktree.
func TestLandlockFlags(t *testing.T) {
	t.Parallel()

	const project = "/home/user/project"
	tests := []struct {
		name      string
		cfg       sandbox.SandboxConfig
		wantPairs [][2]string
		denyPairs [][2]string
		denyNet   bool
	}{
		{
			name:      "linter gets read-only worktree and no network",
			cfg:       sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryLinter},
			wantPairs: [][2]string{{"--ro", project}, {"--ro", "/nix/store"}, {"--ro", "/etc"}, {"--rw", "/tmp"}},
			denyPairs: [][2]string{{"--rw", project}},
			denyNet:   true,
		},
		{
			name:      "formatter gets read-write worktree and no network",
			cfg:       sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryFormatter},
			wantPairs: [][2]string{{"--rw", project}},
			denyPairs: [][2]string{{"--ro", project}},
			denyNet:   true,
		},
		{
			name:      "network-linter keeps network",
			cfg:       sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryNetworkLinter},
			wantPairs: [][2]string{{"--ro", project}},
			denyNet:   false,
		},
		{
			name: "policy worktree access overrides category default",
			cfg: sandbox.SandboxConfig{
				ProjectDir:     project,
				HookCategory:   sandbox.CategoryTestRunner,
				WorktreeAccess: sandbox.WorktreeAccessReadOnly,
			},
			wantPairs: [][2]string{{"--ro", project}},
			denyPairs: [][2]string{{"--rw", project}},
			denyNet:   false,
		},
		{
			name: "extra mounts keep their access mode",
			cfg: sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryLinter,
				Mounts: []sandbox.MountSpec{
					{Source: "/opt/tools", Target: "/opt/tools", ReadOnly: true},
					{Source: "/var/cache/hook", Target: "/var/cache/hook"},
				},
			},
			wantPairs: [][2]string{{"--ro", "/opt/tools"}, {"--rw", "/var/cache/hook"}},
			denyPairs: [][2]string{{"--rw", "/opt/tools"}},
			denyNet:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flags := landlockFlags(&tt.cfg)

			for _, p := range tt.wantPairs {
				if !containsSequence(flags, p[:]) {
					t.Errorf("flags missing %q %q: %v", p[0], p[1], flags)
				}
			}
			for _, p := range tt.denyPairs {
				if containsSequence(flags, p[:]) {
					t.Errorf("flags must not contain %q %q: %v", p[0], p[1], flags)
				}
			}
			if got := slices.Contains(flags, "--deny-net"); got != tt.denyNet {
				t.Errorf("--deny-net present = %v, want %v: %v", got, tt.denyNet, flags)
			}
		})
	}
}

// TestLandlockFlags_SkipsDenyPaths pins that the policy's deny directives
// (self-referential mounts of sensitive paths) are never granted to Landlock.
func TestLandlockFlags_SkipsDenyPaths(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{HookCategory: sandbox.CategoryFormatter}
	for _, p := range denylist.AllDenyPaths() {
		cfg.Mounts = append(cfg.Mounts, sandbox.MountSpec{Source: p, Target: p, ReadOnly: true})
	}

	flags := landlockFlags(&cfg)

	for _, p := range denylist.AllDenyPaths() {
		if slices.Contains(flags, p) {
			t.Errorf("deny path %q granted to Landlock: %v", p, flags)
		}
	}
}

func TestInjectLandlock_Unavailable(t *testing.T) {
	t.Parallel()
	if sandbox.LLRestrictBin() != "" {
		t.Skip("ll-restrict is available, cannot test unavailable path")
	}

	original := []string{"/usr/bin/hook", "--arg1"}
	result := InjectLandlock(original, &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
	})

	if len(result) != len(original) {
		t.Errorf("expected command unchanged, got %v", result)
	}
}
