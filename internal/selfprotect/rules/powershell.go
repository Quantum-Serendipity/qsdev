package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// This file holds the PowerShell dialect of the shared shell predicates. The
// POSIX analysis misreads PowerShell: `\` is an escape to it, so
// `.claude\settings.json` names nothing protected, and an unknown cmdlet is a
// mutation, so `Get-ChildItem .claude` denies. A PowerShell line is judged
// coarsely instead, and fails closed: it mutates a protected path when it
// names one and any of its commands is not a read (cmdscan.PSCommand.IsRead).
// It names one when its text does (case-insensitively, also once quotes and
// concatenation are removed), when the POSIX mention scan finds one, or when
// a word of its commands reaches one through the same resolver the POSIX
// rules use (globs, symlinks, the Set-Location/Push-Location directory).

// psLine is a PowerShell command line tokenized once per evaluation.
type psLine struct {
	cmds        []cmdscan.PSCommand
	norm, loose string // see cmdscan.PowerShellText
	// scanned is cmds as the POSIX path analysis sees them (see
	// psScanned), built on first use.
	scanned     []scannedCommand
	scannedDone bool
}

// psPOSIXVerbs maps the PowerShell cmdlets and aliases the POSIX path
// analysis models to their POSIX verb: the location cmdlets, which dirState
// tracks, and the delete, move and rename cmdlets, which remove their
// operands as a whole (replacesProtectedAncestor).
var psPOSIXVerbs = map[string]string{
	"set-location": "cd", "sl": "cd", "cd": "cd", "chdir": "cd",
	"push-location": "pushd", "pushd": "pushd",
	"pop-location": "popd", "popd": "popd",
	"remove-item": "rm", "ri": "rm", "rm": "rm", "del": "rm", "erase": "rm", "rd": "rm", "rmdir": "rm",
	"move-item": "mv", "mi": "mv", "move": "mv", "mv": "mv",
	"rename-item": "mv", "rni": "mv", "ren": "mv",
}

// psHomeVariables are the PowerShell variables holding the home directory,
// lower-cased, which psPathWord renders as ~.
var psHomeVariables = []string{"$home", "${home}", "$env:home", "${env:home}", "$env:userprofile", "${env:userprofile}"}

// isPowerShell reports whether ctx's command is written in PowerShell.
func isPowerShell(ctx *EvalContext) bool {
	return cmdscan.ToolDialect(ctx.ToolName) == cmdscan.PowerShell
}

// powerShell returns ctx.Command tokenized as PowerShell, memoized on ctx.
func (ctx *EvalContext) powerShell() *psLine {
	if ctx.ps == nil {
		norm, loose := cmdscan.PowerShellText(ctx.Command)
		ctx.ps = &psLine{cmds: cmdscan.ParsePowerShell(ctx.Command), norm: norm, loose: loose}
	}
	return ctx.ps
}

// psScanned returns the commands of a PowerShell line as the POSIX path
// analysis sees them, memoized on ctx: path words with `\` mapped to `/` and
// a home variable rendered as ~ (psPathWord), the verbs it models renamed
// (psPOSIXVerbs), and each command annotated with the directory it runs in,
// tracked through Set-Location, Push-Location and Pop-Location by the same
// dirState the POSIX scan uses.
func (ctx *EvalContext) psScanned() []scannedCommand {
	line := ctx.powerShell()
	if line.scannedDone {
		return line.scanned
	}
	line.scannedDone = true
	st := dirState{cwd: ctx.CWD, inProtected: ctx.CWD != "" && isProtectedDir(ctx.CWD)}
	for _, c := range line.cmds {
		cmd := cmdscan.Command{Name: c.Name, WriteRedirects: make([]string, len(c.WriteRedirects))}
		if verb, ok := psPOSIXVerbs[c.Name]; ok {
			cmd.Name = verb
		}
		for _, a := range c.Args {
			w, expanded := psPathWord(a)
			cmd.Args = append(cmd.Args, w)
			if expanded {
				cmd.HasExpansion = true
				cmd.ExpandedArgs = append(cmd.ExpandedArgs, w)
			}
		}
		for i, r := range c.WriteRedirects {
			cmd.WriteRedirects[i], _ = psPathWord(r)
		}
		sc := scannedCommand{Command: cmd, cwd: st.cwd, inProtectedDir: st.inProtected,
			cwdUnknown: st.unknown, cwdHint: st.hint, fs: &ctx.fs}
		line.scanned = append(line.scanned, sc)
		st.apply(sc)
	}
	return line.scanned
}

