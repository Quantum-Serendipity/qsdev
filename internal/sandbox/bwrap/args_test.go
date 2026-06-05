//go:build !windows

package bwrap

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestBuildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		cfg   sandbox.SandboxConfig
		tier  sandbox.DegradationTier
		check func(t *testing.T, args []string)
	}{
		{
			name: "linter gets read-only worktree",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "/home/user/project",
				HookCategory: sandbox.CategoryLinter,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				want := []string{"--ro-bind", "/home/user/project", "/home/user/project"}
				if !containsSequence(args, want) {
					t.Errorf("expected --ro-bind for project dir, got args: %v", args)
				}
				// Must not have a bare --bind for the project dir.
				if containsBindRW(args, "/home/user/project") {
					t.Error("linter worktree should be read-only, found --bind (rw)")
				}
			},
		},
		{
			name: "formatter gets read-write worktree",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "/home/user/project",
				HookCategory: sandbox.CategoryFormatter,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				if !containsBindRW(args, "/home/user/project") {
					t.Errorf("expected --bind (rw) for project dir, got args: %v", args)
				}
			},
		},
		{
			name: "network denied for linter",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "/home/user/project",
				HookCategory: sandbox.CategoryLinter,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				if !slices.Contains(args, "--unshare-net") {
					t.Error("expected --unshare-net for linter with deny network")
				}
			},
		},
		{
			name: "network allowed for network-linter",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "/home/user/project",
				HookCategory: sandbox.CategoryNetworkLinter,
				Network:      sandbox.NetworkPolicy{Mode: "filtered"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				if slices.Contains(args, "--unshare-net") {
					t.Error("--unshare-net should be absent for network-linter")
				}
				want := []string{"--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf"}
				if !containsSequence(args, want) {
					t.Error("expected resolv.conf mount for network-allowed category")
				}
			},
		},
		{
			name: "extra mounts included",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "/home/user/project",
				HookCategory: sandbox.CategoryFormatter,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
				Mounts: []sandbox.MountSpec{
					{Source: "/data/cache", Target: "/cache", ReadOnly: true},
					{Source: "/data/out", Target: "/out", ReadOnly: false},
				},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				roWant := []string{"--ro-bind", "/data/cache", "/cache"}
				if !containsSequence(args, roWant) {
					t.Errorf("expected ro-bind mount for /data/cache, got args: %v", args)
				}
				rwWant := []string{"--bind", "/data/out", "/out"}
				if !containsSequence(args, rwWant) {
					t.Errorf("expected bind mount for /data/out, got args: %v", args)
				}
			},
		},
		{
			name: "nix store paths included",
			cfg: sandbox.SandboxConfig{
				ProjectDir:    "/home/user/project",
				HookCategory:  sandbox.CategoryLinter,
				Network:       sandbox.NetworkPolicy{Mode: "deny"},
				NixStorePaths: []string{"/nix/store/abc123-go", "/nix/store/def456-node"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				for _, p := range []string{"/nix/store/abc123-go", "/nix/store/def456-node"} {
					want := []string{"--ro-bind", p, p}
					if !containsSequence(args, want) {
						t.Errorf("expected --ro-bind for nix path %s, got args: %v", p, args)
					}
				}
			},
		},
		{
			name: "empty project dir",
			cfg: sandbox.SandboxConfig{
				ProjectDir:   "",
				HookCategory: sandbox.CategoryLinter,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				// Should still produce valid args with the common flags.
				if !slices.Contains(args, "--unshare-user") {
					t.Error("expected --unshare-user even with empty project dir")
				}
				if !slices.Contains(args, "--die-with-parent") {
					t.Error("expected --die-with-parent even with empty project dir")
				}
			},
		},
		{
			name: "valid nix store paths pass validation",
			cfg: sandbox.SandboxConfig{
				ProjectDir:    "/home/user/project",
				HookCategory:  sandbox.CategoryLinter,
				Network:       sandbox.NetworkPolicy{Mode: "deny"},
				NixStorePaths: []string{"/nix/store/abc-go", "/nix/store/def-node"},
			},
			tier: sandbox.TierFull,
			check: func(t *testing.T, args []string) {
				t.Helper()
				for _, p := range []string{"/nix/store/abc-go", "/nix/store/def-node"} {
					want := []string{"--ro-bind", p, p}
					if !containsSequence(args, want) {
						t.Errorf("expected --ro-bind for nix path %s", p)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args, err := BuildArgs(&tt.cfg, tt.tier)
			if err != nil {
				t.Fatalf("BuildArgs returned unexpected error: %v", err)
			}
			tt.check(t, args)
		})
	}
}

func TestBuildArgs_RejectsDeniedMountTarget(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryFormatter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Mounts: []sandbox.MountSpec{
			{Source: "/data/safe", Target: "/etc/shadow", ReadOnly: true},
		},
	}

	_, err := BuildArgs(&cfg, sandbox.TierFull)
	if err == nil {
		t.Error("expected error for denied mount target /etc/shadow, got nil")
	}
}

func TestBuildArgs_RejectsDeniedMountSource(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryFormatter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Mounts: []sandbox.MountSpec{
			{Source: "/etc/shadow", Target: "/mnt/shadow", ReadOnly: true},
		},
	}

	_, err := BuildArgs(&cfg, sandbox.TierFull)
	if err == nil {
		t.Error("expected error for denied mount source /etc/shadow, got nil")
	}
}

func TestBuildArgs_RejectsDeniedNixStorePath(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		ProjectDir:    "/home/user/project",
		HookCategory:  sandbox.CategoryLinter,
		Network:       sandbox.NetworkPolicy{Mode: "deny"},
		NixStorePaths: []string{"/etc/shadow"},
	}

	_, err := BuildArgs(&cfg, sandbox.TierFull)
	if err == nil {
		t.Error("expected error for denied nix store path /etc/shadow, got nil")
	}
}

func TestBuildArgs_RejectsRelativeNixStorePath(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		ProjectDir:    "/home/user/project",
		HookCategory:  sandbox.CategoryLinter,
		Network:       sandbox.NetworkPolicy{Mode: "deny"},
		NixStorePaths: []string{"relative/path"},
	}

	_, err := BuildArgs(&cfg, sandbox.TierFull)
	if err == nil {
		t.Error("expected error for relative nix store path, got nil")
	}
}

// containsSequence reports whether seq appears as a contiguous subsequence
// within args.
func containsSequence(args, seq []string) bool {
	if len(seq) == 0 {
		return true
	}
	for i := 0; i <= len(args)-len(seq); i++ {
		if slices.Equal(args[i:i+len(seq)], seq) {
			return true
		}
	}
	return false
}

// containsBindRW reports whether args contains a read-write --bind for
// the given path (i.e. --bind <path> <path> NOT preceded by --ro-bind).
func containsBindRW(args []string, path string) bool {
	for i := 0; i < len(args)-2; i++ {
		if args[i] == "--bind" && args[i+1] == path && args[i+2] == path {
			return true
		}
	}
	return false
}
