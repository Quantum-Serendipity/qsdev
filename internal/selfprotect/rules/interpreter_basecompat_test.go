package rules

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// scannedLine parses a Bash line and returns its scanned commands.
func scannedLine(t *testing.T, line string) []scannedCommand {
	t.Helper()
	ctx := EvalContext{ToolName: "Bash", Command: line}
	scs, err := ctx.scannedCommands()
	if err != nil {
		t.Fatalf("scannedCommands(%q): %v", line, err)
	}
	return scs
}

// TestInterpreterCallSites_StillWorkAsOnBase pins what each call site of the
// shell and interpreter taxonomy (cmdscan.IsInterpreter, IsScriptShell,
// ReadsProgramFromStdin) gives for the spellings base's own name tables
// listed, so moving the taxonomy into cmdscan changes no verdict for them.
func TestInterpreterCallSites_StillWorkAsOnBase(t *testing.T) {
	t.Parallel()
	t.Run("mutationTargets", testMutationTargetsAsOnBase)
	t.Run("commandPrograms", testCommandProgramsAsOnBase)
	t.Run("guardedfiles", testGuardedFilesInterpreterAsOnBase)
	t.Run("SP-009", testSP009InterpreterAsOnBase)
	t.Run("readsProgramFromStdin", testReadsProgramFromStdinAsOnBase)
}

