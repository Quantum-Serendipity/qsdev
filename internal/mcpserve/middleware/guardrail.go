package middleware

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Verdict is the result of evaluating the permission cascade.
type Verdict int

const (
	// VerdictAllow permits the call.
	VerdictAllow Verdict = iota
	// VerdictDeny rejects the call.
	VerdictDeny
)

// Rule is one subject's (agent or user) permission rule. Matching is by tool
// name first, then by category. DenyTools/DenyCategories reject; AllowTools/
// AllowCategories explicitly permit (so a permissive default can be narrowed
// without denying everything). A rule that matches nothing yields no opinion and
// the cascade falls through to the next level.
type Rule struct {
	DenyTools       []string
	DenyCategories  []string
	AllowTools      []string
	AllowCategories []string
}

// match reports whether the rule has an opinion about (tool, category) and, if
// so, the verdict. Deny entries take precedence over allow entries within a
// rule so an explicit deny is never silently overridden.
func (r Rule) match(tool, cat string) (Verdict, bool) {
	if contains(r.DenyTools, tool) || contains(r.DenyCategories, cat) {
		return VerdictDeny, true
	}
	if contains(r.AllowTools, tool) || contains(r.AllowCategories, cat) {
		return VerdictAllow, true
	}
	return VerdictAllow, false
}

// Policy is the permission source consulted by the Guardrail layer. It encodes
// the cascade by_agent > by_user > by_tool_type > default.
//
//   - ByAgent maps an agent id to its Rule (highest precedence).
//   - ByUser maps a user id to its Rule. The universal server does not yet
//     resolve a user identity distinct from the agent (V1), so the Guardrail
//     layer currently passes an empty user; the level is fully implemented and
//     unit-tested so policy files can populate it and a later identity layer can
//     feed it.
//   - ByToolType maps a category to a category-wide default Verdict.
//   - Default is the global fallback when no level above had an opinion.
type Policy struct {
	ByAgent    map[string]Rule
	ByUser     map[string]Rule
	ByToolType map[string]Verdict
	Default    Verdict
}

// DefaultPolicy returns the permissive-by-default policy used when no
// .qsdev.yaml MCP policy is configured. The spec models Guardrail as enforcing
// deny rules drawn from project config; with no such config every category is
// allowed and deny rules subtract from that baseline. This is documented
// permissive-by-default: absence of policy must not block tooling.
func DefaultPolicy() *Policy {
	return &Policy{Default: VerdictAllow}
}

// Decide resolves the effective verdict for (agentID, user, category, tool)
// through the cascade. The first level with an opinion wins; if none has one,
// Default applies.
func (p *Policy) Decide(agentID, user, category, tool string) Verdict {
	if p == nil {
		return VerdictAllow
	}
	if r, ok := p.ByAgent[agentID]; ok {
		if v, matched := r.match(tool, category); matched {
			return v
		}
	}
	if r, ok := p.ByUser[user]; ok {
		if v, matched := r.match(tool, category); matched {
			return v
		}
	}
	if v, ok := p.ByToolType[category]; ok {
		return v
	}
	return p.Default
}

// Guardrail is the permission-enforcement layer (Order 20). On a deny verdict it
// SHORT-CIRCUITS: it returns a tool-level error result (IsError) WITHOUT calling
// next. A deny is deliberately a tool error, not a Go error — returning a Go
// error here would surface to the client as a JSON-RPC protocol error rather
// than a tool result. It marks the audit outcome as denied so the Audit layer
// (which sits outside it) records the precise reason.
type Guardrail struct {
	policy *Policy
}

// Order returns 20.
func (Guardrail) Order() int { return orderGuardrail }

// Handle evaluates the cascade and either short-circuits with a deny result or
// continues the chain.
func (g Guardrail) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	tool := toolName(cc, req)
	cat := category(cc)
	// V1 has no user identity distinct from the agent; pass an empty user so the
	// by_user cascade level is consulted but never matches until populated.
	if g.policy.Decide(agentID(cc), "", cat, tool) == VerdictDeny {
		markDecision(ctx, DecisionDenied)
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("tool %q (category %q) denied by policy", tool, cat),
		}, nil
	}
	return next(ctx, cc, req)
}

func contains(list []string, v string) bool {
	if v == "" {
		return false
	}
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}
