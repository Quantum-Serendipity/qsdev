// Package instance is the entry point for building tools on the qsdev framework.
// Downstream tools import this package and call its functions during initialization,
// before cmd.Main() is invoked.
//
// Initialization order:
//  1. SetBranding — configure app name, env vars, file paths, GitHub coordinates
//  2. Addon Configure() calls — devenv.Configure(), claudecode.Configure(), etc.
//  3. AddCommands / AddCommandBuilders — register custom CLI commands
//  4. cmd.Main() — starts the application
//
// All customization must happen before cmd.Main() is called. The gdev lifecycle
// enforces this: calls to SetBranding or AddEcosystemModules after lockdown will panic.
package instance

import (
	"github.com/spf13/cobra"

	gdevinstance "fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// SetBranding configures all brand-specific naming for this *dev tool.
// Must be called before any addon Configure() or cmd.Main().
func SetBranding(cfg branding.Config) {
	gdevinstance.CheckCanCustomize()
	branding.Set(cfg)
	gdevinstance.SetAppName(cfg.AppName)
}

// EcosystemRegistry returns the default ecosystem module registry.
// Downstream tools can Register() additional modules on this registry.
func EcosystemRegistry() *ecosystem.Registry {
	return ecosystem.DefaultRegistry()
}

// AddEcosystemModules registers one or more ecosystem modules into the
// default registry. Must be called before cmd.Main().
func AddEcosystemModules(modules ...ecosystem.EcosystemModule) {
	gdevinstance.CheckCanCustomize()
	for _, m := range modules {
		ecosystem.MustRegisterModule(m)
	}
}

// SetVersionOverride sets a custom version and commit for the binary.
func SetVersionOverride(version, commit string) {
	gdevinstance.SetVersionOverride(version, commit)
}

// AddCommands adds cobra commands to the root command tree.
func AddCommands(cmds ...*cobra.Command) {
	gdevinstance.AddCommands(cmds...)
}

// AddCommandBuilders adds deferred command builders to the root command tree.
func AddCommandBuilders(fns ...func() *cobra.Command) {
	gdevinstance.AddCommandBuilders(fns...)
}
