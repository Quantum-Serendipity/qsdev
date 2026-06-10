package policy

import (
	"os"
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
	// Full interactive prompting deferred to v2; for now, apply the
	// DefaultOnTimeout fallback unconditionally.
	exitCode := 0
	action := Prompt
	if !isTerminal() || rule.Action.DefaultOnTimeout == "block" {
		exitCode = 2
		action = Block
	}

	return PolicyDecision{
		Action:   action,
		ExitCode: exitCode,
		RuleID:   rule.ID,
		Message:  interpolateMessage(rule.Action.Message, ctx),
	}
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
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
