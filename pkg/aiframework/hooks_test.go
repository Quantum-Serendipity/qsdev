package aiframework

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type mockHookDeployer struct{}

var _ HookDeployer = (*mockHookDeployer)(nil)

func (m *mockHookDeployer) FrameworkID() FrameworkID     { return "" }
func (m *mockHookDeployer) SupportedEvents() []HookEvent { return nil }
func (m *mockHookDeployer) Protocol() HookProtocol       { return HookProtocol{} }
func (m *mockHookDeployer) Deploy(_ context.Context, _ []HookPolicy) ([]types.GeneratedFile, error) {
	return nil, nil
}
func (m *mockHookDeployer) Undeploy(_ context.Context, _ string) error { return nil }

func TestHookEventRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value HookEvent
		text  string
	}{
		{"pre_tool_use", EventPreToolUse, "pre_tool_use"},
		{"post_tool_use", EventPostToolUse, "post_tool_use"},
		{"session_start", EventSessionStart, "session_start"},
		{"session_end", EventSessionEnd, "session_end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.value.String(); got != tt.text {
				t.Fatalf("String() = %q, want %q", got, tt.text)
			}

			b, err := tt.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.text {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.text)
			}

			var got HookEvent
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tt.value {
				t.Fatalf("UnmarshalText() = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestHookEventUnknown(t *testing.T) {
	t.Parallel()

	unknown := HookEvent(99)
	if got := unknown.String(); got != "unknown" {
		t.Fatalf("String() = %q, want %q", got, "unknown")
	}
	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() expected error for unknown value")
	}
}

func TestHookEventUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var e HookEvent
	if err := e.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText() expected error for invalid input")
	}
}

func TestHookInputFormatRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value HookInputFormat
		text  string
	}{
		{"json_stdin", InputJSONStdin, "json_stdin"},
		{"starlark_eval", InputStarlarkEval, "starlark_eval"},
		{"shell_exec", InputShellExec, "shell_exec"},
		{"actions_yaml", InputActionsYAML, "actions_yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.value.String(); got != tt.text {
				t.Fatalf("String() = %q, want %q", got, tt.text)
			}

			b, err := tt.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.text {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.text)
			}

			var got HookInputFormat
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tt.value {
				t.Fatalf("UnmarshalText() = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestHookInputFormatUnknown(t *testing.T) {
	t.Parallel()

	unknown := HookInputFormat(99)
	if got := unknown.String(); got != "unknown" {
		t.Fatalf("String() = %q, want %q", got, "unknown")
	}
	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() expected error for unknown value")
	}
}

func TestHookInputFormatUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var f HookInputFormat
	if err := f.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText() expected error for invalid input")
	}
}

func TestHookResponseFormatRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value HookResponseFormat
		text  string
	}{
		{"exit_code", ResponseExitCode, "exit_code"},
		{"stdout_json", ResponseStdoutJSON, "stdout_json"},
		{"starlark_return", ResponseStarlarkReturn, "starlark_return"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.value.String(); got != tt.text {
				t.Fatalf("String() = %q, want %q", got, tt.text)
			}

			b, err := tt.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.text {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.text)
			}

			var got HookResponseFormat
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tt.value {
				t.Fatalf("UnmarshalText() = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestHookResponseFormatUnknown(t *testing.T) {
	t.Parallel()

	unknown := HookResponseFormat(99)
	if got := unknown.String(); got != "unknown" {
		t.Fatalf("String() = %q, want %q", got, "unknown")
	}
	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() expected error for unknown value")
	}
}

func TestHookResponseFormatUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var f HookResponseFormat
	if err := f.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText() expected error for invalid input")
	}
}

func TestHookEnforcementModeRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value HookEnforcementMode
		text  string
	}{
		{"hard_deny", EnforcementHardDeny, "hard_deny"},
		{"advisory", EnforcementAdvisory, "advisory"},
		{"audit_only", EnforcementAuditOnly, "audit_only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.value.String(); got != tt.text {
				t.Fatalf("String() = %q, want %q", got, tt.text)
			}

			b, err := tt.value.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.text {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.text)
			}

			var got HookEnforcementMode
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText() error: %v", err)
			}
			if got != tt.value {
				t.Fatalf("UnmarshalText() = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestHookEnforcementModeUnknown(t *testing.T) {
	t.Parallel()

	unknown := HookEnforcementMode(99)
	if got := unknown.String(); got != "unknown" {
		t.Fatalf("String() = %q, want %q", got, "unknown")
	}
	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() expected error for unknown value")
	}
}

func TestHookEnforcementModeUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var m HookEnforcementMode
	if err := m.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText() expected error for invalid input")
	}
}
