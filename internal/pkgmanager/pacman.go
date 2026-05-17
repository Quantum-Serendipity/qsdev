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

// aurHelper returns the best available AUR helper, or empty string if none found.
func (p *Pacman) aurHelper() string {
	if _, err := p.runner.LookPath("paru"); err == nil {
		return "paru"
	}
	if _, err := p.runner.LookPath("yay"); err == nil {
		return "yay"
	}
	return ""
}

func (p *Pacman) UpdateIndex(ctx context.Context) error {
	return p.runner.Run(ctx, "pacman", "-Sy")
}

func (p *Pacman) Install(ctx context.Context, packages ...string) error {
	args := append([]string{"-S", "--noconfirm"}, packages...)
	return p.runner.Run(ctx, "pacman", args...)
}

func (p *Pacman) IsInstalled(ctx context.Context, pkg string) bool {
	err := p.runner.Run(ctx, "pacman", "-Qi", pkg)
	return err == nil
}

func (p *Pacman) SearchCmd() string {
	if h := p.aurHelper(); h != "" {
		return h + " -Ss"
	}
	return "pacman -Ss"
}
