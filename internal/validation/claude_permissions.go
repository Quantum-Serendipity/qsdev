package validation

import (
	"errors"
	"strings"
	"unicode"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ErrPermissionRule reports a claude_code.permissions entry that is not a
// Claude Code permission rule.
var ErrPermissionRule = errors.New("not a Claude Code permission rule: use Tool or Tool(specifier), e.g. Bash(make *)")

// CheckPermissionRule returns ErrPermissionRule unless rule is a bare tool
// name pattern ("Read", "mcp__context7__*") or a tool name followed by a
// non-empty parenthesised specifier ("Bash(make *)"), with no control or
// invisible formatting character, or nil when it is.
func CheckPermissionRule(rule string) error {
	tool, args := denyutil.ParseToolPattern(rule)
	if !IsValidToolNamePattern(tool) {
		return ErrPermissionRule
	}
	if tool == rule {
		return nil
	}
	if args == "" || tool+"("+args+")" != rule {
		return ErrPermissionRule
	}
	if strings.ContainsFunc(args, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) {
		return ErrPermissionRule
	}
	return nil
}

// CheckClaudePermissions returns every invalid entry of the
// claude_code.permissions block p (see CheckPermissionRule), allow entries
// first. It is shared by .qsdev.yaml parsing and settings generation, so
// they agree on what reaches settings.json.
func CheckClaudePermissions(p types.ClaudePermissionsConfig) []PolicyEntryError {
	var errs []PolicyEntryError
	errs = checkEntries(errs, "claude_code.permissions.allow", p.Allow, CheckPermissionRule)
	errs = checkEntries(errs, "claude_code.permissions.deny", p.Deny, CheckPermissionRule)
	return errs
}
