// Package mcpconfig is the single schema for a project's .mcp.json, the file
// Claude Code reads its project MCP servers from. It only parses: deciding
// whether an entry may be started or dialed is mcpregistry's job. The
// registry reader, the three-way merge and the claudecode generator all use
// these types.
//
// It is a foundation package and imports only the standard library.
package mcpconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileName is the project-relative name of the MCP server config file.
const FileName = ".mcp.json"

// Transport values Claude Code assigns when an entry's type is implicit.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// File is the parsed .mcp.json document.
type File struct {
	MCPServers map[string]Server `json:"mcpServers"`
}

// Server is one .mcp.json entry. Stdio servers use Command, Args and Env;
// remote servers use Type ("http" or "sse") with URL and optional Headers.
type Server struct {
	Type    string            `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Transport returns the entry's transport as Claude Code resolves it: the
// explicit type when set, otherwise "http" for an entry with a URL and
// "stdio" for one without.
func (s Server) Transport() string {
	switch {
	case s.Type != "":
		return s.Type
	case s.URL != "":
		return TransportHTTP
	default:
		return TransportStdio
	}
}

// Read parses root/.mcp.json. A missing file yields an empty File and no
// error; any other read or parse failure is returned.
func Read(root string) (File, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("reading %s: %w", FileName, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	return f, nil
}
