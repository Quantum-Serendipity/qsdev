package merge

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	ErrMarkersNotFound  = errors.New("section markers not found in existing file")
	ErrMalformedMarkers = errors.New("malformed section markers: begin without matching end")
)

const BeginMarkerPrefix = "<!-- BEGIN GENERATED SECTION"
const EndMarker = "<!-- END GENERATED SECTION -->"

// SectionMarkers replaces the content between section markers in existing
// with the marked section from newGenerated. Content before the begin
// marker and after the end marker in existing is preserved.
//
// Both existing and newGenerated must contain a matching begin/end marker pair.
// If existing lacks markers, ErrMarkersNotFound is returned.
// If markers are malformed (begin without end, or end before begin), ErrMalformedMarkers is returned.
func SectionMarkers(existing, newGenerated []byte) ([]byte, error) {
	// Find markers in existing.
	existBegin := indexLinePrefix(existing, []byte(BeginMarkerPrefix))
	existEnd := indexLinePrefix(existing, []byte(EndMarker))

	// Determine which markers are present to give the right error.
	if existBegin < 0 && existEnd < 0 {
		return nil, ErrMarkersNotFound
	}
	if existBegin < 0 && existEnd >= 0 {
		// End without begin.
		return nil, ErrMalformedMarkers
	}
	if existBegin >= 0 && existEnd < 0 {
		// Begin without end.
		return nil, ErrMalformedMarkers
	}
	if existEnd <= existBegin {
		// End before begin.
		return nil, ErrMalformedMarkers
	}

	// Find end of the end-marker line in existing (include trailing newline if present).
	existEndLineEnd := existEnd + len(EndMarker)
	if existEndLineEnd < len(existing) && existing[existEndLineEnd] == '\n' {
		existEndLineEnd++
	}

	// Find markers in newGenerated.
	newBegin := indexLinePrefix(newGenerated, []byte(BeginMarkerPrefix))
	newEnd := indexLinePrefix(newGenerated, []byte(EndMarker))

	if newBegin < 0 || newEnd < 0 {
		return nil, fmt.Errorf("section markers not found in new generated content")
	}
	if newEnd <= newBegin {
		return nil, fmt.Errorf("malformed section markers in new generated content")
	}

	// Find end of end-marker line in newGenerated (include trailing newline if present).
	newEndLineEnd := newEnd + len(EndMarker)
	if newEndLineEnd < len(newGenerated) && newGenerated[newEndLineEnd] == '\n' {
		newEndLineEnd++
	}

	// Splice: existing before begin + new section + existing after end.
	var buf bytes.Buffer
	buf.Write(existing[:existBegin])
	buf.Write(newGenerated[newBegin:newEndLineEnd])
	buf.Write(existing[existEndLineEnd:])

	return buf.Bytes(), nil
}

// indexLinePrefix returns the byte offset of the first line that starts with prefix, or -1.
func indexLinePrefix(data, prefix []byte) int {
	// Check if data starts with prefix (first line).
	if bytes.HasPrefix(data, prefix) {
		return 0
	}
	// Search for \n followed by prefix.
	search := append([]byte{'\n'}, prefix...)
	idx := bytes.Index(data, search)
	if idx < 0 {
		return -1
	}
	return idx + 1 // skip the newline to point at the start of the line
}
