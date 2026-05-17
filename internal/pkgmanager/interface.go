// Package pkgmanager provides a unified interface for system package managers
// and a registry mapping tool names to platform-specific package names.
package pkgmanager

import "context"

// PackageManager abstracts system package manager operations.
type PackageManager interface {
	// Name returns the package manager identifier (e.g. "apt", "brew", "nix").
	Name() string

	// Available reports whether this package manager is installed on the system.
	Available() bool

	// NeedsElevation reports whether this package manager requires root privileges.
	// The caller is responsible for wrapping commands with sudo; implementations
	// must NOT prepend sudo themselves.
	NeedsElevation() bool

	// UpdateIndex refreshes the package index/cache.
	UpdateIndex(ctx context.Context) error

	// Install installs one or more packages.
	Install(ctx context.Context, packages ...string) error

	// IsInstalled reports whether a package is currently installed.
	IsInstalled(ctx context.Context, pkg string) bool

	// SearchCmd returns the shell command prefix for searching packages,
	// e.g. "apt-cache search" or "brew search".
	SearchCmd() string
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	// LookPath searches for an executable in PATH.
	LookPath(name string) (string, error)

	// Run executes a command, inheriting stdout/stderr.
	Run(ctx context.Context, name string, args ...string) error

	// Output executes a command and returns its combined output.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}
