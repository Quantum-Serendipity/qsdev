package agentpostmortem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/postmortem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func writeSession(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// call invokes a handler the way the server does and fails on a Go error,
// which a tool must never return for a tool-level failure.
func call(t *testing.T, h spi.ToolHandler, args map[string]any) *spi.ToolResult {
	t.Helper()
	res, err := h(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Arguments: args})
	if err != nil {
		t.Fatalf("handler returned a Go error: %v", err)
	}
	if res == nil {
		t.Fatal("handler returned a nil result")
	}
	return res
}

// status returns the structured status of an error result ("denied",
// "not_configured", "error"), or "" for a success.
func status(res *spi.ToolResult) string {
	if !res.IsError {
		return ""
	}
	if m, ok := res.Structured.(map[string]any); ok {
		s, _ := m["status"].(string)
		return s
	}
	return "error"
}

// TestTranscriptToolsConfinePaths proves agent-supplied paths cannot direct the
// transcript tools outside the Claude sessions directory, nor at a file that
// is not a session transcript.
func TestTranscriptToolsConfinePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	inside := filepath.Join(root, "proj", "s.jsonl")
	writeSession(t, inside, `{"type":"user","sessionId":"in"}`)
	writeSession(t, filepath.Join(root, "proj", "notes.md"), `# memory`)
	writeSession(t, filepath.Join(outside, "o.jsonl"), `{"type":"user","sessionId":"out"}`)

	m := newModule(root, nil)
	tests := []struct {
		name       string
		handler    spi.ToolHandler
		args       map[string]any
		wantStatus string
	}{
		{"analyze inside", m.analyzeSession, map[string]any{"session_path": inside}, ""},
		{"analyze relative to the sessions root", m.analyzeSession, map[string]any{"session_path": "proj/s.jsonl"}, ""},
		{"analyze outside", m.analyzeSession, map[string]any{"session_path": filepath.Join(outside, "o.jsonl")}, "denied"},
		{"analyze traversal", m.analyzeSession, map[string]any{"session_path": filepath.Join(root, "..", filepath.Base(outside), "o.jsonl")}, "denied"},
		{"analyze relative traversal", m.analyzeSession, map[string]any{"session_path": "../" + filepath.Base(outside) + "/o.jsonl"}, "denied"},
		{"analyze non-transcript inside", m.analyzeSession, map[string]any{"session_path": filepath.Join(root, "proj", "notes.md")}, "denied"},
		{"analyze missing argument", m.analyzeSession, map[string]any{}, "error"},
		{"list default root", m.listFailurePatterns, map[string]any{}, ""},
		{"list subdirectory", m.listFailurePatterns, map[string]any{"sessions_dir": filepath.Join(root, "proj")}, ""},
		{"list filesystem root", m.listFailurePatterns, map[string]any{"sessions_dir": "/"}, "denied"},
		{"list outside", m.listFailurePatterns, map[string]any{"sessions_dir": outside}, "denied"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := call(t, tt.handler, tt.args)
			if got := status(res); got != tt.wantStatus {
				t.Errorf("status = %q, want %q (text %q)", got, tt.wantStatus, res.Text)
			}
			if strings.Contains(res.Text, `"out"`) {
				t.Errorf("result leaked the outside transcript: %s", res.Text)
			}
		})
	}
}

// TestTranscriptToolsRejectSymlinkEscape proves a symlink inside the sessions
// root cannot lead the tools to a directory or transcript outside it.
func TestTranscriptToolsRejectSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	writeSession(t, filepath.Join(outside, "o.jsonl"), `{"type":"user","sessionId":"out"}`)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "o.jsonl"), filepath.Join(root, "file.jsonl")); err != nil {
		t.Fatal(err)
	}

	m := newModule(root, nil)
	tests := []struct {
		name    string
		handler spi.ToolHandler
		args    map[string]any
	}{
		{"list through a directory symlink", m.listFailurePatterns, map[string]any{"sessions_dir": filepath.Join(root, "link")}},
		{"analyze through a file symlink", m.analyzeSession, map[string]any{"session_path": filepath.Join(root, "file.jsonl")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := status(call(t, tt.handler, tt.args)); got != "denied" {
				t.Errorf("status = %q, want denied", got)
			}
		})
	}

	// The default walk of the root must not follow the file symlink either.
	res := call(t, m.listFailurePatterns, map[string]any{})
	report, ok := res.Structured.(*postmortem.FailureReport)
	if !ok || report.TotalSessions != 0 {
		t.Errorf("walk of the root = %s, want no sessions (the only transcript is a symlink out of the root)", res.Text)
	}
}

