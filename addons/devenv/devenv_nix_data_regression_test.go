package devenv

import (
	"maps"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCollectToolPackages_Deterministic(t *testing.T) {
	t.Parallel()
	// Enable every catalog tool that contributes a package or expression so
	// map iteration order has room to reshuffle the output.
	enabled := map[string]bool{}
	for name := range defaultToolNixPackages() {
		enabled[name] = true
	}
	for name := range defaultToolNixExprs() {
		enabled[name] = true
	}
	if len(enabled) < 2 {
		t.Skipf("catalog has %d tools with Nix packages; need at least 2", len(enabled))
	}
	answers := types.WizardAnswers{EnabledTools: enabled}

	wantPkgs, wantExprs := collectToolPackages(answers)
	for i := range 50 {
		gotPkgs, gotExprs := collectToolPackages(answers)
		if !slices.Equal(gotPkgs, wantPkgs) || !slices.Equal(gotExprs, wantExprs) {
			t.Fatalf("run %d produced a different order:\n pkgs %v vs %v\n exprs %v vs %v",
				i, gotPkgs, wantPkgs, gotExprs, wantExprs)
		}
	}

	// The order is the sorted tool-name order, not merely stable by chance.
	nixPkgs := defaultToolNixPackages()
	var sortedPkgs []string
	for _, name := range slices.Sorted(maps.Keys(enabled)) {
		if p, ok := nixPkgs[name]; ok {
			sortedPkgs = append(sortedPkgs, p)
		}
	}
	if !slices.Equal(wantPkgs, sortedPkgs) {
		t.Errorf("packages = %v, want tool-name order %v", wantPkgs, sortedPkgs)
	}
}

func TestOverlayPathExprs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"bare file name is made a path", "go-overlay.nix", "./go-overlay.nix", false},
		{"dot-slash kept", "./nix/go-overlay.nix", "./nix/go-overlay.nix", false},
		{"cleaned", "nix//./go-overlay.nix", "./nix/go-overlay.nix", false},
		{"inner dotdot inside project", "nix/../go-overlay.nix", "./go-overlay.nix", false},
		{"space needs string path", "nix/my overlay.nix", `(./. + "/nix/my overlay.nix")`, false},
		{"nix metachars escaped", `nix/a${b}"c.nix`, `(./. + "/nix/a\${b}\"c.nix")`, false},
		{"absolute rejected", "/tmp/x.nix", "", true},
		{"escape rejected", "../x.nix", "", true},
		{"nested escape rejected", "nix/../../x.nix", "", true},
		{"project root rejected", ".", "", true},
		{"empty rejected", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := overlayPathExprs([]string{tt.in})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("overlayPathExprs(%q) = %v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("overlayPathExprs(%q): %v", tt.in, err)
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("overlayPathExprs(%q) = %v, want [%s]", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildDevenvNixData_EnvVarNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"dotted becomes nested attrset", "API.URL", true},
		{"dash", "MY-VAR", true},
		{"space", "MY VAR", true},
		{"leading digit", "1ABC", true},
		{"equals", "A=B", true},
		{"non-ascii", "Xé", true},
		{"valid", "_Ok_Name1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{EnvVars: map[string]string{tt.key: "v"}}
			_, err := BuildDevenvNixData(answers, ecosystem.NewRegistry())
			if (err != nil) != tt.wantErr {
				t.Errorf("env var name %q: err = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestBuildDevenvNixData_ServiceEnvNotUnset(t *testing.T) {
	t.Parallel()
	base, err := BuildDevenvNixData(types.WizardAnswers{}, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		if !slices.Contains(base.UnsetEnvVars, v) {
			t.Skipf("catalog no longer unsets %s; test premise gone", v)
		}
	}

	answers := types.WizardAnswers{Services: []types.ServiceChoice{{Name: "minio"}}}
	data, err := BuildDevenvNixData(answers, ecosystem.NewRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		if data.EnvVars[v] == "" {
			t.Errorf("MinIO did not set %s", v)
		}
		if slices.Contains(data.UnsetEnvVars, v) {
			t.Errorf("%s is set by MinIO but still unset at shell start", v)
		}
	}
	// Unrelated credentials stay stripped.
	if !slices.Contains(data.UnsetEnvVars, "GITHUB_TOKEN") {
		t.Error("GITHUB_TOKEN dropped from unset list")
	}
	// The leak probes must not fail on the service's own variable.
	if strings.Contains(data.EnterTest, "AWS_SECRET_ACCESS_KEY") {
		t.Error("enterTest still fails when MinIO's AWS_SECRET_ACCESS_KEY is present")
	}
	if strings.Contains(data.EnterShell, "AWS_SECRET_ACCESS_KEY") {
		t.Error("enterShell still warns about MinIO's AWS_SECRET_ACCESS_KEY")
	}
	if !strings.Contains(base.EnterTest, "AWS_SECRET_ACCESS_KEY") {
		t.Error("enterTest no longer probes AWS_SECRET_ACCESS_KEY without MinIO")
	}
}

func TestBuildTaskScripts(t *testing.T) {
	t.Parallel()
	scripts := buildTaskScripts([]ecosystem.TaskDefinition{
		{Name: "build", Description: "Build", Commands: []string{"go build ./..."}},
		{Name: "test", Description: "Test", Commands: []string{"go test ./..."}, DependsOn: []string{"build"}},
		{Name: "lint", Description: "Lint", Commands: []string{"go vet ./...", "golangci-lint run"}, DependsOn: []string{"missing"}},
	})
	want := []TaskScript{
		{Name: "qsdev-build", Description: "Build", Exec: "set -euo pipefail\ngo build ./..."},
		{Name: "qsdev-test", Description: "Test", Exec: "set -euo pipefail\nqsdev-build\ngo test ./..."},
		{Name: "qsdev-lint", Description: "Lint", Exec: "set -euo pipefail\ngo vet ./...\ngolangci-lint run"},
	}
	if !slices.Equal(scripts, want) {
		t.Errorf("buildTaskScripts =\n%#v\nwant\n%#v", scripts, want)
	}
}

func TestBuildTaskScripts_FailingCommandFailsTask(t *testing.T) {
	t.Parallel()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	scripts := buildTaskScripts([]ecosystem.TaskDefinition{
		{Name: "lint", Commands: []string{"false", "true"}},
	})
	if err := exec.Command(bash, "-c", scripts[0].Exec).Run(); err == nil {
		t.Error("task exited 0 although its first command failed")
	}
}
