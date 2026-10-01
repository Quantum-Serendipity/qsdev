package toolreg

import (
	"fmt"
	"io"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// InferEnabledTools builds the EnabledTools map from legacy WizardAnswers
// fields for projects created before the lifecycle system existed.
// It only runs when EnabledTools is nil (first lifecycle operation on a
// pre-lifecycle project). Subsequent operations use the persisted map. It
// reports what the answers configure and enforces nothing.
func InferEnabledTools(answers *types.WizardAnswers, registry *Registry) {
	if answers.EnabledTools != nil {
		return
	}
	InferTools(answers, registry)
}

// InferTools augments an existing EnabledTools map with implicitly enabled
// tools (from answers fields and AlwaysOn registry defaults) without
// overriding entries that are already present. It enforces nothing, so
// readers that report what a project is configured with (status, list) use
// it rather than MergeInferredTools.
func InferTools(answers *types.WizardAnswers, registry *Registry) {
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

// MergeInferredTools runs InferTools and then EnforceAlwaysOn, the step every
// path that generates or persists answers takes once all answer sources have
// run. It returns the names EnforceAlwaysOn reports, so callers can warn.
func MergeInferredTools(answers *types.WizardAnswers, registry *Registry) []string {
	InferTools(answers, registry)
	return EnforceAlwaysOn(answers, registry)
}

// EnforceAlwaysOn keeps every always-on tool that applies to the answers (see
// Tool.EnforcedFor) enabled whatever the other answer sources chose. The only
// opt-out is an explicit EnabledTools[name] == false, which is what
// tools.disabled in .qsdev.yaml maps to and what `disable --force` writes;
// such a tool is left untouched. Every other such tool is marked enabled and
// its ForceOnFunc switches on what backs it. It returns, sorted, the names
// whose backing had been explicitly switched off.
func EnforceAlwaysOn(answers *types.WizardAnswers, registry *Registry) []string {
	if answers.EnabledTools == nil {
		answers.EnabledTools = make(map[string]bool)
	}
	var overridden []string
	for _, tool := range enforceable(answers, registry) {
		answers.EnabledTools[tool.Name] = true
		if tool.ForceOnFunc != nil && tool.ForceOnFunc(answers) {
			overridden = append(overridden, tool.Name)
		}
	}
	slices.Sort(overridden)
	return overridden
}

// SeedAlwaysOn switches on what backs every always-on tool EnforceAlwaysOn
// would enforce, without recording anything in EnabledTools. Answer builders
// seed their defaults with it, so the later enforcement warns only about a
// source that explicitly switched a backing off.
func SeedAlwaysOn(answers *types.WizardAnswers, registry *Registry) {
	for _, tool := range enforceable(answers, registry) {
		if tool.ForceOnFunc != nil {
			tool.ForceOnFunc(answers)
		}
	}
}

// enforceable returns the always-on tools that apply to answers and that the
// answers have not explicitly disabled.
func enforceable(answers *types.WizardAnswers, registry *Registry) []*Tool {
	var out []*Tool
	for _, tool := range registry.All() {
		if !tool.EnforcedFor(answers) {
			continue
		}
		if enabled, set := answers.EnabledTools[tool.Name]; set && !enabled {
			continue
		}
		out = append(out, tool)
	}
	return out
}

// WarnAlwaysOnRestored writes a warning to w for each always-on tool name
// that was kept enabled against an answer source, naming the only way to opt
// out.
func WarnAlwaysOnRestored(w io.Writer, names []string) {
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "Warning: always-on tool %q kept enabled; opt out with `qsdev disable %s --force`\n", name, name)
	}
}

// WarnSafetyBlockOptOut writes a warning to w when answers configure Claude
// Code with the package-install guard opted out, which only a committed
// tools.disabled entry can keep (see Reconcile). It warns at every tier: at
// supply-chain-only the guard is the only defence.
func WarnSafetyBlockOptOut(w io.Writer, answers *types.WizardAnswers) {
	if !answers.ClaudeCode || !answers.Hooks.SafetyBlockOptOut {
		return
	}
	_, _ = fmt.Fprintf(w, "Warning: the package-install guard (%s) is disabled by tools.disabled in .qsdev.yaml; re-enable with `qsdev enable %s`\n",
		ToolAttachGuard, ToolAttachGuard)
}

// OptOutFlagError rejects a flag that would switch off the always-on tool
// named tool, pointing at the only supported opt-out.
func OptOutFlagError(flag, tool string) error {
	return fmt.Errorf("%s is not supported: %s is always on; opt out explicitly with `qsdev disable %s --force`", flag, tool, tool)
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
		// Accept the legacy name too, so pre-rename .qsdev.yaml files still
		// migrate (the skill was renamed security-review → security-review-owasp
		// to avoid colliding with Claude Code's built-in /security-review).
		return slices.Contains(answers.Skills, "security-review-owasp") ||
			slices.Contains(answers.Skills, "security-review")
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
