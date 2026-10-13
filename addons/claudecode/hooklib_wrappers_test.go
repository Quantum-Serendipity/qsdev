package claudecode_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// pythonOnlyWrappers are the exec wrappers the Python hooks' shared table
// (_qsdev_hooklib.WRAPPERS) follows but selfprotect's cmdscan does not yet.
// cmdscan follows positionals before the command (flock FILE, chroot
// NEWROOT, script FILE), command-string options (flock -c, script -c) and
// joined operands (watch), but has no grammar for subcommands (tmux new),
// session commands (screen -X) or the remaining tracers, sandboxes and
// launchers, so adding them there needs its own work. Shrink this list as
// cmdscan gains them; never add a name here to hide a wrapper cmdscan
// dropped.
var pythonOnlyWrappers = []string{
	"busybox", "catchsegv", "firejail", "ltrace", "mono",
	"nsenter", "proot", "screen", "setpriv", "strace",
	"systemd-run", "tmux",
}

// hookLibEval loads the shared hook library by path, as the hooks do, runs
// body (Python statements that set `result`) and decodes result's JSON.
func hookLibEval(t *testing.T, body string, out any) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping hook library test")
	}
	lib, err := filepath.Abs(hookLibTemplate)
	if err != nil {
		t.Fatal(err)
	}
	prog := "import importlib.util, json, os\n" +
		"spec = importlib.util.spec_from_file_location('_qsdev_hooklib', os.environ['LIB_PATH'])\n" +
		"lib = importlib.util.module_from_spec(spec)\n" +
		"spec.loader.exec_module(lib)\n" +
		body + "\nprint(json.dumps(result))\n"
	cmd := exec.Command(python, "-c", prog)
	cmd.Env = hookEnv(t, "LIB_PATH="+lib)
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v (stdout %q)", err, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("bad output %q: %v", raw, err)
	}
}

// TestHookLibWrappersMatchCmdscan guards drift between the two exec-wrapper
// tables: every wrapper selfprotect's cmdscan follows is in the hooks' table,
// and the hooks' extras are exactly pythonOnlyWrappers. A wrapper added to
// only one side fails.
func TestHookLibWrappersMatchCmdscan(t *testing.T) {
	t.Parallel()
	var lib []string
	hookLibEval(t, "result = sorted(lib.WRAPPERS)", &lib)
	goNames := cmdscan.WrapperNames()
	for _, name := range goNames {
		if !slices.Contains(lib, name) {
			t.Errorf("cmdscan follows wrapper %q but _qsdev_hooklib.WRAPPERS does not", name)
		}
	}
	var extra []string
	for _, name := range lib {
		if !slices.Contains(goNames, name) {
			extra = append(extra, name)
		}
	}
	if want := slices.Sorted(slices.Values(pythonOnlyWrappers)); !slices.Equal(extra, want) {
		t.Errorf("wrappers only the hooks follow = %v, want pythonOnlyWrappers %v", extra, want)
	}
}

// TestHookLib_OneWrapperTable pins that the hooks share one exec-wrapper
// table, in the hook library, instead of a diverging copy per hook (U17-07).
func TestHookLib_OneWrapperTable(t *testing.T) {
	t.Parallel()
	scripts, err := filepath.Glob(filepath.Join("templates", "hooks", "*.py"))
	if err != nil {
		t.Fatal(err)
	}
	table := regexp.MustCompile(`(?m)^_?WRAPPERS\b`)
	var holders []string
	for _, script := range scripts {
		src, err := os.ReadFile(script)
		if err != nil {
			t.Fatal(err)
		}
		if table.Match(src) {
			holders = append(holders, filepath.Base(script))
		}
	}
	if want := []string{filepath.Base(hookLibTemplate)}; !slices.Equal(holders, want) {
		t.Errorf("wrapper tables in %v, want only %v", holders, want)
	}
}

