package extlog

import (
	"os"
	"time"
)

// FileModTime returns the modification time of the file at path, or the zero
// time if the file cannot be stat'd. This is used by log providers as a
// fallback timestamp when no inline timestamp is available.
func FileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
