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

func (x *Xbps) UpdateIndex(ctx context.Context) error {
	return x.runner.Run(ctx, "xbps-install", "-S")
}

func (x *Xbps) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"-y"}, packages...)
	return x.runner.Run(ctx, "xbps-install", args...)
}

func (x *Xbps) IsInstalled(ctx context.Context, pkg string) bool {
	err := x.runner.Run(ctx, "xbps-query", pkg)
	return err == nil
}

func (x *Xbps) SearchCmd() string { return "xbps-query -Rs" }
