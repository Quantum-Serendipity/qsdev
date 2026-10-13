package rules

import (
	"path"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// GitCodeExecutionRuleID identifies the denials of GitCodeExecution.
const GitCodeExecutionRuleID = "GIT-001"

// maxScriptDepth bounds how deep a check follows shell scripts nested in
// `sh -c` or eval (GitCodeExecution, the worktree rewrite, the ancestor
// checks).
const maxScriptDepth = 3

// reGitWord finds git in a command that cannot be parsed (fail closed).
var reGitWord = regexp.MustCompile(`(^|[^A-Za-z0-9_.-])git([^A-Za-z0-9_.-]|$)`)

// GitCodeExecution reports why a shell command runs git in a way that can run a
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
//   - --output, which writes git's output to any file, .git/config included;
//   - --template (or GIT_TEMPLATE_DIR) on init or clone, which copies hooks
//     into the new repository, and clone's -c/--config, which sets its
//     configuration (core.hooksPath) before the clone's own checkout runs
//     hooks.
//
// It also guards the commit gate itself (canon.CommitGateEnvVars,
// canon.CommitGateManagers): a commit-gate variable set for a git subcommand
// that runs hooks (`SKIP=gitleaks git commit`), and a hook manager's
// uninstall, or an install that points the hook at another configuration or
// lets it pass without one. A manual `SKIP=x pre-commit run` is not the gate
// and stays allowed.
//
// It judges every shell tool (cmdscan.IsShellTool). A PowerShell line is
// judged over the commands its tokenizer finds, whose names already drop a
// directory and .exe (`C:\Tools\git.exe`), and which include the commands of
// a quoted `$(...)` subexpression and of an assignment's value (`$x = git
// commit --no-verify`); the tokenizer cannot fail, so there is no
// unparseable fallback for it. A PowerShell session keeps the variables it
// sets for the commands it runs later, so a line that writes a commit-gate
// variable is denied whether or not it runs git.
func GitCodeExecution(ctx *EvalContext) (string, bool) {
	if !cmdscan.IsShellTool(ctx.ToolName) || ctx.Command == "" {
		return "", false
	}
	if isPowerShell(ctx) {
		if psCommitGateWrites.inLine(ctx) {
			return "the PowerShell session would set or clear " + strings.Join(commitGateEnvVars, " or ") +
				", which skips or disables the pre-commit checks of every git command it runs later", true
		}
		ps := ctx.powerShell().cmds
		cmds := make([]cmdscan.Command, len(ps))
		for i, c := range ps {
			cmds[i] = cmdscan.Command{Name: c.Name, Args: c.Args}
		}
		return gitCodeExecution(cmds, 0, gitLineEnv{})
	}
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		if reGitWord.MatchString(looseText(ctx.Command)) {
			return "the command runs git but cannot be parsed, so its options cannot be checked", true
		}
		return "", false
	}
	return gitCodeExecution(cmds, 0, gitLineEnv{})
}

// gitLineEnv names the variables the commands of a line, or of the shell
// that runs a script, set for the git commands they run: a git variable
// (isGitEnvVar) in config and a commit-gate variable in gate, or "".
type gitLineEnv struct {
	config, gate string
}

// with returns e updated with the variables cmds set.
func (e gitLineEnv) with(cmds []cmdscan.Command) gitLineEnv {
	for _, c := range cmds {
		if v := gitEnvAssignment(c); v != "" {
			e.config = v
		}
		if v := envAssignment(c, isCommitGateEnvVar); v != "" {
			e.gate = v
		}
	}
	return e
}

