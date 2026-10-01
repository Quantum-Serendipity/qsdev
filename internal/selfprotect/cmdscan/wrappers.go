package cmdscan

import (
	"path"
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

// scriptShells run a script string passed with -c.
var scriptShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "mksh": true, "ash": true,
}

// CommandWordIndexes returns the indexes of the words that can name the program
// run: the first word, and when that is a wrapper, every later word.
func CommandWordIndexes(words []string) []int {
	if len(words) == 0 {
		return nil
	}
	idx := []int{0}
	if commandWrappers[path.Base(words[0])] {
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
// program (no words, or a wrapper that takes the command as one string).
// Unlike CommandWordIndexes, which over-approximates for blocking rules, it
// names exactly one word.
func ProgramWordIndex(words []string) int {
	i := 0
	for i < len(words) {
		name := path.Base(words[i])
		if !commandWrappers[name] {
			return i
		}
		i++
	operands:
		for i < len(words) {
			w := words[i]
			switch {
			case w == "--":
				i++
				break operands
			case slices.Contains(wrapperCommandStrings[name], w):
				return -1
			case len(w) > 1 && w[0] == '-':
				if slices.Contains(wrapperOptionArgs[name], w) {
					i++
				}
				i++
			case w != "" && w[0] >= '0' && w[0] <= '9', isAssignment(w):
				i++
			default:
				break operands
			}
		}
	}
	return -1
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
		name := path.Base(words[i])
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
func IsScriptShell(name string) bool { return scriptShells[path.Base(name)] }
