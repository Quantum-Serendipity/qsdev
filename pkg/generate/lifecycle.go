package generate

import (
	"fmt"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// LifecyclePhase identifies a point in the generation pipeline where hooks
// can observe or mutate intermediate state.
type LifecyclePhase int

const (
	// PostCollect runs after all fragments are collected but before they are
	// resolved into files. Hooks may inspect or mutate the fragment slice.
	PostCollect LifecyclePhase = iota

	// PostResolve runs after fragments are resolved into GeneratedFile slices
	// but before they are written to disk. Hooks may inspect or mutate the
	// file slice.
	PostResolve
)

var lifecyclePhaseNames = [...]string{
	PostCollect: "post-collect",
	PostResolve: "post-resolve",
}

func (p LifecyclePhase) String() string {
	if int(p) >= 0 && int(p) < len(lifecyclePhaseNames) {
		return lifecyclePhaseNames[p]
	}
	return "unknown"
}

// LifecycleContext carries mutable state into a lifecycle hook. Exactly one of
// Fragments or Files is non-nil depending on the phase.
type LifecycleContext struct {
	Phase     LifecyclePhase
	Fragments *[]types.FragmentEntry // mutable; set for PostCollect
	Files     *[]types.GeneratedFile // mutable; set for PostResolve
	Answers   types.WizardAnswers
}

// LifecycleHook is the interface that lifecycle hooks implement.
type LifecycleHook interface {
	Execute(ctx LifecycleContext) error
}

// LifecycleHookFunc is an adapter that allows using ordinary functions as
// LifecycleHook implementations.
type LifecycleHookFunc func(ctx LifecycleContext) error

// Execute calls the underlying function.
func (f LifecycleHookFunc) Execute(ctx LifecycleContext) error {
	return f(ctx)
}

// HookRegistration binds a hook to a phase with an owner label and priority.
type HookRegistration struct {
	Owner    string
	Phase    LifecyclePhase
	Priority int // lower runs first
	Hook     LifecycleHook
}

// LifecycleHookRegistry stores and executes lifecycle hooks. A nil registry
// is safe to call Execute on (it returns nil).
type LifecycleHookRegistry struct {
	hooks []HookRegistration
}

// NewLifecycleHookRegistry returns an empty registry ready for use.
func NewLifecycleHookRegistry() *LifecycleHookRegistry {
	return &LifecycleHookRegistry{}
}

// Register adds a hook registration to the registry.
func (r *LifecycleHookRegistry) Register(reg HookRegistration) {
	r.hooks = append(r.hooks, reg)
}

// Execute runs all hooks registered for the given phase in priority order.
// If any hook returns an error, execution stops and the error is returned.
func (r *LifecycleHookRegistry) Execute(phase LifecyclePhase, ctx LifecycleContext) error {
	if r == nil {
		return nil
	}

	// Collect hooks for this phase.
	var phaseHooks []HookRegistration
	for _, h := range r.hooks {
		if h.Phase == phase {
			phaseHooks = append(phaseHooks, h)
		}
	}

	// Sort by priority (lower first), preserving registration order for ties.
	sort.SliceStable(phaseHooks, func(i, j int) bool {
		return phaseHooks[i].Priority < phaseHooks[j].Priority
	})

	for _, h := range phaseHooks {
		if err := h.Hook.Execute(ctx); err != nil {
			return fmt.Errorf("lifecycle hook %q (phase %s): %w", h.Owner, phase, err)
		}
	}
	return nil
}

// RemoveByOwner removes all hooks registered by the given owner.
func (r *LifecycleHookRegistry) RemoveByOwner(owner string) {
	var kept []HookRegistration
	for _, h := range r.hooks {
		if h.Owner != owner {
			kept = append(kept, h)
		}
	}
	r.hooks = kept
}

// HookCount returns the total number of registered hooks.
func (r *LifecycleHookRegistry) HookCount() int {
	if r == nil {
		return 0
	}
	return len(r.hooks)
}
