package dart_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/dart"
)

// newModule returns a fresh Module for testing.
func newModule() *dart.Module {
	return &dart.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*dart.Module)(nil)
	var _ ecosystem.PackageProvider = (*dart.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "dart", "Dart/Flutter", 3)
}

// --- Detection tests ---

func TestDetect_PubspecYaml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte("name: test_app\nversion: 1.0.0\n"), 0o644); err != nil {
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
	if !containsSubstr(r.Evidence, "pubspec.yaml") {
		t.Errorf("Evidence = %v, want entry containing %q", r.Evidence, "pubspec.yaml")
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

func TestDevenvPackages_NoFlutter(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{
		Extras: map[string]string{"flutter": "false"},
	})
	if pkgs != nil {
		t.Errorf("DevenvPackages(no flutter) = %v, want nil", pkgs)
	}
}

func TestDevenvPackages_WithFlutter(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{
		Extras: map[string]string{"flutter": "true"},
	})
	if len(pkgs) != 1 || pkgs[0] != "flutter" {
		t.Errorf("DevenvPackages(flutter) = %v, want [flutter]", pkgs)
	}
}

func TestDevenvPackages_Default(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})
	if pkgs != nil {
		t.Errorf("DevenvPackages(default) = %v, want nil", pkgs)
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
	if !containsSubstr([]string{frag}, "languages.dart.enable = true") {
		t.Errorf("fragment missing languages.dart.enable = true:\n%s", frag)
	}
	// Fragment should not contain packages — those are in DevenvPackages.
	if containsSubstr([]string{frag}, "packages") {
		t.Errorf("fragment should not contain packages block:\n%s", frag)
	}
}

func TestDevenvNixFragment_FlutterNoPackages(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		Extras: map[string]string{"flutter": "true"},
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	// Even with flutter=true, packages should not be in the fragment.
	if containsSubstr([]string{frag}, "packages") {
		t.Errorf("flutter fragment should not contain packages block:\n%s", frag)
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
