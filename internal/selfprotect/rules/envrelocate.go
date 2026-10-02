package rules

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Protection covers the protected locations the hook process sees. A command
// that changes the environment a regeneration runs with can move a generator
// input to a location the hook does not protect: the org-config variable
// (canon.ProtectedEnvVars) names the overlay directly. The CLI reads the
// default overlay below the account's home directory, not below HOME (see
// catalog.OrgConfigPath), so the home variables (canon.HomeEnvVars) no longer
// move it; a line that runs the CLI may still not change them, as defense in
// depth.

// relocatesProtected reports whether the shell line command, parsed as scs,
// relocates a protected location through the environment: a command sets a
// canon.ProtectedEnvVars variable (`QSDEV_ORG_CONFIG=/tmp/x qsdev claude
// update`, `export QSDEV_ORG_CONFIG=...`), or the line may run the CLI and
// sets or clears a canon.HomeEnvVars variable, in any of the forms
// cmdscan.Command.Assigns lists (`HOME=/tmp/e qsdev init --update`, `for HOME
// in /tmp/e; do qsdev ...`, `read HOME <<< /tmp/e; qsdev ...`), or clears it
// or a ProtectedEnvVars variable through env or exec (`env -u HOME qsdev
// ...`, `exec -c qsdev ...`), or sets a variable whose name it computes
// (`export $X=/tmp/e; qsdev ...`). Scripts the line runs through a shell (`sh
// -c`, eval) are followed.
func relocatesProtected(command string, scs []scannedCommand) bool {
	cmds := make([]cmdscan.Command, len(scs))
	for i, sc := range scs {
		cmds[i] = sc.Command
	}
	org := canon.ProtectedEnvVars()
	if anyCommandIn(cmds, 0, assignsEnv(org), mentionsEnvName(org)) {
		return true
	}
	if !mayRunCLI(command, cmds) {
		return false
	}
	home := canon.HomeEnvVars()
	cleared := slices.Concat(home, org)
	changesEnv := func(c cmdscan.Command) bool {
		return c.AssignsDynamic || assignsEnv(home)(c) || clearsEnv(cleared)(c)
	}
	return anyCommandIn(cmds, 0, changesEnv, mentionsEnvName(home))
}

// unparsedRelocatesHome is relocatesProtected for a line that cannot be
// parsed: it fails closed when the line names the CLI and a home variable.
func unparsedRelocatesHome(command string) bool {
	return cmdscan.InvokesProgram(command, branding.Get().AppName) && mentionsEnvName(canon.HomeEnvVars())(command)
}

// mayRunCLI reports whether the line may run the CLI: its text names it (see
// cmdscan.InvokesProgram, which also sees it inside `sh -c` and eval
// scripts), or a command word of a parsed command, or of a script it runs,
// may be it (see commandMayRunCLI). A script that cannot be parsed may.
func mayRunCLI(command string, cmds []cmdscan.Command) bool {
	app := branding.Get().AppName
	if cmdscan.InvokesProgram(command, app) {
		return true
	}
	mayRun := func(c cmdscan.Command) bool { return commandMayRunCLI(c, app) }
	return anyCommandIn(cmds, 0, mayRun, func(string) bool { return true })
}

// commandMayRunCLI reports whether c may run the program app: its command
// word is built from an expansion, or a word that can name the program (see
// cmdscan.CommandWordIndexes) names app once quotes and escapes are removed
// (`$'qsdev'`, `qs$'d'ev`), after brace expansion (`{qsdev,}`), or as a glob
// that can match it (`qsde?`). Names are compared case-insensitively, as
// Windows and macOS resolve them.
func commandMayRunCLI(c cmdscan.Command, app string) bool {
	if c.NameHasExpansion {
		return true
	}
	words := append([]string{c.Name}, c.Args...)
	for _, i := range cmdscan.CommandWordIndexes(words) {
		variants, ok := expandBraces(words[i])
		if !ok {
			return true
		}
		for _, v := range variants {
			name := strings.ToLower(cmdscan.ProgramName(v))
			if name == strings.ToLower(app) || (hasGlobMeta(name) && shellSegMatch(name, strings.ToLower(app))) {
				return true
			}
		}
	}
	return false
}

// anyCommandIn reports whether pred holds for a command of cmds, or of a
// shell script one of them runs (`sh -c '...'`, eval), followed up to
// maxSettingsScriptDepth. A script that cannot be parsed, or is nested
// deeper, counts when unparsed holds for its text.
func anyCommandIn(cmds []cmdscan.Command, depth int, pred func(cmdscan.Command) bool, unparsed func(string) bool) bool {
	for _, c := range cmds {
		if pred(c) {
			return true
		}
		script, ok := cmdscan.ShellScript(append([]string{c.Name}, c.Args...))
		if !ok {
			continue
		}
		sub, err := cmdscan.Parse(script)
		if err != nil || depth >= maxSettingsScriptDepth {
			if unparsed(script) {
				return true
			}
			continue
		}
		if anyCommandIn(sub, depth+1, pred, unparsed) {
			return true
		}
	}
	return false
}

// isEnvName reports whether name is one of vars. Names are compared
// case-insensitively, as Windows environment names are.
func isEnvName(name string, vars []string) bool {
	return slices.ContainsFunc(vars, func(v string) bool { return strings.EqualFold(v, name) })
}

// mentionsEnvName returns a check that text names one of vars, for text that
// cannot be parsed.
func mentionsEnvName(vars []string) func(string) bool {
	return func(text string) bool {
		upper := strings.ToUpper(text)
		return slices.ContainsFunc(vars, func(v string) bool { return strings.Contains(upper, strings.ToUpper(v)) })
	}
}

