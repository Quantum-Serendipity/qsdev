package hookio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// ToolCall represents the JSON envelope received from Claude Code's hook system.
type ToolCall struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// ToolInput represents the parsed tool_input fields relevant to self-protection.
type ToolInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	Command  string `json:"command"`
}

// ParseToolCall reads and parses a PreToolUse JSON envelope from the reader.
// Returns an error if the JSON is malformed or the reader fails.
func ParseToolCall(ctx context.Context, r io.Reader) (*ToolCall, error) {
	// Use a LimitReader to cap input at 1MB to prevent OOM.
	limited := io.LimitReader(r, 1<<20)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading hook input: %w", err)
	}

	// Check for context cancellation between read and parse.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty hook input")
	}

	var call ToolCall
	if err := json.Unmarshal(data, &call); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}

	if call.ToolName == "" {
		return nil, fmt.Errorf("missing tool_name in hook input")
	}

	return &call, nil
}

// ParseInput extracts the file_path, content, and command fields from tool_input.
func ParseInput(raw json.RawMessage) ToolInput {
	var input ToolInput
	// Ignore errors — missing fields are simply empty strings.
	_ = json.Unmarshal(raw, &input)
	return input
}
