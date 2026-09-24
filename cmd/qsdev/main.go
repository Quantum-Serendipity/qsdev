package main

import (
	"fastcat.org/go/gdev/addons/bootstrap"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func main() {
	configure()
	// The runtime wiring (framework adapters, external-log providers, build
	// version, project defaults, the self-update, logs and bug-report commands,
	// and redacting session logging) lives in the instance package so
	// downstream tools, including scaffold-instance output, get exactly the
	// same runtime as qsdev. Main exits the process.
	instance.Main()
}

// configure applies qsdev's branding and addon configuration: every
// qsdev-specific customization that must happen before the command tree is
// built. The command-tree tests call it to build the same tree as the binary.
func configure() {
	instance.SetBranding(branding.Default())

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
}
