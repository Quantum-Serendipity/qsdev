package denyutil

import (
	"path"
	"strings"
)

// PathRuleContext holds the directories a Read or Edit rule anchors at.
// Paths may be POSIX or Windows ("C:\Users\a"); they are normalised the way
// the path being matched is.
type PathRuleContext struct {
	// Home is the user's home directory, the anchor of "~/" patterns.
	Home string
	// ProjectRoot is the directory of the settings source that defines the
	// rule (the primary working directory for project settings), the anchor
	// of "/" patterns.
	ProjectRoot string
	// Cwd is the current directory, the anchor of bare and "./" patterns.
	Cwd string
	// Deny is true for a deny or ask rule and false for an allow rule; it
	// decides how deep a single-segment directory pattern matches.
	Deny bool
}

// MatchesPathRule reports whether a "Read(...)" or "Edit(...)" permission
// rule matches the absolute file path p, under the semantics Claude Code
// documents (code.claude.com/docs/en/permissions, "Read and Edit"): rules
// "use gitignore pattern syntax with four distinct pattern types".
//
//   - "//path" is absolute from the filesystem root, "~/path" is relative to
//     the home directory, "/path" is relative to the settings source
//     (ctx.ProjectRoot; it is not the filesystem root), and "path" or "./path"
//     is relative to the current directory.
//   - "*" matches within a single path segment and "**" across directories;
//     a trailing "/**" matches everything inside the directory, not the
//     directory itself.
//   - "A rule only matches files under its anchor." A relative pattern
//     without an inner slash (a bare filename such as ".env" or "*.env")
//     matches at any depth below the current directory; one with an inner
//     slash ("src/components/**") matches only at the anchor.
//   - A single-segment directory pattern ("secrets/**") matches "a directory
//     named secrets at any depth" for a deny or ask rule, but only at the
//     anchor for an allow rule (ctx.Deny).
//   - "On Windows, paths are normalized to POSIX form before matching.
//     C:\Users\alice becomes /c/Users/alice."
//   - "A deny or ask rule whose path isn't usable as a gitignore pattern still
//     guards that exact path. An allow rule with an unusable pattern doesn't
//     approve anything."
//
// Out of scope, and reported as no match: "!" negation patterns, which only
// carve exceptions out of earlier rules from the same source; rules for any
// tool but Read and Edit (Claude Code never consults a path rule for Write,
// Glob or NotebookEdit); a bare tool name without a path; and a relative p.
// Matching is case-sensitive.
func MatchesPathRule(rule, p string, ctx PathRuleContext) bool {
	tool, pattern := ParseToolPattern(rule)
	if (tool != "Read" && tool != "Edit") || pattern == "" || strings.HasPrefix(pattern, "!") {
		return false
	}
	anchor, rest, anchored := anchorPathPattern(pattern, ctx)
	target := toPOSIXPath(p)
	if anchor == "" || !strings.HasPrefix(target, "/") {
		return false
	}
	if !validGlob(rest) {
		return ctx.Deny && target == path.Join(anchor, rest)
	}
	rel, ok := relativeTo(target, anchor)
	if !ok {
		return false
	}
	if strings.HasSuffix(rest, "/") {
		rest += "**"
	}
	if !anchored {
		rest = "**/" + rest
	}
	return matchSegments(strings.Split(rest, "/"), strings.Split(rel, "/"))
}

// anchorPathPattern splits a path pattern into its anchor directory (POSIX,
// cleaned) and the gitignore pattern below it, and reports whether that
// pattern is anchored there or may match at any depth.
func anchorPathPattern(pattern string, ctx PathRuleContext) (anchor, rest string, anchored bool) {
	if rest, ok := strings.CutPrefix(pattern, "//"); ok {
		return "/", rest, true
	}
	if rest, ok := strings.CutPrefix(pattern, "~/"); ok {
		return toPOSIXPath(ctx.Home), rest, true
	}
	if rest, ok := strings.CutPrefix(pattern, "/"); ok {
		return toPOSIXPath(ctx.ProjectRoot), rest, true
	}
	rest = strings.TrimPrefix(pattern, "./")
	anchored = strings.Contains(strings.TrimSuffix(rest, "/"), "/")
	if dir, ok := strings.CutSuffix(rest, "/**"); ok && ctx.Deny && dir != "**" && !strings.Contains(dir, "/") {
		anchored = false
	}
	return toPOSIXPath(ctx.Cwd), rest, anchored
}

// toPOSIXPath normalises p as Claude Code does before matching: backslashes
// become slashes and a drive prefix "C:" becomes "/c". The result is cleaned;
// an empty p stays empty.
func toPOSIXPath(p string) string {
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, `\`, "/")
	if len(p) >= 2 && p[1] == ':' && isASCIILetter(p[0]) {
		p = "/" + strings.ToLower(p[:1]) + "/" + p[2:]
	}
	return path.Clean(p)
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// relativeTo returns target relative to anchor when target lies strictly
// below it; both are cleaned POSIX paths.
func relativeTo(target, anchor string) (string, bool) {
	if anchor == "/" {
		return target[1:], target != "/"
	}
	return strings.CutPrefix(target, anchor+"/")
}

// validGlob reports whether every segment of pattern is a well-formed glob.
func validGlob(pattern string) bool {
	for _, seg := range strings.Split(pattern, "/") {
		if _, err := path.Match(seg, ""); err != nil {
			return false
		}
	}
	return true
}

// matchSegments matches path segments against gitignore pattern segments: a
// "**" segment matches zero or more segments, or one or more when it ends the
// pattern; any other segment is a single-segment glob.
func matchSegments(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		if len(pattern) == 1 {
			return len(name) > 0
		}
		for i := range len(name) + 1 {
			if matchSegments(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, _ := path.Match(pattern[0], name[0])
	return ok && matchSegments(pattern[1:], name[1:])
}
