package nix

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
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func init() {
	extlog.RegisterProvider(&Provider{})
}

var _ extlog.LogProvider = (*Provider)(nil)

// Provider discovers and parses nix build logs and devenv capture files.
type Provider struct{}

func (p *Provider) Name() string        { return "nix" }
func (p *Provider) DisplayName() string { return "nix build" }

func (p *Provider) Detect(projectRoot, homeDir string) bool {
	captureDir := filepath.Join(projectRoot, "."+branding.Get().AppName, "logs", "capture")
	entries, _ := os.ReadDir(captureDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "nix-") {
			return true
		}
	}

	// Check /tmp for nix build logs.
	entries, _ = os.ReadDir("/tmp")
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "nix-build-") {
			return true
		}
	}
	return false
}

func (p *Provider) Discover(projectRoot, _ string, since time.Time) ([]extlog.LogFile, error) {
	var files []extlog.LogFile

	// Check capture directory.
	captureDir := filepath.Join(projectRoot, "."+branding.Get().AppName, "logs", "capture")
	if entries, err := os.ReadDir(captureDir); err == nil {
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), "nix-") {
				continue
			}
			info, err := e.Info()
			if err != nil || info.ModTime().Before(since) {
				continue
			}
			files = append(files, extlog.LogFile{
				Path:     filepath.Join(captureDir, e.Name()),
				Provider: "nix",
				ModTime:  info.ModTime(),
				Size:     info.Size(),
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})
	return files, nil
}

var nixLevelRe = regexp.MustCompile(`^(error|warning|trace):\s*(.*)$`)

func (p *Provider) Parse(r io.Reader, sourceFile string) ([]extlog.LogEntry, error) {
	fileMtime := extlog.FileModTime(sourceFile)
	scanner := bufio.NewScanner(r)
	var entries []extlog.LogEntry
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := extlog.StripANSI(scanner.Text())

		entry := extlog.LogEntry{
			Source:          "nix",
			File:            sourceFile,
			LineNumber:      lineNo,
			Timestamp:       fileMtime,
			TimestampSource: extlog.TSMtime,
		}

		if m := nixLevelRe.FindStringSubmatch(line); m != nil {
			entry.Level = mapNixLevel(m[1])
			entry.Message = m[2]
		} else {
			entry.Level = extlog.LevelInfo
			entry.Message = line
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

func mapNixLevel(raw string) extlog.LogLevel {
	switch strings.ToLower(raw) {
	case "trace":
		return extlog.LevelDebug
	case "warning":
		return extlog.LevelWarn
	case "error":
		return extlog.LevelError
	default:
		return extlog.LevelInfo
	}
}
