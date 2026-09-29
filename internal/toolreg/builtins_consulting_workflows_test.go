package toolreg

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// consultingWorkflowNames lists the consulting workflow skills the catalog
// is expected to define.
var consultingWorkflowNames = []string{
	"review-pr",
	"add-tests",
	"upgrade-dep",
	"onboard-me",
	"write-adr",
	"incident-debug",
	"migration-plan",
	"handoff-doc",
}

func TestConsultingWorkflowToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	count := 0
	for _, name := range consultingWorkflowNames {
		toolKey := "consulting-workflow-" + name
		tool, ok := reg.ByName(toolKey)
		if !ok {
			t.Errorf("consulting workflow tool %q not found in registry", toolKey)
			continue
		}
		if tool.Category != CategoryAIAgent {
			t.Errorf("tool %q has category %v, want %v", toolKey, tool.Category, CategoryAIAgent)
		}
		count++
	}

	if count != 8 {
		t.Errorf("expected 8 consulting workflow tools registered, found %d", count)
	}
}

func TestConsultingWorkflowToolsOptIn(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range consultingWorkflowNames {
		toolKey := "consulting-workflow-" + name
		tool, ok := reg.ByName(toolKey)
		if !ok {
			t.Errorf("tool %q not found", toolKey)
			continue
		}
		if tool.Default != OptIn {
			t.Errorf("tool %q has Default = %v, want OptIn", toolKey, tool.Default)
		}
	}
}

func TestConsultingWorkflowReviewPrSupersedesBasic(t *testing.T) {
	reg := DefaultRegistry()

	tool, ok := reg.ByName("consulting-workflow-review-pr")
	if !ok {
		t.Fatal("consulting-workflow-review-pr not found in registry")
	}

	if tool.EnableFunc == nil {
		t.Fatal("EnableFunc is nil for consulting-workflow-review-pr")
	}

	// Start with an answers that has "review-pr" in Skills.
	answers := &types.WizardAnswers{
		Skills: []string{"deploy", "review-pr", "security-review"},
	}

	tool.EnableFunc(answers)

	// The basic "review-pr" should be removed from Skills.
	for _, s := range answers.Skills {
		if s == "review-pr" {
			t.Error("EnableFunc should remove 'review-pr' from Skills, but it's still present")
		}
	}

	// Other skills should remain.
	if !slices.Contains(answers.Skills, "deploy") {
		t.Error("EnableFunc removed 'deploy' from Skills unexpectedly")
	}
	if !slices.Contains(answers.Skills, "security-review") {
		t.Error("EnableFunc removed 'security-review' from Skills unexpectedly")
	}

}

func TestConsultingWorkflowToolsHaveOwnedFiles(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range consultingWorkflowNames {
		toolKey := "consulting-workflow-" + name
		tool, ok := reg.ByName(toolKey)
		if !ok {
			t.Errorf("tool %q not found", toolKey)
			continue
		}

		if len(tool.OwnedFiles) == 0 {
			t.Errorf("tool %q has no owned files", toolKey)
			continue
		}

		expectedPath := ".claude/skills/" + name + "/SKILL.md"
		found := false
		for _, f := range tool.OwnedFiles {
			if f.Path == expectedPath {
				found = true
				if f.Ownership != Exclusive {
					t.Errorf("tool %q file %q ownership = %v, want Exclusive", toolKey, f.Path, f.Ownership)
				}
			}
		}
		if !found {
			t.Errorf("tool %q missing owned file %q", toolKey, expectedPath)
		}
	}
}

func TestConsultingWorkflowToolsLifecycleOnly(t *testing.T) {
	reg := DefaultRegistry()
	var names []string
	for _, name := range consultingWorkflowNames {
		if toolKey := "consulting-workflow-" + name; toolKey != consultingWorkflowReviewPR {
			names = append(names, toolKey)
		}
	}
	assertLifecycleOnly(t, reg, names...)

	// review-pr has a real EnableFunc (see ReviewPrSupersedesBasic) but no
	// disable-time effect.
	tool, ok := reg.ByName(consultingWorkflowReviewPR)
	if !ok {
		t.Fatalf("%s not found", consultingWorkflowReviewPR)
	}
	if tool.DisableFunc != nil {
		t.Errorf("%s has a DisableFunc; it only needs lifecycle bookkeeping", consultingWorkflowReviewPR)
	}
}

func TestConsultingWorkflowToolsHaveDescriptions(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range consultingWorkflowNames {
		toolKey := "consulting-workflow-" + name
		tool, ok := reg.ByName(toolKey)
		if !ok {
			t.Errorf("tool %q not found", toolKey)
			continue
		}

		if tool.Description == "" {
			t.Errorf("tool %q has empty description", toolKey)
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty display name", toolKey)
		}
		if !strings.Contains(tool.DisplayName, "Consulting") {
			t.Errorf("tool %q display name %q should contain 'Consulting'", toolKey, tool.DisplayName)
		}
	}
}
