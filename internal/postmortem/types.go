package postmortem

import "time"

type SessionAnalysis struct {
	SessionID        string
	StartTime        time.Time
	ToolUseCount     int
	FailureSequences []FailureSequence
	RecoveryPatterns []string
}

type FailureSequence struct {
	ToolName       string
	ErrorMessage   string
	RetryCount     int
	Recovered      bool
	RecoveryAction string
}

type FailureReport struct {
	TotalSessions int
	Patterns      []PatternEntry
	// SkippedFiles counts session files and directories that could not be
	// read or parsed, so an incomplete report is not mistaken for a clean one.
	SkippedFiles int
	// Truncated reports that the file limit stopped the scan early.
	Truncated bool
}

type PatternEntry struct {
	ToolName  string
	Error     string
	Count     int
	Recovered int
}
