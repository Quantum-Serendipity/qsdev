package config

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func boolP(v bool) *bool { return &v }

func TestMergePointerBool_NilNil(t *testing.T) {
	result := mergePointerBool(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", *result)
	}
}

func TestMergePointerBool_NilNonNil(t *testing.T) {
	result := mergePointerBool(nil, boolP(true))
	if result == nil || !*result {
		t.Errorf("expected true, got %v", result)
	}
}

func TestMergePointerBool_NonNilNil(t *testing.T) {
	result := mergePointerBool(boolP(true), nil)
	if result == nil || !*result {
		t.Errorf("expected true (inherited from base), got %v", result)
	}
}

func TestMergePointerBool_NonNilNonNil(t *testing.T) {
	result := mergePointerBool(boolP(true), boolP(false))
	if result == nil || *result {
		t.Errorf("expected false (overlay wins), got %v", result)
	}
}

func TestMergePointerBool_TrueFalse(t *testing.T) {
	result := mergePointerBool(boolP(true), boolP(false))
	if result == nil || *result != false {
		t.Errorf("expected false (overlay wins), got %v", result)
	}
}

func TestMergePointerBool_FalseTrue(t *testing.T) {
	result := mergePointerBool(boolP(false), boolP(true))
	if result == nil || *result != true {
		t.Errorf("expected true (overlay wins), got %v", result)
	}
}

func TestMergeUnionStrings_Empty(t *testing.T) {
	result := mergeUnionStrings(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestMergeUnionStrings_Dedup(t *testing.T) {
	result := mergeUnionStrings([]string{"a", "b"}, []string{"b", "c"})
	expected := []string{"a", "b", "c"}
	if len(result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

func TestMergeUnionStrings_OrderPreserved(t *testing.T) {
	result := mergeUnionStrings([]string{"z", "a"}, []string{"m", "a"})
	expected := []string{"z", "a", "m"}
	if len(result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

func TestMergeReplaceLanguages_EmptyPreserves(t *testing.T) {
	base := []types.LanguageConfig{{Name: "go", Version: "1.22"}}
	result := mergeReplaceLanguages(base, nil)
	if len(result) != 1 || result[0].Name != "go" {
		t.Errorf("expected base preserved, got %v", result)
	}
}

func TestMergeReplaceLanguages_NonEmptyReplaces(t *testing.T) {
	base := []types.LanguageConfig{{Name: "go", Version: "1.22"}}
	overlay := []types.LanguageConfig{{Name: "python", Version: "3.12"}}
	result := mergeReplaceLanguages(base, overlay)
	if len(result) != 1 || result[0].Name != "python" {
		t.Errorf("expected overlay replacement, got %v", result)
	}
}

func TestMergeMapStringAny_NestedMerge(t *testing.T) {
	base := map[string]map[string]any{
		"tool-a": {"key1": "val1", "key2": "val2"},
	}
	overlay := map[string]map[string]any{
		"tool-a": {"key2": "override", "key3": "val3"},
		"tool-b": {"k1": "v1"},
	}
	result := mergeMapStringAny(base, overlay)

	// tool-a should have all three keys.
	if result["tool-a"]["key1"] != "val1" {
		t.Errorf("expected key1=val1, got %v", result["tool-a"]["key1"])
	}
	if result["tool-a"]["key2"] != "override" {
		t.Errorf("expected key2=override, got %v", result["tool-a"]["key2"])
	}
	if result["tool-a"]["key3"] != "val3" {
		t.Errorf("expected key3=val3, got %v", result["tool-a"]["key3"])
	}

	// tool-b should exist.
	if result["tool-b"]["k1"] != "v1" {
		t.Errorf("expected tool-b.k1=v1, got %v", result["tool-b"]["k1"])
	}
}

func TestMergeMapStringAny_OverrideKey(t *testing.T) {
	base := map[string]map[string]any{
		"tool": {"key": "base-value"},
	}
	overlay := map[string]map[string]any{
		"tool": {"key": "overlay-value"},
	}
	result := mergeMapStringAny(base, overlay)
	if result["tool"]["key"] != "overlay-value" {
		t.Errorf("expected overlay-value, got %v", result["tool"]["key"])
	}
}

// TestMergePackageGuard checks that no layer loosens hooks.package_guard:
// lists union, min_age_days takes the larger, new_dep_gate the stricter.
func TestMergePackageGuard(t *testing.T) {
	t.Parallel()
	three, seven := 3, 7
	tests := []struct {
		name          string
		base, overlay types.PackageGuardConfig
		wantDays      *int
		wantGate      string
	}{
		{name: "empty"},
		{name: "overlay raises", base: types.PackageGuardConfig{MinAgeDays: &three, NewDepGate: types.NewDepGateAsk},
			overlay: types.PackageGuardConfig{MinAgeDays: &seven, NewDepGate: types.NewDepGateDeny}, wantDays: &seven, wantGate: types.NewDepGateDeny},
		{name: "overlay cannot lower", base: types.PackageGuardConfig{MinAgeDays: &seven, NewDepGate: types.NewDepGateDeny},
			overlay: types.PackageGuardConfig{MinAgeDays: &three, NewDepGate: types.NewDepGateAllow}, wantDays: &seven, wantGate: types.NewDepGateDeny},
		{name: "overlay only", overlay: types.PackageGuardConfig{MinAgeDays: &three, NewDepGate: types.NewDepGateAllow}, wantDays: &three, wantGate: types.NewDepGateAllow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.base.Denylist, tt.overlay.Denylist = []string{"a"}, []string{"b", "a"}
			got := mergePackageGuard(tt.base, tt.overlay)
			if (got.MinAgeDays == nil) != (tt.wantDays == nil) || (got.MinAgeDays != nil && *got.MinAgeDays != *tt.wantDays) {
				t.Errorf("min_age_days = %v, want %v", got.MinAgeDays, tt.wantDays)
			}
			if got.MinAgeDays != nil && (got.MinAgeDays == tt.base.MinAgeDays || got.MinAgeDays == tt.overlay.MinAgeDays) {
				t.Error("min_age_days aliases an input pointer")
			}
			if got.NewDepGate != tt.wantGate {
				t.Errorf("new_dep_gate = %q, want %q", got.NewDepGate, tt.wantGate)
			}
			if len(got.Denylist) != 2 || got.Denylist[0] != "a" || got.Denylist[1] != "b" {
				t.Errorf("denylist = %v, want [a b]", got.Denylist)
			}
		})
	}
}
