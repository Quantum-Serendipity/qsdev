package toolreg

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestReconcile verifies the one opt-out rule every regeneration path shares:
// only the committed tools.disabled opts out of an always-on tool; an explicit
// off it does not list is dropped and reported, also when the project has no
// committed config at all.
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
			name:        "no committed config drops the answers' off",
			enabled:     map[string]bool{ToolAttachGuard: false},
			wantEnabled: true,
			wantKept:    true,
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

// TestReconcile_OutOfScopeAlwaysOnOffNotPersisted verifies that an explicit
// off for an always-on tool that does not apply to the answers (Claude Code
// switched off) is dropped like any other uncommitted off, so it never
// reaches tools.disabled to be read back later as a committed opt-out, with
// or without a committed config.
func TestReconcile_OutOfScopeAlwaysOnOffNotPersisted(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	probe := &types.WizardAnswers{ClaudeCode: false}
	offs := map[string]bool{}
	for _, tool := range reg.All() {
		if tool.Default == AlwaysOn && !tool.EnforcedFor(probe) {
			offs[tool.Name] = false
		}
	}
	if _, ok := offs[ToolAttachGuard]; !ok || len(offs) < 2 {
		t.Fatalf("catalog out-of-scope always-on tools = %v, want attach-guard among several", offs)
	}
	tests := []struct {
		name      string
		committed *types.ToolsConfig
	}{
		{"no committed config", nil},
		{"empty tools block", &types.ToolsConfig{}},
		{"committed enables them", &types.ToolsConfig{Enabled: []string{ToolAttachGuard}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{ClaudeCode: false, EnabledTools: maps.Clone(offs)}

			kept := Reconcile(answers, reg, tt.committed)

			for name := range offs {
				if enabled, set := answers.EnabledTools[name]; set && !enabled {
					t.Errorf("EnabledTools[%s] = false survived Reconcile; it would persist into tools.disabled", name)
				}
				if !slices.Contains(kept, name) {
					t.Errorf("kept = %v, want %s reported so the caller warns", kept, name)
				}
			}

			// Switching Claude Code back on enforces them again.
			answers.ClaudeCode = true
			Reconcile(answers, reg, tt.committed)
			if !answers.EnabledTools[ToolAttachGuard] || !answers.Hooks.SafetyBlock || answers.Hooks.SafetyBlockOptOut {
				t.Errorf("after re-enabling Claude Code: attach-guard = %v, Hooks = %+v, want the guard on",
					answers.EnabledTools[ToolAttachGuard], answers.Hooks)
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

// TestReconcileAndWarn verifies the warnings every regeneration path shares:
// an always-on tool kept enabled against an explicit off, or one a committed
// config lists as neither enabled nor disabled, is named with its opt-out; a
// new project (no committed config) that keeps its defaults draws none; and a
// committed opt-out of the package guard is reported.
func TestReconcileAndWarn(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	const keptWarning = "always-on tool \"" + ToolAttachGuard + "\" kept enabled"
	const optOutWarning = "package-install guard (" + ToolAttachGuard + ") is disabled"
	tests := []struct {
		name      string
		enabled   map[string]bool
		committed *types.ToolsConfig
		restored  []string
		want      string // "" means no warning
	}{
		{name: "restored by the caller and lacking from the committed config, warned once", enabled: map[string]bool{ToolAttachGuard: true},
			committed: &types.ToolsConfig{Enabled: []string{}}, restored: []string{ToolAttachGuard}, want: keptWarning},
		{name: "no committed config, defaults kept", enabled: map[string]bool{ToolAttachGuard: true}},
		{name: "no committed config, off dropped", enabled: map[string]bool{ToolAttachGuard: false}, want: keptWarning},
		{name: "committed config lists the tool", enabled: map[string]bool{ToolAttachGuard: true},
			committed: &types.ToolsConfig{Enabled: []string{ToolAttachGuard}}},
		{name: "committed config lacks the tool", enabled: map[string]bool{ToolAttachGuard: true},
			committed: &types.ToolsConfig{Enabled: []string{}}, want: keptWarning},
		{name: "committed opt-out", committed: &types.ToolsConfig{Disabled: []string{ToolAttachGuard}}, want: optOutWarning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{ClaudeCode: true, Tier: "standard"}
			SeedAlwaysOn(answers, reg) // the defaults an answers builder starts from
			answers.EnabledTools = tt.enabled
			var buf strings.Builder

			ReconcileAndWarn(&buf, answers, reg, tt.committed, tt.restored...)

			out := buf.String()
			if n := strings.Count(out, keptWarning); n > 1 {
				t.Errorf("warned %d times about the same tool:\n%s", n, out)
			}
			for _, w := range []string{keptWarning, optOutWarning} {
				if got, want := strings.Contains(out, w), w == tt.want; got != want {
					t.Errorf("output contains %q = %v, want %v:\n%s", w, got, want, out)
				}
			}
		})
	}
}

// TestSwitchedOff verifies only an always-on backing switched off between
// before and after is named, and that neither argument is modified.
func TestSwitchedOff(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	before := types.WizardAnswers{ClaudeCode: true, Tier: "standard", EnabledTools: map[string]bool{}}
	SeedAlwaysOn(&before, reg)
	tests := []struct {
		name  string
		after func(a *types.WizardAnswers)
		want  []string
	}{
		{name: "unchanged", after: func(*types.WizardAnswers) {}},
		{name: "safety block switched off", after: func(a *types.WizardAnswers) { a.Hooks.SafetyBlock = false }, want: []string{ToolAttachGuard}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			after := before
			after.EnabledTools = map[string]bool{}
			tt.after(&after)
			wasOn := after.Hooks.SafetyBlock

			if got := SwitchedOff(before, after, reg); !slices.Equal(got, tt.want) {
				t.Errorf("SwitchedOff = %v, want %v", got, tt.want)
			}
			if after.Hooks.SafetyBlock != wasOn || len(after.EnabledTools) != 0 {
				t.Errorf("SwitchedOff modified its argument: %+v", after)
			}
		})
	}
}

// TestReconcileProject verifies the shared load-and-reconcile step: with no
// committed config an answers-file off is dropped and warned about, and a
// committed config that exists but does not load is an error, not a missing
// config.
func TestReconcileProject(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)

	t.Run("no committed config", func(t *testing.T) {
		t.Parallel()
		a := &types.WizardAnswers{ClaudeCode: true, EnabledTools: map[string]bool{ToolAttachGuard: false}}
		var out strings.Builder
		if err := ReconcileProject(&out, t.TempDir(), a, reg); err != nil {
			t.Fatal(err)
		}
		if !a.EnabledTools[ToolAttachGuard] {
			t.Errorf("EnabledTools[attach-guard] = false, want the uncommitted off dropped")
		}
		if !strings.Contains(out.String(), `always-on tool "attach-guard" kept enabled`) {
			t.Errorf("no warning for the dropped off:\n%s", out.String())
		}
	})

	t.Run("unloadable committed config", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, branding.Get().ConfigFile), []byte("tools: [not, a, block\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ReconcileProject(io.Discard, dir, &types.WizardAnswers{}, reg); err == nil {
			t.Error("ReconcileProject succeeded on an unloadable committed config")
		}
	})
}
