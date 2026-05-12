// Package tmpl provides template rendering with Nix-specific and general-purpose
// template functions for generating devenv.nix, CLAUDE.md, and related config files.
package tmpl

import (
	"fmt"
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
		"nixPkgList":   nixPkgList,
		"nixList":      nixList,
		"nixStringList": nixStringList,
		"nixString":    nixString,
		"nixBool":      nixBool,
		"nixMultiline": nixMultiline,
		"nixAttrSet":   nixAttrSet,
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

// nixPkgList formats a string slice as a Nix package list with pkgs. prefix.
// Example: ["git", "curl"] -> "[ pkgs.git pkgs.curl ]"
// Empty input returns "[ ]".
func nixPkgList(items []string) string {
	if len(items) == 0 {
		return "[ ]"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = "pkgs." + item
	}
	return "[ " + strings.Join(parts, " ") + " ]"
}

// nixList formats a string slice as a bare Nix list (no quoting, no pkgs. prefix).
// Example: ["git", "curl"] -> "[ git curl ]"
// Empty input returns "[ ]".
func nixList(items []string) string {
	if len(items) == 0 {
		return "[ ]"
	}
	return "[ " + strings.Join(items, " ") + " ]"
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

// nixMultiline escapes a string for use inside Nix '' ... '' multiline strings.
// '' -> ''' and ${ -> ''${
func nixMultiline(s string) string {
	s = strings.ReplaceAll(s, "''", "'''")
	s = strings.ReplaceAll(s, "${", "''${")
	return s
}

// nixAttrSet formats a map as a Nix attribute set with sorted keys and
// nixString-escaped values.
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
		parts[i] = k + " = " + nixString(kvPairs[k]) + ";"
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
// Panics if an odd number of arguments is provided.
func dict(keyvals ...any) map[string]any {
	if len(keyvals)%2 != 0 {
		panic(fmt.Sprintf("dict: odd number of arguments (%d)", len(keyvals)))
	}
	m := make(map[string]any, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			panic(fmt.Sprintf("dict: key at position %d is not a string: %T", i, keyvals[i]))
		}
		m[key] = keyvals[i+1]
	}
	return m
}

// default_ returns val if it is non-zero, otherwise returns defaultVal.
// Registered in the FuncMap as "default".
func default_(defaultVal, val any) any {
	if isZero(val) {
		return defaultVal
	}
	return val
}

// hasAny returns true if the slice has at least one element.
func hasAny(items []string) bool {
	return len(items) > 0
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
