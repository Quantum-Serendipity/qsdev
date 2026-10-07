package denyutil

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrMalformedRule reports a permission rule that is neither a bare tool
	// name ("WebFetch", "mcp__server__*") nor a non-empty "Tool(pattern)".
	ErrMalformedRule = errors.New("malformed permission rule")

	// ErrLegacyPrefixSuffix reports a Bash or PowerShell rule whose pattern
	// ends in ":*".
	ErrLegacyPrefixSuffix = errors.New(`command rule ends in the legacy ":*" prefix marker`)
)

// Validate reports whether rule is a permission rule qsdev may generate.
//
// It rejects a rule that is not a bare tool name or a "Tool(pattern)" with a
// tool name and a non-empty pattern, and a Bash or PowerShell rule whose
// pattern ends in ":*". Claude Code recognises ":*" only at the end of a
// pattern and reads it as the legacy spelling of a trailing " *"
// (code.claude.com/docs/en/permissions, "Wildcard patterns"), so
// "Bash(deno npm:*)" means "Bash(deno npm *)" and never matches
// "deno npm:evil-cli". Write "<base> *" for a word prefix, or "<base>**"
// when the text after the colon is the payload.
func Validate(rule string) error {
	tool, pattern, hasArgs := strings.Cut(rule, "(")
	if !validToolName(tool) {
		return fmt.Errorf("%w %q: missing or invalid tool name", ErrMalformedRule, rule)
	}
	if !hasArgs {
		return nil
	}
	pattern, closed := strings.CutSuffix(pattern, ")")
	if !closed {
		return fmt.Errorf("%w %q: missing closing parenthesis", ErrMalformedRule, rule)
	}
	if pattern == "" {
		return fmt.Errorf("%w %q: empty pattern", ErrMalformedRule, rule)
	}
	if (tool == "Bash" || tool == "PowerShell") && strings.HasSuffix(pattern, ":*") {
		return fmt.Errorf(`%w: Claude Code reads %q as the word prefix %q; write that, or %q to match text after the colon`,
			ErrLegacyPrefixSuffix, rule,
			tool+"("+strings.TrimSuffix(pattern, ":*")+" *)",
			tool+"("+pattern+"*)")
	}
	return nil
}

// validToolName reports whether name is a tool name as rules spell it:
// letters, digits, '_' and '-', with an optional trailing '*' for MCP server
// wildcards such as "mcp__github__*".
func validToolName(name string) bool {
	name = strings.TrimSuffix(name, "*")
	if name == "" {
		return false
	}
	return !strings.ContainsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-'
	})
}
