package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const securityScanWorkflow = ".github/workflows/security-scan.yml"

// TestGenerate_WorkflowRunsModuleCICommands is the F406 regression test at
// the generator level: the selected languages' CICommands, configured with
// each language's package manager, must reach the generated CI workflow in
// phase order, so a drifted lock file fails CI instead of installing.
func TestGenerate_WorkflowRunsModuleCICommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		langs   []types.LanguageChoice
		ordered []string // must appear, in this order
		absent  []string
	}{
		{
			name:    "rust locked build before audit",
			langs:   []types.LanguageChoice{{Name: "rust"}},
			ordered: []string{"cargo build --locked", "cargo audit"},
		},
		{
			name:    "package manager selects the frozen install",
			langs:   []types.LanguageChoice{{Name: "javascript", PackageManager: "pnpm"}},
			ordered: []string{"pnpm install --frozen-lockfile"},
			absent:  []string{"npm ci"},
		},
		{
			name:    "installs of every language precede any test or scan",
			langs:   []types.LanguageChoice{{Name: "go"}, {Name: "dotnet"}},
			ordered: []string{"go mod download", "dotnet restore --locked-mode", "go mod verify", "govulncheck ./...", "dotnet list package --vulnerable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files, err := generateInfra(t, infraAnswers("", types.InfraConfig{}, tt.langs...))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			wf, ok := files[securityScanWorkflow]
			if !ok {
				t.Fatalf("no %s generated", securityScanWorkflow)
			}
			if !strings.Contains(wf, "\n  ecosystem-ci:\n") {
				t.Fatalf("workflow has no ecosystem-ci job:\n%s", wf)
			}
			last := -1
			for _, cmd := range tt.ordered {
				i := strings.Index(wf, "\n          "+cmd)
				if i < 0 {
					t.Errorf("workflow does not run %q", cmd)
					continue
				}
				if i < last {
					t.Errorf("%q runs out of phase order", cmd)
				}
				last = i
			}
			for _, cmd := range tt.absent {
				if strings.Contains(wf, cmd) {
					t.Errorf("workflow runs %q, which the language's package manager does not use", cmd)
				}
			}
		})
	}
}

func TestGenerate_WorkflowWithoutLanguagesHasNoEcosystemJob(t *testing.T) {
	t.Parallel()

	files, err := generateInfra(t, infraAnswers("", types.InfraConfig{}))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.Contains(files[securityScanWorkflow], "ecosystem-ci:") {
		t.Error("ecosystem-ci job generated for a project without languages")
	}
}

// TestGenerate_DevenvProvidesCIScanTools guards the ecosystem-ci job's
// runtime: its steps run in the devenv shell, so a scan tool a module's
// CICommands invoke must be a devenv.nix package, or the step fails with
// "command not found" on every run.
func TestGenerate_DevenvProvidesCIScanTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang string
		want []string // nixpkgs attributes in devenv.nix packages
	}{
		{"rust", []string{"pkgs.cargo-audit"}},
		{"python", []string{"pkgs.pip-audit"}},
		{"ruby", []string{"pkgs.bundler-audit"}},
		{"container", []string{"pkgs.syft", "pkgs.grype"}},
		{"go", []string{"pkgs.govulncheck"}},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()
			files, err := generateInfra(t, infraAnswers("", types.InfraConfig{}, types.LanguageChoice{Name: tt.lang}))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			nix := files["devenv.nix"]
			for _, pkg := range tt.want {
				if !strings.Contains(nix, " "+pkg+" ") {
					t.Errorf("devenv.nix does not provide %s, which the ecosystem-ci job runs", pkg)
				}
			}
		})
	}
}
