package pkgmanager

import (
	"context"
	"fmt"
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

func (n *Nix) Install(ctx context.Context, packages ...string) error {
	if n.isNixOS {
		return fmt.Errorf(
			"on NixOS, add packages to your configuration.nix or home-manager config instead of installing imperatively; "+
				"add the following to environment.systemPackages: %v", packages,
		)
	}
	for _, pkg := range packages {
		bin, args := n.InstallArgs(pkg)
		if err := n.runner.Run(ctx, bin, args...); err != nil {
			return fmt.Errorf("nix profile install %s: %w", pkg, err)
		}
	}
	return nil
}

// InstallArgs returns the command Install runs for packages (Install runs it
// once per package so a failure names the package).
func (n *Nix) InstallArgs(packages ...string) (string, []string) {
	args := []string{"profile", "install"}
	for _, pkg := range packages {
		args = append(args, "nixpkgs#"+pkg)
	}
	return "nix", args
}
