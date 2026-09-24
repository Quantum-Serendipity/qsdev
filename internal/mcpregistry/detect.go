package mcpregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// mcpJSONFile mirrors the .mcp.json schema written by the claudecode addon.
type mcpJSONFile struct {
	MCPServers map[string]mcpJSONEntry `json:"mcpServers"`
}

// mcpJSONEntry represents a single server entry in .mcp.json. Stdio servers
// use Command/Args/Env; remote servers use Type ("http" or "sse") with URL and
// optional Headers.
type mcpJSONEntry struct {
	Type    string            `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// transport returns the entry's transport. An entry with a URL but no type is
// a remote server, which Claude Code treats as HTTP.
func (e mcpJSONEntry) transport() McpTransport {
	if e.Type == "" && e.URL != "" {
		return TransportHTTP
	}
	return parseMcpTransport(e.Type)
}

// ScanMcpJSON reads .mcp.json from projectRoot and returns a map of
// McpServerDefinition keyed by server name. If the file does not exist
// the function returns an empty map and nil error.
func ScanMcpJSON(projectRoot string) (map[string]McpServerDefinition, error) {
	path := filepath.Join(projectRoot, ".mcp.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return make(map[string]McpServerDefinition), nil
		}
		return nil, fmt.Errorf("reading .mcp.json: %w", err)
	}

	var file mcpJSONFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing .mcp.json: %w", err)
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
			Transport: entry.transport(),
			Source:    SourceConfig,
		}
	}
	return result, nil
}
