package scala_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/scala"
)

// newModule returns a fresh Module for testing.
func newModule() *scala.Module {
	return &scala.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*scala.Module)(nil)
	var _ ecosystem.PackageProvider = (*scala.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "scala", "Scala", 2)
}

// --- Detection tests ---

func TestDetect_BuildSbt(t *testing.T) {
	dir := t.TempDir()
	content := `scalaVersion := "3.3.1"

name := "myproject"
`
	if err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if len(r.Evidence) == 0 {
		t.Error("expected non-empty Evidence")
	}

	foundBuildSbt := false
	for _, e := range r.Evidence {
		if strings.Contains(e, "build.sbt") {
			foundBuildSbt = true
		}
	}
	if !foundBuildSbt {
		t.Error("Evidence should mention build.sbt")
	}

	if r.SuggestedConfig.Version != "3.3.1" {
		t.Errorf("SuggestedConfig.Version = %q, want %q", r.SuggestedConfig.Version, "3.3.1")
	}
	if bt := r.SuggestedConfig.Extras["build_tool"]; bt != "sbt" {
		t.Errorf("build_tool = %q, want %q", bt, "sbt")
	}
}

func TestDetect_BuildSc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.sc"), []byte("import mill._\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if bt := r.SuggestedConfig.Extras["build_tool"]; bt != "mill" {
		t.Errorf("build_tool = %q, want %q", bt, "mill")
	}
}

func TestDetect_WithBuildProperties(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte("name := \"myproject\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project", "build.properties"), []byte("sbt.version=1.9.7\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if sv := r.SuggestedConfig.Extras["sbt_version"]; sv != "1.9.7" {
		t.Errorf("sbt_version = %q, want %q", sv, "1.9.7")
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
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"build_tool":  "sbt",
			"jdk_version": "21",
		},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
	if !strings.Contains(frag, "languages.scala") {
		t.Errorf("fragment missing languages.scala:\n%s", frag)
	}
	if !strings.Contains(frag, "enable = true") {
		t.Errorf("fragment missing enable = true:\n%s", frag)
	}
	if !strings.Contains(frag, "languages.java") {
		t.Errorf("fragment missing languages.java:\n%s", frag)
	}
}

func TestDevenvNixFragment_Mill(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"build_tool":  "mill",
			"jdk_version": "17",
		},
	}

	frag, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	// Mill packages are now provided via DevenvPackages, not the fragment.
	if strings.Contains(frag, "packages") {
		t.Errorf("Mill fragment should not contain packages block:\n%s", frag)
	}
	if !strings.Contains(frag, "jdk17") {
		t.Errorf("fragment missing jdk17:\n%s", frag)
	}
	if !strings.Contains(frag, "languages.scala") {
		t.Errorf("fragment missing languages.scala:\n%s", frag)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages_Sbt(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "sbt"},
	})
	if pkgs != nil {
		t.Errorf("DevenvPackages(sbt) = %v, want nil", pkgs)
	}
}

func TestDevenvPackages_Mill(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{
		Extras: map[string]string{"build_tool": "mill"},
	})
	if len(pkgs) != 1 || pkgs[0] != "mill" {
		t.Errorf("DevenvPackages(mill) = %v, want [mill]", pkgs)
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

// --- SecurityConfigs tests ---

func TestSecurityConfigs(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(files) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(files))
	}
	if files[0].Path != ".qsdev/sbt-security-plugins.sbt" {
		t.Errorf("Path = %q, want %q", files[0].Path, ".qsdev/sbt-security-plugins.sbt")
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}
	if hooks[0].ID != "scalafmt" {
		t.Errorf("hook ID = %q, want %q", hooks[0].ID, "scalafmt")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("PackageManagers() returned %d entries, want 1", len(pms))
	}
	if pms[0].Name != "sbt" {
		t.Errorf("Name = %q, want %q", pms[0].Name, "sbt")
	}
	if pms[0].LockFile != "build.sbt.lock" {
		t.Errorf("LockFile = %q, want %q", pms[0].LockFile, "build.sbt.lock")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("WizardFields() returned %d fields, want 2", len(fields))
	}

	keys := make(map[string]bool)
	for _, f := range fields {
		keys[f.Key] = true
	}
	if !keys["scala_build_tool"] {
		t.Error("missing wizard field scala_build_tool")
	}
	if !keys["scala_jdk_version"] {
		t.Error("missing wizard field scala_jdk_version")
	}
}
