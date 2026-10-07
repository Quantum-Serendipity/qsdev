// Package judge runs the self-protection checks that apply to any tool call
// in the order the selfprotect hook applies them: the command-size limit,
// the evasion layer, the Tier 1 rules, the shell gate-dodge check and git
// code execution. The hook adds the Write/Edit content checks after it; the
// MCP server's qsdev_nix_run tool judges each Bash command line it is
// equivalent to with it, so the tool cannot do what a Bash call may not.
package judge

import (
	"fmt"
	"path"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/evasion"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// LimitRuleID is the rule a command line with more simple commands than
// hookio.MaxSimpleCommands is denied under.
const LimitRuleID = "SP-LIMIT"

// Denial is why a tool call is refused: the rule, or for the evasion layer
// its category, and the reason.
type Denial struct {
	RuleID string
	// Evasion is set when the evasion layer refused the call; RuleID then
	// holds its category.
	Evasion bool
	Reason  string
}

// Evaluate judges ctx and returns why it is denied, or false when these
// checks allow it. ctx.ParsedCommands is memoized, so the checks share one
// parse.
func Evaluate(ctx *rules.EvalContext) (Denial, bool) {
	cmds, parseErr := ctx.ParsedCommands()
	// Over the cap the rules could outlast the hook's deadline. This must be
	// a deny, not a parse error: a parse error makes rules fall back to
	// substring tests rather than deny.
	if len(cmds) > hookio.MaxSimpleCommands {
		return Denial{RuleID: LimitRuleID, Reason: fmt.Sprintf("command has %d simple commands, more than the %d evaluated; "+
			"split it up or write it to a script with the Write tool", len(cmds), hookio.MaxSimpleCommands)}, true
	}
	if blocked, category, reason := evasion.CheckParsed(ctx.ToolName, ctx.Command, ctx.FilePath, cmds, parseErr); blocked {
		return Denial{RuleID: category, Evasion: true, Reason: reason}, true
	}
	if verdict, matches := rules.Tier1Rules.EvaluateAll(ctx); verdict == rules.Deny {
		return Denial{RuleID: matches[0].Rule.ID, Reason: matches[0].Reason}, true
	}
	if ruleID, reason, blocked := shellGateDodge(ctx); blocked {
		return Denial{RuleID: ruleID, Reason: reason}, true
	}
	if reason, blocked := rules.GitCodeExecution(ctx); blocked {
		return Denial{RuleID: rules.GitCodeExecutionRuleID, Reason: reason}, true
	}
	return Denial{}, false
}

// shellGateDodge blocks a shell command that rewrites a file guarded by a
// gate-dodge result rule: the before/after check runs only for Write and
// Edit, so a shell append, delete or in-place edit, or a package manager's
// config command, would skip it. Reading the file stays allowed.
func shellGateDodge(ctx *rules.EvalContext) (ruleID, reason string, blocked bool) {
	name, ok := rules.BashRewritesFile(ctx, canon.GuardedConfigFiles())
	if !ok {
		name = configCommandTarget(ctx)
	}
	if name == "" {
		return "", "", false
	}
	return gatedodge.ResultRuleFor(name).ID,
		"shell command changes " + name + ", whose security settings are only verified for the Edit and Write tools; make the change with Edit or Write",
		true
}

// configCommandTarget returns the guarded file a package-manager command on
// the line rewrites (see gatedodge.ConfigCommandTarget), or "". An
// unparseable line is left to BashRewritesFile, which fails closed when it
// names a guarded file.
func configCommandTarget(ctx *rules.EvalContext) string {
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		return ""
	}
	for _, c := range cmds {
		if name := gatedodge.ConfigCommandTarget(path.Base(c.Name), c.Args); name != "" {
			return name
		}
	}
	return ""
}
