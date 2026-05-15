package logging

import (
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
			os.Remove(filepath.Join(dir, e.Name()))
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
		os.Remove(filepath.Join(dir, logs[i].name))
	}
}
