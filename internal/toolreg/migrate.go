package toolreg

import "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"

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

// isToolImplicitlyEnabled checks existing WizardAnswers fields to determine
// whether a tool was effectively enabled before the lifecycle system existed.
func isToolImplicitlyEnabled(tool *Tool, answers *types.WizardAnswers) bool {
	switch tool.Name {
	case "attach-guard":
		return answers.Hooks.SafetyBlock
	case "agent-postmortem":
		return answers.AgentTools.PostmortemEnabled
	case "version-sentinel":
		return answers.AgentTools.VersionSentinel
	case "semble":
		return answers.AgentTools.SembleEnabled
	case "trail-of-bits-skills":
		return containsStr(answers.Skills, "security-review")
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

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
