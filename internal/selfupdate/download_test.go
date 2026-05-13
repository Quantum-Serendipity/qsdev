package selfupdate

import (
	"archive/tar"
	"archive/zip"
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

// createTestTarGz creates a .tar.gz archive containing a single file
// with the given name and content.
func createTestTarGz(t *testing.T, dir, filename, content string) string {
	t.Helper()
	archivePath := filepath.Join(dir, "test.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	data := []byte(content)
	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Size: int64(len(data)),
		Mode: 0o755,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}

	return archivePath
}

// createTestZip creates a .zip archive containing a single file
// with the given name and content.
func createTestZip(t *testing.T, dir, filename, content string) string {
	t.Helper()
	archivePath := filepath.Join(dir, "test.zip")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	w, err := zw.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	return archivePath
}

// sha256sum computes the SHA256 hex digest of a file.
func sha256sum(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestExtractFromTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := createTestTarGz(t, tmpDir, "qsdev", "#!/bin/sh\necho hello\n")

	destPath := filepath.Join(tmpDir, "extracted_qsdev")
	if err := extractFromTarGz(archivePath, "qsdev", destPath); err != nil {
		t.Fatalf("extractFromTarGz() error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(data) != "#!/bin/sh\necho hello\n" {
		t.Errorf("extracted content = %q, want %q", string(data), "#!/bin/sh\necho hello\n")
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Error("extracted binary should be executable")
	}
}

func TestExtractFromTarGz_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := createTestTarGz(t, tmpDir, "other_binary", "content")

	destPath := filepath.Join(tmpDir, "qsdev")
	err := extractFromTarGz(archivePath, "qsdev", destPath)
	if err == nil {
		t.Fatal("expected error when binary not in archive")
	}
}

func TestExtractFromZip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := createTestZip(t, tmpDir, "qsdev.exe", "MZ fake binary")

	destPath := filepath.Join(tmpDir, "extracted_qsdev.exe")
	if err := extractFromZip(archivePath, "qsdev.exe", destPath); err != nil {
		t.Fatalf("extractFromZip() error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(data) != "MZ fake binary" {
		t.Errorf("extracted content = %q, want %q", string(data), "MZ fake binary")
	}
}

func TestExtractFromZip_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := createTestZip(t, tmpDir, "other.exe", "content")

	destPath := filepath.Join(tmpDir, "qsdev.exe")
	err := extractFromZip(archivePath, "qsdev.exe", destPath)
	if err == nil {
		t.Fatal("expected error when binary not in archive")
	}
}

func TestVerifyChecksum_Pass(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file.
	testFile := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(testFile, []byte("archive content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Compute its SHA256.
	hash := sha256sum(t, testFile)

	// Write checksums file.
	checksumsFile := filepath.Join(tmpDir, "checksums.txt")
	content := fmt.Sprintf("%s  test.tar.gz\n", hash)
	if err := os.WriteFile(checksumsFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyChecksum(testFile, "test.tar.gz", checksumsFile); err != nil {
		t.Fatalf("verifyChecksum() error: %v", err)
	}
}

func TestVerifyChecksum_Fail(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(testFile, []byte("archive content"), 0o644); err != nil {
		t.Fatal(err)
	}

	checksumsFile := filepath.Join(tmpDir, "checksums.txt")
	content := "0000000000000000000000000000000000000000000000000000000000000000  test.tar.gz\n"
	if err := os.WriteFile(checksumsFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyChecksum(testFile, "test.tar.gz", checksumsFile)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestVerifyChecksum_MissingEntry(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.tar.gz")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	checksumsFile := filepath.Join(tmpDir, "checksums.txt")
	content := "abcdef1234567890  other_file.tar.gz\n"
	if err := os.WriteFile(checksumsFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyChecksum(testFile, "test.tar.gz", checksumsFile)
	if err == nil {
		t.Fatal("expected error for missing checksum entry")
	}
}

func TestFindChecksum(t *testing.T) {
	checksums := `abc123  file1.tar.gz
def456  file2.tar.gz
789abc  file3.zip
`
	tests := []struct {
		filename string
		want     string
		wantErr  bool
	}{
		{"file1.tar.gz", "abc123", false},
		{"file2.tar.gz", "def456", false},
		{"file3.zip", "789abc", false},
		{"missing.tar.gz", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got, err := findChecksum(checksums, tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("findChecksum(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("findChecksum(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDownloadAndVerify(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test tar.gz with the "binary".
	archiveDir := t.TempDir()
	archivePath := createTestTarGz(t, archiveDir, "qsdev", "#!/bin/sh\necho v2.0.0\n")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	// Compute checksum.
	hash := sha256sum(t, archivePath)
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

	binaryPath, err := DownloadAndVerify(context.Background(), release, cfg, "linux", "amd64", tmpDir)
	if err != nil {
		t.Fatalf("DownloadAndVerify() error: %v", err)
	}

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading binary: %v", err)
	}
	if string(data) != "#!/bin/sh\necho v2.0.0\n" {
		t.Errorf("binary content = %q", string(data))
	}
}

func TestDownloadAndVerify_ChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	archiveDir := t.TempDir()
	archivePath := createTestTarGz(t, archiveDir, "qsdev", "content")
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	// Wrong checksum.
	checksumsContent := "0000000000000000000000000000000000000000000000000000000000000000  qsdev_1.0.0_Linux_x86_64.tar.gz\n"

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
		Version: "1.0.0",
		Assets: []Asset{
			{Name: "qsdev_1.0.0_Linux_x86_64.tar.gz", URL: srv.URL + "/archive"},
			{Name: "checksums.txt", URL: srv.URL + "/checksums"},
		},
	}

	cfg := Config{BinaryName: "qsdev"}

	_, err = DownloadAndVerify(context.Background(), release, cfg, "linux", "amd64", tmpDir)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestDownloadAndVerify_MissingArchive(t *testing.T) {
	release := &Release{
		Version: "1.0.0",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "http://example.com/checksums"},
		},
	}

	cfg := Config{BinaryName: "qsdev"}

	_, err := DownloadAndVerify(context.Background(), release, cfg, "linux", "amd64", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing archive asset")
	}
}

func TestDownloadAndVerify_MissingChecksums(t *testing.T) {
	release := &Release{
		Version: "1.0.0",
		Assets: []Asset{
			{Name: "qsdev_1.0.0_Linux_x86_64.tar.gz", URL: "http://example.com/archive"},
		},
	}

	cfg := Config{BinaryName: "qsdev"}

	_, err := DownloadAndVerify(context.Background(), release, cfg, "linux", "amd64", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing checksums asset")
	}
}
