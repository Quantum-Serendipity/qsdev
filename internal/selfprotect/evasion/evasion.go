package evasion

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// Pre-compiled regexes for obfuscation detection.
var (
	reBase64PipeShell = regexp.MustCompile(`base64\s+(-d|--decode).*\|.*\b(bash|sh|zsh|source)\b`)
	rePrintfHexShell  = regexp.MustCompile(`printf\s+['"]\\x.*\|.*\b(bash|sh)\b`)
	// reEvalExpansion is `eval` followed by an expansion character: `$` or a
	// backtick command substitution.
	reEvalExpansion = regexp.MustCompile("\\beval\\b.*[$`]")
)

// Pre-compiled regexes for hardlink detection.
var (
	reLnCommand   = regexp.MustCompile(`\bln\b`)
	reLnSymlink   = regexp.MustCompile(`\bln\s+(-\w*s\w*\s+|--symbolic\s+)`)
	reLinkCommand = regexp.MustCompile(`\blink\b`)
)

// Pre-compiled regexes for file descriptor tricks.
var (
	reDevFD       = regexp.MustCompile(`/dev/fd/`)
	reProcSelfFD  = regexp.MustCompile(`/proc/self/fd/`)
	reProcSubst   = regexp.MustCompile(`[<>]\(`)
	reExecFDRedir = regexp.MustCompile(`exec\s+\d+>`)
)

// Pre-compiled regexes for proc root traversal.
var (
	reProcSelfRoot = regexp.MustCompile(`/proc/self/root/`)
	reProcPIDRoot  = regexp.MustCompile(`/proc/\d+/root/`)
	reProcSelfInfo = regexp.MustCompile(`/proc/self/(environ|cmdline)`)
)

// Check examines a tool call for evasion techniques, parsing the command itself.
// Prefer CheckParsed when the command has already been parsed (the hook path) to
// avoid a redundant shell parse; Check remains for callers with only the raw
// string (e.g. tests). Returns (blocked, category, reason).
func Check(toolName string, command string, filePath string) (bool, string, string) {
	var cmds []cmdscan.Command
	var parseErr error
	if toolName == "Bash" && command != "" {
		cmds, parseErr = cmdscan.Parse(command)
	}
	return CheckParsed(toolName, command, filePath, cmds, parseErr)
}

// CheckParsed is Check with the command already shell-parsed. cmds/parseErr come
// from a single cmdscan.Parse of command (parseErr non-nil ⇒ unparseable, so the
// obfuscation check falls back to its whole-string regex).
func CheckParsed(toolName string, command string, filePath string, cmds []cmdscan.Command, parseErr error) (bool, string, string) {
	if toolName == "Bash" && command != "" {
		if blocked, reason := checkObfuscation(command, cmds, parseErr); blocked {
			return true, "obfuscation", reason
		}
	}

	if toolName == "Bash" && command != "" {
		if blocked, reason := checkHardlink(command, cmds, parseErr); blocked {
			return true, "hardlink", reason
		}
	}

	if blocked, reason := checkFDTricks(toolName, command, filePath); blocked {
		return true, "fdtricks", reason
	}

	if blocked, reason := checkProcRoot(toolName, command, filePath); blocked {
		return true, "procroot", reason
	}

	return false, "", ""
}

// checkObfuscation detects base64-to-shell, printf hex-to-shell, and eval
// expansion patterns that attempt to hide malicious commands.
func checkObfuscation(command string, cmds []cmdscan.Command, parseErr error) (bool, string) {
	if reBase64PipeShell.MatchString(command) {
		return true, "base64 decode piped to shell execution"
	}
	if rePrintfHexShell.MatchString(command) {
		return true, "printf hex escape piped to shell execution"
	}
	if evalExpandsVariables(command, cmds, parseErr) {
		return true, "eval with variable expansion"
	}
	if shellRunsExpandedScript(cmds) {
		return true, "shell -c script built from an expansion"
	}
	return false, ""
}

// shells are the interpreters that run a script string passed with -c.
var shells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true, "mksh": true, "ash": true,
}

