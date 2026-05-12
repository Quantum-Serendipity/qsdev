package devenv

import (
	"fmt"
	"os"
	"os/exec"

	"fastcat.org/go/gdev/addons/bootstrap"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/toolcheck"
)

const (
	// StepNameInstallDevenv is the name of the bootstrap step that installs devenv.
	StepNameInstallDevenv = "Install devenv"
	// StepNameInstallDirenv is the name of the bootstrap step that installs direnv.
	StepNameInstallDirenv = "Install direnv"
)

// InstallDevenvStep returns a bootstrap step that ensures devenv is installed.
func InstallDevenvStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallDevenv,
		installDevenv,
		bootstrap.SimFunc(simInstallDevenv),
		bootstrap.SkipInContainer(),
	)
}

func installDevenv(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "devenv", "--version")
	if info.Found {
		fmt.Printf("devenv already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}

	if _, err := exec.LookPath("nix"); err == nil {
		fmt.Println("Installing devenv via nix profile install...")
		cmd := exec.CommandContext(ctx, "nix", "profile", "install", "--accept-flake-config", "nixpkgs#devenv")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing devenv: %w", err)
		}
		fmt.Println("devenv installed successfully.")
		return nil
	}

	fmt.Println("devenv is not installed and nix is not available.")
	fmt.Println("Install options:")
	fmt.Println("  1. Install Nix: https://nixos.org/download")
	fmt.Println("     Then: nix profile install nixpkgs#devenv")
	fmt.Println("  2. Direct: https://devenv.sh/getting-started/")
	return fmt.Errorf("devenv not installed; manual installation required")
}

func simInstallDevenv(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "devenv", "--version")
	if info.Found {
		fmt.Printf("devenv already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}
	if _, err := exec.LookPath("nix"); err == nil {
		fmt.Println("Would run: nix profile install nixpkgs#devenv")
	} else {
		fmt.Println("Would need manual devenv installation (nix not available)")
	}
	return nil
}

// InstallDirenvStep returns a bootstrap step that ensures direnv is installed.
func InstallDirenvStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallDirenv,
		installDirenv,
		bootstrap.SimFunc(simInstallDirenv),
		bootstrap.SkipInContainer(),
	)
}

func installDirenv(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "direnv", "--version")
	if info.Found {
		fmt.Printf("direnv already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}

	if _, err := exec.LookPath("nix"); err == nil {
		fmt.Println("Installing direnv via nix profile install...")
		cmd := exec.CommandContext(ctx, "nix", "profile", "install", "--accept-flake-config", "nixpkgs#direnv")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installing direnv: %w", err)
		}
		fmt.Println("direnv installed successfully.")
		return nil
	}

	fmt.Println("direnv is not installed and nix is not available.")
	fmt.Println("Install options:")
	fmt.Println("  1. Install Nix: https://nixos.org/download")
	fmt.Println("     Then: nix profile install nixpkgs#direnv")
	fmt.Println("  2. Direct: https://direnv.net/docs/installation.html")
	return fmt.Errorf("direnv not installed; manual installation required")
}

func simInstallDirenv(ctx *bootstrap.Context) error {
	info := toolcheck.Detect(ctx, "direnv", "--version")
	if info.Found {
		fmt.Printf("direnv already installed: %s (%s)\n", info.Version, info.Path)
		return nil
	}
	if _, err := exec.LookPath("nix"); err == nil {
		fmt.Println("Would run: nix profile install nixpkgs#direnv")
	} else {
		fmt.Println("Would need manual direnv installation (nix not available)")
	}
	return nil
}
