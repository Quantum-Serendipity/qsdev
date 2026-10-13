package termutil

import (
	"strconv"
	"strings"
	"unicode"
)

// Safe renders text that comes from repository content (an .mcp.json server
// name, a program a hook runs, a pre-commit hook entry, a path inside the
// project) for the terminal. Text holding a control, format or other
// non-printable character (a carriage return or an ANSI escape sequence could
// rewrite or hide the lines around it, a bidi control could reorder it) is
// shown quoted with those characters escaped; other text is returned as is.
func Safe(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return strconv.Quote(s)
	}
	return s
}
