package types

import (
	"errors"
	"reflect"
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
