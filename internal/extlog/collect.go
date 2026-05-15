package extlog

import (
	"io"
	"os"
	"time"
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
	registry := DefaultRegistry()
	available := registry.DetectAll(projectRoot, homeDir)
	scrubber := NewScrubber(homeDir, projectRoot)

	allEntries := make(map[string][]LogEntry)
	var summaries []CollectionSummary

	for _, provider := range available {
		files, err := provider.Discover(projectRoot, homeDir, window.Start)
		if err != nil {
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
				continue
			}

			parsed, err := provider.Parse(f, lf.Path)
			f.Close()
			if err != nil {
				continue
			}

			for i := range parsed {
				parsed[i].Message = scrubber.Scrub(parsed[i].Message)
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
