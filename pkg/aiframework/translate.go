package aiframework

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// RulePatterns extracts the non-empty Pattern strings from permission rules,
// preserving order. Framework adapters share it when translating a
// PermissionPolicy into native rule lists.
func RulePatterns(rules []PermissionRule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if r.Pattern != "" {
			out = append(out, r.Pattern)
		}
	}
	return out
}

// HookChoicesFromSpecs maps framework-agnostic hook specs onto the generator
// HookChoices by matching each spec's Command against the known hook logic
// IDs. Unrecognised commands are ignored. Callers that render a Claude Code
// configuration must still apply WizardAnswers.ApplyClaudeHookDefaults so the
// always-on hooks (self-protection) are present even when no spec asks for them.
func HookChoicesFromSpecs(specs []HookSpec) types.HookChoices {
	var c types.HookChoices
	for _, s := range specs {
		switch HookLogicID(s.Command) {
		case LogicPackageGuard:
			c.SafetyBlock = true
		case LogicCredentialScan:
			c.CredentialScan = true
		case LogicDestructiveBlock:
			c.DestructivePrevention = true
		case LogicAgentSelfProtection:
			c.SelfProtection = true
		case LogicFileBoundary:
			c.FileBoundary = true
		case LogicToolGates:
			c.ToolGates = true
		}
	}
	return c
}
