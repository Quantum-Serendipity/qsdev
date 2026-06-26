package mcpserve

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// toolFilter is installed via mcp-go's server.WithToolFilter and runs on every
// tools/list request. mcp-go passes the full set of mounted tools; this returns
// the subset visible to the requesting client under the mount-all-then-filter
// model:
//
//   - In --multi-adapter mode every mounted tool is visible (the testing and
//     diagnostics mode, where the client may identify as none of the
//     frameworks).
//   - Otherwise generic (non-adapter) tools are ALWAYS visible, and a
//     framework-owned tool is visible only when the client matches that
//     framework, as computed by AdapterRegistry.DetectFrameworks. When no
//     framework matches, only generic tools remain — generic-only fallback mode.
//
// A tool with no recorded owner is treated as visible (fail open): every tool
// the server mounts is recorded in the catalog, so this only guards against an
// untracked tool and never hides functionality the client already has access to.
//
// Resource filtering is intentionally not performed here: mcp-go exposes a tool
// filter but no resource-list filter, so a true per-client resource filter is not
// implementable with this library and all mounted resources are listable by every
// client (see mountResource). The compensating security control on resources is
// redaction + audit on READ — every resource read is routed through the
// middleware chain (see resourceReadHandler) — not list-time visibility scoping.
func (s *Server) toolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	if s.multiAdapter {
		return tools
	}

	client := clientInfoFromContext(ctx)
	matched := make(map[string]bool)
	for _, a := range s.adapters.DetectFrameworks(client) {
		matched[string(a.ID())] = true
	}

	out := make([]mcp.Tool, 0, len(tools))
	for _, t := range tools {
		owner, ok := s.catalog.toolOwnerOf(t.Name)
		if !ok || owner == genericOwner || matched[owner] {
			out = append(out, t)
		}
	}
	return out
}
