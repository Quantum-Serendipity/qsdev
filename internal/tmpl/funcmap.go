// Package tmpl provides template rendering with Nix-specific and general-purpose
// template functions for generating devenv.nix, CLAUDE.md, and related config files.
package tmpl

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"
)

// NixFuncMap returns a new template.FuncMap containing all Nix-specific and
// general-purpose template functions. Each call returns an independent copy.
func NixFuncMap() template.FuncMap {
	m := generalFuncMap()
	for k, v := range nixSpecificFuncMap() {
		m[k] = v
	}
	return m
}

// MarkdownFuncMap returns a new template.FuncMap containing only the
// general-purpose template functions (no Nix-specific ones). Each call
// returns an independent copy.
func MarkdownFuncMap() template.FuncMap {
	return generalFuncMap()
}

// nixSpecificFuncMap returns the Nix-only template functions.
func nixSpecificFuncMap() template.FuncMap {
	return template.FuncMap{
		"nixPkgList":    nixPkgList,
		"nixList":       nixList,
		"nixStringList": nixStringList,
		"nixString":     nixString,
		"nixBool":       nixBool,
		"nixMultiline":  nixMultiline,
		"nixAttrSet":    nixAttrSet,
		"nixAttrName":   NixAttrName,
	}
}

// generalFuncMap returns the general-purpose template functions shared by
// both Nix and Markdown renderers.
func generalFuncMap() template.FuncMap {
	return template.FuncMap{
		"indent":              indent,
		"nindent":             nindent,
		"join":                join,
		"lower":               lower,
		"upper":               upper,
		"contains":            containsFunc,
		"dict":                dict,
		"default":             default_,
		"hasAny":              hasAny,
		"comment":             comment,
		"trimTrailingNewline": trimTrailingNewline,
	}
}

// nixIdentPattern matches a single plain Nix identifier (the lexer's ID
// token): a letter or underscore, then letters, digits, underscores,
// apostrophes or hyphens.
const nixIdentPattern = `[A-Za-z_][A-Za-z0-9_'-]*`

var (
	// nixIdentRe matches an attribute name that needs no quoting.
	nixIdentRe = regexp.MustCompile(`^` + nixIdentPattern + `$`)
	// nixAttrPathRe matches a dotted attribute path of plain identifiers,
	// e.g. "jq" or "python312Packages.pip".
	nixAttrPathRe = regexp.MustCompile(`^` + nixIdentPattern + `(\.` + nixIdentPattern + `)*$`)
)

// nixKeywords are the Nix language keywords. They lex as keywords rather
// than identifiers, so they are rejected in attribute paths and quoted when
// used as attribute names.
var nixKeywords = []string{"assert", "else", "if", "in", "inherit", "let", "or", "rec", "then", "with"}

// ValidateNixAttrPath returns an error unless s is a dotted Nix attribute
// path made only of plain identifiers (e.g. "jq", "nodePackages.pnpm").
// Anything else (whitespace, brackets, quotes, ';', interpolation) could
// splice arbitrary Nix code into generated source, so it is rejected.
func ValidateNixAttrPath(s string) error {
	if !nixAttrPathRe.MatchString(s) {
		return fmt.Errorf("invalid Nix attribute path %q: must be dot-separated identifiers matching %s", s, nixIdentPattern)
	}
	for _, part := range strings.Split(s, ".") {
		if slices.Contains(nixKeywords, part) {
			return fmt.Errorf("invalid Nix attribute path %q: %q is a Nix keyword", s, part)
		}
	}
	return nil
}

// NixAttrName renders key as a Nix attribute name: bare when it is a plain
// identifier, otherwise (including keywords) as an escaped Nix string
// (`"my.key" = ...;`), so a key can never break out of the attribute set it
// is written into.
func NixAttrName(key string) string {
	if nixIdentRe.MatchString(key) && !slices.Contains(nixKeywords, key) {
		return key
	}
	return nixString(key)
}

// nixPkgList formats a string slice as a Nix package list with pkgs. prefix.
// Example: ["git", "curl"] -> "[ pkgs.git pkgs.curl ]"
// Empty input returns "[ ]". Every item must be a plain attribute path (see
// ValidateNixAttrPath); anything else is an error, never raw Nix code.
func nixPkgList(items []string) (string, error) {
	if len(items) == 0 {
		return "[ ]", nil
	}
	parts := make([]string, len(items))
	for i, item := range items {
		if err := ValidateNixAttrPath(item); err != nil {
			return "", fmt.Errorf("nixPkgList: %w", err)
		}
		parts[i] = "pkgs." + item
	}
	return "[ " + strings.Join(parts, " ") + " ]", nil
}

