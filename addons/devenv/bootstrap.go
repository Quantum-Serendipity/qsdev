package devenv

import (
	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

const (
	// StepNameInstallDevenv is the name of the bootstrap step that installs devenv.
	StepNameInstallDevenv = "Install devenv"
	// StepNameInstallDirenv is the name of the bootstrap step that installs direnv.
	StepNameInstallDirenv = "Install direnv"
)

var devenvSpec = installer.ToolSpec{
	DisplayName:   "devenv",
	Binary:        "devenv",
	VersionFlag:   "--version",
	InstallCmd:    []string{"nix", "profile", "install", "--accept-flake-config", "nixpkgs#devenv"},
	ManagerBinary: "nix",
	ManagerName:   "Nix",
	FallbackURL:   "https://nixos.org/download",
	DirectURL:     "https://devenv.sh/getting-started/",
}

var direnvSpec = installer.ToolSpec{
	DisplayName:   "direnv",
	Binary:        "direnv",
	VersionFlag:   "--version",
	InstallCmd:    []string{"nix", "profile", "install", "--accept-flake-config", "nixpkgs#direnv"},
	ManagerBinary: "nix",
	ManagerName:   "Nix",
	FallbackURL:   "https://nixos.org/download",
	DirectURL:     "https://direnv.net/docs/installation.html",
}

// InstallDevenvStep returns a bootstrap step that ensures devenv is installed.
func InstallDevenvStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallDevenv,
		func(ctx *bootstrap.Context) error { return installer.Install(ctx, devenvSpec) },
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error { return installer.Simulate(ctx, devenvSpec) }),
		bootstrap.SkipInContainer(),
	)
}

// InstallDirenvStep returns a bootstrap step that ensures direnv is installed.
func InstallDirenvStep() *bootstrap.Step {
	return bootstrap.NewStep(
		StepNameInstallDirenv,
		func(ctx *bootstrap.Context) error { return installer.Install(ctx, direnvSpec) },
		bootstrap.SimFunc(func(ctx *bootstrap.Context) error { return installer.Simulate(ctx, direnvSpec) }),
		bootstrap.SkipInContainer(),
	)
}
