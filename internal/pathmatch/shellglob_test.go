package pathmatch

import (
	"errors"
	"path"
	"testing"
)

// TestShellMatch pins that a glob is read with the shell's bracket rules, not
// path.Match's: '!' negates, a ']' first in a class is literal, POSIX classes
// name sets, and an unterminated '[' is literal. A path.Match reading of
// `.[!x]laude` matches only `.!laude` and `.xlaude`, so a guard built on it
// let `rm -rf .[!x]laude` delete .claude.
func TestShellMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		glob, name string
		want       bool
	}{
		{".[!x]laude", ".claude", true},
		{".[!c]laude", ".claude", false},
		{".[^x]laude", ".claude", true},
		{".[!.]*", ".claude", true},
		{".[!.]*", "..", false},
		{".[[:alpha:]]laude", ".claude", true},
		{".[[:digit:]]laude", ".claude", false},
		{".[![:digit:]]laude", ".claude", true},
		{".[[:lower:]]*", ".claude", true},
		{".[[:upper:]]*", ".claude", false},
		{".[[:alnum:]_]laude", ".claude", true},
		{".[!]]*", ".claude", true},
		{".[!]]*", ".]x", false},
		{"[]]x", "]x", true},
		{"[a-]x", "-x", true},
		{"[-a]x", "-x", true},
		{"[a-c]laude", "claude", true},
		{"[!a-c]laude", "claude", false},
		{"[[=c=]]laude", "claude", true},
		{"[[.c.]]laude", "claude", true},
		{"[[:punct:]]x", "!x", true},
		{"[[:punct:]]x", "ax", false},
		{"[[:space:]]x", "\tx", true},
		{"[!x", "[!x", true}, // unterminated: a literal '['
		{"a[", "a[", true},
		{`.\[x`, ".[x", true},
		{`[\]]x`, "]x", true},
		{"[^^]x", "^x", false},
		{"[!^]x", "ax", true},
		{".cla?de", ".claude", true},
		{".c*e", ".claude", true},
		{"settings.json", "settings.json", true},
	}
	for _, tt := range tests {
		t.Run(tt.glob+" "+tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ShellMatch(tt.glob, tt.name)
			if err != nil {
				t.Fatalf("ShellMatch(%q, %q) error %v", tt.glob, tt.name, err)
			}
			if got != tt.want {
				t.Errorf("ShellMatch(%q, %q) = %v, want %v", tt.glob, tt.name, got, tt.want)
			}
		})
	}
}

// TestShellMatch_Unreadable pins that a construct the rewrite cannot express
// reports path.ErrBadPattern, which the guards treat as a match.
func TestShellMatch_Unreadable(t *testing.T) {
	t.Parallel()
	for _, glob := range []string{".[[:nosuch:]]laude", ".[[.ch.]]laude"} {
		t.Run(glob, func(t *testing.T) {
			t.Parallel()
			if _, err := ShellMatch(glob, ".claude"); !errors.Is(err, path.ErrBadPattern) {
				t.Errorf("ShellMatch(%q) error = %v, want path.ErrBadPattern", glob, err)
			}
		})
	}
}
