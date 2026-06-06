package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/postmortem"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/internal/vsentinel"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run embedded MCP servers",
		Long:  "Start embedded MCP servers that communicate via stdio transport.",
	}
	cmd.AddCommand(mcpAgentPostmortemCmd())
	cmd.AddCommand(mcpVersionSentinelCmd())
	cmd.AddCommand(mcpStatusCmd())
	cmd.AddCommand(mcpListCmd())
	return cmd
}

// --- Agent Postmortem MCP Server ---

func mcpAgentPostmortemCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-postmortem",
		Short: "Run the agent-postmortem MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.NewMCPServer(
				"qsdev-agent-postmortem",
				version.Info().Version,
			)

			srv.AddTool(analyzeSessionTool(), handleAnalyzeSession)
			srv.AddTool(listFailurePatternsTool(), handleListFailurePatterns)
			srv.AddTool(generateVerificationChecklistTool(), handleGenerateVerificationChecklist)

			return server.ServeStdio(srv)
		},
	}
}

func analyzeSessionTool() mcp.Tool {
	return mcp.NewTool("analyze_session",
		mcp.WithDescription("Parse a Claude session JSONL file and return structured analysis including message counts, tool calls, and errors."),
		mcp.WithString("session_path", mcp.Description("Path to the session .jsonl file"), mcp.Required()),
	)
}

func handleAnalyzeSession(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionPath, err := request.RequireString("session_path")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: session_path"), nil
	}

	analysis, err := postmortem.ParseSessionJSONL(sessionPath)
	if err != nil {
		return mcp.NewToolResultErrorf("parsing session: %v", err), nil
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling result: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func listFailurePatternsTool() mcp.Tool {
	return mcp.NewTool("list_failure_patterns",
		mcp.WithDescription("Walk a directory of Claude session files, parse each, and aggregate failure patterns across sessions."),
		mcp.WithString("sessions_dir", mcp.Description("Directory to search for .jsonl session files (defaults to ~/.claude/projects/)")),
	)
}

func handleListFailurePatterns(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionsDir := request.GetString("sessions_dir", "")
	if sessionsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return mcp.NewToolResultErrorf("determining home directory: %v", err), nil
		}
		sessionsDir = filepath.Join(home, ".claude", "projects")
	}

	paths, err := postmortem.FindSessionFiles(sessionsDir)
	if err != nil {
		return mcp.NewToolResultErrorf("finding session files: %v", err), nil
	}

	var analyses []*postmortem.SessionAnalysis
	for _, p := range paths {
		a, err := postmortem.ParseSessionJSONL(p)
		if err != nil {
			continue
		}
		analyses = append(analyses, a)
	}

	report := postmortem.AggregateFailures(analyses)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling report: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func generateVerificationChecklistTool() mcp.Tool {
	return mcp.NewTool("generate_verification_checklist",
		mcp.WithDescription("Generate a project-specific verification checklist based on detected ecosystems and languages."),
	)
}

func handleGenerateVerificationChecklist(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return mcp.NewToolResultErrorf("determining project root: %v", err), nil
	}

	answers, err := loadAnswers(projectRoot)
	if err != nil {
		return mcp.NewToolResultText(`["go build ./...","go test ./...","go vet ./..."]`), nil
	}

	registry := ecosystem.DefaultRegistry()
	cmds := collectVerificationCommands(answers, registry)

	if len(cmds) == 0 {
		cmds = []string{"go build ./...", "go test ./...", "go vet ./..."}
	}

	data, err := json.MarshalIndent(cmds, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling checklist: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// --- Version Sentinel MCP Server ---

func mcpVersionSentinelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version-sentinel",
		Short: "Run the version-sentinel MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.NewMCPServer(
				"qsdev-version-sentinel",
				version.Info().Version,
			)

			srv.AddTool(checkVersionsTool(), handleCheckVersions)
			srv.AddTool(detectDriftTool(), handleDetectDrift)
			srv.AddTool(manifestCoverageTool(), handleManifestCoverage)
			srv.AddTool(versionHistoryTool(), handleVersionHistory)

			return server.ServeStdio(srv)
		},
	}
}

func checkVersionsTool() mcp.Tool {
	return mcp.NewTool("check_versions",
		mcp.WithDescription("Scan manifest files and report current dependency versions."),
		mcp.WithString("project_root", mcp.Description("Project root directory (defaults to current directory)")),
	)
}

func handleCheckVersions(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := request.GetString("project_root", "")
	if root == "" {
		var err error
		root, err = cmdutil.ProjectRoot()
		if err != nil {
			return mcp.NewToolResultErrorf("determining project root: %v", err), nil
		}
	}

	report, err := vsentinel.CheckVersions(root)
	if err != nil {
		return mcp.NewToolResultErrorf("checking versions: %v", err), nil
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling report: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func detectDriftTool() mcp.Tool {
	return mcp.NewTool("detect_drift",
		mcp.WithDescription("Compare lockfile entries against manifest declarations to detect version drift."),
		mcp.WithString("project_root", mcp.Description("Project root directory (defaults to current directory)")),
	)
}

func handleDetectDrift(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := request.GetString("project_root", "")
	if root == "" {
		var err error
		root, err = cmdutil.ProjectRoot()
		if err != nil {
			return mcp.NewToolResultErrorf("determining project root: %v", err), nil
		}
	}

	report, err := vsentinel.DetectDrift(root)
	if err != nil {
		return mcp.NewToolResultErrorf("detecting drift: %v", err), nil
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling report: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func manifestCoverageTool() mcp.Tool {
	return mcp.NewTool("manifest_coverage",
		mcp.WithDescription("Report which ecosystems have version tracking coverage and which are uncovered."),
	)
}

func handleManifestCoverage(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return mcp.NewToolResultErrorf("determining project root: %v", err), nil
	}

	answers, err := loadAnswers(projectRoot)
	if err != nil {
		return mcp.NewToolResultErrorf("loading answers: %v", err), nil
	}

	registry := ecosystem.DefaultRegistry()
	report := collectManifestCoverage(answers, registry)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling coverage: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func versionHistoryTool() mcp.Tool {
	return mcp.NewTool("version_history",
		mcp.WithDescription("Read the version-sentinel event log showing a timeline of version changes."),
		mcp.WithString("log_path", mcp.Description("Path to events.jsonl (defaults to .version-sentinel/events.jsonl)")),
	)
}

func handleVersionHistory(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logPath := request.GetString("log_path", "")
	if logPath == "" {
		root, err := cmdutil.ProjectRoot()
		if err != nil {
			return mcp.NewToolResultErrorf("determining project root: %v", err), nil
		}
		logPath = filepath.Join(root, ".version-sentinel", "events.jsonl")
	}

	events, err := vsentinel.ReadVersionHistory(logPath)
	if err != nil {
		return mcp.NewToolResultErrorf("reading version history: %v", err), nil
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return mcp.NewToolResultErrorf("marshaling events: %v", err), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
