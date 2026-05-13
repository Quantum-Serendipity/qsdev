package rlang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rlang"
)

// newModule returns a fresh Module for testing.
func newModule() *rlang.Module {
	return &rlang.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*rlang.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "r" {
		t.Errorf("Name() = %q, want %q", got, "r")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "R" {
		t.Errorf("DisplayName() = %q, want %q", got, "R")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 4 {
		t.Errorf("Tier() = %d, want %d", got, 4)
	}
}

// --- Detection tests ---

func TestDetect_RenvLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "renv.lock"), []byte("{}\n"), 0o644); err != nil {
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
	if r.SuggestedConfig.PackageManager != "renv" {
		t.Errorf("PackageManager = %q, want %q", r.SuggestedConfig.PackageManager, "renv")
	}
}

func TestDetect_DescriptionAlone_Probable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DESCRIPTION"), []byte("Package: mypackage\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	// DESCRIPTION alone should give Probable, not Certain.
	if r.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable (DESCRIPTION alone is ambiguous)", r.Confidence)
	}
}

func TestDetect_DescriptionWithNamespace_Certain(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DESCRIPTION"), []byte("Package: mypackage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "NAMESPACE"), []byte("exportPattern(\"^[[:alpha:]]\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain (DESCRIPTION + NAMESPACE)", r.Confidence)
	}
}

func TestDetect_RFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "analysis.R"), []byte("library(tidyverse)\n"), 0o644); err != nil {
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
	if !strings.Contains(frag, "r") {
		t.Errorf("fragment missing r reference:\n%s", frag)
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
