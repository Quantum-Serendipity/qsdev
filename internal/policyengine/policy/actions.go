package policy

import (
	"strings"
)

type ActionHandler interface {
	Execute(rule *PolicyRule, ctx *EvalContext) PolicyDecision
}

type blockAction struct{}

func (a blockAction) Execute(rule *PolicyRule, ctx *EvalContext) PolicyDecision {
	return PolicyDecision{
		Action:   Block,
		ExitCode: 2,
		RuleID:   rule.ID,
		Message:  interpolateMessage(rule.Action.Message, ctx),
	}
}

type warnAction struct{}

func (a warnAction) Execute(rule *PolicyRule, ctx *EvalContext) PolicyDecision {
	return PolicyDecision{
		Action:   Warn,
		ExitCode: 0,
		RuleID:   rule.ID,
		Message:  interpolateMessage(rule.Action.Message, ctx),
		Findings: []Finding{
			{
				RuleID:   rule.ID,
				Category: rule.Category,
				Severity: rule.Severity,
				Message:  interpolateMessage(rule.Action.Message, ctx),
				Monitor:  false,
			},
		},
	}
}

type auditAction struct{}

func (a auditAction) Execute(rule *PolicyRule, ctx *EvalContext) PolicyDecision {
	return PolicyDecision{
		Action:   Audit,
		ExitCode: 0,
		RuleID:   rule.ID,
		Message:  interpolateMessage(rule.Action.Message, ctx),
		Findings: []Finding{
			{
				RuleID:   rule.ID,
				Category: rule.Category,
				Severity: rule.Severity,
				Message:  interpolateMessage(rule.Action.Message, ctx),
				Monitor:  true,
			},
		},
	}
}

type promptAction struct{}

func (a promptAction) Execute(rule *PolicyRule, ctx *EvalContext) PolicyDecision {
	// A `prompt` verdict is meant to ask the operator to confirm the call and
	// block on timeout using a configurable default. Genuine interactive
	// confirmation is deferred to v2, so we cannot obtain a real decision on
	// either a piped or a TTY stdin; the verdict resolves to its timeout
	// default. That default is FAIL-CLOSED: only an explicit allow default
	// (default_on_timeout: allow) lets the call through. This holds on an
	// interactive terminal too — the previous implementation inverted the
	// intent and silently ALLOWED on a TTY, so a "prompt" verdict never gated
	// interactive calls.
	if promptDefaultAllows(rule.Action.DefaultOnTimeout) {
		return PolicyDecision{
			Action:   Prompt,
			ExitCode: 0,
			RuleID:   rule.ID,
			Message:  interpolateMessage(rule.Action.Message, ctx),
		}
	}

	return PolicyDecision{
		Action:   Block,
		ExitCode: 2,
		RuleID:   rule.ID,
		Message:  interpolateMessage(rule.Action.Message, ctx),
	}
}

// promptDefaultAllows reports whether a prompt's default_on_timeout explicitly
// opts into allowing the call when no interactive decision is available.
// Anything else — including the empty/unset value and the "deny"/"block"
// values used by the shipped semantic rules — is fail-closed.
func promptDefaultAllows(defaultOnTimeout string) bool {
	switch strings.ToLower(strings.TrimSpace(defaultOnTimeout)) {
	case "allow", "proceed", "continue", "warn":
		return true
	default:
		return false
	}
}

func ResolveAction(actionType ActionType) ActionHandler {
	switch actionType {
	case Block:
		return blockAction{}
	case Warn:
		return warnAction{}
	case Audit:
		return auditAction{}
	case Prompt:
		return promptAction{}
	default:
		// Fail-closed: unknown action types block.
		return blockAction{}
	}
}

func interpolateMessage(template string, ctx *EvalContext) string {
	s := strings.ReplaceAll(template, "{tool_name}", ctx.ToolName)
	s = strings.ReplaceAll(s, "{file_path}", ctx.FilePath)
	s = strings.ReplaceAll(s, "{command}", ctx.Command)
	return s
}
