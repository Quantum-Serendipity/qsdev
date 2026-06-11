package hookio

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseToolCall_Valid(t *testing.T) {
	t.Parallel()

	input := `{"tool_name":"Write","tool_input":{"file_path":"/tmp/test.go","content":"package main"}}`
	call, err := ParseToolCall(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if call.ToolName != "Write" {
		t.Errorf("ToolName = %q, want %q", call.ToolName, "Write")
	}
	if call.ToolInput == nil {
		t.Fatal("ToolInput is nil")
	}

	parsed := ParseInput(call.ToolInput)
	if parsed.FilePath != "/tmp/test.go" {
		t.Errorf("FilePath = %q, want %q", parsed.FilePath, "/tmp/test.go")
	}
	if parsed.Content != "package main" {
		t.Errorf("Content = %q, want %q", parsed.Content, "package main")
	}
}

func TestParseToolCall_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := ParseToolCall(context.Background(), strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "empty hook input") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "empty hook input")
	}
}

func TestParseToolCall_MalformedJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseToolCall(context.Background(), strings.NewReader("{invalid}"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing hook input") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "parsing hook input")
	}
}

func TestParseToolCall_MissingToolName(t *testing.T) {
	t.Parallel()

	_, err := ParseToolCall(context.Background(), strings.NewReader(`{"tool_input":{}}`))
	if err == nil {
		t.Fatal("expected error for missing tool_name")
	}
	if !strings.Contains(err.Error(), "missing tool_name") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing tool_name")
	}
}

func TestParseToolCall_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ParseToolCall(ctx, strings.NewReader(`{"tool_name":"Bash","tool_input":{}}`))
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("error = %v, want %v", err, context.Canceled)
	}
}

func TestParseInput_AllFields(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"file_path":"/etc/passwd","content":"root","command":"cat /etc/passwd"}`)
	input := ParseInput(raw)

	if input.FilePath != "/etc/passwd" {
		t.Errorf("FilePath = %q, want %q", input.FilePath, "/etc/passwd")
	}
	if input.Content != "root" {
		t.Errorf("Content = %q, want %q", input.Content, "root")
	}
	if input.Command != "cat /etc/passwd" {
		t.Errorf("Command = %q, want %q", input.Command, "cat /etc/passwd")
	}
}

func TestParseInput_MissingFields(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"file_path":"/tmp/x"}`)
	input := ParseInput(raw)

	if input.FilePath != "/tmp/x" {
		t.Errorf("FilePath = %q, want %q", input.FilePath, "/tmp/x")
	}
	if input.Content != "" {
		t.Errorf("Content = %q, want empty string", input.Content)
	}
	if input.Command != "" {
		t.Errorf("Command = %q, want empty string", input.Command)
	}
}

func TestParseInput_NilInput(t *testing.T) {
	t.Parallel()

	input := ParseInput(nil)

	if input.FilePath != "" {
		t.Errorf("FilePath = %q, want empty string", input.FilePath)
	}
	if input.Content != "" {
		t.Errorf("Content = %q, want empty string", input.Content)
	}
	if input.Command != "" {
		t.Errorf("Command = %q, want empty string", input.Command)
	}
}

func TestWriteDeny(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteDeny(&buf, "SELF-001", "cannot modify hook configuration")

	want := "qsdev-selfprotect: SELF-001 — cannot modify hook configuration\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteDeny output = %q, want %q", got, want)
	}
}

func TestWriteEvasionDeny(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteEvasionDeny(&buf, "RENAME", "attempted to rename protected file")

	want := "qsdev-selfprotect: EVASION-RENAME — attempted to rename protected file\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteEvasionDeny output = %q, want %q", got, want)
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteError(&buf, "failed to parse config")

	want := "qsdev-selfprotect: internal error: failed to parse config\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteError output = %q, want %q", got, want)
	}
}
