package pathmatch

import (
	"path"
	"strings"
)

// posixClasses maps each POSIX character class a shell glob may name inside a
// bracket expression (`[[:alpha:]]`) to the equivalent path.Match class body
// for the ASCII range. Protected names are ASCII, so these sets decide every
// match the guards make.
var posixClasses = map[string]string{
	"alnum":  `a-zA-Z0-9`,
	"alpha":  `a-zA-Z`,
	"blank":  " \t",
	"cntrl":  "\x00-\x1f\x7f",
	"digit":  `0-9`,
	"graph":  `!-~`,
	"lower":  `a-z`,
	"print":  ` -~`,
	"punct":  `!-/:-@\[-` + "`" + `{-~`,
	"space":  " \t\n\v\f\r",
	"upper":  `A-Z`,
	"word":   `a-zA-Z0-9_`,
	"xdigit": `0-9a-fA-F`,
}

// ShellPattern rewrites a POSIX shell glob into the syntax path.Match reads.
// The shell negates a bracket expression with a leading '!' as well as '^',
// takes a ']' right after the opening '[' (or "[!") as a literal, names
// character classes as `[:alpha:]`, and reads a '[' with no closing ']' as a
// literal; path.Match does none of these. ok is false when the glob holds a
// construct this rewrite cannot express (an unknown class, a collating
// element longer than one byte), so a caller can fail closed.
func ShellPattern(glob string) (pattern string, ok bool) {
	if !strings.Contains(glob, "[") {
		return glob, true
	}
	var b strings.Builder
	b.Grow(len(glob) + 8)
	for i := 0; i < len(glob); i++ {
		switch c := glob[i]; c {
		case '\\':
			if i+1 < len(glob) {
				b.WriteString(glob[i : i+2])
				i++
			} else {
				b.WriteString(`\\`) // a trailing backslash is literal
			}
		case '[':
			class, n, ok := shellBracket(glob[i:])
			if !ok {
				return "", false
			}
			if n == 0 {
				b.WriteString(`\[`) // unterminated: the shell reads '[' literally
				continue
			}
			b.WriteString(class)
			i += n - 1
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// ShellMatch reports whether name matches the shell glob pattern, read with
// the shell's bracket-expression rules (see ShellPattern). It returns
// path.ErrBadPattern for a pattern it cannot read; callers that guard a
// protected path treat that as a match.
func ShellMatch(glob, name string) (bool, error) {
	pattern, ok := ShellPattern(glob)
	if !ok {
		return false, path.ErrBadPattern
	}
	return path.Match(pattern, name)
}

// shellBracket rewrites the bracket expression at the start of s (s[0] is
// '[') into a path.Match class. n is the number of bytes of s it spans, or 0
// when s has no closing ']' and the '[' is a literal.
func shellBracket(s string) (class string, n int, ok bool) {
	var body strings.Builder
	j := 1
	negate := j < len(s) && (s[j] == '!' || s[j] == '^')
	if negate {
		j++
	}
	start := j
	afterClass := false // the previous item was a [:class:], which ends no range
	for ; j < len(s); j++ {
		c := s[j]
		switch {
		case c == ']' && j > start:
			prefix := "["
			if negate {
				prefix = "[^"
			}
			return prefix + body.String() + "]", j + 1, true
		case c == '[' && j+1 < len(s) && strings.IndexByte(":=.", s[j+1]) >= 0:
			end := strings.Index(s[j+2:], string(s[j+1])+"]")
			if end < 0 {
				writeClassLiteral(&body, c)
				afterClass = false
				continue
			}
			name := s[j+2 : j+2+end]
			if s[j+1] == ':' {
				set, known := posixClasses[name]
				if !known {
					return "", 0, false
				}
				body.WriteString(set)
				afterClass = true
			} else {
				if len(name) != 1 {
					return "", 0, false
				}
				writeClassLiteral(&body, name[0])
				afterClass = false
			}
			j += 2 + end + 1 // the class's closing ']'
			continue
		case c == '\\' && j+1 < len(s):
			j++
			writeClassLiteral(&body, s[j])
		case c == '-' && j > start && !afterClass && j+1 < len(s) && s[j+1] != ']':
			body.WriteByte('-') // a range operator
		default:
			writeClassLiteral(&body, c)
		}
		afterClass = false
	}
	return "", 0, true
}

// writeClassLiteral writes c into a path.Match class body as a literal.
func writeClassLiteral(b *strings.Builder, c byte) {
	if strings.IndexByte(`\-]^[`, c) >= 0 {
		b.WriteByte('\\')
	}
	b.WriteByte(c)
}
