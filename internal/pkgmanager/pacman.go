package pkgmanager

import (
	"context"
)

// Pacman implements PackageManager for Arch Linux using pacman.
// It detects AUR helpers (paru > yay) for extended package availability.
type Pacman struct {
	runner CommandRunner
}

// NewPacman creates a Pacman package manager. If runner is nil, DefaultRunner() is used.
func NewPacman(runner CommandRunner) *Pacman {
	return &Pacman{runner: ensureRunner(runner)}
}

func (p *Pacman) Name() string { return "pacman" }

func (p *Pacman) Available() bool {
	_, err := p.runner.LookPath("pacman")
	return err == nil
}

func (p *Pacman) NeedsElevation() bool { return true }

func (p *Pacman) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"-S", "--noconfirm"}, packages...)
	return p.runner.Run(ctx, "pacman", args...)
}
