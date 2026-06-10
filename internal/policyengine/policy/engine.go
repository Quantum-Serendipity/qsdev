package policy

import (
	"fmt"
	"slices"
	"sync/atomic"
)

type EngineOptions struct{}

type SessionStateReader interface {
	SessionOverrides() []string
}

type PolicyEngine struct {
	current    atomic.Pointer[CompiledPolicySet]
	files      []string
	state      SessionStateReader
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
		ctx.SessionOverrides = e.state.SessionOverrides()
	}
	return Evaluate(e.current.Load(), ctx)
}

func (e *PolicyEngine) Reload() error {
	sp, err := LoadPolicyFiles(e.files...)
	if err != nil {
		return fmt.Errorf("reloading policy engine: %w", err)
	}

	compiled, err := Compile(sp)
	if err != nil {
		return fmt.Errorf("reloading policy engine: %w", err)
	}

	old := e.current.Load()
	e.current.Store(compiled)

	if !denyRulesEqual(old.DenyRules, compiled.DenyRules) {
		select {
		case e.denyRuleCh <- compiled.DenyRules:
		default:
		}
	}

	return nil
}

func (e *PolicyEngine) DenyRuleChanges() <-chan []DenyRule {
	return e.denyRuleCh
}

func (e *PolicyEngine) FilePathDenyRules() []DenyRule {
	return e.current.Load().DenyRules
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

func compareDenyRules(a, b DenyRule) int {
	if a.Pattern < b.Pattern {
		return -1
	}
	if a.Pattern > b.Pattern {
		return 1
	}
	if a.Type < b.Type {
		return -1
	}
	if a.Type > b.Type {
		return 1
	}
	return 0
}
