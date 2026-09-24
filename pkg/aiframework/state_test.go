package aiframework

import (
	"context"
	"testing"
)

var _ StateBackend = (*mockStateBackend)(nil)

type mockStateBackend struct{}

func (m *mockStateBackend) ChronicleAppend(_ context.Context, _ ChronicleEntry) error { return nil }
func (m *mockStateBackend) ChronicleRead(_ context.Context, _ ChronicleQuery) ([]ChronicleEntry, error) {
	return nil, nil
}
func (m *mockStateBackend) TaskCreate(_ context.Context, _ TaskSpec) (string, error) {
	return "", nil
}
func (m *mockStateBackend) TaskClaim(_ context.Context, _ string) error { return nil }
func (m *mockStateBackend) TaskUpdate(_ context.Context, _ string, _ TaskStatus, _ string) error {
	return nil
}
func (m *mockStateBackend) TaskComplete(_ context.Context, _ string, _ TaskOutcome) error {
	return nil
}
func (m *mockStateBackend) TaskList(_ context.Context, _ TaskFilter) ([]TaskInfo, error) {
	return nil, nil
}

func TestTaskStatusRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value TaskStatus
		str   string
	}{
		{name: "open", value: TaskOpen, str: "open"},
		{name: "assigned", value: TaskAssigned, str: "assigned"},
		{name: "in_progress", value: TaskInProgress, str: "in_progress"},
		{name: "blocked", value: TaskBlocked, str: "blocked"},
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

			var got TaskStatus
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestTaskStatusUnknown(t *testing.T) {
	t.Parallel()

	unknown := TaskStatus(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestTaskStatusUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var s TaskStatus
	if err := s.UnmarshalText([]byte("cancelled")); err == nil {
		t.Error("UnmarshalText(cancelled) should return error")
	}
}

func TestTaskOutcomeRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value TaskOutcome
		str   string
	}{
		{name: "success", value: OutcomeSuccess, str: "success"},
		{name: "partial", value: OutcomePartial, str: "partial"},
		{name: "failure", value: OutcomeFailure, str: "failure"},
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

			var got TaskOutcome
			if err := got.UnmarshalText(text); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tc.value {
				t.Errorf("UnmarshalText() = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestTaskOutcomeUnknown(t *testing.T) {
	t.Parallel()

	unknown := TaskOutcome(99)
	if got := unknown.String(); got != "unknown" {
		t.Errorf("String() = %q, want %q", got, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Error("MarshalText() should return error for unknown value")
	}
}

func TestTaskOutcomeUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var o TaskOutcome
	if err := o.UnmarshalText([]byte("aborted")); err == nil {
		t.Error("UnmarshalText(aborted) should return error")
	}
}
