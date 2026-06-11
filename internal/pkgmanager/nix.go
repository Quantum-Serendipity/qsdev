package pkgmanager

import (
	"context"
	"fmt"
	"strings"
)

// Nix implements PackageManager for the Nix package manager.
// When isNixOS is true, Install returns an error suggesting declarative
// configuration instead of imperative installs.
type Nix struct {
	runner  CommandRunner
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

func (n *Nix) IsInstalled(ctx context.Context, pkg string) bool {
	out, err := n.runner.Output(ctx, "nix", "profile", "list")
	if err != nil {
		return false
	}
	// Check if the package name appears in the profile listing.
	return strings.Contains(string(out), pkg)
}

func (n *Nix) SearchCmd() string { return "nix search nixpkgs" }
