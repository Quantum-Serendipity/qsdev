package helm_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/helm"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*helm.Module)(nil)
var _ ecosystem.PackageProvider = (*helm.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &helm.Module{}, "helm", "Helm", 2)
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
	t.Parallel()
	tests := []struct {
		name      string
		chartYaml string
		want      string
	}{
		{
			name:      "plain",
			chartYaml: "apiVersion: v2\nname: my-chart\nversion: 1.2.3\n",
			want:      "1.2.3",
		},
		{
			// A dependency's indented version must not be mistaken for the
			// chart's own version, even when it appears first.
			name: "dependencies before version",
			chartYaml: "apiVersion: v2\nname: my-chart\ndependencies:\n" +
				"  - name: redis\n    version: 17.0.0\n" +
				"version: \"1.2.3\" # chart\n",
			want: "1.2.3",
		},
		{
			name:      "single quoted",
			chartYaml: "apiVersion: v2\nversion: '0.4.0'\n",
			want:      "0.4.0",
		},
		{
			name:      "no version",
			chartYaml: "apiVersion: v2\nname: my-chart\n",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(tt.chartYaml), 0o644); err != nil {
				t.Fatal(err)
			}

			result := (&helm.Module{}).Detect(dir)

			if got := result.SuggestedConfig.Extras["chart_version"]; got != tt.want {
				t.Errorf("Extras[chart_version] = %q, want %q", got, tt.want)
			}
			// The chart version is not the Helm tool version.
			if result.SuggestedConfig.Version != "" {
				t.Errorf("Version = %q, want empty", result.SuggestedConfig.Version)
			}
			if tt.want == "" {
				return
			}
			wantEvidence := "chart version " + tt.want
			if !slices.Contains(result.Evidence, wantEvidence) {
				t.Errorf("evidence %v should contain %q", result.Evidence, wantEvidence)
			}
		})
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

func TestDevenvPackages(t *testing.T) {
	m := &helm.Module{}
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"kubernetes-helm", "kubeconform"}
	if len(pkgs) != len(expected) {
		t.Fatalf("DevenvPackages() returned %d packages, want %d", len(pkgs), len(expected))
	}
	for i, pkg := range pkgs {
		if pkg != expected[i] {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &helm.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment != "" {
		t.Errorf("DevenvNixFragment() = %q, want empty string (packages moved to DevenvPackages)", fragment)
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
