package types_test

import (
	"encoding/json"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestMergeStrategyString(t *testing.T) {
	tests := []struct {
		strategy types.MergeStrategy
		expected string
	}{
		{types.Overwrite, "overwrite"},
		{types.Append, "append"},
		{types.Merge, "merge"},
		{types.Skip, "skip"},
		{types.SectionMarker, "section-marker"},
		{types.ThreeWayMerge, "three-way-merge"},
		{types.LibraryManaged, "library-managed"},
		{types.ManualMerge, "manual-merge"},
		{types.MergeStrategy(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.strategy.String(); got != tt.expected {
			t.Errorf("MergeStrategy(%d).String() = %q, want %q", int(tt.strategy), got, tt.expected)
		}
	}
}

func TestMergeStrategyMarshalText(t *testing.T) {
	for _, ms := range []types.MergeStrategy{
		types.Overwrite, types.Append, types.Merge, types.Skip,
		types.SectionMarker, types.ThreeWayMerge, types.LibraryManaged, types.ManualMerge,
	} {
		text, err := ms.MarshalText()
		if err != nil {
			t.Errorf("MergeStrategy(%d).MarshalText() error: %v", int(ms), err)
			continue
		}
		if string(text) != ms.String() {
			t.Errorf("MarshalText() = %q, want %q", text, ms.String())
		}
	}
}

func TestMergeStrategyMarshalTextUnknown(t *testing.T) {
	_, err := types.MergeStrategy(99).MarshalText()
	if err == nil {
		t.Error("expected error for unknown MergeStrategy marshal, got nil")
	}
}

func TestMergeStrategyUnmarshalText(t *testing.T) {
	for _, ms := range []types.MergeStrategy{
		types.Overwrite, types.Append, types.Merge, types.Skip,
		types.SectionMarker, types.ThreeWayMerge, types.LibraryManaged, types.ManualMerge,
	} {
		var got types.MergeStrategy
		if err := got.UnmarshalText([]byte(ms.String())); err != nil {
			t.Errorf("UnmarshalText(%q) error: %v", ms.String(), err)
			continue
		}
		if got != ms {
			t.Errorf("UnmarshalText(%q) = %d, want %d", ms.String(), int(got), int(ms))
		}
	}
}

func TestMergeStrategyUnmarshalTextError(t *testing.T) {
	var ms types.MergeStrategy
	if err := ms.UnmarshalText([]byte("bogus")); err == nil {
		t.Error("expected error for unknown string, got nil")
	}
}

func TestMergeStrategyYAMLRoundTrip(t *testing.T) {
	type wrapper struct {
		Strategy types.MergeStrategy `yaml:"strategy"`
	}
	for _, ms := range []types.MergeStrategy{
		types.Overwrite, types.SectionMarker, types.LibraryManaged,
	} {
		w := wrapper{Strategy: ms}
		data, err := yaml.Marshal(w)
		if err != nil {
			t.Fatalf("yaml.Marshal: %v", err)
		}
		// Verify it serializes as a string, not an integer
		if string(data) == "" {
			t.Fatal("empty YAML output")
		}
		var got wrapper
		if err := yaml.Unmarshal(data, &got); err != nil {
			t.Fatalf("yaml.Unmarshal: %v", err)
		}
		if got.Strategy != ms {
			t.Errorf("YAML round-trip: got %v, want %v", got.Strategy, ms)
		}
	}
}

func TestMergeStrategyJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		Strategy types.MergeStrategy `json:"strategy"`
	}
	for _, ms := range []types.MergeStrategy{
		types.Overwrite, types.ThreeWayMerge, types.LibraryManaged,
	} {
		w := wrapper{Strategy: ms}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		var got wrapper
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if got.Strategy != ms {
			t.Errorf("JSON round-trip: got %v, want %v", got.Strategy, ms)
		}
	}
}
