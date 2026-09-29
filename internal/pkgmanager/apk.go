package pkgmanager

import (
	"context"
)

// Apk implements PackageManager for Alpine Linux.
type Apk struct {
	runner CommandRunner
}

// NewApk creates an Apk package manager. If runner is nil, DefaultRunner() is used.
func NewApk(runner CommandRunner) *Apk {
	return &Apk{runner: ensureRunner(runner)}
}

func (a *Apk) Name() string { return "apk" }

func (a *Apk) Available() bool {
	_, err := a.runner.LookPath("apk")
	return err == nil
}

func (a *Apk) NeedsElevation() bool { return true }

// InstallArgs returns the command Install runs, without sudo.
func (a *Apk) InstallArgs(packages ...string) (string, []string) {
	return "apk", append([]string{"add"}, packages...)
}

func (a *Apk) Install(ctx context.Context, packages ...string) error {
	bin, args := a.InstallArgs(packages...)
	return a.runner.Run(ctx, bin, args...)
}
