package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not track Unix file permission bits")
	}
	dir := t.TempDir()
	content := []byte("same content, different mode")
	relPath := "script.sh"
	absPath := filepath.Join(dir, relPath)
	// Write with mode 0o755 but store with 0o644
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

func TestRecordFragments(t *testing.T) {
	t.Parallel()
	fragments := []types.FragmentEntry{
		{Source: "devenv", Target: "devenv.nix", Content: []byte("nix content"), Priority: 100, ComposeMode: types.ComposeReplace, Provenance: types.FragmentProvenance{Timestamp: time.Now(), Reason: "test"}},
		{Source: "claudecode", Target: ".claude/settings.json", Content: []byte("settings"), Priority: 50, ComposeMode: types.ComposeReplace, Provenance: types.FragmentProvenance{Timestamp: time.Now(), Reason: "test"}},
		{Source: "devenv", Target: ".envrc", Content: []byte("envrc"), Priority: 100, ComposeMode: types.ComposeReplace, Provenance: types.FragmentProvenance{Timestamp: time.Now(), Reason: "test"}},
	}

	ledger := RecordFragments(fragments)

	if len(ledger) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(ledger))
	}
	if len(ledger["devenv.nix"]) != 1 {
		t.Errorf("devenv.nix: expected 1 entry, got %d", len(ledger["devenv.nix"]))
	}
	if ledger["devenv.nix"][0].Source != "devenv" {
		t.Errorf("source = %q, want devenv", ledger["devenv.nix"][0].Source)
	}
	if ledger["devenv.nix"][0].ContentHash == "" {
		t.Error("ContentHash should be non-empty")
	}
}

func TestRecordFragments_SortedWithinTarget(t *testing.T) {
	t.Parallel()
	fragments := []types.FragmentEntry{
		{Source: "z-addon", Target: "shared.json", Content: []byte("z"), Provenance: types.FragmentProvenance{Timestamp: time.Now()}},
		{Source: "a-addon", Target: "shared.json", Content: []byte("a"), Provenance: types.FragmentProvenance{Timestamp: time.Now()}},
	}

	ledger := RecordFragments(fragments)
	entries := ledger["shared.json"]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Source != "a-addon" || entries[1].Source != "z-addon" {
		t.Errorf("entries not sorted by source: [%s, %s]", entries[0].Source, entries[1].Source)
	}
}

func TestFragmentsBySource(t *testing.T) {
	t.Parallel()
	state := types.GeneratedState{
		Fragments: map[string][]types.FragmentLedgerEntry{
			"devenv.nix": {
				{Source: "devenv", Tag: "base"},
				{Source: "claudecode", Tag: "hooks"},
			},
			".envrc": {
				{Source: "devenv", Tag: ""},
			},
		},
	}

	got := FragmentsBySource(state, "devenv")
	if len(got) != 2 {
		t.Errorf("expected 2 devenv fragments, got %d", len(got))
	}

	got = FragmentsBySource(state, "claudecode")
	if len(got) != 1 {
		t.Errorf("expected 1 claudecode fragment, got %d", len(got))
	}

	got = FragmentsBySource(state, "nonexistent")
	if len(got) != 0 {
		t.Errorf("expected 0 fragments for nonexistent, got %d", len(got))
	}
}

func TestFragmentsByTarget(t *testing.T) {
	t.Parallel()
	state := types.GeneratedState{
		Fragments: map[string][]types.FragmentLedgerEntry{
			"devenv.nix": {{Source: "devenv"}, {Source: "security"}},
		},
	}

	got := FragmentsByTarget(state, "devenv.nix")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	got = FragmentsByTarget(state, "nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent target, got %v", got)
	}
}

func TestRemoveFragmentsBySource(t *testing.T) {
	t.Parallel()
	state := types.GeneratedState{
		Fragments: map[string][]types.FragmentLedgerEntry{
			"devenv.nix": {
				{Source: "devenv", Tag: "base"},
				{Source: "security", Tag: "floor"},
			},
			".envrc": {
				{Source: "devenv", Tag: ""},
			},
			"settings.json": {
				{Source: "claudecode", Tag: "perms"},
			},
		},
	}

	affected := RemoveFragmentsBySource(&state, "devenv")

	// Check affected paths.
	if len(affected) != 2 {
		t.Fatalf("expected 2 affected paths, got %d: %v", len(affected), affected)
	}
	// affected should be sorted
	if affected[0] != ".envrc" || affected[1] != "devenv.nix" {
		t.Errorf("affected = %v, want [.envrc, devenv.nix]", affected)
	}

	// .envrc should be deleted entirely (only had devenv fragments).
	if _, ok := state.Fragments[".envrc"]; ok {
		t.Error(".envrc should be removed from ledger")
	}

	// devenv.nix should still have security fragment.
	remaining := state.Fragments["devenv.nix"]
	if len(remaining) != 1 || remaining[0].Source != "security" {
		t.Errorf("devenv.nix: expected only security fragment, got %v", remaining)
	}

	// settings.json untouched.
	if len(state.Fragments["settings.json"]) != 1 {
		t.Error("settings.json should be untouched")
	}
}

func TestRemoveFragmentsBySource_NilFragments(t *testing.T) {
	t.Parallel()
	state := types.GeneratedState{}
	affected := RemoveFragmentsBySource(&state, "devenv")
	if affected != nil {
		t.Errorf("expected nil, got %v", affected)
	}
}

func TestGeneratedState_FragmentsBackwardCompat(t *testing.T) {
	t.Parallel()
	// Old-format state YAML without fragments field should deserialize cleanly.
	oldYAML := `
last_run: 2026-01-01T00:00:00Z
files:
  devenv.nix:
    hash: "sha256:abc123"
    strategy: overwrite
    mode: 420
template_version: ""
skill_library_version: ""
`
	var state types.GeneratedState
	if err := yaml.Unmarshal([]byte(oldYAML), &state); err != nil {
		t.Fatalf("unmarshal old format: %v", err)
	}
	if state.Fragments != nil {
		t.Errorf("Fragments should be nil for old format, got %v", state.Fragments)
	}
	if _, ok := state.Files["devenv.nix"]; !ok {
		t.Error("Files should still be parsed correctly")
	}
}

func TestGeneratedState_FragmentsRoundTrip(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	state := types.GeneratedState{
		LastRun: ts,
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "sha256:abc", Strategy: types.Overwrite, Mode: 0o644},
		},
		Fragments: map[string][]types.FragmentLedgerEntry{
			"devenv.nix": {
				{Source: "devenv", Tag: "base", Priority: 100, ComposeMode: types.ComposeReplace, ContentHash: "sha256:def", Timestamp: ts, Reason: "test"},
			},
		},
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got types.GeneratedState
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Fragments) != 1 {
		t.Fatalf("Fragments: expected 1 target, got %d", len(got.Fragments))
	}
	entry := got.Fragments["devenv.nix"][0]
	if entry.Source != "devenv" || entry.ComposeMode != types.ComposeReplace || entry.ContentHash != "sha256:def" {
		t.Errorf("round-trip mismatch: %+v", entry)
	}
}
