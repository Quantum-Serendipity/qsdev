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
// other programs run, every literal or computed form (never with a literal
// help or version flag); as a mention (an argument, a word of a quoted
// string, an assignment's value), only a written-out subcommand, whichever
// program takes the text.
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
		mention
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
		{"cat qsdev $X", none},
		{"echo qsdev $X", none},
		{"grep qsdev $FILE", none},
		{`rg "qsdev" docs | head; node build.js`, none},
		{"qsdev help defaults pin", none},
		{"qsdev -h $X", none},
		{"qsdev defaults pin --help", none},
		{"qsdev defaults pin --reason=--help", literal},
		{"qsdev defaults pin --reason --help", literal},
		{"qsdev defaults pin -- --help", literal},
		{"qsdev --config x defaults pin", literal},
		{"qsdev --config=x defaults x pin", none},
		// The computed forms the previous fix closed stay closed.
		{"Q=qsdev; $Q defaults pin", computed},
		{"Q=qsdev; $Q $S defaults pin", computed},
		{`c=qsdev; s="defaults pin"; $c $s`, computed},
		{"c=qsdev; $c $s", computed},
		{"Q=qsdev; $Q", none},
		{"Q=qsdev; $Q --help", none},
		{"$A $B", none},
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
		// An argument, a word of a quoted string or an assignment's value
		// names the program as a mention: with the subcommand written out
		// it matches whichever program takes it, since many run text they
		// are given as code; data that only names the command matches too.
		{`find . -exec qsdev teardown \;`, mention},
		{`find . -exec "qsdev" teardown \;`, mention},
		{"devenv shell qsdev defaults pin", mention},
		{"devenv shell qsdev defaults $P", none},
		{"echo pin | xargs devenv shell qsdev defaults", none},
		{`echo "qsdev teardown" | sh`, mention},
		{`echo 'qsdev defaults pin' | bash`, mention},
		{`printf 'qsdev teardown\n' | bash`, mention},
		{`printf 'cd /x\nqsdev teardown' | sh`, literal},
		{`bash <<< "qsdev teardown"`, mention},
		{`read -r x <<< "qsdev teardown"; $x`, mention},
		{`ssh localhost "qsdev teardown"`, mention},
		{`echo "qsdev teardown" > x.sh && sh x.sh`, mention},
		{`trap "qsdev teardown" EXIT`, mention},
		{`git rebase -x "qsdev teardown" HEAD~1`, mention},
		{`flock /tmp/l -c "qsdev teardown"`, mention},
		{`parallel ::: "qsdev teardown"`, mention},
		{`parallel ::: "qsdev teardown" --help`, mention},
		{`npx -c "qsdev teardown"`, mention},
		{`npm exec -c "qsdev defaults pin"`, mention},
		{`vim -c '!qsdev teardown'`, mention},
		{`ex -c '!qsdev teardown'`, mention},
		{`less -c "!qsdev teardown" x`, mention},
		{`osascript -e 'do shell script "qsdev teardown"'`, mention},
		{`sed -n '1e qsdev teardown' x`, mention},
		{`expect -c 'spawn qsdev teardown'`, mention},
		{`gdb -batch -ex "shell qsdev teardown"`, mention},
		{`git -c alias.z='!qsdev teardown' z`, mention},
		{`x="qsdev teardown"; $x`, mention},
		{`x='qsdev defaults pin'; ${x}`, mention},
		{`set -- "qsdev teardown"; $1`, mention},
		{`PROMPT_COMMAND="qsdev teardown" bash -i`, mention},
		{`GIT_EDITOR="qsdev teardown" git commit`, mention},
		{`GIT_SSH_COMMAND="qsdev teardown" git fetch`, mention},
		{`EDITOR="qsdev teardown" crontab -e`, mention},
		{`echo "see qsdev teardown"`, mention},
		{`echo "run: qsdev teardown" > notes.md`, mention},
		{`git commit -m "qsdev teardown"`, mention},
		{`git commit -m "docs: explain qsdev teardown"`, mention},
		{`grep -l "qsdev teardown" docs/*`, mention},
		{`git log --grep="qsdev teardown" -n 5`, mention},
		{`rg -n "qsdev defaults pin" docs/ $DIR`, mention},
		// A mention needs the name to start a word and the subcommand
		// written out after it.
		{`echo "myqsdev teardown"`, none},
		{`echo "qsdev-teardown"`, none},
		{`echo "qsdev" teardown2`, none},
		{`git commit -m "qsdev $X"`, none},
		{`env -S '-u HOME qsdev teardown'`, mention},
		{`"C:\tools\qsdev.exe" teardown`, literal},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			// The most certain hit decides: literal, then mention, then
			// computed.
			rank := map[int]int{none: 0, computed: 1, mention: 2, literal: 3}
			got := none
			for _, h := range InvokedSpecs(tt.command, "qsdev", specs) {
				kind := literal
				switch {
				case h.Computed:
					kind = computed
				case h.Mention:
					kind = mention
				}
				if rank[kind] > rank[got] {
					got = kind
				}
			}
			if got != tt.want {
				t.Errorf("InvokedSpecs(%q) = %d, want %d (0 none, 1 literal, 2 computed, 3 mention)", tt.command, got, tt.want)
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
		"mention flag run": func(n int) string { return "echo " + strings.Repeat("qsdev -v ", n) },
		"assignment run":   func(n int) string { return strings.Repeat("x=qsdev ", n) + "defaults pin" },
	} {
		testutil.AssertLinearTime(t, "InvokedSpecs on "+name, 25000, func(n int) {
			InvokedSpecs(build(n), "qsdev", specs)
		})
	}
}
