package logging

import "strings"

// jsonValueSpan locates the value of a JSON member whose sensitive key has just
// been matched, starting at valStart (the value's first character). depth is
// the JSON escape depth of the surrounding document: 0 for plain JSON, 1 for
// JSON embedded in a JSON string (quotes appear as \"). It returns the byte
// range to replace and the replacement:
//
//   - a string value keeps its quotes and has only its contents replaced, so
//     {"password":"x"} becomes {"password":"[REDACTED]"};
//   - an object, array or bare scalar is replaced whole by a quoted marker, so
//     the document stays well-formed and later members are left intact.
//
// An unterminated value is redacted to the end of its line: fail closed.
func jsonValueSpan(s string, valStart, depth int) (from, to int, replacement string) {
	quote := `"`
	if depth == 1 {
		quote = `\"`
	}
	switch {
	case strings.HasPrefix(s[valStart:], quote):
		from = valStart + len(quote)
		return from, jsonStringEnd(s, from, depth), redacted
	case s[valStart] == '{' || s[valStart] == '[':
		return valStart, jsonCompositeEnd(s, valStart, depth), quote + redacted + quote
	default:
		end := lineEnd(s, valStart)
		if i := strings.IndexAny(s[valStart:end], ",}]"); i >= 0 {
			end = valStart + i
		}
		end = valStart + len(strings.TrimRight(s[valStart:end], nameValueTrimCutset))
		if depth == 1 {
			// A bare scalar inside an escaped document ends before the
			// backslash of any escape sequence that follows it.
			if i := strings.IndexByte(s[valStart:end], '\\'); i >= 0 {
				end = valStart + i
			}
		}
		return valStart, end, quote + redacted + quote
	}
}

// jsonStringEnd returns the offset at which the contents of a JSON string
// starting at from end — the first byte of its closing delimiter, which at
// depth 1 is the backslash of \" — or the end of the line when the string is
// unterminated.
func jsonStringEnd(s string, from, depth int) int {
	end := lineEnd(s, from)
	for i := from; i < end; i++ {
		if isQuoteDelim(s, i, depth) {
			return i - depth
		}
	}
	return end
}

// jsonCompositeEnd returns the offset just past the bracket that closes the
// object or array opening at start, skipping brackets inside strings, or the
// end of the line when it is not closed on that line.
func jsonCompositeEnd(s string, start, depth int) int {
	end := lineEnd(s, start)
	level := 0
	inString := false
	for i := start; i < end; i++ {
		c := s[i]
		if c == '"' && isQuoteDelim(s, i, depth) {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{', '[':
			level++
		case '}', ']':
			level--
			if level == 0 {
				return i + 1
			}
		}
	}
	return end
}

// isQuoteDelim reports whether the '"' at s[i] delimits a string at the given
// escape depth rather than being an escaped quote inside one. At depth 0 a
// delimiter is preceded by an even number of backslashes; at depth 1 (JSON
// inside a JSON string) it is preceded by 1, 5, 9, … backslashes — one to
// escape the quote itself plus pairs encoding literal backslashes — while 3,
// 7, … backslashes encode a quote escaped within the inner string.
func isQuoteDelim(s string, i, depth int) bool {
	if s[i] != '"' {
		return false
	}
	n := 0
	for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
		n++
	}
	unit := 1 << depth
	return n%(2*unit) == unit-1
}
