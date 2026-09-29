package merge

import (
	"bytes"
)

// SectionBeginLine returns the begin-marker line (without trailing newline)
// that opens the generated section identified by tag.
func SectionBeginLine(tag string) string {
	return BeginMarkerPrefix + " — " + tag + " -->"
}

// ReplaceTaggedSection replaces the generated section identified by tag in
// existing with block, which should be a complete marked section (begin line,
// content, end marker). Unlike SectionMarkers, which always targets the first
// marker pair, only the section whose begin line matches tag exactly is
// replaced, so several tagged sections can coexist in one document.
//
// It returns ErrMarkersNotFound when existing has no section for tag, and
// ErrMalformedMarkers when the tag's begin line has no end marker after it.
func ReplaceTaggedSection(existing []byte, tag string, block []byte) ([]byte, error) {
	begin := indexLine(existing, []byte(SectionBeginLine(tag)))
	if begin < 0 {
		return nil, ErrMarkersNotFound
	}

	afterBegin := lineEnd(existing, begin)
	relEnd := indexLinePrefix(existing[afterBegin:], []byte(EndMarker))
	if relEnd < 0 {
		return nil, ErrMalformedMarkers
	}
	end := lineEnd(existing, afterBegin+relEnd)

	var buf bytes.Buffer
	buf.Grow(len(existing) - (end - begin) + len(block))
	buf.Write(existing[:begin])
	buf.Write(block)
	buf.Write(existing[end:])
	return buf.Bytes(), nil
}

// indexLine returns the byte offset of the first line exactly equal to line
// (ignoring a trailing carriage return), or -1.
func indexLine(data, line []byte) int {
	for off := 0; off < len(data); {
		next := lineEnd(data, off)
		content := bytes.TrimSuffix(bytes.TrimSuffix(data[off:next], []byte{'\n'}), []byte{'\r'})
		if bytes.Equal(content, line) {
			return off
		}
		off = next
	}
	return -1
}

// lineEnd returns the offset just past the newline ending the line that
// starts at off, or len(data) for a final unterminated line.
func lineEnd(data []byte, off int) int {
	if i := bytes.IndexByte(data[off:], '\n'); i >= 0 {
		return off + i + 1
	}
	return len(data)
}
