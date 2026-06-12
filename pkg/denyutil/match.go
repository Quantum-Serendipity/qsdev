package denyutil

import "strings"

// MatchesDenyRule checks whether a deny rule pattern would block a tool operation.
func MatchesDenyRule(denyRule, operation string) bool {
	denyTool, denyArgs := ParseToolPattern(denyRule)
	opTool, opArgs := ParseToolPattern(operation)

	if denyTool != opTool {
		return false
	}

	return GlobMatchArgs(denyArgs, opArgs)
}

// ParseToolPattern splits "Bash(npm install *)" into ("Bash", "npm install *").
func ParseToolPattern(pattern string) (string, string) {
	tool, args, found := strings.Cut(pattern, "(")
	if !found {
		return pattern, ""
	}
	args, _ = strings.CutSuffix(args, ")")
	return tool, args
}

// GlobMatchArgs checks if the deny rule's args pattern matches the operation's args.
func GlobMatchArgs(denyArgs, opArgs string) bool {
	if denyArgs == "" && opArgs == "" {
		return true
	}
	if denyArgs == "" || opArgs == "" {
		return false
	}

	if denyArgs == "*" {
		return true
	}

	if denyArgs == opArgs {
		return true
	}

	if strings.Contains(denyArgs, "*") && !strings.HasSuffix(denyArgs, "*") {
		return false
	}

	if prefix, ok := strings.CutSuffix(denyArgs, "*"); ok {
		if opPrefix, hasWild := strings.CutSuffix(opArgs, "*"); hasWild {
			return strings.HasPrefix(opPrefix, prefix)
		}
		return strings.HasPrefix(opArgs, prefix)
	}

	return false
}
