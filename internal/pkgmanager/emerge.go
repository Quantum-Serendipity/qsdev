package pkgmanager

import (
	"context"
)

// Emerge implements PackageManager for Gentoo Linux using Portage.
// Package names use the category/package format (e.g. "dev-lang/go").
type Emerge struct {
	runner CommandRunner
}

// NewEmerge creates an Emerge package manager. If runner is nil, DefaultRunner() is used.
func NewEmerge(runner CommandRunner) *Emerge {
	return &Emerge{runner: ensureRunner(runner)}
}

func (e *Emerge) Name() string { return "emerge" }

func (e *Emerge) Available() bool {
	_, err := e.runner.LookPath("emerge")
	return err == nil
}

func (e *Emerge) NeedsElevation() bool { return true }

// InstallArgs returns the command Install runs, without sudo.
func (e *Emerge) InstallArgs(packages ...string) (string, []string) {
	return "emerge", append([]string{"--ask=n"}, packages...)
}

func (e *Emerge) Install(ctx context.Context, packages ...string) error {
	bin, args := e.InstallArgs(packages...)
	return e.runner.Run(ctx, bin, args...)
}
