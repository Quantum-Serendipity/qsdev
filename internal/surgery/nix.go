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
// If the section already exists, its content is replaced. If it doesn't
// exist, the section is appended before the last closing '}' that is real
// Nix code (braces inside comments and string literals are ignored). An open
// marker with no close marker after it is an error rather than a duplicate.
func NixInsertSection(existing []byte, sectionID string, content []byte) ([]byte, error) {
	openMarker := []byte(fmt.Sprintf("# --- %s ---", sectionID))
	closeMarker := []byte(fmt.Sprintf("# --- end %s ---", sectionID))

	if openIdx := bytes.Index(existing, openMarker); openIdx >= 0 {
		// Search for the close marker only after the open marker, so a stray
		// earlier close marker cannot divert us into inserting a duplicate.
		rel := bytes.Index(existing[openIdx:], closeMarker)
		if rel < 0 {
			return nil, fmt.Errorf("cannot update section %q: open marker has no matching close marker", sectionID)
		}
		closeIdx := openIdx + rel

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

	// Find the last closing brace of actual code to insert before.
	lastBrace := lastNixCodeBrace(existing)
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

// lastNixCodeBrace returns the offset of the last '}' in Nix source that is
// not inside a comment ('#' or '/* */') or a string literal (double-quoted
// or indented, i.e. delimited by two single quotes), or -1 if there is none.
func lastNixCodeBrace(src []byte) int {
	last := -1
	for i := 0; i < len(src); i++ {
		switch {
		case src[i] == '#':
			i = skipTo(src, i, "\n")
		case src[i] == '/' && i+1 < len(src) && src[i+1] == '*':
			i = skipTo(src, i+2, "*/") + 1
		case src[i] == '"':
			i = skipDoubleQuoted(src, i+1)
		case src[i] == '\'' && i+1 < len(src) && src[i+1] == '\'':
			i = skipIndented(src, i+2)
		case src[i] == '}':
			last = i
		}
	}
	return last
}

// skipTo returns the offset of the first occurrence of end at or after from
// (or len(src)-1 when it never occurs, which ends the scan).
func skipTo(src []byte, from int, end string) int {
	if from >= len(src) {
		return len(src) - 1
	}
	idx := bytes.Index(src[from:], []byte(end))
	if idx < 0 {
		return len(src) - 1
	}
	return from + idx
}

// skipDoubleQuoted returns the offset of the closing '"' of a string whose
// body starts at from, honouring backslash escapes.
func skipDoubleQuoted(src []byte, from int) int {
	i := from
	for i < len(src) && src[i] != '"' {
		if src[i] == '\\' {
			i++
		}
		i++
	}
	return i
}

// skipIndented returns the offset of the second quote of the two-single-quote
// terminator of an indented string whose body starts at from. Two single
// quotes followed by a third quote, '$' or '\' are escapes inside the body,
// not terminators.
func skipIndented(src []byte, from int) int {
	i := from
	for i+1 < len(src) {
		if src[i] == '\'' && src[i+1] == '\'' {
			if i+2 < len(src) && (src[i+2] == '\'' || src[i+2] == '$' || src[i+2] == '\\') {
				i += 3
				continue
			}
			return i + 1
		}
		i++
	}
	return len(src) - 1
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
