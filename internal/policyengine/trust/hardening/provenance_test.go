package hardening

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogProvenance_AppendsJSONLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "provenance.jsonl")
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	entries := []ProvenanceEntry{
		{Timestamp: ts, Server: "man-pages", Tool: "man", Tier: 1, ContentHash: "sha256:aa", Detections: 0},
		{Timestamp: ts, Server: "fetch", Tool: "fetch", Tier: 3, ContentHash: "sha256:bb", Detections: 2},
	}
	for _, e := range entries {
		if err := LogProvenance(path, e); err != nil {
			t.Fatalf("LogProvenance: %v", err)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening log: %v", err)
	}
	defer func() { _ = f.Close() }()

	var got []ProvenanceEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e ProvenanceEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("line %q is not a JSON entry: %v", sc.Text(), err)
		}
		got = append(got, e)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanning log: %v", err)
	}

	if len(got) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(got), len(entries))
	}
	for i := range entries {
		if !got[i].Timestamp.Equal(entries[i].Timestamp) {
			t.Errorf("entry %d timestamp = %v, want %v", i, got[i].Timestamp, entries[i].Timestamp)
		}
		got[i].Timestamp = entries[i].Timestamp
		if got[i] != entries[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], entries[i])
		}
	}
}

func TestLogProvenance_UnwritableDirectory(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatalf("writing blocker: %v", err)
	}
	// The parent "directory" is a regular file, so MkdirAll must fail.
	if err := LogProvenance(filepath.Join(blocker, "provenance.jsonl"), ProvenanceEntry{}); err == nil {
		t.Fatal("expected an error when the log directory cannot be created")
	}
}
