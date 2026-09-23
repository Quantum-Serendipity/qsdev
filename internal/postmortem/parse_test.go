package postmortem_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/postmortem"
)

func writeFixture(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture %s: %v", name, err)
	}
	return path
}

// The fixtures below follow the Claude Code transcript schema: every record
// carries sessionId and timestamp; tool calls are tool_use blocks in
// "assistant" records, and their results are tool_result blocks in "user"
// records whose content is a string or an array of text blocks. Summary lines
// carry no sessionId.

func toolUse(id, name string) string {
	return `{"type":"assistant","sessionId":"sess-1","timestamp":"2026-01-02T03:04:06.000Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"` + id + `","name":"` + name + `","input":{}}]}}`
}

func toolResult(id string, isError bool, content string) string {
	errField := ""
	if isError {
		errField = `,"is_error":true`
	}
	return `{"type":"user","sessionId":"sess-1","timestamp":"2026-01-02T03:04:07.000Z","message":{"role":"user","content":[{"tool_use_id":"` + id + `","type":"tool_result","content":` + content + errField + `}]}}`
}

const (
	summaryLine = `{"type":"summary","summary":"Fix the build","leafUuid":"0b1c2d3e"}`
	promptLine  = `{"type":"user","sessionId":"sess-1","timestamp":"2026-01-02T03:04:05.000Z","message":{"role":"user","content":"please fix the build"}}`
)

func TestParseSessionJSONL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		lines            []string
		wantToolUseCount int
		wantFailures     int
		wantRecovered    []bool
		wantErrors       []string
		wantSessionID    string
	}{
		{
			name:  "empty file",
			lines: []string{},
		},
		{
			name: "successful session",
			lines: []string{
				summaryLine, promptLine,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", false, `"file1.txt"`),
			},
			wantToolUseCount: 1,
			wantSessionID:    "sess-1",
		},
		{
			name: "single failure no retry",
			lines: []string{
				promptLine,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", true, `"command not found"`),
			},
			wantToolUseCount: 1,
			wantFailures:     1,
			wantRecovered:    []bool{false},
			wantErrors:       []string{"command not found"},
			wantSessionID:    "sess-1",
		},
		{
			name: "array result content",
			lines: []string{
				promptLine,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", true, `[{"type":"text","text":"exit status 2"},{"type":"text","text":"no such file"}]`),
			},
			wantToolUseCount: 1,
			wantFailures:     1,
			wantRecovered:    []bool{false},
			wantErrors:       []string{"exit status 2\nno such file"},
			wantSessionID:    "sess-1",
		},
		{
			name: "failure and recovery",
			lines: []string{
				promptLine,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", true, `"command not found"`),
				toolUse("tu_2", "Bash"),
				toolResult("tu_2", false, `"success"`),
			},
			wantToolUseCount: 2,
			wantFailures:     1,
			wantRecovered:    []bool{true},
			wantSessionID:    "sess-1",
		},
		{
			name: "multiple failures",
			lines: []string{
				promptLine,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", true, `"error one"`),
				toolUse("tu_2", "Read"),
				toolResult("tu_2", true, `"file not found"`),
			},
			wantToolUseCount: 2,
			wantFailures:     2,
			wantRecovered:    []bool{false, false},
			wantSessionID:    "sess-1",
		},
		{
			name: "malformed lines skipped",
			lines: []string{
				`not json at all`,
				promptLine,
				`{"broken json`,
				toolUse("tu_1", "Bash"),
				toolResult("tu_1", false, `"ok"`),
			},
			wantToolUseCount: 1,
			wantSessionID:    "sess-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := writeFixture(t, dir, "session.jsonl", tc.lines)

			analysis, err := postmortem.ParseSessionJSONL(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if analysis.SessionID != tc.wantSessionID {
				t.Errorf("SessionID = %q, want %q", analysis.SessionID, tc.wantSessionID)
			}

			if analysis.ToolUseCount != tc.wantToolUseCount {
				t.Errorf("ToolUseCount = %d, want %d", analysis.ToolUseCount, tc.wantToolUseCount)
			}

			if len(analysis.FailureSequences) != tc.wantFailures {
				t.Fatalf("FailureSequences count = %d, want %d", len(analysis.FailureSequences), tc.wantFailures)
			}

			for i, wantRecov := range tc.wantRecovered {
				if analysis.FailureSequences[i].Recovered != wantRecov {
					t.Errorf("FailureSequences[%d].Recovered = %v, want %v", i, analysis.FailureSequences[i].Recovered, wantRecov)
				}
			}
			for i, wantErr := range tc.wantErrors {
				if got := analysis.FailureSequences[i].ErrorMessage; got != wantErr {
					t.Errorf("FailureSequences[%d].ErrorMessage = %q, want %q", i, got, wantErr)
				}
			}
		})
	}
}

