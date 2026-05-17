package pkgmanager

import (
	"context"
)

// Zypper implements PackageManager for openSUSE/SLES systems.
type Zypper struct {
	runner CommandRunner
}

// NewZypper creates a Zypper package manager. If runner is nil, DefaultRunner() is used.
func NewZypper(runner CommandRunner) *Zypper {
	return &Zypper{runner: ensureRunner(runner)}
}

func (z *Zypper) Name() string { return "zypper" }

func (z *Zypper) Available() bool {
	_, err := z.runner.LookPath("zypper")
	return err == nil
}

func (z *Zypper) NeedsElevation() bool { return true }

func (z *Zypper) UpdateIndex(ctx context.Context) error {
	return z.runner.Run(ctx, "zypper", "refresh")
}

func (z *Zypper) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install", "-y"}, packages...)
	return z.runner.Run(ctx, "zypper", args...)
}

func (z *Zypper) IsInstalled(ctx context.Context, pkg string) bool {
	err := z.runner.Run(ctx, "zypper", "se", "--installed-only", "-x", pkg)
	return err == nil
}

func (z *Zypper) SearchCmd() string { return "zypper search" }
