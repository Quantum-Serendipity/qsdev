package rules

import (
	"path"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// GitCodeExecutionRuleID identifies the denials of GitCodeExecution.
const GitCodeExecutionRuleID = "GIT-001"

// maxGitScriptDepth bounds how deep GitCodeExecution follows shell scripts
// nested in `sh -c` or eval.
const maxGitScriptDepth = 3

// reGitWord finds git in a command that cannot be parsed (fail closed).
var reGitWord = regexp.MustCompile(`(^|[^A-Za-z0-9_.-])git([^A-Za-z0-9_.-]|$)`)

// GitCodeExecution reports why a Bash command runs git in a way that can run a
// program no permission rule or hook inspects, or that skips the repository's
// hooks; ok is false when it does neither. The generated permission rules deny
// the plain spellings (`git -c *`, `git config *`, `git * --no-verify*`), but a
// prefix glob cannot see a global option placed after another one (`git -C .
// -c alias.x='!sh' x`), an abbreviated or clustered flag (`--no-veri`, `-qn`),
// configuration passed in the environment, or git inside a shell wrapper, and
// cannot tell an option from the same text in a commit message. It denies:
//
//   - per-invocation configuration: a global -c or --config-env option, or a
//     GIT_CONFIG_* variable set on the line (alias.<name>=!cmd, core.pager,
//     core.fsmonitor, core.hooksPath and filter drivers all run programs);
//   - --exec-path=<dir> and GIT_EXEC_PATH, which choose the programs git runs,
//     and GIT_EXTERNAL_DIFF, which git diff runs;
//   - the config subcommand, which persists such settings;
//   - --no-verify (or an abbreviation of it) on a subcommand that runs hooks,
//     and -n in a commit or am short-option cluster;
//   - --output, which writes git's output to any file, .git/config included.
func GitCodeExecution(ctx *EvalContext) (string, bool) {
	if ctx.ToolName != "Bash" || ctx.Command == "" {
		return "", false
	}
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		if reGitWord.MatchString(looseText(ctx.Command)) {
			return "the command runs git but cannot be parsed, so its options cannot be checked", true
		}
		return "", false
	}
	return gitCodeExecution(cmds, 0)
}

func gitCodeExecution(cmds []cmdscan.Command, depth int) (string, bool) {
	envSet := ""
	for _, c := range cmds {
		if v := gitEnvAssignment(c); v != "" {
			envSet = v
		}
	}
	for _, c := range cmds {
		words := append([]string{c.Name}, c.Args...)
		if script, ok := shellScript(words); ok {
			if depth >= maxGitScriptDepth {
				if reGitWord.MatchString(script) {
					return "git runs inside too many nested shell scripts to be checked", true
				}
				continue
			}
			sub, err := cmdscan.Parse(script)
			if err != nil {
				if reGitWord.MatchString(script) {
					return "a shell script runs git but cannot be parsed, so its options cannot be checked", true
				}
				continue
			}
			if reason, bad := gitCodeExecution(sub, depth+1); bad {
				return reason, true
			}
		}
		args, ok := gitInvocation(words)
		if !ok {
			continue
		}
		if envSet != "" {
			return envSet + " is set for a git command: it can make git run any program", true
		}
		if reason := gitArgsReason(args); reason != "" {
			return reason, true
		}
	}
	return "", false
}

// isGitEnvVar reports whether setting the variable can make git run another
// program: configuration from the environment (GIT_CONFIG_PARAMETERS,
// GIT_CONFIG_COUNT/KEY_n/VALUE_n, GIT_CONFIG_GLOBAL, ...), the directory git
// runs its programs from, and the external diff program.
func isGitEnvVar(name string) bool {
	return strings.HasPrefix(name, "GIT_CONFIG") || name == "GIT_EXEC_PATH" || name == "GIT_EXTERNAL_DIFF"
}

// gitEnvAssignment returns the name of a git variable that c sets: as a prefix
// or bare assignment, through export/declare, or as an env operand.
func gitEnvAssignment(c cmdscan.Command) string {
	for _, name := range c.Assigns {
		if isGitEnvVar(name) {
			return name
		}
	}
	switch path.Base(c.Name) {
	case "export", "declare", "typeset", "local", "readonly", "env":
		for _, a := range c.Args {
			if name, _, _ := strings.Cut(a, "="); isGitEnvVar(name) {
				return name
			}
		}
	}
	return ""
}

