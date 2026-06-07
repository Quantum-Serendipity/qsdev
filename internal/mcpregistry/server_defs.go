package mcpregistry

type serverInstallInfo struct {
	Method      McpInstallMethod
	PackageName string
}

var knownServerInstalls = map[string]serverInstallInfo{
	"local-docs-devdocs": {InstallNpmGlobal, "devdocs-mcp-server"},
	"local-docs-zim":     {InstallUvTool, "openzim-mcp"},
	"man-pages":          {InstallUvTool, "man-mcp-server"},
	"mcp-nixos":          {InstallUvTool, "mcp-nixos"},
	"context7":           {InstallNpmGlobal, "@upstash/context7-mcp"},
	"github":             {InstallNpmGlobal, "@anthropic-ai/mcp-github"},
	"socket":             {InstallNpmGlobal, "@anthropic-ai/mcp-socket"},
	"semble":             {InstallUvTool, "semble[mcp]"},
}
