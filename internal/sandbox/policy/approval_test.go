//go:build !windows

package policy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// newPolicyDir writes policy.nix (and a shared.nix it may import) to a fresh
// directory and returns the policy path.
func newPolicyDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.nix")
	writePolicyFile(t, policyPath, "import ./shared.nix")
	writePolicyFile(t, filepath.Join(dir, "shared.nix"), "{ backend = \"auto\"; }")
	return policyPath
}

func mustSnapshot(t *testing.T, policyPath string) *Snapshot {
	t.Helper()
	snap, err := ReadSnapshot(policyPath)
	if err != nil {
		t.Fatalf("ReadSnapshot(%s): %v", policyPath, err)
	}
	return snap
}

func TestReadSnapshot_DigestCoversEvaluatedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		change      func(t *testing.T, dir, policyPath string)
		wantChanged bool
	}{
		{
			name:        "policy file edited",
			change:      func(t *testing.T, _, p string) { t.Helper(); writePolicyFile(t, p, "{ }") },
			wantChanged: true,
		},
		{
			name: "sibling nix file edited",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				writePolicyFile(t, filepath.Join(dir, "shared.nix"), "{ backend = \"none\"; }")
			},
			wantChanged: true,
		},
		{
			name: "sibling nix file added",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				writePolicyFile(t, filepath.Join(dir, "extra.nix"), "{ }")
			},
			wantChanged: true,
		},
		{
			name: "non-nix file added",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				writePolicyFile(t, filepath.Join(dir, "notes.txt"), "hello")
			},
			wantChanged: false,
		},
		{
			name: "symlinked nix sibling is not part of the snapshot",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				secret := filepath.Join(t.TempDir(), "secret")
				writePolicyFile(t, secret, "top secret")
				if err := os.Symlink(secret, filepath.Join(dir, "leak.nix")); err != nil {
					t.Fatalf("symlink: %v", err)
				}
			},
			wantChanged: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policyPath := newPolicyDir(t)
			before := mustSnapshot(t, policyPath)
			tt.change(t, filepath.Dir(policyPath), policyPath)
			after := mustSnapshot(t, policyPath)

			if changed := before.Digest != after.Digest; changed != tt.wantChanged {
				t.Errorf("digest changed = %v, want %v", changed, tt.wantChanged)
			}
			if slices.Contains(after.FileNames(), "leak.nix") {
				t.Error("a symlinked sibling was read into the snapshot")
			}
		})
	}
}

func TestReadSnapshot_Refuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{
			name:  "policy is a directory",
			setup: func(t *testing.T) string { t.Helper(); return t.TempDir() },
		},
		{
			name: "policy is too large",
			setup: func(t *testing.T) string {
				t.Helper()
				p := filepath.Join(t.TempDir(), "policy.nix")
				writePolicyFile(t, p, strings.Repeat(" ", maxSnapshotFileSize+1))
				return p
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if snap, err := ReadSnapshot(tt.setup(t)); err == nil {
				t.Errorf("ReadSnapshot = %+v, want an error", snap)
			}
		})
	}
}

func TestApprovalStore_CheckAndApprove(t *testing.T) {
	t.Parallel()

	policyPath := newPolicyDir(t)
	store := NewApprovalStore(filepath.Join(t.TempDir(), "approvals.json"))

	snap := mustSnapshot(t, policyPath)
	if err := store.Check(snap); !errors.Is(err, ErrPolicyNotApproved) {
		t.Fatalf("Check before approval = %v, want ErrPolicyNotApproved", err)
	}
	if err := store.Approve(snap, time.Now()); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if err := store.Check(snap); err != nil {
		t.Fatalf("Check after approval = %v, want nil", err)
	}

	// Approving a second policy keeps the first.
	other := mustSnapshot(t, newPolicyDir(t))
	if err := store.Approve(other, time.Now()); err != nil {
		t.Fatalf("Approve other: %v", err)
	}
	if err := store.Check(snap); err != nil {
		t.Errorf("first approval lost after approving another policy: %v", err)
	}

	// Any change needs a new approval.
	writePolicyFile(t, filepath.Join(filepath.Dir(policyPath), "shared.nix"), "{ backend = \"none\"; }")
	err := store.Check(mustSnapshot(t, policyPath))
	if !errors.Is(err, ErrPolicyNotApproved) || !strings.Contains(err.Error(), "changed since it was approved") {
		t.Errorf("Check after edit = %v, want ErrPolicyNotApproved naming the change", err)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("approval store mode = %v, want no group/other access", perm)
	}
}