// TestTranscriptToolsMissingRoot proves an absent sessions directory is
// reported as not configured rather than as a failure.
func TestTranscriptToolsMissingRoot(t *testing.T) {
	t.Parallel()
	m := newModule(filepath.Join(t.TempDir(), "absent"), nil)
	if got := status(call(t, m.listFailurePatterns, map[string]any{})); got != "not_configured" {
		t.Errorf("status = %q, want not_configured", got)
	}
}

// TestGenerateChecklist proves a checklist failure is reported instead of
// being replaced by commands that do not apply to the project.
func TestGenerateChecklist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fn         func() ([]string, error)
		want       string
		wantStatus string
	}{
		{"no source", nil, "", "not_configured"},
		{"source error", func() ([]string, error) { return nil, errors.New("no answers file") }, "no answers file", "not_configured"},
		{"no commands", func() ([]string, error) { return nil, nil }, "[]", ""},
		{"commands", func() ([]string, error) { return []string{"npm test"}, nil }, "npm test", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := call(t, newModule("", tt.fn).generateChecklist, nil)
			if got := status(res); got != tt.wantStatus {
				t.Errorf("status = %q, want %q", got, tt.wantStatus)
			}
			if strings.Contains(res.Text, "go test") {
				t.Errorf("checklist fell back to Go commands: %s", res.Text)
			}
			if !strings.Contains(res.Text, tt.want) {
				t.Errorf("checklist = %s, want it to contain %q", res.Text, tt.want)
			}
		})
	}
}

// TestVerificationCommandsReadsPrimaryAnswers proves the checklist comes from
// the project's primary answers file and that a project without one is
// reported rather than answered with an empty checklist.
func TestVerificationCommandsReadsPrimaryAnswers(t *testing.T) {
	t.Parallel()

	uninitialized := t.TempDir()
	if _, err := verificationCommands(uninitialized); err == nil {
		t.Error("verificationCommands succeeded without an answers file")
	}

	project := t.TempDir()
	if err := answers.SavePrimary(project, types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "unregistered-language"}}}); err != nil {
		t.Fatal(err)
	}
	cmds, err := verificationCommands(project)
	if err != nil {
		t.Fatalf("verificationCommands: %v", err)
	}
	if len(cmds) != 0 {
		t.Errorf("commands for an unregistered language = %v, want none", cmds)
	}
}

// TestListFailurePatternsReportsSkipped proves unreadable transcripts are
// counted rather than silently dropped, so an incomplete report is not
// mistaken for a clean one.
func TestListFailurePatternsReportsSkipped(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can read unreadable files")
	}
	root := t.TempDir()
	writeSession(t, filepath.Join(root, "a.jsonl"), `{"type":"user","sessionId":"a"}`)
	bad := filepath.Join(root, "b.jsonl")
	writeSession(t, bad, `{}`)
	if err := os.Chmod(bad, 0o000); err != nil {
		t.Fatal(err)
	}

	res := call(t, newModule(root, nil).listFailurePatterns, map[string]any{})
	report, ok := res.Structured.(*postmortem.FailureReport)
	if !ok {
		t.Fatalf("structured result = %T, want *postmortem.FailureReport", res.Structured)
	}
	if report.TotalSessions != 1 || report.SkippedFiles != 1 {
		t.Errorf("report = %+v, want 1 session and 1 skipped file", report)
	}
}
