package powershell_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/powershell"
)

// newModule returns a fresh Module for testing.
func newModule() *powershell.Module {
	return &powershell.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*powershell.Module)(nil)
	var _ ecosystem.PackageProvider = (*powershell.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "powershell", "PowerShell", 4)
}

// --- Detection tests ---

func TestDetect_Ps1File(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.ps1"), []byte("Write-Host 'hello'\n"), 0o644); err != nil {
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
	if !containsStr(r.Evidence, "*.ps1 files found") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "*.ps1 files found")
	}
}

func TestDetect_RequirementsPsd1(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.psd1"), []byte("@{}\n"), 0o644); err != nil {
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
}

func TestDetect_Psm1File(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "module.psm1"), []byte("function Get-Thing {}\n"), 0o644); err != nil {
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

// --- DevenvPackages tests ---

func TestDevenvPackages(t *testing.T) {
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"powershell"}
	if len(pkgs) != len(expected) {
		t.Fatalf("DevenvPackages() returned %d packages, want %d", len(pkgs), len(expected))
	}
	for i, pkg := range pkgs {
		if pkg != expected[i] {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment(t *testing.T) {
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

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
