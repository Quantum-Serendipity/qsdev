package cmdscan

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
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
		// A quoted string of several words is text unless the line runs it
		// as code; a word quoted on its own is the argument it spells.
		{`echo "see qsdev teardown"`, none},
		{`echo "run: qsdev teardown" > notes.md`, none},
		{`git commit -m "qsdev teardown"`, none},
		{`git commit -m "docs: explain qsdev teardown"`, none},
		{`git commit -m "don't run qsdev teardown"`, none},
		{`echo "docs mention qsdev defaults pin here"`, none},
		{`grep "qsdev defaults pin" README.md`, none},
		{`git log --grep="qsdev teardown" -n 5`, none},
		{`rg -n "qsdev defaults pin" docs/ $DIR`, none},
		{`echo "qsdev teardown" > notes.md; cat notes.md`, none},
		{`echo "qsdev teardown" | sh`, literal},
		{`echo 'qsdev defaults pin' | bash`, literal},
		{`printf 'qsdev teardown\n' | bash`, literal},
		{`printf 'cd /x\nqsdev teardown' | sh`, literal},
		{`bash <<< "qsdev teardown"`, literal},
		{`sh <<< 'qsdev defaults pin'`, literal},
		{`ssh localhost "qsdev teardown"`, literal},
		{`watch -n1 "qsdev teardown"`, literal},
		{`tmux new -d "qsdev teardown"`, literal},
		{`echo "qsdev teardown" | python3`, literal},
		{`echo "qsdev teardown" | xargs -0 sh -c`, literal},
		{`echo "qsdev teardown" > x.sh && sh x.sh`, literal},
		{`echo "qsdev teardown" > x.sh; chmod +x x.sh; ./x.sh`, literal},
		{`echo "qsdev teardown" | tee /tmp/x.sh; /tmp/x.sh`, literal},
		{`env -S '-u HOME qsdev teardown'`, literal},
		{`find . -exec "qsdev" teardown \;`, literal},
		{`"C:\tools\qsdev.exe" teardown`, literal},
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

// TestInvokedSpecsLinear pins that judging computed words stays linear on
// adversarial text (U18-WS2's budget): unclosed and deeply nested brackets,
// and long flag runs under xargs.
func TestInvokedSpecsLinear(t *testing.T) {
	t.Parallel()
	specs := []CommandSpec{{Path: [][]string{{"defaults"}, {"pin"}}, ReadOnly: []string{"--help"}, Flags: []FlagCond{{Spellings: []string{"--force"}, Value: true}}}}
	for name, build := range map[string]func(n int) string{
		"unclosed braces":  func(n int) string { return strings.Repeat("{", 4*n) },
		"nested braces":    func(n int) string { return strings.Repeat("{", 2*n) + strings.Repeat("}", 2*n) },
		"unclosed substs":  func(n int) string { return strings.Repeat("$(", 2*n) },
		"xargs flag run":   func(n int) string { return "xargs " + strings.Repeat("-v ", 2*n) },
		"dynamic word run": func(n int) string { return strings.Repeat("$x ", 2*n) },
		"invocation run":   func(n int) string { return strings.Repeat("qsdev defaults pin ", n) },
		"mention run":      func(n int) string { return "echo " + strings.Repeat("qsdev defaults ", n) + "pin" },
		"command line run": func(n int) string { return strings.Repeat("sh -c ", n) + "qsdev defaults pin" },
		"quoted text run":  func(n int) string { return `echo "` + strings.Repeat("qsdev defaults ", n) + `pin" | sh` },
		"script path run":  func(n int) string { return strings.Repeat("./x.sh ", n) + "x.sh" },
	} {
		testutil.AssertLinearTime(t, "InvokedSpecs on "+name, 25000, func(n int) {
			InvokedSpecs(build(n), "qsdev", specs)
		})
	}
}
