package helm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules/helm"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*helm.Module)(nil)

func TestName(t *testing.T) {
	m := &helm.Module{}
	if got := m.Name(); got != "helm" {
		t.Errorf("Name() = %q, want %q", got, "helm")
	}
}

func TestDisplayName(t *testing.T) {
	m := &helm.Module{}
	got := m.DisplayName()
	if !strings.Contains(got, "Helm") {
		t.Errorf("DisplayName() = %q, want it to contain %q", got, "Helm")
	}
}

func TestTier(t *testing.T) {
	m := &helm.Module{}
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
}

func TestDetect_ChartYamlPresent(t *testing.T) {
	dir := t.TempDir()
	chartYaml := "apiVersion: v2\nname: my-chart\nversion: 1.2.3\n"
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(chartYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &helm.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when Chart.yaml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	found := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "Chart.yaml") {
			found = true
		}
	}
	if !found {
		t.Error("evidence should mention Chart.yaml")
	}
}

func TestDetect_ChartYamlVersionExtracted(t *testing.T) {
	dir := t.TempDir()
	chartYaml := "apiVersion: v2\nname: my-chart\nversion: 1.2.3\n"
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(chartYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &helm.Module{}
	result := m.Detect(dir)

	if result.SuggestedConfig.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, "1.2.3")
	}

	foundVersion := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "1.2.3") {
			foundVersion = true
		}
	}
	if !foundVersion {
		t.Error("evidence should mention the detected version")
	}
}

func TestDetect_ChartLockProbable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Chart.lock"), []byte("dependencies: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &helm.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when Chart.lock is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &helm.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no Helm indicators present")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &helm.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment == "" {
		t.Fatal("DevenvNixFragment() returned empty string")
	}

	requiredStrings := []string{
		"kubernetes-helm",
		"kubeconform",
		"packages",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
}

func TestDenyRules(t *testing.T) {
	m := &helm.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() returned %d rules, want 2", len(rules))
	}

	expected := []string{
		"Bash(helm install *)",
		"Bash(helm upgrade *)",
	}
	for i, rule := range rules {
		if rule != expected[i] {
			t.Errorf("rules[%d] = %q, want %q", i, rule, expected[i])
		}
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &helm.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}
	if hooks[0].ID != "helmlint" {
		t.Errorf("hooks[0].ID = %q, want %q", hooks[0].ID, "helmlint")
	}
}

func TestSecurityConfigs(t *testing.T) {
	m := &helm.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if configs != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", configs)
	}
}

func TestCICommands(t *testing.T) {
	m := &helm.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 3 {
		t.Fatalf("CICommands() returned %d commands, want 3", len(cmds))
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("helm")
	if !ok {
		t.Fatal("expected module 'helm' to be registered in DefaultRegistry")
	}
	if mod.Name() != "helm" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "helm")
	}
}
