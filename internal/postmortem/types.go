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
}

type PatternEntry struct {
	ToolName  string
	Error     string
	Count     int
	Recovered int
}