func TestApprovalStore_CorruptStoreIsAnError(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "approvals.json")
	writePolicyFile(t, storePath, "{not json")
	store := NewApprovalStore(storePath)
	snap := mustSnapshot(t, newPolicyDir(t))

	if err := store.Check(snap); err == nil {
		t.Error("Check with a corrupt store = nil, want an error")
	}
	if err := store.Approve(snap, time.Now()); err == nil {
		t.Error("Approve over a corrupt store = nil, want an error (it would drop every approval)")
	}
}

// TestCompiler_RequiresApproval pins that a policy file is never evaluated
// until its exact content is approved, and that an edit after approval blocks
// it again (and is not served from the cache).
func TestCompiler_RequiresApproval(t *testing.T) {
	t.Parallel()

	policyPath := newPolicyDir(t)
	store := NewApprovalStore(filepath.Join(t.TempDir(), "approvals.json"))
	ev := &fakeEval{out: `{"backend":"bubblewrap"}`}
	c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: store.Check}
	ctx := context.Background()

	if spec, err := c.compile(ctx, policyPath); !errors.Is(err, ErrPolicyNotApproved) {
		t.Fatalf("compile unapproved = %+v, %v; want ErrPolicyNotApproved", spec, err)
	}
	if ev.calls != 0 {
		t.Fatalf("an unapproved policy was evaluated %d times", ev.calls)
	}

	if err := store.Approve(mustSnapshot(t, policyPath), time.Now()); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	spec, err := c.compile(ctx, policyPath)
	if err != nil || spec.Backend != "bubblewrap" {
		t.Fatalf("compile approved = %+v, %v; want the evaluated policy", spec, err)
	}

	writePolicyFile(t, policyPath, "{ backend = \"none\"; }")
	calls := ev.calls
	if _, err := c.compile(ctx, policyPath); !errors.Is(err, ErrPolicyNotApproved) {
		t.Errorf("compile after edit: err = %v, want ErrPolicyNotApproved", err)
	}
	if ev.calls != calls {
		t.Error("an edited, unapproved policy was evaluated")
	}
}

// TestCompiler_EvaluatesPrivateSnapshot pins that evaluation reads a private
// copy of the snapshot, not the repository files, and removes it afterwards.
func TestCompiler_EvaluatesPrivateSnapshot(t *testing.T) {
	t.Parallel()

	policyPath := newPolicyDir(t)
	var seen string
	var seenFiles []string
	c := compiler{
		approved: approveAll,
		eval: func(_ context.Context, p string) ([]byte, error) {
			seen = p
			entries, err := os.ReadDir(filepath.Dir(p))
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				seenFiles = append(seenFiles, e.Name())
			}
			return []byte(`{}`), nil
		},
	}
	if _, err := c.compile(context.Background(), policyPath); err != nil {
		t.Fatalf("compile: %v", err)
	}
	if filepath.Dir(seen) == filepath.Dir(policyPath) {
		t.Errorf("evaluated the repository file %s, want a private snapshot copy", seen)
	}
	if want := []string{"policy.nix", "shared.nix"}; !slices.Equal(seenFiles, want) {
		t.Errorf("snapshot files = %v, want %v", seenFiles, want)
	}
	if _, err := os.Stat(filepath.Dir(seen)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("snapshot directory %s was not removed: %v", filepath.Dir(seen), err)
	}
}

func TestNixEvalArgs_Restricted(t *testing.T) {
	t.Parallel()

	args := strings.Join(nixEvalArgs("/tmp/snap/policy.nix"), " ")
	for _, want := range []string{
		"--option restrict-eval true",
		"--option allowed-uris  ",
		"--option nix-path  ",
		"--option allow-import-from-derivation false",
		"--option allow-unsafe-native-code-during-evaluation false",
		"-I policy=/tmp/snap",
		"--file /tmp/snap/policy.nix",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("nix eval args %q missing %q", args, want)
		}
	}
}

