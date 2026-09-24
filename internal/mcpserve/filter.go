package mcpserve

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
// The same visibility is enforced at call time by enforceToolVisibility, so a
// client cannot call a tool it was never listed. This is relevance scoping, not
// an authorization boundary: the match is based on the client's self-asserted
// clientInfo name, so any client can claim to be a given framework. Protection
// for sensitive tools comes from the Guardrail/category policy in the chain.
//
// Resource filtering is intentionally not performed here: mcp-go exposes a tool
// filter but no resource-list filter, so a true per-client resource filter is not
// implementable with this library and all mounted resources are listable by every
// client (see mountResource). The compensating security control on resources is
// redaction + audit on READ — every resource read is routed through the
// middleware chain (see resourceReadHandler) — not list-time visibility scoping.
func (s *Server) toolFilter(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	visible := s.toolVisibility(ctx)
	out := make([]mcp.Tool, 0, len(tools))
	for _, t := range tools {
		if visible(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// toolVisibility returns the predicate deciding which tools the client in ctx
// may see and call (see toolFilter for the rules).
func (s *Server) toolVisibility(ctx context.Context) func(name string) bool {
	if s.multiAdapter {
		return func(string) bool { return true }
	}
	matched := make(map[string]bool)
	for _, a := range s.adapters.DetectFrameworks(clientInfoFromContext(ctx)) {
		matched[string(a.ID())] = true
	}
	return func(name string) bool {
		owner, ok := s.catalog.toolOwnerOf(name)
		return !ok || owner == genericOwner || matched[owner]
	}
}

// enforceToolVisibility is a tool-handler middleware that applies toolFilter's
// visibility at call time: mcp-go consults tool filters only for tools/list, so
// without it a client could call a framework tool it was never shown by naming
// it directly. Such a call is rejected with a tool error before the handler or
// middleware chain runs.
func (s *Server) enforceToolVisibility(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.Params.Name
		if !s.toolVisibility(ctx)(name) {
			slog.Warn("rejected call to a tool not visible to this client",
				"tool", name, "client", clientInfoFromContext(ctx).Name)
			return mcp.NewToolResultError(fmt.Sprintf("tool %q is not available to this client", name)), nil
		}
		return next(ctx, req)
	}
}
