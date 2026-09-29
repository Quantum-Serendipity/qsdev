package merge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// EnforceMCPPolicy removes from merged .mcp.json content every server the
// client MCP policy does not permit. A merge keeps servers that exist only on
// disk (as user-added), so without this a server the committed client block
// forbids would survive in a checked-in or hand-edited .mcp.json. Other paths,
// and any content under an empty policy, are returned unchanged.
func EnforceMCPPolicy(relPath string, content []byte, policy types.MCPPolicy) ([]byte, error) {
	if policy.IsZero() || !strings.HasSuffix(relPath, ".mcp.json") {
		return content, nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(content, &top); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", relPath, err)
	}
	servers := rawServers(top["mcpServers"])
	for name := range servers {
		if !policy.Permits(name) {
			delete(servers, name)
		}
	}
	raw, err := json.Marshal(servers)
	if err != nil {
		return nil, fmt.Errorf("marshaling %s servers: %w", relPath, err)
	}
	top["mcpServers"] = raw
	out, err := json.Marshal(top)
	if err != nil {
		return nil, fmt.Errorf("marshaling %s: %w", relPath, err)
	}
	return CanonicalJSON(out)
}

// MergeOnCreateWithMCPPolicy returns MergeOnCreate followed by
// EnforceMCPPolicy, for create-path writers whose answers carry a client MCP
// policy.
func MergeOnCreateWithMCPPolicy(policy types.MCPPolicy) func(relPath string, theirs, ours []byte) ([]byte, error) {
	return func(relPath string, theirs, ours []byte) ([]byte, error) {
		merged, err := MergeOnCreate(relPath, theirs, ours)
		if err != nil {
			return nil, err
		}
		return EnforceMCPPolicy(relPath, merged, policy)
	}
}
