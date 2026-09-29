package bazel_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/bazel"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// newModule returns a fresh Module for testing.
func newModule() *bazel.Module {
	return &bazel.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*bazel.Module)(nil)
	var _ ecosystem.PackageProvider = (*bazel.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "bazel", "Bazel", 3)
}

// --- Detection tests ---

func TestDetect_ModuleBazel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MODULE.bazel"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !containsSubstr(r.Evidence, "MODULE.bazel") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "MODULE.bazel")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages(t *testing.T) {
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"buildifier"}
	if len(pkgs) != len(expected) {
		t.Fatalf("DevenvPackages() returned %d packages, want %d", len(pkgs), len(expected))
	}
	for i, pkg := range pkgs {
		if pkg != expected[i] {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

// TestBazelVersion_FollowsBazelversion verifies .bazelversion selects the
// matching nixpkgs bazel_<major> instead of a hard-coded Bazel 7.
func TestBazelVersion_FollowsBazelversion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bazelversion string // .bazelversion content; "" = absent
		wantVersion  string
		wantExpr     string
	}{
		{"bazel 8 pin", "8.2.1\n", "8.2.1", "pkgs.bazel_8 or null"},
		{"bazel 7 with comment", "# pinned\n7.6.0\n", "7.6.0", "pkgs.bazel_7 or null"},
		{"release candidate", "9.0.0rc1\n", "9.0.0rc1", "pkgs.bazel_9 or null"},
		{"fork is not a version", "mycorp/7.0.0\n", "", "pkgs.bazel"},
		{"absent", "", "", "pkgs.bazel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "MODULE.bazel"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.bazelversion != "" {
				if err := os.WriteFile(filepath.Join(dir, ".bazelversion"), []byte(tt.bazelversion), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			m := newModule()
			cfg := m.Detect(dir).SuggestedConfig
			if cfg.Version != tt.wantVersion {
				t.Fatalf("Version = %q, want %q", cfg.Version, tt.wantVersion)
			}
			exprs := m.DevenvPackageExprs(cfg)
			if len(exprs) != 1 || !strings.Contains(exprs[0], tt.wantExpr) {
				t.Errorf("DevenvPackageExprs = %v, want one containing %q", exprs, tt.wantExpr)
			}
		})
	}
}

// TestBuildifierHook_CoversBzlmodManifest verifies the buildifier Files
// pattern selects MODULE.bazel and *.MODULE.bazel includes, not only
// BUILD/WORKSPACE/.bzl files.
func TestBuildifierHook_CoversBzlmodManifest(t *testing.T) {
	t.Parallel()
	hooks := newModule().PreCommitHooks(ecosystem.ModuleConfig{})
	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks returned %d hooks, want 1", len(hooks))
	}
	re := regexp.MustCompile(hooks[0].Files)
	for _, path := range []string{"MODULE.bazel", "third_party/deps.MODULE.bazel", "BUILD", "pkg/BUILD.bazel", "WORKSPACE", "defs.bzl"} {
		if !re.MatchString(path) {
			t.Errorf("Files %q does not match %s", hooks[0].Files, path)
		}
	}
	for _, path := range []string{"README.md", "MODULE.bazel.lock", "src/BUILDING.md"} {
		if re.MatchString(path) {
			t.Errorf("Files %q unexpectedly matches %s", hooks[0].Files, path)
		}
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_Empty(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag != "" {
		t.Errorf("DevenvNixFragment() = %q, want empty string (packages moved to DevenvPackages)", frag)
	}
}

// --- helpers ---

func containsSubstr(ss []string, substr string) bool {
	for _, s := range ss {
		if len(s) >= len(substr) && contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- SecurityConfigs tests ---

// TestSecurityConfigs_PreservesUserBazelrc verifies the hardening lives in a
// qsdev-owned .bazelrc.qsdev and that the user's .bazelrc (a detection
// marker, so it exists in most Bazel repos) is only created when absent.
func TestSecurityConfigs_PreservesUserBazelrc(t *testing.T) {
	t.Parallel()
	files := newModule().SecurityConfigs(ecosystem.ModuleConfig{})

	byPath := make(map[string]types.GeneratedFile, len(files))
	for _, f := range files {
		byPath[f.Path] = f
	}

	owned, ok := byPath[".bazelrc.qsdev"]
	if !ok {
		t.Fatal("missing .bazelrc.qsdev")
	}
	// lockfile_mode=update is Bazel's default and hardens nothing; the flag
	// must apply to every command (common), not only build.
	for _, flag := range []string{"\ncommon --lockfile_mode=error\n", "--spawn_strategy=sandboxed", "--sandbox_default_allow_network=false"} {
		if !strings.Contains(string(owned.Content), flag) {
			t.Errorf(".bazelrc.qsdev missing %q", flag)
		}
	}

	user, ok := byPath[".bazelrc"]
	if !ok {
		t.Fatal("missing .bazelrc")
	}
	if user.Strategy != types.Skip {
		t.Errorf(".bazelrc Strategy = %v, want Skip (never replace a user .bazelrc)", user.Strategy)
	}
	if !strings.Contains(string(user.Content), "try-import %workspace%/.bazelrc.qsdev\n") {
		t.Errorf(".bazelrc does not import .bazelrc.qsdev:\n%s", user.Content)
	}
}
