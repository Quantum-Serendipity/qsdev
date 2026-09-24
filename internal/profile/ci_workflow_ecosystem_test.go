package profile

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // registers every module
)

// parsedJob is the subset of a GitHub Actions job the ecosystem-ci tests
// inspect.
type parsedJob struct {
	RunsOn      string            `yaml:"runs-on"`
	Permissions map[string]string `yaml:"permissions"`
	Defaults    struct {
		Run struct {
			Shell string `yaml:"shell"`
		} `yaml:"run"`
	} `yaml:"defaults"`
	Steps []struct {
		Name  string         `yaml:"name"`
		Uses  string         `yaml:"uses"`
		Shell string         `yaml:"shell"`
		Run   string         `yaml:"run"`
		With  map[string]any `yaml:"with"`
	} `yaml:"steps"`
}

func renderJobs(t *testing.T, p *InfraProfile, in ProjectInputs) map[string]parsedJob {
	t.Helper()
	f, err := p.generateSecurityScanWorkflow(in)
	if err != nil {
		t.Fatalf("generateSecurityScanWorkflow: %v", err)
	}
	var wf struct {
		Jobs map[string]parsedJob `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(f.Content, &wf); err != nil {
		t.Fatalf("generated workflow is not valid YAML: %v\n%s", err, f.Content)
	}
	return wf.Jobs
}

// ciGroups builds phase groups from commands, keeping the order given.
func ciGroups(cmds ...ecosystem.CICommand) []ecosystem.CIPhaseGroup {
	var groups []ecosystem.CIPhaseGroup
	for _, c := range cmds {
		if n := len(groups); n > 0 && groups[n-1].Phase == c.Phase {
			groups[n-1].Commands = append(groups[n-1].Commands, c)
			continue
		}
		groups = append(groups, ecosystem.CIPhaseGroup{Phase: c.Phase, Commands: []ecosystem.CICommand{c}})
	}
	return groups
}

// TestEcosystemCIJob_RunsModuleCommandsInPhaseOrder is the F406 regression
// test: module CICommands (frozen installs, tests, audits) used to be
// implemented by every module but emitted into no workflow, so a drifted
// lock file installed in CI without error.
func TestEcosystemCIJob_RunsModuleCommandsInPhaseOrder(t *testing.T) {
	t.Parallel()

	cmds := []ecosystem.CICommand{
		{Name: "cargo-build-locked", Command: "cargo build --locked", Description: "Build with the lockfile", Phase: ecosystem.CIPhaseInstall},
		{Name: "npm-install", Command: "npm ci --ignore-scripts", Phase: ecosystem.CIPhaseInstall},
		{Name: "go-test", Command: "go test ./...", Phase: ecosystem.CIPhaseTest},
		{Name: "cargo-audit", Command: "cargo audit", Phase: ecosystem.CIPhaseScan},
	}
	job, ok := renderJobs(t, ConsultingDefault, ProjectInputs{CI: ciGroups(cmds...)})["ecosystem-ci"]
	if !ok {
		t.Fatal("workflow has no ecosystem-ci job")
	}

	if job.Permissions["contents"] != "read" || len(job.Permissions) != 1 {
		t.Errorf("permissions = %v, want only contents: read", job.Permissions)
	}
	if !strings.HasPrefix(job.Defaults.Run.Shell, "devenv shell bash -- -eo pipefail") {
		t.Errorf("default shell = %q, want the project's devenv shell with errexit and pipefail", job.Defaults.Run.Shell)
	}

	var runs []string
	var sawNix, sawDevenv, sawCheckout bool
	for _, s := range job.Steps {
		switch {
		case s.Uses == cigeneration.ActionInstallNix.String():
			sawNix = true
		case s.Uses == cigeneration.ActionCheckout.String():
			sawCheckout = true
			if s.With["persist-credentials"] != false {
				t.Error("checkout must set persist-credentials: false")
			}
		case s.Name == "Install devenv":
			sawDevenv = true
			if s.Shell != "bash" {
				t.Errorf("Install devenv shell = %q, want bash (devenv is not installed yet)", s.Shell)
			}
		case s.Run != "":
			if !sawNix || !sawDevenv || !sawCheckout {
				t.Errorf("step %q runs before checkout, Nix and devenv are set up", s.Name)
			}
			runs = append(runs, s.Name+" => "+strings.TrimSuffix(s.Run, "\n"))
		}
	}

	want := []string{
		"install: cargo-build-locked => cargo build --locked",
		"install: npm-install => npm ci --ignore-scripts",
		"test: go-test => go test ./...",
		"scan: cargo-audit => cargo audit",
	}
	if strings.Join(runs, "\n") != strings.Join(want, "\n") {
		t.Errorf("ecosystem-ci steps =\n%s\nwant\n%s", strings.Join(runs, "\n"), strings.Join(want, "\n"))
	}
}

// TestEcosystemCIJob_CommandsSurviveVerbatim checks that shell and YAML
// metacharacters in module commands reach the runner unchanged.
func TestEcosystemCIJob_CommandsSurviveVerbatim(t *testing.T) {
	t.Parallel()

	commands := []string{
		`Rscript -e "renv::restore()"`,
		`find . -name '*.sh' -type f -exec shellcheck {} +`,
		`helm template . | kubeconform --strict`,
		`syft scan docker:img:latest -o spdx-json=sbom.spdx.json`,
		`echo '{{.Repository}}' # trailing: comment`,
		"cmake -B build &&\n  cmake --build build",
	}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			in := ProjectInputs{CI: ciGroups(ecosystem.CICommand{
				Name: "odd: name #1", Command: command, Description: "multi\nline", Phase: ecosystem.CIPhaseTest,
			})}
			job := renderJobs(t, ConsultingDefault, in)["ecosystem-ci"]
			last := job.Steps[len(job.Steps)-1]
			if last.Name != "test: odd: name #1" {
				t.Errorf("step name = %q", last.Name)
			}
			if got := strings.TrimSuffix(last.Run, "\n"); got != command {
				t.Errorf("run = %q, want %q", got, command)
			}
		})
	}
}

