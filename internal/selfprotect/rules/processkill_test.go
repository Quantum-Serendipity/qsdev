package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// sp009Shells are the POSIX shell tools SP-009 judges alike (IsShellTool
// parity): Monitor sends the same tool_input.command as Bash.
var sp009Shells = []string{"Bash", "Monitor"}

// assertSP009 evaluates every command with every POSIX shell tool and
// requires verdict want from SP-009.
func assertSP009(t *testing.T, want Verdict, commands []string) {
	t.Helper()
	for _, tool := range sp009Shells {
		for _, command := range commands {
			t.Run(tool+"/"+command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, Command: command}
				if got, reason := sp009.Evaluate(&ctx); got != want {
					t.Errorf("SP-009(%q) = %v (%q), want %v", command, got, reason, want)
				}
			})
		}
	}
}

// TestSP009_Patterns verifies SP-009 judges the process a kill verb
// targets from its parsed argv (U18-05): a pattern operand is compiled and
// matched against the guarded names, a computed operand may be one, and a
// unit or label holds a guard as a token.
func TestSP009_Patterns(t *testing.T) {
	t.Parallel()
	app := branding.Get().AppName
	assertSP009(t, Deny, []string{
		`pkill -f "q[s]dev"`,
		`pkill -f 'q.dev'`,
		`killall -r 'cl.ude'`,
		`p=cla; pkill ${p}ude`,
		`pkill -f "$PAT"`,
		`taskkill /IM qsdev.exe /F`,
		`taskkill /im QSDEV.EXE`,
		`taskkill /IM qs*.exe /F`,
		`taskkill /IM "$IMG" /F`,
		`systemctl --user stop qsdev-mcp`,
		`launchctl bootout gui/501/com.qsdev.mcp`,
		`pkill -f 'q(s'`,
		`pkill -f '` + strings.Repeat("x", 1025) + `'`,
		`pkill -f '` + app + ` selfprotect'`,
		`pkill -9 -f 'cla.de'`,
		`pkill -HUP q.dev`,
		`pkill -u root q.dev`,
		`sudo pkill -f q.dev`,
		`/usr/bin/pkill claude`,
		`killall --regexp 'q.*'`,
		`killall 'qsde?'`,
		`pgrep qsdev | xargs kill`,
		`echo qsdev | xargs pkill`,
		`env -S 'pkill qsdev'`,
		`busybox pkill qsdev`,
		// A finder's pattern is judged as pkill's is when a kill reads its
		// output, and a command-line pattern that runs through a guarded
		// name counts though it does not match the bare name.
		`kill $(pgrep -f 'q[s]dev')`,
		`pgrep -f 'q.dev' | xargs kill`,
		`kill -9 $(pidof "$NAME")`,
		`kill $(pgrep -f '[c]laude --resume')`,
		`pkill -f 'q[s]dev mcp'`,
		`pkill -f '[c]laude --resume'`,
		`pkill -f '/usr/bin/q[s]dev.exe'`,
		`pkill --full 'cl(a|x)ude --print'`,
		`pgrep -af '/opt/bin/q[s]dev mcp' | xargs kill`,
		`pkill -f 'sde[v] mcp'`,
	})
}

// TestSP009_FinderPatternsAllowed pins the finder and pattern forms that
// cannot reach a guarded process.
func TestSP009_FinderPatternsAllowed(t *testing.T) {
	t.Parallel()
	assertSP009(t, Allow, []string{
		`kill $(pgrep node)`,
		`pgrep -f 'node server' | xargs kill`,
		`kill $(pgrep -f 'n.de')`,
		`pkill -f 'q[s]devtools'`,
		`pkill -f 'x[q]sdev mcp'`,
		`pkill -f 'node.*vite'`,
		`pkill 'q[s]dev mcp'`, // a name, not a command line, is matched
		`killall -r 'node.*'`,
		`pgrep -f 'q[s]dev'`,
	})
}

