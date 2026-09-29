package teardown

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// CreateArchive creates a tar.gz archive of the given files relative to
// projectRoot. Files that do not exist are silently skipped. Returns the
// path to the created archive.
func CreateArchive(projectRoot string, files []string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf(".%s-archive-%s.tar.gz", branding.Get().AppName, timestamp)
	return createArchive(projectRoot, archiveName, files)
}

// createArchive writes the archive to projectRoot/archiveName. The archive is
// always a new file: O_EXCL refuses an existing file and, because it never
// follows a symbolic link, a committed symlink at the archive path cannot
// redirect the write outside the project.
func createArchive(projectRoot, archiveName string, files []string) (string, error) {
	archivePath := filepath.Join(projectRoot, archiveName)

	outFile, err := os.OpenFile(archivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileutil.ModeReadWrite)
	if err != nil {
		return "", fmt.Errorf("creating archive file: %w", err)
	}

	gw := gzip.NewWriter(outFile)
	tw := tar.NewWriter(gw)

	var writeErr error
	for _, relPath := range files {
		absPath := filepath.Join(projectRoot, relPath)
		if err := addFileToTar(tw, absPath, relPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			writeErr = fmt.Errorf("adding %s to archive: %w", relPath, err)
			break
		}
	}

	// Close in order: tar -> gzip -> file. Each flush may fail on full disk.
	if err := tw.Close(); err != nil && writeErr == nil {
		writeErr = fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gw.Close(); err != nil && writeErr == nil {
		writeErr = fmt.Errorf("closing gzip writer: %w", err)
	}
	if err := outFile.Close(); err != nil && writeErr == nil {
		writeErr = fmt.Errorf("closing archive file: %w", err)
	}

	if writeErr != nil {
		os.Remove(archivePath)
		return "", writeErr
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
