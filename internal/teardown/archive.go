package teardown

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateArchive creates a tar.gz archive of the given files relative to
// projectRoot. Files that do not exist are silently skipped. Returns the
// path to the created archive.
func CreateArchive(projectRoot string, files []string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf(".qsdev-archive-%s.tar.gz", timestamp)
	archivePath := filepath.Join(projectRoot, archiveName)

	outFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating archive file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, relPath := range files {
		absPath := filepath.Join(projectRoot, relPath)
		if err := addFileToTar(tw, absPath, relPath); err != nil {
			// Skip files that don't exist.
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("adding %s to archive: %w", relPath, err)
		}
	}

	return archivePath, nil
}

func addFileToTar(tw *tar.Writer, absPath, relPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relPath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}
