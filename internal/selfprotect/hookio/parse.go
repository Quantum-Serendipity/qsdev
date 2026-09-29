package hookio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// MaxInputBytes caps a hook's stdin envelope to prevent OOM. Claude Code puts
// a Write's whole file content in the envelope, so the cap sits well above any
// realistic source or fixture file rather than at a size a legitimate Write
// can reach.
const MaxInputBytes = 16 << 20

// ErrInputTooLarge reports a hook envelope larger than MaxInputBytes. Input is
// rejected with this error rather than truncated, which would otherwise surface
// as a confusing JSON syntax error.
var ErrInputTooLarge = fmt.Errorf("hook input exceeds %d bytes", MaxInputBytes)

// ToolCall represents the JSON envelope received from Claude Code's hook system.
//
// CWD is the session's working directory when the tool call was made. Claude
// Code's Bash tool keeps its working directory across calls, so a `cd` in one
// call changes where the next call's relative paths land; rules resolve
// relative paths against this rather than the hook process's own directory.
type ToolCall struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	CWD       string          `json:"cwd"`
}

// ToolInput represents the parsed tool_input fields relevant to self-protection.
//
// Write carries the introduced text in Content; Edit uses OldString/NewString;
// MultiEdit uses Edits. Content-scanning rules must read the introduced text via
// EditedContent so an Edit/MultiEdit (which has no `content` field) is inspected
// too — otherwise injected content slips past those rules.
type ToolInput struct {
	// FilePath is the tool's target path: `file_path`, or NotebookEdit's
	// `notebook_path` when there is no `file_path`.
	FilePath   string   `json:"file_path"`
	Content    string   `json:"content"`     // Write
	Command    string   `json:"command"`     // Bash
	OldString  string   `json:"old_string"`  // Edit
	NewString  string   `json:"new_string"`  // Edit
	ReplaceAll bool     `json:"replace_all"` // Edit
	Edits      []EditOp `json:"edits"`       // MultiEdit
}

// EditOp is one edit within a MultiEdit tool call.
type EditOp struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
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

// ResultingContent returns the whole content the target file will have once
// this Write, Edit or MultiEdit succeeds, given its current content ("" when
// it does not exist). Unlike EditedContent it also reflects what the call
// removes, so an Edit whose new_string is empty is visible as a deletion.
//
// ok is false when the result cannot be determined exactly: another tool, or
// an edit whose old_string does not occur exactly once (without replace_all)
// in the text it applies to. Claude Code rejects such an edit or matches it
// only after normalizing the text, so a caller guarding the file's content
// must fail closed rather than assume the file is unchanged.
func (i ToolInput) ResultingContent(toolName, current string) (result string, ok bool) {
	switch toolName {
	case "Write":
		return i.Content, true
	case "Edit":
		return applyEdit(current, EditOp{OldString: i.OldString, NewString: i.NewString, ReplaceAll: i.ReplaceAll})
	case "MultiEdit":
		result = current
		for _, e := range i.Edits {
			if result, ok = applyEdit(result, e); !ok {
				return "", false
			}
		}
		return result, true
	default:
		return "", false
	}
}

// applyEdit applies one old_string -> new_string replacement the way the Edit
// tool does: an empty old_string creates the file, replace_all replaces every
// occurrence, and otherwise old_string must occur exactly once.
func applyEdit(current string, e EditOp) (string, bool) {
	if e.OldString == "" {
		if current != "" {
			return "", false
		}
		return e.NewString, true
	}
	n := strings.Count(current, e.OldString)
	switch {
	case n == 0:
		return "", false
	case e.ReplaceAll:
		return strings.ReplaceAll(current, e.OldString, e.NewString), true
	case n == 1:
		return strings.Replace(current, e.OldString, e.NewString, 1), true
	default:
		return "", false
	}
}

// ReadInput reads a hook's stdin envelope. It fails with ErrInputTooLarge
// instead of truncating input over MaxInputBytes, and returns as soon as ctx
// is done even while the read is still blocked. (The abandoned read ends when
// the hook process exits.)
func ReadInput(ctx context.Context, r io.Reader) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("reading hook input: %w", err)
	}

	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		// Read one byte past the cap so an oversized envelope is detected.
		data, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
		done <- result{data, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("reading hook input: %w", ctx.Err())
	case res := <-done:
		if res.err != nil {
			return nil, fmt.Errorf("reading hook input: %w", res.err)
		}
		if len(res.data) > MaxInputBytes {
			return nil, ErrInputTooLarge
		}
		return res.data, nil
	}
}

// ParseToolCall reads and parses a PreToolUse JSON envelope from the reader.
// Returns an error if the input is too large, the JSON is malformed, the
// reader fails, or ctx ends before the input has been read.
func ParseToolCall(ctx context.Context, r io.Reader) (*ToolCall, error) {
	data, err := ReadInput(ctx, r)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.New("empty hook input")
	}

	var call ToolCall
	if err := json.Unmarshal(data, &call); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}

	if call.ToolName == "" {
		return nil, errors.New("missing tool_name in hook input")
	}

	return &call, nil
}

// ParseInput extracts the fields self-protection evaluates from toolName's
// tool_input. A missing field is simply empty. It returns an error — which the
// hook must treat as a deny — when tool_input is not a JSON object, or when a
// field that is evaluated for toolName has the wrong type: a path field for
// any tool, `command` for Bash, and the content fields for Write, Edit and
// MultiEdit. Other fields of other tools (an MCP tool's array-valued
// `content`, say) are not inspected, so their types do not matter.
func ParseInput(toolName string, raw json.RawMessage) (ToolInput, error) {
	var input ToolInput
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return input, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return ToolInput{}, fmt.Errorf("parsing tool_input: %w", err)
	}

	var notebookPath string
	targets := []struct {
		name string
		dst  any
	}{
		{"file_path", &input.FilePath},
		{"notebook_path", &notebookPath},
		{"command", &input.Command},
		{"content", &input.Content},
		{"old_string", &input.OldString},
		{"new_string", &input.NewString},
		{"replace_all", &input.ReplaceAll},
		{"edits", &input.Edits},
	}
	for _, t := range targets {
		value, ok := fields[t.name]
		if !ok {
			continue
		}
		if err := json.Unmarshal(value, t.dst); err != nil && fieldEvaluated(toolName, t.name) {
			return ToolInput{}, fmt.Errorf("parsing tool_input.%s: %w", t.name, err)
		}
	}
	if input.FilePath == "" {
		input.FilePath = notebookPath
	}
	return input, nil
}

// fieldEvaluated reports whether self-protection evaluates the tool_input
// field name for toolName, so a malformed value must fail closed.
func fieldEvaluated(toolName, name string) bool {
	switch name {
	case "file_path", "notebook_path":
		return true
	case "command":
		return cmdscan.IsShellTool(toolName)
	default:
		return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
	}
}