// TestParseSessionJSONL_StartTime proves StartTime is the session's first
// record timestamp, not the time of analysis.
func TestParseSessionJSONL_StartTime(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, t.TempDir(), "session.jsonl", []string{summaryLine, promptLine, toolUse("tu_1", "Bash")})
	analysis, err := postmortem.ParseSessionJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if !analysis.StartTime.Equal(want) {
		t.Errorf("StartTime = %v, want %v", analysis.StartTime, want)
	}
}

func TestParseSessionJSONL_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := postmortem.ParseSessionJSONL("/nonexistent/path/session.jsonl")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestAggregateFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		sessions     []*postmortem.SessionAnalysis
		wantTotal    int
		wantPatterns int
		wantFirst    *postmortem.PatternEntry
	}{
		{
			name:         "empty sessions",
			sessions:     nil,
			wantTotal:    0,
			wantPatterns: 0,
		},
		{
			name: "multiple sessions same failure",
			sessions: []*postmortem.SessionAnalysis{
				{
					FailureSequences: []postmortem.FailureSequence{
						{ToolName: "Bash", ErrorMessage: "command not found", Recovered: false},
					},
				},
				{
					FailureSequences: []postmortem.FailureSequence{
						{ToolName: "Bash", ErrorMessage: "command not found", Recovered: true},
					},
				},
			},
			wantTotal:    2,
			wantPatterns: 1,
			wantFirst: &postmortem.PatternEntry{
				ToolName:  "Bash",
				Error:     "command not found",
				Count:     2,
				Recovered: 1,
			},
		},
		{
			name: "mixed failures sorted by count",
			sessions: []*postmortem.SessionAnalysis{
				{
					FailureSequences: []postmortem.FailureSequence{
						{ToolName: "Read", ErrorMessage: "file not found"},
						{ToolName: "Bash", ErrorMessage: "timeout"},
						{ToolName: "Bash", ErrorMessage: "timeout"},
					},
				},
				{
					FailureSequences: []postmortem.FailureSequence{
						{ToolName: "Bash", ErrorMessage: "timeout"},
					},
				},
			},
			wantTotal:    2,
			wantPatterns: 2,
			wantFirst: &postmortem.PatternEntry{
				ToolName: "Bash",
				Error:    "timeout",
				Count:    3,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			report := postmortem.AggregateFailures(tc.sessions)

			if report.TotalSessions != tc.wantTotal {
				t.Errorf("TotalSessions = %d, want %d", report.TotalSessions, tc.wantTotal)
			}

			if len(report.Patterns) != tc.wantPatterns {
				t.Fatalf("Patterns count = %d, want %d", len(report.Patterns), tc.wantPatterns)
			}

			if tc.wantFirst != nil {
				got := report.Patterns[0]
				if got.ToolName != tc.wantFirst.ToolName {
					t.Errorf("first pattern ToolName = %q, want %q", got.ToolName, tc.wantFirst.ToolName)
				}
				if got.Error != tc.wantFirst.Error {
					t.Errorf("first pattern Error = %q, want %q", got.Error, tc.wantFirst.Error)
				}
				if got.Count != tc.wantFirst.Count {
					t.Errorf("first pattern Count = %d, want %d", got.Count, tc.wantFirst.Count)
				}
				if got.Recovered != tc.wantFirst.Recovered {
					t.Errorf("first pattern Recovered = %d, want %d", got.Recovered, tc.wantFirst.Recovered)
				}
			}
		})
	}
}
