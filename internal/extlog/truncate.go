package extlog

import "sort"

const (
	// errorContextBefore and errorContextAfter bound the window of entries kept
	// around each error-level entry.
	errorContextBefore = 5
	errorContextAfter  = 3

	// entryByteOverhead approximates the per-entry rendering overhead (timestamp,
	// level tag, separators, newline) counted against a byte budget.
	entryByteOverhead = 20
)

// Truncate reduces entries to at most maxEntries, preserving error-adjacent
// context. If no errors found, keeps head + tail. It is TruncateWithBudget
// with no byte budget.
func Truncate(entries []LogEntry, maxEntries int) []LogEntry {
	return TruncateWithBudget(entries, maxEntries, 0)
}

// TruncateWithBudget reduces entries to at most maxEntries entries whose
// rendered size (see EntrySize) totals at most maxBytes; maxBytes <= 0 means no
// byte budget.
//
// When the input contains error-level entries, error clusters (each error plus
// the entries just before and after it) are kept starting from the MOST RECENT
// error, because the last error is usually the failure that ended the run.
// Without errors, the tail (most recent entries) is preferred over the head.
// The selected entries are always returned in their original order.
func TruncateWithBudget(entries []LogEntry, maxEntries, maxBytes int) []LogEntry {
	if len(entries) <= maxEntries && (maxBytes <= 0 || totalSize(entries) <= maxBytes) {
		return entries
	}

	// Find error indices.
	var errorIdx []int
	for i, e := range entries {
		if e.Level >= LevelError {
			errorIdx = append(errorIdx, i)
		}
	}

	var priority []int
	if len(errorIdx) > 0 {
		priority = errorCentricPriority(len(entries), errorIdx)
	} else {
		priority = headTailPriority(len(entries), maxEntries)
	}
	return selectByPriority(entries, priority, maxEntries, maxBytes)
}

// EntrySize is the number of bytes an entry is assumed to occupy when
// rendered, used to enforce TruncateWithBudget's byte budget.
func EntrySize(e LogEntry) int {
	return len(e.Message) + entryByteOverhead
}

func totalSize(entries []LogEntry) int {
	n := 0
	for _, e := range entries {
		n += EntrySize(e)
	}
	return n
}

// errorCentricPriority orders entry indices so that the most recent error
// cluster comes first. Within a cluster the error itself is first, followed by
// its context ordered by distance from the error (preceding lines first).
func errorCentricPriority(n int, errorIdx []int) []int {
	priority := make([]int, 0, len(errorIdx)*(errorContextBefore+errorContextAfter+1))
	for k := len(errorIdx) - 1; k >= 0; k-- {
		idx := errorIdx[k]
		priority = append(priority, idx)
		for d := 1; d <= max(errorContextBefore, errorContextAfter); d++ {
			if d <= errorContextBefore && idx-d >= 0 {
				priority = append(priority, idx-d)
			}
			if d <= errorContextAfter && idx+d < n {
				priority = append(priority, idx+d)
			}
		}
	}
	return priority
}

// headTailPriority orders entry indices for logs without errors: the tail
// (most recent first) then a short head for context.
func headTailPriority(n, maxEntries int) []int {
	limit := min(n, maxEntries)
	head := min(20, limit/3)
	tail := limit - head

	priority := make([]int, 0, limit)
	for i := n - 1; i >= 0 && i >= n-tail && i >= head; i-- {
		priority = append(priority, i)
	}
	for i := 0; i < head && i < n; i++ {
		priority = append(priority, i)
	}
	return priority
}

// selectByPriority greedily keeps indices in priority order while both the
// entry and byte budgets allow, then returns the kept entries in original
// order. Duplicate indices (overlapping clusters) are counted once. An entry
// too large for the remaining byte budget is skipped so that smaller, more
// recent entries can still be kept.
func selectByPriority(entries []LogEntry, priority []int, maxEntries, maxBytes int) []LogEntry {
	kept := make(map[int]bool, maxEntries)
	usedBytes := 0
	for _, idx := range priority {
		if len(kept) >= maxEntries {
			break
		}
		if kept[idx] {
			continue
		}
		size := EntrySize(entries[idx])
		if maxBytes > 0 && usedBytes+size > maxBytes {
			continue
		}
		kept[idx] = true
		usedBytes += size
	}

	indices := make([]int, 0, len(kept))
	for idx := range kept {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	result := make([]LogEntry, 0, len(indices))
	for _, idx := range indices {
		result = append(result, entries[idx])
	}
	return result
}
