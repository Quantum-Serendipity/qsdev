package logging

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
)

// BenchmarkRedactString measures RedactString, which runs on every log record
// and MCP tool result, over a clean line, a line carrying one token and a
// roughly 4 KiB excerpt that mixes clean lines with every canon sample.
func BenchmarkRedactString(b *testing.B) {
	samples := secretstest.ValuePatternSamples()
	clean := "2026-10-05T12:00:00Z INFO building module internal/logging with 42 files in 1.3s"
	oneToken := "error: authentication failed with key " + samples["aws"][0] + " for request 17"

	var mixed strings.Builder
	for _, vp := range secrets.ValuePatterns {
		mixed.WriteString(clean)
		mixed.WriteByte('\n')
		mixed.WriteString("loaded " + vp.Name + " credential " + samples[vp.Name][0] + " from cache\n")
	}
	for mixed.Len() < 4096 {
		mixed.WriteString(clean)
		mixed.WriteByte('\n')
	}

	cases := []struct {
		name  string
		input string
	}{
		{name: "clean", input: clean},
		{name: "one-token", input: oneToken},
		{name: "mixed-4KiB", input: mixed.String()[:4096]},
	}
	r := NewRedactor()
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.SetBytes(int64(len(c.input)))
			b.ReportAllocs()
			for b.Loop() {
				_ = r.RedactString(c.input)
			}
		})
	}
}
