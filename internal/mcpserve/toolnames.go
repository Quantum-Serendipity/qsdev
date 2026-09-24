package mcpserve

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
)

// MountableToolNames returns the sorted, de-duplicated names of every tool the
// universal MCP server can mount: the generic project context tools, the
// security/devenv/status tools, and the tools of each adapter in adapters
// (every one of them is mounted in multi-adapter mode, so all count). It is the
// namespace of mcp.disabled_tools, which `qsdev check` validates against it.
//
// adapters is passed in rather than read from spi.DefaultRegistry because the
// concrete adapters import addon packages that this package must not import;
// callers pass adapters.All(), the list the qsdev entry point registers.
func MountableToolNames(adapters []spi.FrameworkAdapter) []string {
	names := projectctx.ToolNames()
	// The registrations are only inspected for their names, so they are built
	// unbound: no project root and no enforced policy.
	for _, r := range tools.All("", nil) {
		names = append(names, r.Name)
	}
	for _, a := range adapters {
		for _, r := range a.Tools() {
			names = append(names, r.Name)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}
