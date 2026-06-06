package postmortem_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestParseSessionJSONL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		lines            []string
		wantToolUseCount int
		wantFailures     int
		wantRecovered    []bool
		wantSessionID    string
	}{
		{
			name:             "empty file",
			lines:            []string{},
			wantToolUseCount: 0,
			wantFailures:     0,
			wantSessionID:    "",
		},
		{
			name: "successful session",
			lines: []string{
				`{"type":"summary","sessionId":"sess-ok"}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_1","content":"file1.txt"}}`,
			},
			wantToolUseCount: 1,
			wantFailures:     0,
			wantSessionID:    "sess-ok",
		},
		{
			name: "single failure no retry",
			lines: []string{
				`{"type":"summary","sessionId":"sess-fail"}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"bad-cmd"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_1","is_error":true,"content":"command not found"}}`,
			},
			wantToolUseCount: 1,
			wantFailures:     1,
			wantRecovered:    []bool{false},
			wantSessionID:    "sess-fail",
		},
		{
			name: "failure and recovery",
			lines: []string{
				`{"type":"summary","sessionId":"sess-recover"}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"bad-cmd"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_1","is_error":true,"content":"command not found"}}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_2","name":"Bash","input":{"command":"good-cmd"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_2","content":"success"}}`,
			},
			wantToolUseCount: 2,
			wantFailures:     1,
			wantRecovered:    []bool{true},
			wantSessionID:    "sess-recover",
		},
		{
			name: "multiple failures",
			lines: []string{
				`{"type":"summary","sessionId":"sess-multi"}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"bad1"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_1","is_error":true,"content":"error one"}}`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_2","name":"Read","input":{"path":"/missing"}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_2","is_error":true,"content":"file not found"}}`,
			},
			wantToolUseCount: 2,
			wantFailures:     2,
			wantRecovered:    []bool{false, false},
			wantSessionID:    "sess-multi",
		},
		{
			name: "malformed lines skipped",
			lines: []string{
				`not json at all`,
				`{"type":"summary","sessionId":"sess-malformed"}`,
				`{"broken json`,
				`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{}}]}}`,
				`{"type":"result","result":{"type":"tool_result","tool_use_id":"tu_1","content":"ok"}}`,
			},
			wantToolUseCount: 1,
			wantFailures:     0,
			wantSessionID:    "sess-malformed",
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
		})
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
