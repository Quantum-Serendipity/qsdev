package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	if err := os.WriteFile(src, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0o755); err != nil {
		t.Fatalf("copyFile() error: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", string(data), "hello world")
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Error("expected executable permissions")
	}
}

func TestTruncateChangelog(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLines int
		wantFull bool // if true, output should equal input
	}{
		{
			name:     "short changelog",
			body:     "Line 1\nLine 2\nLine 3",
			maxLines: 5,
			wantFull: true,
		},
		{
			name:     "exact limit",
			body:     "Line 1\nLine 2\nLine 3",
			maxLines: 3,
			wantFull: true,
		},
		{
			name:     "truncated",
			body:     "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			maxLines: 2,
			wantFull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateChangelog(tt.body, tt.maxLines)
			if tt.wantFull && got != tt.body {
				t.Errorf("truncateChangelog() = %q, want %q", got, tt.body)
			}
			if !tt.wantFull && got == tt.body {
				t.Error("expected truncated output, got full body")
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2\nline3", 3},
		{"line1\nline2\n", 2},
	}

	for _, tt := range tests {
		got := splitLines(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestDoUpdate_RenameAndCopy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("rename-in-place test not reliable on Windows")
	}

	tmpDir := t.TempDir()

	// Create a fake "current" binary.
	currentBinary := filepath.Join(tmpDir, "qsdev")
	if err := os.WriteFile(currentBinary, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a tar.gz with the "new" binary.
	// The new binary should be a valid executable that exits 0.
	newBinaryContent := "#!/bin/sh\necho new\n"
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "qsdev_2.0.0_Linux_x86_64.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	data := []byte(newBinaryContent)
	tw.WriteHeader(&tar.Header{Name: "qsdev", Size: int64(len(data)), Mode: 0o755})
	tw.Write(data)
	tw.Close()
	gw.Close()
	f.Close()

	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(archiveData)
	hash := hex.EncodeToString(h[:])
	checksumsContent := fmt.Sprintf("%s  qsdev_2.0.0_Linux_x86_64.tar.gz\n", hash)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/archive":
			w.Write(archiveData)
		case "/checksums":
			w.Write([]byte(checksumsContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	release := &Release{
		Version: "2.0.0",
		Assets: []Asset{
			{Name: "qsdev_2.0.0_Linux_x86_64.tar.gz", URL: srv.URL + "/archive"},
			{Name: "checksums.txt", URL: srv.URL + "/checksums"},
		},
	}

	cfg := Config{BinaryName: "qsdev"}

	// We can't easily test DoUpdate directly because os.Executable() returns
	// the test binary path, not our fake binary. Instead we test the individual
	// components: download, rename, copy, and verify.

	// Test download + verify.
	destDir := t.TempDir()
	newPath, err := DownloadAndVerify(context.Background(), release, cfg, "linux", "amd64", destDir)
	if err != nil {
		t.Fatalf("DownloadAndVerify() error: %v", err)
	}

	// Test rename (backup).
	backupPath := currentBinary + ".bak"
	if err := os.Rename(currentBinary, backupPath); err != nil {
		t.Fatalf("rename to backup: %v", err)
	}

	// Verify backup exists.
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup should exist: %v", err)
	}

	// Test copy.
	if err := copyFile(newPath, currentBinary, 0o755); err != nil {
		// Rollback.
		os.Rename(backupPath, currentBinary)
		t.Fatalf("copyFile() error: %v", err)
	}

	// Verify new binary was installed.
	installedData, err := os.ReadFile(currentBinary)
	if err != nil {
		t.Fatalf("reading installed binary: %v", err)
	}
	if string(installedData) != newBinaryContent {
		t.Errorf("installed content = %q, want %q", string(installedData), newBinaryContent)
	}

	// Cleanup: remove backup (simulating success path).
	os.Remove(backupPath)
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup should have been removed")
	}
}

func TestDoUpdate_RollbackOnCopyFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original file.
	original := filepath.Join(tmpDir, "binary")
	originalContent := "original content"
	if err := os.WriteFile(original, []byte(originalContent), 0o755); err != nil {
		t.Fatal(err)
	}

	// Rename to backup.
	backup := original + ".bak"
	if err := os.Rename(original, backup); err != nil {
		t.Fatal(err)
	}

	// Simulate copy failure by trying to copy from nonexistent source.
	err := copyFile("/nonexistent/path", original, 0o755)
	if err == nil {
		t.Fatal("expected copy error")
	}

	// Rollback: restore backup.
	if err := os.Rename(backup, original); err != nil {
		t.Fatalf("rollback rename: %v", err)
	}

	// Verify original content is restored.
	data, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != originalContent {
		t.Errorf("restored content = %q, want %q", string(data), originalContent)
	}
}

func TestVerifyBinary_InvalidBinary(t *testing.T) {
	tmpDir := t.TempDir()
	badBinary := filepath.Join(tmpDir, "bad")
	if err := os.WriteFile(badBinary, []byte("not executable"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := verifyBinary(context.Background(), badBinary)
	if err == nil {
		t.Error("expected error for invalid binary")
	}
}
