package merge

import (
	"encoding/json"
	"fmt"
	"slices"
)

// mcpJSON mirrors the claudecode.McpJSON structure.
type mcpJSON struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpServerEntry struct {
	Type    string            `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MergeMcpJson performs a three-way merge of .mcp.json content.
// base: original generated content (from last generation — may be nil for first update)
// theirs: current on-disk content (may have user modifications)
// ours: newly generated content
func MergeMcpJson(base, theirs, ours []byte) ([]byte, error) {
	var baseParsed, theirsParsed, oursParsed mcpJSON

	if len(base) > 0 {
		if err := json.Unmarshal(base, &baseParsed); err != nil {
			return nil, fmt.Errorf("parsing base mcp.json: %w", err)
		}
	}
	if baseParsed.MCPServers == nil {
		baseParsed.MCPServers = make(map[string]mcpServerEntry)
	}

	if len(theirs) == 0 {
		return nil, fmt.Errorf("parsing theirs mcp.json: unexpected end of JSON input")
	}
	if err := json.Unmarshal(theirs, &theirsParsed); err != nil {
		return nil, fmt.Errorf("parsing theirs mcp.json: %w", err)
	}
	if theirsParsed.MCPServers == nil {
		theirsParsed.MCPServers = make(map[string]mcpServerEntry)
	}

	if err := json.Unmarshal(ours, &oursParsed); err != nil {
		return nil, fmt.Errorf("parsing ours mcp.json: %w", err)
	}
	if oursParsed.MCPServers == nil {
		oursParsed.MCPServers = make(map[string]mcpServerEntry)
	}

	// Capture raw top-level keys and per-server JSON so unmodeled fields (e.g. a
	// per-server "headers" block) and sibling top-level keys survive: the merge
	// decisions are made on the typed structs, but output is assembled from the
	// raw JSON.
	theirsTop := map[string]json.RawMessage{}
	if err := json.Unmarshal(theirs, &theirsTop); err != nil {
		return nil, fmt.Errorf("parsing theirs mcp.json (raw): %w", err)
	}
	theirsServers := rawServers(theirsTop["mcpServers"])
	oursServers, err := rawServersFrom(ours)
	if err != nil {
		return nil, err
	}

	resultRaw := map[string]json.RawMessage{}

	// Process servers from ours.
	for name, oursEntry := range oursParsed.MCPServers {
		baseEntry, inBase := baseParsed.MCPServers[name]
		theirsEntry, inTheirs := theirsParsed.MCPServers[name]

		if inBase {
			// Generated server being updated.
			if !inTheirs {
				// User deleted it — respect deletion.
				continue
			}
			switch {
			case isEmptyServer(theirsEntry) && !isEmptyServer(oursEntry):
				// Theirs was corrupted to empty — use ours.
				resultRaw[name] = oursServers[name]
			case !serverEqual(theirsEntry, baseEntry):
				// User modified it — keep theirs version (raw preserves unknowns).
				resultRaw[name] = theirsServers[name]
			default:
				// User didn't touch modeled fields — use ours (updated) version,
				// but preserve any unmodeled fields the user added (e.g. headers).
				resultRaw[name] = mergeServerRaw(theirsServers[name], oursServers[name])
			}
		} else {
			// Newly generated server — add from ours.
			resultRaw[name] = oursServers[name]
		}
	}

	// Process servers from theirs that aren't in ours.
	for name, theirsEntry := range theirsParsed.MCPServers {
		if _, inOurs := oursParsed.MCPServers[name]; inOurs {
			continue // Already handled above.
		}
		baseEntry, inBase := baseParsed.MCPServers[name]
		if !inBase {
			// User-added server — preserve.
			resultRaw[name] = theirsServers[name]
		} else {
			// Was in base but removed from ours (generator removed it).
			// Only preserve if user modified it.
			if !serverEqual(theirsEntry, baseEntry) {
				resultRaw[name] = theirsServers[name]
			}
			// Otherwise: generator removed and user didn't modify → drop.
		}
	}

	// json.Marshal sorts map keys, giving deterministic server ordering.
	serversBytes, err := json.Marshal(resultRaw)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged mcp servers: %w", err)
	}

	// Preserve sibling top-level keys from theirs, replacing mcpServers with the
	// merged set. MarshalIndent re-indents the embedded raw JSON.
	top := make(map[string]json.RawMessage, len(theirsTop)+1)
	for k, v := range theirsTop {
		top[k] = v
	}
	top["mcpServers"] = serversBytes

	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling merged mcp.json: %w", err)
	}
	return append(out, '\n'), nil
}

// rawServersFrom parses the mcpServers object of a document into per-server raw
// JSON. Empty input yields an empty (non-nil) map.
func rawServersFrom(doc []byte) (map[string]json.RawMessage, error) {
	if len(doc) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(doc, &top); err != nil {
		return nil, fmt.Errorf("parsing mcp.json (raw): %w", err)
	}
	return rawServers(top["mcpServers"]), nil
}

// rawServers unmarshals a raw mcpServers object into per-server raw JSON.
func rawServers(raw json.RawMessage) map[string]json.RawMessage {
	servers := map[string]json.RawMessage{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &servers)
	}
	return servers
}

// mergeServerRaw overlays ours' server JSON onto theirs so ours' modeled fields
// win while unmodeled fields the user added (e.g. headers) survive. It falls
// back to ours when either side is absent or unparseable.
func mergeServerRaw(theirsRaw, oursRaw json.RawMessage) json.RawMessage {
	var theirsMap, oursMap map[string]any
	if len(theirsRaw) == 0 || json.Unmarshal(theirsRaw, &theirsMap) != nil {
		return oursRaw
	}
	if len(oursRaw) == 0 || json.Unmarshal(oursRaw, &oursMap) != nil {
		return oursRaw
	}
	merged, err := json.Marshal(DeepMergeJSON(theirsMap, oursMap))
	if err != nil {
		return oursRaw
	}
	return merged
}

// isEmptyServer returns true if the entry has no meaningful fields set.
// An empty object is never a valid user customization — it's corruption.
func isEmptyServer(s mcpServerEntry) bool {
	return s.Command == "" && s.URL == "" && s.Type == "" && len(s.Args) == 0 && len(s.Env) == 0
}

// serverEqual returns true if two mcpServerEntry values are equal.
func serverEqual(a, b mcpServerEntry) bool {
	if a.Type != b.Type || a.URL != b.URL {
		return false
	}
	if a.Command != b.Command {
		return false
	}
	if !slices.Equal(a.Args, b.Args) {
		return false
	}
	if len(a.Env) != len(b.Env) {
		return false
	}
	for k, v := range a.Env {
		if bv, ok := b.Env[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
