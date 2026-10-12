package validation

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

// Errors for hooks.package_guard values.
var (
	ErrPackageGuardEntry = errors.New("not a package name: use name or ecosystem:name (characters @ A-Z a-z 0-9 . _ / -) " +
		"with an ecosystem package-guard knows: " + strings.Join(PackageGuardQualifiers(), ", "))
	ErrPackageGuardMinAgeDays = errors.New("must be at least 1 day")
)

// packageGuardQualifiers are the ecosystem qualifiers package-guard.py
// accepts on a list entry: the keys of its _ECOSYSTEM_QUALIFIERS, sorted
// (pinned by the claudecode drift test TestPackageGuardQualifiersMatchScript).
// The script compares them case-insensitively.
var packageGuardQualifiers = []string{
	"cargo", "composer", "crates", "crates.io", "gem", "go", "golang", "jsr",
	"npm", "nuget", "packagist", "pip", "pub", "pypi", "rubygems",
}

// PackageGuardQualifiers returns the ecosystem qualifiers a package-guard
// list entry may carry, sorted; a fresh slice.
func PackageGuardQualifiers() []string { return slices.Clone(packageGuardQualifiers) }

// packageNameRe is the character set of a package name across the
// ecosystems package-guard checks (npm scopes, Go module paths, PyPI and
// crates names). It excludes ':', ',', whitespace and control characters.
var packageNameRe = regexp.MustCompile(`^[@A-Za-z0-9._/-]+$`)

// CheckPackageGuardEntry returns ErrPackageGuardEntry unless entry is a bare
// package name or "<qualifier>:<name>" with a qualifier from
// PackageGuardQualifiers (case-insensitive, as the script compares it). An
// unknown qualifier is rejected: the script would treat it as a bare name
// that never matches, a silent no-op.
func CheckPackageGuardEntry(entry string) error {
	name := entry
	if qualifier, rest, ok := strings.Cut(entry, ":"); ok {
		if !slices.Contains(packageGuardQualifiers, strings.ToLower(qualifier)) {
			return ErrPackageGuardEntry
		}
		name = rest
	}
	if !packageNameRe.MatchString(name) {
		return ErrPackageGuardEntry
	}
	return nil
}