// TestHookLibUnwrap pins the shared wrapper grammar: the command a wrapped
// argv really runs, and the shell scripts the wrappers run themselves.
func TestHookLibUnwrap(t *testing.T) {
	t.Parallel()
	type result struct {
		Argv    []string `json:"argv"`
		Scripts []string `json:"scripts"`
	}
	cases := []struct {
		name string
		argv []string
		want result
	}{
		{"plain", []string{"rm", "-rf", "x"}, result{[]string{"rm", "-rf", "x"}, nil}},
		{"assignment and env", []string{"A=1", "env", "-u", "X", "B=2", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"env -S splits", []string{"env", "-S", "rm -rf x", "y"}, result{[]string{"rm", "-rf", "x", "y"}, nil}},
		{"env -S unbalanced quote", []string{"env", "-S", "rm -rf 'x"}, result{[]string{"rm", "-rf", "x"}, nil}},
		{"sudo by path", []string{"/usr/bin/SUDO", "-u", "root", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"timeout duration", []string{"timeout", "-s", "KILL", "5", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"timeout fractional duration", []string{"timeout", ".5", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"timeout inf duration", []string{"timeout", "inf", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"timeout blank-led duration", []string{"timeout", " 5", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"timeout -v then duration", []string{"timeout", "-v", "infinity", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"command -v looks up", []string{"command", "-v", "rm"}, result{[]string{}, nil}},
		{"unbuffer", []string{"unbuffer", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"flock file", []string{"flock", "/tmp/l", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"flock -c", []string{"flock", "/tmp/l", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"watch joins", []string{"watch", "-n", "5", "rm", "-rf", "x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"script -c", []string{"script", "-q", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"chroot", []string{"chroot", "--userspec", "u:g", "/", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen detached", []string{"screen", "-dmS", "s", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -D -m runs in the foreground", []string{"screen", "-Dm", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen reattach", []string{"screen", "-r", "s"}, result{[]string{}, nil}},
		{"screen list", []string{"screen", "-ls"}, result{[]string{}, nil}},
		{"tmux new", []string{"tmux", "new", "-d", "-s", "s", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux top-level options", []string{"tmux", "-L", "sock", "new-window", "-t", "s:1", "rm", "x"}, result{[]string{}, []string{"rm x"}}},
		{"tmux split-window -f is boolean", []string{"tmux", "splitw", "-f", "rm", "x"}, result{[]string{}, []string{"rm x"}}},
		{"tmux -c script", []string{"tmux", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		// Any other tmux command is judged as a command line of its own words,
		// so an install behind an unmodelled subcommand still meets the guards.
		{"tmux ls", []string{"tmux", "ls"}, result{[]string{}, []string{"ls"}}},
		{"tmux attach", []string{"tmux", "attach", "-t", "s"}, result{[]string{}, []string{"attach -t s"}}},
		// tmux resolves a unique prefix of a command name.
		{"tmux new-s prefix", []string{"tmux", "new-s", "-d", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux new-w prefix", []string{"tmux", "new-w", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux split prefix", []string{"tmux", "split", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		// `;` chains tmux commands: a leading harmless one hides nothing.
		{"tmux chain", []string{"tmux", "ls", ";", "new", "-d", "rm -rf x"}, result{[]string{}, []string{"ls", "rm -rf x"}}},
		{"tmux chain glued", []string{"tmux", "ls;", "new", "-d", "rm -rf x"}, result{[]string{}, []string{"ls", "rm -rf x"}}},
		{"tmux run-shell", []string{"tmux", "run-shell", "-b", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux run", []string{"tmux", "run", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux display-popup", []string{"tmux", "display-popup", "-w", "80", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux pipe-pane", []string{"tmux", "pipe-pane", "-o", "rm -rf x"}, result{[]string{}, []string{"rm -rf x"}}},
		{"tmux if-shell", []string{"tmux", "if-shell", "rm -rf x", "new -d 'rm -rf y'"}, result{[]string{}, []string{"rm -rf x", "rm -rf y"}}},
		{"tmux unknown command keeps its words and follows its operands", []string{"tmux", "foo", "npm", "install", "x"}, result{[]string{}, []string{"foo npm install x", "npm", "install", "x"}}},
		// screen -X/-Q send a command to a running session.
		{"screen -X exec", []string{"screen", "-X", "exec", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -X exec fdpat", []string{"screen", "-X", "exec", ".!.", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -Q exec", []string{"screen", "-Q", "exec", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -X screen", []string{"screen", "-S", "w", "-X", "screen", "-t", "t", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -x then -X", []string{"screen", "-x", "s", "-X", "exec", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -r then -X", []string{"screen", "-r", "s", "-X", "exec", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"screen -X stuff", []string{"screen", "-X", "stuff", `rm x\n`}, result{[]string{}, []string{"rm x\n"}}},
		{"screen -X other command", []string{"screen", "-X", "quit"}, result{[]string{"quit"}, nil}},
		{"screen reattach to session", []string{"screen", "-r", "s"}, result{[]string{}, nil}},
		// screen -R reattaches, else creates a session running the command.
		// The word after -R may be the session name or the command, so the
		// command line from that word on is judged as a script too.
		{"screen -R creates", []string{"screen", "-R", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -RR without a name", []string{"screen", "-RR", "rm", "x"}, result{[]string{"x"}, []string{"rm x"}}},
		{"screen -dRR creates", []string{"screen", "-dRR", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -xRR creates", []string{"screen", "-xRR", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -dR creates", []string{"screen", "-dR", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -r -R creates", []string{"screen", "-r", "-R", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -D -RR creates", []string{"screen", "-D", "-RR", "s", "rm", "x"}, result{[]string{"rm", "x"}, []string{"s rm x"}}},
		{"screen -R alone", []string{"screen", "-R"}, result{[]string{}, nil}},
		{"screen -dRR session only", []string{"screen", "-dRR", "s"}, result{[]string{}, []string{"s"}}},
		{"script -qc screen -R", []string{"script", "-qc", "screen -R s rm x", "/dev/null"}, result{[]string{}, []string{"screen -R s rm x"}}},
		// tmux send-keys types into a shell; other subcommands' operands are
		// followed as tmux command lines.
		{"tmux send-keys", []string{"tmux", "send-keys", "-t", "x", "rm x", "Enter"}, result{[]string{}, []string{"rm x Enter"}}},
		{"tmux set default-command", []string{"tmux", "set", "-g", "default-command", "rm x"},
			result{[]string{}, []string{"set -g default-command rm x", "default-command", "rm x", "x"}}},
		{"tmux bind run-shell", []string{"tmux", "bind", "k", "run-shell", "rm x"},
			result{[]string{}, []string{"bind k run-shell rm x", "k", "rm x", "x"}}},
		{"tmux set-hook nested", []string{"tmux", "set-hook", "-g", "h", `run-shell "rm x"`},
			result{[]string{}, []string{`set-hook -g h run-shell "rm x"`, "h", "rm x"}}},
		// script's operand is its typescript file; -c may follow it.
		{"script FILE -c", []string{"script", "/dev/null", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"script -q FILE -c", []string{"script", "-q", "/dev/null", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"script -c FILE", []string{"script", "-qc", "rm x", "/dev/null"}, result{[]string{}, []string{"rm x"}}},
		{"script FILE only", []string{"script", "-q", "/dev/null"}, result{[]string{}, nil}},
		{"flock FILE -n -c", []string{"flock", "/tmp/l", "-n", "-c", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"sudo -h host", []string{"sudo", "-h", "host", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		{"ssh is remote, not a wrapper", []string{"ssh", "h", "rm", "-rf", "~"}, result{[]string{"ssh", "h", "rm", "-rf", "~"}, nil}},
		{"nested wrappers", []string{"sudo", "nice", "-n", "5", "nohup", "rm", "x"}, result{[]string{"rm", "x"}, nil}},
		// parallel with a command word appends the ::: arguments to it; with
		// none, each argument up to the next :::-family separator is a
		// command line (W1N-08).
		// Each command line takes one argument of every ::: source (the
		// cartesian product); a template's replacement strings take them,
		// else they are appended, spliced as words and quoted as one; each
		// combination is also judged alone.
		{"parallel command word", []string{"parallel", "rm", "-rf", ":::", "x"}, result{[]string{"rm", "-rf", ":::", "x"}, []string{"x", "rm -rf x"}}},
		{"parallel ::: lines", []string{"parallel", "-j1", ":::", "ls", "rm -rf x"}, result{[]string{}, []string{"ls", "rm -rf x"}}},
		{"parallel ::: sources combine", []string{"parallel", ":::", "rm x", ":::", "a", "b"}, result{[]string{}, []string{"rm x a", "rm x b"}}},
		{"parallel ::: one word per source", []string{"parallel", ":::", "npm", ":::", "install", ":::", "evil"}, result{[]string{}, []string{"npm install evil"}}},
		{"parallel :::: files are not read", []string{"parallel", ":::", "rm x", "::::", "f"}, result{[]string{}, []string{"rm x"}}},
		{"parallel :::+ links", []string{"parallel", "--will-cite", ":::", "rm x", ":::+", "a"}, result{[]string{}, []string{"rm x a"}}},
		{"parallel :::+ lines", []string{"parallel", ":::+", "rm x"}, result{[]string{}, []string{"rm x"}}},
		{"parallel {} runs the argument", []string{"parallel", "{}", ":::", "npm install evil"},
			result{[]string{"{}", ":::", "npm install evil"}, []string{"npm install evil", "'npm install evil'"}}},
		{"parallel sh -c appends the argument", []string{"parallel", "sh", "-c", ":::", "npm install evil"},
			result{[]string{"sh", "-c", ":::", "npm install evil"}, []string{"npm install evil", "sh -c npm install evil", "sh -c 'npm install evil'"}}},
		{"parallel positional replacement", []string{"parallel", "echo", "{1}", "{2}", ":::", "a", ":::", "b"},
			result{[]string{"echo", "{1}", "{2}", ":::", "a", ":::", "b"}, []string{"a b", "echo a b"}}},
		{"parallel --env takes a value", []string{"parallel", "--env", "X", ":::", "npm install evil"}, result{[]string{}, []string{"npm install evil"}}},
		{"parallel past the combination cap", []string{"parallel", ":::", "a", "b", "c", "d", "e", "f", "g", "h", ":::", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			result{[]string{}, []string{"a", "b", "c", "d", "e", "f", "g", "h", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a b c d e f g h 1 2 3 4 5 6 7 8 9"}}},
		{"parallel :::: files keeps its words", []string{"parallel", "::::", "cmds.txt"}, result{[]string{"::::", "cmds.txt"}, nil}},
		{"parallel ::: empty", []string{"parallel", ":::"}, result{[]string{}, nil}},
		// No command word and no ::: source: the jobs come from stdin or a
		// file, which cannot be read, so no line is known.
		{"parallel alone", []string{"parallel"}, result{[]string{}, nil}},
		{"parallel options only", []string{"parallel", "-j4"}, result{[]string{}, nil}},
		{"parallel --version", []string{"parallel", "--version"}, result{[]string{}, nil}},
		{"sudo parallel", []string{"sudo", "parallel"}, result{[]string{}, nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in, err := json.Marshal(tc.argv)
			if err != nil {
				t.Fatal(err)
			}
			var got result
			hookLibEval(t, "argv, scripts = lib.unwrap(json.loads("+pyStringLiteral(string(in))+"))\n"+
				"result = {'argv': argv, 'scripts': scripts or None}", &got)
			if got.Argv == nil {
				got.Argv = []string{}
			}
			if !slices.Equal(got.Argv, tc.want.Argv) || !slices.Equal(got.Scripts, tc.want.Scripts) {
				t.Errorf("unwrap(%q) = %+v, want %+v", tc.argv, got, tc.want)
			}
		})
	}
}

// pyStringLiteral renders s as a Python string literal (JSON string syntax is a
// subset Python accepts).
func pyStringLiteral(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
