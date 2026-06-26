package mcpserve_test

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestMain registers the framework adapters into spi.DefaultRegistry() once for
// the whole test binary, mirroring the explicit wiring cmd/qsdev/main.go performs
// (the adapters no longer self-register via init()). The server under test reads
// the default registry, so the integration tests in this package need it
// populated before they construct a server.
func TestMain(m *testing.M) {
	reg := spi.DefaultRegistry()
	for _, a := range []spi.FrameworkAdapter{
		claudecode.New(), cline.New(), codex.New(), cursor.New(), windsurf.New(),
	} {
		if err := reg.Register(a); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}
