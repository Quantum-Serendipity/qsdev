package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultMaxFiles = 50

// CleanOldLogs deletes .jsonl files in dir older than maxAge.
func CleanOldLogs(dir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, e.Name())
			if err := os.Remove(path); err != nil {
				slog.Debug("failed to prune old log file", "path", path, "error", err)
			}
		}
	}
	return nil
}

// pruneExcessLogs removes the oldest .jsonl files in dir if the count
// exceeds maxFiles.
func pruneExcessLogs(dir string, maxFiles int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type fileInfo struct {
		name    string
		modTime time.Time
	}

	var logs []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		logs = append(logs, fileInfo{name: e.Name(), modTime: info.ModTime()})
	}

	if len(logs) <= maxFiles {
		return
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].modTime.Before(logs[j].modTime)
	})

	toRemove := len(logs) - maxFiles
	for i := 0; i < toRemove; i++ {
		path := filepath.Join(dir, logs[i].name)
		if err := os.Remove(path); err != nil {
			slog.Debug("failed to prune excess log file", "path", path, "error", err)
		}
	}
}
