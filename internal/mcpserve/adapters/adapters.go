// Package adapters lists the framework adapters the universal qsdev MCP server
// ships. It is the single source of that set: the qsdev entry point registers
// All() into spi.DefaultRegistry(), and the mcpserve integration tests register
// the same list, so an adapter added here is both served and exercised.
//
// The concrete adapters delegate to addon packages (e.g. addons/claudecode), so
// neither they nor this package may be imported by the mcpserve server package
// itself — that would create an import cycle. Wiring stays explicit (no init()
// self-registration): callers register the returned adapters themselves.
package adapters

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// All returns a fresh instance of every shipped framework adapter, in
// registration order.
func All() []spi.FrameworkAdapter {
	return []spi.FrameworkAdapter{
		claudecode.New(),
		cline.New(),
		codex.New(),
		cursor.New(),
		windsurf.New(),
	}
}
