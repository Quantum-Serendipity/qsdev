// Package pkgmanager provides a unified interface for system package managers
// and a registry mapping tool names to platform-specific package names.
package pkgmanager

import "context"

// PackageManager abstracts system package manager operations.
type PackageManager interface {
	// Name returns the canonical package manager identifier (e.g. "apt",
	// "brew", "nix"). Package-name lookups (PackageFor, ResolvePackageName)
	// are keyed by this name.
	Name() string

	// Available reports whether this package manager is installed on the system.
	Available() bool

	// NeedsElevation reports whether this package manager requires root privileges.
	// The caller is responsible for wrapping commands with sudo; implementations
	// must NOT prepend sudo themselves.
	NeedsElevation() bool

	// InstallArgs returns the binary and arguments that Install runs for the
	// given packages, without any sudo prefix. It is the single source of the
	// install command line: callers that must run the install themselves (for
	// example under sudo) or display it use this rather than rebuilding it.
	// Managers that install one package per invocation (winget, nix) run it
	// once per package, so callers describing an install pass one package.
	InstallArgs(packages ...string) (bin string, args []string)

	// Install installs one or more packages.
	Install(ctx context.Context, packages ...string) error
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	// LookPath searches for an executable in PATH.
	LookPath(name string) (string, error)

	// Run executes a command, inheriting stdout/stderr.
	Run(ctx context.Context, name string, args ...string) error
}