func testMutationTargetsAsOnBase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want []string
	}{
		// An interpreter's script operand is run, not written.
		{"python3 .claude/hooks/x.py", nil},
		{"bash .claude/hooks/test.sh", nil},
		{"node .claude/hooks/x.js", nil},
		{"perl .claude/hooks/x.pl", nil},
		{"ruby .claude/hooks/x.rb", nil},
		{"python .claude/hooks/x.py", nil},
		// The intended change, name folding: an alias counts as the
		// interpreter it names.
		{"nodejs .claude/hooks/x.js", nil},
		{"pypy3 .claude/hooks/x.py", nil},
		{"nodejs x.js .claude/settings.json", []string{".claude/settings.json"}},
		// Its further operands stay opaque.
		{"python3 x.py .claude/settings.json", []string{".claude/settings.json"}},
		{"sh x.sh .claude/settings.json", []string{".claude/settings.json"}},
		// An inline program is opaque in full.
		{"python3 -c 'x' .claude/settings.json", []string{"-c", "x", ".claude/settings.json"}},
		{"sh -c 'rm .claude/settings.json'", []string{"-c", "rm .claude/settings.json"}},
		// Any other program may write every operand.
		{"rm .claude/settings.json", []string{".claude/settings.json"}},
		{"mytool .claude/hooks/x.py", []string{".claude/hooks/x.py"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			scs := scannedLine(t, tt.line)
			if got := mutationTargets(scs[0]); !slices.Equal(got, tt.want) {
				t.Errorf("mutationTargets(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func testCommandProgramsAsOnBase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want []string
	}{
		{"python3 hook.py", []string{"python3", "hook.py"}},
		{"/usr/bin/python3 hook.py", []string{"/usr/bin/python3", "hook.py"}},
		{"bash hook.sh", []string{"bash", "hook.sh"}},
		{"node hook.js --flag", []string{"node", "hook.js"}},
		{"bash -e hook.sh", []string{"bash"}},
		{"python3", []string{"python3"}},
		{"jq .", []string{"jq"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			cmds, err := cmdscan.Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := commandPrograms(cmds[0]); !slices.Equal(got, tt.want) {
				t.Errorf("commandPrograms(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func testGuardedFilesInterpreterAsOnBase(t *testing.T) {
	t.Parallel()
	names := []string{".npmrc"}
	tests := []struct {
		line string
		want bool
	}{
		// An inline program names the file inside its code word.
		{`python3 -c "open('.npmrc','w')"`, true},
		{`node -e "require('fs').writeFileSync('.npmrc','')"`, true},
		{`perl -e 'unlink ".npmrc"'`, true},
		{`ruby -e 'File.delete(".npmrc")'`, true},
		{`sh -c 'echo x >> .npmrc'`, true},
		{`bash -c 'rm .npmrc'`, true},
		// Reading the file is not a rewrite.
		{`cat .npmrc`, false},
		{`grep registry .npmrc`, false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.line}
			if _, got := ShellRewritesFile(&ctx, names); got != tt.want {
				t.Errorf("ShellRewritesFile(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func testSP009InterpreterAsOnBase(t *testing.T) {
	t.Parallel()
	// opaqueText: a here-string is data for a program given a script
	// operand, and the program of one reading stdin.
	assertSP009(t, Allow, []string{
		`python3 script.py <<< 'pkill qsdev'`,
		`bash script.sh <<< 'pkill qsdev'`,
		`node app.js <<< 'pkill qsdev'`,
	})
	assertSP009(t, Deny, []string{
		`python3 <<< 'pkill qsdev'`,
		`bash <<< 'pkill qsdev'`,
		`perl <<< 'system("pkill qsdev")'`,
		`ruby -w <<< 'system("pkill qsdev")'`,
		"node <<'EOF'\nrequire('child_process').execSync('pkill qsdev')\nEOF",
		// The kill code piped to a program reading it (list's fed stage).
		`echo 'pkill qsdev' | python3`,
		`echo 'pkill qsdev' | bash -s`,
		// Under xargs the input is the program's argv, which may carry its
		// code (`sh -c 'pkill qsdev'`).
		`echo 'pkill qsdev' | xargs sh`,
		`echo '-c "pkill qsdev"' | xargs bash`,
		`xargs sh <<< '-c "pkill qsdev"'`,
		`xargs -n1 python3 <<< '-c "import os; os.system(\"pkill qsdev\")"'`,
		// runsCode: an interpreter given a computed word may run text the
		// line holds elsewhere.
		`X='pkill qsdev'; python3 "$X"`,
		`X='pkill qsdev'; sh "$X"`,
	})
}

func testReadsProgramFromStdinAsOnBase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want bool
	}{
		{"python3 <<< 'x'", true},
		{"bash < script.sh", true},
		{"node <<'EOF'\nx\nEOF", true},
		{"python3 -u <<< 'x'", true},
		{"python3 script.py <<< 'x'", false},
		{"sh -c 'x' < f", false},
		{"python3", false}, // no input redirect
		{"cat <<< 'x'", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			scs := scannedLine(t, tt.line)
			if got := readsProgramFromStdin(scs[0]); got != tt.want {
				t.Errorf("readsProgramFromStdin(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// TestInterpreterCallSites_FoldNames pins the one intended change of
// moving the taxonomy into cmdscan: an interpreter named by path, in
// upper case or with a version suffix counts as the interpreter.
func TestInterpreterCallSites_FoldNames(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"/usr/bin/python3 .claude/hooks/x.py",
		"PYTHON3 .claude/hooks/x.py",
		"python3.12 .claude/hooks/x.py",
	} {
		t.Run(line, func(t *testing.T) {
			t.Parallel()
			if got := mutationTargets(scannedLine(t, line)[0]); len(got) != 0 {
				t.Errorf("mutationTargets(%q) = %q, want none", line, got)
			}
		})
	}
	for _, line := range []string{
		`python3.12 <<< 'pkill qsdev'`,
		`/usr/bin/python3 <<< 'pkill qsdev'`,
	} {
		ctx := EvalContext{ToolName: "Bash", Command: line}
		if got, reason := sp009.Evaluate(&ctx); got != Deny {
			t.Errorf("SP-009(%q) = %v (%q), want Deny", line, got, reason)
		}
	}
	ctx := EvalContext{ToolName: "Bash", Command: `python3.12 -c "open('.npmrc','w')"`}
	if _, ok := ShellRewritesFile(&ctx, []string{".npmrc"}); !ok {
		t.Errorf("ShellRewritesFile(%q) = false, want true", ctx.Command)
	}
}
