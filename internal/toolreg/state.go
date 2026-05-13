package toolreg

// ToolState tracks the enabled/disabled status of all tools.
type ToolState struct {
	Enabled map[string]bool `yaml:"enabled" json:"enabled"`
}

// NewToolState creates a ToolState with an initialized map.
func NewToolState() ToolState {
	return ToolState{Enabled: make(map[string]bool)}
}

// IsEnabled returns whether a tool is currently enabled.
func (ts *ToolState) IsEnabled(name string) bool {
	if ts.Enabled == nil {
		return false
	}
	return ts.Enabled[name]
}

// Enable marks a tool as enabled.
func (ts *ToolState) Enable(name string) {
	if ts.Enabled == nil {
		ts.Enabled = make(map[string]bool)
	}
	ts.Enabled[name] = true
}

// Disable marks a tool as disabled.
func (ts *ToolState) Disable(name string) {
	if ts.Enabled == nil {
		return
	}
	ts.Enabled[name] = false
}
