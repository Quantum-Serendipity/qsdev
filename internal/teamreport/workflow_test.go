package teamreport

import (
	"strings"
	"testing"

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
		{"download-artifact", actionDownloadArtifact},
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
		"Download posture reports",
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
		"qsdev status --json",
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
