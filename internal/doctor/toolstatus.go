package doctor

// ToolStatus represents the outcome of checking a single tool.
type ToolStatus struct {
	Name            string `json:"name"`
	Required        bool   `json:"required"`
	Installed       bool   `json:"installed"`
	Version         string `json:"version,omitempty"`
	MinVersion      string `json:"min_version,omitempty"`
	VersionOK       bool   `json:"version_ok"`
	Path            string `json:"path,omitempty"`
	AutoInstallable bool   `json:"auto_installable"`
	Notes           string `json:"notes,omitempty"`
}
