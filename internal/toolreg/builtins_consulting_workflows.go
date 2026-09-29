package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// consultingWorkflowReviewPR is the consulting workflow that supersedes the
// basic "review-pr" skill.
const consultingWorkflowReviewPR = "consulting-workflow-review-pr"

func consultingWorkflowBehaviors() map[string]ToolBehavior {
	return map[string]ToolBehavior{
		// Enabling the consulting review-pr workflow replaces the basic
		// review-pr skill, which would otherwise deploy to the same path.
		consultingWorkflowReviewPR: {
			EnableFunc: func(a *types.WizardAnswers) {
				a.Skills = sliceutil.Remove(a.Skills, "review-pr")
			},
		},
	}
}
