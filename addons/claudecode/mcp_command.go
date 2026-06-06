package claudecode

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

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

	cmd.AddCommand(mcpStatusCmd())
	cmd.AddCommand(mcpListCmd())
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
}

func init() {
	registerMCPProviders()
}
