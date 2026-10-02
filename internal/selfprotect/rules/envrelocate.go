package rules

import (
	"path"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Protection covers the protected locations the hook process sees. A command
// that changes the environment a regeneration runs with can move a generator
// input to a location the hook does not protect: the org-config variable
// (canon.ProtectedEnvVars) names the overlay directly, and the home variables
// (canon.HomeEnvVars) name the directory the default overlay lives below.

// relocatesProtected reports whether the shell line command, parsed as scs,
// relocates a protected location through the environment: a command sets a
// canon.ProtectedEnvVars variable (`QSDEV_ORG_CONFIG=/tmp/x qsdev claude
// update`, `export QSDEV_ORG_CONFIG=...`), or the line may run the CLI and
// sets or clears a canon.HomeEnvVars variable (`HOME=/tmp/e qsdev init
// --update`, `export HOME=/tmp/e; qsdev ...`, `env -u HOME qsdev ...`).
// Scripts the line runs through a shell (`sh -c`, eval) are followed.
func relocatesProtected(command string, scs []scannedCommand) bool {
	cmds := make([]cmdscan.Command, len(scs))
	for i, sc := range scs {
		cmds[i] = sc.Command
	}
	org := canon.ProtectedEnvVars()
	if anyCommandIn(cmds, 0, assignsEnv(org), mentionsEnvName(org)) {
		return true
	}
	home := canon.HomeEnvVars()
	changesHome := func(c cmdscan.Command) bool { return assignsEnv(home)(c) || clearsEnv(home)(c) }
	return mayRunCLI(command, cmds) && anyCommandIn(cmds, 0, changesHome, mentionsEnvName(home))
}

// unparsedRelocatesHome is relocatesProtected for a line that cannot be
// parsed: it fails closed when the line names the CLI and a home variable.
func unparsedRelocatesHome(command string) bool {
	return cmdscan.InvokesProgram(command, branding.Get().AppName) && mentionsEnvName(canon.HomeEnvVars())(command)
}

// mayRunCLI reports whether the line may run the CLI: a word names it (see
// cmdscan.InvokesProgram, which also sees it inside `sh -c` and eval
// scripts), or a command word is built from an expansion and so may be it.
func mayRunCLI(command string, cmds []cmdscan.Command) bool {
	if cmdscan.InvokesProgram(command, branding.Get().AppName) {
		return true
	}
	expandedName := func(c cmdscan.Command) bool { return c.NameHasExpansion }
	return anyCommandIn(cmds, 0, expandedName, func(string) bool { return false })
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

// assignsEnv returns a check that a command sets one of vars: as a prefix or
// bare assignment, or as a NAME=value word of a command such as export,
// declare, env or sudo.
func assignsEnv(vars []string) func(cmdscan.Command) bool {
	return func(c cmdscan.Command) bool {
		if slices.ContainsFunc(c.Assigns, func(name string) bool { return isEnvName(name, vars) }) {
			return true
		}
		return slices.ContainsFunc(c.Args, func(w string) bool {
			name, _, ok := strings.Cut(w, "=")
			return ok && isEnvName(name, vars)
		})
	}
}

// clearsEnv returns a check that a command removes one of vars from the
// environment: `unset NAME`, or an env that unsets it (-u NAME, -uNAME,
// --unset[=]NAME) or starts from an empty environment (-i, -,
// --ignore-environment), directly or behind a wrapper.
func clearsEnv(vars []string) func(cmdscan.Command) bool {
	return func(c cmdscan.Command) bool {
		words := append([]string{c.Name}, c.Args...)
		for _, i := range cmdscan.CommandWordIndexes(words) {
			switch path.Base(words[i]) {
			case "unset":
				if slices.ContainsFunc(words[i+1:], func(w string) bool { return isEnvName(w, vars) }) {
					return true
				}
			case "env":
				if envClears(words[i+1:], vars) {
					return true
				}
			}
		}
		return false
	}
}

// envClears reports whether env's arguments args remove one of vars from the
// environment of the program it runs. Options end at the first word that is
// neither an option nor an assignment.
func envClears(args []string, vars []string) bool {
	for j := 0; j < len(args); j++ {
		a := args[j]
		switch {
		case a == "-" || a == "-i" || a == "--ignore-environment":
			return true
		case a == "--unset" || a == "-u":
			if j+1 < len(args) && isEnvName(args[j+1], vars) {
				return true
			}
			j++
		case strings.HasPrefix(a, "--unset="):
			if isEnvName(strings.TrimPrefix(a, "--unset="), vars) {
				return true
			}
		case strings.HasPrefix(a, "--"):
			// another long option
		case len(a) > 1 && a[0] == '-':
			// A short-option cluster: -i empties the environment, and -u
			// takes the rest of the cluster, or the next word, as a name.
			flags, name, hasU := strings.Cut(a[1:], "u")
			if strings.ContainsRune(flags, 'i') {
				return true
			}
			if hasU {
				if name == "" && j+1 < len(args) {
					j++
					name = args[j]
				}
				if isEnvName(name, vars) {
					return true
				}
			}
		case strings.Contains(a, "="):
			// an assignment for the program
		default:
			return false
		}
	}
	return false
}
