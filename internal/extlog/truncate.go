package extlog

// Truncate reduces entries to at most maxEntries, preserving error-adjacent
// context. If no errors found, keeps head + tail.
func Truncate(entries []LogEntry, maxEntries int) []LogEntry {
	if len(entries) <= maxEntries {
		return entries
	}

	// Find error indices.
	var errorIdx []int
	for i, e := range entries {
		if e.Level >= LevelError {
			errorIdx = append(errorIdx, i)
		}
	}

	if len(errorIdx) > 0 {
		return truncateErrorCentric(entries, errorIdx, maxEntries)
	}
	return truncateHeadTail(entries, maxEntries)
}

func truncateErrorCentric(entries []LogEntry, errorIdx []int, max int) []LogEntry {
	const contextBefore = 5
	const contextAfter = 3

	keep := make(map[int]bool)
	for _, idx := range errorIdx {
		start := idx - contextBefore
		if start < 0 {
			start = 0
		}
		end := idx + contextAfter + 1
		if end > len(entries) {
			end = len(entries)
		}
		for i := start; i < end; i++ {
			keep[i] = true
		}
	}

	var result []LogEntry
	for i, e := range entries {
		if keep[i] {
			result = append(result, e)
			if len(result) >= max {
				break
			}
		}
	}
	return result
}

func truncateHeadTail(entries []LogEntry, max int) []LogEntry {
	head := 20
	if head > max/3 {
		head = max / 3
	}
	tail := max - head

	result := make([]LogEntry, 0, max)
	result = append(result, entries[:head]...)
	if tail > 0 && len(entries) > head {
		start := len(entries) - tail
		if start < head {
			start = head
		}
		result = append(result, entries[start:]...)
	}
	return result
}
