package bwrap

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
)

// TestMain makes the test binary the in-sandbox shim: RunHook runs the host
// executable (here, this test binary) as `sandbox shim --ready-fd N -- hook`,
// so the dispatch must come before the tests run, as in instance.Main.
func TestMain(m *testing.M) {
	if shim.Invoked(os.Args) {
		os.Exit(shim.Main(os.Args, os.Stderr))
	}
	os.Exit(m.Run())
}