// commandWrappers run their operand as a command, so `sudo git -c ...` and
// `env X=1 git ...` run git.
var commandWrappers = map[string]bool{
	"env": true, "command": true, "exec": true, "builtin": true, "nice": true,
	"nohup": true, "sudo": true, "doas": true, "xargs": true, "time": true,
	"timeout": true, "stdbuf": true, "ionice": true, "setsid": true,
	"chrt": true, "taskset": true, "unbuffer": true,
}

// scriptShells run a script string passed with -c.
var scriptShells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "mksh": true, "ash": true,
}

// commandWordIndexes returns the indexes of the words that can name the program
// run: the first word, and when that is a wrapper, every later word.
func commandWordIndexes(words []string) []int {
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

// gitInvocation returns the arguments of the git program the words run.
func gitInvocation(words []string) ([]string, bool) {
	for _, i := range commandWordIndexes(words) {
		if path.Base(words[i]) == "git" {
			return words[i+1:], true
		}
	}
	return nil, false
}

// shellScript returns the script the words run through a shell's -c option
// (`sh -c '...'`, `bash -ec '...'`) or eval.
func shellScript(words []string) (string, bool) {
	for _, i := range commandWordIndexes(words) {
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

// gitGlobalValueOptions are git's global options that take their value as the
// next word.
var gitGlobalValueOptions = map[string]bool{
	"-C": true, "--git-dir": true, "--work-tree": true, "--namespace": true, "--attr-source": true,
}

// gitArgsReason checks git's global options and then its subcommand.
func gitArgsReason(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c", strings.HasPrefix(a, "--config-env"):
			return "git " + a + " sets configuration for one command, which can run any program (alias.<name>=!cmd, core.pager, core.fsmonitor, core.hooksPath)"
		case strings.HasPrefix(a, "--exec-path="):
			return "git --exec-path= chooses the directory git runs its programs from"
		case gitGlobalValueOptions[a]:
			i++
		case !isFlag(a):
			return gitSubcommandReason(a, args[i+1:])
		}
	}
	return ""
}

// gitHookSubcommands run repository hooks that --no-verify skips.
var gitHookSubcommands = map[string]bool{
	"commit": true, "merge": true, "pull": true, "push": true, "am": true, "rebase": true,
}

// gitShortOptions lists, for the subcommands whose -n skips hooks, the short
// options that take a value (the rest of the cluster, or the next word) and
// those whose value may only be attached; the rest of a cluster after either is
// that value, not more options.
var gitShortOptions = map[string]struct{ value, attached string }{
	"commit": {value: "mFCct", attached: "Su"},
	"am":     {value: "Cp", attached: "S"},
}

func gitSubcommandReason(sub string, rest []string) string {
	if sub == "config" {
		return "git config changes repository configuration, which can make later git commands run any program"
	}
	hookSkip := "git " + sub + " without its hooks skips the repository's checks (the pre-commit secret scan among them)"
	short, skipsWithN := gitShortOptions[sub]
	for j := 0; j < len(rest); j++ {
		a := rest[j]
		if a == "--" {
			break
		}
		if name, ok := strings.CutPrefix(a, "--"); ok {
			name, _, _ = strings.Cut(name, "=")
			if name == "output" {
				return "git --output writes to a file of the caller's choosing, .git/config included"
			}
			// Git accepts any unambiguous abbreviation (--no-veri).
			if gitHookSubcommands[sub] && len(name) >= len("no-v") && strings.HasPrefix("no-verify", name) {
				return hookSkip
			}
			continue
		}
		if len(a) < 2 || a[0] != '-' || !skipsWithN {
			continue
		}
		for k := 1; k < len(a); k++ {
			switch ch := a[k]; {
			case ch == 'n':
				return hookSkip
			case strings.IndexByte(short.value, ch) >= 0:
				if k == len(a)-1 {
					j++ // the value is the next word
				}
				k = len(a)
			case strings.IndexByte(short.attached, ch) >= 0:
				k = len(a)
			}
		}
	}
	return ""
}
