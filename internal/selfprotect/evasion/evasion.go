package evasion

import (
	"regexp"
	"strings"
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

// protectedPathPatterns are path fragments that identify protected configuration.
var protectedPathPatterns = []string{
	`.claude/`,
	`.qsdev/`,
	`.gdev/`,
	`/etc/gdev/`,
	`/etc/claude-code/`,
	`.qsdev/audit/`,
}

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

// Check examines a tool call for evasion techniques.
// Returns (blocked, category, reason) where category identifies the evasion type.
func Check(toolName string, command string, filePath string) (bool, string, string) {
	if toolName == "Bash" && command != "" {
		if blocked, reason := checkObfuscation(command); blocked {
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
func checkObfuscation(command string) (bool, string) {
	if reBase64PipeShell.MatchString(command) {
		return true, "base64 decode piped to shell execution"
	}
	if rePrintfHexShell.MatchString(command) {
		return true, "printf hex escape piped to shell execution"
	}
	if reEvalExpansion.MatchString(command) {
		return true, "eval with variable expansion"
	}
	return false, ""
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
	if containsProtectedPath(command) {
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
		if reProcSubst.MatchString(command) && containsProtectedPath(command) {
			return true, "process substitution targeting protected path"
		}
		if reExecFDRedir.MatchString(command) && containsProtectedPath(command) {
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

// containsProtectedPath reports whether s contains any protected path pattern.
func containsProtectedPath(s string) bool {
	for _, p := range protectedPathPatterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
