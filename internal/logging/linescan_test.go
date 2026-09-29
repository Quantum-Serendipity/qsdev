package logging

import (
	"strings"
	"testing"
)

func TestNewLineScanner(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", MaxLogLineBytes+10)
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain lines", "one\ntwo\n", []string{"one", "two"}},
		{"crlf", "one\r\ntwo", []string{"one", "two"}},
		{"line at the limit is intact", strings.Repeat("b", MaxLogLineBytes-1) + "\nnext", []string{strings.Repeat("b", MaxLogLineBytes-1), "next"}},
		{"over-long line truncated, scan continues", "head\n" + long + "\ntail\n", []string{"head", long[:MaxLogLineBytes] + TruncatedLineSuffix, "tail"}},
		{"over-long final line without newline", "head\n" + long, []string{"head", long[:MaxLogLineBytes] + TruncatedLineSuffix}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sc := NewLineScanner(strings.NewReader(tt.input))
			var got []string
			for sc.Scan() {
				got = append(got, sc.Text())
			}
			if err := sc.Err(); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %d bytes %.20q…, want %d bytes %.20q…", i, len(got[i]), got[i], len(tt.want[i]), tt.want[i])
				}
			}
		})
	}
}
