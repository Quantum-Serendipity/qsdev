package aiframework

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// Compile-time interface check.
var _ DetectionAdapter = (*mockDetectionAdapter)(nil)

type mockDetectionAdapter struct{}

func (m *mockDetectionAdapter) FrameworkID() FrameworkID                     { return ClaudeCode }
func (m *mockDetectionAdapter) Detect(_ string) (*FrameworkDetection, error) { return nil, nil }
func (m *mockDetectionAdapter) Markers() []DetectionMarker                   { return nil }

func TestMarkerTypeRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value MarkerType
		str   string
	}{
		{name: "directory", value: MarkerDirectory, str: "directory"},
		{name: "file", value: MarkerFile, str: "file"},
		{name: "binary", value: MarkerBinary, str: "binary"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.value.String(); got != tc.str {
				t.Errorf("String() = %q, want %q", got, tc.str)
			}

			text, err := tc.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(text) != tc.str {
				t.Errorf("MarshalText() = %q, want %q", string(text), tc.str)
			}

			var got MarkerType
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestMarkerTypeUnknown(t *testing.T) {
	t.Parallel()

	unknown := MarkerType(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestMarkerTypeUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var m MarkerType
	if err := m.UnmarshalText([]byte("bogus")); err == nil {
		t.Error("UnmarshalText(bogus) should return error")
	}
}

func TestDetectionMarkerUsesConfidence(t *testing.T) {
	t.Parallel()

	marker := DetectionMarker{
		Type:   MarkerFile,
		Path:   ".claude/settings.json",
		Weight: ecosystem.ConfidenceCertain,
	}
	if marker.Type != MarkerFile {
		t.Errorf("Type = %v, want %v", marker.Type, MarkerFile)
	}
	if marker.Path != ".claude/settings.json" {
		t.Errorf("Path = %q, want %q", marker.Path, ".claude/settings.json")
	}
	if marker.Weight != ecosystem.ConfidenceCertain {
		t.Errorf("Weight = %v, want %v", marker.Weight, ecosystem.ConfidenceCertain)
	}
}
