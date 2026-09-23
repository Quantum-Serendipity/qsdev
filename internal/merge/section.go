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
// If existing lacks markers, ErrMarkersNotFound is returned; write paths use
// SectionMarkersOrAppend, which defines that case as an append.
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

	block, err := markedSection(newGenerated)
	if err != nil {
		return nil, err
	}

	// Splice: existing before begin + new section + existing after end.
	var buf bytes.Buffer
	buf.Write(existing[:existBegin])
	buf.Write(block)
	buf.Write(existing[existEndLineEnd:])

	return buf.Bytes(), nil
}

// SectionMarkersOrAppend is SectionMarkers with a defined policy for an
// existing file that has no generated-section markers at all (e.g. a
// hand-written CLAUDE.md): the existing content is kept verbatim and the
// marked section from newGenerated is appended after it, separated by a blank
// line. Later runs then find the markers and replace only that section.
// Malformed markers in existing are still an error, since there is no safe
// way to tell generated text from user text.
func SectionMarkersOrAppend(existing, newGenerated []byte) ([]byte, error) {
	merged, err := SectionMarkers(existing, newGenerated)
	if !errors.Is(err, ErrMarkersNotFound) {
		return merged, err
	}

	block, err := markedSection(newGenerated)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.Write(existing)
	if len(existing) > 0 {
		// Separate user text from the block by a blank line.
		if !bytes.HasSuffix(existing, []byte("\n")) {
			buf.WriteByte('\n')
		}
		if !bytes.HasSuffix(buf.Bytes(), []byte("\n\n")) {
			buf.WriteByte('\n')
		}
	}
	buf.Write(block)
	return buf.Bytes(), nil
}

// RemoveSection removes the generated-section block (the begin marker line
// through the end marker line) from existing and keeps everything around it;
// the blank line SectionMarkersOrAppend put between user text and the block
// goes with it. existing is returned unchanged when it has no markers, and
// ErrMalformedMarkers when they are malformed.
func RemoveSection(existing []byte) ([]byte, error) {
	begin := indexLinePrefix(existing, []byte(BeginMarkerPrefix))
	end := indexLinePrefix(existing, []byte(EndMarker))
	switch {
	case begin < 0 && end < 0:
		return existing, nil
	case begin < 0 || end <= begin:
		return nil, ErrMalformedMarkers
	}
	endLineEnd := end + len(EndMarker)
	if endLineEnd < len(existing) && existing[endLineEnd] == '\n' {
		endLineEnd++
	}
	before, after := existing[:begin], existing[endLineEnd:]
	var buf bytes.Buffer
	if len(after) == 0 {
		if trimmed := bytes.TrimRight(before, "\n"); len(trimmed) > 0 {
			buf.Write(trimmed)
			buf.WriteByte('\n')
		}
	} else {
		buf.Write(before)
		buf.Write(after)
	}
	return buf.Bytes(), nil
}

// markedSection returns the begin..end marker block (including the end
// marker's trailing newline, if any) from generated content.
func markedSection(generated []byte) ([]byte, error) {
	begin := indexLinePrefix(generated, []byte(BeginMarkerPrefix))
	end := indexLinePrefix(generated, []byte(EndMarker))
	if begin < 0 || end < 0 {
		return nil, fmt.Errorf("section markers not found in new generated content")
	}
	if end <= begin {
		return nil, fmt.Errorf("malformed section markers in new generated content")
	}
	endLineEnd := end + len(EndMarker)
	if endLineEnd < len(generated) && generated[endLineEnd] == '\n' {
		endLineEnd++
	}
	return generated[begin:endLineEnd], nil
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
