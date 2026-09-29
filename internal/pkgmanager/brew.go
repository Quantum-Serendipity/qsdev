package pkgmanager

import (
	"context"
)

// Brew implements PackageManager for macOS and Linux using Homebrew.
type Brew struct {
	runner CommandRunner
}

// NewBrew creates a Brew package manager. If runner is nil, DefaultRunner() is used.
func NewBrew(runner CommandRunner) *Brew {
	return &Brew{runner: ensureRunner(runner)}
}

func (b *Brew) Name() string { return "brew" }

func (b *Brew) Available() bool {
	_, err := b.runner.LookPath("brew")
	return err == nil
}

func (b *Brew) NeedsElevation() bool { return false }

// InstallArgs returns the command Install runs, without sudo.
func (b *Brew) InstallArgs(packages ...string) (string, []string) {
	return "brew", append([]string{"install"}, packages...)
}

func (b *Brew) Install(ctx context.Context, packages ...string) error {
	bin, args := b.InstallArgs(packages...)
	return b.runner.Run(ctx, bin, args...)
}
