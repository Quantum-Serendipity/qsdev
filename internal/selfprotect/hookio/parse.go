package hookio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ToolCall represents the JSON envelope received from Claude Code's hook system.
type ToolCall struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// ToolInput represents the parsed tool_input fields relevant to self-protection.
//
// Write carries the introduced text in Content; Edit uses OldString/NewString;
// MultiEdit uses Edits. Content-scanning rules must read the introduced text via
// EditedContent so an Edit/MultiEdit (which has no `content` field) is inspected
// too — otherwise injected content slips past those rules.
type ToolInput struct {
	FilePath  string   `json:"file_path"`
	Content   string   `json:"content"`    // Write
	Command   string   `json:"command"`    // Bash
	OldString string   `json:"old_string"` // Edit
	NewString string   `json:"new_string"` // Edit
	Edits     []EditOp `json:"edits"`      // MultiEdit
}

// EditOp is one edit within a MultiEdit tool call.
type EditOp struct {
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// EditedContent returns the text a Write/Edit/MultiEdit introduces: Content for
// Write, else the newly-written string(s) for Edit/MultiEdit. Content-scanning
// self-protection rules must inspect this rather than Content alone, since
// Edit/MultiEdit tool calls never populate `content`.
func (i ToolInput) EditedContent() string {
	if i.Content != "" {
		return i.Content
	}
	if i.NewString != "" {
		return i.NewString
	}
	if len(i.Edits) > 0 {
		var b strings.Builder
		for _, e := range i.Edits {
			b.WriteString(e.NewString)
			b.WriteByte('\n')
		}
		return b.String()
	}
	return ""
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
