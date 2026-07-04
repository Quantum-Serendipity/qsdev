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
// Wildcards (*) match any substring. Multiple wildcards are supported:
// prefix ("foo *"), suffix ("* bar"), embedded ("foo * bar"), and
// combinations ("*foo*bar*") all work as expected.
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

	if !strings.Contains(denyArgs, "*") {
		return false
	}

	// When the operation itself contains wildcards, compare the deny
	// pattern's prefix against the operation's prefix so that a broader
	// operation (e.g. "git push *") does not match a narrower deny rule
	// (e.g. "git push --force *").
	if strings.Contains(opArgs, "*") {
		denyPrefix, _ := strings.CutSuffix(denyArgs, "*")
		opPrefix, _ := strings.CutSuffix(opArgs, "*")
		return strings.HasPrefix(opPrefix, denyPrefix)
	}

	// A deny pattern ending in whitespace followed by a terminal '*'
	// (e.g. "aws sts assume-role *") must act as a token-boundary prefix: it
	// has to deny the bare subcommand ("aws sts assume-role"), its hyphenated
	// continuations ("aws sts assume-role-with-web-identity"), and its
	// space-separated continuations ("aws sts assume-role --role-arn x")
	// alike. Dropping the whitespace before the terminal '*' makes the
	// separating space optional while the '*' still matches any suffix, so the
	// prefix matches at a token boundary. This intentionally broadens denial —
	// the safe direction for the credential-emitting subcommands these rules
	// guard. Only the terminal whitespace-'*' case is touched; embedded
	// wildcards (e.g. "git * --force") keep their semantics.
	if strings.HasSuffix(denyArgs, "*") {
		denyArgs = strings.TrimRight(denyArgs[:len(denyArgs)-1], " \t") + "*"
	}

	segments := strings.Split(denyArgs, "*")

	pos := 0
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		idx := strings.Index(opArgs[pos:], seg)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			return false
		}
		pos += idx + len(seg)
	}
	if !strings.HasSuffix(denyArgs, "*") {
		return pos == len(opArgs)
	}
	return true
}
