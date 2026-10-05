package ecosystem

import "regexp"

// tokenRe matches a single bare word: letters, digits, '.', '_' and '-',
// starting with a letter or digit. It excludes every character that can end
// or open a Nix expression when emitted unquoted, and the mirrorOf operators
// ',', '!' and '*'.
var tokenRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// maxTokenLen keeps a pathological value out of generated files; it sits well
// above any real token.
const maxTokenLen = 64

// IsValidToken reports whether s is a single bare word of letters, digits,
// '.', '_' and '-' (starting with a letter or digit), such as a service
// version ("16"), a package manager ("pnpm") or a repository id. Generators
// use it as a defense-in-depth guard for values they splice unquoted into
// generated files; internal/validation.IsValidToken delegates here.
func IsValidToken(s string) bool {
	return len(s) <= maxTokenLen && tokenRe.MatchString(s)
}
