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

func TestIngestDevDocs_SizeBound(t *testing.T) {
	t.Parallel()

	// 30 bytes of valid JSON, comfortably larger than the tiny limit below.
	raw := `{"k":"vvvvvvvvvvvvvvvvvvvv"}`

	tests := []struct {
		name      string
		maxBytes  int64
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "oversize is rejected before any write",
			maxBytes:  10,
			wantErr:   true,
			errSubstr: "size exceeds",
		},
		{
			name:     "generous limit succeeds",
			maxBytes: maxIngestBytes,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			dbPath := filepath.Join(dir, "db.json")
			if err := os.WriteFile(dbPath, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := ingestDevDocs(context.Background(), dir, DefaultSanitizeOptions(), tt.maxBytes)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ingestDevDocs(maxBytes=%d): want error, got nil", tt.maxBytes)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.errSubstr)
				}
				// The oversize check must happen before any write: the file must
				// still hold the original bytes untouched.
				out, readErr := os.ReadFile(dbPath)
				if readErr != nil {
					t.Fatalf("reading db.json: %v", readErr)
				}
				if string(out) != raw {
					t.Errorf("file was rewritten: got %q, want original %q", string(out), raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ingestDevDocs(maxBytes=%d): unexpected error: %v", tt.maxBytes, err)
			}
		})
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
