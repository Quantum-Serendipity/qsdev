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

func (s *Scoop) UpdateIndex(ctx context.Context) error {
	return s.runner.Run(ctx, "scoop", "update")
}

func (s *Scoop) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install"}, packages...)
	return s.runner.Run(ctx, "scoop", args...)
}

func (s *Scoop) IsInstalled(pkg string) bool {
	err := s.runner.Run(context.Background(), "scoop", "info", pkg)
	return err == nil
}

func (s *Scoop) SearchCmd() string { return "scoop search" }
