package ecosystem

import (
	"os"
	"regexp"
)

// FileDeclares reports whether the code of the source file at path matches
// token, for a module's Detect to tell whether a manifest written in a
// programming language (deps.edn, project.clj, mix.exs) declares something,
// without a parser for that language. Line comments, which start at the
// lineComment character (';' for Clojure and EDN, '#' for Elixir), and the
// contents of double-quoted strings, which may span lines, are blanked out
// first, so a commented-out declaration or a mention in a string does not
// count; a backslash escapes the character after it. A file that cannot be
// read declares nothing.
func FileDeclares(path string, lineComment byte, token *regexp.Regexp) bool {
	src, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return token.Match(codeOnly(src, lineComment))
}

// codeOnly returns src with its line comments and string contents replaced
// by spaces (see FileDeclares). Newlines and the quotes themselves are kept,
// so tokens on either side of a string or comment stay separated.
func codeOnly(src []byte, lineComment byte) []byte {
	out := make([]byte, len(src))
	inString, inComment := false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		out[i] = ' '
		switch {
		case c == '\n' || c == '\r':
			out[i] = c
			inComment = false
		case inComment:
		case c == '\\':
			// The escaped character is never a quote or comment start.
			if i+1 < len(src) && src[i+1] != '\n' && src[i+1] != '\r' {
				i++
				out[i] = ' '
			}
		case c == '"':
			out[i] = c
			inString = !inString
		case inString:
		case c == lineComment:
			inComment = true
		default:
			out[i] = c
		}
	}
	return out
}
