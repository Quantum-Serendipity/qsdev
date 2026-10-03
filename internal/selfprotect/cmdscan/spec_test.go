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

// TestInvokedSpecs pins where the program counts as invoked and whether the
// match rests on computed words: in command position and in command lines
// other programs run, every literal or computed form; as an unquoted
// argument of another program, only a written-out subcommand; as a quoted
// argument, never; and never with a literal help or version flag.
func TestInvokedSpecs(t *testing.T) {
	t.Parallel()
	specs := []CommandSpec{
		{Path: [][]string{{"defaults"}, {"pin"}}, ReadOnly: []string{"--help", "-h", "--version"}, ValueFlags: []string{"--reason"}},
		{Path: [][]string{{"teardown"}}, ReadOnly: []string{"--dry-run", "--help", "-h", "--version"}},
	}
	const (
		none = iota
		literal
		computed
	)
	tests := []struct {
		command string
		want    int
	}{
		// The regression: an argument mention followed by a glob or a
		// variable, a quoted mention, and help and version forms.
		{"grep -rn qsdev internal/*.go", none},
		{"rg qsdev *.md", none},
		{"qsdev --version $V", none},
		{"qsdev $CMD --help", none},
		{`grep -l "qsdev teardown" docs/*`, none},
		{"cat qsdev $X", none},
		{"qsdev help defaults pin", none},
		{"qsdev -h $X", none},
		{"qsdev defaults pin --reason=--help", literal},
		{"qsdev defaults pin --reason --help", literal},
		{"qsdev defaults pin -- --help", literal},
		{"qsdev --config x defaults pin", literal},
		{"qsdev --config=x defaults x pin", none},
		// The computed forms the previous fix closed stay closed.
		{"Q=qsdev; $Q defaults pin", computed},
		{"P=pin; qsdev defaults $P", computed},
		{"qsdev defaults $(echo pin)", computed},
		{"echo pin | xargs qsdev defaults", computed},
		{`Q=qsdev; env -u CLAUDECODE script -qec "$Q defaults pin" /dev/null`, computed},
		{"echo qsdev | xargs -I X X defaults pin", computed},
		{"alias q=qsdev; q defaults pin", literal},
		// Command position past reserved words, assignments and wrappers,
		// and in command lines other programs run.
		{"qsdev defaults pin", literal},
		{"if A=1 qsdev teardown; then :; fi", literal},
		{"timeout 5 env -u X qsdev teardown", literal},
		{`bash -ec "qsdev teardown"`, literal},
		{`eval "qsdev teardown"`, literal},
		{`env -S "qsdev teardown"`, literal},
		{`env --split-string="qsdev teardown"`, literal},
		{`su -c "qsdev teardown" root`, literal},
		{`script --command="qsdev teardown"`, literal},
		{`powershell -command "qsdev teardown"`, literal},
		{`CMD /C "qsdev teardown"`, literal},
		{`sh -c "$Q defaults pin"`, computed},
		// An unquoted argument with the subcommand written out.
		{`find . -exec qsdev teardown \;`, literal},
		{"devenv shell qsdev defaults pin", literal},
		{"devenv shell qsdev defaults $P", none},
		{"echo pin | xargs devenv shell qsdev defaults", none},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			got := none
			for _, h := range InvokedSpecs(tt.command, "qsdev", specs) {
				if h.Computed && got == none {
					got = computed
				} else if !h.Computed {
					got = literal
				}
			}
			if got != tt.want {
				t.Errorf("InvokedSpecs(%q) = %d, want %d (0 none, 1 literal, 2 computed)", tt.command, got, tt.want)
			}
		})
	}
}

// TestInvokedSpecsLinear pins that judging computed words stays fast on
// adversarial text (U18-WS2's budget): unclosed and deeply nested brackets,
// and long flag runs under xargs.
func TestInvokedSpecsLinear(t *testing.T) {
	t.Parallel()
	specs := []CommandSpec{{Path: [][]string{{"defaults"}, {"pin"}}, ReadOnly: []string{"--help"}, Flags: []FlagCond{{Spellings: []string{"--force"}, Value: true}}}}
	for name, cmd := range map[string]string{
		"unclosed braces":  strings.Repeat("{", 200000),
		"nested braces":    strings.Repeat("{", 100000) + strings.Repeat("}", 100000),
		"unclosed substs":  strings.Repeat("$(", 100000),
		"xargs flag run":   "xargs " + strings.Repeat("-v ", 100000),
		"dynamic word run": strings.Repeat("$x ", 100000),
		"invocation run":   strings.Repeat("qsdev defaults pin ", 50000),
		"mention run":      "echo " + strings.Repeat("qsdev defaults ", 50000) + "pin",
		"command line run": strings.Repeat("sh -c ", 50000) + "qsdev defaults pin",
	} {
		start := time.Now()
		InvokedSpecs(cmd, "qsdev", specs)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("InvokedSpecs on %s took %v, want well under 2s", name, d)
		}
	}
}