// shellRunsExpandedScript reports whether a parsed command runs a shell with
// -c on words that use an expansion — like eval, the code that runs is
// computed at run time (`bash -c "$(echo … | base64 -d)"`), so no check can
// see it. The shell may be named by path (/bin/sh) or run through a wrapper
// (env, sudo, command, nice, ...), so every word of the command is considered,
// and the -c flag may be clustered with others (-ec, -xc). An unparseable
// command yields no cmds; the regex checks cover it.
func shellRunsExpandedScript(cmds []cmdscan.Command) bool {
	for _, c := range cmds {
		if !c.HasExpansion || cmdscan.IsSafeReadCommand(c) {
			continue
		}
		words := append([]string{c.Name}, c.Args...)
		for i, w := range words {
			if !shells[path.Base(w)] {
				continue
			}
			for _, a := range words[i+1:] {
				if len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 'c') {
					return true
				}
			}
		}
	}
	return false
}

// mayRunCode reports whether a parsed command could run something other than a
// proven safe reader: it has a command word that is not one (including a word
// computed by an expansion), or it sets variables that change what later
// commands run. A nameless command that only carries redirects runs nothing.
func mayRunCode(c cmdscan.Command) bool {
	if c.Name == "" && len(c.Assigns) == 0 && !c.HasExpansion {
		return false
	}
	return !cmdscan.IsSafeReadCommand(c)
}

// evalExpandsVariables reports whether the command invokes `eval` on an argument
// that performs a shell expansion — the dangerous, obfuscation-prone case.
//
// When the command parses, the parse decides first: an `eval` whose words use
// any expansion ($VAR, $(...), or a backtick substitution) is blocked. The
// `eval`+expansion-character regex is then a secondary trigger, cleared only
// when argv parsing proves `eval` is not invoked and every command is a safe
// reader. So a benign `grep 'eval "$("'` (where `eval` is only a search
// pattern) clears, while `eval "$X"`, "eval `printf rm` x", `sh -c 'eval
// "$X"'`, and `command eval "$X"` are blocked (a wrapper could hide the eval, so
// it fails closed). On a parse error it fails closed to the whole-string regex.
// cmds/parseErr are the shared parse of command (see CheckParsed).
func evalExpandsVariables(command string, cmds []cmdscan.Command, parseErr error) bool {
	if parseErr != nil {
		return reEvalExpansion.MatchString(command) // unparseable ⇒ fail closed on the text
	}
	for _, c := range cmds {
		if c.Name == "eval" && c.HasExpansion {
			return true // eval of a string computed at run time ⇒ block
		}
	}
	if !reEvalExpansion.MatchString(command) {
		return false // trigger absent: no `eval` followed by `$` or a backtick
	}
	for _, c := range cmds {
		if c.Name == "eval" {
			return true // eval is actually invoked (with an expansion present) ⇒ block
		}
		if mayRunCode(c) {
			return true // a wrapper/unknown/computed word could hide the eval ⇒ fail closed
		}
	}
	return false // every word is a safe reader; the `eval` text is inert
}

// checkHardlink detects hard link creation targeting protected paths. A hard
// link gives a protected file a second, unprotected name: the alias
// canonicalizes to itself, so a later write through it modifies the protected
// inode unseen. Symlinks (ln -s) are permitted because they go through normal
// path resolution. Besides `ln`, hard links are made by `link`, `cp -l`/
// `cp --link`, and `rsync --link-dest`; cp and rsync are otherwise treated as
// benign in-repo copies by the Tier-1 rules, so they are caught here.
func checkHardlink(command string, cmds []cmdscan.Command, parseErr error) (bool, string) {
	if !canon.ContainsProtectedPath(command) {
		return false, ""
	}

	// `ln` without -s makes a hard link; with -s it is an allowed symlink.
	if reLnCommand.MatchString(command) && !reLnSymlink.MatchString(command) {
		return true, "hardlink creation targeting protected path"
	}

	if linkVerbMayRun(command, cmds, parseErr) {
		return true, "hardlink creation (link) targeting protected path"
	}

	// cp/rsync behind a wrapper or in an unparseable command already fail
	// closed in the Tier-1 copy rule; the parsed direct invocation is the gap.
	for _, c := range cmds {
		if createsHardlinks(c) && anyArgProtected(c.Args) {
			return true, "hardlink creation (" + c.Name + ") targeting protected path"
		}
	}

	return false, ""
}

