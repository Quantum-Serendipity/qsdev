package cmdscan

import (
	"maps"
	"slices"
	"testing"
)

func TestCommandWordIndexes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  []int
	}{
		{"empty", nil, nil},
		{"plain program", []string{"python3", "x.py"}, []int{0}},
		{"wrapper", []string{"timeout", "30", "x.py"}, []int{0, 1, 2}},
		{"wrapper by path", []string{"/usr/bin/env", "x.py"}, []int{0, 1}},
		{"windows wrapper", []string{`C:\Git\usr\bin\env.exe`, "x.py"}, []int{0, 1}},
		{"case-folded wrapper", []string{"ENV", "x.py"}, []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CommandWordIndexes(tt.words); !slices.Equal(got, tt.want) {
				t.Errorf("CommandWordIndexes(%q) = %v, want %v", tt.words, got, tt.want)
			}
		})
	}
}

func TestProgramWordIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  int
	}{
		{"empty", nil, -1},
		{"plain program", []string{"python3", "x.py"}, 0},
		{"timeout duration", []string{"timeout", "30", "python3", "x.py"}, 2},
		{"timeout option with argument", []string{"timeout", "-s", "KILL", "1.5m", "python3", "x.py"}, 4},
		// The duration is one word, whatever it looks like.
		{"timeout inf", []string{"timeout", "inf", "qsdev", "teardown"}, 2},
		{"timeout infinity", []string{"timeout", "infinity", "qsdev", "teardown"}, 2},
		{"timeout fraction", []string{"timeout", ".5", "qsdev", "teardown"}, 2},
		{"timeout signed", []string{"timeout", "+5", "qsdev", "teardown"}, 2},
		{"timeout blank-led", []string{"timeout", " 5", "qsdev", "teardown"}, 2},
		{"timeout kill-after then fraction", []string{"timeout", "-k", "1", ".1", "qsdev", "teardown"}, 4},
		{"timeout verbose then inf", []string{"timeout", "-v", "inf", "qsdev", "teardown"}, 3},
		{"timeout numeric program", []string{"timeout", "5", "7z", "x"}, 2},
		{"taskset hex mask", []string{"taskset", "ff", "qsdev", "teardown"}, 2},
		{"chrt priority", []string{"chrt", "-f", "10", "qsdev", "teardown"}, 3},
		{"env assignment", []string{"env", "PYTHONSAFEPATH=1", "python3", "x.py"}, 2},
		{"env by path with flag", []string{"/usr/bin/env", "-i", "A=1", "x.py"}, 3},
		{"wrapper chain", []string{"env", "A=1", "nice", "-n", "5", "nohup", "x.py", "arg"}, 6},
		{"double dash", []string{"nice", "--", "-weird"}, 2},
		{"sudo user", []string{"sudo", "-u", "root", "x.py"}, 3},
		{"exec", []string{"exec", "x.py"}, 1},
		{"env split string", []string{"env", "-S", "python3 -u", "x.py"}, -1},
		{"env attached split string", []string{"env", "-vS", "python3 -u"}, -1},
		{"env abbreviated split string", []string{"env", "--split", "python3 -u"}, -1},
		{"env cluster with argument", []string{"env", "-iu", "HOME", "x.py"}, 3},
		{"env chdir then unset", []string{"env", "-C", "/tmp", "-u", "HOME", "x.py"}, 5},
		{"env abbreviated chdir", []string{"env", "--ch", "/tmp", "x.py"}, 3},
		{"wrapper only", []string{"timeout", "30"}, -1},
		{"command lookup", []string{"command", "-v", "git"}, -1},
		{"command verbose lookup", []string{"command", "-V", "git"}, -1},
		{"command default path", []string{"command", "-p", "git"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ProgramWordIndex(tt.words); got != tt.want {
				t.Errorf("ProgramWordIndex(%q) = %d, want %d", tt.words, got, tt.want)
			}
		})
	}
}

