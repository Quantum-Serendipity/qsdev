package toolreg

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestReconcile verifies the one opt-out rule every regeneration path shares:
// only the committed tools.disabled opts out of an always-on tool; an explicit
// off it does not list is dropped and reported, and without a committed
// config the answers' explicit off stands.
func TestReconcile(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	tests := []struct {
		name        string
		enabled     map[string]bool
		committed   *types.ToolsConfig
		wantEnabled bool
		wantKept    bool
	}{
		{
			name:        "uncommitted off is dropped",
			enabled:     map[string]bool{ToolAttachGuard: false},
			committed:   &types.ToolsConfig{Enabled: []string{ToolAttachGuard}},
			wantEnabled: true,
			wantKept:    true,
		},
		{
			name:        "committed opt-out is kept",
			enabled:     map[string]bool{ToolAttachGuard: false},
			committed:   &types.ToolsConfig{Disabled: []string{ToolAttachGuard}},
			wantEnabled: false,
		},
		{
			name:        "committed opt-out is adopted",
			committed:   &types.ToolsConfig{Disabled: []string{ToolAttachGuard}},
			wantEnabled: false,
		},
		{
			name:        "no committed config keeps the answers' off",
			enabled:     map[string]bool{ToolAttachGuard: false},
			wantEnabled: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{ClaudeCode: true, EnabledTools: tt.enabled}

			kept := Reconcile(answers, reg, tt.committed)

			if got := answers.EnabledTools[ToolAttachGuard]; got != tt.wantEnabled {
				t.Errorf("EnabledTools[attach-guard] = %v, want %v", got, tt.wantEnabled)
			}
			if answers.Hooks.SafetyBlock != tt.wantEnabled {
				t.Errorf("Hooks.SafetyBlock = %v, want %v", answers.Hooks.SafetyBlock, tt.wantEnabled)
			}
			if got := slices.Contains(kept, ToolAttachGuard); got != tt.wantKept {
				t.Errorf("kept = %v, want attach-guard reported = %v", kept, tt.wantKept)
			}
		})
	}
}

// attachGuard returns the catalog's attach-guard tool.
func attachGuard(t *testing.T) *Tool {
	t.Helper()
	tool, ok := catalogRegistry(t).ByName(ToolAttachGuard)
	if !ok {
		t.Fatalf("catalog has no %s tool", ToolAttachGuard)
	}
	return tool
}

// TestToggle_SafetyBlockDisableRecordsOptOut verifies that disabling
// attach-guard records the safety-block opt-out, so the generator-side
// invariant keeps it off, and that enabling it clears the opt-out.
func TestToggle_SafetyBlockDisableRecordsOptOut(t *testing.T) {
	t.Parallel()
	tool := attachGuard(t)
	a := &types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{SafetyBlock: true}}

	tool.DisableFunc(a)
	if a.Hooks.SafetyBlock || !a.Hooks.SafetyBlockOptOut {
		t.Fatalf("after DisableFunc Hooks = %+v, want safety block off and opt-out recorded", a.Hooks)
	}
	a.ApplyClaudeHookDefaults()
	if a.Hooks.SafetyBlock {
		t.Error("ApplyClaudeHookDefaults turned the opted-out safety block back on")
	}

	tool.EnableFunc(a)
	if !a.Hooks.SafetyBlock || a.Hooks.SafetyBlockOptOut {
		t.Errorf("after EnableFunc Hooks = %+v, want safety block on and no opt-out", a.Hooks)
	}
}

// TestToggle_SafetyBlockForceOnClearsUncommittedOptOut verifies that forcing
// attach-guard on clears an opt-out and reports it as an override, while a
// safety block that is already effectively on is not reported.
func TestToggle_SafetyBlockForceOnClearsUncommittedOptOut(t *testing.T) {
	t.Parallel()
	tool := attachGuard(t)
	tests := []struct {
		name         string
		hooks        types.HookChoices
		wantOverrode bool
	}{
		{"opt-out", types.HookChoices{SafetyBlockOptOut: true}, true},
		{"opt-out with a stale safety block", types.HookChoices{SafetyBlock: true, SafetyBlockOptOut: true}, true},
		{"off without opt-out", types.HookChoices{}, true},
		{"already on", types.HookChoices{SafetyBlock: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := &types.WizardAnswers{ClaudeCode: true, Hooks: tt.hooks}

			if got := tool.ForceOnFunc(a); got != tt.wantOverrode {
				t.Errorf("ForceOnFunc() overrode = %v, want %v", got, tt.wantOverrode)
			}
			if !a.Hooks.SafetyBlock || a.Hooks.SafetyBlockOptOut {
				t.Errorf("Hooks = %+v, want safety block on and no opt-out", a.Hooks)
			}
		})
	}
}

// TestReconcile_UncommittedSafetyBlockOptOutDropped verifies that an opt-out
// recorded only in the answers (a hand edit) is dropped and reported when the
// committed config does not list attach-guard under tools.disabled.
func TestReconcile_UncommittedSafetyBlockOptOutDropped(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	tests := []struct {
		name    string
		enabled map[string]bool
	}{
		{"opt-out bit only", nil},
		{"opt-out bit with an explicit off", map[string]bool{ToolAttachGuard: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{
				ClaudeCode:   true,
				EnabledTools: tt.enabled,
				Hooks:        types.HookChoices{SafetyBlockOptOut: true},
			}

			kept := Reconcile(answers, reg, &types.ToolsConfig{Disabled: []string{}})

			if !slices.Contains(kept, ToolAttachGuard) {
				t.Errorf("kept = %v, want %s", kept, ToolAttachGuard)
			}
			if !answers.Hooks.SafetyBlock || answers.Hooks.SafetyBlockOptOut {
				t.Errorf("Hooks = %+v, want safety block on and no opt-out", answers.Hooks)
			}
		})
	}
}

// TestReconcile_CommittedOptOutSetsSafetyBlockOptOut verifies that a
// committed attach-guard opt-out records the safety-block opt-out in the
// answers, including answers saved before the opt-out bit existed, so the
// generator-side invariant keeps the safety block off.
func TestReconcile_CommittedOptOutSetsSafetyBlockOptOut(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	tests := []struct {
		name    string
		enabled map[string]bool
		hooks   types.HookChoices
	}{
		{"undecided answers", nil, types.HookChoices{SafetyBlock: true}},
		{"legacy decided-false answers", map[string]bool{ToolAttachGuard: false}, types.HookChoices{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{ClaudeCode: true, EnabledTools: tt.enabled, Hooks: tt.hooks}

			kept := Reconcile(answers, reg, &types.ToolsConfig{Disabled: []string{ToolAttachGuard}})

			if slices.Contains(kept, ToolAttachGuard) {
				t.Errorf("kept = %v, want no %s", kept, ToolAttachGuard)
			}
			if enabled, set := answers.EnabledTools[ToolAttachGuard]; !set || enabled {
				t.Errorf("EnabledTools[attach-guard] = %v (set %v), want explicit false", enabled, set)
			}
			if !answers.Hooks.SafetyBlockOptOut {
				t.Errorf("Hooks = %+v, want the opt-out recorded", answers.Hooks)
			}
			answers.ApplyClaudeHookDefaults()
			if answers.Hooks.SafetyBlock {
				t.Error("ApplyClaudeHookDefaults turned the committed opt-out back on")
			}
		})
	}
}
