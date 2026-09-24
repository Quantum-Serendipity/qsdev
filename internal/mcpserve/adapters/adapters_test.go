package adapters

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestAllRegistersCleanly verifies the shipped adapter list has no duplicate
// framework ids, so both the entry point and the mcpserve test harness can
// register it into a registry without error.
func TestAllRegistersCleanly(t *testing.T) {
	t.Parallel()
	all := All()
	if len(all) == 0 {
		t.Fatal("All() returned no adapters")
	}
	reg := spi.NewAdapterRegistry()
	for _, a := range all {
		if err := reg.Register(a); err != nil {
			t.Errorf("registering adapter %q: %v", a.ID(), err)
		}
	}
	if got := len(reg.All()); got != len(all) {
		t.Errorf("registry holds %d adapters, want %d", got, len(all))
	}
}
