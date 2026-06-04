package aiframework

// FrameworkID identifies a supported AI coding framework.
type FrameworkID string

const (
	ClaudeCode  FrameworkID = "claudecode"
	Codex       FrameworkID = "codex"
	GeminiCLI   FrameworkID = "gemini"
	Copilot     FrameworkID = "copilot"
	Aider       FrameworkID = "aider"
	AmazonQ     FrameworkID = "amazonq"
	Cursor      FrameworkID = "cursor"
	Windsurf    FrameworkID = "windsurf"
	ContinueDev FrameworkID = "continue"
)

// AllFrameworks returns all defined framework IDs.
func AllFrameworks() []FrameworkID {
	return []FrameworkID{
		ClaudeCode, Codex, GeminiCLI, Copilot, Aider,
		AmazonQ, Cursor, Windsurf, ContinueDev,
	}
}

// FrameworkCapabilities describes what configuration features a framework supports.
type FrameworkCapabilities struct {
	SupportsContextFile  bool
	SupportsSettingsFile bool
	SupportsMCP          bool
	SupportsHooks        bool
	SupportsSandbox      bool
	SupportsIgnoreFile   bool
	SupportsPermissions  bool

	MaxToolCount    int
	MaxContextBytes int
	MaxRuleFiles    int
	MaxRuleBytes    int

	ContextFileFormat  string
	SettingsFileFormat string
	IgnoreFileSyntax   string
}

// FrameworkInfo provides metadata about a framework.
type FrameworkInfo struct {
	ID           FrameworkID
	Name         string
	Description  string
	Version      string
	Capabilities FrameworkCapabilities
}
