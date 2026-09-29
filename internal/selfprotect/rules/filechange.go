package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// TextEdit is one old->new string replacement of an Edit/MultiEdit tool call.
type TextEdit struct {
	OldString  string
	NewString  string
	ReplaceAll bool
}

// FileChange returns the content of a Write/Edit/MultiEdit target before the
// tool call (read from disk; empty when the file does not exist yet) and the
// content it will have afterwards: the Write content, or the current content
// with the Edits applied in order. Rules that guard a file's invariants
// compare the two, since an Edit's new_string alone says nothing about what
// the file loses. An edit whose old_string is absent is skipped, as Claude
// Code rejects it without writing. An error means the change cannot be
// reconstructed; callers must fail closed.
func (ctx *EvalContext) FileChange() (before, after string, err error) {
	target := ctx.CanonicalPath
	if target == "" {
		target = ctx.FilePath
	}
	if target == "" {
		return "", "", errors.New("tool call has no target file")
	}
	data, err := os.ReadFile(target)
	switch {
	case err == nil:
		before = string(data)
	case errors.Is(err, fs.ErrNotExist):
	default:
		return "", "", fmt.Errorf("reading %s: %w", target, err)
	}

	switch ctx.ToolName {
	case "Write":
		return before, ctx.Content, nil
	case "Edit", "MultiEdit":
		if len(ctx.Edits) == 0 {
			return "", "", errors.New("edit replacements are unavailable")
		}
		after = before
		for _, e := range ctx.Edits {
			after = applyTextEdit(after, e)
		}
		return before, after, nil
	default:
		return "", "", fmt.Errorf("%s does not write a file", ctx.ToolName)
	}
}

// applyTextEdit applies one Edit replacement the way Claude Code does: the
// first occurrence, or every occurrence with ReplaceAll. An empty OldString on
// an empty file creates it with NewString.
func applyTextEdit(s string, e TextEdit) string {
	if e.OldString == "" {
		if s == "" {
			return e.NewString
		}
		return s
	}
	if e.ReplaceAll {
		return strings.ReplaceAll(s, e.OldString, e.NewString)
	}
	return strings.Replace(s, e.OldString, e.NewString, 1)
}
