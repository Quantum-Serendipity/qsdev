package instance

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
)

// TestMain lets a test re-execute the test binary as `<exe> sandbox shim …`
// and reach the real Main, so the dispatch is tested where it lives.
func TestMain(m *testing.M) {
	if shim.Invoked(os.Args) {
		Main()
	}
	os.Exit(m.Run())
}