// TestPatternSpansGuard checks the walk of a pattern over a command line
// that runs through a guarded name as a word.
func TestPatternSpansGuard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern string
		want    bool
	}{
		{"q[s]dev mcp", true},
		{"/usr/bin/q[s]dev.exe", true},
		{"Q[S]DEV", true},
		{"q.dev serve", true},
		{"dev mcp", true},
		{"^qsdev$", true},
		{"q[s]dev(tools|-mcp)", true},
		{"q[s]devtools", false},
		{"xqsdev", false},
		{"node server", false},
		{"q[^s]dev mcp", false},
		{"q\\wdev", true},
		{"q[a-r]dev", false},
		{"node.*", false},
		{"node.*qsdev", true},
		{"bin/q[s]dev mcp", true},
		{".q[s]dev mcp", true},
		{"/usr/.*/q[s]dev", true},
		{"node.*vite", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()
			prog, err := compileKillPattern(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}
			if got := patternSpansGuard(prog, "qsdev"); got != tt.want {
				t.Errorf("patternSpansGuard(%q, qsdev) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// TestSP009_StillDeniesAsOnBase pins the forms the raw-text regexes denied,
// which the argv judgement keeps denying, also when the kill reaches a
// shell or interpreter as code the parser cannot see.
func TestSP009_StillDeniesAsOnBase(t *testing.T) {
	t.Parallel()
	app := branding.Get().AppName
	assertSP009(t, Deny, []string{
		`kill $(pgrep qsdev)`,
		`kill -9 $(pgrep -f claude)`,
		`pkill qsdev`,
		`killall claude`,
		`systemctl stop ` + app,
		`systemctl disable qsdev.service`,
		`sh -c 'pkill qsdev'`,
		`eval "pkill qsdev"`,
		"bash <<'EOF'\npkill qsdev\nEOF",
		"bash -s <<'EOF'\necho hi\nkill $(pgrep qsdev)\nEOF",
		`bash <<< 'pkill qsdev'`,
		`echo 'pkill qsdev' | sh`,
		"cat <<'EOF' | bash\npkill qsdev\nEOF",
		`python3 -c "import os; os.system('pkill qsdev')"`,
		`perl -e 'system("killall claude")'`,
		`node -e "require('child_process').execSync('pkill qsdev')"`,
		"if then pkill qsdev", // unparsable: the whole line is text
	})
}

// TestSP009_OpaqueRunnersDenyAsOnBase pins the kills base's line-wide text
// check denied through a program whose argv SP-009 does not model: one that
// runs a command line it is given (watch, flock, su, tmux, ssh, at, trap,
// git rebase -x, an interpreter's code), a shell reading a process
// substitution, and a kill verb a line assignment names. A program not
// proven read-only (cmdscan.IsSafeReadCommand) keeps base's statement-local
// text check.
func TestSP009_OpaqueRunnersDenyAsOnBase(t *testing.T) {
	t.Parallel()
	assertSP009(t, Deny, []string{
		`watch -n1 pkill qsdev`,
		`flock /tmp/l pkill qsdev`,
		`script -qc 'pkill qsdev' /dev/null`,
		`su -c 'pkill qsdev'`,
		`runuser -u colin -- pkill qsdev`,
		`strace -f pkill qsdev`,
		`awk 'BEGIN{system("pkill qsdev")}'`,
		`php -r 'system("pkill qsdev");'`,
		`sh < <(echo pkill qsdev)`,
		`bash <(echo pkill qsdev)`,
		`source <(echo pkill qsdev)`,
		`trap 'pkill qsdev' EXIT`,
		`git rebase -x 'pkill qsdev' HEAD~1`,
		`at now <<< 'pkill qsdev'`,
		`systemd-run --user pkill qsdev`,
		`tmux new -d 'pkill qsdev'`,
		`screen -dm pkill qsdev`,
		`parallel kill ::: $(pgrep qsdev)`,
		`K=pkill; $K qsdev`,
		`x=kill; $x $(pgrep qsdev)`,
		`ssh localhost pkill qsdev`,
		`X=1 echo 'pkill qsdev'`,
	})
	assertSP009(t, Allow, []string{
		`K=pkill; $K node`,
		`x=kill; $x $!`,
		`watch -n1 pkill node`,
		`tmux new -d 'make test'`,
		`source <(echo hi)`,
	})
}

// TestSP009_DataTextAllowed verifies text that only mentions a kill verb
// and a guarded name, without a program running it, is data (W1N-63).
func TestSP009_DataTextAllowed(t *testing.T) {
	t.Parallel()
	assertSP009(t, Allow, []string{
		"python3 - <<'EOF'\nrows = ['timeout -k 5 kill-after', 'qsdev']\nfor r in rows:\n    print(r)\nEOF",
		"cat > notes.txt <<'EOF'\nkill the claude process\nEOF",
		`grep -rn 'pkill qsdev' docs/`,
		`echo 'kill qsdev'`,
		`git commit -m 'stop killing qsdev in tests'`,
		`echo 'kill qsdev' | cat`,
		`python3 script.py <<< 'pkill qsdev'`,
		`python3 -c "import os; os.kill(1234, 9)"`,
	})
}

// TestSP009_StillAllowsAsOnBase pins the kills of other processes and the
// read-only process queries SP-009 never denied.
func TestSP009_StillAllowsAsOnBase(t *testing.T) {
	t.Parallel()
	assertSP009(t, Allow, []string{
		`kill %1`,
		`kill -9 1234`,
		`kill $!`,
		`pkill node`,
		`pkill -f 'node server'`,
		`pkill -P $PPID node`,
		`killall -r 'node.*'`,
		`systemctl --user stop foo`,
		`systemctl restart nginx`,
		`systemctl status qsdev`,
		`launchctl list com.qsdev.mcp`,
		`taskkill /IM node.exe /F`,
		`taskkill /PID 1234 /F`,
		`pgrep -a qsdev`,
		`timeout -k 5 30 go test ./...`,
		`go test -run TestKill ./cmd/qsdev/`,
		`qsdev status`,
	})
}

// TestNoGdevLiteralInRules pins that the rules name the shipped processes
// through branding, not the upstream framework's name: gdev is never a
// process qsdev runs.
func TestNoGdevLiteralInRules(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(data)), "gdev") {
			t.Errorf("%s names gdev; name guarded processes through branding", f)
		}
	}
}
