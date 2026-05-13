package surgery

import (
	"bytes"
	"fmt"
)

// NixInsertSection inserts content between section markers in a Nix file.
// Markers use the format:
//
//	# --- <sectionID> ---
//	... content ...
//	# --- end <sectionID> ---
//
// If the section already exists, its content is replaced.
// If it doesn't exist, the section is appended before the last closing '}'.
func NixInsertSection(existing []byte, sectionID string, content []byte) ([]byte, error) {
	openMarker := []byte(fmt.Sprintf("# --- %s ---", sectionID))
	closeMarker := []byte(fmt.Sprintf("# --- end %s ---", sectionID))

	openIdx := bytes.Index(existing, openMarker)
	closeIdx := bytes.Index(existing, closeMarker)

	if openIdx >= 0 && closeIdx > openIdx {
		// Replace existing section content.
		var buf bytes.Buffer
		buf.Write(existing[:openIdx])
		buf.Write(openMarker)
		buf.WriteByte('\n')
		buf.Write(bytes.TrimRight(content, "\n"))
		buf.WriteByte('\n')
		buf.Write(closeMarker)
		endPos := closeIdx + len(closeMarker)
		buf.Write(existing[endPos:])
		return buf.Bytes(), nil
	}

	// Find the last closing brace to insert before.
	lastBrace := bytes.LastIndex(existing, []byte("}"))
	if lastBrace < 0 {
		return nil, fmt.Errorf("cannot insert section %q: no closing '}' found in Nix file", sectionID)
	}

	// Insert section with markers before the closing brace.
	var buf bytes.Buffer
	buf.Write(existing[:lastBrace])
	buf.WriteByte('\n')
	buf.Write([]byte("  "))
	buf.Write(openMarker)
	buf.WriteByte('\n')
	buf.Write(bytes.TrimRight(content, "\n"))
	buf.WriteByte('\n')
	buf.Write([]byte("  "))
	buf.Write(closeMarker)
	buf.WriteByte('\n')
	buf.Write(existing[lastBrace:])
	return buf.Bytes(), nil
}

// NixRemoveSection removes a section between markers from a Nix file.
// Returns unchanged content if the section is not found.
func NixRemoveSection(existing []byte, sectionID string) ([]byte, error) {
	openMarker := []byte(fmt.Sprintf("# --- %s ---", sectionID))
	closeMarker := []byte(fmt.Sprintf("# --- end %s ---", sectionID))

	openIdx := bytes.Index(existing, openMarker)
	if openIdx < 0 {
		return existing, nil
	}

	closeIdx := bytes.Index(existing[openIdx:], closeMarker)
	if closeIdx < 0 {
		return existing, nil
	}
	closeIdx += openIdx

	endPos := closeIdx + len(closeMarker)
	if endPos < len(existing) && existing[endPos] == '\n' {
		endPos++
	}

	// Find the start of the line containing the open marker.
	startPos := openIdx
	for startPos > 0 && existing[startPos-1] != '\n' {
		startPos--
	}

	// Remove any trailing blank line from the removal.
	if endPos < len(existing) && existing[endPos] == '\n' {
		endPos++
	}

	var buf bytes.Buffer
	buf.Write(existing[:startPos])
	buf.Write(existing[endPos:])
	return buf.Bytes(), nil
}

// NixHasSection returns true if the section markers exist in the content.
func NixHasSection(content []byte, sectionID string) bool {
	marker := []byte(fmt.Sprintf("# --- %s ---", sectionID))
	return bytes.Contains(content, marker)
}
