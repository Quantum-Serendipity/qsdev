package logging

import (
	"bufio"
	"bytes"
	"io"
)

// MaxLogLineBytes is the longest log line NewLineScanner yields intact. Longer
// lines are truncated to this length and suffixed with TruncatedLineSuffix.
const MaxLogLineBytes = 1 << 20

// TruncatedLineSuffix marks a line NewLineScanner cut short.
const TruncatedLineSuffix = " …[line truncated]"

// NewLineScanner returns a line scanner for log files that never fails on a
// long line. bufio.Scanner's default 64 KiB token limit makes Scan stop with
// bufio.ErrTooLong at the first longer line (common in nix build logs and in
// large debug records), silently dropping everything after it. Here a line
// longer than MaxLogLineBytes is truncated instead, the rest of it is
// discarded, and scanning continues with the next line.
func NewLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLogLineBytes)

	discarding := false
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if discarding {
			// Skip the remainder of an over-long line, through its newline.
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				return len(data), nil, nil
			}
			discarding = false
			return i + 1, nil, nil
		}
		advance, token, err := bufio.ScanLines(data, atEOF)
		if advance == 0 && token == nil && err == nil && len(data) >= MaxLogLineBytes {
			// The buffer is full and holds no newline: emit the head of the
			// line and drop the rest of it.
			discarding = true
			truncated := make([]byte, 0, MaxLogLineBytes+len(TruncatedLineSuffix))
			truncated = append(truncated, data[:MaxLogLineBytes]...)
			return len(data), append(truncated, TruncatedLineSuffix...), nil
		}
		return advance, token, err
	})
	return sc
}