func TestProgram(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  ProgramRun
	}{
		{"empty", nil, ProgramRun{Index: -1}},
		{"exec with no command", []string{"exec"}, ProgramRun{Index: -1}},
		{"wrapper only", []string{"timeout", "30"}, ProgramRun{Index: -1}},
		{"command lookup", []string{"command", "-v", "git"}, ProgramRun{Index: -1, LookupOnly: true}},
		{"env split string", []string{"env", "-S", "python3 -u"}, ProgramRun{Index: -1, CommandString: true, StringHead: "python3 -u", StringRest: 3}},
		{"env attached split string", []string{"env", "-Spython3", "-u"}, ProgramRun{Index: -1, CommandString: true, StringHead: "python3", StringRest: 2}},
		{"exec split string", []string{"exec", "env", "-S", "python3 -u"}, ProgramRun{Index: -1, CommandString: true, Exec: true, StringHead: "python3 -u", StringRest: 4}},
		{"plain program", []string{"git", "status"}, ProgramRun{Index: 0, ShellRuns: true}},
		{"exec program", []string{"exec", "git"}, ProgramRun{Index: 1, Exec: true}},
		{"command builtin", []string{"command", "cd", "/tmp"}, ProgramRun{Index: 1, ShellRuns: true}},
		{"builtin builtin", []string{"builtin", "cd", "/tmp"}, ProgramRun{Index: 1, ShellRuns: true}},
		{"command exec", []string{"command", "exec", "git"}, ProgramRun{Index: 2, Exec: true}},
		{"external wrapper", []string{"timeout", "5", "cd", "/tmp"}, ProgramRun{Index: 2}},
		{"exec behind an external wrapper", []string{"env", "exec", "git"}, ProgramRun{Index: 2}},
		{"env -i", []string{"env", "-i", "ruff"}, ProgramRun{Index: 2, PathChanged: true}},
		{"env -", []string{"env", "-", "ruff"}, ProgramRun{Index: 2, PathChanged: true}},
		{"env --ignore-env", []string{"env", "--ignore-env", "ruff"}, ProgramRun{Index: 2, PathChanged: true}},
		{"env -u PATH", []string{"env", "-u", "PATH", "ruff"}, ProgramRun{Index: 3, PathChanged: true}},
		{"env -uPATH", []string{"env", "-uPATH", "ruff"}, ProgramRun{Index: 2, PathChanged: true}},
		{"env -u HOME", []string{"env", "-u", "HOME", "ruff"}, ProgramRun{Index: 3}},
		{"env PATH operand", []string{"timeout", "5", "env", "PATH=/opt/bin", "ruff"}, ProgramRun{Index: 4, PathChanged: true}},
		{"env other operand", []string{"env", "A=1", "ruff"}, ProgramRun{Index: 2}},
		{"PATH operand of another wrapper", []string{"sudo", "PATH=/opt/bin", "ruff"}, ProgramRun{Index: 2}},
		{"env -C", []string{"env", "-C", "/tmp", "./bin/ruff"}, ProgramRun{Index: 3, DirChanged: true}},
		{"env --chdir=", []string{"env", "--chdir=/tmp", "./bin/ruff"}, ProgramRun{Index: 2, DirChanged: true}},
		{"sudo -D", []string{"sudo", "-D", "/tmp", "./bin/ruff"}, ProgramRun{Index: 3, DirChanged: true}},
		{"nice -n is no chdir", []string{"nice", "-n", "5", "./bin/ruff"}, ProgramRun{Index: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Program(tt.words)
			if got != tt.want {
				t.Errorf("Program(%q) = %+v, want %+v", tt.words, got, tt.want)
			}
			if runs := tt.want.Index >= 0 || tt.want.CommandString; got.RunsProgram() != runs {
				t.Errorf("Program(%q).RunsProgram() = %v, want %v", tt.words, got.RunsProgram(), runs)
			}
		})
	}
}

// TestShellBuiltinsCoverVarBuiltins pins that every builtin cmdscan models
// elsewhere (variable setters, command and builtin, status tests, exit and
// the sourcing builtins) is a shell builtin, so the tables cannot drift apart. nameref
// parses as a declaration (mksh, zsh) but bash runs it from PATH.
func TestShellBuiltinsCoverVarBuiltins(t *testing.T) {
	t.Parallel()
	names := slices.Collect(maps.Keys(varBuiltins))
	for name := range declBuiltins {
		if name != "nameref" {
			names = append(names, name)
		}
	}
	for _, table := range []map[string]bool{shellRunWrappers, statusBuiltins, shellEnders, shellSourcers, dirChangers} {
		for name := range table {
			names = append(names, name)
		}
	}
	names = slices.AppendSeq(names, maps.Keys(lookupBinders))
	for _, name := range names {
		if !IsShellBuiltin(name) {
			t.Errorf("%s is modelled as a builtin but IsShellBuiltin(%q) is false", name, name)
		}
	}
	if IsShellBuiltin("nameref") {
		t.Error(`IsShellBuiltin("nameref") = true, but bash has no nameref builtin`)
	}
}