func TestEcosystemCIJob_AbsentWithoutCommands(t *testing.T) {
	t.Parallel()

	if _, ok := renderJobs(t, ConsultingDefault, ProjectInputs{})["ecosystem-ci"]; ok {
		t.Error("ecosystem-ci job generated for a project without ecosystem CI commands")
	}
}

func TestEcosystemCIJob_HardenRunnerFollowsProfile(t *testing.T) {
	t.Parallel()

	in := ProjectInputs{CI: ciGroups(ecosystem.CICommand{Name: "t", Command: "true", Phase: ecosystem.CIPhaseTest})}
	for _, p := range []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise} {
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			job := renderJobs(t, p, in)["ecosystem-ci"]
			has := false
			for _, s := range job.Steps {
				if s.Uses == cigeneration.ActionHardenRunner.String() {
					has = true
				}
			}
			if want := p.Scanning.CIProtection == CIProtectionHardenRunner; has != want {
				t.Errorf("harden-runner present = %v, want %v", has, want)
			}
		})
	}
}

// TestEcosystemCIJob_RejectsExpressions guards against a command that GitHub
// would rewrite: ${{ }} is evaluated before the shell runs the step.
func TestEcosystemCIJob_RejectsExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  ecosystem.CICommand
	}{
		{"expression in command", ecosystem.CICommand{Name: "x", Command: "echo ${{ secrets.TOKEN }}"}},
		{"expression in name", ecosystem.CICommand{Name: "${{ github.event.issue.title }}", Command: "true"}},
		{"empty command", ecosystem.CICommand{Name: "x", Command: "  "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ConsultingDefault.generateSecurityScanWorkflow(ProjectInputs{CI: ciGroups(tt.cmd)}); err == nil {
				t.Error("generateSecurityScanWorkflow accepted the command")
			}
		})
	}
}

// TestEcosystemCIJob_RealModules renders every catalog module's CI commands
// into the workflow and checks each one arrives as its own verbatim step.
func TestEcosystemCIJob_RealModules(t *testing.T) {
	t.Parallel()

	modules := ecosystem.DefaultRegistry().All()
	groups, err := ecosystem.AggregateCICommands(modules, func(ecosystem.EcosystemModule) ecosystem.ModuleConfig {
		return ecosystem.ModuleConfig{}
	})
	if err != nil {
		t.Fatalf("AggregateCICommands: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("catalog CI commands cover %d phases, want install, test and scan", len(groups))
	}
	job := renderJobs(t, ConsultingDefault, ProjectInputs{CI: groups})["ecosystem-ci"]

	runs := make(map[string]bool)
	for _, s := range job.Steps {
		runs[strings.TrimSuffix(s.Run, "\n")] = true
	}
	for _, g := range groups {
		for _, c := range g.Commands {
			if !runs[c.Command] {
				t.Errorf("command %q (%s) missing from the ecosystem-ci job", c.Command, c.Name)
			}
		}
	}
}
