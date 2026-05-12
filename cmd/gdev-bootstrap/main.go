package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devenv"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devinit"
)

func main() {
	instance.SetAppName("gdev-secure-bootstrap")

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

	cmd.Main()
}
