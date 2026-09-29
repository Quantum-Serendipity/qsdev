package policy

import (
	"cmp"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type EngineOptions struct{}

// SessionStateReader resolves the bypass grants in force for a call's scope.
type SessionStateReader interface {
	ActiveOverrides(scope BypassScope, now time.Time) ActiveOverrides
}

// CommandTokenConsumer redeems one-shot command-tier bypass tokens.
type CommandTokenConsumer interface {
	ConsumeCommandTokens(scope BypassScope, ruleIDs []string, now time.Time) error
}

type PolicyEngine struct {
	current atomic.Pointer[CompiledPolicySet]
	files   []string
	state   SessionStateReader
	// reloadMu serializes Reload so the swap-compare-publish sequence of one
	// reload cannot interleave with another's.
	reloadMu sync.Mutex
	// denyRuleCh carries the latest normalized deny-rule set after a reload
	// changed it. It holds at most one value, and a stale value still buffered
	// is replaced by the newer one (latest wins).
	denyRuleCh chan []DenyRule
}

func NewPolicyEngine(files []string, state SessionStateReader, _ EngineOptions) (*PolicyEngine, error) {
	sp, err := LoadPolicyFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("creating policy engine: %w", err)
	}

	compiled, err := Compile(sp)
	if err != nil {
		return nil, fmt.Errorf("creating policy engine: %w", err)
	}

	e := &PolicyEngine{
		files:      files,
		state:      state,
		denyRuleCh: make(chan []DenyRule, 1),
	}
	e.current.Store(compiled)

	return e, nil
}

func (e *PolicyEngine) Evaluate(ctx *EvalContext) PolicyDecision {
	if e.state != nil {
		ctx.Overrides = e.state.ActiveOverrides(ctx.BypassScope(), time.Now())
	}
	return Evaluate(e.current.Load(), ctx)
}

// ConsumeCommandTokens redeems the command-tier tokens a decision for ctx
// relied on (its ConsumedTokens). It fails, and the call must then be blocked,
// when the engine's session state cannot redeem tokens or a token was already
// spent by another call.
func (e *PolicyEngine) ConsumeCommandTokens(ctx *EvalContext, ruleIDs []string) error {
	if len(ruleIDs) == 0 {
		return nil
	}
	consumer, ok := e.state.(CommandTokenConsumer)
	if !ok {
		return fmt.Errorf("consuming command bypass tokens %v: %w", ruleIDs, ErrBypassTokenUnavailable)
	}
	if err := consumer.ConsumeCommandTokens(ctx.BypassScope(), ruleIDs, time.Now()); err != nil {
		return fmt.Errorf("consuming command bypass tokens: %w", err)
	}
	return nil
}

// Reload re-reads and recompiles the policy files and swaps in the result. When
// the file-path deny rules changed, the new normalized set is published on
// DenyRuleChanges, replacing any earlier set a consumer has not yet received.
func (e *PolicyEngine) Reload() error {
	e.reloadMu.Lock()
	defer e.reloadMu.Unlock()

	sp, err := LoadPolicyFiles(e.files...)
	if err != nil {
		return fmt.Errorf("reloading policy engine: %w", err)
	}

	compiled, err := Compile(sp)
	if err != nil {
		return fmt.Errorf("reloading policy engine: %w", err)
	}

	old := e.current.Swap(compiled)

	next := normalizeDenyRules(compiled.DenyRules)
	if !denyRulesEqual(normalizeDenyRules(old.DenyRules), next) {
		e.publishDenyRules(next)
	}

	return nil
}

// publishDenyRules delivers rules on denyRuleCh with latest-wins semantics: a
// previously published set that the consumer has not received yet is dropped
// in favor of rules, so a consumer never applies an outdated deny set.
func (e *PolicyEngine) publishDenyRules(rules []DenyRule) {
	select {
	case <-e.denyRuleCh:
	default:
	}
	select {
	case e.denyRuleCh <- rules:
	default:
	}
}

// DenyRuleChanges returns a channel that receives the normalized file-path deny
// rules (the same form FilePathDenyRules returns) whenever Reload changes them.
func (e *PolicyEngine) DenyRuleChanges() <-chan []DenyRule {
	return e.denyRuleCh
}

func (e *PolicyEngine) FilePathDenyRules() []DenyRule {
	return normalizeDenyRules(e.current.Load().DenyRules)
}

// normalizeDenyRules returns a copy of raw with every type normalized by
// normalizeDenyRuleType.
func normalizeDenyRules(raw []DenyRule) []DenyRule {
	normalized := make([]DenyRule, len(raw))
	for i, r := range raw {
		normalized[i] = normalizeDenyRuleType(r)
	}
	return normalized
}

// normalizeDenyRuleType maps the compiler's condition-specific deny-rule types
// (denied_path from denied_path_check, path_glob from path_glob) onto the single
// "path" type that the MCP confused-deputy check and deny-rule projection act on.
// Without this normalization the trust layer skips every real compiled deny rule
// (it only matches Type=="path"), so the confused-deputy defense is a silent
// no-op against any policy an operator can actually author.
func normalizeDenyRuleType(r DenyRule) DenyRule {
	switch r.Type {
	case "denied_path", "path_glob":
		r.Type = "path"
	}
	return r
}

func (e *PolicyEngine) CurrentRules() []CompiledRule {
	return e.current.Load().Rules
}

func denyRulesEqual(a, b []DenyRule) bool {
	if len(a) != len(b) {
		return false
	}

	aSorted := make([]DenyRule, len(a))
	copy(aSorted, a)
	slices.SortFunc(aSorted, compareDenyRules)

	bSorted := make([]DenyRule, len(b))
	copy(bSorted, b)
	slices.SortFunc(bSorted, compareDenyRules)

	for i := range aSorted {
		if aSorted[i] != bSorted[i] {
			return false
		}
	}
	return true
}

// compareDenyRules totally orders DenyRules over every field, so sorting two
// equal sets yields identical sequences for the element-wise comparison.
func compareDenyRules(a, b DenyRule) int {
	return cmp.Or(
		cmp.Compare(a.Pattern, b.Pattern),
		cmp.Compare(a.Type, b.Type),
		cmp.Compare(a.RuleID, b.RuleID),
		cmp.Compare(a.BypassTier, b.BypassTier),
	)
}
