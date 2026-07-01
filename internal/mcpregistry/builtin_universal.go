package mcpregistry

// universalServerName is the registry key for qsdev's own universal MCP server.
const universalServerName = "qsdev-universal"

// registerUniversalServer adds qsdev's universal MCP server (Phase 32) to the
// registry as a first-class built-in definition. Unlike the embedded providers
// populated by populateFromEmbeddedProviders, the universal server is launched
// with the explicit `qsdev mcp serve` subcommand and carries a verified
// compliance grade: it is qsdev's own binary, runs entirely locally over stdio,
// and ships no plaintext secrets. Registering it here means `qsdev mcp list`
// surfaces it like any other known server, since the registry is that command's
// single source of truth.
//
// Registration is best-effort and idempotent: if a definition with the same
// name already exists (e.g. it was also declared in the catalog), the existing
// entry wins and this is a no-op.
func registerUniversalServer(r *McpServerRegistry) {
	if _, exists := r.ByName(universalServerName); exists {
		return
	}

	_ = r.Register(McpServerDefinition{
		Name:        universalServerName,
		DisplayName: "qsdev Universal MCP Server",
		Category:    CategoryInfrastructure,
		Description: "qsdev's universal MCP server: exposes project detection, " +
			"configuration, security, devenv, and AI-framework tooling to any MCP " +
			"client over stdio.",
		Command:         "qsdev",
		Args:            []string{"mcp", "serve"},
		Transport:       TransportStdio,
		ProtocolVersion: "2025-11-25",
		Source:          SourceBuiltin,
		ComplianceGrade: ComplianceVerified,
	})
}
