package config

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// VersionConstraint wraps a parsed semver constraint for qsdev_version checks.
type VersionConstraint struct {
	raw        string
	constraint *semver.Constraints
}

// VersionMismatchError is returned when the binary version does not satisfy
// the qsdev_version constraint in .qsdev.yaml.
type VersionMismatchError struct {
	BinaryVersion  string
	Constraint     string
	UpgradeCommand string
}

// Error implements the error interface with an actionable message.
func (e *VersionMismatchError) Error() string {
	app := branding.Get().AppName
	msg := fmt.Sprintf(
		"%s version %s does not satisfy the project's qsdev_version constraint %q",
		app, e.BinaryVersion, e.Constraint)
	if e.UpgradeCommand != "" {
		msg += fmt.Sprintf("; run %q to update", e.UpgradeCommand)
	}
	return msg
}

// RatchetWarning is returned when the current binary version is older than the
// version that last generated files.
type RatchetWarning struct {
	CurrentVersion string
	LastRunVersion string
}

// Error implements the error interface.
func (w *RatchetWarning) Error() string {
	app := branding.Get().AppName
	return fmt.Sprintf(
		"current %s version %s is older than the version (%s) that last generated this project's files; "+
			"use --force to proceed anyway, or update %s",
		app, w.CurrentVersion, w.LastRunVersion, app)
}

// ParseVersionConstraint parses a version constraint string. The Masterminds
// semver library natively supports comparison operators, caret (^), OR groups
// (||) and the pessimistic operator (~>), which it treats as a tilde range:
//
//   - ~> X     means  >= X.0.0, < (X+1).0.0
//   - ~> X.Y   means  >= X.Y.0, < X.(Y+1).0
//   - ~> X.Y.Z means  >= X.Y.Z, < X.(Y+1).0
func ParseVersionConstraint(raw string) (*VersionConstraint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("version constraint must not be empty")
	}

	c, err := semver.NewConstraint(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid version constraint %q: %w", raw, err)
	}

	return &VersionConstraint{raw: raw, constraint: c}, nil
}

// Check tests whether the given version string satisfies this constraint.
// It tolerates a leading "v" prefix on the version.
func (vc *VersionConstraint) Check(version string) (bool, error) {
	version = strings.TrimPrefix(version, "v")

	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", version, err)
	}

	return vc.constraint.Check(v), nil
}

// String returns the original raw constraint string.
func (vc *VersionConstraint) String() string {
	return vc.raw
}

// CheckBinaryVersion checks whether binaryVersion satisfies the qsdev_version
// constraint from .qsdev.yaml. Returns nil if:
//   - qsdevVersionConstraint is empty (no constraint specified)
//   - binaryVersion is "dev" or "(devel)" (development build)
//   - the constraint is satisfied
//
// Returns a *VersionMismatchError if the constraint is not satisfied.
func CheckBinaryVersion(qsdevVersionConstraint, binaryVersion string) error {
	if qsdevVersionConstraint == "" {
		return nil
	}

	// Dev builds always pass.
	if isDevBuild(binaryVersion) {
		return nil
	}

	vc, err := ParseVersionConstraint(qsdevVersionConstraint)
	if err != nil {
		return fmt.Errorf("parsing qsdev_version constraint: %w", err)
	}

	ok, err := vc.Check(binaryVersion)
	if err != nil {
		return fmt.Errorf("checking qsdev_version constraint: %w", err)
	}

	if !ok {
		return &VersionMismatchError{
			BinaryVersion:  binaryVersion,
			Constraint:     qsdevVersionConstraint,
			UpgradeCommand: "nix flake update",
		}
	}

	return nil
}

// CheckVersionRatchet compares the current binary version against the version
// that last generated files. Returns nil if:
//   - either version is a dev build
//   - current >= lastRun
//
// Returns a *RatchetWarning if current < lastRun.
func CheckVersionRatchet(currentVersion, lastRunVersion string) *RatchetWarning {
	if lastRunVersion == "" {
		return nil
	}

	if isDevBuild(currentVersion) || isDevBuild(lastRunVersion) {
		return nil
	}

	current, err := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return nil // Can't parse, don't block.
	}

	last, err := semver.NewVersion(strings.TrimPrefix(lastRunVersion, "v"))
	if err != nil {
		return nil // Can't parse, don't block.
	}

	if current.LessThan(last) {
		return &RatchetWarning{
			CurrentVersion: currentVersion,
			LastRunVersion: lastRunVersion,
		}
	}

	return nil
}

// MinimumVersionConstraint returns the qsdev_version constraint that a newly
// generated .qsdev.yaml should carry for a project initialized by
// binaryVersion. It is a lower bound (">= X.Y.Z") with build metadata
// stripped, so teammates and CI runners on the same or any newer release
// satisfy it. A pre-release suffix is kept, because semver constraints
// without one never match pre-release binaries (including the one writing
// the file). It returns "" (no constraint) for development builds and for
// versions that are not valid semver, because a point version or an
// unparseable string would pin every other binary out of the project.
func MinimumVersionConstraint(binaryVersion string) string {
	if isDevBuild(binaryVersion) {
		return ""
	}
	v, err := semver.NewVersion(strings.TrimPrefix(binaryVersion, "v"))
	if err != nil {
		return ""
	}
	if pre := v.Prerelease(); pre != "" {
		return fmt.Sprintf(">= %d.%d.%d-%s", v.Major(), v.Minor(), v.Patch(), pre)
	}
	return fmt.Sprintf(">= %d.%d.%d", v.Major(), v.Minor(), v.Patch())
}

// isDevBuild returns true for development/unreleased builds.
func isDevBuild(version string) bool {
	return version == "" || version == "dev" || version == "(devel)"
}
