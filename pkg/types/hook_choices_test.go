package types

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

// TestHookChoiceNames_CoverEveryField fails when a HookChoices field is added
// without a hook name, so no hook can become unreachable from names.
func TestHookChoiceNames_CoverEveryField(t *testing.T) {
	t.Parallel()

	covered := make(map[string]bool)
	for _, name := range HookChoiceNames() {
		var h HookChoices
		if err := h.EnableHook(name); err != nil {
			t.Fatalf("EnableHook(%q): %v", name, err)
		}
		v := reflect.ValueOf(h)
		set := 0
		for i := range v.NumField() {
			if v.Field(i).Bool() {
				covered[v.Type().Field(i).Name] = true
				set++
			}
		}
		if set != 1 {
			t.Errorf("EnableHook(%q) set %d fields, want exactly 1", name, set)
		}
	}

	// SafetyBlockOptOut records an opt-out, not a selectable hook, so it has
	// no hook name by design.
	covered["SafetyBlockOptOut"] = true
	typ := reflect.TypeFor[HookChoices]()
	for i := range typ.NumField() {
		if f := typ.Field(i); !covered[f.Name] {
			t.Errorf("HookChoices.%s has no hook name", f.Name)
		}
	}
}

func TestEnableHook_UnknownName(t *testing.T) {
	t.Parallel()
	var h HookChoices
	err := h.EnableHook("safety_block")
	if !errors.Is(err, ErrUnknownHook) {
		t.Fatalf("EnableHook(unknown) error = %v, want ErrUnknownHook", err)
	}
	if h != (HookChoices{}) {
		t.Errorf("unknown name changed choices: %+v", h)
	}
}

func TestHookChoices_Union(t *testing.T) {
	t.Parallel()

	all := HookChoices{}
	for _, name := range HookChoiceNames() {
		if err := all.EnableHook(name); err != nil {
			t.Fatalf("EnableHook(%q): %v", name, err)
		}
	}
	tests := []struct {
		name string
		h, o HookChoices
		want HookChoices
	}{
		{"both empty", HookChoices{}, HookChoices{}, HookChoices{}},
		{"left only", HookChoices{SafetyBlock: true}, HookChoices{}, HookChoices{SafetyBlock: true}},
		{"right only", HookChoices{}, HookChoices{AuditLog: true}, HookChoices{AuditLog: true}},
		{
			"disjoint fields are added",
			HookChoices{SafetyBlock: true, SelfProtection: true},
			HookChoices{AuditLog: true, CredentialScan: true},
			HookChoices{SafetyBlock: true, SelfProtection: true, AuditLog: true, CredentialScan: true},
		},
		{"overlap stays on", HookChoices{AuditLog: true}, HookChoices{AuditLog: true}, HookChoices{AuditLog: true}},
		{"every field", HookChoices{}, all, all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.h.Union(tt.o); got != tt.want {
				t.Errorf("Union() = %+v, want %+v", got, tt.want)
			}
			if got := tt.o.Union(tt.h); got != tt.want {
				t.Errorf("Union() reversed = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestHookChoices_EnabledNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		h    HookChoices
		want []string
	}{
		{"none", HookChoices{}, nil},
		{"one", HookChoices{AutoFormat: true}, []string{"auto-format"}},
		{"sorted", HookChoices{SafetyBlock: true, AuditLog: true, PreCommit: true}, []string{"audit-log", "pre-commit", "safety-block"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.h.EnabledNames(); !slices.Equal(got, tt.want) {
				t.Errorf("EnabledNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHookChoices_SetSafetyBlock verifies that switching the safety block off
// is the opt-out and switching it on clears it.
func TestHookChoices_SetSafetyBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   HookChoices
		on   bool
		want HookChoices
	}{
		{"off records the opt-out", HookChoices{SafetyBlock: true}, false, HookChoices{SafetyBlockOptOut: true}},
		{"on clears the opt-out", HookChoices{SafetyBlockOptOut: true}, true, HookChoices{SafetyBlock: true}},
		{"other hooks are untouched", HookChoices{AuditLog: true}, false, HookChoices{AuditLog: true, SafetyBlockOptOut: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.in
			h.SetSafetyBlock(tt.on)
			if h != tt.want {
				t.Errorf("SetSafetyBlock(%v) = %+v, want %+v", tt.on, h, tt.want)
			}
		})
	}
}
