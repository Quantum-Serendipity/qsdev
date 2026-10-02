package cmdscan

import (
	"slices"
	"strings"
	"unicode"
)

// commandWrappers run their operand as a command, so `sudo git -c ...` and
// `env X=1 git ...` run git.
var commandWrappers = map[string]bool{
	"env": true, "command": true, "exec": true, "builtin": true, "nice": true,
	"nohup": true, "sudo": true, "doas": true, "xargs": true, "time": true,
	"timeout": true, "stdbuf": true, "ionice": true, "setsid": true,
	"chrt": true, "taskset": true, "unbuffer": true,
}

// wrapperOptionArgs lists the options of a wrapper that take the next word as
// their argument, so it is not the program (`timeout -s KILL 30 cmd`).
var wrapperOptionArgs = map[string][]string{
	"env":     {"-u", "--unset", "-C", "--chdir"},
	"exec":    {"-a"},
	"nice":    {"-n", "--adjustment"},
	"sudo":    {"-u", "--user", "-g", "--group", "-C", "--close-from", "-D", "--chdir", "-h", "--host", "-p", "--prompt", "-r", "--role", "-t", "--type", "-U", "--other-user", "-T", "--command-timeout"},
	"doas":    {"-u", "-C"},
	"xargs":   {"-a", "--arg-file", "-d", "--delimiter", "-E", "-I", "-L", "-n", "--max-args", "-P", "--max-procs", "-s", "--max-chars"},
	"time":    {"-f", "--format", "-o", "--output"},
	"timeout": {"-s", "--signal", "-k", "--kill-after"},
	"stdbuf":  {"-i", "-o", "-e"},
	"ionice":  {"-c", "--class", "-n", "--classdata", "-p", "--pid", "-P", "--pgid", "-u", "--uid"},
}

// wrapperCommandStrings are options whose argument is itself a command line
// (`env -S 'python3 -u'`), so the program is not a single later word.
var wrapperCommandStrings = map[string][]string{
	"env": {"-S", "--split-string"},
}

// wrapperLookups are options with which a wrapper only looks its operands up
// instead of running them (`command -v git` prints where git is).
var wrapperLookups = map[string][]string{
	"command": {"-v", "-V"},
}

// scriptShells run a script string passed with -c.
var scriptShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "mksh": true, "ash": true,
}

// shellBuiltins are the builtins of bash, which Claude Code runs hooks
// with, and the POSIX special builtins: the shell runs them itself, never
// looking them up on PATH, so a missing file of that name does not stop
// them. Reserved words (if, for, time, ...) parse as shell syntax, not as
// command words, so they need no entry. Every builtin varBuiltins and
// declBuiltins model is listed here too, except nameref, a mksh and zsh
// declaration bash runs from PATH.
var shellBuiltins = map[string]bool{
	".": true, ":": true, "[": true, "alias": true, "bg": true, "bind": true,
	"break": true, "builtin": true, "caller": true, "cd": true, "command": true,
	"compgen": true, "complete": true, "compopt": true, "continue": true,
	"declare": true, "dirs": true, "disown": true, "echo": true, "enable": true,
	"eval": true, "exec": true, "exit": true, "export": true, "false": true,
	"fc": true, "fg": true, "getopts": true, "hash": true, "help": true,
	"history": true, "jobs": true, "kill": true, "let": true, "local": true,
	"logout": true, "mapfile": true, "popd": true, "printf": true, "pushd": true,
	"pwd": true, "read": true, "readarray": true, "readonly": true, "return": true,
	"set": true, "shift": true, "shopt": true, "source": true, "suspend": true,
	"test": true, "times": true, "trap": true, "true": true, "type": true,
	"typeset": true, "ulimit": true, "umask": true, "unalias": true, "unset": true,
	"wait": true,
}

// IsShellBuiltin reports whether the command word name is a shell builtin,
// which the shell runs without a PATH lookup. Builtins are matched exactly:
// a path such as /usr/bin/cd, or another case, names a file.
func IsShellBuiltin(name string) bool { return shellBuiltins[name] }

