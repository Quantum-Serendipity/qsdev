package ansible_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules/ansible"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*ansible.Module)(nil)

func TestName(t *testing.T) {
	m := &ansible.Module{}
	if got := m.Name(); got != "ansible" {
		t.Errorf("Name() = %q, want %q", got, "ansible")
	}
}

func TestDisplayName(t *testing.T) {
	m := &ansible.Module{}
	if got := m.DisplayName(); got != "Ansible" {
		t.Errorf("DisplayName() = %q, want %q", got, "Ansible")
	}
}

func TestTier(t *testing.T) {
	m := &ansible.Module{}
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
}

func TestDetect_AnsibleCfgPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ansible.cfg"), []byte("[defaults]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when ansible.cfg is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	found := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "ansible.cfg") {
			found = true
		}
	}
	if !found {
		t.Error("evidence should mention ansible.cfg")
	}
}

func TestDetect_GalaxyYmlPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "galaxy.yml"), []byte("namespace: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when galaxy.yml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
}

func TestDetect_PlaybooksDirProbable(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "playbooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when playbooks/ is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &ansible.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no Ansible indicators present")
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &ansible.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment == "" {
		t.Fatal("DevenvNixFragment() returned empty string")
	}

	requiredStrings := []string{
		"ansible",
		"ansible-lint",
		"packages",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
}

func TestDenyRules(t *testing.T) {
	m := &ansible.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}

	expected := []string{
		"Bash(ansible-galaxy install *)",
		"Bash(ansible-galaxy collection install *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
		}
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &ansible.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}

	if hooks[0].ID != "ansible-lint" {
		t.Errorf("hooks[0].ID = %q, want %q", hooks[0].ID, "ansible-lint")
	}
}

func TestSecurityConfigs(t *testing.T) {
	m := &ansible.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d files, want 1", len(configs))
	}
	if configs[0].Path != ".ansible-security.cfg" {
		t.Errorf("SecurityConfigs()[0].Path = %q, want %q", configs[0].Path, ".ansible-security.cfg")
	}
}

func TestCICommands(t *testing.T) {
	m := &ansible.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("ansible")
	if !ok {
		t.Fatal("expected module 'ansible' to be registered in DefaultRegistry")
	}
	if mod.Name() != "ansible" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "ansible")
	}
}
