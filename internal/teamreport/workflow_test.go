package teamreport

import (
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
)

func TestGenerateTeamWorkflowContainsSHAPins(t *testing.T) {
	workflow := GenerateTeamWorkflow()

	// Verify SHA-pinned action references are present.
	expectedPins := []struct {
		name string
		ref  cigeneration.ActionRef
	}{
		{"checkout", cigeneration.ActionCheckout},
		{"harden-runner", cigeneration.ActionHardenRunner},
		{"upload-artifact", cigeneration.ActionUploadArtifact},
	}

	for _, pin := range expectedPins {
		if !strings.Contains(workflow, pin.ref.SHA) {
			t.Errorf("expected SHA pin for %s (%s) in workflow", pin.name, pin.ref.SHA)
		}
	}
}

func TestGenerateTeamWorkflowValidStructure(t *testing.T) {
	workflow := GenerateTeamWorkflow()

	requiredElements := []string{
		"name: Team Posture Dashboard",
		"on:",
		"schedule:",
		"workflow_dispatch:",
		"permissions:",
		"jobs:",
		"aggregate:",
		"runs-on: ubuntu-latest",
		"steps:",
		"Harden Runner",
		"Checkout",
		"Collect posture reports",
		"Restore posture history",
		"Aggregate posture reports",
		"Upload dashboard",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(workflow, elem) {
			t.Errorf("expected %q in workflow output", elem)
		}
	}
}

func TestGenerateTeamWorkflowCronSchedule(t *testing.T) {
	workflow := GenerateTeamWorkflow()

	if !strings.Contains(workflow, "cron:") {
		t.Error("expected cron schedule in workflow")
	}
}

func TestGenerateTeamWorkflowIssueCreation(t *testing.T) {
	workflow := GenerateTeamWorkflow()

	if !strings.Contains(workflow, "create-issues") {
		t.Error("expected issue creation step in workflow")
	}

	if !strings.Contains(workflow, "GH_TOKEN") {
		t.Error("expected GH_TOKEN environment variable")
	}
}

func TestGeneratePerProjectStepsContainsSHAPins(t *testing.T) {
	steps := GeneratePerProjectSteps()

	if !strings.Contains(steps, cigeneration.ActionHardenRunner.SHA) {
		t.Error("expected harden-runner SHA pin in per-project steps")
	}

	if !strings.Contains(steps, cigeneration.ActionUploadArtifact.SHA) {
		t.Error("expected upload-artifact SHA pin in per-project steps")
	}
}

func TestGeneratePerProjectStepsValidStructure(t *testing.T) {
	steps := GeneratePerProjectSteps()

	requiredElements := []string{
		"Harden Runner",
		"Generate posture report",
		"qsdev status --scan --json --audit-level none",
		"Upload posture report",
		"posture-report.json",
		"retention-days:",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(steps, elem) {
			t.Errorf("expected %q in per-project steps output", elem)
		}
	}
}

func TestGeneratePerProjectStepsArtifactNaming(t *testing.T) {
	steps := GeneratePerProjectSteps()

	// Artifact name should include repo info to avoid collisions.
	if !strings.Contains(steps, "posture-report-") {
		t.Error("expected artifact name with posture-report- prefix")
	}
}

