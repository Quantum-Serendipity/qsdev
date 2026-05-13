//go:build windows

package pkgmanager

import (
	"context"
)

// Winget implements PackageManager for Windows using winget.
type Winget struct {
	runner CommandRunner
}

// NewWinget creates a Winget package manager. If runner is nil, DefaultRunner() is used.
func NewWinget(runner CommandRunner) *Winget {
	return &Winget{runner: ensureRunner(runner)}
}

func (w *Winget) Name() string { return "winget" }

func (w *Winget) Available() bool {
	_, err := w.runner.LookPath("winget")
	return err == nil
}

func (w *Winget) NeedsElevation() bool { return false }

func (w *Winget) UpdateIndex(ctx context.Context) error {
	return w.runner.Run(ctx, "winget", "source", "update")
}

func (w *Winget) Install(ctx context.Context, packages ...string) error {
	for _, pkg := range packages {
		err := w.runner.Run(ctx, "winget", "install",
			"--id", pkg, "-e",
			"--accept-source-agreements",
			"--accept-package-agreements",
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Winget) IsInstalled(pkg string) bool {
	err := w.runner.Run(context.Background(), "winget", "list", "--id", pkg, "-e")
	return err == nil
}

func (w *Winget) SearchCmd() string { return "winget search" }
