package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
