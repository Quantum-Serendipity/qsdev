package helm_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
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

// TestDenyRules covers W126 with Claude Code's own matching semantics:
// cluster-changing, code-running, lock-bypassing and secret-printing helm
// subcommands are denied, including after global flags, while read-only
// commands stay allowed.
func TestDenyRules(t *testing.T) {
	t.Parallel()
	rules := (&helm.Module{}).DenyRules(ecosystem.ModuleConfig{})
	matches := func(cmd string) bool {
		return slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, cmd) })
	}
	denied := []string{
		"helm install api ./charts/api",
		"helm --kube-context prod install api ./charts/api",
		"helm -n prod upgrade api ./charts/api",
		"helm upgrade --install api .",
		"helm --kube-context prod uninstall api",
		"helm delete api",
		"helm rollback api 1",
		"helm plugin install https://github.com/example/helm-x",
		"helm dependency update",
		"helm dep update charts/api",
		"helm repo add example https://charts.example.com",
		"helm get values api --all",
		"helm -n prod get manifest api",
		"helm un api",
		"helm del api",
		"helm dep up charts/api",
		"helm dependencies update",
		"helm dependency up",
		"helm plugin add https://github.com/example/helm-x",
		"helm plugin up x",
		"env HELM_DEBUG=1 helm install api .",
		"env helm --kube-context prod upgrade api .",
		"kubectl config view --raw",
		"kubectl config view --minify --raw",
		"cat ~/.kube/config",
	}
	allowed := []string{
		"helm lint charts/api",
		"helm template . --set install=true",
		"helm dependency build",
		"helm list -n prod",
		"helm plugin list",
		"helm get notes api",
		"helm repo list",
		"helm dep build",
		"helm lint charts/uninstaller",
		"kubectl config view",
		"kubectl get pods",
	}
	for _, cmd := range denied {
		if !matches(cmd) {
			t.Errorf("no deny rule blocks %q", cmd)
		}
	}
	for _, cmd := range allowed {
		if matches(cmd) {
			t.Errorf("deny rules over-block %q", cmd)
		}
	}
}

// TestReadDenyRules covers W132: kubeconfig and helm registry/repository
// credentials are read-denied.
func TestReadDenyRules(t *testing.T) {
	t.Parallel()
	rules := (&helm.Module{}).ReadDenyRules(ecosystem.ModuleConfig{})
	for _, want := range []string{"~/.kube/*", "~/.config/helm/registry/*", "~/.config/helm/repositories.yaml"} {
		if !slices.Contains(rules, want) {
			t.Errorf("ReadDenyRules missing %q: %v", want, rules)
		}
	}
}

// TestDetect_ChartSubdirectories covers W129: charts under charts/<name>/
// are detected and linted where they are.
func TestDetect_ChartSubdirectories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     []string
		wantFound bool
		wantConf  ecosystem.Confidence
		wantDirs  string
		wantLint  string
	}{
		{
			name:      "charts layout",
			files:     []string{"charts/api/Chart.yaml", "charts/worker/Chart.yaml"},
			wantFound: true,
			wantConf:  ecosystem.ConfidenceCertain,
			wantDirs:  "charts/api,charts/worker",
			wantLint:  "helm lint charts/api charts/worker",
		},
		{
			name:      "root chart only",
			files:     []string{"Chart.yaml"},
			wantFound: true,
			wantConf:  ecosystem.ConfidenceCertain,
			wantLint:  "helm lint",
		},
		{
			name:      "lock file in subdirectory",
			files:     []string{"deploy/app/Chart.lock"},
			wantFound: true,
			wantConf:  ecosystem.ConfidenceProbable,
			wantLint:  "helm lint",
		},
		{name: "vendored chart ignored", files: []string{"vendor/x/Chart.yaml"}, wantConf: ecosystem.ConfidenceAbsent, wantLint: "helm lint"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				full := filepath.Join(dir, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte("apiVersion: v2\nname: x\nversion: 0.1.0\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			result := (&helm.Module{}).Detect(dir)
			if result.Detected != tt.wantFound || result.Confidence != tt.wantConf {
				t.Fatalf("Detect = (%v, %v), want (%v, %v)", result.Detected, result.Confidence, tt.wantFound, tt.wantConf)
			}
			if got := result.SuggestedConfig.Extra(helm.ExtraChartDirs, ""); got != tt.wantDirs {
				t.Errorf("%s = %q, want %q", helm.ExtraChartDirs, got, tt.wantDirs)
			}
			hooks := (&helm.Module{}).PreCommitHooks(result.SuggestedConfig)
			if hooks[0].Entry != tt.wantLint {
				t.Errorf("helmlint entry = %q, want %q", hooks[0].Entry, tt.wantLint)
			}
		})
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
