package perl_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/perl"
)

// newModule returns a fresh Module for testing.
func newModule() *perl.Module {
	return &perl.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*perl.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "perl" {
		t.Errorf("Name() = %q, want %q", got, "perl")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "Perl" {
		t.Errorf("DisplayName() = %q, want %q", got, "Perl")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 4 {
		t.Errorf("Tier() = %d, want %d", got, 4)
	}
}

// --- Detection tests ---

func TestDetect_Cpanfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cpanfile"), []byte("requires 'Mojolicious';\n"), 0o644); err != nil {
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
	if !containsStr(r.Evidence, "cpanfile found") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "cpanfile found")
	}
}

func TestDetect_MakefilePL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile.PL"), []byte("use ExtUtils::MakeMaker;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", r.Confidence)
	}
}

func TestDetect_CpanfileSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cpanfile.snapshot"), []byte("# carton snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.SuggestedConfig.PackageManager != "carton" {
		t.Errorf("PackageManager = %q, want %q", r.SuggestedConfig.PackageManager, "carton")
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

func TestDevenvNixFragment(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
	if !strings.Contains(frag, "perl") {
		t.Errorf("fragment missing perl reference:\n%s", frag)
	}
}

// --- helpers ---

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
