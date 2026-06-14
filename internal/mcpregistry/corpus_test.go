package mcpregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	responses map[string]*http.Response
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	resp, ok := m.responses[req.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
		}, nil
	}
	return resp, nil
}

func makeResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestManifest_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewDocsCorpusManager(dir, nil)

	original := &DocsManifest{
		DocSets: map[string]*DocSetEntry{
			"devdocs:go": {
				Type:      DocSetDevDocs,
				Slug:      "go",
				Version:   "1.22",
				SizeBytes: 12345,
				SHA256:    "abc123",
				Files:     []string{"/some/path"},
			},
		},
	}

	if err := mgr.SaveManifest(original); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	loaded, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	entry, ok := loaded.DocSets["devdocs:go"]
	if !ok {
		t.Fatal("expected devdocs:go in loaded manifest")
	}
	if entry.Slug != "go" {
		t.Errorf("Slug = %q, want %q", entry.Slug, "go")
	}
	if entry.SizeBytes != 12345 {
		t.Errorf("SizeBytes = %d, want 12345", entry.SizeBytes)
	}
	if entry.SHA256 != "abc123" {
		t.Errorf("SHA256 = %q, want %q", entry.SHA256, "abc123")
	}
}

func TestManifest_EmptyOnMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewDocsCorpusManager(filepath.Join(dir, "nonexistent"), nil)

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if manifest.DocSets == nil {
		t.Fatal("expected non-nil DocSets map")
	}
	if len(manifest.DocSets) != 0 {
		t.Errorf("expected empty DocSets, got %d entries", len(manifest.DocSets))
	}
}

func TestDownloadDevDocs_CreatesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	slug := "go"

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			"https://documents.devdocs.io/go/index.json": makeResponse(`{"entries":[]}`),
			"https://documents.devdocs.io/go/db.json":    makeResponse(`{"content":"test"}`),
			"https://documents.devdocs.io/go/meta.json":  makeResponse(`{"name":"Go"}`),
		},
	}

	mgr := NewDocsCorpusManager(dir, client)
	if err := mgr.DownloadDevDocs(context.Background(), slug, "https://documents.devdocs.io"); err != nil {
		t.Fatalf("DownloadDevDocs: %v", err)
	}

	for _, f := range []string{"index.json", "db.json", "meta.json"} {
		path := filepath.Join(dir, "devdocs", slug, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", path)
		}
	}

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := manifest.DocSets["devdocs:go"]; !ok {
		t.Error("expected devdocs:go entry in manifest")
	}
}

func TestDownloadZIM_VerifyHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "fake zim content"
	h := sha256.Sum256([]byte(content))
	expectedHash := hex.EncodeToString(h[:])

	entry := ZIMEntry{
		Slug:         "test_2025-06",
		DisplayName:  "Test ZIM",
		URL:          "https://example.com/test.zim",
		ExpectedHash: expectedHash,
		SizeBytes:    int64(len(content)),
		Ecosystems:   []string{"go"},
	}

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			"https://example.com/test.zim": makeResponse(content),
		},
	}

	mgr := NewDocsCorpusManager(dir, client)
	if err := mgr.DownloadZIM(context.Background(), entry); err != nil {
		t.Fatalf("DownloadZIM: %v", err)
	}

	zimPath := filepath.Join(dir, "zim", "test_2025-06.zim")
	data, err := os.ReadFile(zimPath)
	if err != nil {
		t.Fatalf("reading zim file: %v", err)
	}
	if string(data) != content {
		t.Errorf("zim content = %q, want %q", string(data), content)
	}

	manifest, err := mgr.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if _, ok := manifest.DocSets["zim:test_2025-06"]; !ok {
		t.Error("expected zim:test_2025-06 entry in manifest")
	}
}

func TestDownloadZIM_HashMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := "fake zim content"

	entry := ZIMEntry{
		Slug:         "test_bad_hash",
		DisplayName:  "Test ZIM Bad Hash",
		URL:          "https://example.com/bad.zim",
		ExpectedHash: "0000000000000000000000000000000000000000000000000000000000000000",
		SizeBytes:    int64(len(content)),
		Ecosystems:   []string{"go"},
	}

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			"https://example.com/bad.zim": makeResponse(content),
		},
	}

	mgr := NewDocsCorpusManager(dir, client)
	err := mgr.DownloadZIM(context.Background(), entry)
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error = %q, want hash mismatch message", err.Error())
	}

	zimPath := filepath.Join(dir, "zim", "test_bad_hash.zim")
	if _, statErr := os.Stat(zimPath); !os.IsNotExist(statErr) {
		t.Error("expected zim file to be cleaned up after hash mismatch")
	}
}

func TestClean_ZIMOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewDocsCorpusManager(dir, nil)

	// Create test directories and files.
	zimDir := filepath.Join(dir, "zim")
	devdocsDir := filepath.Join(dir, "devdocs", "go")
	if err := os.MkdirAll(zimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(devdocsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zimDir, "test.zim"), []byte("zim"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devdocsDir, "index.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed manifest with both entries.
	manifest := &DocsManifest{
		DocSets: map[string]*DocSetEntry{
			"zim:test":   {Type: DocSetZIM, Slug: "test"},
			"devdocs:go": {Type: DocSetDevDocs, Slug: "go"},
		},
	}
	if err := mgr.SaveManifest(manifest); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Clean(CleanOptions{ZIMOnly: true}); err != nil {
		t.Fatalf("Clean ZIMOnly: %v", err)
	}

	// ZIM directory should be removed.
	if _, err := os.Stat(zimDir); !os.IsNotExist(err) {
		t.Error("expected zim dir to be removed")
	}

	// DevDocs should still exist.
	if _, err := os.Stat(filepath.Join(devdocsDir, "index.json")); os.IsNotExist(err) {
		t.Error("expected devdocs file to still exist")
	}

	// Manifest should only contain devdocs entry.
	loaded, err := mgr.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.DocSets["zim:test"]; ok {
		t.Error("expected zim:test removed from manifest")
	}
	if _, ok := loaded.DocSets["devdocs:go"]; !ok {
		t.Error("expected devdocs:go preserved in manifest")
	}
}

func TestClean_All(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewDocsCorpusManager(dir, nil)

	// Create test directories.
	zimDir := filepath.Join(dir, "zim")
	devdocsDir := filepath.Join(dir, "devdocs")
	if err := os.MkdirAll(zimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(devdocsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := &DocsManifest{
		DocSets: map[string]*DocSetEntry{
			"zim:test":   {Type: DocSetZIM, Slug: "test"},
			"devdocs:go": {Type: DocSetDevDocs, Slug: "go"},
		},
	}
	if err := mgr.SaveManifest(manifest); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Clean(CleanOptions{All: true}); err != nil {
		t.Fatalf("Clean All: %v", err)
	}

	if _, err := os.Stat(zimDir); !os.IsNotExist(err) {
		t.Error("expected zim dir to be removed")
	}
	if _, err := os.Stat(devdocsDir); !os.IsNotExist(err) {
		t.Error("expected devdocs dir to be removed")
	}

	loaded, err := mgr.LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.DocSets) != 0 {
		t.Errorf("expected empty manifest, got %d entries", len(loaded.DocSets))
	}
}

func TestDefaultDocsDataDir(t *testing.T) {
	got := DefaultDocsDataDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".qsdev", "docs")
	if got != want {
		t.Errorf("DefaultDocsDataDir() = %q, want %q", got, want)
	}
}
