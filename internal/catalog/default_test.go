package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// These tests exercise the global Default/SetProjectRoot/ResetDefault
// functions. They are NOT marked t.Parallel because they mutate shared
// package-level state (defaultOnce, defaultCat, defaultErr, projectRootDir).
// Each test calls ResetDefault in both setup and cleanup to ensure isolation.

func TestDefault_Concurrent(t *testing.T) {
	ResetDefault()
	t.Cleanup(func() { ResetDefault() })

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			// Half the goroutines set a project root, half call Default.
			if n%2 == 0 {
				SetProjectRoot(fmt.Sprintf("/tmp/fake-root-%d", n))
			}
			cat, err := Default()
			if err != nil {
				errs <- fmt.Errorf("goroutine %d: Default() error: %w", n, err)
				return
			}
			if cat == nil {
				errs <- fmt.Errorf("goroutine %d: Default() returned nil catalog", n)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestDefault_ErrorReturn(t *testing.T) {
	ResetDefault()
	t.Cleanup(func() { ResetDefault() })

	// Create a temp directory with a malformed .qsdev/defaults.yaml so that
	// Load fails when it tries to parse the project config file.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".qsdev")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	// Write YAML that is syntactically valid but produces a catalog with a
	// tier missing its required description and having order=0. The merged
	// catalog will fail validation.
	badYAML := []byte("tiers:\n  bad-tier:\n    order: 0\n")
	if err := os.WriteFile(filepath.Join(configDir, "defaults.yaml"), badYAML, 0o644); err != nil {
		t.Fatalf("writing bad config: %v", err)
	}

	SetProjectRoot(tmpDir)
	cat, err := Default()
	if err == nil {
		t.Fatal("Default() should return an error for invalid project config")
	}
	if cat != nil {
		t.Error("Default() should return nil catalog on error")
	}
	if !strings.Contains(err.Error(), "catalog validation") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "catalog validation")
	}
}

func TestMustDefault_Panics(t *testing.T) {
	ResetDefault()
	t.Cleanup(func() { ResetDefault() })

	// Set up a project root with malformed config to force Load to fail.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".qsdev")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}
	badYAML := []byte("tiers:\n  bad-tier:\n    order: 0\n")
	if err := os.WriteFile(filepath.Join(configDir, "defaults.yaml"), badYAML, 0o644); err != nil {
		t.Fatalf("writing bad config: %v", err)
	}

	SetProjectRoot(tmpDir)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustDefault() should panic when catalog loading fails")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value should be a string, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "catalog: failed to load") {
			t.Errorf("panic message = %q, want it to contain %q", msg, "catalog: failed to load")
		}
	}()

	MustDefault()
}

// Regression: an invalid user-level org overlay used to make Default fail,
// and MustDefault (called during package init) then panicked every qsdev
// command, including `defaults validate/reset` that exist to repair it.
func TestDefault_InvalidOrgOverlayFallsBack(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		// A partial entry that only restates a built-in field is a valid
		// overlay since entries are deep-merged; these stay invalid.
		{"partially uncommented tier renumbering a built-in", "tiers:\n  standard:\n    order: 9\n"},
		{"partially uncommented tier with misspelled key", "tiers:\n  standard:\n    ordr: 2\n"},
		{"yaml syntax error", "tiers: [unclosed\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDefault()
			t.Cleanup(ResetDefault)

			orgFile := filepath.Join(t.TempDir(), "defaults.yaml")
			if err := os.WriteFile(orgFile, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", orgFile)

			cat, err := Default()
			if err != nil {
				t.Fatalf("Default() error = %v, want fallback to built-in defaults", err)
			}
			if cat == nil {
				t.Fatal("Default() returned nil catalog")
			}
			if _, ok := cat.TierDef("standard"); !ok {
				t.Error("fallback catalog is missing the built-in standard tier")
			}
			overlayErr := OrgOverlayError()
			if overlayErr == nil {
				t.Fatal("OrgOverlayError() = nil, want the overlay's load error")
			}
			if !strings.Contains(overlayErr.Error(), orgFile) {
				t.Errorf("OrgOverlayError() = %q, want it to name %s", overlayErr, orgFile)
			}

			// The init-time accessor must not panic either.
			_ = MustDefault()
		})
	}
}

func TestDefault_ValidOrgOverlayHasNoOverlayError(t *testing.T) {
	ResetDefault()
	t.Cleanup(ResetDefault)

	orgFile := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(orgFile, []byte("# all commented out\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", orgFile)

	if _, err := Default(); err != nil {
		t.Fatalf("Default() error: %v", err)
	}
	if err := OrgOverlayError(); err != nil {
		t.Errorf("OrgOverlayError() = %v, want nil", err)
	}
}

// Regression: tests loaded the developer's ~/.config/qsdev/defaults.yaml, so
// local results depended on the host's org overlay.
func TestOrgConfigPath_IgnoresHomeOverlayInTests(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")

	overlay := filepath.Join(home, ".config", branding.Get().AppName, "defaults.yaml")
	if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlay, []byte("tiers: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := OrgConfigPath(); got != "" {
		t.Errorf("OrgConfigPath() = %q in a test binary, want \"\"", got)
	}
	if got := OrgConfigFile(); got != "" {
		t.Errorf("OrgConfigFile() = %q in a test binary, want \"\"", got)
	}
	if got := homeOrgConfigPath(); got != overlay {
		t.Errorf("homeOrgConfigPath() = %q, want %q", got, overlay)
	}

	// An explicit env override is still honoured.
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", overlay)
	if got := OrgConfigPath(); got != overlay {
		t.Errorf("OrgConfigPath() with env override = %q, want %q", got, overlay)
	}
}

func TestResetDefault(t *testing.T) {
	ResetDefault()
	t.Cleanup(func() { ResetDefault() })

	// First call should succeed and cache the catalog.
	cat1, err := Default()
	if err != nil {
		t.Fatalf("first Default() error: %v", err)
	}
	if cat1 == nil {
		t.Fatal("first Default() returned nil catalog")
	}

	// Second call should return the same cached instance.
	cat2, err := Default()
	if err != nil {
		t.Fatalf("second Default() error: %v", err)
	}
	if cat1 != cat2 {
		t.Error("second Default() should return same cached instance")
	}

	// Reset clears the cache.
	ResetDefault()

	// Third call should re-initialize, returning a fresh catalog.
	cat3, err := Default()
	if err != nil {
		t.Fatalf("third Default() after ResetDefault() error: %v", err)
	}
	if cat3 == nil {
		t.Fatal("third Default() returned nil catalog")
	}
	if cat3 == cat1 {
		t.Error("Default() after ResetDefault() should return a new catalog instance")
	}
}
