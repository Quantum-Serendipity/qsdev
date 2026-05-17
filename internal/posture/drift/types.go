package drift

// Report summarizes configuration drift findings across categories.
type Report struct {
	Categories    []Category       `json:"categories"`
	TotalFindings int              `json:"totalFindings"`
	BySeverity    map[Severity]int `json:"bySeverity"`
}

// Severity categorizes the importance of a drift finding.
type Severity string

const (
	Critical Severity = "critical"
	Error    Severity = "error"
	Warning  Severity = "warning"
	Info     Severity = "info"
)

// Category groups drift findings under a named category.
type Category struct {
	Name     string    `json:"name"`
	Findings []Finding `json:"findings"`
}

// Finding describes a single configuration drift issue.
type Finding struct {
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	Expected    string   `json:"expected,omitempty"`
	Actual      string   `json:"actual,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	AutoFixable bool     `json:"autoFixable"`
}
