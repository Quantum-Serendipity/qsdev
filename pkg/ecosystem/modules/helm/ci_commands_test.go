package helm_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/helm"
)

// ciCommand returns the named CI command for config, failing the test when
// the module does not emit it.
func ciCommand(t *testing.T, config ecosystem.ModuleConfig, name string) ecosystem.CICommand {
	t.Helper()
	for _, c := range (&helm.Module{}).CICommands(config) {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %s CI command", name)
	return ecosystem.CICommand{}
}

const (
	chartWithDeps = "apiVersion: v2\nname: app\nversion: 1.0.0\ndependencies:\n  - name: redis\n    version: 1.2.3\n    repository: oci://example.test/charts\n"
	chartNoDeps   = "apiVersion: v2\nname: app\nversion: 1.0.0\n"
	// An apiVersion v1 chart lists its dependencies in requirements.yaml,
	// locked by requirements.lock.
	chartV1        = "apiVersion: v1\nname: app\nversion: 1.0.0\n"
	requirementsV1 = "dependencies:\n  - name: redis\n    version: 1.2.3\n    repository: https://example.test/charts\n"
)

// TestCICommands_DependencyBuildEnforcesLock runs helm-dependency-build with
// a stub helm: a chart that declares dependencies must have a Chart.lock, and
// a failing build of any chart fails the step.
func TestCICommands_DependencyBuildEnforcesLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dirs      string // ExtraChartDirs
		files     map[string]string
		helmExit  int
		wantFail  bool
		wantCalls []string
	}{
		{
			name:      "locked chart builds",
			files:     map[string]string{"Chart.yaml": chartWithDeps, "Chart.lock": "x"},
			wantCalls: []string{"helm dependency build ."},
		},
		{
			name:      "chart without dependencies needs no lock",
			files:     map[string]string{"Chart.yaml": chartNoDeps},
			wantCalls: []string{"helm dependency build ."},
		},
		{
			name:     "dependencies without lock refused",
			files:    map[string]string{"Chart.yaml": chartWithDeps},
			wantFail: true,
		},
		{
			name:     "flow-style dependencies without lock refused",
			files:    map[string]string{"Chart.yaml": chartNoDeps + "dependencies: [{name: redis, version: 1.2.3}]\n"},
			wantFail: true,
		},
		{
			name:      "empty flow-style dependencies need no lock",
			files:     map[string]string{"Chart.yaml": chartNoDeps + "dependencies: []\n"},
			wantCalls: []string{"helm dependency build ."},
		},
		{
			name:     "v1 requirements without lock refused",
			files:    map[string]string{"Chart.yaml": chartV1, "requirements.yaml": requirementsV1},
			wantFail: true,
		},
		{
			name:      "v1 requirements with lock builds",
			files:     map[string]string{"Chart.yaml": chartV1, "requirements.yaml": requirementsV1, "requirements.lock": "x"},
			wantCalls: []string{"helm dependency build ."},
		},
		{
			name:      "out-of-sync lock fails",
			files:     map[string]string{"Chart.yaml": chartWithDeps, "Chart.lock": "x"},
			helmExit:  1,
			wantFail:  true,
			wantCalls: []string{"helm dependency build ."},
		},
		{
			name: "every recorded chart is built",
			dirs: "charts/a,charts/b",
			files: map[string]string{
				"charts/a/Chart.yaml": chartNoDeps,
				"charts/b/Chart.yaml": chartWithDeps, "charts/b/Chart.lock": "x",
			},
			wantCalls: []string{"helm dependency build charts/a", "helm dependency build charts/b"},
		},
		{
			name: "unlocked second chart fails",
			dirs: "charts/a,charts/b",
			files: map[string]string{
				"charts/a/Chart.yaml": chartNoDeps,
				"charts/b/Chart.yaml": chartWithDeps,
			},
			wantFail:  true,
			wantCalls: []string{"helm dependency build charts/a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := ecosystem.ModuleConfig{Extras: map[string]string{helm.ExtraChartDirs: tt.dirs}}
			cmd := ciCommand(t, config, "helm-dependency-build")
			dir := shelltest.WriteTree(t, t.TempDir(), tt.files)
			res := shelltest.Run(t, dir, cmd.Command, map[string]shelltest.Stub{"helm": {Exit: tt.helmExit}})
			if (res.Exit != 0) != tt.wantFail {
				t.Errorf("exit = %d, want failure %v; output:\n%s", res.Exit, tt.wantFail, res.Output)
			}
			if strings.Join(res.Calls, "\n") != strings.Join(tt.wantCalls, "\n") {
				t.Errorf("calls = %q, want %q", res.Calls, tt.wantCalls)
			}
		})
	}
}

// TestCICommands_TemplateValidateFailsWhenRenderFails guards the
// render-and-validate pipeline: a chart helm cannot render must fail the step
// even though kubeconform accepts the empty stream, whatever shell options
// the CI runner uses (F436).
func TestCICommands_TemplateValidateFailsWhenRenderFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		helmExit   int
		kubeExit   int
		wantFail   bool
		chartDirs  string
		wantRender []string
	}{
		{name: "valid chart", wantRender: []string{"helm template ."}},
		{name: "render failure", helmExit: 1, wantFail: true},
		{name: "schema failure", kubeExit: 1, wantFail: true},
		{name: "every chart rendered", chartDirs: "a,b", wantRender: []string{"helm template a", "helm template b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := ecosystem.ModuleConfig{Extras: map[string]string{helm.ExtraChartDirs: tt.chartDirs}}
			cmd := ciCommand(t, config, "helm-template-validate")
			res := shelltest.Run(t, t.TempDir(), cmd.Command, map[string]shelltest.Stub{
				"helm":        {Exit: tt.helmExit},
				"kubeconform": {Exit: tt.kubeExit},
			})
			if (res.Exit != 0) != tt.wantFail {
				t.Errorf("exit = %d, want failure %v; output:\n%s", res.Exit, tt.wantFail, res.Output)
			}
			var rendered []string
			for _, call := range res.Calls {
				if strings.HasPrefix(call, "helm ") {
					rendered = append(rendered, call)
				}
			}
			if tt.wantRender != nil && strings.Join(rendered, "\n") != strings.Join(tt.wantRender, "\n") {
				t.Errorf("helm calls = %q, want %q", rendered, tt.wantRender)
			}
		})
	}
}

// TestCICommands_LintsRecordedCharts checks helm-lint covers every recorded
// chart directory, and the root chart when none is recorded.
func TestCICommands_LintsRecordedCharts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dirs string
		want string
	}{
		{dirs: "", want: "helm lint ."},
		{dirs: "charts/b,charts/a", want: "helm lint charts/a charts/b"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			config := ecosystem.ModuleConfig{Extras: map[string]string{helm.ExtraChartDirs: tt.dirs}}
			if got := ciCommand(t, config, "helm-lint").Command; got != tt.want {
				t.Errorf("helm-lint = %q, want %q", got, tt.want)
			}
		})
	}
}