// workflowStep is the subset of a GitHub Actions step the tests inspect.
type workflowStep struct {
	Name string            `yaml:"name"`
	If   string            `yaml:"if"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
}

// parseTeamWorkflowSteps parses the generated aggregation workflow as YAML and
// returns its aggregate job's steps.
func parseTeamWorkflowSteps(t *testing.T) []workflowStep {
	t.Helper()
	var wf struct {
		Jobs map[string]struct {
			Steps []workflowStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(GenerateTeamWorkflow()), &wf); err != nil {
		t.Fatalf("generated team workflow is not valid YAML: %v", err)
	}
	job, ok := wf.Jobs["aggregate"]
	if !ok || len(job.Steps) == 0 {
		t.Fatal("generated team workflow has no aggregate job steps")
	}
	return job.Steps
}

// parsePerProjectSteps parses the generated per-project steps as a YAML list.
func parsePerProjectSteps(t *testing.T) []workflowStep {
	t.Helper()
	var steps []workflowStep
	if err := yaml.Unmarshal([]byte(GeneratePerProjectSteps()), &steps); err != nil {
		t.Fatalf("generated per-project steps are not valid YAML: %v", err)
	}
	return steps
}

func findStep(t *testing.T, steps []workflowStep, name string) workflowStep {
	t.Helper()
	for _, s := range steps {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("step %q not found", name)
	return workflowStep{}
}

func stepIndex(steps []workflowStep, name string) int {
	for i, s := range steps {
		if s.Name == name {
			return i
		}
	}
	return -1
}

// TestPerProjectStepsAlwaysUploadReport guards against survivorship bias: the
// report step must not fail on findings, and the upload must run even if it
// does, so degraded projects still reach the dashboard with real vuln counts.
func TestPerProjectStepsAlwaysUploadReport(t *testing.T) {
	t.Parallel()
	steps := parsePerProjectSteps(t)

	gen := findStep(t, steps, "Generate posture report")
	for _, want := range []string{"--audit-level none", "--scan", "--json"} {
		if !strings.Contains(gen.Run, want) {
			t.Errorf("report step %q is missing %s", gen.Run, want)
		}
	}

	upload := findStep(t, steps, "Upload posture report")
	if upload.If != "always()" {
		t.Errorf("upload step if = %q, want always()", upload.If)
	}
	name, _ := upload.With["name"].(string)
	if !strings.HasPrefix(name, postureArtifactPrefix) {
		t.Errorf("artifact name %q does not start with %q", name, postureArtifactPrefix)
	}

	install := stepIndex(steps, "Install qsdev")
	if install < 0 || install > stepIndex(steps, "Generate posture report") {
		t.Error("per-project steps must install qsdev before generating the report")
	}
}

// TestTeamWorkflowCollectsAcrossRepositories guards the cross-repository
// collection: download-artifact only sees the current run, so reports are
// fetched with gh from each repository in scope, using a token that is not the
// repository-scoped GITHUB_TOKEN.
func TestTeamWorkflowCollectsAcrossRepositories(t *testing.T) {
	t.Parallel()
	steps := parseTeamWorkflowSteps(t)

	for _, s := range steps {
		if strings.Contains(s.Uses, cigeneration.ActionDownloadArtifact.Repo) {
			t.Errorf("step %q uses download-artifact, which cannot read other repositories' runs", s.Name)
		}
	}

	collect := findStep(t, steps, "Collect posture reports")
	for _, want := range []string{"gh run download", "--repo", scopeFile, postureArtifactPrefix + "*"} {
		if !strings.Contains(collect.Run, want) {
			t.Errorf("collect step is missing %q:\n%s", want, collect.Run)
		}
	}
	for _, name := range []string{"Collect posture reports", "Create issues for degraded projects"} {
		tok := findStep(t, steps, name).Env["GH_TOKEN"]
		if strings.Contains(tok, "GITHUB_TOKEN") || !strings.Contains(tok, crossRepoTokenSecret) {
			t.Errorf("step %q GH_TOKEN = %q; want the cross-repository secret %s", name, tok, crossRepoTokenSecret)
		}
	}
}

// TestTeamWorkflowRestoresHistoryBeforeAggregating guards the trend history:
// it must be restored from the previous run's dashboard before aggregation, or
// score-drop alerts can never fire.
func TestTeamWorkflowRestoresHistoryBeforeAggregating(t *testing.T) {
	t.Parallel()
	steps := parseTeamWorkflowSteps(t)

	restore := findStep(t, steps, "Restore posture history")
	if !strings.Contains(restore.Run, "--name "+dashboardArtifact) || !strings.Contains(restore.Run, historyFile) {
		t.Errorf("restore step does not restore %s from %s:\n%s", historyFile, dashboardArtifact, restore.Run)
	}
	if stepIndex(steps, "Restore posture history") > stepIndex(steps, "Aggregate posture reports") {
		t.Error("history must be restored before the aggregate step")
	}
	upload := findStep(t, steps, "Upload dashboard")
	if name, _ := upload.With["name"].(string); name != dashboardArtifact {
		t.Errorf("dashboard artifact name = %q, want %q (the restore step reads it)", name, dashboardArtifact)
	}
}

// pipeToShell matches a download piped straight into a shell interpreter.
var pipeToShell = regexp.MustCompile(`\|\s*(sh|bash)\b`)

// TestGeneratedWorkflowsDoNotPipeToShell guards the install step: it must
// download a pinned release and verify its checksum, never curl | sh.
func TestGeneratedWorkflowsDoNotPipeToShell(t *testing.T) {
	t.Parallel()
	for name, steps := range map[string][]workflowStep{
		"team":        parseTeamWorkflowSteps(t),
		"per-project": parsePerProjectSteps(t),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for _, s := range steps {
				if pipeToShell.MatchString(s.Run) || strings.Contains(s.Run, "curl") {
					t.Errorf("step %q pipes a download into a shell:\n%s", s.Name, s.Run)
				}
			}
			install := findStep(t, steps, "Install qsdev")
			for _, want := range []string{"gh release download", "checksums.txt", "sha256sum --check"} {
				if !strings.Contains(install.Run, want) {
					t.Errorf("install step is missing %q:\n%s", want, install.Run)
				}
			}
			if install.Env["QSDEV_VERSION"] == "" {
				t.Error("install step does not pin QSDEV_VERSION")
			}
		})
	}
}
