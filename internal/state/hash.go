package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// HashPrefix is the algorithm prefix prepended to all content hashes.
const HashPrefix = "sha256:"

// ComputeHash returns the SHA-256 hash of content as "sha256:<64-char-lowercase-hex>".
func ComputeHash(content []byte) string {
	sum := sha256.Sum256(content)
	return HashPrefix + hex.EncodeToString(sum[:])
}

// ComputeFileHash reads the file at path and returns its content hash.
func ComputeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("computing file hash for %s: %w", path, err)
	}
	return ComputeHash(data), nil
}