// CommandWordIndexes returns the indexes of the words that can name the program
// run: the first word, and when that is a wrapper, every later word.
func CommandWordIndexes(words []string) []int {
	if len(words) == 0 {
		return nil
	}
	idx := []int{0}
	if commandWrappers[wrapperName(words[0])] {
		for i := 1; i < len(words); i++ {
			idx = append(idx, i)
		}
	}
	return idx
}

// ProgramWordIndex returns the index of the word naming the program the words
// run, following wrappers (`env A=1 timeout 30 nice -n 5 python3 x.py` runs
// python3): a wrapper's options and their arguments, numeric operands
// (durations, priorities, CPU masks) and VAR=value assignments are skipped up
// to the next command word. It returns -1 when no single word names the
// program (no words, a wrapper given no command, `command -v`, which only
// looks its operands up, or a wrapper that takes the command as one string;
// Program tells these apart). Unlike CommandWordIndexes, which
// over-approximates for blocking rules, it names exactly one word.
func ProgramWordIndex(words []string) int {
	return Program(words).Index
}

// ProgramRun describes how the words of one simple command run a program
// (see Program).
type ProgramRun struct {
	// Index is the index of the word naming the program, as ProgramWordIndex
	// returns it, or -1.
	Index int
	// CommandString is set when a wrapper takes the command as one string
	// (`env -S 'python3 -u'`): a program runs, but no single word names it.
	CommandString bool
	// LookupOnly is set when `command -v`/`-V` only looks its operands up,
	// which tests whether they resolve.
	LookupOnly bool
	// ShellRuns is set when the shell itself runs the program word, directly
	// or through the builtins command and builtin, so a builtin of that name
	// runs. Any other wrapper (env, timeout, exec, ...) execs it as a file.
	ShellRuns bool
	// Exec is set when exec, reached by the shell, runs the program: the
	// shell is replaced by it, so nothing after the statement runs.
	Exec bool
}

// RunsProgram reports whether the words run a program, named or not.
func (r ProgramRun) RunsProgram() bool { return r.Index >= 0 || r.CommandString }

// Program describes the program the words run, following wrappers the way
// ProgramWordIndex does. When Index is -1 and neither CommandString nor
// LookupOnly is set, the words run no program at all: there are no words, or
// a wrapper is given no command (`exec 2>/dev/null`, `timeout 30`).
func Program(words []string) ProgramRun {
	run := ProgramRun{ShellRuns: true}
	i := 0
	for i < len(words) {
		name := wrapperName(words[i])
		if !commandWrappers[name] {
			run.Index = i
			return run
		}
		if run.ShellRuns && words[i] == "exec" {
			run.Exec = true
		}
		run.ShellRuns = run.ShellRuns && shellRunWrappers[words[i]]
		i++
	operands:
		for i < len(words) {
			w := words[i]
			switch {
			case w == "--":
				i++
				break operands
			case isOptionWord(w):
				opts, n := parseWrapperOption(name, words, i)
				if slices.ContainsFunc(opts, func(o WrapperOption) bool { return o.Is(wrapperCommandStrings[name]...) }) {
					return ProgramRun{Index: -1, CommandString: true, Exec: run.Exec}
				}
				if slices.ContainsFunc(opts, func(o WrapperOption) bool { return o.Is(wrapperLookups[name]...) }) {
					return ProgramRun{Index: -1, LookupOnly: true}
				}
				i += n
			case w != "" && w[0] >= '0' && w[0] <= '9', isAssignment(w):
				i++
			default:
				break operands
			}
		}
	}
	return ProgramRun{Index: -1}
}

// shellRunWrappers are the wrappers that are builtins running their operand
// as a command of the shell, builtins included. They are matched exactly, as
// builtins are (see IsShellBuiltin).
var shellRunWrappers = map[string]bool{"command": true, "builtin": true}

// WrapperOption is one option given to a wrapper command: its name as
// written (each letter of a short-option cluster separately, as `-u`; a long
// option possibly abbreviated, as `--ignore-env`) and its argument, when the
// option takes one.
type WrapperOption struct {
	Name, Arg string
}

