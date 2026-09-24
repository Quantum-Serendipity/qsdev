package mcpserve_test

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestMain registers the framework adapters into spi.DefaultRegistry() once for
// the whole test binary, using the same adapters.All() list instance/runtime.go
// registers (the adapters do not self-register via init()), so every shipped
// adapter is exercised here. The server under test reads the default registry,
// so the integration tests in this package need it populated before they
// construct a server.
func TestMain(m *testing.M) {
	reg := spi.DefaultRegistry()
	for _, a := range adapters.All() {
		if err := reg.Register(a); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}
