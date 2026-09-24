package vulnscan

import "strings"

// compareVersions orders two version strings, returning -1, 0 or +1. OSV
// records use each ecosystem's native scheme (SemVer for Go/npm/crates.io, PEP
// 440 for PyPI), so this is a best-effort ordering that covers the shapes those
// schemes share rather than a full implementation of any one of them: a leading
// "v" and any "+build" suffix are ignored, the rest is split into numeric and
// alphabetic tokens, numeric tokens compare numerically, and a version that ends
// where the other continues with an alphabetic token (a pre-release such as
// "-rc1" or "a1") sorts before it. It is used only to pick remediation advice,
// never to decide whether a package is vulnerable.
func compareVersions(a, b string) int {
	ta, tb := versionTokens(a), versionTokens(b)
	for i := 0; i < len(ta) || i < len(tb); i++ {
		switch {
		case i >= len(ta):
			return -extraTokenOrder(tb[i])
		case i >= len(tb):
			return extraTokenOrder(ta[i])
		}
		if c := compareVersionTokens(ta[i], tb[i]); c != 0 {
			return c
		}
	}
	return 0
}

// extraTokenOrder reports how a version with one more token compares to the
// version it extends: a numeric token (1.0 -> 1.0.1) or a PEP 440 post-release
// (1.0 -> 1.0.post1) makes it greater, any other alphabetic one (1.0 -> 1.0-rc1)
// marks a pre-release and makes it smaller.
func extraTokenOrder(tok string) int {
	if isNumericToken(tok) || strings.EqualFold(tok, "post") {
		return 1
	}
	return -1
}

// compareVersionTokens compares two tokens: numerically when both are numeric,
// lexically when both are alphabetic, and with a numeric token ranking above an
// alphabetic one (1.0.0 > 1.0.rc).
func compareVersionTokens(x, y string) int {
	xn, yn := isNumericToken(x), isNumericToken(y)
	switch {
	case xn && yn:
		x, y = strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
		if len(x) != len(y) {
			if len(x) < len(y) {
				return -1
			}
			return 1
		}
		return strings.Compare(x, y)
	case xn:
		return 1
	case yn:
		return -1
	default:
		return strings.Compare(strings.ToLower(x), strings.ToLower(y))
	}
}

// versionTokens splits a version into maximal runs of digits and of letters,
// dropping separators, a leading "v" and any "+build" metadata.
func versionTokens(v string) []string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	var toks []string
	start := -1
	digits := false
	for i := 0; i <= len(v); i++ {
		var c byte
		if i < len(v) {
			c = v[i]
		}
		isDigit := c >= '0' && c <= '9'
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if start >= 0 && (i == len(v) || (!isDigit && !isLetter) || isDigit != digits) {
			toks = append(toks, v[start:i])
			start = -1
		}
		if start < 0 && (isDigit || isLetter) {
			start, digits = i, isDigit
		}
	}
	return toks
}

func isNumericToken(tok string) bool {
	return tok != "" && tok[0] >= '0' && tok[0] <= '9'
}