// nixList formats a string slice as a bare Nix list (no quoting, no pkgs. prefix).
// Example: ["git", "curl"] -> "[ git curl ]"
// Empty input returns "[ ]". Every item must be a plain attribute path (see
// ValidateNixAttrPath); anything else is an error, never raw Nix code.
func nixList(items []string) (string, error) {
	if len(items) == 0 {
		return "[ ]", nil
	}
	for _, item := range items {
		if err := ValidateNixAttrPath(item); err != nil {
			return "", fmt.Errorf("nixList: %w", err)
		}
	}
	return "[ " + strings.Join(items, " ") + " ]", nil
}

// nixStringList formats a string slice as a Nix list of quoted strings,
// with proper Nix escaping applied to each element.
// Example: ["git", "curl"] -> `[ "git" "curl" ]`
// Empty input returns "[ ]".
func nixStringList(items []string) string {
	if len(items) == 0 {
		return "[ ]"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = nixString(item)
	}
	return "[ " + strings.Join(parts, " ") + " ]"
}

// nixString wraps a string in Nix double quotes with proper escaping.
// Escaping order: \ -> \\, " -> \", ${ -> \${
// Example: `hello ${world}` -> `"hello \${world}"`
func nixString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `${`, `\${`)
	return `"` + s + `"`
}

// nixBool converts a Go bool to a Nix boolean literal string.
func nixBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// nixMultiline escapes a string for use inside Nix ” ... ” multiline strings.
// ” -> ”' and ${ -> ”${
//
// It works in a single pass over runs of quotes so escapes cannot merge: a
// lone quote directly before ${ (or at the very end) would otherwise fuse with
// the following two-quote escape into three quotes (an escaped pair) and leave
// ${ as a live antiquotation, so that quote is written as ${"'"} instead.
func nixMultiline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\'':
			j := i
			for j < len(s) && s[j] == '\'' {
				j++
			}
			n := j - i
			b.WriteString(strings.Repeat("'''", n/2))
			if n%2 == 1 {
				if j == len(s) || strings.HasPrefix(s[j:], "${") {
					b.WriteString(`${"'"}`)
				} else {
					b.WriteByte('\'')
				}
			}
			i = j
		case strings.HasPrefix(s[i:], "${"):
			b.WriteString("''${")
			i += 2
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// nixAttrSet formats a map as a Nix attribute set with sorted keys and
// nixString-escaped values. Keys that are not plain identifiers are quoted
// (see NixAttrName).
// Example: {"a": "1", "b": "2"} -> `{ a = "1"; b = "2"; }`
// Empty map returns "{ }".
func nixAttrSet(kvPairs map[string]string) string {
	if len(kvPairs) == 0 {
		return "{ }"
	}
	keys := make([]string, 0, len(kvPairs))
	for k := range kvPairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = NixAttrName(k) + " = " + nixString(kvPairs[k]) + ";"
	}
	return "{ " + strings.Join(parts, " ") + " }"
}

// indent indents ALL lines of s by n spaces. Empty lines remain empty.
func indent(n int, s string) string {
	if n <= 0 {
		return s
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// nindent prepends a newline, then indents all lines of s by n spaces.
func nindent(n int, s string) string {
	return "\n" + indent(n, s)
}

// join wraps strings.Join.
func join(sep string, items []string) string {
	return strings.Join(items, sep)
}

// lower wraps strings.ToLower.
func lower(s string) string {
	return strings.ToLower(s)
}

// upper wraps strings.ToUpper.
func upper(s string) string {
	return strings.ToUpper(s)
}

// containsFunc wraps slices.Contains, checking if needle is in haystack.
func containsFunc(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

// dict builds a map[string]any from alternating key-value pairs.
func dict(keyvals ...any) (map[string]any, error) {
	if len(keyvals)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (%d)", len(keyvals))
	}
	m := make(map[string]any, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at position %d is not a string: %T", i, keyvals[i])
		}
		m[key] = keyvals[i+1]
	}
	return m, nil
}

// default_ returns val if it is non-zero, otherwise returns defaultVal.
// Registered in the FuncMap as "default".
func default_(defaultVal, val any) any {
	if isZero(val) {
		return defaultVal
	}
	return val
}

// hasAny returns true if the given value is a non-empty slice.
// It accepts any slice type via reflect.
func hasAny(items any) bool {
	if items == nil {
		return false
	}
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return false
}

// comment prefixes each line of text with the given prefix and a space.
func comment(prefix, text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + " " + line
		} else {
			lines[i] = prefix
		}
	}
	return strings.Join(lines, "\n")
}

// trimTrailingNewline removes trailing newline characters from a string.
func trimTrailingNewline(s string) string {
	return strings.TrimRight(s, "\n")
}

// isZero checks whether a value is the zero value for its type.
func isZero(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case int:
		return v == 0
	case int64:
		return v == 0
	case float64:
		return v == 0
	case bool:
		return !v
	case []string:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case map[string]string:
		return len(v) == 0
	default:
		return false
	}
}
