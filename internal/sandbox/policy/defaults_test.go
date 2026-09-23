//go:build !windows

package policy

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestToSandboxConfig_WorktreeAccess pins that the policy's per-category
// worktreeAccess decides whether the worktree is read-only, instead of being
// parsed and ignored in favour of the hard-coded category default.
func TestToSandboxConfig_WorktreeAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		category     sandbox.HookCategory
		access       string
		wantReadOnly bool
	}{
		{name: "policy makes test-runner read-only", category: sandbox.CategoryTestRunner, access: "ro", wantReadOnly: true},
		{name: "policy makes formatter read-only", category: sandbox.CategoryFormatter, access: "ro", wantReadOnly: true},
		{name: "policy makes linter read-write", category: sandbox.CategoryLinter, access: "rw", wantReadOnly: false},
		{name: "empty keeps linter default", category: sandbox.CategoryLinter, access: "", wantReadOnly: true},
		{name: "empty keeps formatter default", category: sandbox.CategoryFormatter, access: "", wantReadOnly: false},
		{name: "invalid value fails closed", category: sandbox.CategoryGenerator, access: "read-write", wantReadOnly: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := DefaultPolicy()
			spec.HookCategories[tt.category.String()] = CategoryPolicy{WorktreeAccess: tt.access, Network: "deny"}

			cfg := ToSandboxConfig(spec, tt.category, "hook")

			if got := cfg.WorktreeReadOnly(); got != tt.wantReadOnly {
				t.Errorf("WorktreeReadOnly() = %v, want %v (WorktreeAccess=%q)", got, tt.wantReadOnly, cfg.WorktreeAccess)
			}
		})
	}
}

// TestToSandboxConfig_WorktreeAccessFollowsCategoryOverride pins that a hook
// reassigned to another category gets that category's worktree access.
func TestToSandboxConfig_WorktreeAccessFollowsCategoryOverride(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.HookOverrides = map[string]HookOverride{"fmt": {Category: "linter"}}

	cfg := ToSandboxConfig(spec, sandbox.CategoryFormatter, "fmt")

	if !cfg.WorktreeReadOnly() {
		t.Errorf("hook reassigned to linter must get a read-only worktree (WorktreeAccess=%q)", cfg.WorktreeAccess)
	}
}

func TestToSandboxConfig_AllowReadWrite(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.Filesystem.AllowRead = []string{"/opt/toolchain", "relative/ignored"}
	spec.Filesystem.AllowWrite = []string{"/var/cache/hook", "/etc/shadow"}

	cfg := ToSandboxConfig(spec, sandbox.CategoryLinter, "hook")

	want := []sandbox.MountSpec{
		{Source: "/opt/toolchain", Target: "/opt/toolchain", ReadOnly: true},
		{Source: "/var/cache/hook", Target: "/var/cache/hook", ReadOnly: false},
	}
	for _, m := range want {
		if !slices.Contains(cfg.Mounts, m) {
			t.Errorf("Mounts missing %+v", m)
		}
	}
	for _, m := range cfg.Mounts {
		if m.Source == "relative/ignored" {
			t.Errorf("relative allowRead path must be rejected, got mount %+v", m)
		}
		if m.Source == "/etc/shadow" && !m.ReadOnly {
			t.Errorf("allowWrite of a deny path must be rejected, got mount %+v", m)
		}
	}
}

func TestToSandboxConfig_Backend(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.Backend = "bubblewrap"

	if got := ToSandboxConfig(spec, sandbox.CategoryLinter, "hook").Backend; got != "bubblewrap" {
		t.Errorf("Backend = %q, want %q", got, "bubblewrap")
	}
}
