package hookio

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func mustParseInput(t *testing.T, toolName string, raw json.RawMessage) ToolInput {
	t.Helper()
	input, err := ParseInput(toolName, raw)
	if err != nil {
		t.Fatalf("ParseInput(%q, %s): %v", toolName, raw, err)
	}
	return input
}

// TestParseToolCall_SizeLimit verifies that input over MaxInputBytes is
// rejected with an explicit size error instead of being truncated into a
// confusing JSON syntax error, and that a Write well above the old 1 MiB cap
// still parses.
func TestParseToolCall_SizeLimit(t *testing.T) {
	t.Parallel()

	envelope := func(contentLen int) string {
		return `{"tool_name":"Write","tool_input":{"file_path":"f","content":"` +
			strings.Repeat("a", contentLen) + `"}}`
	}
	overhead := len(envelope(0))

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"1.5 MiB write", envelope(3 << 19), nil},
		{"exactly at the limit", envelope(MaxInputBytes - overhead), nil},
		{"one byte over the limit", envelope(MaxInputBytes - overhead + 1), ErrInputTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			call, err := ParseToolCall(context.Background(), strings.NewReader(tt.input))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseToolCall error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && call.ToolName != "Write" {
				t.Errorf("ToolName = %q, want Write", call.ToolName)
			}
		})
	}
}

// TestParseToolCall_TimeoutWhileReading verifies the context bounds the read
// itself: a stdin that never closes must not hang the hook past its deadline.
func TestParseToolCall_TimeoutWhileReading(t *testing.T) {
	t.Parallel()

	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := ParseToolCall(ctx, pr)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ParseToolCall error = %v, want %v", err, context.DeadlineExceeded)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("ParseToolCall returned after %v; the read ignored the deadline", elapsed)
	}
}

// TestParseInput_MalformedFieldsFailClosed verifies that a tool_input that is
// not an object, or an evaluated field with the wrong type, is an error rather
// than a silently empty field that every rule would allow.
func TestParseInput_MalformedFieldsFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		raw     string
		wantErr bool
	}{
		{"not an object", "Write", `"x"`, true},
		{"array", "Bash", `["rm","-rf"]`, true},
		{"file_path not a string", "Write", `{"file_path":["a"],"content":"x"}`, true},
		{"mcp file_path not a string", "mcp__fs__write", `{"file_path":{"p":"a"}}`, true},
		{"notebook_path not a string", "NotebookEdit", `{"notebook_path":1}`, true},
		{"bash command not a string", "Bash", `{"command":["rm","x"]}`, true},
		{"edit new_string not a string", "Edit", `{"file_path":"f","old_string":"a","new_string":1}`, true},
		{"multiedit edits not an array", "MultiEdit", `{"file_path":"f","edits":"x"}`, true},
		{"write content not a string", "Write", `{"file_path":"f","content":{"a":1}}`, true},

		{"mcp tool with array content", "mcp__x__send", `{"content":[{"type":"text"}],"file_path":"f"}`, false},
		{"mcp tool with array command", "mcp__docker__run", `{"command":["ls","-l"]}`, false},
		{"null input", "Write", `null`, false},
		{"empty input", "Write", ``, false},
		{"empty object", "Read", `{}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseInput(tt.tool, json.RawMessage(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInput(%q, %s) error = %v, want error %v", tt.tool, tt.raw, err, tt.wantErr)
			}
		})
	}

	t.Run("mcp tool keeps its path", func(t *testing.T) {
		t.Parallel()
		in := mustParseInput(t, "mcp__x__send", json.RawMessage(`{"content":[1],"file_path":"/p"}`))
		if in.FilePath != "/p" {
			t.Errorf("FilePath = %q, want /p", in.FilePath)
		}
	})
}

func TestParseInput_NotebookPath(t *testing.T) {
	t.Parallel()

	in := mustParseInput(t, "NotebookEdit", json.RawMessage(`{"notebook_path":"/p/.claude/hooks/x.ipynb","new_source":"x"}`))
	if in.FilePath != "/p/.claude/hooks/x.ipynb" {
		t.Errorf("FilePath = %q, want the notebook_path", in.FilePath)
	}
	both := mustParseInput(t, "X", json.RawMessage(`{"file_path":"a","notebook_path":"b"}`))
	if both.FilePath != "a" {
		t.Errorf("FilePath = %q, want file_path to win", both.FilePath)
	}
}

// TestResultingContent verifies the post-call file content, including edits
// that only delete text — which EditedContent reports as empty.
func TestResultingContent(t *testing.T) {
	t.Parallel()

	const npmrc = "registry=https://r\nignore-scripts=true\n"
	tests := []struct {
		name    string
		tool    string
		in      ToolInput
		current string
		want    string
		wantOK  bool
	}{
		{"write replaces", "Write", ToolInput{Content: "x=1\n"}, npmrc, "x=1\n", true},
		{"empty write", "Write", ToolInput{}, npmrc, "", true},
		{"edit deletes a line", "Edit", ToolInput{OldString: "ignore-scripts=true\n"}, npmrc, "registry=https://r\n", true},
		{"edit creates a file", "Edit", ToolInput{NewString: "a"}, "", "a", true},
		{"edit replace_all", "Edit", ToolInput{OldString: "a", NewString: "b", ReplaceAll: true}, "a a", "b b", true},
		{"edit not found", "Edit", ToolInput{OldString: "zzz"}, npmrc, "", false},
		{"edit ambiguous", "Edit", ToolInput{OldString: "a"}, "a a", "", false},
		{"edit with empty old on existing file", "Edit", ToolInput{NewString: "a"}, npmrc, "", false},
		{"multiedit applies in order", "MultiEdit", ToolInput{Edits: []EditOp{
			{OldString: "registry=https://r\n", NewString: ""},
			{OldString: "true", NewString: "false"},
		}}, npmrc, "ignore-scripts=false\n", true},
		{"multiedit fails as a whole", "MultiEdit", ToolInput{Edits: []EditOp{
			{OldString: "registry", NewString: "x"},
			{OldString: "missing", NewString: ""},
		}}, npmrc, "", false},
		{"other tool", "Bash", ToolInput{Command: "true"}, npmrc, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.in.ResultingContent(tt.tool, tt.current)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ResultingContent(%q) = (%q, %v), want (%q, %v)", tt.tool, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
