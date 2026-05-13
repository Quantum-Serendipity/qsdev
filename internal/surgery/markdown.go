package surgery

import (
	"bytes"
	"fmt"
)

// MarkdownInsertSection inserts content between HTML comment markers in a
// Markdown file. Markers use the format:
//
//	<!-- gdev:<sectionID> -->
//	... content ...
//	<!-- /gdev:<sectionID> -->
//
// If the section already exists, its content is replaced.
// If it doesn't exist, the section is inserted before <!-- END GENERATED SECTION -->.
// Returns an error if neither existing markers nor the end marker are found.
func MarkdownInsertSection(existing []byte, sectionID string, content []byte) ([]byte, error) {
	openMarker := []byte(fmt.Sprintf("<!-- gdev:%s -->", sectionID))
	closeMarker := []byte(fmt.Sprintf("<!-- /gdev:%s -->", sectionID))

	openIdx := bytes.Index(existing, openMarker)
	closeIdx := bytes.Index(existing, closeMarker)

	if openIdx >= 0 && closeIdx > openIdx {
		// Replace existing section content.
		var buf bytes.Buffer
		buf.Write(existing[:openIdx])
		buf.Write(openMarker)
		buf.WriteByte('\n')
		buf.Write(bytes.TrimSpace(content))
		buf.WriteByte('\n')
		buf.Write(closeMarker)
		buf.Write(existing[closeIdx+len(closeMarker):])
		return buf.Bytes(), nil
	}

	// Insert before <!-- END GENERATED SECTION -->.
	endMarker := []byte("<!-- END GENERATED SECTION -->")
	endIdx := bytes.Index(existing, endMarker)
	if endIdx < 0 {
		return nil, fmt.Errorf("cannot insert section %q: no <!-- END GENERATED SECTION --> marker found", sectionID)
	}

	var buf bytes.Buffer
	buf.Write(existing[:endIdx])
	buf.Write(openMarker)
	buf.WriteByte('\n')
	buf.Write(bytes.TrimSpace(content))
	buf.WriteByte('\n')
	buf.Write(closeMarker)
	buf.WriteByte('\n')
	buf.Write(existing[endIdx:])
	return buf.Bytes(), nil
}

// MarkdownRemoveSection removes a tool section from a Markdown file.
// Returns unchanged content if the section is not found.
func MarkdownRemoveSection(existing []byte, sectionID string) ([]byte, error) {
	openMarker := []byte(fmt.Sprintf("<!-- gdev:%s -->", sectionID))
	closeMarker := []byte(fmt.Sprintf("<!-- /gdev:%s -->", sectionID))

	openIdx := bytes.Index(existing, openMarker)
	if openIdx < 0 {
		return existing, nil
	}

	closeIdx := bytes.Index(existing[openIdx:], closeMarker)
	if closeIdx < 0 {
		return existing, nil
	}
	closeIdx += openIdx

	// Include trailing newline if present.
	endPos := closeIdx + len(closeMarker)
	if endPos < len(existing) && existing[endPos] == '\n' {
		endPos++
	}

	// Strip leading blank line if removal creates a double blank line.
	startPos := openIdx
	if startPos > 0 && existing[startPos-1] == '\n' {
		if endPos < len(existing) && existing[endPos] == '\n' {
			endPos++
		}
	}

	var buf bytes.Buffer
	buf.Write(existing[:startPos])
	buf.Write(existing[endPos:])
	return buf.Bytes(), nil
}

// MarkdownHasSection returns true if the section markers exist in the content.
func MarkdownHasSection(content []byte, sectionID string) bool {
	marker := []byte(fmt.Sprintf("<!-- gdev:%s -->", sectionID))
	return bytes.Contains(content, marker)
}
