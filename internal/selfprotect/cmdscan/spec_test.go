package cmdscan

import (
	"strings"
	"testing"
	"time"
)

// TestOpaqueExpansions pins which spans opaqueExpansions turns into one
// dynamic word: substitutions, braced parameters, brace expansions and the
// {} placeholder, but not a brace group.
func TestOpaqueExpansions(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"qsdev defaults $(echo pin)", "qsdev defaults $_"},
		{"$(printf qs)dev x", "$_dev x"},
		{"qsdev `echo a` b", "qsdev $_ b"},
		{"${Q} x", "$_ x"},
		{"$((1+(2))) x", "$_ x"},
		{"{qsdev,} x", "$_ x"},
		{"echo {1..3}", "echo $_"},
		{"find . -exec qsdev {} ;", "find . -exec qsdev $_ ;"},
		{"{ qsdev teardown; }", "{ qsdev teardown; }"},
		{"{x}", "{x}"},
		{"echo $(unterminated", "echo $_"},
		{"echo {", "echo {"},
	}
	for _, tt := range tests {
		if got := opaqueExpansions(tt.in); got != tt.want {
			t.Errorf("opaqueExpansions(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestInvokedSpecsLinear pins that judging computed words stays fast on
// adversarial text (U18-WS2's budget): unclosed and deeply nested brackets,
// and long flag runs under xargs.
func TestInvokedSpecsLinear(t *testing.T) {
	t.Parallel()
	specs := []CommandSpec{{Path: [][]string{{"defaults"}, {"pin"}}}}
	for name, cmd := range map[string]string{
		"unclosed braces":  strings.Repeat("{", 200000),
		"nested braces":    strings.Repeat("{", 100000) + strings.Repeat("}", 100000),
		"unclosed substs":  strings.Repeat("$(", 100000),
		"xargs flag run":   "xargs " + strings.Repeat("-v ", 100000),
		"dynamic word run": strings.Repeat("$x ", 100000),
	} {
		start := time.Now()
		InvokedSpecs(cmd, "qsdev", specs)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("InvokedSpecs on %s took %v, want well under 2s", name, d)
		}
	}
}
