package merge

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
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

	result := mcpJSON{
		MCPServers: make(map[string]mcpServerEntry),
	}

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
			if isEmptyServer(theirsEntry) && !isEmptyServer(oursEntry) {
				// Theirs was corrupted to empty — use ours.
				result.MCPServers[name] = oursEntry
			} else if !serverEqual(theirsEntry, baseEntry) {
				// User modified it — keep theirs version.
				result.MCPServers[name] = theirsEntry
			} else {
				// User didn't touch — use ours (updated) version.
				result.MCPServers[name] = oursEntry
			}
		} else {
			// Newly generated server — add from ours.
			result.MCPServers[name] = oursEntry
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
			result.MCPServers[name] = theirsEntry
		} else {
			// Was in base but removed from ours (generator removed it).
			// Only preserve if user modified it.
			if !serverEqual(theirsEntry, baseEntry) {
				result.MCPServers[name] = theirsEntry
			}
			// Otherwise: generator removed and user didn't modify → drop.
		}
	}

	// Sort keys for deterministic output by marshaling through an ordered structure.
	out, err := marshalMcpSorted(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged mcp.json: %w", err)
	}
	return append(out, '\n'), nil
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

// marshalMcpSorted marshals mcpJSON with sorted server names for deterministic output.
func marshalMcpSorted(m mcpJSON) ([]byte, error) {
	names := make([]string, 0, len(m.MCPServers))
	for name := range m.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build an ordered map using json.RawMessage to control key order.
	ordered := make([]serverKV, 0, len(names))
	for _, name := range names {
		ordered = append(ordered, serverKV{Name: name, Entry: m.MCPServers[name]})
	}

	wrapper := orderedMcp{Servers: ordered}
	return json.MarshalIndent(wrapper, "", "  ")
}

type serverKV struct {
	Name  string
	Entry mcpServerEntry
}

type orderedMcp struct {
	Servers []serverKV
}

func (o orderedMcp) MarshalJSON() ([]byte, error) {
	// Build {"mcpServers": {sorted...}}
	inner := make(map[string]json.RawMessage)
	for _, kv := range o.Servers {
		b, err := json.Marshal(kv.Entry)
		if err != nil {
			return nil, err
		}
		inner[kv.Name] = b
	}

	// We need to control key ordering inside mcpServers too.
	// Use a manual builder for the inner object.
	buf := []byte("{")
	for i, kv := range o.Servers {
		if i > 0 {
			buf = append(buf, ',')
		}
		key, _ := json.Marshal(kv.Name)
		val, err := json.Marshal(kv.Entry)
		if err != nil {
			return nil, err
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = append(buf, val...)
	}
	buf = append(buf, '}')

	// Wrap in {"mcpServers": ...}
	outerKey, _ := json.Marshal("mcpServers")
	result := []byte("{")
	result = append(result, outerKey...)
	result = append(result, ':')
	result = append(result, buf...)
	result = append(result, '}')
	return result, nil
}
