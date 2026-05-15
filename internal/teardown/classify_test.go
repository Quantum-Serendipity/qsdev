package teardown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func testRegistry() *toolreg.Registry {
	r := toolreg.NewRegistry()
	_ = r.Register(toolreg.Tool{
		Name:     "test-tool",
		Category: toolreg.CategorySecurity,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: "exclusive.txt", Ownership: toolreg.Exclusive},
			{Path: "shared.md", Ownership: toolreg.Shared, SectionID: "test-tool"},
		},
	})
	_ = r.Register(toolreg.Tool{
		Name:     "other-tool",
		Category: toolreg.CategoryDevEx,
		OwnedFiles: []toolreg.FileOwnership{
			{Path: "shared.md", Ownership: toolreg.Shared, SectionID: "other-tool"},
		},
	})
	return r
}

func TestClassifyFiles_Unmodified(t *testing.T) {
	dir := t.TempDir()
	content := []byte("original content")
	absPath := filepath.Join(dir, "exclusive.txt")
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"exclusive.txt": {
				Hash:  state.ComputeHash(content),
				Owner: "test-tool",
			},
		},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if len(classified) != 1 {
		t.Fatalf("expected 1 classified file, got %d", len(classified))
	}

	cf := classified[0]
	if cf.Path != "exclusive.txt" {
		t.Errorf("Path = %q, want %q", cf.Path, "exclusive.txt")
	}
	if cf.Modified {
		t.Errorf("Modified = true, want false (file matches hash)")
	}
	if cf.Deleted {
		t.Errorf("Deleted = true, want false (file exists)")
	}
	if cf.Ownership != toolreg.Exclusive {
		t.Errorf("Ownership = %v, want Exclusive", cf.Ownership)
	}
}

func TestClassifyFiles_Modified(t *testing.T) {
	dir := t.TempDir()
	originalContent := []byte("original content")
	modifiedContent := []byte("user edited this")
	absPath := filepath.Join(dir, "exclusive.txt")
	if err := os.WriteFile(absPath, modifiedContent, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"exclusive.txt": {
				Hash:  state.ComputeHash(originalContent),
				Owner: "test-tool",
			},
		},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if len(classified) != 1 {
		t.Fatalf("expected 1 classified file, got %d", len(classified))
	}

	cf := classified[0]
	if !cf.Modified {
		t.Errorf("Modified = false, want true (hash differs)")
	}
	if cf.Deleted {
		t.Errorf("Deleted = true, want false")
	}
}

func TestClassifyFiles_Deleted(t *testing.T) {
	dir := t.TempDir()
	// File does not exist on disk.
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"exclusive.txt": {
				Hash:  state.ComputeHash([]byte("original")),
				Owner: "test-tool",
			},
		},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if len(classified) != 1 {
		t.Fatalf("expected 1 classified file, got %d", len(classified))
	}

	cf := classified[0]
	if !cf.Deleted {
		t.Errorf("Deleted = false, want true (file does not exist)")
	}
}

func TestClassifyFiles_SharedSectionIDs(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# Shared content\n<!-- qsdev:test-tool -->\nstuff\n<!-- /qsdev:test-tool -->\n")
	absPath := filepath.Join(dir, "shared.md")
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"shared.md": {
				Hash:  state.ComputeHash(content),
				Owner: "test-tool",
			},
		},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if len(classified) != 1 {
		t.Fatalf("expected 1 classified file, got %d", len(classified))
	}

	cf := classified[0]
	if cf.Ownership != toolreg.Shared {
		t.Errorf("Ownership = %v, want Shared", cf.Ownership)
	}
	if len(cf.SectionIDs) != 2 {
		t.Errorf("SectionIDs = %v, want 2 entries (test-tool, other-tool)", cf.SectionIDs)
	}

	// Verify both section IDs are present.
	sectionMap := make(map[string]bool)
	for _, id := range cf.SectionIDs {
		sectionMap[id] = true
	}
	if !sectionMap["test-tool"] {
		t.Errorf("SectionIDs missing 'test-tool'")
	}
	if !sectionMap["other-tool"] {
		t.Errorf("SectionIDs missing 'other-tool'")
	}
}

func TestClassifyFiles_EmptyState(t *testing.T) {
	dir := t.TempDir()
	genState := types.GeneratedState{
		Files: map[string]types.FileState{},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if classified != nil {
		t.Errorf("expected nil for empty state, got %v", classified)
	}
}

func TestClassifyFiles_UnknownFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("something")
	absPath := filepath.Join(dir, "unknown.txt")
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"unknown.txt": {
				Hash:  state.ComputeHash(content),
				Owner: "mystery-tool",
			},
		},
	}

	registry := testRegistry()
	classified := ClassifyFiles(genState, dir, registry)

	if len(classified) != 1 {
		t.Fatalf("expected 1 classified file, got %d", len(classified))
	}

	cf := classified[0]
	// File not in any tool's OwnedFiles -> default to Exclusive.
	if cf.Ownership != toolreg.Exclusive {
		t.Errorf("Ownership = %v, want Exclusive (default for unknown files)", cf.Ownership)
	}
}
