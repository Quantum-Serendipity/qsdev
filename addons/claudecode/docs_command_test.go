package claudecode_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

func TestDocsCmd_HasVerifySubcommand(t *testing.T) {
	cmd := claudecode.ExportDocsCmd()
	var verify *bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "verify" {
			found := true
			verify = &found
			// Confirm the documented flags are wired.
			for _, name := range []string{"json", "keys", "require-trusted"} {
				if sub.Flags().Lookup(name) == nil {
					t.Errorf("verify subcommand missing --%s flag", name)
				}
			}
		}
	}
	if verify == nil {
		t.Fatal("docs command is missing the 'verify' subcommand")
	}
}

// newTempCorpus writes a DevDocs doc set into a temp data dir and records a
// manifest entry with the correct combined hash, returning the manager and the
// installed entry.
func newTempCorpus(t *testing.T) (*mcpregistry.DocsCorpusManager, *mcpregistry.DocSetEntry) {
	t.Helper()
	dataDir := t.TempDir()
	mgr := mcpregistry.NewDocsCorpusManager(dataDir, nil)

	docDir := filepath.Join(dataDir, "devdocs", "go")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := []string{
		filepath.Join(docDir, "index.json"),
		filepath.Join(docDir, "db.json"),
	}
	for i, p := range files {
		if err := os.WriteFile(p, []byte("doc-content-"+string(rune('a'+i))), 0o644); err != nil {
			t.Fatalf("writing %s: %v", p, err)
		}
	}

	// VerifyHash with an empty recorded SHA256 returns ok=false but yields the
	// freshly computed digest, which is exactly what we record in the manifest.
	_, computed, err := mgr.VerifyHash(&mcpregistry.DocSetEntry{Files: files})
	if err != nil {
		t.Fatalf("VerifyHash setup: %v", err)
	}

	entry := &mcpregistry.DocSetEntry{
		Type:    mcpregistry.DocSetDevDocs,
		Slug:    "go",
		Version: "latest",
		SHA256:  computed,
		Files:   files,
	}
	manifest := &mcpregistry.DocsManifest{
		DocSets: map[string]*mcpregistry.DocSetEntry{"devdocs:go": entry},
	}
	if err := mgr.SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}
	return mgr, entry
}

func TestVerifyDocSet_HashVerified(t *testing.T) {
	t.Parallel()
	mgr, entry := newTempCorpus(t)

	res := claudecode.ExportVerifyDocSet(context.Background(), mgr, entry, nil, false)
	if !res.Verified {
		t.Fatalf("expected verified, got %+v", res)
	}
	if res.Status != "hash-verified" {
		t.Errorf("status = %q, want hash-verified", res.Status)
	}
	if res.Slug != "go" || res.Type != "devdocs" {
		t.Errorf("slug/type = %q/%q, want go/devdocs", res.Slug, res.Type)
	}
}

func TestVerifyDocSet_CorruptFails(t *testing.T) {
	t.Parallel()
	mgr, entry := newTempCorpus(t)

	// Corrupt one of the files after the manifest hash was recorded.
	if err := os.WriteFile(entry.Files[1], []byte("tampered"), 0o644); err != nil {
		t.Fatalf("corrupting file: %v", err)
	}

	res := claudecode.ExportVerifyDocSet(context.Background(), mgr, entry, nil, false)
	if res.Verified {
		t.Fatalf("expected failure for corrupted file, got %+v", res)
	}
	if res.Status != "failed" {
		t.Errorf("status = %q, want failed", res.Status)
	}
	if res.Reason == "" {
		t.Error("expected a failure reason")
	}
}

func TestVerifyDocSet_MissingFileFails(t *testing.T) {
	t.Parallel()
	mgr, entry := newTempCorpus(t)

	if err := os.Remove(entry.Files[0]); err != nil {
		t.Fatalf("removing file: %v", err)
	}

	res := claudecode.ExportVerifyDocSet(context.Background(), mgr, entry, nil, false)
	if res.Verified {
		t.Fatalf("expected failure for missing file, got %+v", res)
	}
	if res.Status != "failed" {
		t.Errorf("status = %q, want failed", res.Status)
	}
}
