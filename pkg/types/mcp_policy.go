package types

import "slices"

// MCPWildcard in a blocked list blocks every MCP server not explicitly allowed.
const MCPWildcard = "*"

// MCPPolicy is a client's MCP server policy, taken from the committed
// .qsdev.yaml `client.allowed_mcp_servers` and `client.blocked_mcp_servers`.
// The Claude Code generator enforces it on every server it would write to
// .mcp.json, whichever source requested the server (claude_code.mcp_servers,
// catalog defaults, an enabled tool or an agent tool).
type MCPPolicy struct {
	Allowed []string `yaml:"allowed,omitempty" json:"allowed,omitempty"`
	Blocked []string `yaml:"blocked,omitempty" json:"blocked,omitempty"`
}

// IsZero reports whether the policy restricts nothing.
func (p MCPPolicy) IsZero() bool {
	return len(p.Allowed) == 0 && len(p.Blocked) == 0
}

// Permits reports whether the policy allows the named MCP server. With a
// wildcard in Blocked only the Allowed servers are permitted; otherwise every
// server except the Blocked ones is (a server both allowed and blocked by
// name is blocked, failing closed). Allowed alone restricts nothing: it only
// carves exceptions out of a wildcard block.
func (p MCPPolicy) Permits(name string) bool {
	if slices.Contains(p.Blocked, MCPWildcard) {
		return slices.Contains(p.Allowed, name)
	}
	return !slices.Contains(p.Blocked, name)
}

// Filter returns the servers the policy permits, preserving order.
func (p MCPPolicy) Filter(servers []string) []string {
	if p.IsZero() {
		return servers
	}
	var out []string
	for _, s := range servers {
		if p.Permits(s) {
			out = append(out, s)
		}
	}
	return out
}
