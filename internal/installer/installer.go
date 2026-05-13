// Package installer provides a declarative tool installation pattern.
//
// It extracts the common "detect → install → fallback" logic used across
// multiple bootstrap addons so that each addon only needs to declare a
// [ToolSpec] and wire it into a bootstrap step.
package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/toolcheck"
)

// ToolSpec declaratively describes a tool that can be installed.
type ToolSpec struct {
	DisplayName   string   // human-readable name, e.g. "devenv", "Claude Code"
	Binary        string   // executable name on PATH, e.g. "devenv", "claude"
	VersionFlag   string   // flag to print version, e.g. "--version"
	InstallCmd    []string // full install command, e.g. {"nix","profile","install","nixpkgs#devenv"}
	ManagerBinary string   // package-manager binary, e.g. "nix", "npm"
	ManagerName   string   // human-readable manager name, e.g. "Nix", "npm"
	FallbackURL   string   // URL to install the package manager itself
	DirectURL     string   // URL for direct/manual tool installation
}

// Install checks if the tool described by spec is already installed.
// If not, it attempts to install via the configured package manager.
// When the package manager is unavailable it prints fallback instructions
// and returns an error.
func Install(ctx context.Context, spec ToolSpec) error {
	info := toolcheck.Detect(ctx, spec.Binary, spec.VersionFlag)
	if info.Found {
		fmt.Printf("%s already installed: %s (%s)\n", spec.DisplayName, info.Version, info.Path)
		return nil
	}

	if _, err := exec.LookPath(spec.ManagerBinary); err == nil {
		cmdStr := strings.Join(spec.InstallCmd, " ")
		fmt.Printf("Installing %s via %s...\n", spec.DisplayName, cmdStr)
		cmd := exec.CommandContext(ctx, spec.InstallCmd[0], spec.InstallCmd[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing %s: %w", spec.DisplayName, err)
		}
		fmt.Printf("%s installed successfully.\n", spec.DisplayName)
		return nil
	}

	fmt.Printf("%s is not installed and %s is not available.\n", spec.DisplayName, spec.ManagerName)
	fmt.Println("Install options:")
	fmt.Printf("  1. Install %s: %s\n", spec.ManagerName, spec.FallbackURL)
	fmt.Printf("     Then: %s\n", strings.Join(spec.InstallCmd, " "))
	fmt.Printf("  2. Direct: %s\n", spec.DirectURL)
	return fmt.Errorf("%s not installed; manual installation required", spec.DisplayName)
}

// Simulate logs what [Install] would do without executing any commands.
func Simulate(ctx context.Context, spec ToolSpec) error {
	info := toolcheck.Detect(ctx, spec.Binary, spec.VersionFlag)
	if info.Found {
		fmt.Printf("%s already installed: %s (%s)\n", spec.DisplayName, info.Version, info.Path)
		return nil
	}
	if _, err := exec.LookPath(spec.ManagerBinary); err == nil {
		fmt.Printf("Would run: %s\n", strings.Join(spec.InstallCmd, " "))
	} else {
		fmt.Printf("Would need manual %s installation (%s not available)\n", spec.DisplayName, spec.ManagerName)
	}
	return nil
}