// linkVerbMayRun reports whether the coreutils `link` command may run. The
// `link` word is the trigger; like `ln`, it is cleared only when the command
// parses and every command in it is a proven safe reader. Anything else could
// run the word: directly, through a wrapper (`sudo link`, `sh -c 'link …'`),
// or by reading it as a script (`echo 'link …' | sh`).
func linkVerbMayRun(command string, cmds []cmdscan.Command, parseErr error) bool {
	if !reLinkCommand.MatchString(command) {
		return false
	}
	if parseErr != nil {
		return true
	}
	return slices.ContainsFunc(cmds, mayRunCode)
}

// createsHardlinks reports whether a parsed command creates hard links: `link`,
// `cp` with -l/--link (including a combined short cluster such as -al, and any
// --link abbreviation getopt_long accepts), or `rsync --link-dest`.
func createsHardlinks(c cmdscan.Command) bool {
	switch c.Name {
	case "link":
		return true
	case "cp":
		for _, a := range c.Args {
			if a == "--" {
				break
			}
			if len(a) > 2 && strings.HasPrefix("--link", a) {
				return true
			}
			if len(a) > 1 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], 'l') {
				return true
			}
		}
	case "rsync":
		for _, a := range c.Args {
			if a == "--link-dest" || strings.HasPrefix(a, "--link-dest=") {
				return true
			}
		}
	}
	return false
}

// anyArgProtected reports whether any argument — operand or option value such
// as --link-dest=~/.claude — references a protected path.
func anyArgProtected(args []string) bool {
	for _, a := range args {
		if canon.ContainsProtectedPath(a) {
			return true
		}
	}
	return false
}

// checkFDTricks detects file descriptor and /proc/self/fd tricks that bypass
// normal path-based access controls.
func checkFDTricks(toolName string, command string, filePath string) (bool, string) {
	// Check filePath regardless of tool.
	if filePath != "" {
		if reDevFD.MatchString(filePath) {
			return true, "file descriptor path /dev/fd/ used to bypass path controls"
		}
		if reProcSelfFD.MatchString(filePath) {
			return true, "proc self fd path used to bypass path controls"
		}
	}

	// Check command for Bash tool.
	if toolName == "Bash" && command != "" {
		if reDevFD.MatchString(command) {
			return true, "file descriptor path /dev/fd/ in command"
		}
		if reProcSelfFD.MatchString(command) {
			return true, "proc self fd path in command"
		}
		if reProcSubst.MatchString(command) && canon.ContainsProtectedPath(command) {
			return true, "process substitution targeting protected path"
		}
		if reExecFDRedir.MatchString(command) && canon.ContainsProtectedPath(command) {
			return true, "fd redirection targeting protected path"
		}
	}

	return false, ""
}

// checkProcRoot detects /proc/self/root and /proc/<pid>/root traversals that
// provide an alternative path to any file on the filesystem.
func checkProcRoot(toolName string, command string, filePath string) (bool, string) {
	// Check filePath regardless of tool.
	if filePath != "" {
		if reProcSelfRoot.MatchString(filePath) {
			return true, "proc self root traversal in file path"
		}
		if reProcPIDRoot.MatchString(filePath) {
			return true, "proc pid root traversal in file path"
		}
		if reProcSelfInfo.MatchString(filePath) {
			return true, "access to proc self environ or cmdline via file path"
		}
	}

	// Check command for Bash tool.
	if toolName == "Bash" && command != "" {
		if reProcSelfRoot.MatchString(command) {
			return true, "proc self root traversal in command"
		}
		if reProcPIDRoot.MatchString(command) {
			return true, "proc pid root traversal in command"
		}
		if reProcSelfInfo.MatchString(command) {
			return true, "access to proc self environ or cmdline in command"
		}
	}

	return false, ""
}
