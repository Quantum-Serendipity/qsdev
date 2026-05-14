package info

import "time"

// ProjectInfo holds the data displayed by `gdev info`.
type ProjectInfo struct {
	ProjectName       string         `json:"project_name"`
	Ecosystems        []string       `json:"ecosystems"`
	ActiveToolCount   int            `json:"active_tool_count"`
	SecurityProfile   string         `json:"security_profile"`
	GdevVersion       string         `json:"gdev_version"`
	ConfigVersion     int            `json:"config_version"`
	LastUpdated       time.Time      `json:"last_updated"`
	ToolsByCategory   map[string]int `json:"tools_by_category"`
	ManagedFileCount  int            `json:"managed_file_count"`
	ClaudeCodeEnabled bool           `json:"claude_code_enabled"`
}

// OutputMode selects the rendering format.
type OutputMode int

const (
	ModeDefault OutputMode = iota
	ModeOneline
	ModeJSON
)
