package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadAndVerify downloads the appropriate archive for the given OS/arch
// from the release, verifies its SHA256 checksum against checksums.txt,
// extracts the binary, and returns the path to the extracted binary.
func DownloadAndVerify(ctx context.Context, release *Release, cfg Config, targetOS, targetArch, destDir string) (string, error) {
	archiveName := ArchiveFilename(cfg.BinaryName, release.Version, targetOS, targetArch)

	// Find the archive asset.
	archiveURL := ""
	checksumsURL := ""
	for _, a := range release.Assets {
		if a.Name == archiveName {
			archiveURL = a.URL
		}
		if a.Name == "checksums.txt" {
			checksumsURL = a.URL
		}
	}
	if archiveURL == "" {
		return "", fmt.Errorf("no asset found matching %q in release %s", archiveName, release.Version)
	}
	if checksumsURL == "" {
		return "", fmt.Errorf("no checksums.txt found in release %s", release.Version)
	}

	// Create temp directory for downloads.
	tmpDir, err := os.MkdirTemp("", "qsdev-update-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download the archive.
	archivePath := filepath.Join(tmpDir, archiveName)
	if err := downloadFile(ctx, archiveURL, archivePath); err != nil {
		return "", fmt.Errorf("downloading archive: %w", err)
	}

	// Download checksums.
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(ctx, checksumsURL, checksumsPath); err != nil {
		return "", fmt.Errorf("downloading checksums: %w", err)
	}

	// Verify checksum.
	if err := verifyChecksum(archivePath, archiveName, checksumsPath); err != nil {
		return "", err
	}

	// Extract binary from archive.
	binaryName := cfg.BinaryName
	if targetOS == "windows" {
		binaryName += ".exe"
	}

	extractedPath := filepath.Join(destDir, binaryName)
	if targetOS == "windows" {
		err = extractFromZip(archivePath, binaryName, extractedPath)
	} else {
		err = extractFromTarGz(archivePath, binaryName, extractedPath)
	}
	if err != nil {
		return "", fmt.Errorf("extracting binary: %w", err)
	}

	return extractedPath, nil
}

// downloadFile downloads a URL to a local file path.
func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", url, err)
	}

	// Support optional GITHUB_TOKEN for private repos.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s returned HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return f.Close()
}

// verifyChecksum checks the SHA256 of the archive against the checksums file.
func verifyChecksum(archivePath, archiveName, checksumsPath string) error {
	// Compute the SHA256 of the downloaded archive.
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	// Read the expected checksum from checksums.txt.
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("reading checksums file: %w", err)
	}

	expectedHash, err := findChecksum(string(data), archiveName)
	if err != nil {
		return err
	}

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
			archiveName, expectedHash, actualHash)
	}

	return nil
}

// findChecksum parses a checksums.txt file and returns the hash for the
// given filename. The expected format is "hash  filename" per line.
func findChecksum(checksums, filename string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>" or "<hash> <filename>"
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s in checksums.txt", filename)
}

// extractFromTarGz extracts a single named file from a .tar.gz archive.
func extractFromTarGz(archivePath, binaryName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive %s: %w", archivePath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Match by base name — the binary might be at the root or in a directory.
		if filepath.Base(header.Name) == binaryName && header.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return fmt.Errorf("creating output file %s: %w", destPath, err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return fmt.Errorf("extracting %s: %w", binaryName, err)
			}
			return out.Close()
		}
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

// extractFromZip extracts a single named file from a .zip archive.
func extractFromZip(archivePath, binaryName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip archive %s: %w", archivePath, err)
	}
	defer r.Close()

	for _, zf := range r.File {
		if filepath.Base(zf.Name) == binaryName {
			src, err := zf.Open()
			if err != nil {
				return fmt.Errorf("opening %s in zip: %w", zf.Name, err)
			}
			defer src.Close()

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return fmt.Errorf("creating output file %s: %w", destPath, err)
			}
			defer out.Close()

			if _, err := io.Copy(out, src); err != nil {
				return fmt.Errorf("extracting %s: %w", binaryName, err)
			}
			return out.Close()
		}
	}

	return fmt.Errorf("binary %q not found in zip archive", binaryName)
}
