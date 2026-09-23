package extlog

import (
	"strings"
	"testing"
)

func TestTruncateKeepsMostRecentError(t *testing.T) {
	t.Parallel()

	// 100 entries with an error every 10th index plus the final fatal error:
	// more clusters than fit, so the most recent ones must win.
	levels := make([]LogLevel, 100)
	for i := range levels {
		levels[i] = LevelInfo
		if i%10 == 0 {
			levels[i] = LevelError
		}
	}
	levels[99] = LevelFatal

	entries := makeEntries(levels...)
	for i := range entries {
		entries[i].LineNumber = i
	}

	got := Truncate(entries, 50)

	if len(got) > 50 {
		t.Fatalf("Truncate exceeded limit: got %d, want <= 50", len(got))
	}
	if got[len(got)-1].LineNumber != 99 {
		t.Errorf("last kept entry = %d, want the final fatal error at 99", got[len(got)-1].LineNumber)
	}
	for i := 1; i < len(got); i++ {
		if got[i].LineNumber <= got[i-1].LineNumber {
			t.Fatalf("entries not in chronological order at %d: %d after %d", i, got[i].LineNumber, got[i-1].LineNumber)
		}
	}
}

func TestTruncateWithBudget(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 500)
	entrySize := len(long) + entryByteOverhead
	tests := []struct {
		name       string
		levels     []LogLevel
		maxEntries int
		maxBytes   int
		wantLast   int
	}{
		{
			name: "byte budget keeps final error",
			levels: []LogLevel{
				LevelError, LevelInfo, LevelInfo, LevelInfo, LevelInfo,
				LevelInfo, LevelInfo, LevelInfo, LevelInfo, LevelError,
			},
			maxEntries: 50,
			maxBytes:   3 * entrySize,
			wantLast:   9,
		},
		{
			name:       "byte budget without errors keeps tail",
			levels:     []LogLevel{LevelInfo, LevelInfo, LevelInfo, LevelInfo, LevelInfo, LevelInfo},
			maxEntries: 50,
			maxBytes:   2 * entrySize,
			wantLast:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entries := makeEntries(tt.levels...)
			for i := range entries {
				entries[i].LineNumber = i
				entries[i].Message = long
			}
			got := TruncateWithBudget(entries, tt.maxEntries, tt.maxBytes)
			if len(got) == 0 {
				t.Fatal("TruncateWithBudget returned nothing")
			}
			size := 0
			for _, e := range got {
				size += EntrySize(e)
			}
			if size > tt.maxBytes {
				t.Errorf("kept %d bytes, budget %d", size, tt.maxBytes)
			}
			if last := got[len(got)-1].LineNumber; last != tt.wantLast {
				t.Errorf("last kept entry = %d, want %d", last, tt.wantLast)
			}
		})
	}
}
