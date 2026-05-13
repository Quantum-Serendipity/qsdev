package pkgmanager

import (
	"context"
	"fmt"
)

// Nix implements PackageManager for the Nix package manager.
// When isNixOS is true, Install returns an error suggesting declarative
// configuration instead of imperative installs.
type Nix struct {
	runner CommandRunner
	isNixOS bool
}

// NewNix creates a Nix package manager. If runner is nil, DefaultRunner() is used.
// When isNixOS is true, Install returns an error with a declarative config suggestion.
func NewNix(runner CommandRunner, isNixOS bool) *Nix {
	return &Nix{runner: ensureRunner(runner), isNixOS: isNixOS}
}

func (n *Nix) Name() string { return "nix" }

func (n *Nix) Available() bool {
	_, err := n.runner.LookPath("nix")
	return err == nil
}

func (n *Nix) NeedsElevation() bool { return false }

func (n *Nix) UpdateIndex(ctx context.Context) error {
	return n.runner.Run(ctx, "nix", "flake", "update")
}

func (n *Nix) Install(ctx context.Context, packages ...string) error {
	if n.isNixOS {
		return fmt.Errorf(
			"on NixOS, add packages to your configuration.nix or home-manager config instead of installing imperatively; "+
				"add the following to environment.systemPackages: %v", packages,
		)
	}
	for _, pkg := range packages {
		if err := n.runner.Run(ctx, "nix", "profile", "install", "nixpkgs#"+pkg); err != nil {
			return fmt.Errorf("nix profile install %s: %w", pkg, err)
		}
	}
	return nil
}

func (n *Nix) IsInstalled(pkg string) bool {
	out, err := n.runner.Output(context.Background(), "nix", "profile", "list")
	if err != nil {
		return false
	}
	// Check if the package name appears in the profile listing.
	return containsWord(string(out), pkg)
}

func (n *Nix) SearchCmd() string { return "nix search nixpkgs" }

// containsWord checks if s contains word as a substring.
func containsWord(s, word string) bool {
	return len(word) > 0 && len(s) > 0 && contains(s, word)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
