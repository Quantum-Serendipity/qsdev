package logcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJsonField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		key  string
		want string
	}{
		{
			name: "string value",
			line: `{"level":"INFO","msg":"hello"}`,
			key:  "level",
			want: "INFO",
		},
		{
			name: "message field",
			line: `{"level":"INFO","msg":"hello world"}`,
			key:  "msg",
			want: "hello world",
		},
		{
			name: "missing key",
			line: `{"level":"INFO"}`,
			key:  "msg",
			want: "",
		},
		{
			name: "empty JSON",
			line: `{}`,
			key:  "anything",
			want: "",
		},
		{
			name: "invalid JSON",
			line: `not json at all`,
			key:  "level",
			want: "",
		},
		{
			name: "empty string",
			line: ``,
			key:  "level",
			want: "",
		},
		{
			name: "numeric value",
			line: `{"duration_ms":1234}`,
			key:  "duration_ms",
			want: "1234",
		},
		{
			name: "nested object is raw",
			line: `{"meta":{"foo":"bar"}}`,
			key:  "meta",
			want: `{"foo":"bar"}`,
		},
		{
			name: "timestamp field",
			line: `{"time":"2024-01-15T10:30:00Z","msg":"test"}`,
			key:  "time",
			want: "2024-01-15T10:30:00Z",
		},
		{
			name: "session field",
			line: `{"session":"abc123","command":"init"}`,
			key:  "session",
			want: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jsonField(tt.line, tt.key)
			if got != tt.want {
				t.Errorf("jsonField(%q, %q) = %q, want %q", tt.line, tt.key, got, tt.want)
			}
		})
	}
}

func TestLevelPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level string
		want  string
	}{
		{"DEBUG", "DBG"},
		{"debug", "DBG"},
		{"Debug", "DBG"},
		{"INFO", "INF"},
		{"info", "INF"},
		{"WARN", "WRN"},
		{"warn", "WRN"},
		{"WARNING", "WRN"},
		{"warning", "WRN"},
		{"ERROR", "ERR"},
		{"error", "ERR"},
		{"unknown", "???"},
		{"", "???"},
		{"TRACE", "???"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			got := levelPrefix(tt.level)
			if got != tt.want {
				t.Errorf("levelPrefix(%q) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "short string unchanged",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exact length unchanged",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "long string truncated",
			s:    "hello world",
			max:  5,
			want: "hell…",
		},
		{
			name: "empty string",
			s:    "",
			max:  5,
			want: "",
		},
		{
			name: "unicode string within limit",
			s:    "héllo",
			max:  5,
			want: "héllo",
		},
		{
			name: "unicode string truncated",
			s:    "héllo wörld",
			max:  5,
			want: "héll…",
		},
		{
			name: "single char max",
			s:    "hello",
			max:  1,
			want: "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"sub-second", 500, "500ms"},
		{"one millisecond", 1, "1ms"},
		{"zero", 0, "0ms"},
		{"exactly one second", 1000, "1.0s"},
		{"seconds", 2500, "2.5s"},
		{"minutes range", 60000, "60.0s"},
		{"large value", 123456, "123.5s"},
		{"just under 1s", 999, "999ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDuration(tt.ms)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		b    int64
		want string
	}{
		{"zero bytes", 0, "0B"},
		{"small bytes", 512, "512B"},
		{"just under 1KB", 1023, "1023B"},
		{"exactly 1KB", 1024, "1.0KB"},
		{"kilobytes", 2048, "2.0KB"},
		{"fractional KB", 1536, "1.5KB"},
		{"just under 1MB", 1024*1024 - 1, "1024.0KB"},
		{"exactly 1MB", 1024 * 1024, "1.0MB"},
		{"megabytes", 5 * 1024 * 1024, "5.0MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatBytes(tt.b)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"one day", "1d", 24 * time.Hour, false},
		{"thirty days", "30d", 30 * 24 * time.Hour, false},
		{"hours", "24h", 24 * time.Hour, false},
		{"minutes", "30m", 30 * time.Minute, false},
		{"seconds", "10s", 10 * time.Second, false},
		{"mixed", "1h30m", time.Hour + 30*time.Minute, false},
		{"whitespace trimmed", "  7d  ", 7 * 24 * time.Hour, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadHeadAndTail(t *testing.T) {
	t.Parallel()

	writeTemp := func(t *testing.T, content string) string {
		t.Helper()
		f, err := os.CreateTemp(t.TempDir(), "log-*.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(content); err != nil {
			t.Fatal(err)
		}
		f.Close()
		return f.Name()
	}

	t.Run("normal file", func(t *testing.T) {
		t.Parallel()
		path := writeTemp(t, `{"line":1}
{"line":2}
{"line":3}
{"line":4}
{"line":5}
`)
		head, last, err := readHeadAndTail(path, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(head) != 3 {
			t.Errorf("got %d head lines, want 3", len(head))
		}
		if head[0] != `{"line":1}` {
			t.Errorf("head[0] = %q, want %q", head[0], `{"line":1}`)
		}
		if last != `{"line":5}` {
			t.Errorf("last = %q, want %q", last, `{"line":5}`)
		}
	})

	t.Run("fewer lines than head count", func(t *testing.T) {
		t.Parallel()
		path := writeTemp(t, `{"line":1}
{"line":2}
`)
		head, last, err := readHeadAndTail(path, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(head) != 2 {
			t.Errorf("got %d head lines, want 2", len(head))
		}
		if last != `{"line":2}` {
			t.Errorf("last = %q, want %q", last, `{"line":2}`)
		}
	})

	t.Run("single line", func(t *testing.T) {
		t.Parallel()
		path := writeTemp(t, `{"only":true}`)
		head, last, err := readHeadAndTail(path, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(head) != 1 {
			t.Errorf("got %d head lines, want 1", len(head))
		}
		if last != `{"only":true}` {
			t.Errorf("last = %q, want %q", last, `{"only":true}`)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()
		path := writeTemp(t, "")
		head, last, err := readHeadAndTail(path, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(head) != 0 {
			t.Errorf("got %d head lines, want 0", len(head))
		}
		if last != "" {
			t.Errorf("last = %q, want empty", last)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, _, err := readHeadAndTail("/nonexistent/path/file.jsonl", 3)
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}

func TestDiscoverSessions(t *testing.T) {
	t.Parallel()

	makeLogFile := func(t *testing.T, dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("discovers jsonl files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		now := time.Now().UTC().Truncate(time.Second)
		ts := now.Format(time.RFC3339Nano)

		makeLogFile(t, dir, "init-abc123.jsonl", strings.Join([]string{
			`{"session":"abc123","command":"init","time":"` + ts + `"}`,
			`{"level":"INFO","msg":"started"}`,
			`{"level":"INFO","msg":"done","duration_ms":1500}`,
		}, "\n"))

		makeLogFile(t, dir, "build-def456.jsonl", strings.Join([]string{
			`{"session":"def456","command":"build","time":"` + ts + `"}`,
			`{"level":"INFO","msg":"compiled","duration_ms":300}`,
		}, "\n"))

		// Non-jsonl file should be ignored
		makeLogFile(t, dir, "notes.txt", "ignore me")

		sessions, err := discoverSessions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("got %d sessions, want 2", len(sessions))
		}

		// Sessions should be sorted newest first (both have same timestamp
		// so ordering within same timestamp is stable but implementation defined)
		ids := map[string]bool{}
		for _, s := range sessions {
			ids[s.ID] = true
		}
		if !ids["abc123"] || !ids["def456"] {
			t.Errorf("expected sessions abc123 and def456, got %v", ids)
		}
	})

	t.Run("extracts session info from head lines", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		now := time.Now().UTC().Truncate(time.Second)
		ts := now.Format(time.RFC3339Nano)

		makeLogFile(t, dir, "init-sess1.jsonl", strings.Join([]string{
			`{"session":"sess1","command":"init","time":"` + ts + `"}`,
			`{"level":"INFO","msg":"running"}`,
			`{"level":"INFO","msg":"complete","duration_ms":2500}`,
		}, "\n"))

		sessions, err := discoverSessions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("got %d sessions, want 1", len(sessions))
		}

		s := sessions[0]
		if s.ID != "sess1" {
			t.Errorf("ID = %q, want %q", s.ID, "sess1")
		}
		if s.Command != "init" {
			t.Errorf("Command = %q, want %q", s.Command, "init")
		}
		if s.Started.IsZero() {
			t.Error("Started is zero, expected a valid time")
		}
		if s.Duration != 2500 {
			t.Errorf("Duration = %d, want 2500", s.Duration)
		}
		if s.Size == 0 {
			t.Error("Size is 0, expected non-zero")
		}
	})

	t.Run("falls back to filename for session ID", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		// File without session field in JSON
		makeLogFile(t, dir, "init-fallback1.jsonl", `{"level":"INFO","msg":"no session field"}`)

		sessions, err := discoverSessions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("got %d sessions, want 1", len(sessions))
		}
		if sessions[0].ID != "fallback1" {
			t.Errorf("ID = %q, want %q (from filename)", sessions[0].ID, "fallback1")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		sessions, err := discoverSessions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("got %d sessions, want 0", len(sessions))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		t.Parallel()
		sessions, err := discoverSessions("/nonexistent/dir")
		if err != nil {
			t.Fatalf("expected nil error for nonexistent dir, got: %v", err)
		}
		if sessions != nil {
			t.Errorf("expected nil sessions, got %v", sessions)
		}
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "subdir.jsonl"), 0o755); err != nil {
			t.Fatal(err)
		}
		makeLogFile(t, dir, "real-log1.jsonl", `{"session":"log1","msg":"test"}`)

		sessions, err := discoverSessions(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sessions) != 1 {
			t.Errorf("got %d sessions, want 1", len(sessions))
		}
	})
}

func TestFindSessionFile(t *testing.T) {
	t.Parallel()

	t.Run("finds by session ID substring", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "init-abc123.jsonl")
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		found, err := findSessionFile(dir, "abc123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found != path {
			t.Errorf("found %q, want %q", found, path)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "init-abc123.jsonl"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := findSessionFile(dir, "xyz789")
		if err == nil {
			t.Error("expected error for missing session, got nil")
		}
	})

	t.Run("ignores non-jsonl files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "abc123.txt"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := findSessionFile(dir, "abc123")
		if err == nil {
			t.Error("expected error since only non-jsonl file exists")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		t.Parallel()
		_, err := findSessionFile("/nonexistent/dir", "abc123")
		if err == nil {
			t.Error("expected error for nonexistent directory")
		}
	})
}

func TestWriteTo(t *testing.T) {
	t.Parallel()

	t.Run("writes all lines without filter", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader(strings.Join([]string{
			`{"level":"INFO","msg":"first"}`,
			`{"level":"WARN","msg":"second"}`,
			`{"level":"ERROR","msg":"third"}`,
		}, "\n"))

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 3 {
			t.Errorf("got %d output lines, want 3", len(lines))
		}
	})

	t.Run("filters by level", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader(strings.Join([]string{
			`{"level":"INFO","msg":"first"}`,
			`{"level":"WARN","msg":"second"}`,
			`{"level":"ERROR","msg":"third"}`,
			`{"level":"INFO","msg":"fourth"}`,
		}, "\n"))

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "INFO", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("level filter is case-insensitive", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader(`{"level":"info","msg":"test"}`)

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "INFO", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("respects maxLines limit", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader(strings.Join([]string{
			`{"level":"INFO","msg":"1"}`,
			`{"level":"INFO","msg":"2"}`,
			`{"level":"INFO","msg":"3"}`,
			`{"level":"INFO","msg":"4"}`,
			`{"level":"INFO","msg":"5"}`,
		}, "\n"))

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "", 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	t.Run("maxLines with filter", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader(strings.Join([]string{
			`{"level":"INFO","msg":"1"}`,
			`{"level":"WARN","msg":"2"}`,
			`{"level":"INFO","msg":"3"}`,
			`{"level":"WARN","msg":"4"}`,
			`{"level":"INFO","msg":"5"}`,
		}, "\n"))

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "INFO", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("")

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
		if buf.Len() != 0 {
			t.Errorf("buf not empty: %q", buf.String())
		}
	})

	t.Run("non-JSON lines passed through", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("not json\nalso not json\n")

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("non-JSON lines filtered out with level filter", func(t *testing.T) {
		t.Parallel()
		input := strings.NewReader("not json\n")

		var buf bytes.Buffer
		count, err := WriteTo(&buf, input, "INFO", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0 (non-JSON should not match level filter)", count)
		}
	})
}

func TestCommandTree(t *testing.T) {
	t.Parallel()

	cmd := Command()

	t.Run("root command", func(t *testing.T) {
		t.Parallel()
		if cmd.Use != "logs" {
			t.Errorf("Use = %q, want %q", cmd.Use, "logs")
		}
	})

	t.Run("has expected subcommands", func(t *testing.T) {
		t.Parallel()
		subNames := map[string]bool{}
		for _, sub := range cmd.Commands() {
			subNames[sub.Use] = true
		}
		for _, want := range []string{"list", "show <session-id>", "path", "clean"} {
			if !subNames[want] {
				t.Errorf("missing subcommand %q", want)
			}
		}
	})

	t.Run("global flag on root", func(t *testing.T) {
		t.Parallel()
		f := cmd.PersistentFlags().Lookup("global")
		if f == nil {
			t.Fatal("expected --global persistent flag")
			return
		}
		if f.DefValue != "false" {
			t.Errorf("--global default = %q, want %q", f.DefValue, "false")
		}
	})

	t.Run("list has flags", func(t *testing.T) {
		t.Parallel()
		list, _, err := cmd.Find([]string{"list"})
		if err != nil {
			t.Fatalf("finding list command: %v", err)
		}
		if list.Flags().Lookup("since") == nil {
			t.Error("list missing --since flag")
		}
		if list.Flags().Lookup("json") == nil {
			t.Error("list missing --json flag")
		}
	})

	t.Run("show has flags", func(t *testing.T) {
		t.Parallel()
		show, _, err := cmd.Find([]string{"show"})
		if err != nil {
			t.Fatalf("finding show command: %v", err)
		}
		if show.Flags().Lookup("level") == nil {
			t.Error("show missing --level flag")
		}
		if show.Flags().Lookup("raw") == nil {
			t.Error("show missing --raw flag")
		}
	})

	t.Run("clean has flags", func(t *testing.T) {
		t.Parallel()
		clean, _, err := cmd.Find([]string{"clean"})
		if err != nil {
			t.Fatalf("finding clean command: %v", err)
		}
		if f := clean.Flags().Lookup("older-than"); f == nil {
			t.Error("clean missing --older-than flag")
		} else if f.DefValue != "30d" {
			t.Errorf("--older-than default = %q, want %q", f.DefValue, "30d")
		}
		if clean.Flags().Lookup("all") == nil {
			t.Error("clean missing --all flag")
		}
		if clean.Flags().Lookup("force") == nil {
			t.Error("clean missing --force flag")
		}
	})
}

func TestDiscoverSessionsSortOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create sessions with different timestamps, newest should come first
	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		ts   time.Time
		id   string
	}{
		{"first-aaa.jsonl", t1, "aaa"},
		{"second-bbb.jsonl", t2, "bbb"},
		{"third-ccc.jsonl", t3, "ccc"},
	} {
		content := `{"session":"` + tc.id + `","time":"` + tc.ts.Format(time.RFC3339Nano) + `"}`
		if err := os.WriteFile(filepath.Join(dir, tc.name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := discoverSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}

	// Should be sorted newest first: bbb (12:00), ccc (11:00), aaa (10:00)
	wantOrder := []string{"bbb", "ccc", "aaa"}
	for i, want := range wantOrder {
		if sessions[i].ID != want {
			t.Errorf("sessions[%d].ID = %q, want %q", i, sessions[i].ID, want)
		}
	}
}

func TestRunListJSON(t *testing.T) {
	t.Parallel()

	// Set up a temp directory with log files, then invoke the list subcommand
	// with --json to verify JSON output rendering.
	dir := t.TempDir()
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC).Format(time.RFC3339Nano)
	content := strings.Join([]string{
		`{"session":"test1","command":"init","time":"` + ts + `"}`,
		`{"level":"INFO","msg":"done","duration_ms":500}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "init-test1.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build a cobra command that uses our discoverSessions + JSON output path
	// We can't easily invoke runList because it calls resolveLogDir via cmd flags,
	// but we can test the JSON encoding of sessionInfo directly.
	sessions, err := discoverSessions(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sessions); err != nil {
		t.Fatalf("JSON encode error: %v", err)
	}

	// Verify it parses back
	var decoded []sessionInfo
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("decoded %d sessions, want 1", len(decoded))
	}
	if decoded[0].ID != "test1" {
		t.Errorf("decoded ID = %q, want %q", decoded[0].ID, "test1")
	}
	if decoded[0].Command != "init" {
		t.Errorf("decoded Command = %q, want %q", decoded[0].Command, "init")
	}
	if decoded[0].Duration != 500 {
		t.Errorf("decoded Duration = %d, want 500", decoded[0].Duration)
	}
}

func TestSessionInfoJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := sessionInfo{
		ID:       "abc123",
		Command:  "init",
		Started:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		Duration: 2500,
		Size:     4096,
		File:     "/tmp/logs/init-abc123.jsonl",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded sessionInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Command != original.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, original.Command)
	}
	if !decoded.Started.Equal(original.Started) {
		t.Errorf("Started = %v, want %v", decoded.Started, original.Started)
	}
	if decoded.Duration != original.Duration {
		t.Errorf("Duration = %d, want %d", decoded.Duration, original.Duration)
	}
	if decoded.Size != original.Size {
		t.Errorf("Size = %d, want %d", decoded.Size, original.Size)
	}
	if decoded.File != original.File {
		t.Errorf("File = %q, want %q", decoded.File, original.File)
	}

	// Verify JSON field names
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map error: %v", err)
	}
	expectedKeys := []string{"id", "command", "started", "duration_ms", "size_bytes", "file"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing expected key %q", key)
		}
	}
}
