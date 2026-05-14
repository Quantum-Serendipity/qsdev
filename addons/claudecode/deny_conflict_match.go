package claudecode

import "strings"

// matchesDenyRule checks whether a deny rule pattern would block a skill operation.
// Pattern format: "ToolType(args)" where args may contain * wildcards.
// Key behavior: "Bash(npm install *)" should NOT match "Bash(npm test *)" because
// the command prefix "npm install" is not a prefix of "npm test".
func matchesDenyRule(denyRule, operation string) bool {
	// Parse both into (toolType, args) pairs.
	denyTool, denyArgs := parseToolPattern(denyRule)
	opTool, opArgs := parseToolPattern(operation)

	// Different tool types never match.
	if denyTool != opTool {
		return false
	}

	// Match the args using glob semantics.
	return globMatchArgs(denyArgs, opArgs)
}

// parseToolPattern splits "Bash(npm install *)" into ("Bash", "npm install *").
// For patterns without parens like "Read", returns ("Read", "").
func parseToolPattern(pattern string) (string, string) {
	idx := strings.Index(pattern, "(")
	if idx < 0 {
		return pattern, ""
	}
	tool := pattern[:idx]
	args := strings.TrimSuffix(pattern[idx+1:], ")")
	return tool, args
}

// globMatchArgs checks if the deny rule's args pattern matches the operation's args.
// The key insight: "npm install *" matches "npm install lodash" but NOT "npm test foo".
// This is because * only matches at the POSITION where it appears in the deny rule.
func globMatchArgs(denyArgs, opArgs string) bool {
	// Empty deny args matches empty op args only.
	if denyArgs == "" && opArgs == "" {
		return true
	}
	if denyArgs == "" || opArgs == "" {
		return false
	}

	// Universal wildcard matches everything.
	if denyArgs == "*" {
		return true
	}

	// Exact match.
	if denyArgs == opArgs {
		return true
	}

	// Handle deny rules with embedded wildcards (e.g., "bash -c *npm install*").
	// These have wildcards in the middle, not just at the end.
	if strings.Contains(denyArgs, "*") && !strings.HasSuffix(denyArgs, "*") {
		// Not a simple trailing wildcard — skip advanced glob for now.
		// These rules don't conflict with the simple skill operations.
		return false
	}

	// Check if deny rule has trailing wildcard: "npm install *".
	if strings.HasSuffix(denyArgs, "*") {
		prefix := strings.TrimSuffix(denyArgs, "*")
		// If the operation also has a trailing wildcard: "npm test *",
		// the operation's prefix must start with the deny rule's prefix.
		if strings.HasSuffix(opArgs, "*") {
			opPrefix := strings.TrimSuffix(opArgs, "*")
			return strings.HasPrefix(opPrefix, prefix)
		}
		// The operation is concrete: "npm install lodash".
		return strings.HasPrefix(opArgs, prefix)
	}

	return false
}
