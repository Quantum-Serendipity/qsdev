package extlog

import (
	"testing"
)

func makeEntries(levels ...LogLevel) []LogEntry {
	entries := make([]LogEntry, len(levels))
	for i, l := range levels {
		entries[i] = LogEntry{
			Level:   l,
			Source:  "test",
			Message: "msg",
		}
	}
	return entries
}

func TestTruncateNilSlice(t *testing.T) {
	t.Parallel()
	got := Truncate(nil, 10)
	if got != nil {
		t.Errorf("Truncate(nil, 10) = %v, want nil", got)
	}
}

func TestTruncateEmptySlice(t *testing.T) {
	t.Parallel()
	got := Truncate([]LogEntry{}, 10)
	if len(got) != 0 {
		t.Errorf("Truncate([], 10) length = %d, want 0", len(got))
	}
}

func TestTruncateBelowLimit(t *testing.T) {
	t.Parallel()

	entries := makeEntries(LevelInfo, LevelInfo, LevelInfo)
	got := Truncate(entries, 10)
	if len(got) != 3 {
		t.Errorf("Truncate with %d entries and limit 10: got %d, want 3", len(entries), len(got))
	}
}

func TestTruncateExactlyAtLimit(t *testing.T) {
	t.Parallel()

	entries := makeEntries(LevelInfo, LevelInfo, LevelInfo, LevelInfo, LevelInfo)
	got := Truncate(entries, 5)
	if len(got) != 5 {
		t.Errorf("Truncate at exact limit: got %d, want 5", len(got))
	}
}

func TestTruncateHeadTailNoErrors(t *testing.T) {
	t.Parallel()

	// 100 entries, no errors, max 30.
	levels := make([]LogLevel, 100)
	for i := range levels {
		levels[i] = LevelInfo
	}
	entries := makeEntries(levels...)

	// Tag entries with index for tracking.
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 30)

	if len(got) > 30 {
		t.Errorf("Truncate exceeded limit: got %d, want <= 30", len(got))
	}

	// Head portion: first entries should be from the beginning.
	if got[0].LineNumber != 0 {
		t.Errorf("first entry LineNumber = %d, want 0 (head)", got[0].LineNumber)
	}

	// Tail portion: last entries should be from the end.
	lastGot := got[len(got)-1]
	if lastGot.LineNumber != 99 {
		t.Errorf("last entry LineNumber = %d, want 99 (tail)", lastGot.LineNumber)
	}
}

func TestTruncateErrorCentric(t *testing.T) {
	t.Parallel()

	// 50 entries: one error at index 25.
	levels := make([]LogLevel, 50)
	for i := range levels {
		levels[i] = LevelInfo
	}
	levels[25] = LevelError

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 15)

	if len(got) > 15 {
		t.Errorf("Truncate exceeded limit: got %d, want <= 15", len(got))
	}

	// Error entry must be present.
	foundError := false
	for _, e := range got {
		if e.Level == LevelError {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("error entry not found in truncated output")
	}

	// Context entries around the error should be present.
	// Error is at index 25, context before = 5, context after = 3.
	// So indices 20-28 should be included.
	gotIndices := make(map[int]bool)
	for _, e := range got {
		gotIndices[e.LineNumber] = true
	}

	for i := 20; i <= 28; i++ {
		if !gotIndices[i] {
			t.Errorf("expected context entry at index %d to be present", i)
		}
	}
}

func TestTruncateMultipleErrors(t *testing.T) {
	t.Parallel()

	// 100 entries with errors at indices 10 and 90.
	levels := make([]LogLevel, 100)
	for i := range levels {
		levels[i] = LevelInfo
	}
	levels[10] = LevelError
	levels[90] = LevelFatal

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 30)

	if len(got) > 30 {
		t.Errorf("Truncate exceeded limit: got %d, want <= 30", len(got))
	}

	// Both error entries should be present.
	var foundError, foundFatal bool
	for _, e := range got {
		if e.Level == LevelError {
			foundError = true
		}
		if e.Level == LevelFatal {
			foundFatal = true
		}
	}
	if !foundError {
		t.Error("LevelError entry not found in truncated output")
	}
	if !foundFatal {
		t.Error("LevelFatal entry not found in truncated output")
	}
}

func TestTruncateErrorAtStart(t *testing.T) {
	t.Parallel()

	// Error at index 0 — context before would go negative.
	levels := make([]LogLevel, 30)
	for i := range levels {
		levels[i] = LevelInfo
	}
	levels[0] = LevelError

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 10)

	if len(got) == 0 {
		t.Fatal("Truncate returned empty result")
	}
	if got[0].LineNumber != 0 {
		t.Errorf("first entry LineNumber = %d, want 0", got[0].LineNumber)
	}
	if got[0].Level != LevelError {
		t.Errorf("first entry Level = %v, want LevelError", got[0].Level)
	}
}

func TestTruncateErrorAtEnd(t *testing.T) {
	t.Parallel()

	// Error at last index — context after would exceed bounds.
	levels := make([]LogLevel, 30)
	for i := range levels {
		levels[i] = LevelInfo
	}
	levels[29] = LevelError

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 10)

	foundError := false
	for _, e := range got {
		if e.Level == LevelError {
			foundError = true
		}
	}
	if !foundError {
		t.Error("error at end not found in truncated output")
	}
}

func TestTruncateFatalLevel(t *testing.T) {
	t.Parallel()

	// FATAL entries should also be treated as error-centric.
	levels := make([]LogLevel, 50)
	for i := range levels {
		levels[i] = LevelDebug
	}
	levels[30] = LevelFatal

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 10)

	foundFatal := false
	for _, e := range got {
		if e.Level == LevelFatal {
			foundFatal = true
		}
	}
	if !foundFatal {
		t.Error("fatal entry not found in truncated output")
	}
}

func TestTruncateSmallMax(t *testing.T) {
	t.Parallel()

	entries := makeEntries(LevelInfo, LevelInfo, LevelInfo, LevelInfo, LevelInfo)
	got := Truncate(entries, 2)

	if len(got) > 2 {
		t.Errorf("Truncate(5 entries, max=2) = %d entries, want <= 2", len(got))
	}
}

func TestTruncateMaxOne(t *testing.T) {
	t.Parallel()

	entries := makeEntries(LevelInfo, LevelError, LevelInfo)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 1)
	if len(got) != 1 {
		t.Errorf("Truncate(3 entries, max=1) = %d entries, want 1", len(got))
	}
}

func TestTruncatePreservesEntryContent(t *testing.T) {
	t.Parallel()

	entries := []LogEntry{
		{Level: LevelInfo, Source: "npm", Message: "installing packages"},
		{Level: LevelWarn, Source: "npm", Message: "deprecated package"},
		{Level: LevelError, Source: "npm", Message: "ENOENT file not found"},
		{Level: LevelInfo, Source: "npm", Message: "done"},
	}

	got := Truncate(entries, 10)

	// All entries should be returned since count < max.
	if len(got) != 4 {
		t.Fatalf("Truncate returned %d entries, want 4", len(got))
	}
	if got[2].Message != "ENOENT file not found" {
		t.Errorf("entry message = %q, want %q", got[2].Message, "ENOENT file not found")
	}
	if got[2].Source != "npm" {
		t.Errorf("entry source = %q, want %q", got[2].Source, "npm")
	}
}
