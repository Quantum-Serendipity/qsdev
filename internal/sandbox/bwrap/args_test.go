//go:build !windows

package bwrap

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
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

// TestBuildArgs_DefaultPolicyDenyDoesNotBreakExec is the primary regression
// for the exec-is-non-functional defect: DefaultPolicy denies every deny-list
// path (including /etc/shadow), and BuildArgs must succeed and MASK each one
// that exists on the host (never bind/expose it).
func TestBuildArgs_DefaultPolicyDenyDoesNotBreakExec(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Deny:         denylist.AllDenyPaths(),
	}

	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs must not error on the default deny list, got: %v", err)
	}

	// Every existing deny path must be MASKED (empty tmpfs or ro /dev/null),
	// never bound with its real contents.
	for _, p := range denylist.AllDenyPaths() {
		if containsSequence(args, []string{"--bind", p, p}) ||
			containsSequence(args, []string{"--ro-bind", p, p}) {
			t.Errorf("deny path %q must not be self-bound/exposed, got args: %v", p, args)
		}
		if _, statErr := os.Lstat(p); statErr != nil {
			continue // absent on this host: nothing to expose, nothing to mask
		}
		if !containsMask(args, p) {
			t.Errorf("deny path %q must be masked, got args: %v", p, args)
		}
	}
	// Sanity: the safe system files are still mounted.
	if !containsSequence(args, []string{"--ro-bind", "/etc/passwd", "/etc/passwd"}) {
		t.Errorf("expected /etc/passwd to still be mounted, got args: %v", args)
	}
}

// TestBuildArgs_MasksDenyPaths verifies the defense-in-depth mask is emitted
// AFTER every bind, so the denied path stays hidden even if a broader bind
// exposed one of its ancestors.
func TestBuildArgs_MasksDenyPaths(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryFormatter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Deny:         []string{"/etc/shadow"},
	}

	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs returned unexpected error: %v", err)
	}

	if !containsSequence(args, []string{"--ro-bind", "/dev/null", "/etc/shadow"}) {
		t.Errorf("expected /etc/shadow (a file) masked with /dev/null, got args: %v", args)
	}
	projectIdx := indexOfSequence(args, []string{"--bind", "/home/user/project", "/home/user/project"})
	if projectIdx < 0 {
		t.Fatalf("expected project dir bind, got args: %v", args)
	}
	if maskIdx := indexOfMask(args, "/etc/shadow"); maskIdx <= projectIdx {
		t.Errorf("deny mask (idx %d) must be emitted after the project bind (idx %d), got args: %v", maskIdx, projectIdx, args)
	}
}

// policyDenyFixture creates a directory holding a secret directory and a
// secret file that a policy denies, plus a public file that stays visible.
func policyDenyFixture(t *testing.T) (root, secretDir, secretFile string) {
	t.Helper()
	root = t.TempDir()
	secretDir = filepath.Join(root, "secrets")
	secretFile = filepath.Join(root, "token.txt")
	if err := os.Mkdir(secretDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(secretDir, "secret.txt"): "TOPSECRET",
		secretFile:                             "TOKEN",
		filepath.Join(root, "public.txt"):      "PUBLIC",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root, secretDir, secretFile
}

// TestBuildArgs_MasksCustomPolicyDenyPaths is the regression for the inverted
// deny: a filesystem.deny entry that is NOT on the built-in credential list
// used to fall through to the mount branch and be emitted as
// `--ro-bind <path> <path>`, exposing exactly what the policy denied.
func TestBuildArgs_MasksCustomPolicyDenyPaths(t *testing.T) {
	t.Parallel()

	_, secretDir, secretFile := policyDenyFixture(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	cfg := sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Deny:         []string{secretDir, secretFile, missing},
	}

	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs returned unexpected error: %v", err)
	}

	for _, p := range []string{secretDir, secretFile} {
		if containsSequence(args, []string{"--ro-bind", p, p}) || containsSequence(args, []string{"--bind", p, p}) {
			t.Errorf("policy deny path %q must never be bound, got args: %v", p, args)
		}
	}
	if !containsSequence(args, []string{"--tmpfs", secretDir}) {
		t.Errorf("expected denied directory masked with tmpfs, got args: %v", args)
	}
	if !containsSequence(args, []string{"--ro-bind", "/dev/null", secretFile}) {
		t.Errorf("expected denied file masked with /dev/null, got args: %v", args)
	}
	if slices.Contains(args, missing) {
		t.Errorf("a deny path absent on the host must not become a mount point, got args: %v", args)
	}
}