// Is reports whether o is one of spellings: a short option exactly, or a
// long option by any abbreviation getopt_long accepts (`--ignore-env` is
// --ignore-environment). A lone "-" matches only "-".
func (o WrapperOption) Is(spellings ...string) bool {
	for _, s := range spellings {
		long, isLong := strings.CutPrefix(s, "--")
		name, oLong := strings.CutPrefix(o.Name, "--")
		if isLong && oLong && isLongOptionPrefix(name, long) || !isLong && o.Name == s {
			return true
		}
	}
	return false
}

// WrapperOptions returns the options the wrapper program wrapper (`env`,
// `sudo`, ...) is given in args, the words after it, up to its first operand:
// "--" or the first word that is not an option. A short-option cluster
// (`-iu HOME`, `-uHOME`) yields each option, and an option that takes an
// argument (wrapperOptionArgs, wrapperCommandStrings) takes the rest of the
// cluster, the text after "=" of a long option, or the next word.
func WrapperOptions(wrapper string, args []string) []WrapperOption {
	name := wrapperName(wrapper)
	var out []WrapperOption
	for i := 0; i < len(args) && args[i] != "--" && isOptionWord(args[i]); {
		opts, n := parseWrapperOption(name, args, i)
		out = append(out, opts...)
		i += n
	}
	return out
}

// isOptionWord reports whether w is an option word: it starts with "-" and
// is not "--". A lone "-" is env's spelling of -i.
func isOptionWord(w string) bool {
	return strings.HasPrefix(w, "-") && w != "--"
}

// parseWrapperOption parses the option word words[i] of the wrapper name and
// returns its options and how many words they take (1, or 2 when the last
// option's argument is the next word).
func parseWrapperOption(name string, words []string, i int) ([]WrapperOption, int) {
	w := words[i]
	takesArg := func(opt string) bool {
		return slices.ContainsFunc(append(slices.Clone(wrapperOptionArgs[name]), wrapperCommandStrings[name]...),
			func(s string) bool { return WrapperOption{Name: opt}.Is(s) })
	}
	nextArg := func() (string, int) {
		if i+1 < len(words) {
			return words[i+1], 2
		}
		return "", 1
	}
	if long, ok := strings.CutPrefix(w, "--"); ok {
		if opt, arg, hasEq := strings.Cut(long, "="); hasEq {
			return []WrapperOption{{Name: "--" + opt, Arg: arg}}, 1
		}
		if takesArg(w) {
			arg, n := nextArg()
			return []WrapperOption{{Name: w, Arg: arg}}, n
		}
		return []WrapperOption{{Name: w}}, 1
	}
	if w == "-" {
		return []WrapperOption{{Name: w}}, 1
	}
	var opts []WrapperOption
	for k := 1; k < len(w); k++ {
		opt := "-" + w[k:k+1]
		if !takesArg(opt) {
			opts = append(opts, WrapperOption{Name: opt})
			continue
		}
		if k+1 < len(w) {
			return append(opts, WrapperOption{Name: opt, Arg: w[k+1:]}), 1
		}
		arg, n := nextArg()
		return append(opts, WrapperOption{Name: opt, Arg: arg}), n
	}
	return opts, 1
}

// wrapperName returns the name a command word is looked up by in the wrapper
// and shell tables: its program name (see ProgramName), case-folded, since
// Windows and macOS file systems resolve ENV.EXE or Env to env.
func wrapperName(word string) string {
	return strings.ToLower(ProgramName(word))
}

// isAssignment reports whether w is a VAR=value word.
func isAssignment(w string) bool {
	name, _, ok := strings.Cut(w, "=")
	if !ok || name == "" {
		return false
	}
	for i, r := range name {
		if r != '_' && !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

// ShellScript returns the script the words run through a shell's -c option
// (`sh -c '...'`, `bash -ec '...'`) or eval.
func ShellScript(words []string) (string, bool) {
	for _, i := range CommandWordIndexes(words) {
		name := wrapperName(words[i])
		if name == "eval" {
			return strings.Join(words[i+1:], " "), true
		}
		if !scriptShells[name] {
			continue
		}
		rest := words[i+1:]
		for j, a := range rest {
			if len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 'c') && j+1 < len(rest) {
				return rest[j+1], true
			}
		}
	}
	return "", false
}

// IsScriptShell reports whether the program named name is a shell that runs
// a script string passed with -c.
func IsScriptShell(name string) bool { return scriptShells[wrapperName(name)] }
