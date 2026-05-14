package surgery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// JSONAddMCPServer adds an MCP server entry to .mcp.json content.
// Creates the mcpServers key if it doesn't exist.
// Replaces the entry if the server name already exists.
// Returns the updated JSON with sorted keys.
func JSONAddMCPServer(existing []byte, serverName string, entry json.RawMessage) ([]byte, error) {
	var doc map[string]json.RawMessage
	if len(existing) == 0 || len(bytes.TrimSpace(existing)) == 0 {
		doc = make(map[string]json.RawMessage)
	} else {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("parsing .mcp.json: %w", err)
		}
	}

	var servers map[string]json.RawMessage
	if raw, ok := doc["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, fmt.Errorf("parsing mcpServers: %w", err)
		}
	} else {
		servers = make(map[string]json.RawMessage)
	}

	servers[serverName] = entry

	serversJSON, err := marshalSorted(servers)
	if err != nil {
		return nil, fmt.Errorf("marshaling mcpServers: %w", err)
	}
	doc["mcpServers"] = serversJSON

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling .mcp.json: %w", err)
	}
	return append(result, '\n'), nil
}

// JSONRemoveMCPServer removes an MCP server entry from .mcp.json content.
// Returns unchanged content if the server is not found.
func JSONRemoveMCPServer(existing []byte, serverName string) ([]byte, error) {
	if len(existing) == 0 {
		return existing, nil
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(existing, &doc); err != nil {
		return nil, fmt.Errorf("parsing .mcp.json: %w", err)
	}

	raw, ok := doc["mcpServers"]
	if !ok {
		return existing, nil
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("parsing mcpServers: %w", err)
	}

	if _, ok := servers[serverName]; !ok {
		return existing, nil
	}
	delete(servers, serverName)

	if len(servers) == 0 {
		delete(doc, "mcpServers")
	} else {
		serversJSON, err := marshalSorted(servers)
		if err != nil {
			return nil, fmt.Errorf("marshaling mcpServers: %w", err)
		}
		doc["mcpServers"] = serversJSON
	}

	if len(doc) == 0 {
		return []byte("{}\n"), nil
	}

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling .mcp.json: %w", err)
	}
	return append(result, '\n'), nil
}

// JSONHasMCPServer returns true if the named server exists in .mcp.json content.
func JSONHasMCPServer(content []byte, serverName string) bool {
	if len(content) == 0 {
		return false
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		return false
	}
	_, ok := doc.MCPServers[serverName]
	return ok
}

// marshalSorted marshals a map with sorted keys for deterministic output.
func marshalSorted(m map[string]json.RawMessage) (json.RawMessage, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(m[k])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
