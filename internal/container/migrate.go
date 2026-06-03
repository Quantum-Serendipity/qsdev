package container

// IssueSeverity classifies how urgently a migration issue must be addressed.
type IssueSeverity string

const (
	SeverityCritical IssueSeverity = "critical"
	SeverityWarning  IssueSeverity = "warning"
	SeverityInfo     IssueSeverity = "info"
)

// IssueCategory classifies the type of migration issue.
type IssueCategory string

const (
	CategoryVolumePerms IssueCategory = "volume_permissions"
	CategoryImageName   IssueCategory = "image_qualification"
	CategoryPrivPorts   IssueCategory = "privileged_ports"
	CategoryPrivileged  IssueCategory = "privileged_mode"
	CategorySocketMount IssueCategory = "docker_socket_mount"
	CategorySELinux     IssueCategory = "selinux_labels"
)

// MigrationIssue represents a single incompatibility found during
// Docker-to-Podman migration analysis of a compose file.
type MigrationIssue struct {
	Category    IssueCategory `json:"category"`
	Severity    IssueSeverity `json:"severity"`
	File        string        `json:"file"`
	Service     string        `json:"service,omitempty"`
	Line        int           `json:"line,omitempty"`
	Description string        `json:"description"`
	AutoFixable bool          `json:"auto_fixable"`
	Fix         *MigrationFix `json:"fix,omitempty"`
}

// MigrationFix describes how to resolve a MigrationIssue.
type MigrationFix struct {
	Description string   `json:"description"`
	YAMLPath    string   `json:"yaml_path,omitempty"`
	YAMLValue   string   `json:"yaml_value,omitempty"`
	EnvVar      string   `json:"env_var,omitempty"`
	EnvValue    string   `json:"env_value,omitempty"`
	ManualSteps []string `json:"manual_steps,omitempty"`
}

// MigrationReport is the output of Analyze: a summary of all issues found
// across one or more compose files in a project directory.
type MigrationReport struct {
	Timestamp     string           `json:"timestamp"`
	ProjectRoot   string           `json:"project_root"`
	SourceRuntime Runtime          `json:"source_runtime"`
	TargetRuntime Runtime          `json:"target_runtime"`
	ComposeFiles  []string         `json:"compose_files"`
	RuntimeInfo   *RuntimeInfo     `json:"runtime_info"`
	Capabilities  *Capabilities    `json:"capabilities,omitempty"`
	Issues        []MigrationIssue `json:"issues"`
	Summary       MigrationSummary `json:"summary"`
}

// MigrationSummary tallies issue counts by severity and fixability.
type MigrationSummary struct {
	Total       int `json:"total"`
	Critical    int `json:"critical"`
	Warning     int `json:"warning"`
	Info        int `json:"info"`
	AutoFixable int `json:"auto_fixable"`
	ManualOnly  int `json:"manual_only"`
}
