package info

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func makeTestInfo() *ProjectInfo {
	return &ProjectInfo{
		ProjectName:       "test-project",
		Ecosystems:        []string{"go", "python"},
		ActiveToolCount:   5,
		SecurityProfile:   "enhanced",
		QsdevVersion:       "1.2.3",
		ConfigVersion:     1,
		LastUpdated:       time.Time{}, // zero = "never"
		ToolsByCategory:   map[string]int{"Security": 2, "AI Agent": 3},
		ManagedFileCount:  10,
		ClaudeCodeEnabled: true,
	}
}

func TestFormatDefault(t *testing.T) {
	info := makeTestInfo()
	var buf bytes.Buffer

	if err := FormatDefault(info, &buf); err != nil {
		t.Fatalf("FormatDefault: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"Project:       test-project",
		"Ecosystems:    go, python",
		"Security:      enhanced",
		"qsdev Version:  1.2.3",
		"Config:        v1",
		"Managed Files: 10",
		"Active Tools:  5",
		"Claude Code:   enabled",
		"Last Updated:  never",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("output missing %q\nGot:\n%s", s, output)
		}
	}

	// Check category breakdown.
	if !strings.Contains(output, "Security: 2") {
		t.Errorf("output missing category breakdown 'Security: 2'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "AI Agent: 3") {
		t.Errorf("output missing category breakdown 'AI Agent: 3'\nGot:\n%s", output)
	}
}

func TestFormatDefault_NoEcosystems(t *testing.T) {
	info := makeTestInfo()
	info.Ecosystems = nil
	var buf bytes.Buffer

	if err := FormatDefault(info, &buf); err != nil {
		t.Fatalf("FormatDefault: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Ecosystems:") {
		t.Errorf("output should not contain Ecosystems line when none set\nGot:\n%s", output)
	}
}

func TestFormatDefault_NoClaudeCode(t *testing.T) {
	info := makeTestInfo()
	info.ClaudeCodeEnabled = false
	var buf bytes.Buffer

	if err := FormatDefault(info, &buf); err != nil {
		t.Fatalf("FormatDefault: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Claude Code:") {
		t.Errorf("output should not contain Claude Code line when disabled\nGot:\n%s", output)
	}
}

func TestFormatOneline(t *testing.T) {
	info := makeTestInfo()
	var buf bytes.Buffer

	if err := FormatOneline(info, &buf); err != nil {
		t.Fatalf("FormatOneline: %v", err)
	}

	output := buf.String()

	// Should be a single line.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %q", len(lines), output)
	}

	// Should contain key fields.
	if !strings.Contains(output, "test-project") {
		t.Errorf("missing project name in oneline: %q", output)
	}
	if !strings.Contains(output, "go,python") {
		t.Errorf("missing ecosystems in oneline: %q", output)
	}
	if !strings.Contains(output, "enhanced") {
		t.Errorf("missing security profile in oneline: %q", output)
	}
	if !strings.Contains(output, "5 tools") {
		t.Errorf("missing tool count in oneline: %q", output)
	}
	if !strings.Contains(output, "never") {
		t.Errorf("missing last updated in oneline: %q", output)
	}
}

func TestFormatOneline_NoEcosystems(t *testing.T) {
	info := makeTestInfo()
	info.Ecosystems = nil
	var buf bytes.Buffer

	if err := FormatOneline(info, &buf); err != nil {
		t.Fatalf("FormatOneline: %v", err)
	}

	if !strings.Contains(buf.String(), "none") {
		t.Errorf("expected 'none' for empty ecosystems, got: %q", buf.String())
	}
}

func TestFormatJSON(t *testing.T) {
	info := makeTestInfo()
	var buf bytes.Buffer

	if err := FormatJSON(info, &buf); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	// Should be valid JSON.
	var decoded ProjectInfo
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify key fields.
	if decoded.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q, want %q", decoded.ProjectName, "test-project")
	}
	if decoded.ActiveToolCount != 5 {
		t.Errorf("ActiveToolCount = %d, want 5", decoded.ActiveToolCount)
	}
	if decoded.SecurityProfile != "enhanced" {
		t.Errorf("SecurityProfile = %q, want %q", decoded.SecurityProfile, "enhanced")
	}
	if decoded.QsdevVersion != "1.2.3" {
		t.Errorf("QsdevVersion = %q, want %q", decoded.QsdevVersion, "1.2.3")
	}
	if decoded.ConfigVersion != 1 {
		t.Errorf("ConfigVersion = %d, want 1", decoded.ConfigVersion)
	}
	if decoded.ManagedFileCount != 10 {
		t.Errorf("ManagedFileCount = %d, want 10", decoded.ManagedFileCount)
	}
	if !decoded.ClaudeCodeEnabled {
		t.Error("ClaudeCodeEnabled = false, want true")
	}
	if len(decoded.Ecosystems) != 2 {
		t.Errorf("len(Ecosystems) = %d, want 2", len(decoded.Ecosystems))
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		offset   time.Duration
		expected string
	}{
		{"zero value", 0, "never"},
		{"just now", 30 * time.Second, "just now"},
		{"1 minute", 90 * time.Second, "1 minute ago"},
		{"5 minutes", 5 * time.Minute, "5 minutes ago"},
		{"1 hour", 90 * time.Minute, "1 hour ago"},
		{"3 hours", 3 * time.Hour, "3 hours ago"},
		{"1 day", 36 * time.Hour, "1 day ago"},
		{"4 days", 4 * 24 * time.Hour, "4 days ago"},
		{"1 week", 10 * 24 * time.Hour, "1 week ago"},
		{"3 weeks", 21 * 24 * time.Hour, "3 weeks ago"},
		{"1 month", 35 * 24 * time.Hour, "1 month ago"},
		{"3 months", 90 * 24 * time.Hour, "3 months ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input time.Time
			if tt.name != "zero value" {
				input = time.Now().Add(-tt.offset)
			}
			got := RelativeTime(input)
			if got != tt.expected {
				t.Errorf("RelativeTime(%v ago) = %q, want %q", tt.offset, got, tt.expected)
			}
		})
	}
}

func TestRelativeTime_Future(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	got := RelativeTime(future)
	if got != "in the future" {
		t.Errorf("RelativeTime(future) = %q, want %q", got, "in the future")
	}
}
