package aiframework

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface check.
var _ ConfigRenderer = (*mockConfigRenderer)(nil)

type mockConfigRenderer struct{}

func (m *mockConfigRenderer) FrameworkID() FrameworkID         { return ClaudeCode }
func (m *mockConfigRenderer) Capabilities() ConfigCapabilities { return ConfigCapabilities{} }
func (m *mockConfigRenderer) Render(_ context.Context, _ *PolicyInput) ([]types.GeneratedFile, error) {
	return nil, nil
}
func (m *mockConfigRenderer) Validate(_ context.Context, _ []types.GeneratedFile) []ValidationIssue {
	return nil
}
func (m *mockConfigRenderer) Format() string { return "json" }

func TestValidationSeverityRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value ValidationSeverity
		str   string
	}{
		{name: "warning", value: SeverityWarning, str: "warning"},
		{name: "error", value: SeverityError, str: "error"},
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

			var got ValidationSeverity
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestValidationSeverityUnknown(t *testing.T) {
	t.Parallel()

	unknown := ValidationSeverity(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestValidationSeverityUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var s ValidationSeverity
	if err := s.UnmarshalText([]byte("critical")); err == nil {
		t.Error("UnmarshalText(critical) should return error")
	}
}

func TestAllFrameworksCount(t *testing.T) {
	t.Parallel()

	frameworks := AllFrameworks()
	if got := len(frameworks); got != 9 {
		t.Errorf("AllFrameworks() returned %d entries, want 9", got)
	}
}

func TestAllFrameworksContainsExpectedIDs(t *testing.T) {
	t.Parallel()

	expected := map[FrameworkID]bool{
		ClaudeCode: true, Codex: true, GeminiCLI: true,
		Copilot: true, Aider: true, AmazonQ: true,
		Cursor: true, Windsurf: true, ContinueDev: true,
	}

	for _, fw := range AllFrameworks() {
		if !expected[fw] {
			t.Errorf("unexpected framework ID %q in AllFrameworks()", fw)
		}
		delete(expected, fw)
	}
	for missing := range expected {
		t.Errorf("missing framework ID %q from AllFrameworks()", missing)
	}
}
