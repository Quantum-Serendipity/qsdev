// Package instance is the entry point for building tools on the qsdev framework.
// Downstream tools import this package and call its functions during initialization,
// then call Main.
//
// Initialization order:
//  1. SetBranding — configure app name, env vars, file paths, GitHub coordinates
//  2. Addon Configure() calls — devenv.Configure(), claudecode.Configure(), etc.
//  3. AddCommands / AddCommandBuilders — register custom CLI commands
//  4. Main — installs the default runtime (DefaultRuntime) and starts the
//     application (use it rather than gdev's cmd.Main)
//
// All customization must happen before Main is called. The gdev lifecycle
// enforces this: calls to SetBranding or AddEcosystemModules after lockdown will panic.
package instance

import (
	"sync/atomic"

	"github.com/spf13/cobra"

	gdevinstance "fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// SetBranding configures all brand-specific naming for this *dev tool.
// Must be called before any addon Configure() or Main().
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
// default registry. Must be called before Main().
func AddEcosystemModules(modules ...ecosystem.EcosystemModule) {
	gdevinstance.CheckCanCustomize()
	for _, m := range modules {
		ecosystem.MustRegisterModule(m)
	}
}

// versionOverridden records that the tool set its own version with
// SetVersionOverride, which then takes precedence over the version stamped
// into VersionPackage (see ApplyBuildVersion).
var versionOverridden atomic.Bool

// SetVersionOverride sets a custom version and commit for the binary. It
// takes precedence over the version stamped into VersionPackage.
func SetVersionOverride(version, commit string) {
	gdevinstance.SetVersionOverride(version, commit)
	versionOverridden.Store(true)
}

// AddCommands adds cobra commands to the root command tree.
func AddCommands(cmds ...*cobra.Command) {
	gdevinstance.AddCommands(cmds...)
}

// AddCommandBuilders adds deferred command builders to the root command tree.
func AddCommandBuilders(fns ...func() *cobra.Command) {
	gdevinstance.AddCommandBuilders(fns...)
}
