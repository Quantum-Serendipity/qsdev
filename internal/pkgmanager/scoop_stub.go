//go:build !windows

package pkgmanager

import (
	"context"
	"fmt"
)

// Scoop is a stub for non-Windows platforms.
type Scoop struct{}

// NewScoop returns a stub Scoop. The runner parameter is ignored.
func NewScoop(_ CommandRunner) *Scoop { return &Scoop{} }

func (s *Scoop) Name() string                                    { return "scoop" }
func (s *Scoop) Available() bool                                  { return false }
func (s *Scoop) NeedsElevation() bool                             { return false }
func (s *Scoop) UpdateIndex(_ context.Context) error              { return nil }
func (s *Scoop) Install(_ context.Context, _ ...string) error     { return fmt.Errorf("scoop not available") }
func (s *Scoop) IsInstalled(_ string) bool                        { return false }
func (s *Scoop) SearchCmd() string                                { return "" }
