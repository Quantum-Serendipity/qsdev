package generic

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
)

var _ extlog.LogProvider = (*Provider)(nil)

// Provider is a fallback parser for arbitrary log files specified
// via --log-file. It is NOT auto-registered — it's used explicitly.
type Provider struct {
	FilePath string
}

func (p *Provider) Name() string        { return "generic" }
func (p *Provider) DisplayName() string { return "log file" }

func (p *Provider) Detect(_, _ string) bool {
	if p.FilePath == "" {
		return false
	}
	_, err := os.Stat(p.FilePath)
	return err == nil
}

func (p *Provider) Discover(_, _ string, _ time.Time) ([]extlog.LogFile, error) {
	if p.FilePath == "" {
		return nil, nil
	}
	info, err := os.Stat(p.FilePath)
	if err != nil {
		return nil, err
	}
	return []extlog.LogFile{{
		Path:     p.FilePath,
		Provider: "generic",
		ModTime:  info.ModTime(),
		Size:     info.Size(),
	}}, nil
}

var (
	timestampRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	levelRe     = regexp.MustCompile(`(?i)\b(DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL)\b`)
)

func (p *Provider) Parse(r io.Reader, sourceFile string) ([]extlog.LogEntry, error) {
	fileMtime := fileModTime(sourceFile)
	scanner := bufio.NewScanner(r)
	var entries []extlog.LogEntry
	lineNo := 0
	var lastTS time.Time

	for scanner.Scan() {
		lineNo++
		line := extlog.StripANSI(scanner.Text())

		entry := extlog.LogEntry{
			Source:     "generic",
			File:       sourceFile,
			LineNumber: lineNo,
			Message:    line,
		}

		if ts := timestampRe.FindString(line); ts != "" {
			for _, layout := range []string{
				time.RFC3339,
				"2006-01-02T15:04:05",
				"2006-01-02 15:04:05",
			} {
				if t, err := time.Parse(layout, ts); err == nil {
					entry.Timestamp = t
					entry.TimestampSource = extlog.TSParsed
					lastTS = t
					break
				}
			}
		}
		if entry.Timestamp.IsZero() && !lastTS.IsZero() {
			entry.Timestamp = lastTS
			entry.TimestampSource = extlog.TSCarried
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = fileMtime
			entry.TimestampSource = extlog.TSMtime
		}

		if m := levelRe.FindString(line); m != "" {
			entry.Level = mapGenericLevel(m)
		} else {
			entry.Level = extlog.LevelUnknown
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

func mapGenericLevel(raw string) extlog.LogLevel {
	switch strings.ToUpper(raw) {
	case "DEBUG":
		return extlog.LevelDebug
	case "INFO":
		return extlog.LevelInfo
	case "WARN", "WARNING":
		return extlog.LevelWarn
	case "ERROR":
		return extlog.LevelError
	case "FATAL":
		return extlog.LevelFatal
	default:
		return extlog.LevelUnknown
	}
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
