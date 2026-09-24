package validation

import (
	"errors"
	"regexp"
)

// Syntax validators for free-form answer values that generators splice into
// devenv.nix and other generated files. These values reach generation from
// team-shared inputs (the committed .qsdev.yaml via join, --answers-file), so
// they are an injection boundary: several are emitted unquoted (e.g.
// `pkgs.<package>`, `pkgs.postgresql_<version>`, `<ENV_KEY> = ...;`), where
// string escaping cannot help. Each validator is an allow-list that excludes
// every Nix token able to end an expression or open a new one (`;`, braces,
// brackets, parentheses, quotes, `$`, `/`, `:`, `#`, whitespace, ...).
var (
	envKeyRe      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	nixAttrPathRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_'-]*(\.[A-Za-z_][A-Za-z0-9_'-]*)*$`)
	tokenRe       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	// Language versions may be constraints or aliases detected from project
	// manifests (package.json engines ">=18 <21", "18.x || 20.x", .nvmrc
	// "lts/iron"), so comparison and range operators, '/' and spaces are
	// allowed on top of the token set. Generators only ever emit a language
	// version inside a Nix string (or map it to a fixed attribute), where none
	// of these characters is special; quotes, '\' and '$' stay excluded.
	versionConstraintRe = regexp.MustCompile(`^[A-Za-z0-9._+*^~<>=!|,/ -]+$`)
	// Claude Code tool names are letters, digits, '_' and '-' (MCP tools are
	// mcp__<server>__<tool> with both parts normalized to that set); a
	// tool-gates entry may add '*' wildcards. Commas, which separate the
	// entries handed to the hook, and every other character are excluded.
	toolNamePatternRe = regexp.MustCompile(`^[A-Za-z0-9_*-]+$`)
)

// Length caps keep a pathological value from being written into generated
// files; they sit well above any real identifier or version.
const (
	maxEnvKeyLen            = 256
	maxNixAttrPathLen       = 256
	maxTokenLen             = 64
	maxVersionConstraintLen = 128
	maxToolNamePatternLen   = 256
)

// IsValidEnvKey reports whether key is a portable environment variable name
// ([A-Za-z_][A-Za-z0-9_]*).
func IsValidEnvKey(key string) bool {
	return len(key) <= maxEnvKeyLen && envKeyRe.MatchString(key)
}

// IsValidNixAttrPath reports whether attr is a plain, unquoted Nix attribute
// path such as "jq" or "python3Packages.black", safe to emit as pkgs.<attr>.
func IsValidNixAttrPath(attr string) bool {
	return len(attr) <= maxNixAttrPathLen && nixAttrPathRe.MatchString(attr)
}

// IsValidToken reports whether s is a single bare word of letters, digits,
// '.', '_' and '-' (starting with a letter or digit), such as a service
// version ("16"), a package manager ("pnpm") or a simple setting value.
func IsValidToken(s string) bool {
	return len(s) <= maxTokenLen && tokenRe.MatchString(s)
}

// IsValidVersionConstraint reports whether v is a version or version
// constraint ("1.24", "3.12.1", ">=18 <21", "^20.0.0", "nightly-2024-01-01").
func IsValidVersionConstraint(v string) bool {
	return len(v) <= maxVersionConstraintLen && versionConstraintRe.MatchString(v)
}

// IsValidToolNamePattern reports whether s is a Claude Code tool name, or a
// tool name pattern with '*' wildcards, as a tool-gates policy entry
// ("Bash", "mcp__github__delete_repo", "mcp__github__*").
func IsValidToolNamePattern(s string) bool {
	return len(s) <= maxToolNamePatternLen && toolNamePatternRe.MatchString(s)
}

// ErrToolNamePattern reports a tool-gates entry that is not a tool name
// pattern.
var ErrToolNamePattern = errors.New("not a Claude Code tool name: use letters, digits, '_' and '-', with '*' as a wildcard")

// CheckToolNamePattern returns ErrToolNamePattern when s is not a valid
// tool-gates entry (see IsValidToolNamePattern), or nil when it is.
func CheckToolNamePattern(s string) error {
	if !IsValidToolNamePattern(s) {
		return ErrToolNamePattern
	}
	return nil
}
