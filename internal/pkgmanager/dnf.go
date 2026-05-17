package pkgmanager

import (
	"context"
)

// Dnf implements PackageManager for RHEL/Fedora systems using dnf (with yum fallback).
type Dnf struct {
	runner CommandRunner
}

// NewDnf creates a Dnf package manager. If runner is nil, DefaultRunner() is used.
func NewDnf(runner CommandRunner) *Dnf {
	return &Dnf{runner: ensureRunner(runner)}
}

func (d *Dnf) Name() string { return "dnf" }

func (d *Dnf) Available() bool {
	_, err := d.runner.LookPath("dnf")
	if err == nil {
		return true
	}
	_, err = d.runner.LookPath("yum")
	return err == nil
}

func (d *Dnf) NeedsElevation() bool { return true }

// cmd returns the actual binary to use: dnf if available, otherwise yum.
func (d *Dnf) cmd() string {
	if _, err := d.runner.LookPath("dnf"); err == nil {
		return "dnf"
	}
	return "yum"
}

func (d *Dnf) UpdateIndex(ctx context.Context) error {
	return d.runner.Run(ctx, d.cmd(), "makecache")
}

func (d *Dnf) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install", "-y"}, packages...)
	return d.runner.Run(ctx, d.cmd(), args...)
}

func (d *Dnf) IsInstalled(ctx context.Context, pkg string) bool {
	err := d.runner.Run(ctx, "rpm", "-q", pkg)
	return err == nil
}

func (d *Dnf) SearchCmd() string { return d.cmd() + " search" }
