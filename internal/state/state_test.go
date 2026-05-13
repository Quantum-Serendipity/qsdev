package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestRecordFiles_ThreeFiles(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "a.txt", Content: []byte("aaa"), Mode: 0o644, Strategy: types.Overwrite},
		{Path: "b.sh", Content: []byte("bbb"), Mode: 0o755, Strategy: types.Skip},
		{Path: "c.yaml", Content: []byte("ccc"), Mode: 0o600, Strategy: types.Merge},
	}

	state := RecordFiles(files)

	if len(state.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(state.Files))
	}

	for _, f := range files {
		fs, ok := state.Files[f.Path]
		if !ok {
			t.Fatalf("missing file %q in state", f.Path)
		}
		expectedHash := ComputeHash(f.Content)
		if fs.Hash != expectedHash {
			t.Errorf("file %q hash: got %q, want %q", f.Path, fs.Hash, expectedHash)
		}
	}
}

func TestRecordFiles_LastRunCloseToNow(t *testing.T) {
	before := time.Now().UTC()
	state := RecordFiles([]types.GeneratedFile{
		{Path: "x.txt", Content: []byte("x"), Mode: 0o644, Strategy: types.Overwrite},
	})
	after := time.Now().UTC()

	if state.LastRun.Before(before) || state.LastRun.After(after) {
		t.Fatalf("LastRun %v not between %v and %v", state.LastRun, before, after)
	}
}

func TestRecordFiles_PreservesStrategyAndMode(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "script.sh", Content: []byte("#!/bin/sh"), Mode: 0o755, Strategy: types.SectionMarker},
	}
	state := RecordFiles(files)

	fs := state.Files["script.sh"]
	if fs.Strategy != types.SectionMarker {
		t.Errorf("strategy: got %v, want %v", fs.Strategy, types.SectionMarker)
	}
	if fs.Mode != 0o755 {
		t.Errorf("mode: got %o, want %o", fs.Mode, 0o755)
	}
}

func TestRecordFiles_StoresBaseContentForThreeWayMerge(t *testing.T) {
	files := []types.GeneratedFile{
		{Path: "settings.json", Content: []byte(`{"permissions":{}}`), Mode: 0o644, Strategy: types.ThreeWayMerge},
		{Path: "devenv.nix", Content: []byte("{ pkgs, ... }: {}"), Mode: 0o644, Strategy: types.Overwrite},
	}
	state := RecordFiles(files)

	// ThreeWayMerge file should have BaseContent
	settingsState := state.Files["settings.json"]
	if settingsState.BaseContent == nil {
		t.Error("expected BaseContent for ThreeWayMerge file, got nil")
	}
	if string(settingsState.BaseContent) != `{"permissions":{}}` {
		t.Errorf("BaseContent mismatch: got %q", string(settingsState.BaseContent))
	}

	// Non-ThreeWayMerge file should NOT have BaseContent
	nixState := state.Files["devenv.nix"]
	if nixState.BaseContent != nil {
		t.Errorf("expected nil BaseContent for Overwrite file, got %q", string(nixState.BaseContent))
	}
}

func TestCheckModified_Unmodified(t *testing.T) {
	dir := t.TempDir()
	content := []byte("unchanged content")
	relPath := "file.txt"
	absPath := filepath.Join(dir, relPath)
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     ComputeHash(content),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	results := CheckModified(stored, dir)
	fs, ok := results[relPath]
	if !ok {
		t.Fatalf("missing result for %q", relPath)
	}
	if fs.Status != types.Unmodified {
		t.Errorf("expected Unmodified, got %v", fs.Status)
	}
}

func TestCheckModified_ModifiedContent(t *testing.T) {
	dir := t.TempDir()
	relPath := "file.txt"
	absPath := filepath.Join(dir, relPath)

	original := []byte("original content")
	modified := []byte("modified content")
	if err := os.WriteFile(absPath, modified, 0o644); err != nil {
		t.Fatal(err)
	}

	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     ComputeHash(original),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	results := CheckModified(stored, dir)
	fs := results[relPath]
	if fs.Status != types.Modified {
		t.Errorf("expected Modified, got %v", fs.Status)
	}
}

func TestCheckModified_Deleted(t *testing.T) {
	dir := t.TempDir()
	relPath := "gone.txt"

	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     ComputeHash([]byte("was here")),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	results := CheckModified(stored, dir)
	fs := results[relPath]
	if fs.Status != types.Deleted {
		t.Errorf("expected Deleted, got %v", fs.Status)
	}
}

func TestCheckModified_PermissionChange(t *testing.T) {
	dir := t.TempDir()
	content := []byte("same content, different mode")
	relPath := "script.sh"
	absPath := filepath.Join(dir, relPath)
	// Write with mode 0755 but store with 0644
	if err := os.WriteFile(absPath, content, 0o755); err != nil {
		t.Fatal(err)
	}

	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     ComputeHash(content),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	results := CheckModified(stored, dir)
	fs := results[relPath]
	if fs.Status != types.Modified {
		t.Errorf("expected Modified for permission change, got %v", fs.Status)
	}
}

func TestCheckModified_EmptyState(t *testing.T) {
	results := CheckModified(types.GeneratedState{}, "/tmp")
	if len(results) != 0 {
		t.Fatalf("expected empty results for empty state, got %d entries", len(results))
	}
}

func TestCheckModified_ReadError(t *testing.T) {
	dir := t.TempDir()
	relPath := "unreadable.txt"
	absPath := filepath.Join(dir, relPath)

	// Create a directory where a file is expected - Stat will succeed but
	// reading will fail when computing the hash.
	if err := os.Mkdir(absPath, 0o755); err != nil {
		t.Fatal(err)
	}

	stored := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     ComputeHash([]byte("something")),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	results := CheckModified(stored, dir)
	fs := results[relPath]
	if fs.Status != types.Unknown {
		t.Errorf("expected Unknown for read error, got %v", fs.Status)
	}
	if fs.Error == nil {
		t.Error("expected non-nil error for read error case")
	}
}
