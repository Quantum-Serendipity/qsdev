package contentsign

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngestDevDocs_StripsInvisibleChars(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// db.json with a zero-width space (U+200B) and a tag character (U+E0041) in
	// string values; keys are preserved untouched.
	raw := "{\"title\":\"He\u200bllo\",\"body\":\"do this\U000E0041\"}"
	dbPath := filepath.Join(dir, "db.json")
	if err := os.WriteFile(dbPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := IngestDevDocs(context.Background(), dir, DefaultSanitizeOptions())
	if err != nil {
		t.Fatalf("IngestDevDocs: %v", err)
	}

	out, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("reading db.json: %v", err)
	}

	if strings.ContainsRune(string(out), '\u200b') {
		t.Error("expected zero-width space to be stripped")
	}
	if strings.ContainsRune(string(out), '\U000E0041') {
		t.Error("expected tag character to be stripped")
	}

	// On-disk content must remain valid JSON with keys and visible text preserved.
	var parsed map[string]string
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["title"] != "Hello" {
		t.Errorf("title = %q, want %q", parsed["title"], "Hello")
	}
	if parsed["body"] != "do this" {
		t.Errorf("body = %q, want %q", parsed["body"], "do this")
	}

	if report.RunesStripped != 2 {
		t.Errorf("RunesStripped = %d, want 2", report.RunesStripped)
	}
	if report.Categories["zero-width"] != 1 {
		t.Errorf("zero-width count = %d, want 1", report.Categories["zero-width"])
	}
	if report.Categories["tag"] != 1 {
		t.Errorf("tag count = %d, want 1", report.Categories["tag"])
	}
}

func TestIngestDevDocs_MissingDB(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	report, err := IngestDevDocs(context.Background(), dir, DefaultSanitizeOptions())
	if err != nil {
		t.Fatalf("IngestDevDocs with missing db.json: %v", err)
	}
	if report.RunesStripped != 0 || report.NFKCChanged || len(report.Categories) != 0 {
		t.Errorf("expected zero report, got %+v", report)
	}
}

func TestIngestDevDocs_NoChangeLeavesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	raw := `{"title":"Hello","body":"clean content"}`
	dbPath := filepath.Join(dir, "db.json")
	if err := os.WriteFile(dbPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := IngestDevDocs(context.Background(), dir, DefaultSanitizeOptions())
	if err != nil {
		t.Fatalf("IngestDevDocs: %v", err)
	}
	if report.RunesStripped != 0 {
		t.Errorf("RunesStripped = %d, want 0", report.RunesStripped)
	}

	out, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("reading db.json: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["title"] != "Hello" || parsed["body"] != "clean content" {
		t.Errorf("content changed unexpectedly: %+v", parsed)
	}
}
