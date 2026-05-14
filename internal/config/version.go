package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// VersionConstraint wraps a parsed semver constraint for gdev_version checks.
type VersionConstraint struct {
	raw        string
	constraint *semver.Constraints
}

// VersionMismatchError is returned when the binary version does not satisfy
// the gdev_version constraint in .gdev.yaml.
type VersionMismatchError struct {
	BinaryVersion  string
	Constraint     string
	UpgradeCommand string
}

// Error implements the error interface with an actionable message.
func (e *VersionMismatchError) Error() string {
	msg := fmt.Sprintf(
		"qsdev version %s does not satisfy the project's gdev_version constraint %q",
		e.BinaryVersion, e.Constraint)
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
	return fmt.Sprintf(
		"current qsdev version %s is older than the version (%s) that last generated this project's files; "+
			"use --force to proceed anyway, or update qsdev",
		w.CurrentVersion, w.LastRunVersion)
}

// ParseVersionConstraint parses a version constraint string. It pre-processes
// Terraform's pessimistic operator (~>) into Masterminds-compatible syntax:
//
//   - ~> X.Y   becomes  >= X.Y.0, < X.(Y+1).0
//   - ~> X.Y.Z becomes  >= X.Y.Z, < X.(Y+1).0
//
// The caret (^) and comparison operators are natively supported by the
// Masterminds semver library.
func ParseVersionConstraint(raw string) (*VersionConstraint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("version constraint must not be empty")
	}

	processed := preprocessConstraint(raw)

	c, err := semver.NewConstraint(processed)
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

// CheckBinaryVersion checks whether binaryVersion satisfies the gdev_version
// constraint from .gdev.yaml. Returns nil if:
//   - gdevVersionConstraint is empty (no constraint specified)
//   - binaryVersion is "dev" or "(devel)" (development build)
//   - the constraint is satisfied
//
// Returns a *VersionMismatchError if the constraint is not satisfied.
func CheckBinaryVersion(gdevVersionConstraint, binaryVersion string) error {
	if gdevVersionConstraint == "" {
		return nil
	}

	// Dev builds always pass.
	if isDevBuild(binaryVersion) {
		return nil
	}

	vc, err := ParseVersionConstraint(gdevVersionConstraint)
	if err != nil {
		return fmt.Errorf("parsing gdev_version constraint: %w", err)
	}

	ok, err := vc.Check(binaryVersion)
	if err != nil {
		return fmt.Errorf("checking gdev_version constraint: %w", err)
	}

	if !ok {
		return &VersionMismatchError{
			BinaryVersion:  binaryVersion,
			Constraint:     gdevVersionConstraint,
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

// isDevBuild returns true for development/unreleased builds.
func isDevBuild(version string) bool {
	return version == "" || version == "dev" || version == "(devel)"
}

// preprocessConstraint converts Terraform-style ~> operators to
// Masterminds-compatible constraint syntax.
func preprocessConstraint(raw string) string {
	// Split on comma for compound constraints.
	parts := strings.Split(raw, ",")
	var result []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "~>") {
			expanded := expandPessimistic(strings.TrimSpace(strings.TrimPrefix(part, "~>")))
			result = append(result, expanded)
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, ", ")
}

// expandPessimistic converts a pessimistic version constraint:
//
//	~> X.Y   -> >= X.Y.0, < X.(Y+1).0
//	~> X.Y.Z -> >= X.Y.Z, < X.(Y+1).0
func expandPessimistic(version string) string {
	version = strings.TrimPrefix(version, "v")
	segments := strings.Split(version, ".")

	switch len(segments) {
	case 2:
		// ~> X.Y -> >= X.Y.0, < X.(Y+1).0
		major := segments[0]
		minor, err := strconv.Atoi(segments[1])
		if err != nil {
			return ">= " + version // Fallback, let semver library handle error.
		}
		return fmt.Sprintf(">= %s.%d.0, < %s.%d.0", major, minor, major, minor+1)
	case 3:
		// ~> X.Y.Z -> >= X.Y.Z, < X.(Y+1).0
		major := segments[0]
		minor, err := strconv.Atoi(segments[1])
		if err != nil {
			return ">= " + version
		}
		return fmt.Sprintf(">= %s.%s, < %s.%d.0", major, strings.Join(segments[1:], "."), major, minor+1)
	default:
		return ">= " + version // Fallback.
	}
}
