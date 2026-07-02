package middleware

import (
	"sort"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// PolicyFromConfig derives the Guardrail permission Policy from a project's
// .qsdev.yaml configuration. It is the SINGLE source of truth for the MCP deny
// set: the serve command feeds the returned Policy into the enforcing middleware
// chain, and qsdev_policy_check reports from the same derivation (see
// Policy.DenyToolSet), so a tool reported "denied" and a tool actually blocked on
// an MCP call can never diverge.
//
// Every entry in cfg.Tools.Disabled becomes a global tool-name deny. The deny is
// installed on the ByUser cascade level under the empty-user key: Guardrail.Handle
// evaluates every call with an EMPTY user (V1 has no user identity distinct from
// the agent), so a ByUser[""] rule is consulted for every caller regardless of
// agent id. Tools not named remain allowed — deny rules subtract from the
// permissive-by-default baseline.
//
// A nil cfg or an empty disabled list yields a nil Policy. Both WithPolicy and
// GatewayOptions.Policy treat nil as "keep the permissive default", which is
// exactly correct when the project disables nothing.
func PolicyFromConfig(cfg *types.QsdevConfig) *Policy {
	if cfg == nil || len(cfg.Tools.Disabled) == 0 {
		return nil
	}
	deny := make([]string, len(cfg.Tools.Disabled))
	copy(deny, cfg.Tools.Disabled)
	return &Policy{
		ByUser:  map[string]Rule{"": {DenyTools: deny}},
		Default: VerdictAllow,
	}
}

// DenyToolSet returns the sorted set of tool names this policy denies for every
// caller — the project-wide DenyTools installed by PolicyFromConfig on the
// empty-user cascade level. qsdev_policy_check reports this exact set as the MCP
// Guardrail's enforced deny set, so reporting and enforcement are read from the
// same object. A nil policy or one carrying no global deny returns nil.
func (p *Policy) DenyToolSet() []string {
	if p == nil {
		return nil
	}
	r, ok := p.ByUser[""]
	if !ok || len(r.DenyTools) == 0 {
		return nil
	}
	out := make([]string, len(r.DenyTools))
	copy(out, r.DenyTools)
	sort.Strings(out)
	return out
}