func gitCodeExecution(cmds []cmdscan.Command, depth int, env gitLineEnv) (string, bool) {
	env = env.with(cmds)
	for _, c := range cmds {
		words := append([]string{c.Name}, c.Args...)
		if script, ok := cmdscan.ShellScript(words); ok {
			if depth >= maxScriptDepth {
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
			// The script inherits what its shell's line sets.
			if reason, bad := gitCodeExecution(sub, depth+1, env); bad {
				return reason, true
			}
		}
		if reason := commitGateManagerReason(words); reason != "" {
			return reason, true
		}
		args, ok := gitInvocation(words)
		if !ok {
			continue
		}
		if env.config != "" {
			return env.config + " is set for a git command: it can make git run any program", true
		}
		if reason := gitArgsReason(args); reason != "" {
			return reason, true
		}
		if sub, _, _ := gitSubcommand(args); env.gate != "" && gitHookSubcommands[sub] {
			return env.gate + " is set for git " + sub + ": it skips or disables the pre-commit checks", true
		}
	}
	return "", false
}

// isGitEnvVar reports whether setting the variable can make git run another
// program: configuration from the environment (GIT_CONFIG_PARAMETERS,
// GIT_CONFIG_COUNT/KEY_n/VALUE_n, GIT_CONFIG_GLOBAL, ...), the directory git
// runs its programs from, the external diff program, and the template
// directory init and clone copy hooks from.
func isGitEnvVar(name string) bool {
	return strings.HasPrefix(name, "GIT_CONFIG") || name == "GIT_EXEC_PATH" || name == "GIT_EXTERNAL_DIFF" ||
		name == "GIT_TEMPLATE_DIR"
}

// commitGateEnvVars are canon.CommitGateEnvVars.
var commitGateEnvVars = canon.CommitGateEnvVars()

// psCommitGateWrites finds a PowerShell write to one of commitGateEnvVars.
var psCommitGateWrites = newPSEnvWrites(commitGateEnvVars)

// envNamesFoldCase reports whether the programs the hook's commands start
// look up environment names case-insensitively, as on Windows, where a
// `skip=x` set in Git Bash reaches pre-commit as SKIP. Elsewhere pre-commit
// reads only SKIP.
var envNamesFoldCase = runtime.GOOS == "windows"

// isCommitGateEnvVar reports whether name is a commit-gate variable, ignoring
// case where environment names do (envNamesFoldCase).
func isCommitGateEnvVar(name string) bool {
	if envNamesFoldCase {
		return isEnvName(name, commitGateEnvVars)
	}
	return slices.Contains(commitGateEnvVars, name)
}

// commitGateManagers are canon.CommitGateManagers.
var commitGateManagers = canon.CommitGateManagers()

// commitGateManagerValueOptions are the hook managers' options before the
// subcommand that take their value as the next word (prek's global -c/--config
// and -C/--cd among them).
var commitGateManagerValueOptions = map[string]bool{
	"-c": true, "--config": true, "-C": true, "--cd": true, "--color": true, "--log-file": true,
}

// commitGateManagerReason returns why words run a hook manager that installs
// the commit gate (canon.CommitGateManagers) in a way that removes it, or
// "": uninstall, or install with another configuration (-c, --config, also
// before the subcommand) or with --allow-missing-config, whose hook passes
// when the configuration is missing.
func commitGateManagerReason(words []string) string {
	manager := ""
	args, ok := programInvocation(words, func(name string) bool {
		manager = strings.ToLower(cmdscan.ProgramName(name))
		return slices.Contains(commitGateManagers, manager)
	})
	if !ok {
		return ""
	}
	switch commitGateManagerSubcommand(args) {
	case "uninstall":
		return manager + " uninstall removes the commit gate's hooks, so commits skip the repository's checks (the pre-commit secret scan among them)"
	case "install":
		if slices.ContainsFunc(args[:argsEnd(args)], isCommitGateConfigOption) {
			return manager + " install with another or a missing configuration installs a hook that skips the repository's checks"
		}
	}
	return ""
}

// commitGateManagerSubcommand returns the hook manager subcommand args run,
// or "".
func commitGateManagerSubcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case commitGateManagerValueOptions[a]:
			i++
		case !isFlag(a):
			return a
		}
	}
	return ""
}

// argsEnd returns the index of the `--` that ends args' options, or len(args).
func argsEnd(args []string) int {
	if i := slices.Index(args, "--"); i >= 0 {
		return i
	}
	return len(args)
}

// isCommitGateConfigOption reports whether a is an install option that
// chooses the configuration the hook runs or lets it run without one: -c in
// a short-option cluster (`-fc x`, `-c/tmp/x`), or an abbreviation argparse
// accepts of --config or --allow-missing-config.
func isCommitGateConfigOption(a string) bool {
	if name, ok := strings.CutPrefix(a, "--"); ok {
		name, _, _ = strings.Cut(name, "=")
		return len(name) >= len("con") && strings.HasPrefix("config", name) ||
			name != "" && strings.HasPrefix("allow-missing-config", name)
	}
	if len(a) < 2 || a[0] != '-' {
		return false
	}
	for _, ch := range a[1:] {
		switch ch {
		case 'c':
			return true
		case 't', 'C': // the rest of the cluster is the option's value
			return false
		}
	}
	return false
}

// gitEnvAssignment returns the name of a git variable that c sets: as a prefix
// or bare assignment, through export/declare, or as an env operand.
func gitEnvAssignment(c cmdscan.Command) string {
	return envAssignment(c, isGitEnvVar)
}

// envAssignment returns the name of a variable accepted by match that c sets:
// as a prefix or bare assignment, through export/declare, or as an env operand.
func envAssignment(c cmdscan.Command, match func(string) bool) string {
	for _, name := range c.Assigns {
		if match(name) {
			return name
		}
	}
	switch path.Base(c.Name) {
	case "export", "declare", "typeset", "local", "readonly", "env":
		for _, a := range c.Args {
			if name, _, _ := strings.Cut(a, "="); match(name) {
				return name
			}
		}
	}
	return ""
}

// gitInvocation returns the arguments of the git program the words run.
func gitInvocation(words []string) ([]string, bool) {
	return programInvocation(words, func(name string) bool { return path.Base(name) == "git" })
}

