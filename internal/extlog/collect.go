package extlog

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
)

// CollectionWindow defines the time range for log collection.
type CollectionWindow struct {
	Start time.Time
	End   time.Time
}

// DefaultWindow returns a window spanning the last N minutes.
func DefaultWindow(minutes int) CollectionWindow {
	now := time.Now()
	return CollectionWindow{
		Start: now.Add(-time.Duration(minutes) * time.Minute),
		End:   now.Add(1 * time.Minute),
	}
}

// CollectAll discovers and collects logs from all detected providers
// within the given time window.
func CollectAll(projectRoot, homeDir string, window CollectionWindow) (map[string][]LogEntry, []CollectionSummary) {
	return collectFrom(DefaultRegistry(), projectRoot, homeDir, window)
}

// collectFrom is CollectAll over an explicit registry.
func collectFrom(registry *Registry, projectRoot, homeDir string, window CollectionWindow) (map[string][]LogEntry, []CollectionSummary) {
	available := registry.DetectAll(projectRoot, homeDir)
	scrubber := NewScrubber(homeDir, projectRoot)

	allEntries := make(map[string][]LogEntry)
	var summaries []CollectionSummary

	for _, provider := range available {
		files, err := provider.Discover(projectRoot, homeDir, window.Start)
		if err != nil {
			summaries = append(summaries, CollectionSummary{
				Provider:         provider.Name(),
				CollectionErrors: []string{fmt.Sprintf("discover: %v", err)},
			})
			continue
		}

		summary := CollectionSummary{
			Provider:  provider.Name(),
			FileCount: len(files),
		}

		var entries []LogEntry
		for _, lf := range files {
			summary.TotalBytes += lf.Size

			f, err := os.Open(lf.Path)
			if err != nil {
				summary.CollectionErrors = append(summary.CollectionErrors,
					fmt.Sprintf("open %s: %v", lf.Path, err))
				continue
			}

			parsed, err := provider.Parse(f, lf.Path)
			f.Close()
			if err != nil {
				// Keep what was parsed before the failure: a read error late
				// in a file must not discard the entries that precede it.
				summary.CollectionErrors = append(summary.CollectionErrors,
					fmt.Sprintf("parse %s: %v", lf.Path, err))
			}

			// Private-key blocks span lines, so their state is carried across
			// the entries of one file; the per-line scrub alone would see only
			// the BEGIN line and pass the key body through.
			var keys logging.PrivateKeyLineFilter
			for i := range parsed {
				parsed[i].Message = scrubber.Scrub(keys.Filter(parsed[i].Message))
				if parsed[i].File != "" {
					parsed[i].File = scrubber.Scrub(parsed[i].File)
				}

				if !parsed[i].Timestamp.IsZero() &&
					(parsed[i].Timestamp.Before(window.Start) || parsed[i].Timestamp.After(window.End)) {
					continue
				}

				if parsed[i].Level >= LevelError {
					summary.ErrorCount++
				}
				entries = append(entries, parsed[i])
			}
		}

		summary.EntryCount = len(entries)
		if len(entries) > 0 {
			allEntries[provider.Name()] = entries
		}
		summaries = append(summaries, summary)
	}

	return allEntries, summaries
}

// FormatEntries writes log entries as text lines to w.
func FormatEntries(w io.Writer, entries []LogEntry) error {
	for _, e := range entries {
		ts := ""
		if !e.Timestamp.IsZero() {
			ts = e.Timestamp.Format("15:04:05")
		}
		_, err := io.WriteString(w, ts+" ["+e.Level.String()+"] "+e.Source+": "+e.Message+"\n")
		if err != nil {
			return err
		}
	}
	return nil
}
