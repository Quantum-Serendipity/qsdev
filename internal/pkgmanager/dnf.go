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

// InstallArgs returns the command Install runs, without sudo.
func (d *Dnf) InstallArgs(packages ...string) (string, []string) {
	return d.cmd(), append([]string{"install", "-y"}, packages...)
}

func (d *Dnf) Install(ctx context.Context, packages ...string) error {
	bin, args := d.InstallArgs(packages...)
	return d.runner.Run(ctx, bin, args...)
}