// psPathWord returns a PowerShell word as a path the POSIX analysis reads:
// `\` mapped to `/`, and a leading home variable (psHomeVariables) rendered
// as ~. expanded reports that some other variable remains, whose value is
// unknown.
func psPathWord(word string) (path string, expanded bool) {
	path = strings.ReplaceAll(word, `\`, "/")
	lower := strings.ToLower(path)
	for _, v := range psHomeVariables {
		if rest, ok := strings.CutPrefix(lower, v); ok && (rest == "" || rest[0] == '/') {
			path = "~" + path[len(v):]
			break
		}
	}
	return path, strings.Contains(path, "$")
}

// psMentionsProtected reports whether a PowerShell line names a protected
// path: in its normalised text or its loose form (case-insensitively), as
// the POSIX mention scan reads the line (which joins `$()`-split words), or
// in a word of its commands resolved like a POSIX word (globs, symlinks, the
// directory it runs in, which may itself be protected). Each test can only
// add a mention, so the union fails closed.
func psMentionsProtected(ctx *EvalContext) bool {
	line := ctx.powerShell()
	if canon.ContainsProtectedPathFold(line.norm) || canon.ContainsProtectedPathFold(line.loose) ||
		(ctx.CWD != "" && isProtectedDir(ctx.CWD)) || posixMentionsProtected(ctx) {
		return true
	}
	// PowerShell paths are case-insensitive: a word is also tried
	// lower-cased, so `.CLA?DE` globs like `.cla?de` on every platform.
	refers := func(sc scannedCommand, words []string, ref func(scannedCommand, string) bool) bool {
		return slices.ContainsFunc(words, func(w string) bool {
			return ref(sc, w) || ref(sc, strings.ToLower(w))
		})
	}
	return slices.ContainsFunc(ctx.psScanned(), func(sc scannedCommand) bool {
		return sc.inProtectedDir || refers(sc, sc.Args, argRefersProtected) ||
			refers(sc, sc.WriteRedirects, refersProtected)
	})
}

// psMutates reports whether any command of a PowerShell line may write: one
// that is not a read verb, writes a redirect, or runs a subexpression.
func psMutates(ctx *EvalContext) bool {
	return slices.ContainsFunc(ctx.powerShell().cmds, func(c cmdscan.PSCommand) bool { return !c.IsRead() })
}

// psDeleteVerbs are the cmdlets and aliases that delete their operands.
var psDeleteVerbs = map[string]bool{
	"remove-item": true, "ri": true, "rm": true, "del": true, "erase": true, "rd": true, "rmdir": true,
}

// psDeletes reports whether a PowerShell line runs a delete verb.
func psDeletes(ctx *EvalContext) bool {
	return slices.ContainsFunc(ctx.powerShell().cmds, func(c cmdscan.PSCommand) bool { return psDeleteVerbs[c.Name] })
}

// psKillVerbs are the cmdlets, aliases and programs that end processes.
var psKillVerbs = map[string]bool{"stop-process": true, "spps": true, "kill": true, "taskkill": true}

// psKillsProtected reports whether a PowerShell line ends a security process:
// it runs a kill verb (or a process object's .Kill() method) and either names
// the CLI, claude or gdev as a whole word anywhere in the line (so
// `Get-Process qsdev | Stop-Process` and `taskkill /IM qsdev.exe` count), or
// has a word that is computed or a glob, which may stand for one (fail
// closed). `Stop-Process -Id 1234` stays allowed.
func psKillsProtected(ctx *EvalContext) bool {
	line := ctx.powerShell()
	kills := strings.Contains(line.norm, ".kill(") ||
		slices.ContainsFunc(line.cmds, func(c cmdscan.PSCommand) bool { return psKillVerbs[c.Name] })
	if !kills {
		return false
	}
	// The loose form joins a name built by concatenation
	// (`('qs'+'dev')`) back into one word.
	return slices.ContainsFunc(line.cmds, psNamesKillTarget) ||
		slices.ContainsFunc(cmdscan.ParsePowerShell(line.loose), psNamesKillTarget)
}

// psNamesKillTarget reports whether a command of a line that kills names a
// security process: a word that is the CLI, claude or gdev, or that is
// computed or a glob and so may be one. The value of an -Id or -PID
// parameter is a process ID, which cannot name one, so `Stop-Process -Id
// $p.Id` stays allowed.
func psNamesKillTarget(c cmdscan.PSCommand) bool {
	if psKillTargetWord(c.Name) {
		return true
	}
	for i := 0; i < len(c.Args); i++ {
		a := c.Args[i]
		if name, ok := cmdscan.PSParamName(a); ok && (name == "id" || name == "pid") {
			if !strings.Contains(a, ":") {
				i++ // the value is the next word
			}
			continue
		}
		if psKillTargetWord(a) {
			return true
		}
	}
	return false
}

// psKillTargetWord reports whether word is computed or a glob, or names the
// CLI, claude or gdev; `-Name:qsdev` and `qsdev,claude` name each value.
func psKillTargetWord(word string) bool {
	if strings.ContainsAny(word, "$*?[") {
		return true
	}
	app := strings.ToLower(branding.Get().AppName)
	for _, v := range strings.FieldsFunc(word, func(r rune) bool { return r == ',' || r == ':' || r == '=' }) {
		switch cmdscan.PSCommandName(v) {
		case app, "claude", "gdev":
			return true
		}
	}
	return false
}

// psMutatesArea reports whether a PowerShell line may write and names the
// area (or an ancestor) in its text or in a word of its commands, resolved
// against the directory the command runs in.
func psMutatesArea(ctx *EvalContext, a protectedArea) bool {
	if !psMutates(ctx) {
		return false
	}
	line := ctx.powerShell()
	if a.touches(line.norm) || a.touches(line.loose) {
		return true
	}
	return slices.ContainsFunc(ctx.psScanned(), func(sc scannedCommand) bool {
		return slices.ContainsFunc(append(slices.Clip(sc.Args), sc.WriteRedirects...), func(w string) bool {
			if isFlag(w) && w != "-" {
				w = flagValue(w)
			}
			return w != "" && wordTouchesArea(sc, w, a)
		})
	})
}

// psMutatesMcpConfig reports whether a PowerShell line names an MCP config
// and may write.
func psMutatesMcpConfig(ctx *EvalContext) bool {
	line := ctx.powerShell()
	return (commandMentionsMcpConfig(line.norm) || commandMentionsMcpConfig(line.loose)) && psMutates(ctx)
}

// psSessionEnv are the variables a PowerShell line must not set or clear: an
// agent marker (canon.AgentEnvMarkers), which the CLI's human gate reads, and
// the variables that change the settings and hooks a Claude Code session
// loads (settingsOverrideEnv). Unlike a Bash call, a PowerShell session may
// keep what it sets for the commands and the programs it starts later.
var psSessionEnv = append(slices.Clone(canon.AgentEnvMarkers), settingsOverrideEnv...)

// rePSEnvWrite finds, in a PowerShell line's loose text (lower-cased, quotes
// removed), a write to one of psSessionEnv: an assignment to $env:NAME (any
// operator but a comparison), a reference to the Env: drive (env:NAME,
// env:\NAME), which only a non-read command (Remove-Item, Set-Item, ...)
// changes, or a [Environment]::SetEnvironmentVariable call.
var rePSEnvWrite, rePSEnvDrive, rePSSetEnvVar = func() (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp) {
	names := make([]string, len(psSessionEnv))
	for i, n := range psSessionEnv {
		names[i] = regexp.QuoteMeta(strings.ToLower(n))
	}
	alt := `(` + strings.Join(names, "|") + `)\b`
	return regexp.MustCompile(`\$\{?env:` + alt + `\}?\s*[-+*/%?]*=([^=]|$)`),
		regexp.MustCompile(`(^|[^a-z0-9_$:{])env:[/\\]*` + alt),
		regexp.MustCompile(`setenvironmentvariable\W*` + alt)
}()

// psEnvOverride reports why a PowerShell line sets or clears one of
// psSessionEnv, or "".
func psEnvOverride(ctx *EvalContext) string {
	line := ctx.powerShell()
	for _, text := range []string{line.norm, line.loose} {
		if rePSEnvWrite.MatchString(text) || rePSSetEnvVar.MatchString(text) ||
			rePSEnvDrive.MatchString(text) && psMutates(ctx) {
			return "the PowerShell session would set or clear an agent marker or Claude Code settings variable, " +
				"which the commands and programs it runs later inherit"
		}
	}
	return ""
}