// assignsEnv returns a check that a command sets one of vars: in any form
// cmdscan.Command.Assigns lists, as a NAME=value word of a command such as
// env or sudo, or as an argument of a command word built from an expansion,
// which may be read, export or declare (`$R HOME <<< /tmp/e`).
func assignsEnv(vars []string) func(cmdscan.Command) bool {
	return func(c cmdscan.Command) bool {
		if slices.ContainsFunc(c.Assigns, func(name string) bool { return isEnvName(name, vars) }) {
			return true
		}
		return slices.ContainsFunc(c.Args, func(w string) bool {
			name, _, ok := strings.Cut(w, "=")
			return (ok || c.NameHasExpansion) && isEnvName(name, vars)
		})
	}
}

// clearsEnv returns a check that a command runs env or bash's exec builtin,
// directly or behind a wrapper, so that it removes one of vars from the
// environment of the program it runs (see envChanges and execClearsEnv).
// unset is covered by cmdscan.Command.Assigns.
func clearsEnv(vars []string) func(cmdscan.Command) bool {
	return func(c cmdscan.Command) bool {
		words := append([]string{c.Name}, c.Args...)
		for _, i := range cmdscan.CommandWordIndexes(words) {
			switch strings.ToLower(cmdscan.ProgramName(words[i])) {
			case "env":
				if envChanges(words[i+1:], vars) {
					return true
				}
			case "exec":
				if execClearsEnv(words[i+1:]) {
					return true
				}
			}
		}
		return false
	}
}

// execClearsEnv reports whether exec, given args, runs its command with an
// empty environment: -c, bash's form of env -i, also in a cluster (`-cl`,
// `-ca NAME`).
func execClearsEnv(args []string) bool {
	return slices.ContainsFunc(cmdscan.WrapperOptions("exec", args), func(o cmdscan.WrapperOption) bool {
		return o.Is("-c")
	})
}

// envChanges reports whether env's arguments args remove one of vars from
// the environment of the program it runs: -u NAME (in any spelling cmdscan's
// option table accepts, including `-C dir -u NAME` and an abbreviated
// --unset), or -i/-/--ignore-environment, which start from an empty one. A
// -S string is split and read as more arguments, which may also set a
// variable.
func envChanges(args []string, vars []string) bool {
	for _, o := range cmdscan.WrapperOptions("env", args) {
		switch {
		case o.Is("-", "-i", "--ignore-environment"):
			return true
		case o.Is("-u", "--unset") && isEnvName(o.Arg, vars):
			return true
		case o.Is("-S", "--split-string"):
			split := strings.Fields(o.Arg)
			if envChanges(split, vars) || slices.ContainsFunc(split, func(w string) bool {
				name, _, ok := strings.Cut(w, "=")
				return ok && isEnvName(name, vars)
			}) {
				return true
			}
		}
	}
	return false
}

// envFileName returns the entry of canon.EnvSourceFiles that ctx's Write or
// Edit target is named, by the name the tool uses or its resolved one, or "".
func envFileName(ctx *EvalContext) string {
	names := canon.EnvSourceFiles()
	for _, p := range []string{ctx.FilePath, ctx.CanonicalPath} {
		if base := strings.ToLower(filepath.Base(p)); p != "" && slices.Contains(names, base) {
			return base
		}
	}
	return ""
}

// envMentionNoise is what a spelling of a variable name can be split by in
// a shell or Nix file without changing the name it builds: quotes, escapes,
// line continuations, string concatenation and interpolation.
var envMentionNoise = strings.NewReplacer(
	`"`, "", "'", "", `\`, "", "+", "", "$", "", "{", "", "}", "", "(", "", ")", "",
	" ", "", "\t", "", "\n", "", "\r", "",
)

// mentionsProtectedEnv reports whether text names a canon.ProtectedEnvVars
// variable, also when quotes, concatenation or interpolation split the name
// (`"QSDEV_ORG_" + "CONFIG"`, `${"QSDEV_ORG"}_CONFIG`).
func mentionsProtectedEnv(text string) bool {
	return mentionsEnvName(canon.ProtectedEnvVars())(envMentionNoise.Replace(text))
}

// envFileRelocation returns why a tool call changes a variable the CLI's
// environment inherits that relocates a protected location, or "": a Write or
// Edit of an environment source file (canon.EnvSourceFiles: devenv.local.nix,
// a shell startup file, ...) whose content before or after names a
// canon.ProtectedEnvVars variable (`{ env.QSDEV_ORG_CONFIG = "/tmp/x"; }`,
// or removing the line that sets it), or a shell command that rewrites one of
// those files at all, since its content cannot be checked (`printf ... >>
// ~/.bashrc`). It fails closed when the change cannot be reconstructed.
func envFileRelocation(ctx *EvalContext) string {
	if cmdscan.IsShellTool(ctx.ToolName) {
		if name, ok := BashRewritesFile(ctx, canon.EnvSourceFiles()); ok {
			return "shell command changes " + name + ", which sets the environment " + branding.Get().AppName +
				" runs in and is only checked for the Edit and Write tools; make the change with Edit or Write"
		}
		return ""
	}
	if !isWriteOrEdit(ctx.ToolName) {
		return ""
	}
	name := envFileName(ctx)
	if name == "" {
		return ""
	}
	before, after, err := ctx.FileChange()
	if err != nil {
		return "cannot verify the change to " + name + ": " + err.Error()
	}
	if before != after && (mentionsProtectedEnv(before) || mentionsProtectedEnv(after)) {
		return name + " sets the environment " + branding.Get().AppName + " runs in; changing " +
			strings.Join(canon.ProtectedEnvVars(), " or ") + " there would relocate the org overlay"
	}
	return ""
}
