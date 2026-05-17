//go:build !windows

package pkgmanager

import (
	"context"
	"fmt"
)

// Winget is a stub for non-Windows platforms.
type Winget struct{}

// NewWinget returns a stub Winget. The runner parameter is ignored.
func NewWinget(_ CommandRunner) *Winget { return &Winget{} }

func (w *Winget) Name() string                                    { return "winget" }
func (w *Winget) Available() bool                                  { return false }
func (w *Winget) NeedsElevation() bool                             { return false }
func (w *Winget) UpdateIndex(_ context.Context) error              { return nil }
func (w *Winget) Install(_ context.Context, _ ...string) error     { return fmt.Errorf("winget not available") }
func (w *Winget) IsInstalled(_ context.Context, _ string) bool                        { return false }
func (w *Winget) SearchCmd() string                                { return "" }
