package mcphealth

import "time"

// ServerConfig describes one MCP server to probe, as configured in .mcp.json.
// Command, Args, URL, Env values and Headers values may contain ${VAR} and
// ${VAR:-default} references, which are expanded as Claude Code expands them.
type ServerConfig struct {
	Name        string
	Command     string
	Args        []string
	URL         string
	Env         map[string]string
	Headers     map[string]string
	RequiredEnv []string
}

type ServerHealth struct {
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	ToolCount     int                  `json:"tool_count"`
	ResponseMs    int64                `json:"response_ms"`
	Error         string               `json:"error,omitempty"`
	Prerequisites []PrerequisiteStatus `json:"prerequisites,omitempty"`
}

type HealthReport struct {
	Servers      []ServerHealth `json:"servers"`
	HealthyCount int            `json:"healthy_count"`
	TotalCount   int            `json:"total_count"`
	CheckedAt    time.Time      `json:"checked_at"`
}

type PrerequisiteStatus struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Met    bool   `json:"met"`
	Detail string `json:"detail,omitempty"`
}

// ConfigWarning is one problem ValidateConfig found in a server entry. Severity
// is SeverityError when the server cannot start as configured and
// SeverityWarning when it may start but lack something it needs.
type ConfigWarning struct {
	Server      string `json:"server"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

const (
	StatusHealthy       = "healthy"
	StatusDegraded      = "degraded"
	StatusUnreachable   = "unreachable"
	StatusMisconfigured = "misconfigured"
)

// ConfigWarning severities.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)
