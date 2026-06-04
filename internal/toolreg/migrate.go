package toolreg

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// InferEnabledTools builds the EnabledTools map from legacy WizardAnswers
// fields for projects created before the lifecycle system existed.
// It only runs when EnabledTools is nil (first lifecycle operation on a
// pre-lifecycle project). Subsequent operations use the persisted map.
func InferEnabledTools(answers *types.WizardAnswers, registry *Registry) {
	if answers.EnabledTools != nil {
		return
	}

	answers.EnabledTools = make(map[string]bool)

	for _, tool := range registry.All() {
		if isToolImplicitlyEnabled(tool, answers) {
			answers.EnabledTools[tool.Name] = true
		}
	}
}

// MergeInferredTools augments an existing EnabledTools map with implicitly
// enabled tools (from answers fields and AlwaysOn registry defaults) without
// overriding entries that are already present.
func MergeInferredTools(answers *types.WizardAnswers, registry *Registry) {
	if answers.EnabledTools == nil {
		answers.EnabledTools = make(map[string]bool)
	}
	for _, tool := range registry.All() {
		if _, explicit := answers.EnabledTools[tool.Name]; explicit {
			continue
		}
		if isToolImplicitlyEnabled(tool, answers) {
			answers.EnabledTools[tool.Name] = true
		}
	}
}

// isToolImplicitlyEnabled checks existing WizardAnswers fields to determine
// whether a tool was effectively enabled before the lifecycle system existed.
func isToolImplicitlyEnabled(tool *Tool, answers *types.WizardAnswers) bool {
	switch tool.Name {
	case ToolAttachGuard:
		return answers.Hooks.SafetyBlock
	case ToolAgentPostmortem:
		return answers.AgentTools.PostmortemEnabled
	case ToolVersionSentinel:
		return answers.AgentTools.VersionSentinel
	case ToolSemble:
		return answers.AgentTools.SembleEnabled
	case ToolTrailOfBitsSkills:
		return slices.Contains(answers.Skills, "security-review")
	default:
		// For tools added in Phase 12+, they weren't present in pre-lifecycle
		// projects, so default to the tool's DefaultPolicy.
		if tool.Default == AlwaysOn {
			return true
		}
		if tool.Default == OnWhenDetected && tool.DetectFunc != nil {
			return tool.DetectFunc(answers.Detected)
		}
		return false
	}
}
