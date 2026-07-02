package pkgmanager

import (
	"context"
)

// Xbps implements PackageManager for Void Linux.
type Xbps struct {
	runner CommandRunner
}

// NewXbps creates an Xbps package manager. If runner is nil, DefaultRunner() is used.
func NewXbps(runner CommandRunner) *Xbps {
	return &Xbps{runner: ensureRunner(runner)}
}

func (x *Xbps) Name() string { return "xbps" }

func (x *Xbps) Available() bool {
	_, err := x.runner.LookPath("xbps-install")
	return err == nil
}

func (x *Xbps) NeedsElevation() bool { return true }

func (x *Xbps) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"-y"}, packages...)
	return x.runner.Run(ctx, "xbps-install", args...)
}
