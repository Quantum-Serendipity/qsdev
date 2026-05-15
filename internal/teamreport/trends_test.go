package teamreport

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendNewProject(t *testing.T) {
	store := newHistoryStore()

	projects := []ProjectSummary{
		projectSummaryHelper("alpha", 85.0, true, true, 0, 0, "v1.0.0", time.Now()),
	}

	store.Append(projects)

	points, ok := store.Entries["alpha"]
	if !ok {
		t.Fatal("expected entries for 'alpha'")
	}

	if len(points) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(points))
	}

	if points[0].Score != 85.0 {
		t.Errorf("expected score 85.0, got %.1f", points[0].Score)
	}

	today := time.Now().UTC().Format("2006-01-02")
	if points[0].Date != today {
		t.Errorf("expected date %s, got %s", today, points[0].Date)
	}
}

func TestAppendExistingProject(t *testing.T) {
	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	store := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"alpha": {{Date: yesterday, Score: 80.0}},
		},
	}

	projects := []ProjectSummary{
		projectSummaryHelper("alpha", 85.0, true, true, 0, 0, "v1.0.0", time.Now()),
	}

	store.Append(projects)

	points := store.Entries["alpha"]
	if len(points) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(points))
	}

	if points[0].Score != 80.0 {
		t.Errorf("expected first point score 80.0, got %.1f", points[0].Score)
	}
	if points[1].Score != 85.0 {
		t.Errorf("expected second point score 85.0, got %.1f", points[1].Score)
	}
}

func TestAppendSameDayDedup(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	store := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"alpha": {{Date: today, Score: 80.0}},
		},
	}

	projects := []ProjectSummary{
		projectSummaryHelper("alpha", 90.0, true, true, 0, 0, "v1.0.0", time.Now()),
	}

	store.Append(projects)

	points := store.Entries["alpha"]
	if len(points) != 1 {
		t.Fatalf("expected 1 data point after same-day dedup, got %d", len(points))
	}

	if points[0].Score != 90.0 {
		t.Errorf("expected updated score 90.0, got %.1f", points[0].Score)
	}
}

func TestPrune90Days(t *testing.T) {
	oldDate := time.Now().UTC().Add(-100 * 24 * time.Hour).Format("2006-01-02")
	recentDate := time.Now().UTC().Add(-10 * 24 * time.Hour).Format("2006-01-02")

	store := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"alpha": {
				{Date: oldDate, Score: 70.0},
				{Date: recentDate, Score: 80.0},
			},
			"beta": {
				{Date: oldDate, Score: 60.0}, // only old entry, should be removed
			},
		},
	}

	store.Prune(90 * 24 * time.Hour)

	points := store.Entries["alpha"]
	if len(points) != 1 {
		t.Fatalf("expected 1 point for alpha after prune, got %d", len(points))
	}
	if points[0].Score != 80.0 {
		t.Errorf("expected remaining score 80.0, got %.1f", points[0].Score)
	}

	if _, ok := store.Entries["beta"]; ok {
		t.Error("expected beta to be removed after prune (all entries old)")
	}
}

func TestLoadNonexistentHistory(t *testing.T) {
	store, err := LoadHistory("/tmp/nonexistent-history-file-qsdev-test.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if len(store.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(store.Entries))
	}
}

func TestLoadEmptyPath(t *testing.T) {
	store, err := LoadHistory("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	store := newHistoryStore()
	store.Entries["alpha"] = []TrendPoint{
		{Date: "2025-01-01", Score: 80.0},
		{Date: "2025-01-02", Score: 85.0},
	}
	store.Entries["beta"] = []TrendPoint{
		{Date: "2025-01-01", Score: 70.0},
	}

	if err := SaveHistory(path, store); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Entries) != 2 {
		t.Errorf("expected 2 projects, got %d", len(loaded.Entries))
	}

	alphaPoints := loaded.Entries["alpha"]
	if len(alphaPoints) != 2 {
		t.Errorf("expected 2 alpha points, got %d", len(alphaPoints))
	}
}

func TestPreviousScore(t *testing.T) {
	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")

	store := &HistoryStore{
		Entries: map[string][]TrendPoint{
			"alpha": {
				{Date: yesterday, Score: 80.0},
				{Date: today, Score: 85.0},
			},
			"beta": {
				{Date: today, Score: 70.0}, // only today's entry
			},
		},
	}

	score, ok := store.PreviousScore("alpha")
	if !ok {
		t.Fatal("expected to find previous score for alpha")
	}
	if score != 80.0 {
		t.Errorf("expected previous score 80.0, got %.1f", score)
	}

	_, ok = store.PreviousScore("beta")
	if ok {
		t.Error("expected no previous score for beta (only today)")
	}

	_, ok = store.PreviousScore("nonexistent")
	if ok {
		t.Error("expected no previous score for nonexistent project")
	}
}

func TestSaveHistoryEmptyPath(t *testing.T) {
	err := SaveHistory("", newHistoryStore())
	if err != nil {
		t.Fatalf("unexpected error saving to empty path: %v", err)
	}
}

func TestLoadHistoryCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")

	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadHistory(path)
	if err == nil {
		t.Fatal("expected error for corrupt history file")
	}
}
