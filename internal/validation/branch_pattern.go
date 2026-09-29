package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// maxBranchPatternLen caps a branch naming pattern; real patterns are a few
// dozen characters.
const maxBranchPatternLen = 256

// ErrInvalidBranchPattern is wrapped by every CheckBranchPattern failure.
var ErrInvalidBranchPattern = errors.New("invalid branch pattern")

// CheckBranchPattern validates git.branch_pattern, the POSIX extended regular
// expression the branch-naming pre-push hook passes to `grep -E`. The pattern
// is spliced into a shell single-quoted string inside a Nix indented string in
// devenv.nix, and it comes from the team-shared .qsdev.yaml, so besides
// compiling as a POSIX ERE it must be printable ASCII without a single quote
// (the one character neither quoting layer can carry). An empty pattern is
// valid: it selects the default.
func CheckBranchPattern(pattern string) error {
	if pattern == "" {
		return nil
	}
	if len(pattern) > maxBranchPatternLen {
		return fmt.Errorf("%w: longer than %d characters", ErrInvalidBranchPattern, maxBranchPatternLen)
	}
	for i, r := range pattern {
		if r < 0x20 || r > 0x7e {
			return fmt.Errorf("%w: only printable ASCII characters are allowed (found %U at offset %d)", ErrInvalidBranchPattern, r, i)
		}
	}
	if strings.ContainsRune(pattern, '\'') {
		return fmt.Errorf("%w: a single quote (') is not allowed", ErrInvalidBranchPattern)
	}
	if _, err := regexp.CompilePOSIX(pattern); err != nil {
		return fmt.Errorf("%w: not a POSIX extended regular expression: %w", ErrInvalidBranchPattern, err)
	}
	return nil
}
