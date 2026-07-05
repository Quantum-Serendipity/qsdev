//go:build windows

package pkgmanager

import (
	"context"
)

// Choco implements PackageManager for Windows using Chocolatey.
type Choco struct {
	runner CommandRunner
}

// NewChoco creates a Choco package manager. If runner is nil, DefaultRunner() is used.
func NewChoco(runner CommandRunner) *Choco {
	return &Choco{runner: ensureRunner(runner)}
}

func (c *Choco) Name() string { return "choco" }

func (c *Choco) Available() bool {
	_, err := c.runner.LookPath("choco")
	return err == nil
}

// NeedsElevation returns false because Chocolatey handles its own UAC elevation.
func (c *Choco) NeedsElevation() bool { return false }

func (c *Choco) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"install", "-y"}, packages...)
	return c.runner.Run(ctx, "choco", args...)
}
