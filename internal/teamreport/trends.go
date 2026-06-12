package teamreport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

const historySchemaVersion = "1.0.0"

// HistoryStore persists project score history across team report runs,
// enabling trend detection and score-drop alerting.
type HistoryStore struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Entries       map[string][]TrendPoint `json:"entries"`
}

// LoadHistory reads a HistoryStore from the given file path.
// If the file does not exist, an empty store is returned without error.
// This allows callers to transparently bootstrap history on first run.
func LoadHistory(path string) (*HistoryStore, error) {
	if path == "" {
		return newHistoryStore(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newHistoryStore(), nil
		}
		return nil, fmt.Errorf("reading history file: %w", err)
	}

	var store HistoryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parsing history file: %w", err)
	}

	if store.Entries == nil {
		store.Entries = make(map[string][]TrendPoint)
	}

	return &store, nil
}

// SaveHistory writes the HistoryStore to the given file path.
func SaveHistory(path string, store *HistoryStore) error {
	if path == "" {
		return nil
	}

	store.SchemaVersion = historySchemaVersion

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling history: %w", err)
	}

	data = append(data, '\n')
	if err := os.WriteFile(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing history file: %w", err)
	}
	return nil
}

// Append adds the current project scores to the history. If a project
// already has an entry for today's date, the existing entry is updated
// (same-day deduplication) rather than creating a duplicate.
func (h *HistoryStore) Append(projects []ProjectSummary) {
	today := time.Now().UTC().Format("2006-01-02")

	for _, p := range projects {
		points := h.Entries[p.Name]

		// Same-day dedup: update the score if today already has an entry.
		found := false
		for i := range points {
			if points[i].Date == today {
				points[i].Score = p.Score.Total
				found = true
				break
			}
		}

		if !found {
			points = append(points, TrendPoint{
				Date:  today,
				Score: p.Score.Total,
			})
		}

		h.Entries[p.Name] = points
	}
}

// Prune removes entries older than maxAge from all projects. Projects
// with no remaining entries are removed from the store entirely.
func (h *HistoryStore) Prune(maxAge time.Duration) {
	cutoff := time.Now().UTC().Add(-maxAge).Format("2006-01-02")

	for project, points := range h.Entries {
		var kept []TrendPoint
		for _, pt := range points {
			if pt.Date >= cutoff {
				kept = append(kept, pt)
			}
		}
		if len(kept) == 0 {
			delete(h.Entries, project)
		} else {
			h.Entries[project] = kept
		}
	}
}

// PreviousScore returns the most recent score before today for the given
// project. This is used for score-drop alerting. Returns false if no
// previous score exists.
func (h *HistoryStore) PreviousScore(project string) (float64, bool) {
	points := h.Entries[project]
	if len(points) == 0 {
		return 0, false
	}

	today := time.Now().UTC().Format("2006-01-02")

	// Walk backwards to find the most recent entry that is not today.
	for i := len(points) - 1; i >= 0; i-- {
		if points[i].Date != today {
			return points[i].Score, true
		}
	}

	return 0, false
}

// newHistoryStore creates an empty HistoryStore with initialized fields.
func newHistoryStore() *HistoryStore {
	return &HistoryStore{
		SchemaVersion: historySchemaVersion,
		Entries:       make(map[string][]TrendPoint),
	}
}
