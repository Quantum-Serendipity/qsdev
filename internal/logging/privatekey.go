package logging

// PrivateKeyLineFilter carries private-key block state across the lines of a
// line-oriented stream (external tool logs, capture files). RedactString
// redacts a whole BEGIN…END block when it sees it in one string, but a scrubber
// fed one line at a time only ever sees the BEGIN line; without this state the
// base64 body lines and the END line would pass through untouched.
//
// Use one filter per stream: pass every line through Filter before scrubbing
// it with RedactString. The zero value is ready to use.
type PrivateKeyLineFilter struct {
	inBlock bool
}

// Filter returns line with any key material belonging to a block opened on an
// earlier line replaced by the redaction marker. A BEGIN marker left open at
// the end of line puts the filter into the in-block state (RedactString redacts
// that line from the marker onward), so subsequent lines are suppressed until
// the matching END marker.
func (f *PrivateKeyLineFilter) Filter(line string) string {
	prefix := ""
	if f.inBlock {
		loc := privateKeyEndRe.FindStringIndex(line)
		if loc == nil {
			return redacted
		}
		f.inBlock = false
		prefix, line = redacted, line[loc[1]:]
	}
	if begins := privateKeyBeginRe.FindAllStringIndex(line, -1); len(begins) > 0 {
		lastBegin := begins[len(begins)-1]
		f.inBlock = !privateKeyEndRe.MatchString(line[lastBegin[1]:])
	}
	return prefix + line
}
