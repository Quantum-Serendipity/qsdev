package rules

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
type EvalContext struct {
	ToolName      string
	FilePath      string // original path from tool input
	CanonicalPath string // resolved via canon.Canonicalize
	Command       string // for Bash tool
	Content       string // for Write/Edit tool
	CWD           string
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
