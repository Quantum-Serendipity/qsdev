package denyutil

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rule string
		want error
	}{
		{"Bash(ls *)", nil},
		{"Bash(deno *npm:**)", nil},
		{"Bash(docker run * -v /:/*)", nil},
		{"Read(~/.ssh/**)", nil},
		{"PowerShell(Get-ChildItem *)", nil},
		{"WebFetch", nil},
		{"mcp__github__*", nil},
		{"Read(src:*)", nil}, // the legacy marker only applies to command rules

		{"Bash(ls:*)", ErrLegacyPrefixSuffix},
		{"Bash(deno npm:*)", ErrLegacyPrefixSuffix},
		{"PowerShell(Get-ChildItem:*)", ErrLegacyPrefixSuffix},
		{"Bash(foo", ErrMalformedRule},
		{"()", ErrMalformedRule},
		{"Bash()", ErrMalformedRule},
		{"", ErrMalformedRule},
		{"Bash foo", ErrMalformedRule},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			t.Parallel()
			err := Validate(tt.rule)
			if !errors.Is(err, tt.want) || (tt.want == nil) != (err == nil) {
				t.Errorf("Validate(%q) = %v, want %v", tt.rule, err, tt.want)
			}
		})
	}
}

// TestValidate_LegacySuffixMessage checks that the error names what Claude
// Code makes of the rule and the spelling that does what was meant.
func TestValidate_LegacySuffixMessage(t *testing.T) {
	t.Parallel()
	err := Validate("Bash(deno npm:*)")
	for _, want := range []string{`"Bash(deno npm *)"`, `"Bash(deno npm:**)"`} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Validate error %v does not mention %s", err, want)
		}
	}
}
