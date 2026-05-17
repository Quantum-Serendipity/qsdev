// Package main demonstrates how a downstream tool ("acmedev") can import qsdev
// as a framework, customize branding, register additional ecosystem modules,
// and reuse qsdev's addons — exactly as qsdev imports gdev.
//
// Build: go build -o acmedev ./examples/downstream
package main

import (
	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

func main() {
	instance.SetBranding(branding.Config{
		AppName:      "acmedev",
		ConfigFile:   ".acmedev.yaml",
		LocalConfig:  ".acmedev.local.yaml",
		StateDir:     ".acmedev",
		EnvLogVar:    "ACMEDEV_LOG",
		EnvLogDirVar: "ACMEDEV_LOG_DIR",
		EnvNoUpdate:  "ACMEDEV_NO_UPDATE_CHECK",
		EnvPrefix:    "ACMEDEV_",
		LogFilePrefix: "acmedev-",
		TempPrefix:   ".acmedev-tmp-",
		GitHubOwner:  "acme-corp",
		GitHubRepo:   "acmedev",
	})

	bootstrap.Configure(
		bootstrap.WithSteps(
			devenv.InstallDevenvStep(),
			devenv.InstallDirenvStep(),
			claudecode.InstallClaudeStep(),
		),
	)

	devenv.Configure(
		devenv.WithDefaultLanguages("go", "python"),
		devenv.WithDirenv(true),
	)
	claudecode.Configure(
		claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard),
	)
	devinit.Configure(
		devinit.WithDetectProjectType(true),
	)

	instance.AddCommands(acmeHelloCmd())

	cmd.Main()
}

func acmeHelloCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "A custom acmedev command",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Hello from acmedev!")
		},
	}
}
