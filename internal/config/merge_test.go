package config

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
