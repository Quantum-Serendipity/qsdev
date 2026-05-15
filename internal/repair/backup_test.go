package repair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateBackup_CreatesFile(t *testing.T) {
	root := t.TempDir()

	// Create the source file.
	content := []byte("original content")
	srcRel := "config.yaml"
	if err := os.WriteFile(filepath.Join(root, srcRel), content, 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	backupPath, err := createBackup(root, srcRel)
	if err != nil {
		t.Fatalf("createBackup: %v", err)
	}

	// Verify backup file exists.
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file does not exist: %v", err)
	}

	// Verify content matches.
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("backup content = %q, want %q", string(data), string(content))
	}

	// Verify backup path is under .qsdev/backups.
	dir := backupDir(root)
	if !strings.HasPrefix(backupPath, dir) {
		t.Errorf("backup path %q not under %q", backupPath, dir)
	}

	// Verify filename pattern.
	base := filepath.Base(backupPath)
	if !strings.HasPrefix(base, "config.yaml.") || !strings.HasSuffix(base, ".bak") {
		t.Errorf("unexpected backup filename: %s", base)
	}
}

func TestCreateBackup_CreatesDirectory(t *testing.T) {
	root := t.TempDir()

	// Create the source file.
	srcRel := "some/nested/file.txt"
	srcAbs := filepath.Join(root, srcRel)
	if err := os.MkdirAll(filepath.Dir(srcAbs), 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}
	if err := os.WriteFile(srcAbs, []byte("data"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	backupPath, err := createBackup(root, srcRel)
	if err != nil {
		t.Fatalf("createBackup: %v", err)
	}

	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file does not exist: %v", err)
	}
}

func TestCreateBackup_SourceMissing(t *testing.T) {
	root := t.TempDir()

	_, err := createBackup(root, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for missing source, got nil")
	}
}

func TestPruneBackups_KeepsLastN(t *testing.T) {
	root := t.TempDir()
	dir := backupDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}

	// Create 7 backup files with distinct timestamps.
	baseName := "config.yaml"
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Hour).Format("20060102T150405")
		name := baseName + "." + ts + ".bak"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("creating backup %d: %v", i, err)
		}
	}

	// Prune to keep 3.
	if err := pruneBackups(root, baseName, 3); err != nil {
		t.Fatalf("pruneBackups: %v", err)
	}

	// Count remaining.
	entries, _ := os.ReadDir(dir)
	var remaining []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), baseName+".") && strings.HasSuffix(e.Name(), ".bak") {
			remaining = append(remaining, e.Name())
		}
	}

	if len(remaining) != 3 {
		t.Errorf("got %d remaining backups, want 3: %v", len(remaining), remaining)
	}

	// Verify the newest 3 survived (timestamps T040000, T050000, T060000).
	for _, name := range remaining {
		// Extract timestamp portion.
		ts := strings.TrimPrefix(name, baseName+".")
		ts = strings.TrimSuffix(ts, ".bak")
		hour := ts[9:11] // HH portion
		if hour != "04" && hour != "05" && hour != "06" {
			t.Errorf("unexpected surviving backup: %s (expected hours 04, 05, or 06)", name)
		}
	}
}

func TestPruneBackups_NothingToRemove(t *testing.T) {
	root := t.TempDir()
	dir := backupDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}

	// Create 2 backups and keep 5.
	baseName := "test.txt"
	for i := 0; i < 2; i++ {
		ts := time.Date(2025, 1, 1, i, 0, 0, 0, time.UTC).Format("20060102T150405")
		name := baseName + "." + ts + ".bak"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("creating backup %d: %v", i, err)
		}
	}

	if err := pruneBackups(root, baseName, 5); err != nil {
		t.Fatalf("pruneBackups: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestPruneBackups_NoBackupDir(t *testing.T) {
	root := t.TempDir()

	// Should not error when the backup directory doesn't exist.
	if err := pruneBackups(root, "file.txt", 5); err != nil {
		t.Fatalf("pruneBackups with no dir: %v", err)
	}
}

func TestPruneBackups_IgnoresUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	dir := backupDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating dir: %v", err)
	}

	// Create backups for "a.txt" and an unrelated file.
	baseName := "a.txt"
	for i := 0; i < 3; i++ {
		ts := time.Date(2025, 1, 1, i, 0, 0, 0, time.UTC).Format("20060102T150405")
		name := baseName + "." + ts + ".bak"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("creating backup %d: %v", i, err)
		}
	}
	// Unrelated file.
	if err := os.WriteFile(filepath.Join(dir, "other.txt.20250101T000000.bak"), []byte("data"), 0o644); err != nil {
		t.Fatalf("creating unrelated backup: %v", err)
	}

	if err := pruneBackups(root, baseName, 1); err != nil {
		t.Fatalf("pruneBackups: %v", err)
	}

	// Should have 1 backup for a.txt + 1 unrelated = 2 files.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestBackupDir(t *testing.T) {
	got := backupDir("/project")
	want := filepath.Join("/project", ".qsdev", "backups")
	if got != want {
		t.Errorf("backupDir = %q, want %q", got, want)
	}
}
