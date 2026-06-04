package aiframework

import (
	"context"
	"testing"
)

var _ MetricsProvider = (*mockMetricsProvider)(nil)

type mockMetricsProvider struct{}

func (m *mockMetricsProvider) FrameworkID() FrameworkID                         { return ClaudeCode }
func (m *mockMetricsProvider) EmitEvent(_ context.Context, _ MetricEvent) error { return nil }
func (m *mockMetricsProvider) CollectHealth(_ context.Context) (*HealthReport, error) {
	return nil, nil
}
func (m *mockMetricsProvider) ContentRetention() ContentTier { return ContentFull }

func TestContentTierRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value ContentTier
		str   string
	}{
		{name: "full", value: ContentFull, str: "full"},
		{name: "redacted", value: ContentRedacted, str: "redacted"},
		{name: "metadata_only", value: ContentMetadataOnly, str: "metadata_only"},
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

			var got ContentTier
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestContentTierUnknown(t *testing.T) {
	t.Parallel()

	unknown := ContentTier(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestContentTierUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var c ContentTier
	if err := c.UnmarshalText([]byte("verbose")); err == nil {
		t.Error("UnmarshalText(verbose) should return error")
	}
}

func TestHealthStatusRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value HealthStatus
		str   string
	}{
		{name: "healthy", value: StatusHealthy, str: "healthy"},
		{name: "degraded", value: StatusDegraded, str: "degraded"},
		{name: "unhealthy", value: StatusUnhealthy, str: "unhealthy"},
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

			var got HealthStatus
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestHealthStatusUnknown(t *testing.T) {
	t.Parallel()

	unknown := HealthStatus(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestHealthStatusUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var s HealthStatus
	if err := s.UnmarshalText([]byte("critical")); err == nil {
		t.Error("UnmarshalText(critical) should return error")
	}
}

func TestCheckStatusRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value CheckStatus
		str   string
	}{
		{name: "pass", value: CheckPass, str: "pass"},
		{name: "fail", value: CheckFail, str: "fail"},
		{name: "skip", value: CheckSkip, str: "skip"},
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

			var got CheckStatus
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestCheckStatusUnknown(t *testing.T) {
	t.Parallel()

	unknown := CheckStatus(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestCheckStatusUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var c CheckStatus
	if err := c.UnmarshalText([]byte("warn")); err == nil {
		t.Error("UnmarshalText(warn) should return error")
	}
}
