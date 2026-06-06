package vsentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogVersionEvent(t *testing.T) {
	t.Parallel()

	t.Run("write and read single event", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		logPath := filepath.Join(dir, "events.jsonl")

		event := VersionEvent{
			Timestamp:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			Ecosystem:  "go",
			Package:    "golang.org/x/sys",
			OldVersion: "v0.19.0",
			NewVersion: "v0.20.0",
			Source:     "go.mod",
		}

		if err := LogVersionEvent(logPath, event); err != nil {
			t.Fatalf("LogVersionEvent() error = %v", err)
		}

		events, err := ReadVersionHistory(logPath)
		if err != nil {
			t.Fatalf("ReadVersionHistory() error = %v", err)
		}

		if len(events) != 1 {
			t.Fatalf("event count = %d, want 1", len(events))
		}

		got := events[0]
		if got.Ecosystem != event.Ecosystem {
			t.Errorf("ecosystem = %q, want %q", got.Ecosystem, event.Ecosystem)
		}
		if got.Package != event.Package {
			t.Errorf("package = %q, want %q", got.Package, event.Package)
		}
		if got.OldVersion != event.OldVersion {
			t.Errorf("old version = %q, want %q", got.OldVersion, event.OldVersion)
		}
		if got.NewVersion != event.NewVersion {
			t.Errorf("new version = %q, want %q", got.NewVersion, event.NewVersion)
		}
		if got.Source != event.Source {
			t.Errorf("source = %q, want %q", got.Source, event.Source)
		}
	})

	t.Run("write multiple events and read all", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		logPath := filepath.Join(dir, "events.jsonl")

		events := []VersionEvent{
			{
				Timestamp:  time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
				Ecosystem:  "go",
				Package:    "golang.org/x/sys",
				OldVersion: "v0.19.0",
				NewVersion: "v0.20.0",
				Source:     "go.mod",
			},
			{
				Timestamp:  time.Date(2025, 1, 16, 11, 0, 0, 0, time.UTC),
				Ecosystem:  "javascript",
				Package:    "express",
				OldVersion: "4.18.0",
				NewVersion: "4.19.2",
				Source:     "package.json",
			},
			{
				Timestamp:  time.Date(2025, 1, 17, 12, 0, 0, 0, time.UTC),
				Ecosystem:  "rust",
				Package:    "serde",
				OldVersion: "1.0.200",
				NewVersion: "1.0.203",
				Source:     "Cargo.toml",
			},
		}

		for _, ev := range events {
			if err := LogVersionEvent(logPath, ev); err != nil {
				t.Fatalf("LogVersionEvent() error = %v", err)
			}
		}

		got, err := ReadVersionHistory(logPath)
		if err != nil {
			t.Fatalf("ReadVersionHistory() error = %v", err)
		}

		if len(got) != 3 {
			t.Fatalf("event count = %d, want 3", len(got))
		}

		for i, ev := range got {
			if ev.Package != events[i].Package {
				t.Errorf("event[%d].Package = %q, want %q", i, ev.Package, events[i].Package)
			}
		}
	})

	t.Run("read from non-existent file returns empty slice", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		logPath := filepath.Join(dir, "does-not-exist.jsonl")

		events, err := ReadVersionHistory(logPath)
		if err != nil {
			t.Fatalf("ReadVersionHistory() error = %v", err)
		}

		if events != nil {
			t.Errorf("expected nil, got %v", events)
		}
	})

	t.Run("verify JSON format", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		logPath := filepath.Join(dir, "events.jsonl")

		event := VersionEvent{
			Timestamp:  time.Date(2025, 3, 10, 14, 0, 0, 0, time.UTC),
			Ecosystem:  "python",
			Package:    "requests",
			OldVersion: "2.30.0",
			NewVersion: "2.31.0",
			Source:     "requirements.txt",
		}

		if err := LogVersionEvent(logPath, event); err != nil {
			t.Fatalf("LogVersionEvent() error = %v", err)
		}

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("reading log file: %v", err)
		}

		line := strings.TrimSpace(string(data))

		var parsed map[string]any
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Fatalf("line is not valid JSON: %v", err)
		}

		wantFields := []string{"timestamp", "ecosystem", "package", "old_version", "new_version", "source"}
		for _, field := range wantFields {
			if _, ok := parsed[field]; !ok {
				t.Errorf("missing JSON field %q", field)
			}
		}

		if parsed["ecosystem"] != "python" {
			t.Errorf("ecosystem = %v, want %q", parsed["ecosystem"], "python")
		}
		if parsed["package"] != "requests" {
			t.Errorf("package = %v, want %q", parsed["package"], "requests")
		}
	})
}
