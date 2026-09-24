package validation

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckBranchPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"empty selects the default", "", false},
		{"conventional prefixes", `^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$`, false},
		{"bracket class with dollar anchor", `^[A-Za-z0-9][A-Za-z0-9._/@+-]*$`, false},
		{"posix character class", `^[[:alnum:]]+(/[[:alnum:]_-]+)*$`, false},
		{"backslash escape", `^release/v[0-9]+\.[0-9]+$`, false},
		{"unbalanced group", `^(feat|fix/`, true},
		{"unbalanced bracket", `^[a-z`, true},
		{"single quote breaks shell quoting", `^feat/it's$`, true},
		{"newline", "^feat/\n", true},
		{"tab", "^feat/\t", true},
		{"right-to-left override", "^feat/\u202e", true},
		{"non-ASCII letter", "^f\u00e9at/", true},
		{"too long", "^" + strings.Repeat("a", maxBranchPatternLen), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckBranchPattern(tt.pattern)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("CheckBranchPattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalidBranchPattern) {
				t.Errorf("error %v does not wrap ErrInvalidBranchPattern", err)
			}
		})
	}
}
