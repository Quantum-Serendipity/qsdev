package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeHash_Deterministic(t *testing.T) {
	content := []byte("deterministic content for hashing")
	first := ComputeHash(content)
	for i := 0; i < 100; i++ {
		got := ComputeHash(content)
		if got != first {
			t.Fatalf("iteration %d: got %q, want %q", i, got, first)
		}
	}
}

func TestComputeHash_DifferentContentDifferentHash(t *testing.T) {
	h1 := ComputeHash([]byte("content A"))
	h2 := ComputeHash([]byte("content B"))
	if h1 == h2 {
		t.Fatalf("expected different hashes, both were %q", h1)
	}
}

func TestComputeHash_EmptyContent(t *testing.T) {
	const want = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got := ComputeHash([]byte{})
	if got != want {
		t.Fatalf("empty content hash: got %q, want %q", got, want)
	}
}

func TestComputeHash_HasPrefix(t *testing.T) {
	h := ComputeHash([]byte("test"))
	if !strings.HasPrefix(h, HashPrefix) {
		t.Fatalf("hash %q does not start with %q", h, HashPrefix)
	}
}

func TestComputeHash_HexPartLength(t *testing.T) {
	h := ComputeHash([]byte("test"))
	hex := strings.TrimPrefix(h, HashPrefix)
	if len(hex) != 64 {
		t.Fatalf("hex part length: got %d, want 64 (hex=%q)", len(hex), hex)
	}
}

func TestComputeFileHash_MatchesComputeHash(t *testing.T) {
	content := []byte("file content for hash comparison")
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	fileHash, err := ComputeFileHash(path)
	if err != nil {
		t.Fatal(err)
	}

	memHash := ComputeHash(content)
	if fileHash != memHash {
		t.Fatalf("ComputeFileHash=%q != ComputeHash=%q", fileHash, memHash)
	}
}

func TestComputeFileHash_NonExistent(t *testing.T) {
	_, err := ComputeFileHash("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}
