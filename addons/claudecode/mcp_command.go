package claudecode

import (
	"context"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run embedded MCP servers",
		Long:  "Start embedded MCP servers that communicate via stdio transport.",
	}

	for _, provider := range mcpserver.DefaultRegistry().All() {
		cmd.AddCommand(mcpServerCmd(provider))
	}

	// The universal MCP server (internal/mcpserve) is the single allowed
	// dependency edge from addons/claudecode into mcpserve. mcpserve must never
	// import back into addons/claudecode (see internal/mcpserve/doc.go).
	cmd.AddCommand(mcpserve.Command())

	cmd.AddCommand(mcpStatusCmd())
	cmd.AddCommand(mcpListCmd())
	cmd.AddCommand(mcpGradeCmd())
	cmd.AddCommand(mcpInstallCmd())
	cmd.AddCommand(mcpUpdateCmd())
	cmd.AddCommand(mcpRemoveCmd())
	cmd.AddCommand(mcpHealthCmd())
	return cmd
}

func mcpServerCmd(provider mcpserver.Provider) *cobra.Command {
	return &cobra.Command{
		Use:   provider.Name(),
		Short: provider.Description(),
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.NewMCPServer(
				"qsdev-"+provider.Name(),
				version.Info().Version,
			)

			for _, tool := range provider.Tools() {
				srv.AddTool(buildMCPTool(tool), buildMCPHandler(tool))
			}

			return server.ServeStdio(srv)
		},
	}
}

func buildMCPTool(def mcpserver.ToolDef) mcp.Tool {
	opts := []mcp.ToolOption{
		mcp.WithDescription(def.Description),
	}
	for _, param := range def.Params {
		paramOpts := []mcp.PropertyOption{
			mcp.Description(param.Description),
		}
		if param.Required {
			paramOpts = append(paramOpts, mcp.Required())
		}
		opts = append(opts, mcp.WithString(param.Name, paramOpts...))
	}
	return mcp.NewTool(def.Name, opts...)
}

func buildMCPHandler(def mcpserver.ToolDef) server.ToolHandlerFunc {
	handler := def.Handler
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		if args == nil {
			args = make(map[string]any)
		}

		result, err := handler(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

func registerMCPProviders() {
	reg := mcpserver.DefaultRegistry()

	reg.Register(newPostmortemProvider())
	reg.Register(newVersionSentinelProvider())

	// Wire external-attestation verification into the compliance grader. This is
	// the only place mcpregistry and contentsign are connected (mcpregistry must
	// not import contentsign, to avoid an import cycle).
	mcpregistry.AttestationChecker = attestationChecker
}

// attestationStore caches the trusted-keys-backed attestation store. The keys
// are loaded once on first use (lazily, to keep disk I/O out of init), so
// grading many servers in one `mcp grade` run does not re-read and re-parse the
// trusted-keys directory per server.
var (
	attestationStoreOnce sync.Once
	attestationStore     contentsign.AttestationStore
)

// attestationChecker reports whether a server command has a trusted-key
// attestation, reusing a single loaded key set across invocations.
func attestationChecker(def *mcpregistry.McpServerDefinition) bool {
	attestationStoreOnce.Do(func() {
		// A load failure (incl. a missing keys dir) yields no keys; with no
		// trusted keys nothing can be attested, so we simply report false below.
		keys, _ := contentsign.LoadTrustedKeys("")
		attestationStore = contentsign.AttestationStore{TrustedKeys: keys}
	})
	if len(attestationStore.TrustedKeys) == 0 {
		// Short-circuit: an empty TrustedKeys set would otherwise make
		// AttestationStore reload the keys directory on every IsAttested call.
		return false
	}
	return attestationStore.IsAttested(context.Background(), def.Command)
}

func init() {
	registerMCPProviders()
}
