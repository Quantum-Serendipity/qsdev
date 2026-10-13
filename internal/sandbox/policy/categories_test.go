package policy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

const validCategories = "(valid: linter, formatter, network-linter, generator, test-runner)"

// TestValidateCategories_RejectsUnknownOverrideAndCategoryKey pins that a
// typo in a category name is an error naming the field, instead of being read
// as "linter" (overrides) or silently ignored (hookCategories keys).
func TestValidateCategories_RejectsUnknownOverrideAndCategoryKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*PolicySpec)
		wantErr string
	}{
		{
			name:   "known override category",
			mutate: func(s *PolicySpec) { s.HookOverrides = map[string]HookOverride{"fmt": {Category: "formatter"}} },
		},
		{
			name:   "override without a category",
			mutate: func(s *PolicySpec) { s.HookOverrides = map[string]HookOverride{"fmt": {NetworkOverride: "deny"}} },
		},
		{
			name:    "unknown override category",
			mutate:  func(s *PolicySpec) { s.HookOverrides = map[string]HookOverride{"fmt": {Category: "test-runer"}} },
			wantErr: `hookOverrides.fmt.category: unknown category "test-runer" ` + validCategories,
		},
		{
			name:    "unknown hookCategories key",
			mutate:  func(s *PolicySpec) { s.HookCategories["formater"] = CategoryPolicy{WorktreeAccess: "ro"} },
			wantErr: `hookCategories: unknown category "formater" ` + validCategories,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := DefaultPolicy()
			tt.mutate(spec)
			err := validateCategories(spec)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateCategories: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("validateCategories error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultPolicy_CategoriesValid pins that the built-in defaults, which
// compile does not validate, name only known categories.
func TestDefaultPolicy_CategoriesValid(t *testing.T) {
	t.Parallel()
	if err := validateCategories(DefaultPolicy()); err != nil {
		t.Fatalf("DefaultPolicy has an invalid category: %v", err)
	}
}

// TestCompilePolicy_UnknownCategoryFails pins that a policy naming an unknown
// category does not compile, whether it is freshly evaluated or read back from
// the evaluation cache, so `sandbox exec` fails closed instead of running the
// hook under a misread category.
func TestCompilePolicy_UnknownCategoryFails(t *testing.T) {
	t.Parallel()

	const badPolicy = `{"hookOverrides":{"fmt":{"category":"test-runer"}}}`
	tests := []struct {
		name string
		// seedCache writes badPolicy as the cache entry for the snapshot and
		// makes the evaluator return a valid policy, so only the cached copy
		// is invalid.
		seedCache bool
	}{
		{name: "evaluated spec"},
		{name: "cached spec", seedCache: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policyPath := filepath.Join(t.TempDir(), "policy.nix")
			writePolicyFile(t, policyPath, "{ }")
			ev := &fakeEval{out: badPolicy}
			c := compiler{cacheDir: t.TempDir(), eval: ev.eval, approved: approveAll}

			if tt.seedCache {
				ev.out = `{"backend":"auto"}`
				snap, err := ReadSnapshot(policyPath)
				if err != nil {
					t.Fatal(err)
				}
				cachePath := c.cachePath(snap)
				if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
					t.Fatal(err)
				}
				writePolicyFile(t, cachePath, badPolicy)
			}

			spec, err := c.compile(context.Background(), policyPath)
			if err == nil {
				t.Fatalf("compile = %+v, want an unknown-category error", spec)
			}
			if !strings.Contains(err.Error(), `unknown category "test-runer"`) {
				t.Errorf("compile error = %v, want one naming the unknown category", err)
			}
		})
	}
}

// TestToSandboxConfig_UnknownOverrideCategoryKeepsCallerCategory pins the
// fallback for a spec that bypassed validation: an unknown override category
// leaves the caller's category in force and never widens access.
func TestToSandboxConfig_UnknownOverrideCategoryKeepsCallerCategory(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookOverrides = map[string]HookOverride{"hook": {Category: "test-runer"}}

	cfg := ToSandboxConfig(spec, sandbox.CategoryLinter, "hook", testProjectDir)

	if cfg.HookCategory != sandbox.CategoryLinter {
		t.Errorf("HookCategory = %v, want the caller's linter", cfg.HookCategory)
	}
	if !cfg.WorktreeReadOnly() {
		t.Errorf("worktree is writable (WorktreeAccess=%q), want the linter's read-only", cfg.WorktreeAccess)
	}
	if cfg.Network.Mode != "deny" {
		t.Errorf("Network.Mode = %q, want the linter's deny", cfg.Network.Mode)
	}
}
