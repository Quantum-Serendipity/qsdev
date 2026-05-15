package teardown

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateArchive_BasicFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some test files.
	files := map[string]string{
		"file1.txt":           "content1",
		"subdir/file2.txt":    "content2",
		"another/deep/f3.txt": "content3",
	}

	for relPath, content := range files {
		absPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	relPaths := make([]string, 0, len(files))
	for p := range files {
		relPaths = append(relPaths, p)
	}

	archivePath, err := CreateArchive(dir, relPaths)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the archive was created.
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive file should exist: %v", err)
	}

	// Verify the archive name format.
	archiveName := filepath.Base(archivePath)
	if !strings.HasPrefix(archiveName, ".qsdev-archive-") || !strings.HasSuffix(archiveName, ".tar.gz") {
		t.Errorf("archive name %q doesn't match expected format", archiveName)
	}

	// Open and read the archive.
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	foundFiles := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		foundFiles[hdr.Name] = string(data)
	}

	// Verify all files are in the archive.
	for relPath, expectedContent := range files {
		content, ok := foundFiles[relPath]
		if !ok {
			t.Errorf("file %q not found in archive", relPath)
			continue
		}
		if content != expectedContent {
			t.Errorf("file %q content = %q, want %q", relPath, content, expectedContent)
		}
	}
}

func TestCreateArchive_SkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create only one of two files.
	if err := os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("here"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath, err := CreateArchive(dir, []string{"exists.txt", "missing.txt"})
	if err != nil {
		t.Fatal(err)
	}

	// Read the archive and verify only the existing file is present.
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	count := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "missing.txt" {
			t.Errorf("missing.txt should not be in the archive")
		}
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 file in archive, got %d", count)
	}
}

func TestCreateArchive_EmptyFileList(t *testing.T) {
	dir := t.TempDir()

	archivePath, err := CreateArchive(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Archive should still be created (empty tar.gz).
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive file should exist even with empty file list: %v", err)
	}
}
