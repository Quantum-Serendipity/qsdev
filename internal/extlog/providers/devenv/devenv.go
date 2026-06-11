package devenv

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/extlog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func init() {
	extlog.RegisterProvider(&Provider{})
}

var _ extlog.LogProvider = (*Provider)(nil)

// Provider discovers and parses devenv capture files from .qsdev/logs/capture/.
type Provider struct{}

func (p *Provider) Name() string        { return "devenv" }
func (p *Provider) DisplayName() string { return "devenv" }

func (p *Provider) Detect(projectRoot, _ string) bool {
	captureDir := filepath.Join(projectRoot, "."+branding.Get().AppName, "logs", "capture")
	entries, err := os.ReadDir(captureDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "devenv-") {
			return true
		}
	}
	return false
}

func (p *Provider) Discover(projectRoot, _ string, since time.Time) ([]extlog.LogFile, error) {
	captureDir := filepath.Join(projectRoot, "."+branding.Get().AppName, "logs", "capture")
	entries, err := os.ReadDir(captureDir)
	if err != nil {
		return nil, nil
	}

	var files []extlog.LogFile
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "devenv-") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(since) {
			continue
		}
		files = append(files, extlog.LogFile{
			Path:     filepath.Join(captureDir, e.Name()),
			Provider: "devenv",
			ModTime:  info.ModTime(),
			Size:     info.Size(),
		})
	}
	return files, nil
}

func (p *Provider) Parse(r io.Reader, sourceFile string) ([]extlog.LogEntry, error) {
	fileMtime := extlog.FileModTime(sourceFile)
	scanner := bufio.NewScanner(r)
	var entries []extlog.LogEntry
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := extlog.StripANSI(scanner.Text())

		entry := extlog.LogEntry{
			Source:          "devenv",
			File:            sourceFile,
			LineNumber:      lineNo,
			Timestamp:       fileMtime,
			TimestampSource: extlog.TSMtime,
			Level:           detectLevel(line),
			Message:         line,
		}

		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func detectLevel(line string) extlog.LogLevel {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error"):
		return extlog.LevelError
	case strings.Contains(lower, "warning"):
		return extlog.LevelWarn
	case strings.Contains(lower, "debug"):
		return extlog.LevelDebug
	default:
		return extlog.LevelInfo
	}
}
