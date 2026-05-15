package npm

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
)

func init() {
	extlog.RegisterProvider(&Provider{})
}

var _ extlog.LogProvider = (*Provider)(nil)

// Provider discovers and parses npm debug logs from ~/.npm/_logs/.
type Provider struct{}

func (p *Provider) Name() string        { return "npm" }
func (p *Provider) DisplayName() string { return "npm" }

var npmLogRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}_\d{2}_\d{2}_\d{3}Z-debug-\d+\.log$`)

func (p *Provider) Detect(_, homeDir string) bool {
	logDir := filepath.Join(homeDir, ".npm", "_logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if npmLogRe.MatchString(e.Name()) {
			return true
		}
	}
	return false
}

func (p *Provider) Discover(_, homeDir string, since time.Time) ([]extlog.LogFile, error) {
	logDir := filepath.Join(homeDir, ".npm", "_logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, nil
	}

	var files []extlog.LogFile
	for _, e := range entries {
		if !npmLogRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(since) {
			continue
		}
		files = append(files, extlog.LogFile{
			Path:     filepath.Join(logDir, e.Name()),
			Provider: "npm",
			ModTime:  info.ModTime(),
			Size:     info.Size(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})
	return files, nil
}

var npmLineRe = regexp.MustCompile(`^(\d+)\s+(silly|verbose|info|http|timing|warn|error)\s+(.*)$`)

func (p *Provider) Parse(r io.Reader, sourceFile string) ([]extlog.LogEntry, error) {
	fileMtime := fileModTime(sourceFile)
	scanner := bufio.NewScanner(r)
	var entries []extlog.LogEntry
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		entry := extlog.LogEntry{
			Source:          "npm",
			File:            sourceFile,
			LineNumber:      lineNo,
			Timestamp:       fileMtime,
			TimestampSource: extlog.TSMtime,
		}

		if m := npmLineRe.FindStringSubmatch(line); m != nil {
			entry.Level = mapNpmLevel(m[2])
			entry.Message = m[3]
		} else {
			entry.Level = extlog.LevelUnknown
			entry.Message = line
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

func mapNpmLevel(raw string) extlog.LogLevel {
	switch strings.ToLower(raw) {
	case "silly", "verbose":
		return extlog.LevelDebug
	case "info", "http", "timing":
		return extlog.LevelInfo
	case "warn":
		return extlog.LevelWarn
	case "error":
		return extlog.LevelError
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
