//go:build windows

package pkgmanager

import (
	"context"
)

// Scoop implements PackageManager for Windows using Scoop.
type Scoop struct {
	runner CommandRunner
}

// NewScoop creates a Scoop package manager. If runner is nil, DefaultRunner() is used.
func NewScoop(runner CommandRunner) *Scoop {
	return &Scoop{runner: ensureRunner(runner)}
}

func (s *Scoop) Name() string { return "scoop" }

func (s *Scoop) Available() bool {
	_, err := s.runner.LookPath("scoop")
	return err == nil
}

func (s *Scoop) NeedsElevation() bool { return false }

func (s *Scoop) Install(ctx context.Context, packages ...string) error {
	bin, args := s.InstallArgs(packages...)
	return s.runner.Run(ctx, bin, args...)
}
