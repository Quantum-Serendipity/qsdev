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

func (e *Emerge) UpdateIndex(ctx context.Context) error {
	return e.runner.Run(ctx, "emerge", "--sync")
}

func (e *Emerge) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"--ask=n"}, packages...)
	return e.runner.Run(ctx, "emerge", args...)
}

func (e *Emerge) IsInstalled(pkg string) bool {
	err := e.runner.Run(context.Background(), "equery", "list", pkg)
	return err == nil
}

func (e *Emerge) SearchCmd() string { return "emerge --search" }
