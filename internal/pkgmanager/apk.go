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

func (a *Apk) UpdateIndex(ctx context.Context) error {
	return a.runner.Run(ctx, "apk", "update")
}

func (a *Apk) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"add"}, packages...)
	return a.runner.Run(ctx, "apk", args...)
}

func (a *Apk) IsInstalled(pkg string) bool {
	err := a.runner.Run(context.Background(), "apk", "info", "-e", pkg)
	return err == nil
}

func (a *Apk) SearchCmd() string { return "apk search" }
