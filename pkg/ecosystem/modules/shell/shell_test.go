package shell_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/shell"
)

// newModule returns a fresh Module for testing.
func newModule() *shell.Module {
	return &shell.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*shell.Module)(nil)
	var _ ecosystem.PackageProvider = (*shell.Module)(nil)
}

// --- Basic metadata ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "shell" {
		t.Errorf("Name() = %q, want %q", got, "shell")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	got := m.DisplayName()
	if !strings.Contains(got, "Shell") && !strings.Contains(got, "Bash") {
		t.Errorf("DisplayName() = %q, want it to contain %q or %q", got, "Shell", "Bash")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
}

// --- Detection tests ---

func TestDetect_ShFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash\necho hello\n"), 0o755); err != nil {
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

	foundSh := false
	for _, e := range r.Evidence {
		if strings.Contains(e, ".sh") {
			foundSh = true
		}
	}
	if !foundSh {
		t.Error("Evidence should mention .sh files")
	}
}

func TestDetect_ScriptsDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true for scripts/ directory")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}

	foundScripts := false
	for _, e := range r.Evidence {
		if strings.Contains(e, "scripts") {
			foundScripts = true
		}
	}
	if !foundScripts {
		t.Error("Evidence should mention scripts/ directory")
	}
}

func TestDetect_Envrc(t *testing.T) {
	dir := t.TempDir()
	// .envrc alone is evidence but not a detection trigger per the code;
	// it only adds evidence if something else is also detected.
	// Create a .sh file to trigger detection, then verify .envrc evidence.
	if err := os.WriteFile(filepath.Join(dir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".envrc"), []byte("use nix\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}

	foundEnvrc := false
	for _, e := range r.Evidence {
		if strings.Contains(e, ".envrc") {
			foundEnvrc = true
		}
	}
	if !foundEnvrc {
		t.Error("Evidence should mention .envrc")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages(t *testing.T) {
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"shellcheck", "shfmt"}
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

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}

	ids := make(map[string]bool)
	for _, h := range hooks {
		ids[h.ID] = true
	}
	if !ids["shellcheck"] {
		t.Error("missing hook with ID shellcheck")
	}
	if !ids["shfmt"] {
		t.Error("missing hook with ID shfmt")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 4 {
		t.Fatalf("DenyRules() returned %d rules, want 4", len(rules))
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

	if pms != nil {
		t.Errorf("PackageManagers() = %v, want nil (shell has no package manager)", pms)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if files != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", files)
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if fields != nil {
		t.Errorf("WizardFields() = %v, want nil", fields)
	}
}
