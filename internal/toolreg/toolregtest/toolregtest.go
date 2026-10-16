// Package toolregtest holds test helpers for code that uses the default tool
// registry.
package toolregtest

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// Default returns the default tool registry, failing the test if it cannot
// be built.
func Default(tb testing.TB) *toolreg.Registry {
	tb.Helper()
	r, err := toolreg.Default()
	if err != nil {
		tb.Fatalf("building the default tool registry: %v", err)
	}
	return r
}
