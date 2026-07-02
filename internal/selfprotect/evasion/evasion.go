package evasion

import (
	"regexp"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// Pre-compiled regexes for obfuscation detection.
var (
	reBase64PipeShell = regexp.MustCompile(`base64\s+(-d|--decode).*\|.*\b(bash|sh|zsh|source)\b`)
	rePrintfHexShell  = regexp.MustCompile(`printf\s+['"]\\x.*\|.*\b(bash|sh)\b`)
	reEvalExpansion   = regexp.MustCompile(`\beval\b.*\$`)
)

// Pre-compiled regexes for hardlink detection.
var (
	reLnCommand = regexp.MustCompile(`\bln\b`)
	reLnSymlink = regexp.MustCompile(`\bln\s+(-\w*s\w*\s+|--symbolic\s+)`)
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
		if blocked, reason := checkHardlink(command); blocked {
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
	return false, ""
}

// evalExpandsVariables reports whether the command invokes `eval` on an argument
// that performs a shell expansion — the dangerous, obfuscation-prone case. The
// `eval`+`$` regex is the deny trigger; the command is cleared only when argv
// parsing proves `eval` is not invoked and every command word is a safe reader.
// So a benign `grep 'eval "$("'` (where `eval` is only a search pattern) clears,
// while `eval "$X"`, `sh -c 'eval "$X"'`, and `command eval "$X"` are blocked
// (a wrapper could hide the eval, so it fails closed). On a parse error it fails
// closed to the whole-string regex. cmds/parseErr are the shared parse of
// command (see CheckParsed).
func evalExpandsVariables(command string, cmds []cmdscan.Command, parseErr error) bool {
	if !reEvalExpansion.MatchString(command) {
		return false // trigger absent: no `eval` followed by `$`
	}
	if parseErr != nil {
		return true // triggered and unparseable ⇒ fail closed
	}
	for _, c := range cmds {
		if c.Name == "eval" {
			return true // eval is actually invoked (with `$` present) ⇒ block
		}
		if c.Name != "" && !cmdscan.IsSafeReadVerb(c.Name) {
			return true // a wrapper/unknown word could hide the eval ⇒ fail closed
		}
	}
	return false // every word is a safe reader; the `eval` text is inert
}

// checkHardlink detects hard link creation targeting protected paths. Symlinks
// (ln -s) are permitted because they go through normal path resolution.
func checkHardlink(command string) (bool, string) {
	if !reLnCommand.MatchString(command) {
		return false, ""
	}

	// Allow symlinks: if -s flag is present this is not a hardlink.
	if reLnSymlink.MatchString(command) {
		return false, ""
	}

	// Check whether the command references a protected path.
	if canon.ContainsProtectedPath(command) {
		return true, "hardlink creation targeting protected path"
	}

	return false, ""
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
