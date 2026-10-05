package cmdscan

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// midWordLiterals are characters syntax.Quote always quotes but a shell
// takes literally inside a word (after its first character), as a person
// writes them unquoted: `nixpkgs#jq`, `--arg=x`, `a~b`.
var midWordLiterals = strings.NewReplacer("#", "", "=", "", "~", "")

// QuoteWord returns s as one shell word, quoted with syntax.Quote when the
// shell would otherwise split or expand it, and unchanged when it needs no
// quoting, so a command rebuilt from its words reads as a person would write
// it and a deny rule written that way matches it. A string no shell word can
// hold (one with a NUL byte) is single-quoted as it stands.
func QuoteWord(s string) string {
	if len(s) > 1 {
		probe := s[:1] + midWordLiterals.Replace(s[1:])
		if q, err := syntax.Quote(probe, syntax.LangBash); err == nil && q == probe {
			return s
		}
	}
	q, err := syntax.Quote(s, syntax.LangBash)
	if err != nil {
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return q
}
