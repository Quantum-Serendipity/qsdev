package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devinit"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/selfupdate"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
)

func main() {
	instance.SetAppName("qsdev")

	bootstrap.Configure(
		bootstrap.WithSteps(
			devenv.InstallDevenvStep(),
			devenv.InstallDirenvStep(),
			claudecode.InstallClaudeStep(),
		),
	)

	devenv.Configure(
		devenv.WithDefaultLanguages("go"),
		devenv.WithDirenv(true),
	)
	claudecode.Configure(
		claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard),
	)
	devinit.Configure(
		devinit.WithDetectProjectType(true),
		devinit.WithPlanPreview(true),
	)

	instance.AddCommands(selfupdate.Command())

	updateCh := selfupdate.BackgroundCheck(version.Info().Version)
	cmd.Main()
	selfupdate.PrintNotice(updateCh)
}
