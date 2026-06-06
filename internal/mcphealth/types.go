package mcphealth

import "time"

type ServerConfig struct {
	Name        string
	Command     string
	Args        []string
	Env         map[string]string
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
