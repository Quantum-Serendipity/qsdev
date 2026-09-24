package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeEval stands in for `nix eval`, returning a fixed JSON document and
// counting calls.
type fakeEval struct {
	out   string
	err   error
	calls int
}

func (f *fakeEval) eval(context.Context, string) ([]byte, error) {
	f.calls++
	return []byte(f.out), f.err
}

// approveAll stands in for the approval store, approving every snapshot.
func approveAll(*Snapshot) error { return nil }

func writePolicyFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestCompiler_CachesEvaluation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.nix")
	writePolicyFile(t, policyPath, "{ backend = \"bubblewrap\"; }")
	ev := &fakeEval{out: `{"backend":"bubblewrap"}`}
	c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: approveAll}

	for i := range 3 {
		spec, err := c.compile(context.Background(), policyPath)
		if err != nil {
			t.Fatalf("compile #%d: %v", i, err)
		}
		if spec.Backend != "bubblewrap" {
			t.Fatalf("compile #%d: Backend = %q, want bubblewrap", i, spec.Backend)
		}
	}
	if ev.calls != 1 {
		t.Errorf("nix eval ran %d times for an unchanged policy, want 1", ev.calls)
	}
}

func TestCompiler_CacheInvalidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(t *testing.T, dir, policyPath string)
	}{
		{
			name: "policy file edited",
			change: func(t *testing.T, _, policyPath string) {
				t.Helper()
				writePolicyFile(t, policyPath, "{ backend = \"systemd-run\"; }")
			},
		},
		{
			name: "sibling nix file edited",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				writePolicyFile(t, filepath.Join(dir, "shared.nix"), "{ extra = 2; }")
			},
		},
		{
			name: "sibling nix file added",
			change: func(t *testing.T, dir, _ string) {
				t.Helper()
				writePolicyFile(t, filepath.Join(dir, "new.nix"), "{ }")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			policyPath := filepath.Join(dir, "policy.nix")
			writePolicyFile(t, policyPath, "import ./shared.nix")
			writePolicyFile(t, filepath.Join(dir, "shared.nix"), "{ extra = 1; }")
			ev := &fakeEval{out: `{"backend":"auto"}`}
			c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: approveAll}

			if _, err := c.compile(context.Background(), policyPath); err != nil {
				t.Fatalf("first compile: %v", err)
			}
			tt.change(t, dir, policyPath)
			if _, err := c.compile(context.Background(), policyPath); err != nil {
				t.Fatalf("second compile: %v", err)
			}

			if ev.calls != 2 {
				t.Errorf("nix eval ran %d times, want 2 (the change must invalidate the cache)", ev.calls)
			}
		})
	}
}

// TestCompiler_EvalFailureIsAnError pins that a policy that exists but cannot
// be evaluated or parsed yields an error (never the defaults) and is not
// cached, so fixing the policy takes effect on the next run.
func TestCompiler_EvalFailureIsAnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ev   fakeEval
	}{
		{name: "nix eval fails", ev: fakeEval{err: errors.New("nix: command not found")}},
		{name: "output is not a policy", ev: fakeEval{out: "not json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			policyPath := filepath.Join(dir, "policy.nix")
			writePolicyFile(t, policyPath, "{ }")
			cacheDir := t.TempDir()
			ev := tt.ev
			c := compiler{cacheDir: cacheDir, eval: ev.eval, approved: approveAll}

			spec, err := c.compile(context.Background(), policyPath)
			if err == nil {
				t.Fatalf("compile = %+v, want an error", spec)
			}
			entries, readErr := os.ReadDir(cacheDir)
			if readErr != nil {
				t.Fatalf("reading cache dir: %v", readErr)
			}
			if len(entries) != 0 {
				t.Errorf("failed evaluation was cached: %v", entries)
			}
		})
	}
}

func TestCompiler_CorruptCacheEntryReevaluates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.nix")
	writePolicyFile(t, policyPath, "{ }")
	ev := &fakeEval{out: `{"backend":"bubblewrap"}`}
	c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: approveAll}

	snap, err := ReadSnapshot(policyPath)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	cachePath := c.cachePath(snap)
	if cachePath == "" {
		t.Fatal("cachePath is empty")
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		t.Fatalf("creating cache dir: %v", err)
	}
	writePolicyFile(t, cachePath, "{truncated")

	spec, err := c.compile(context.Background(), policyPath)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if spec.Backend != "bubblewrap" || ev.calls != 1 {
		t.Errorf("Backend = %q after %d evals, want bubblewrap after 1", spec.Backend, ev.calls)
	}
}

func TestCompiler_MissingPolicyUsesDefaults(t *testing.T) {
	t.Parallel()

	ev := &fakeEval{}
	c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: approveAll}

	spec, err := c.compile(context.Background(), filepath.Join(t.TempDir(), "policy.nix"))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if spec.Backend != DefaultPolicy().Backend || ev.calls != 0 {
		t.Errorf("Backend = %q after %d evals, want the default policy without evaluating", spec.Backend, ev.calls)
	}
}
