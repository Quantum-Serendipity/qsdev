package rules

import (
	"encoding/json"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// Verdict represents the outcome of a rule evaluation.
type Verdict int

const (
	Allow Verdict = iota
	Deny
)

func (v Verdict) String() string {
	if v == Deny {
		return "deny"
	}
	return "allow"
}

// EvalContext contains the context for evaluating self-protection rules.
//
// For a Bash tool call the command is shell-parsed exactly once (in
// buildSelfprotectContext) and the result is shared with every rule and the
// evasion checks via Commands/ParseErr, avoiding a re-parse per rule on the
// PreToolUse hot path. A non-nil ParseErr means the command was unparseable;
// rules must fall back to their conservative substring test (fail closed).
type EvalContext struct {
	ToolName      string
	FilePath      string // original path from tool input
	CanonicalPath string // resolved via canon.Canonicalize
	Command       string // for Bash tool
	Content       string // for Write/Edit tool
	// CWD is the session working directory the tool call runs in (the hook
	// envelope's cwd). Relative Bash paths are resolved against it.
	CWD string
	// Edits holds the old->new replacements of an Edit/MultiEdit call, so a
	// rule can reconstruct the resulting file (see FileChange) instead of
	// seeing only the new fragments in Content.
	Edits []TextEdit
	// ToolInput is the raw tool_input object, for rules that inspect fields
	// beyond file_path (the path/paths/source/destination arguments of MCP
	// filesystem tools).
	ToolInput json.RawMessage
	// SensitiveCommands are the CLI's subcommands that only a human may run
	// (cmdutil.SensitiveCommands of the running command tree); SP-014 blocks
	// a shell command invoking one.
	SensitiveCommands []cmdscan.CommandSpec

	// Parsed Bash command, memoized by ParsedCommands. Do not read directly.
	commands       []cmdscan.Command
	parseErr       error
	commandsParsed bool

	// Parsed commands annotated with their effective working directory,
	// memoized by scannedCommands. Do not read directly.
	scanned     []scannedCommand
	scannedDone bool

	// Shared shell verdicts, memoized by lineMentionsProtected and
	// shellMutatesProtected since several rules consult them.
	mentions, mutates         bool
	mentionsDone, mutatesDone bool

	// ps is the command tokenized as PowerShell, memoized by powerShell.
	ps *psLine

	// fs answers every filesystem question the rules ask about the paths of
	// this one decision, so each distinct path or directory is looked up once
	// however many words name it. scannedCommands and hookTargetsFor hand it
	// out; the context must not be copied once they have.
	fs canon.Resolver

	// hookEnv overrides the session description SP-011 resolves hook
	// commands from (tests only; nil describes the running session), and
	// hookTargets memoizes the result. Read through hookTargetsFor.
	hookEnv     *hookEnv
	hookTargets *hookTargets
}

// ParsedCommands returns Command shell-parsed into its simple commands, parsing
// on first use and caching the result on the context. Every rule and the
// evasion checks call this, so the command is parsed exactly once per tool call
// regardless of how many rules inspect it. A non-nil error means the command
// was unparseable; callers must fall back to their conservative substring test
// (fail closed), never fail open.
func (ctx *EvalContext) ParsedCommands() ([]cmdscan.Command, error) {
	if !ctx.commandsParsed {
		if ctx.Command != "" {
			ctx.commands, ctx.parseErr = cmdscan.ParseWithVars(ctx.Command, canon.ShellPathVars())
		}
		ctx.commandsParsed = true
	}
	return ctx.commands, ctx.parseErr
}

// Rule defines a compiled self-protection rule.
type Rule struct {
	ID          string
	Name        string
	Category    string // "self-protection", "mcp-poisoning", "integrity"
	Description string
	Evaluate    func(ctx *EvalContext) (Verdict, string)
}

// RuleMatch records which rule matched and why.
type RuleMatch struct {
	Rule   Rule
	Reason string
}

// RuleSet is an ordered collection of rules that evaluates using deny-overrides combining.
type RuleSet struct {
	rules []Rule
}

// NewRuleSet creates a RuleSet from the given rules.
func NewRuleSet(rules ...Rule) *RuleSet {
	return &RuleSet{rules: rules}
}

// EvaluateAll evaluates all rules against the context.
// Returns (Deny, matches) if any rule denies, (Allow, nil) if all allow.
// All rules are evaluated and all denials are collected (deny-overrides combining).
//
// A shell command that does not parse is also judged by the lines bash still
// runs before the syntax error (cmdscan.ExecutedPrefix), so a mutation
// followed by a stray `fi` cannot hide behind the rules' substring fallback.
func (rs *RuleSet) EvaluateAll(ctx *EvalContext) (Verdict, []RuleMatch) {
	if verdict, matches := rs.evaluate(ctx); verdict == Deny {
		return verdict, matches
	}
	if !cmdscan.IsShellTool(ctx.ToolName) {
		return Allow, nil
	}
	if _, err := ctx.ParsedCommands(); err == nil {
		return Allow, nil
	}
	prefix := cmdscan.ExecutedPrefix(ctx.Command)
	if prefix == "" {
		return Allow, nil
	}
	return rs.evaluate(ctx.withCommand(prefix))
}

// withCommand returns a fresh context for the same tool call with command in
// place of ctx.Command; nothing memoized on ctx carries over.
func (ctx *EvalContext) withCommand(command string) *EvalContext {
	return &EvalContext{
		ToolName:          ctx.ToolName,
		FilePath:          ctx.FilePath,
		CanonicalPath:     ctx.CanonicalPath,
		Command:           command,
		Content:           ctx.Content,
		CWD:               ctx.CWD,
		Edits:             ctx.Edits,
		ToolInput:         ctx.ToolInput,
		SensitiveCommands: ctx.SensitiveCommands,
		hookEnv:           ctx.hookEnv,
	}
}

// evaluate runs every rule against ctx with deny-overrides combining.
func (rs *RuleSet) evaluate(ctx *EvalContext) (Verdict, []RuleMatch) {
	var matches []RuleMatch
	for _, r := range rs.rules {
		verdict, reason := r.Evaluate(ctx)
		if verdict == Deny {
			matches = append(matches, RuleMatch{Rule: r, Reason: reason})
		}
	}
	if len(matches) > 0 {
		return Deny, matches
	}
	return Allow, nil
}

// Rules returns the underlying rule slice for inspection and listing.
func (rs *RuleSet) Rules() []Rule {
	return rs.rules
}
