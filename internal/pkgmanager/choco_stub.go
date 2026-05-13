//go:build !windows

package pkgmanager

import (
	"context"
	"fmt"
)

// Choco is a stub for non-Windows platforms.
type Choco struct{}

// NewChoco returns a stub Choco. The runner parameter is ignored.
func NewChoco(_ CommandRunner) *Choco { return &Choco{} }

func (c *Choco) Name() string                                    { return "choco" }
func (c *Choco) Available() bool                                  { return false }
func (c *Choco) NeedsElevation() bool                             { return false }
func (c *Choco) UpdateIndex(_ context.Context) error              { return nil }
func (c *Choco) Install(_ context.Context, _ ...string) error     { return fmt.Errorf("choco not available") }
func (c *Choco) IsInstalled(_ string) bool                        { return false }
func (c *Choco) SearchCmd() string                                { return "" }
