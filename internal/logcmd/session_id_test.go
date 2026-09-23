package logcmd

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// TestDiscoverSessions_SessionIDSources covers where a session's ID comes from:
// the current record key, the legacy "session" key, and the file name when the
// legacy key was written as the redaction marker (every log made while the
// "session" key matched the sensitive-name canon).
func TestDiscoverSessions_SessionIDSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		file   string
		record string
		wantID string
	}{
		{
			name:   "current key",
			file:   "qsdev-2026-01-02T03-04-05-a1b2c3.jsonl",
			record: `{"msg":"session started","` + logging.SessionAttrKey + `":"a1b2c3"}`,
			wantID: "a1b2c3",
		},
		{
			name:   "legacy key",
			file:   "qsdev-2026-01-02T03-04-05-d4e5f6.jsonl",
			record: `{"msg":"session started","session":"d4e5f6"}`,
			wantID: "d4e5f6",
		},
		{
			name:   "legacy redacted key falls back to file name",
			file:   "qsdev-2026-01-02T03-04-05-0a0b0c.jsonl",
			record: `{"msg":"session started","session":"[REDACTED]"}`,
			wantID: "0a0b0c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.file), []byte(tt.record+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			sessions, err := discoverSessions(dir)
			if err != nil {
				t.Fatalf("discoverSessions: %v", err)
			}
			if len(sessions) != 1 || sessions[0].ID != tt.wantID {
				t.Fatalf("sessions = %+v, want one with ID %q", sessions, tt.wantID)
			}
			if _, err := findSessionFile(dir, sessions[0].ID); err != nil {
				t.Errorf("findSessionFile(%q): %v", sessions[0].ID, err)
			}
		})
	}
}

// TestDiscoverSessions_RoundTripsInit proves the ID `logs list` shows for a
// session written by logging.Init is the real one, usable with `logs show`.
func TestDiscoverSessions_RoundTripsInit(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	t.Setenv("QSDEV_LOG", "")

	root := t.TempDir()
	s, err := logging.Init(logging.Config{ProjectRoot: root, ProjectScoped: true})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	s.Close()

	sessions, err := discoverSessions(logging.ProjectLogDir(root))
	if err != nil {
		t.Fatalf("discoverSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != s.ID {
		t.Fatalf("sessions = %+v, want one with ID %q", sessions, s.ID)
	}
	if _, err := findSessionFile(logging.ProjectLogDir(root), sessions[0].ID); err != nil {
		t.Errorf("findSessionFile(%q): %v", sessions[0].ID, err)
	}
}