// TestBuildArgs_MasksDenyPathUnderAncestorMount verifies that a deny entry
// re-exposed through an extra mount of one of its ancestors is masked at the
// location where that mount places it inside the sandbox.
func TestBuildArgs_MasksDenyPathUnderAncestorMount(t *testing.T) {
	t.Parallel()

	root, secretDir, secretFile := policyDenyFixture(t)

	cfg := sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Mounts:       []sandbox.MountSpec{{Source: root, Target: "/mnt/data", ReadOnly: true}},
		Deny:         []string{secretDir, secretFile},
	}

	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs returned unexpected error: %v", err)
	}

	bindIdx := indexOfSequence(args, []string{"--ro-bind", root, "/mnt/data"})
	if bindIdx < 0 {
		t.Fatalf("expected ancestor mount, got args: %v", args)
	}
	if i := indexOfSequence(args, []string{"--tmpfs", "/mnt/data/secrets"}); i <= bindIdx {
		t.Errorf("denied directory must be masked at its mounted location after the bind, got args: %v", args)
	}
	if i := indexOfSequence(args, []string{"--ro-bind", "/dev/null", "/mnt/data/token.txt"}); i <= bindIdx {
		t.Errorf("denied file must be masked at its mounted location after the bind, got args: %v", args)
	}
}

func TestBuildArgs_RejectsMountExposingPolicyDenyPath(t *testing.T) {
	t.Parallel()

	_, secretDir, _ := policyDenyFixture(t)

	tests := []struct {
		name  string
		mount sandbox.MountSpec
	}{
		{"deny path itself elsewhere", sandbox.MountSpec{Source: secretDir, Target: "/mnt/s", ReadOnly: true}},
		{"deny path itself in place", sandbox.MountSpec{Source: secretDir, Target: secretDir, ReadOnly: true}},
		{"descendant of deny path", sandbox.MountSpec{Source: filepath.Join(secretDir, "secret.txt"), Target: "/mnt/s", ReadOnly: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := sandbox.SandboxConfig{
				HookCategory: sandbox.CategoryLinter,
				Mounts:       []sandbox.MountSpec{tt.mount},
				Deny:         []string{secretDir},
			}
			if _, err := BuildArgs(&cfg, sandbox.TierFull); err == nil {
				t.Errorf("expected an error for a mount exposing policy deny path %q", secretDir)
			}
		})
	}
}

func TestBuildArgs_RejectsRelativeDenyPath(t *testing.T) {
	t.Parallel()

	cfg := sandbox.SandboxConfig{HookCategory: sandbox.CategoryLinter, Deny: []string{"secrets"}}
	if _, err := BuildArgs(&cfg, sandbox.TierFull); err == nil {
		t.Error("expected an error for a relative deny path, got nil")
	}
}

// TestBuildArgs_NetworkModeIsAuthoritative is the regression for the network
// OR: an explicit "deny" used to be ignored for categories whose default
// allows the network, and "allow" was ignored nowhere but Landlock.
func TestBuildArgs_NetworkModeIsAuthoritative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		category     sandbox.HookCategory
		mode         string
		wantIsolated bool
	}{
		{"deny overrides test-runner default", sandbox.CategoryTestRunner, "deny", true},
		{"deny overrides network-linter default", sandbox.CategoryNetworkLinter, "deny", true},
		{"allow opens a linter", sandbox.CategoryLinter, "allow", false},
		{"filtered shares the network", sandbox.CategoryTestRunner, "filtered", false},
		{"empty mode uses category default (isolated)", sandbox.CategoryGenerator, "", true},
		{"empty mode uses category default (network)", sandbox.CategoryTestRunner, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := sandbox.SandboxConfig{HookCategory: tt.category, Network: sandbox.NetworkPolicy{Mode: tt.mode}}
			args, err := BuildArgs(&cfg, sandbox.TierFull)
			if err != nil {
				t.Fatalf("BuildArgs: %v", err)
			}
			if got := slices.Contains(args, "--unshare-net"); got != tt.wantIsolated {
				t.Errorf("--unshare-net present = %v, want %v; args: %v", got, tt.wantIsolated, args)
			}
			resolv := containsSequence(args, []string{"--ro-bind", "/etc/resolv.conf", "/etc/resolv.conf"})
			if resolv == tt.wantIsolated {
				t.Errorf("resolv.conf mounted = %v, want %v", resolv, !tt.wantIsolated)
			}
			// Landlock's --deny-net must agree with bwrap's decision.
			if got := slices.Contains(landlockFlags(&cfg), "--deny-net"); got != tt.wantIsolated {
				t.Errorf("landlock --deny-net = %v, want %v", got, tt.wantIsolated)
			}
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
	return indexOfSequence(args, seq) >= 0
}

// indexOfSequence returns the start index of the first contiguous occurrence of
// seq within args, or -1 if absent.
func indexOfSequence(args, seq []string) int {
	if len(seq) == 0 {
		return 0
	}
	for i := 0; i <= len(args)-len(seq); i++ {
		if slices.Equal(args[i:i+len(seq)], seq) {
			return i
		}
	}
	return -1
}

// containsMask reports whether args masks path with either an empty tmpfs
// (--tmpfs <path>) or a read-only /dev/null bind (--ro-bind /dev/null <path>).
func containsMask(args []string, path string) bool {
	return indexOfMask(args, path) >= 0
}

// indexOfMask returns the start index of the mask for path, or -1 if absent.
func indexOfMask(args []string, path string) int {
	if i := indexOfSequence(args, []string{"--tmpfs", path}); i >= 0 {
		return i
	}
	return indexOfSequence(args, []string{"--ro-bind", "/dev/null", path})
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
