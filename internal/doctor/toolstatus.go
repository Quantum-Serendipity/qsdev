package doctor

// ToolStatus represents the outcome of checking a single tool.
type ToolStatus struct {
	Name       string `json:"name"`
	Required   bool   `json:"required"`
	RequiredBy string `json:"required_by,omitempty"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	MinVersion string `json:"min_version,omitempty"`
	// Constraint is the version constraint the tool must satisfy in place of
	// MinVersion (see ToolCheck.Constraint).
	Constraint string `json:"version_constraint,omitempty"`
	VersionOK  bool   `json:"version_ok"`
	Path       string `json:"path,omitempty"`
	// InProject reports that the binary lies inside the project, so doctor
	// did not run it (see toolcheck.Info.InProject).
	InProject bool `json:"in_project,omitempty"`
	// UpgradeHint is how to upgrade a tool that is installed below its floor
	// when setup cannot (see ToolCheck.UpgradeHint).
	UpgradeHint string `json:"upgrade_hint,omitempty"`
	// PathHint replaces the install and upgrade advice (see
	// ToolCheck.PathHint).
	PathHint        string `json:"path_hint,omitempty"`
	AutoInstallable bool   `json:"auto_installable"`
	Notes           string `json:"notes,omitempty"`
}

// NeedsSetup reports whether the tool is missing or below its floor (a tool
// of unknown version does not meet a floor).
func (s ToolStatus) NeedsSetup() bool {
	return !s.Installed || ((s.MinVersion != "" || s.Constraint != "") && !s.VersionOK)
}
