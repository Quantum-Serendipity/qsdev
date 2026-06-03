package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestDefaultPolicy_HasAllCategories(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()

	expected := []string{
		"linter",
		"formatter",
		"network-linter",
		"generator",
		"test-runner",
	}

	for _, cat := range expected {
		t.Run(cat, func(t *testing.T) {
			t.Parallel()

			if _, ok := spec.HookCategories[cat]; !ok {
				t.Errorf("category %q missing from default policy", cat)
			}
		})
	}

	if len(spec.HookCategories) != len(expected) {
		t.Errorf("expected %d categories, got %d", len(expected), len(spec.HookCategories))
	}
}

func TestDefaultPolicy_DenyPaths(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	expectedSuffixes := []string{
		".ssh",
		".gnupg",
		".aws",
		".azure",
		filepath.Join(".config", "gcloud"),
		".kube",
		filepath.Join(".docker", "config.json"),
		".netrc",
	}

	denied := make(map[string]bool, len(spec.Filesystem.Deny))
	for _, p := range spec.Filesystem.Deny {
		denied[p] = true
	}

	for _, suffix := range expectedSuffixes {
		t.Run(suffix, func(t *testing.T) {
			t.Parallel()

			full := filepath.Join(home, suffix)
			if !denied[full] {
				t.Errorf("expected %q in deny list", full)
			}
		})
	}
}

func TestToSandboxConfig_LinterCategory(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	cfg := ToSandboxConfig(spec, sandbox.CategoryLinter, "golangci-lint")

	if cfg.HookCategory != sandbox.CategoryLinter {
		t.Errorf("expected category %v, got %v", sandbox.CategoryLinter, cfg.HookCategory)
	}

	if cfg.Network.Mode != "deny" {
		t.Errorf("expected network mode %q, got %q", "deny", cfg.Network.Mode)
	}

	// Linter category has ro worktree — verify the hook category method agrees.
	if !cfg.HookCategory.WorktreeReadOnly() {
		t.Error("linter category should have read-only worktree")
	}
}

func TestToSandboxConfig_FormatterCategory(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	cfg := ToSandboxConfig(spec, sandbox.CategoryFormatter, "gofmt")

	if cfg.HookCategory != sandbox.CategoryFormatter {
		t.Errorf("expected category %v, got %v", sandbox.CategoryFormatter, cfg.HookCategory)
	}

	if cfg.Network.Mode != "deny" {
		t.Errorf("expected network mode %q, got %q", "deny", cfg.Network.Mode)
	}

	if cfg.HookCategory.WorktreeReadOnly() {
		t.Error("formatter category should have read-write worktree")
	}
}

func TestToSandboxConfig_HookOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		override        HookOverride
		baseCategory    sandbox.HookCategory
		wantNetwork     string
		wantCategory    sandbox.HookCategory
		wantExtraMounts int
	}{
		{
			name: "network override",
			override: HookOverride{
				NetworkOverride: "allow",
			},
			baseCategory:    sandbox.CategoryLinter,
			wantNetwork:     "allow",
			wantCategory:    sandbox.CategoryLinter,
			wantExtraMounts: 0,
		},
		{
			name: "category reassignment",
			override: HookOverride{
				Category: "formatter",
			},
			baseCategory:    sandbox.CategoryLinter,
			wantNetwork:     "deny",
			wantCategory:    sandbox.CategoryFormatter,
			wantExtraMounts: 0,
		},
		{
			name: "extra mounts",
			override: HookOverride{
				ExtraMounts: []MountDecl{
					{Source: "/opt/tools", Target: "/opt/tools", ReadOnly: true},
					{Source: "/tmp/cache", Target: "/tmp/cache", ReadOnly: false},
				},
			},
			baseCategory:    sandbox.CategoryLinter,
			wantNetwork:     "deny",
			wantCategory:    sandbox.CategoryLinter,
			wantExtraMounts: 2,
		},
		{
			name: "combined overrides",
			override: HookOverride{
				Category:        "test-runner",
				NetworkOverride: "allow",
				ExtraMounts: []MountDecl{
					{Source: "/data", Target: "/data", ReadOnly: true},
				},
			},
			baseCategory:    sandbox.CategoryLinter,
			wantNetwork:     "allow",
			wantCategory:    sandbox.CategoryTestRunner,
			wantExtraMounts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := DefaultPolicy()
			spec.HookOverrides = map[string]HookOverride{
				"custom-hook": tt.override,
			}

			cfg := ToSandboxConfig(spec, tt.baseCategory, "custom-hook")

			if cfg.HookCategory != tt.wantCategory {
				t.Errorf("expected category %v, got %v", tt.wantCategory, cfg.HookCategory)
			}

			if cfg.Network.Mode != tt.wantNetwork {
				t.Errorf("expected network mode %q, got %q", tt.wantNetwork, cfg.Network.Mode)
			}

			// Count extra mounts beyond the deny-list mounts.
			denyCount := len(spec.Filesystem.Deny)
			extraCount := len(cfg.Mounts) - denyCount
			if extraCount != tt.wantExtraMounts {
				t.Errorf("expected %d extra mounts, got %d", tt.wantExtraMounts, extraCount)
			}
		})
	}
}

func TestCompilePolicy_MissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent.nix")

	spec, err := CompilePolicy(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to defaults.
	if spec.Backend != "auto" {
		t.Errorf("expected backend %q, got %q", "auto", spec.Backend)
	}

	if len(spec.HookCategories) != 5 {
		t.Errorf("expected 5 categories, got %d", len(spec.HookCategories))
	}

	if spec.Resources.MemoryBytes != 2*1024*1024*1024 {
		t.Errorf("expected 2GB memory, got %d", spec.Resources.MemoryBytes)
	}
}
