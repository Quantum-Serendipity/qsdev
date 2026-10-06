// Package installer provides a declarative tool installation pattern.
//
// It extracts the common "detect → install → fallback" logic used across
// multiple bootstrap addons so that each addon only needs to declare a
// [ToolSpec] and wire it into a bootstrap step.
package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// ToolSpec declaratively describes a tool that can be installed.
type ToolSpec struct {
	DisplayName   string   // human-readable name, e.g. "devenv", "Claude Code"
	Binary        string   // executable name on PATH, e.g. "devenv", "claude"
	VersionFlag   string   // flag to print version, e.g. "--version"
	InstallCmd    []string // full install command, e.g. {"npm","install","-g","--ignore-scripts","pkg@1.2.3"}
	ManagerBinary string   // package-manager binary, e.g. "nix", "npm"
	ManagerName   string   // human-readable manager name, e.g. "Nix", "npm"
	FallbackURL   string   // URL to install the package manager itself
	DirectURL     string   // URL for direct/manual tool installation
	// MinVersion is the oldest acceptable version; "" accepts any. A found
	// binary below it is upgraded with InstallCmd.
	MinVersion string
	// ParseVersion extracts the version from the full version output; nil
	// uses its first line.
	ParseVersion func(raw string) string
}

// ErrNotFoundAfterInstall marks an install command that exited successfully
// but left the tool's binary unresolvable on PATH, typically because the
// package manager's bin directory (~/.nix-profile/bin, npm's global prefix)
// is not on PATH in this shell.
var ErrNotFoundAfterInstall = errors.New("tool not found on PATH after install")

// ErrOutdatedAfterInstall marks an install command that exited successfully
// but left a version below ToolSpec.MinVersion first on PATH, typically
// because an older copy earlier on PATH shadows the new one.
var ErrOutdatedAfterInstall = errors.New("tool still below the minimum version after install")

// detected is the result of probing for spec's binary.
type detected struct {
	toolcheck.Info
	version string // parsed version; "" when unknown
}

// detect probes for spec's binary and parses its version.
func (spec ToolSpec) detect(ctx context.Context) detected {
	info := toolcheck.Detect(ctx, spec.Binary, spec.VersionFlag)
	d := detected{Info: info, version: info.Version}
	if spec.ParseVersion != nil {
		d.version = ""
		if info.Output != "" {
			d.version = spec.ParseVersion(info.Output)
		}
	}
	return d
}

// satisfied reports whether d is a found binary that meets spec's floor.
func (spec ToolSpec) satisfied(d detected) bool {
	return d.Found && toolcheck.MeetsMinimum(d.version, spec.MinVersion)
}

// Install checks if the tool described by spec is already installed at a
// version meeting spec.MinVersion. If not, it installs (or upgrades) it via
// the configured package manager and then detects the binary again,
// failing with [ErrNotFoundAfterInstall] when it still cannot be resolved
// and with [ErrOutdatedAfterInstall] when it is still below the floor.
// When the package manager is unavailable it prints fallback instructions
// and returns an error.
func Install(ctx context.Context, spec ToolSpec) error {
	d := spec.detect(ctx)
	if spec.satisfied(d) {
		fmt.Printf("%s already installed: %s (%s)\n", spec.DisplayName, d.Version, d.Path)
		return nil
	}
	if d.Found {
		fmt.Printf("%s %s (%s) is below the minimum %s.\n", spec.DisplayName, versionOrUnknown(d.version), d.Path, spec.MinVersion)
	}

	if _, err := exec.LookPath(spec.ManagerBinary); err == nil {
		return spec.runInstall(ctx)
	}

	if d.Found {
		fmt.Printf("%s cannot be upgraded because %s is not available.\n", spec.DisplayName, spec.ManagerName)
	} else {
		fmt.Printf("%s is not installed and %s is not available.\n", spec.DisplayName, spec.ManagerName)
	}
	fmt.Println("Install options:")
	fmt.Printf("  1. Install %s: %s\n", spec.ManagerName, spec.FallbackURL)
	fmt.Printf("     Then: %s\n", strings.Join(spec.InstallCmd, " "))
	fmt.Printf("  2. Direct: %s\n", spec.DirectURL)
	if d.Found {
		return fmt.Errorf("%s %s is below the minimum %s; manual upgrade required", spec.DisplayName, versionOrUnknown(d.version), spec.MinVersion)
	}
	return fmt.Errorf("%s not installed; manual installation required", spec.DisplayName)
}

// runInstall runs spec.InstallCmd and verifies the result.
func (spec ToolSpec) runInstall(ctx context.Context) error {
	cmdStr := strings.Join(spec.InstallCmd, " ")
	fmt.Printf("Installing %s via %s...\n", spec.DisplayName, cmdStr)
	cmd := exec.CommandContext(ctx, spec.InstallCmd[0], spec.InstallCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing %s: %w", spec.DisplayName, err)
	}
	d := spec.detect(ctx)
	switch {
	case !d.Found:
		return fmt.Errorf("installing %s: %w: %q ran but %s is not on PATH; add the %s bin directory to PATH (or open a new shell) and re-run",
			spec.DisplayName, ErrNotFoundAfterInstall, cmdStr, spec.Binary, spec.ManagerName)
	case !spec.satisfied(d):
		return fmt.Errorf("installing %s: %w: %q ran but %s on PATH is %s, below the minimum %s; put the %s bin directory ahead of older copies on PATH (or open a new shell) and re-run",
			spec.DisplayName, ErrOutdatedAfterInstall, cmdStr, d.Path, versionOrUnknown(d.version), spec.MinVersion, spec.ManagerName)
	}
	fmt.Printf("%s installed successfully: %s (%s)\n", spec.DisplayName, d.Version, d.Path)
	return nil
}

// versionOrUnknown returns v, or "(unknown version)" when it is empty.
func versionOrUnknown(v string) string {
	if v == "" {
		return "(unknown version)"
	}
	return v
}

// Simulate logs what [Install] would do without executing any commands.
func Simulate(ctx context.Context, spec ToolSpec) error {
	d := spec.detect(ctx)
	if spec.satisfied(d) {
		fmt.Printf("%s already installed: %s (%s)\n", spec.DisplayName, d.Version, d.Path)
		return nil
	}
	_, mgrErr := exec.LookPath(spec.ManagerBinary)
	switch {
	case mgrErr != nil && d.Found:
		fmt.Printf("Would need manual %s upgrade from %s to >= %s (%s not available)\n", spec.DisplayName, versionOrUnknown(d.version), spec.MinVersion, spec.ManagerName)
	case mgrErr != nil:
		fmt.Printf("Would need manual %s installation (%s not available)\n", spec.DisplayName, spec.ManagerName)
	case d.Found:
		fmt.Printf("Would upgrade %s from %s to >= %s: %s\n", spec.DisplayName, versionOrUnknown(d.version), spec.MinVersion, strings.Join(spec.InstallCmd, " "))
	default:
		fmt.Printf("Would run: %s\n", strings.Join(spec.InstallCmd, " "))
	}
	return nil
}
