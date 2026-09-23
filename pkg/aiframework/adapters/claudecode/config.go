package claudecode

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Capabilities reports which configuration aspects the Claude Code renderer
// covers (all expressed through .claude/settings.json and .mcp.json).
func (a *Adapter) Capabilities() aiframework.ConfigCapabilities {
	return a.addon.Capabilities()
}

// Format reports the on-disk format of the rendered configuration files.
func (a *Adapter) Format() string { return a.addon.Format() }

// Render produces the Claude Code configuration files (.claude/settings.json
// and .mcp.json) for the given policy. The rendered settings always carry the
// self-protection hook, and the policy's permission rules, sandbox
// constraints, and MCP servers are all represented or rejected with an error.
func (a *Adapter) Render(ctx context.Context, input *aiframework.PolicyInput) ([]types.GeneratedFile, error) {
	return a.addon.Render(ctx, input)
}

// Validate checks that every rendered JSON file is well-formed, reporting an
// error-severity issue for any malformed content.
func (a *Adapter) Validate(ctx context.Context, files []types.GeneratedFile) []aiframework.ValidationIssue {
	return a.addon.Validate(ctx, files)
}
