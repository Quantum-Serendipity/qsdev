package types_test

import (
	"encoding/json"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestModificationStatusString(t *testing.T) {
	tests := []struct {
		status   types.ModificationStatus
		expected string
	}{
		{types.Unmodified, "unmodified"},
		{types.Modified, "modified"},
		{types.Deleted, "deleted"},
		{types.New, "new"},
		{types.Unknown, "unknown"},
		{types.ModificationStatus(99), "invalid"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.expected {
			t.Errorf("ModificationStatus(%d).String() = %q, want %q", int(tt.status), got, tt.expected)
		}
	}
}

func TestModificationStatusMarshalText(t *testing.T) {
	for _, ms := range []types.ModificationStatus{
		types.Unmodified, types.Modified, types.Deleted, types.New, types.Unknown,
	} {
		text, err := ms.MarshalText()
		if err != nil {
			t.Errorf("ModificationStatus(%d).MarshalText() error: %v", int(ms), err)
			continue
		}
		if string(text) != ms.String() {
			t.Errorf("MarshalText() = %q, want %q", text, ms.String())
		}
	}
}

func TestModificationStatusMarshalTextInvalid(t *testing.T) {
	_, err := types.ModificationStatus(99).MarshalText()
	if err == nil {
		t.Error("expected error for invalid ModificationStatus marshal, got nil")
	}
}

func TestModificationStatusUnmarshalText(t *testing.T) {
	for _, ms := range []types.ModificationStatus{
		types.Unmodified, types.Modified, types.Deleted, types.New, types.Unknown,
	} {
		var got types.ModificationStatus
		if err := got.UnmarshalText([]byte(ms.String())); err != nil {
			t.Errorf("UnmarshalText(%q) error: %v", ms.String(), err)
			continue
		}
		if got != ms {
			t.Errorf("UnmarshalText(%q) = %d, want %d", ms.String(), int(got), int(ms))
		}
	}
}

func TestModificationStatusUnmarshalTextError(t *testing.T) {
	var ms types.ModificationStatus
	if err := ms.UnmarshalText([]byte("bogus")); err == nil {
		t.Error("expected error for unknown string, got nil")
	}
}

func TestModificationStatusYAMLRoundTrip(t *testing.T) {
	type wrapper struct {
		Status types.ModificationStatus `yaml:"status"`
	}
	for _, ms := range []types.ModificationStatus{
		types.Unmodified, types.Modified, types.Deleted, types.New, types.Unknown,
	} {
		w := wrapper{Status: ms}
		data, err := yaml.Marshal(w)
		if err != nil {
			t.Fatalf("yaml.Marshal: %v", err)
		}
		var got wrapper
		if err := yaml.Unmarshal(data, &got); err != nil {
			t.Fatalf("yaml.Unmarshal: %v", err)
		}
		if got.Status != ms {
			t.Errorf("YAML round-trip: got %v, want %v", got.Status, ms)
		}
	}
}

func TestModificationStatusJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		Status types.ModificationStatus `json:"status"`
	}
	for _, ms := range []types.ModificationStatus{
		types.Unmodified, types.Modified, types.Deleted, types.New, types.Unknown,
	} {
		w := wrapper{Status: ms}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		var got wrapper
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if got.Status != ms {
			t.Errorf("JSON round-trip: got %v, want %v", got.Status, ms)
		}
	}
}

func TestModificationStatusNegativeValueString(t *testing.T) {
	got := types.ModificationStatus(-1).String()
	if got != "invalid" {
		t.Errorf("ModificationStatus(-1).String() = %q, want %q", got, "invalid")
	}
}
