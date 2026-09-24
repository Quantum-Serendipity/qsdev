package pkgmanager

import "context"

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

// InstallArgs returns the command Install runs, without sudo.
func (a *Apt) InstallArgs(packages ...string) (string, []string) {
	return "apt-get", append([]string{"install", "-y"}, packages...)
}

func (a *Apt) Install(ctx context.Context, packages ...string) error {
	bin, args := a.InstallArgs(packages...)
	return a.runner.Run(ctx, bin, args...)
}