// TestNixEvalEnv_DropsNixPath uses t.Setenv, so it is not parallel.
func TestNixEvalEnv_DropsNixPath(t *testing.T) {
	t.Setenv("NIX_PATH", "secrets=/home/user")
	for _, kv := range nixEvalEnv() {
		if strings.HasPrefix(kv, "NIX_PATH=") {
			t.Errorf("nix eval environment keeps %q", kv)
		}
	}
}

// TestNixEval_RestrictedEvaluation runs the real `nix eval` when it is
// installed and pins that a policy can import its snapshot siblings but cannot
// read other files, the environment or URLs.
func TestNixEval_RestrictedEvaluation(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix is not installed")
	}

	secret := filepath.Join(t.TempDir(), "secret")
	writePolicyFile(t, secret, "top secret")

	tests := []struct {
		name    string
		policy  string
		want    string
		wantErr bool
	}{
		{name: "sibling import", policy: `{ backend = (import ./shared.nix).backend; }`, want: "auto"},
		{name: "environment is hidden", policy: `{ backend = builtins.getEnv "HOME"; }`, want: ""},
		{name: "file outside the snapshot", policy: `{ backend = builtins.readFile ` + secret + `; }`, wantErr: true},
		{name: "network fetch", policy: `{ backend = builtins.readFile (builtins.fetchurl "https://example.invalid/x"); }`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policyPath := newPolicyDir(t)
			writePolicyFile(t, policyPath, tt.policy)
			c := compiler{eval: nixEval, approved: approveAll}

			spec, err := c.compile(context.Background(), policyPath)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "forbidden in restricted mode") {
					t.Errorf("compile = %+v, %v; want a restricted-evaluation error", spec, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if spec.Backend != tt.want {
				t.Errorf("Backend = %q, want %q", spec.Backend, tt.want)
			}
		})
	}
}

// TestNixEval_UnsafeNativeCodeStaysOff runs the real `nix eval` when it is
// installed and pins that builtins.exec stays unavailable even when the
// inherited Nix configuration enables it. It uses t.Setenv, so it is not
// parallel.
func TestNixEval_UnsafeNativeCodeStaysOff(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix is not installed")
	}
	t.Setenv("NIX_CONFIG", "allow-unsafe-native-code-during-evaluation = true")

	policyPath := newPolicyDir(t)
	writePolicyFile(t, policyPath, `{ backend = builtins.exec [ "sh" "-c" "echo '\"pwned\"'" ]; }`)
	c := compiler{eval: nixEval, approved: approveAll}

	spec, err := c.compile(context.Background(), policyPath)
	if err == nil {
		t.Fatalf("compile = %+v; want builtins.exec to be undefined", spec)
	}
	if !strings.Contains(err.Error(), "undefined variable 'exec'") && !strings.Contains(err.Error(), "attribute 'exec' missing") {
		t.Errorf("compile error = %v; want builtins.exec to be undefined", err)
	}
}

// TestPolicyCacheDir_BesideApprovals pins the evaluation cache inside the
// self-protected ~/.<app>/ directory, next to the approvals it stands in for.
func TestPolicyCacheDir_BesideApprovals(t *testing.T) {
	t.Parallel()

	store := NewApprovalStore(filepath.Join("/home/u", ".qsdev", "sandbox-policy-approvals.json"))
	want := filepath.Join("/home/u", ".qsdev", "cache", "sandbox-policy")
	if got := policyCacheDir(store); got != want {
		t.Errorf("policyCacheDir = %q, want %q", got, want)
	}
}

func TestRejectedMounts(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.Filesystem.AllowRead = []string{testProjectDir + "/ok", "/opt/outside"}
	spec.HookOverrides = map[string]HookOverride{
		"evil": {ExtraMounts: []MountDecl{{Source: "/run/user/1000", Target: "/run/user/1000"}}},
		"good": {ExtraMounts: []MountDecl{{Source: "/nix/store", Target: "/nix/store", ReadOnly: true}}},
	}

	got := RejectedMounts(spec, testProjectDir)
	if len(got) != 2 {
		t.Fatalf("RejectedMounts = %q, want 2 entries", got)
	}
	if !strings.HasPrefix(got[0], "filesystem.allowRead: ") || !strings.HasPrefix(got[1], "hookOverrides.evil: ") {
		t.Errorf("RejectedMounts = %q, want the allowRead and hookOverrides.evil entries", got)
	}
}
