package rules

import "github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"

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
	CWD           string

	// Parsed Bash command, memoized by ParsedCommands. Do not read directly.
	commands       []cmdscan.Command
	parseErr       error
	commandsParsed bool
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
			ctx.commands, ctx.parseErr = cmdscan.Parse(ctx.Command)
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
func (rs *RuleSet) EvaluateAll(ctx *EvalContext) (Verdict, []RuleMatch) {
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