// programInvocation returns the arguments of the program the words run when
// isProgram accepts its command word, directly or through a wrapper.
func programInvocation(words []string, isProgram func(string) bool) ([]string, bool) {
	for _, i := range cmdscan.CommandWordIndexes(words) {
		if isProgram(words[i]) {
			return words[i+1:], true
		}
	}
	return nil, false
}

// gitGlobalValueOptions are git's global options that take their value as the
// next word.
var gitGlobalValueOptions = map[string]bool{
	"-C": true, "--git-dir": true, "--work-tree": true, "--namespace": true, "--attr-source": true,
}

// gitArgsReason checks git's global options and then its subcommand.
func gitArgsReason(args []string) string {
	sub, rest, reason := gitSubcommand(args)
	if reason != "" || sub == "" {
		return reason
	}
	return gitSubcommandReason(sub, rest)
}

// gitSubcommand returns the subcommand git's arguments args run and the
// arguments after it, or, when a global option before it can run another
// program, why.
func gitSubcommand(args []string) (sub string, rest []string, reason string) {
	g := splitGitArgs(args)
	if g.reason != "" {
		return "", nil, g.reason
	}
	return g.sub, g.rest, ""
}

// gitArgs is a git invocation's arguments split at its subcommand.
type gitArgs struct {
	// chdirs are the directories the -C options change into, in order.
	chdirs []string
	// pinned reports a global option that chooses the repository or work
	// tree (--git-dir, --work-tree, --bare).
	pinned bool
	// reason is why a global option can run another program, or "".
	reason string
	// sub is the subcommand ("" when there is none) and rest the arguments
	// after it.
	sub  string
	rest []string
}

// splitGitArgs walks git's global options up to the subcommand.
func splitGitArgs(args []string) gitArgs {
	var g gitArgs
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, _, hasValue := strings.Cut(a, "=")
		switch {
		case a == "-c", strings.HasPrefix(a, "--config-env"):
			if g.reason == "" {
				g.reason = "git " + a + " sets configuration for one command, which can run any program (alias.<name>=!cmd, core.pager, core.fsmonitor, core.hooksPath)"
			}
			if !hasValue {
				i++
			}
		case strings.HasPrefix(a, "--exec-path="):
			if g.reason == "" {
				g.reason = "git --exec-path= chooses the directory git runs its programs from"
			}
		case a == "-C":
			if i+1 < len(args) {
				g.chdirs = append(g.chdirs, args[i+1])
			}
			i++
		case name == "--git-dir", name == "--work-tree", a == "--bare":
			g.pinned = true
			if a != "--bare" && !hasValue {
				i++
			}
		case gitGlobalValueOptions[a]:
			i++
		case !isFlag(a):
			g.sub, g.rest = a, args[i+1:]
			return g
		}
	}
	return g
}

// gitTemplateSubcommands copy hooks from a template directory into the
// repository they create.
var gitTemplateSubcommands = map[string]bool{"init": true, "clone": true}

// gitHookSubcommands run repository hooks that --no-verify skips.
var gitHookSubcommands = map[string]bool{
	"commit": true, "merge": true, "pull": true, "push": true, "am": true, "rebase": true,
}

// gitShortOptions lists, for the subcommands with a denied short option
// (deny: commit's and am's -n skips hooks, clone's -c sets configuration in
// the new repository), the short options that take a value (the rest of the
// cluster, or the next word) and those whose value may only be attached; the
// rest of a cluster after either is that value, not more options.
var gitShortOptions = map[string]struct {
	deny            byte
	value, attached string
}{
	"commit": {deny: 'n', value: "mFCct", attached: "Su"},
	"am":     {deny: 'n', value: "Cp", attached: "S"},
	"clone":  {deny: 'c', value: "bouj"},
}

func gitSubcommandReason(sub string, rest []string) string {
	if sub == "config" {
		return "git config changes repository configuration, which can make later git commands run any program"
	}
	hookSkip := "git " + sub + " without its hooks skips the repository's checks (the pre-commit secret scan among them)"
	cloneConfig := "git clone --config sets configuration in the new repository (core.hooksPath), which can run any program during and after the clone"
	short, hasShort := gitShortOptions[sub]
	shortDeny := hookSkip
	if sub == "clone" {
		shortDeny = cloneConfig
	}
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
			// Git accepts any unambiguous abbreviation (--templ).
			if gitTemplateSubcommands[sub] && len(name) >= len("te") && strings.HasPrefix("template", name) {
				return "git " + sub + " --template copies hooks from a directory of the caller's choosing into the new repository"
			}
			// Git accepts any unambiguous abbreviation (--no-veri).
			if gitHookSubcommands[sub] && len(name) >= len("no-v") && strings.HasPrefix("no-verify", name) {
				return hookSkip
			}
			// --config is clone's only long option starting with c (--c).
			if sub == "clone" && name != "" && strings.HasPrefix("config", name) {
				return cloneConfig
			}
			continue
		}
		if len(a) < 2 || a[0] != '-' || !hasShort {
			continue
		}
		for k := 1; k < len(a); k++ {
			switch ch := a[k]; {
			case ch == short.deny:
				return shortDeny
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