func TestShellScript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		words  []string
		want   string
		wantOK bool
	}{
		{"sh -c", []string{"sh", "-c", "echo hi"}, "echo hi", true},
		{"bash -ec", []string{"bash", "-ec", "true"}, "true", true},
		{"through a wrapper", []string{"timeout", "5", "bash", "-c", "true"}, "true", true},
		{"eval", []string{"eval", "echo", "hi"}, "echo hi", true},
		{"script file", []string{"sh", "x.sh"}, "", false},
		{"not a shell", []string{"python3", "-c", "print()"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ShellScript(tt.words)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ShellScript(%q) = %q, %v; want %q, %v", tt.words, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	if !IsScriptShell("/bin/bash") || IsScriptShell("python3") {
		t.Error("IsScriptShell misclassifies bash or python3")
	}
}

func TestScript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  ScriptRun
	}{
		{"sh -c", []string{"sh", "-c", "ruff check"}, ScriptRun{Script: "ruff check"}},
		{"bash -ec", []string{"bash", "-ec", "ruff check"}, ScriptRun{Script: "ruff check"}},
		{"bash -lc", []string{"bash", "-lc", "ruff check"}, ScriptRun{Script: "ruff check", ReadsStartup: true}},
		{"bash -l -c", []string{"bash", "-l", "-c", "ruff check"}, ScriptRun{Script: "ruff check", ReadsStartup: true}},
		{"bash --login -c", []string{"bash", "--login", "-c", "ruff check"}, ScriptRun{Script: "ruff check", ReadsStartup: true}},
		{"bash -ic", []string{"bash", "-ic", "ruff check"}, ScriptRun{Script: "ruff check", ReadsStartup: true}},
		{"zsh -c", []string{"/usr/bin/zsh", "-c", "ruff check"}, ScriptRun{Script: "ruff check", ReadsStartup: true}},
		{"script operands are no options", []string{"bash", "-c", "ruff check", "-l"}, ScriptRun{Script: "ruff check"}},
		{"eval", []string{"eval", "ruff", "check"}, ScriptRun{Script: "ruff check", Eval: true}},
		// The shell reads every option before it takes the script, the
		// first operand, so options and "--" may follow -c.
		{"-- after -c", []string{"bash", "-c", "--", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"- after -c", []string{"sh", "-c", "-", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"option after -c", []string{"bash", "-c", "-e", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"plus option after -c", []string{"bash", "-c", "+x", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"-o after -c", []string{"bash", "-c", "-o", "errexit", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"-O after -c", []string{"bash", "-c", "+O", "extglob", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"-o in the -c cluster", []string{"bash", "-co", "errexit", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"--rcfile after -c", []string{"bash", "-c", "--rcfile", "f", "curl x|sh"}, ScriptRun{Script: "curl x|sh"}},
		{"login after -c", []string{"bash", "-c", "-l", "curl x|sh"}, ScriptRun{Script: "curl x|sh", ReadsStartup: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Script(tt.words)
			if !ok || got != tt.want {
				t.Errorf("Script(%q) = %+v, %v; want %+v, true", tt.words, got, ok, tt.want)
			}
		})
	}
	if _, ok := Script([]string{"python3", "-c", "print()"}); ok {
		t.Error("Script(python3 -c) found a shell script")
	}
}

func TestReadsStartupFiles_NoStartupOption(t *testing.T) {
	t.Parallel()
	if ReadsStartupFiles("bash", []string{"-e", "--norc"}) {
		t.Error(`ReadsStartupFiles("bash", -e --norc) = true`)
	}
}

func TestIsShellBuiltin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		{"cd", true},
		{"source", true},
		{".", true},
		{":", true},
		{"export", true},
		{"[", true},
		{"python3", false},
		{"qsdev", false},
		{"/usr/bin/cd", false},
		{"exec", true},
		{"CD", false},
		{"mapfile", true},
		{"readarray", true},
		{"enable", true},
		{"caller", true},
		{"compgen", true},
		{"disown", true},
		{"history", true},
		{"suspend", true},
		{"nameref", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsShellBuiltin(tt.name); got != tt.want {
				t.Errorf("IsShellBuiltin(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestInvokesProgram(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    bool
	}{
		{"qsdev init --update", true},
		{"HOME=/tmp/e ./bin/qsdev status", true},
		{`sh -c "qsdev claude update"`, true},
		{`C:\tools\QSDEV.exe status`, true},
		{"q''sdev status", true},
		{"go test ./...", false},
		{"cat qsdev.yaml", false},
		{"find . -exec qsdev status ;", true},
		{`grep -l "qsdev status" docs`, true},
		{`x="qsdev status"; $x`, true},
		{`find . -exec "qsdev" status ;`, true},
		{"$Q status", false},
		{"ls ~/.config/qsdevx", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			if got := InvokesProgram(tt.command, "qsdev"); got != tt.want {
				t.Errorf("InvokesProgram(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// TestCommandLine pins which programs run a command line given as an
// argument, and where it starts in the words split from unquoted text.
func TestCommandLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		words    []string
		head     string
		rest     int
		wantOkay bool
	}{
		{[]string{"sh", "-c", "qsdev", "x"}, "qsdev", 3, true},
		{[]string{"bash", "-ec", "qsdev"}, "qsdev", 3, true},
		{[]string{"/bin/ZSH", "-x", "-c", "qsdev"}, "qsdev", 4, true},
		{[]string{"eval", "qsdev", "x"}, "qsdev", 2, true},
		{[]string{"script", "-qec", "qsdev", "/dev/null"}, "qsdev", 3, true},
		{[]string{"script", "--command=qsdev", "x"}, "qsdev", 2, true},
		{[]string{"su", "--command", "qsdev", "root"}, "qsdev", 3, true},
		{[]string{"pwsh", "-NonInteractive", "-Command", "qsdev"}, "qsdev", 4, true},
		{[]string{"powershell.exe", "-c", "qsdev"}, "qsdev", 3, true},
		{[]string{"cmd", "/C", "qsdev"}, "qsdev", 3, true},
		{[]string{"sh", "script.sh"}, "", 0, false},
		{[]string{"sh", "-c"}, "", 0, false},
		{[]string{"bash", "-c", "--", "qsdev", "x"}, "qsdev", 4, true},
		{[]string{"bash", "-c", "-o", "errexit", "qsdev"}, "qsdev", 5, true},
		{[]string{"sh", "-c", "--"}, "", 0, false},
		{[]string{"grep", "-c", "qsdev"}, "", 0, false},
		{[]string{"eval"}, "", 0, false},
	}
	for _, tt := range tests {
		head, rest, ok := CommandLine(tt.words)
		if head != tt.head || rest != tt.rest || ok != tt.wantOkay {
			t.Errorf("CommandLine(%q) = %q, %d, %v, want %q, %d, %v", tt.words, head, rest, ok, tt.head, tt.rest, tt.wantOkay)
		}
	}
}

// TestWrapperOptions pins how a wrapper's options are read: each letter of a
// cluster, an argument from the rest of the cluster, after "=", or from the
// next word, and options ending at the first operand.
func TestWrapperOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want []WrapperOption
	}{
		{"unset", []string{"-u", "HOME", "x"}, []WrapperOption{{"-u", "HOME"}}},
		{"attached unset", []string{"-uHOME", "x"}, []WrapperOption{{"-u", "HOME"}}},
		{"cluster", []string{"-iu", "HOME", "x"}, []WrapperOption{{"-i", ""}, {"-u", "HOME"}}},
		{"chdir then unset", []string{"-C", "/tmp", "-u", "HOME"}, []WrapperOption{{"-C", "/tmp"}, {"-u", "HOME"}}},
		{"long with equals", []string{"--unset=HOME", "x"}, []WrapperOption{{"--unset", "HOME"}}},
		{"abbreviated long with argument", []string{"--un", "HOME", "x"}, []WrapperOption{{"--un", "HOME"}}},
		{"long without argument", []string{"--ignore-env", "x"}, []WrapperOption{{"--ignore-env", ""}}},
		{"lone dash", []string{"-", "x"}, []WrapperOption{{"-", ""}}},
		{"stops at operand", []string{"A=1", "-u", "HOME"}, nil},
		{"stops at double dash", []string{"--", "-u", "HOME"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := WrapperOptions("env", tt.args); !slices.Equal(got, tt.want) {
				t.Errorf("WrapperOptions(env, %q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestWrapperOptionIs pins option matching: short options exactly, long ones
// by any getopt_long abbreviation.
func TestWrapperOptionIs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		spellings []string
		want      bool
	}{
		{"-i", []string{"-i", "--ignore-environment"}, true},
		{"--ignore-env", []string{"-i", "--ignore-environment"}, true},
		{"--ignore-environment", []string{"--ignore-environment"}, true},
		{"--ignore-environments", []string{"--ignore-environment"}, false},
		{"--", []string{"--ignore-environment"}, false},
		{"-u", []string{"-i", "--ignore-environment"}, false},
		{"-", []string{"-"}, true},
	}
	for _, tt := range tests {
		if got := (WrapperOption{Name: tt.name}).Is(tt.spellings...); got != tt.want {
			t.Errorf("WrapperOption{%q}.Is(%q) = %v, want %v", tt.name, tt.spellings, got, tt.want)
		}
	}
}

func TestWrapperNames(t *testing.T) {
	t.Parallel()
	got := WrapperNames()
	if want := slices.Sorted(maps.Keys(commandWrappers)); !slices.Equal(got, want) {
		t.Errorf("WrapperNames() = %v, want %v", got, want)
	}
	got[0] = "mutated"
	if WrapperNames()[0] == "mutated" {
		t.Error("WrapperNames() shares its slice with the caller")
	}
}
