package privilege

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsElevated reports whether the current process has admin privileges.
func IsElevated() bool {
	return !NeedsElevation()
}

// ElevatedExec runs a command with elevation via the detected tool.
func ElevatedExec(ctx context.Context, name string, args ...string) error {
	tool := DetectElevationTool()
	if tool == "" {
		return fmt.Errorf("no elevation tool available (sudo, doas, pkexec)")
	}

	// Special handling for PowerShell Start-Process on Windows.
	if tool == "powershell" {
		c := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", powerShellElevationCommand(name, args))
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	allArgs := make([]string, 0, 1+len(args))
	allArgs = append(allArgs, name)
	allArgs = append(allArgs, args...)
	c := exec.CommandContext(ctx, tool, allArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// powerShellElevationCommand builds the PowerShell script that runs name with
// args elevated through Start-Process and propagates its exit code. Without
// -PassThru and an explicit exit, powershell.exe exits 0 however the elevated
// process ended, and ErrorAction Stop turns a refused UAC prompt into a
// non-zero exit instead of a silent success. Start-Process joins -ArgumentList
// with spaces and adds no quoting, so each element carries its own Windows
// command-line quoting; -ArgumentList is omitted when there are no arguments
// because it rejects an empty value.
func powerShellElevationCommand(name string, args []string) string {
	var b strings.Builder
	b.WriteString("$ErrorActionPreference = 'Stop'; $p = Start-Process -Verb RunAs -Wait -PassThru -FilePath ")
	b.WriteString(quotePowerShellArg(name))
	if len(args) > 0 {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = quotePowerShellArg(windowsCommandLineArg(a))
		}
		b.WriteString(" -ArgumentList ")
		b.WriteString(strings.Join(quoted, ","))
	}
	b.WriteString("; exit $p.ExitCode")
	return b.String()
}

// quotePowerShellArg returns s as a PowerShell single-quoted string literal.
func quotePowerShellArg(s string) string {
	return "'" + escapePowerShellArg(s) + "'"
}

// escapePowerShellArg escapes a string for safe use inside a PowerShell
// single-quoted string literal. Single-quoted strings are literal; only a
// quote character needs escaping, by doubling it. The PowerShell tokenizer
// also treats the typographic quotes U+2018, U+2019, U+201A and U+201B as
// single quotes, so they are doubled too; otherwise one of them could close
// the literal and inject a command.
func escapePowerShellArg(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(r)
		if isPowerShellSingleQuote(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isPowerShellSingleQuote(r rune) bool {
	switch r {
	case '\'', '\u2018', '\u2019', '\u201A', '\u201B':
		return true
	default:
		return false
	}
}

// windowsCommandLineArg quotes s as a single argument of a Windows command
// line, so that CommandLineToArgvW (and the MSVC runtime) parse it back
// unchanged. It follows the rules of syscall.EscapeArg, which is only built on
// Windows.
func windowsCommandLineArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s // backslashes are literal unless they precede a quote
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			slashes++
		case '"':
			// Backslashes before a quote must be doubled, plus one more
			// backslash to escape the quote itself.
			b.WriteString(strings.Repeat(`\`, slashes+1))
			slashes = 0
		default:
			slashes = 0
		}
		b.WriteByte(c)
	}
	// Backslashes before the closing quote must be doubled as well.
	b.WriteString(strings.Repeat(`\`, slashes))
	b.WriteByte('"')
	return b.String()
}

// BatchElevatedInstall runs a single elevated package install command
// for multiple packages. pm is the package manager binary (e.g., "apt-get"),
// pmArgs is the subcommand (e.g., ["install", "-y"]), and packages are appended.
func BatchElevatedInstall(ctx context.Context, pm string, pmArgs []string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}
	allArgs := make([]string, 0, len(pmArgs)+len(packages))
	allArgs = append(allArgs, pmArgs...)
	allArgs = append(allArgs, packages...)
	return ElevatedExec(ctx, pm, allArgs...)
}
