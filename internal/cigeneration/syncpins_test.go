package cigeneration

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readText reads a testdata file with its line endings normalised, so a
// Windows checkout that converts golden files to CRLF compares the same.
func readText(t *testing.T, path string) []byte {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// TestSyncPinsRewritesSHAAndTag applies a golden workflow to a golden catalog:
// the entries the workflow bumped change, the one it pins identically and the
// one it does not use stay byte-identical, and the result is gofmt-clean.
func TestSyncPinsRewritesSHAAndTag(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("testdata", "syncpins")
	pins, err := ParseWorkflowPins(dir)
	if err != nil {
		t.Fatal(err)
	}

	src := readText(t, filepath.Join(dir, "sha_pins.go.in"))
	want := readText(t, filepath.Join(dir, "sha_pins.go.golden"))

	got, err := SyncActionPins(src, pins)
	if err != nil {
		t.Fatalf("SyncActionPins() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("SyncActionPins() output differs from golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	formatted, err := format.Source(got)
	if err != nil {
		t.Fatalf("output does not parse: %v", err)
	}
	if !bytes.Equal(formatted, got) {
		t.Error("SyncActionPins() output is not gofmt-clean")
	}

	// The unused entry (Grype) must survive untouched.
	if !bytes.Contains(got, []byte(`SHA:   "1638637db639e0ade3258b51db49a9a137574c3e"`)) {
		t.Error("SyncActionPins() changed an entry the workflows do not use")
	}
}

// TestSyncPinsNoOpAtHead fails when the committed catalog is out of step with
// the committed workflows, i.e. when go generate would change sha_pins.go.
func TestSyncPinsNoOpAtHead(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("sha_pins.go")
	if err != nil {
		t.Fatal(err)
	}
	got, err := SyncActionPins(src, repoWorkflowPins(t))
	if err != nil {
		t.Fatalf("SyncActionPins() error = %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Error("go generate ./internal/cigeneration/ would rewrite sha_pins.go; run it and commit the result")
	}
}

func TestSyncPinsRejectsMalformedEntries(t *testing.T) {
	t.Parallel()

	pins := map[string]WorkflowPin{
		"actions/checkout": {SHA: strings.Repeat("1", 40), Tag: "v6.1.0", File: "ci.yml"},
	}
	tests := []struct {
		name string
		src  string
	}{
		{"not Go", "package x\nvar ="},
		{"non-literal SHA", "package x\nvar sha = \"a\"\nvar A = " + actionRefType + "{Owner: \"actions\", Repo: \"checkout\", SHA: sha, Tag: \"v1\"}\n"},
		{"missing Tag field", "package x\nvar A = " + actionRefType + "{Owner: \"actions\", Repo: \"checkout\", SHA: \"a\"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := SyncActionPins([]byte(tt.src), pins); err == nil {
				t.Error("SyncActionPins() returned no error")
			}
		})
	}
}
