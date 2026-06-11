package extlog

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestLogLevelString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level LogLevel
		want  string
	}{
		{LevelUnknown, "UNKNOWN"},
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
		{LogLevel(-1), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestLogLevelOrdering(t *testing.T) {
	t.Parallel()

	// Verify the ordering is correct for comparison operations
	// (used by Truncate for >= LevelError checks).
	if LevelDebug >= LevelInfo {
		t.Error("LevelDebug should be less than LevelInfo")
	}
	if LevelInfo >= LevelWarn {
		t.Error("LevelInfo should be less than LevelWarn")
	}
	if LevelWarn >= LevelError {
		t.Error("LevelWarn should be less than LevelError")
	}
	if LevelError >= LevelFatal {
		t.Error("LevelError should be less than LevelFatal")
	}
	if LevelUnknown >= LevelDebug {
		t.Error("LevelUnknown should be less than LevelDebug")
	}
}

func TestTimestampSourceValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ts   TimestampSource
		want string
	}{
		{"parsed", TSParsed, "parsed"},
		{"carried", TSCarried, "carried"},
		{"mtime", TSMtime, "mtime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.ts) != tt.want {
				t.Errorf("TimestampSource = %q, want %q", tt.ts, tt.want)
			}
		})
	}
}

func TestFormatEntries(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		entries []LogEntry
		want    string
	}{
		{
			name:    "empty entries",
			entries: nil,
			want:    "",
		},
		{
			name: "single entry with timestamp",
			entries: []LogEntry{
				{Timestamp: ts, Level: LevelInfo, Source: "npm", Message: "installing packages"},
			},
			want: "14:30:45 [INFO] npm: installing packages\n",
		},
		{
			name: "entry without timestamp",
			entries: []LogEntry{
				{Level: LevelError, Source: "nix", Message: "build failed"},
			},
			want: " [ERROR] nix: build failed\n",
		},
		{
			name: "multiple entries",
			entries: []LogEntry{
				{Timestamp: ts, Level: LevelDebug, Source: "devenv", Message: "checking inputs"},
				{Timestamp: ts, Level: LevelWarn, Source: "devenv", Message: "outdated flake lock"},
				{Timestamp: ts, Level: LevelFatal, Source: "devenv", Message: "evaluation failed"},
			},
			want: "14:30:45 [DEBUG] devenv: checking inputs\n" +
				"14:30:45 [WARN] devenv: outdated flake lock\n" +
				"14:30:45 [FATAL] devenv: evaluation failed\n",
		},
		{
			name: "unknown level",
			entries: []LogEntry{
				{Timestamp: ts, Level: LevelUnknown, Source: "generic", Message: "some line"},
			},
			want: "14:30:45 [UNKNOWN] generic: some line\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := FormatEntries(&buf, tt.entries)
			if err != nil {
				t.Fatalf("FormatEntries returned error: %v", err)
			}
			got := buf.String()
			if got != tt.want {
				t.Errorf("FormatEntries output:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestFormatEntriesWriteError(t *testing.T) {
	t.Parallel()

	entries := []LogEntry{
		{Level: LevelInfo, Source: "test", Message: "msg"},
	}

	w := &failWriter{failAfter: 0}
	err := FormatEntries(w, entries)
	if err == nil {
		t.Error("FormatEntries should return error on write failure")
	}
}

type failWriter struct {
	failAfter int
	written   int
}

func (w *failWriter) Write(p []byte) (int, error) {
	if w.written >= w.failAfter {
		return 0, bytes.ErrTooLarge
	}
	w.written += len(p)
	return len(p), nil
}

func TestDefaultWindow(t *testing.T) {
	t.Parallel()

	before := time.Now()
	w := DefaultWindow(30)
	after := time.Now()

	// Start should be approximately 30 minutes before now.
	expectedStart := before.Add(-30 * time.Minute)
	if w.Start.Before(expectedStart.Add(-time.Second)) || w.Start.After(after.Add(-29*time.Minute)) {
		t.Errorf("DefaultWindow(30).Start = %v, expected ~30 minutes before now", w.Start)
	}

	// End should be approximately 1 minute after now.
	expectedEnd := before.Add(1 * time.Minute)
	if w.End.Before(expectedEnd.Add(-time.Second)) || w.End.After(after.Add(2*time.Minute)) {
		t.Errorf("DefaultWindow(30).End = %v, expected ~1 minute after now", w.End)
	}

	// Window duration should be ~31 minutes.
	duration := w.End.Sub(w.Start)
	if duration < 30*time.Minute || duration > 32*time.Minute {
		t.Errorf("window duration = %v, want ~31 minutes", duration)
	}
}

func TestDefaultWindowZeroMinutes(t *testing.T) {
	t.Parallel()

	w := DefaultWindow(0)
	duration := w.End.Sub(w.Start)
	// With 0 minutes, window is from now to now+1min.
	if duration < 50*time.Second || duration > 70*time.Second {
		t.Errorf("DefaultWindow(0) duration = %v, want ~1 minute", duration)
	}
}

func TestStripANSI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no ANSI codes", "plain text", "plain text"},
		{"bold", "\x1b[1mbold text\x1b[0m", "bold text"},
		{"color red", "\x1b[31merror\x1b[0m", "error"},
		{"color green", "\x1b[32msuccess\x1b[0m", "success"},
		{"256 color", "\x1b[38;5;208morange\x1b[0m", "orange"},
		{"true color", "\x1b[38;2;255;128;0mtrue color\x1b[0m", "true color"},
		{"multiple codes", "\x1b[1m\x1b[31mbold red\x1b[0m", "bold red"},
		{"cursor movement", "\x1b[2Aup two lines", "up two lines"},
		{"empty string", "", ""},
		{"only ANSI", "\x1b[0m\x1b[1m\x1b[0m", ""},
		{"mixed content", "before \x1b[33myellow\x1b[0m after", "before yellow after"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCollectionSummaryFields(t *testing.T) {
	t.Parallel()

	s := CollectionSummary{
		Provider:         "npm",
		FileCount:        3,
		TotalBytes:       1024,
		EntryCount:       50,
		ErrorCount:       2,
		CollectionErrors: []string{"open failed", "parse error"},
	}

	if s.Provider != "npm" {
		t.Errorf("Provider = %q, want %q", s.Provider, "npm")
	}
	if s.FileCount != 3 {
		t.Errorf("FileCount = %d, want 3", s.FileCount)
	}
	if s.TotalBytes != 1024 {
		t.Errorf("TotalBytes = %d, want 1024", s.TotalBytes)
	}
	if s.EntryCount != 50 {
		t.Errorf("EntryCount = %d, want 50", s.EntryCount)
	}
	if s.ErrorCount != 2 {
		t.Errorf("ErrorCount = %d, want 2", s.ErrorCount)
	}
	if len(s.CollectionErrors) != 2 {
		t.Errorf("CollectionErrors length = %d, want 2", len(s.CollectionErrors))
	}
}

func TestLogEntryFields(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	e := LogEntry{
		Timestamp:       ts,
		TimestampSource: TSParsed,
		Level:           LevelError,
		Source:          "npm",
		Message:         "ENOENT no such file",
		File:            "/tmp/npm-debug.log",
		LineNumber:      42,
	}

	if e.Timestamp != ts {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, ts)
	}
	if e.Message != "ENOENT no such file" {
		t.Errorf("Message = %q, want %q", e.Message, "ENOENT no such file")
	}
	if e.TimestampSource != TSParsed {
		t.Errorf("TimestampSource = %q, want %q", e.TimestampSource, TSParsed)
	}
	if e.Level != LevelError {
		t.Errorf("Level = %v, want LevelError", e.Level)
	}
	if e.Source != "npm" {
		t.Errorf("Source = %q, want %q", e.Source, "npm")
	}
	if e.File != "/tmp/npm-debug.log" {
		t.Errorf("File = %q, want %q", e.File, "/tmp/npm-debug.log")
	}
	if e.LineNumber != 42 {
		t.Errorf("LineNumber = %d, want 42", e.LineNumber)
	}
}

func TestLogFileFields(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	lf := LogFile{
		Path:     "/home/user/.npm/_logs/2024-06-01.log",
		Provider: "npm",
		ModTime:  ts,
		Size:     4096,
	}

	if lf.Path != "/home/user/.npm/_logs/2024-06-01.log" {
		t.Errorf("Path = %q", lf.Path)
	}
	if lf.Provider != "npm" {
		t.Errorf("Provider = %q", lf.Provider)
	}
	if lf.ModTime != ts {
		t.Errorf("ModTime = %v", lf.ModTime)
	}
	if lf.Size != 4096 {
		t.Errorf("Size = %d", lf.Size)
	}
}

func TestFormatEntriesSpecialCharacters(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Timestamp: ts, Level: LevelInfo, Source: "test", Message: "line with\ttab"},
		{Timestamp: ts, Level: LevelWarn, Source: "test", Message: "unicode: éèê"},
		{Timestamp: ts, Level: LevelError, Source: "test", Message: ""},
	}

	var buf bytes.Buffer
	err := FormatEntries(&buf, entries)
	if err != nil {
		t.Fatalf("FormatEntries error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "line with\ttab") {
		t.Error("tab character not preserved")
	}
	if !strings.Contains(out, "unicode: éèê") {
		t.Error("unicode not preserved")
	}
	// Empty message should still produce a line.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 output lines, got %d", len(lines))
	}
}
