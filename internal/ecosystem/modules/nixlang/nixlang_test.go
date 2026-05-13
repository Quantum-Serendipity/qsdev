package nixlang_test

import (
	"os"
	"path/filepath"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/nixlang"
)

// newModule returns a fresh Module for testing.
func newModule() *nixlang.Module {
	return &nixlang.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*nixlang.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "nix" {
		t.Errorf("Name() = %q, want %q", got, "nix")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "Nix" {
		t.Errorf("DisplayName() = %q, want %q", got, "Nix")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 3 {
		t.Errorf("Tier() = %d, want %d", got, 3)
	}
}

// --- Detection tests ---

func TestDetect_FlakeNix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(""), 0o644); err != nil {
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
	if !containsSubstr(r.Evidence, "flake.nix") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "flake.nix")
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

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_NonEmpty(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
}

// --- helpers ---

func containsSubstr(ss []string, substr string) bool {
	for _, s := range ss {
		if len(s) >= len(substr) && searchString(s, substr) {
			return true
		}
	}
	return false
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
