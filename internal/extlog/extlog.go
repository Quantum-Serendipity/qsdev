// Package extlog provides external log source detection, normalization,
// and privacy scrubbing for inclusion in bug reports.
package extlog

import (
	"io"
	"time"
)

// LogLevel represents a normalized severity level.
type LogLevel int

const (
	LevelUnknown LogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the level name.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// TimestampSource indicates how a log entry's timestamp was determined.
type TimestampSource string

const (
	TSParsed  TimestampSource = "parsed"
	TSCarried TimestampSource = "carried"
	TSMtime   TimestampSource = "mtime"
)

// LogEntry is the normalized representation of a single external log line.
type LogEntry struct {
	Timestamp       time.Time       `json:"ts"`
	TimestampSource TimestampSource `json:"ts_source"`
	Level           LogLevel        `json:"level"`
	Source          string          `json:"source"`
	Message         string          `json:"msg"`
	File            string          `json:"file,omitempty"`
	LineNumber      int             `json:"line,omitempty"`
}

// LogFile describes a discovered external log file.
type LogFile struct {
	Path     string
	Provider string
	ModTime  time.Time
	Size     int64
}

// CollectionSummary describes what was found for display in the wizard.
type CollectionSummary struct {
	Provider   string
	FileCount  int
	TotalBytes int64
	EntryCount int
	ErrorCount int
}

// LogProvider is the interface that external log source modules implement.
type LogProvider interface {
	Name() string
	DisplayName() string
	Detect(projectRoot, homeDir string) bool
	Discover(projectRoot, homeDir string, since time.Time) ([]LogFile, error)
	Parse(r io.Reader, sourceFile string) ([]LogEntry, error)
}
