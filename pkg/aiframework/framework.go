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
