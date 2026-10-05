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

	// denyMasks skips deny entries absent on the host (nothing to expose), so
	// the fixture must be a file that exists everywhere: /etc/shadow does not
	// exist on macOS, where it would be skipped and nothing asserted.
	secret := filepath.Join(t.TempDir(), "shadow")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("creating deny fixture: %v", err)
	}

	cfg := sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryFormatter,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Deny:         []string{secret},
	}

	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs returned unexpected error: %v", err)
	}

	if !containsSequence(args, []string{"--ro-bind", "/dev/null", secret}) {
		t.Errorf("expected %s (a file) masked with /dev/null, got args: %v", secret, args)
	}
	projectIdx := indexOfSequence(args, []string{"--bind", "/home/user/project", "/home/user/project"})
	if projectIdx < 0 {
		t.Fatalf("expected project dir bind, got args: %v", args)
	}
	if maskIdx := indexOfMask(args, secret); maskIdx <= projectIdx {
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

// TestBuildArgs_RejectsInternalMountCollision: RunHook mounts qsdev itself at
// /.qsdev/bin/qsdev, so no project or mount may land on or under /.qsdev,
// where it would replace or shadow that trusted mount.
func TestBuildArgs_RejectsInternalMountCollision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     sandbox.SandboxConfig
		wantErr bool
	}{
		{name: "project dir is the internal root", cfg: sandbox.SandboxConfig{ProjectDir: "/.qsdev"}, wantErr: true},
		{name: "project dir under the internal root", cfg: sandbox.SandboxConfig{ProjectDir: "/.qsdev/bin"}, wantErr: true},
		{
			name:    "mount target replaces the shim",
			cfg:     sandbox.SandboxConfig{Mounts: []sandbox.MountSpec{{Source: "/opt/tool", Target: "/.qsdev/bin/qsdev", ReadOnly: true}}},
			wantErr: true,
		},
		{
			name:    "mount target with a trailing slash",
			cfg:     sandbox.SandboxConfig{Mounts: []sandbox.MountSpec{{Source: "/opt/tool", Target: "/.qsdev/", ReadOnly: true}}},
			wantErr: true,
		},
		{name: "sibling name is not a collision", cfg: sandbox.SandboxConfig{ProjectDir: "/.qsdevx"}, wantErr: false},
		{
			name:    "ordinary mount",
			cfg:     sandbox.SandboxConfig{Mounts: []sandbox.MountSpec{{Source: "/opt/tool", Target: "/opt/tool", ReadOnly: true}}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.cfg.HookCategory = sandbox.CategoryLinter
			_, err := BuildArgs(&tt.cfg, sandbox.TierBwrapOnly)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildArgs error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// guardrailFixture creates a project holding a real .git/hooks, .claude and
// .envrc, and nothing for the other guardrail paths. It returns the project
// and the guardrail paths that exist in it.
func guardrailFixture(t *testing.T) (project string, present []string) {
	t.Helper()
	project = t.TempDir()
	for _, dir := range []string{".git/hooks", ".claude"} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(project, ".envrc"), []byte("use devenv\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, g := range sandbox.GuardrailPaths(project) {
		if _, err := os.Lstat(g); err == nil {
			present = append(present, g)
		}
	}
	if len(present) != 3 {
		t.Fatalf("fixture guardrails = %v, want .git, .claude and .envrc", present)
	}
	return project, present
}

// lastIndexOfMountTarget returns the index of the last bind (--bind/--ro-bind
// SRC DST) whose source is src, or -1.
func lastIndexOfBindSource(args []string, src string) int {
	idx := -1
	for i := 0; i+2 < len(args); i++ {
		if (args[i] == "--bind" || args[i] == "--ro-bind") && args[i+1] == src {
			idx = i
		}
	}
	return idx
}

// TestBuildArgs_GuardrailOverlaysAfterProjectBind is the U05-05 regression: a
// read-write category exposes the whole project, so every existing guardrail
// path is re-bound read-only after the project bind and after every policy
// mount (which therefore cannot re-widen it), but before the deny masks
// (which must still win). A read-only worktree needs no overlay, and a
// guardrail absent on the host gets none (bwrap would create it).
func TestBuildArgs_GuardrailOverlaysAfterProjectBind(t *testing.T) {
	t.Parallel()

	project, present := guardrailFixture(t)
	secret := filepath.Join(project, "secret.txt")
	if err := os.WriteFile(secret, []byte("s"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(project, ".cache")
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, cat := range []sandbox.HookCategory{sandbox.CategoryGenerator, sandbox.CategoryTestRunner, sandbox.CategoryFormatter} {
		t.Run(cat.String(), func(t *testing.T) {
			t.Parallel()
			cfg := sandbox.SandboxConfig{
				ProjectDir:   project,
				HookCategory: cat,
				Network:      sandbox.NetworkPolicy{Mode: "deny"},
				Mounts:       []sandbox.MountSpec{{Source: cache, Target: cache}},
				Deny:         []string{secret},
			}
			args, err := BuildArgs(&cfg, sandbox.TierFull)
			if err != nil {
				t.Fatalf("BuildArgs: %v", err)
			}
			projectIdx := indexOfSequence(args, []string{"--bind", project, project})
			mountIdx := lastIndexOfBindSource(args, cache)
			maskIdx := indexOfMask(args, secret)
			if projectIdx < 0 || mountIdx < 0 || maskIdx < 0 {
				t.Fatalf("missing project bind, policy mount or deny mask: %v", args)
			}
			for _, g := range present {
				i := indexOfSequence(args, []string{"--ro-bind", g, g})
				if i < 0 {
					t.Errorf("guardrail %s not overlaid read-only: %v", g, args)
					continue
				}
				if i < projectIdx || i < mountIdx || i > maskIdx {
					t.Errorf("overlay of %s at %d must follow the project bind (%d) and policy mounts (%d) and precede the deny masks (%d)",
						g, i, projectIdx, mountIdx, maskIdx)
				}
			}
			for _, g := range sandbox.GuardrailPaths(project) {
				if !slices.Contains(present, g) && slices.Contains(args, g) {
					t.Errorf("absent guardrail %s must not appear in args (bwrap would create it): %v", g, args)
				}
			}
		})
	}

	t.Run("linter", func(t *testing.T) {
		t.Parallel()
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryLinter, Network: sandbox.NetworkPolicy{Mode: "deny"}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		for _, g := range present {
			if slices.Contains(args, g) {
				t.Errorf("read-only worktree must get no overlay for %s: %v", g, args)
			}
		}
	})

	t.Run("linter with a writable policy mount over the project", func(t *testing.T) {
		t.Parallel()
		// A writable mount of an ancestor re-exposes the control plane even
		// though the worktree itself is read-only.
		cfg := sandbox.SandboxConfig{
			ProjectDir: project, HookCategory: sandbox.CategoryLinter, Network: sandbox.NetworkPolicy{Mode: "deny"},
			Mounts: []sandbox.MountSpec{{Source: project, Target: project}},
		}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		mountIdx := lastIndexOfBindSource(args, project)
		for _, g := range present {
			if i := indexOfSequence(args, []string{"--ro-bind", g, g}); i < mountIdx {
				t.Errorf("guardrail %s must be overlaid after the writable mount (%d), got %d: %v", g, mountIdx, i, args)
			}
		}
	})
}

// TestBuildArgs_GuardrailOverlayImagesUnderPolicyMount: a policy mount whose
// source is an ancestor of a guardrail places another copy of it inside the
// sandbox, which is overlaid read-only too.
func TestBuildArgs_GuardrailOverlayImagesUnderPolicyMount(t *testing.T) {
	t.Parallel()

	project, present := guardrailFixture(t)
	cfg := sandbox.SandboxConfig{
		ProjectDir:   project,
		HookCategory: sandbox.CategoryGenerator,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Mounts:       []sandbox.MountSpec{{Source: project, Target: "/work"}},
	}
	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	mountIdx := indexOfSequence(args, []string{"--bind", project, "/work"})
	if mountIdx < 0 {
		t.Fatalf("policy mount missing: %v", args)
	}
	for _, g := range present {
		rel, err := filepath.Rel(project, g)
		if err != nil {
			t.Fatal(err)
		}
		image := filepath.Join("/work", rel)
		if i := indexOfSequence(args, []string{"--ro-bind", g, image}); i < mountIdx {
			t.Errorf("guardrail %s must be overlaid at its image %s after the mount (%d), got %d: %v", g, image, mountIdx, i, args)
		}
	}
}

// TestBuildArgs_GuardrailSymlinkOverlaysResolvedTarget: a guardrail that is a
// symlink to another project file is protected where its content lives (the
// link itself is checked by the post-run snapshot); one pointing outside the
// project is not visible in the sandbox and is not bound, which would make
// bwrap fail every hook.
func TestBuildArgs_GuardrailSymlinkOverlaysResolvedTarget(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	real := filepath.Join(project, "config", "claude")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("config", "claude"), filepath.Join(project, ".claude")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "envrc")
	if err := os.WriteFile(outside, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, ".envrc")); err != nil {
		t.Fatal(err)
	}

	cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryGenerator, Network: sandbox.NetworkPolicy{Mode: "deny"}}
	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	if !containsSequence(args, []string{"--ro-bind", real, real}) {
		t.Errorf("symlinked .claude must be overlaid at its in-project target %s: %v", real, args)
	}
	if slices.Contains(args, outside) || slices.Contains(args, filepath.Join(project, ".envrc")) {
		t.Errorf(".envrc resolves outside the project and must not be bound: %v", args)
	}
}

// TestBuildArgs_GuardrailOverlayImagesUnderResolvedMount: a policy mount of
// the symlink-resolved project re-exposes the guardrails under its target
// even though the project is spelled through the link; that image is
// overlaid too.
func TestBuildArgs_GuardrailOverlayImagesUnderResolvedMount(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, ".git", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	cfg := sandbox.SandboxConfig{
		ProjectDir:   link,
		HookCategory: sandbox.CategoryGenerator,
		Network:      sandbox.NetworkPolicy{Mode: "deny"},
		Mounts:       []sandbox.MountSpec{{Source: real, Target: "/work"}},
	}
	args, err := BuildArgs(&cfg, sandbox.TierFull)
	if err != nil {
		t.Fatalf("BuildArgs: %v", err)
	}
	hooks := filepath.Join(link, ".git", "hooks")
	mountIdx := indexOfSequence(args, []string{"--bind", real, "/work"})
	if i := indexOfSequence(args, []string{"--ro-bind", hooks, "/work/.git/hooks"}); i < 0 || i < mountIdx {
		t.Errorf("guardrail %s must be overlaid at /work/.git/hooks after the mount (%d), got %d: %v", hooks, mountIdx, i, args)
	}
}

// TestBuildArgs_HookLogDirStaysWritable is the regression for the generated
// audit and analytics hooks: their log directory lies inside the read-only
// .claude overlay, so it is bound writable again right after the overlays
// (and before the deny masks), for a writable worktree only, and never
// through a symlink.
func TestBuildArgs_HookLogDirStaysWritable(t *testing.T) {
	t.Parallel()

	newProject := func(t *testing.T, logs func(t *testing.T, dir string)) (string, string) {
		t.Helper()
		project := t.TempDir()
		if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		dir := sandbox.HookLogDir(project)
		logs(t, dir)
		return project, dir
	}
	realDir := func(t *testing.T, dir string) {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("writable worktree", func(t *testing.T) {
		t.Parallel()
		project, dir := newProject(t, realDir)
		secret := filepath.Join(project, "secret")
		if err := os.WriteFile(secret, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryGenerator,
			Network: sandbox.NetworkPolicy{Mode: "deny"}, Deny: []string{secret}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		claude := filepath.Join(project, ".claude")
		overlay := indexOfSequence(args, []string{"--ro-bind", claude, claude})
		bind := indexOfSequence(args, []string{"--bind", dir, dir})
		mask := indexOfMask(args, secret)
		if overlay < 0 || bind < overlay || mask < bind {
			t.Errorf("want .claude overlay (%d) < log dir bind (%d) < deny mask (%d): %v", overlay, bind, mask, args)
		}
	})

	t.Run("read-only worktree", func(t *testing.T) {
		t.Parallel()
		project, dir := newProject(t, realDir)
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryLinter, Network: sandbox.NetworkPolicy{Mode: "deny"}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		if slices.Contains(args, dir) {
			t.Errorf("a read-only worktree must not get a writable log dir: %v", args)
		}
	})

	t.Run("symlinked log dir", func(t *testing.T) {
		t.Parallel()
		project, dir := newProject(t, func(t *testing.T, dir string) {
			hooks := filepath.Join(filepath.Dir(dir), "hooks")
			if err := os.Mkdir(hooks, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(hooks, dir); err != nil {
				t.Fatal(err)
			}
		})
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryGenerator, Network: sandbox.NetworkPolicy{Mode: "deny"}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		hooks := filepath.Join(filepath.Dir(dir), "hooks")
		if slices.Contains(args, dir) || slices.Contains(args, hooks) {
			t.Errorf("a symlinked log dir must not be bound writable: %v", args)
		}
	})
}

// TestBuildArgs_DevenvStateStaysWritable is the regression for devenv's
// mutable state (GOPATH, the venv) under the read-only .devenv overlay: it is
// bound writable again after the overlay for a writable worktree only, while
// .devenv itself stays read-only.
func TestBuildArgs_DevenvStateStaysWritable(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	state := sandbox.DevenvStateDir(project)
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	dotDir := filepath.Dir(state)
	for _, tc := range []struct {
		cat      sandbox.HookCategory
		writable bool
	}{
		{sandbox.CategoryTestRunner, true},
		{sandbox.CategoryGenerator, true},
		{sandbox.CategoryFormatter, true},
		{sandbox.CategoryLinter, false},
	} {
		t.Run(tc.cat.String(), func(t *testing.T) {
			t.Parallel()
			cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: tc.cat, Network: sandbox.NetworkPolicy{Mode: "deny"}}
			args, err := BuildArgs(&cfg, sandbox.TierFull)
			if err != nil {
				t.Fatalf("BuildArgs: %v", err)
			}
			bind := indexOfSequence(args, []string{"--bind", state, state})
			if !tc.writable {
				if bind >= 0 {
					t.Errorf("a read-only worktree must not get a writable devenv state: %v", args)
				}
				return
			}
			overlay := indexOfSequence(args, []string{"--ro-bind", dotDir, dotDir})
			if overlay < 0 || bind < overlay {
				t.Errorf("want .devenv overlay (%d) before the state bind (%d): %v", overlay, bind, args)
			}
		})
	}
}

// TestBuildArgs_GuardrailPins: every in-project directory holding a guardrail
// is bound onto itself, before the overlays, so a hook cannot rename it away
// and plant a writable replacement; nothing else is pinned.
func TestBuildArgs_GuardrailPins(t *testing.T) {
	t.Parallel()

	project, _ := guardrailFixture(t)
	git := filepath.Join(project, ".git")
	hooks := filepath.Join(git, "hooks")

	t.Run("writable worktree", func(t *testing.T) {
		t.Parallel()
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryGenerator, Network: sandbox.NetworkPolicy{Mode: "deny"}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		projectIdx := indexOfSequence(args, []string{"--bind", project, project})
		pinIdx := indexOfSequence(args, []string{"--bind", git, git})
		overlayIdx := indexOfSequence(args, []string{"--ro-bind", hooks, hooks})
		if pinIdx < 0 || pinIdx < projectIdx || pinIdx > overlayIdx {
			t.Errorf("pin of %s at %d must follow the project bind (%d) and precede the hooks overlay (%d): %v",
				git, pinIdx, projectIdx, overlayIdx, args)
		}
		var pins []string
		for i := 0; i+2 < len(args); i++ {
			if args[i] == "--bind" && args[i+1] == args[i+2] && args[i+1] != project {
				pins = append(pins, args[i+1])
			}
		}
		// The hook log directory is re-opened only when it exists, which the
		// fixture does not create.
		if !slices.Equal(pins, []string{git}) {
			t.Errorf("pinned %v, want only %s", pins, git)
		}
	})

	t.Run("read-only worktree", func(t *testing.T) {
		t.Parallel()
		cfg := sandbox.SandboxConfig{ProjectDir: project, HookCategory: sandbox.CategoryLinter, Network: sandbox.NetworkPolicy{Mode: "deny"}}
		args, err := BuildArgs(&cfg, sandbox.TierFull)
		if err != nil {
			t.Fatalf("BuildArgs: %v", err)
		}
		if slices.Contains(args, git) {
			t.Errorf("a read-only worktree with no writable mount needs no pin: %v", args)
		}
	})
}
