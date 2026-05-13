package pkgmanager

import (
	"context"
	"strings"
)

// Apt implements PackageManager for Debian/Ubuntu systems using apt-get.
type Apt struct {
	runner CommandRunner
}

// NewApt creates an Apt package manager. If runner is nil, DefaultRunner() is used.
func NewApt(runner CommandRunner) *Apt {
	return &Apt{runner: ensureRunner(runner)}
}

func (a *Apt) Name() string { return "apt" }

func (a *Apt) Available() bool {
	_, err := a.runner.LookPath("apt-get")
	return err == nil
}

func (a *Apt) NeedsElevation() bool { return true }

func (a *Apt) UpdateIndex(ctx context.Context) error {
	return a.runner.Run(ctx, "apt-get", "update")
}

func (a *Apt) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install", "-y"}, packages...)
	return a.runner.Run(ctx, "apt-get", args...)
}

func (a *Apt) IsInstalled(pkg string) bool {
	out, err := a.runner.Output(context.Background(), "dpkg", "-l", pkg)
	if err != nil {
		return false
	}
	// dpkg -l output has "ii" prefix for installed packages.
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ii") {
			return true
		}
	}
	return false
}

func (a *Apt) SearchCmd() string { return "apt-cache search" }
