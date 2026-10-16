package mcpregistry

import (
	"fmt"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
)

// ScanMcpJSON reads .mcp.json from projectRoot and returns a map of
// McpServerDefinition keyed by server name. If the file does not exist
// the function returns an empty map and nil error. Each definition's Dir is
// the absolute projectRoot, the directory Claude Code launches the servers in.
func ScanMcpJSON(projectRoot string) (map[string]McpServerDefinition, error) {
	dir, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving project root %q: %w", projectRoot, err)
	}
	file, err := mcpconfig.Read(projectRoot)
	if err != nil {
		// mcpconfig already names the file and the failing step ("reading
		// .mcp.json: ..." / "parsing .mcp.json: ..."), the message callers
		// and their users have always seen.
		return nil, err
	}

	result := make(map[string]McpServerDefinition, len(file.MCPServers))
	for name, entry := range file.MCPServers {
		result[name] = McpServerDefinition{
			Name:      name,
			Command:   entry.Command,
			Args:      entry.Args,
			URL:       entry.URL,
			Headers:   entry.Headers,
			Env:       entry.Env,
			Transport: parseMcpTransport(entry.Transport()),
			Source:    SourceConfig,
			Dir:       dir,
		}
	}
	return result, nil
}
