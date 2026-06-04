package ecosystem

import "strings"

// NixEscapeString escapes a value for safe interpolation inside Nix
// double-quoted strings. It handles the three Nix metacharacters: backslash,
// double-quote, and the antiquotation opener "${".
func NixEscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `${`, `\${`)
	return s
}

// TOMLEscapeString escapes a value for safe interpolation inside TOML basic
// (double-quoted) strings.
func TOMLEscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// INIEscapeValue sanitizes a value for INI/config files by stripping
// characters that would create multi-line injection or comments.
func INIEscapeValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// GradleEscapeString escapes a value for safe interpolation inside Gradle
// Groovy single-quoted strings.
func GradleEscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}
