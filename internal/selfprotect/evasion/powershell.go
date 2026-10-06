package evasion

import (
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// psEvalVerbs run a string as PowerShell code.
var psEvalVerbs = map[string]bool{"iex": true, "invoke-expression": true}

// psAliasVerbs define an alias, which can give an eval verb another name.
var psAliasVerbs = map[string]bool{"set-alias": true, "sal": true, "new-alias": true, "nal": true}

// psHosts are the PowerShell programs that take an -EncodedCommand.
var psHosts = map[string]bool{"pwsh": true, "powershell": true}

// psStartVerbs start a process, which -Verb can elevate.
var psStartVerbs = map[string]bool{"start-process": true, "saps": true, "start": true}

// psScriptBlockFromString are the spellings, in the loose text with all
// blanks removed, of the .NET calls that turn a string into code:
// [scriptblock]::Create under any namespace
// ([System.Management.Automation.ScriptBlock]::Create), and the
// $ExecutionContext.InvokeCommand methods.
var psScriptBlockFromString = []string{"scriptblock]::create", ".newscriptblock(", ".invokescript("}

// maxPSNesting bounds how deep host command lines (`pwsh -c 'pwsh -c ...'`)
// are re-scanned; a deeper nesting is blocked (fail closed).
const maxPSNesting = 8

// checkPowerShellObfuscation detects the PowerShell forms whose code no check
// can see: a string run as code (iex, Invoke-Expression, an alias for either,
// a script block built from a string), a command line passed Base64-encoded
// to pwsh or powershell, and a process started through -Verb (RunAs elevates
// it out of the session). A command line handed to pwsh or powershell is
// checked the same way. Like the Bash checks, each blocks whatever the code
// is.
func checkPowerShellObfuscation(command string) (bool, string) {
	return checkPowerShellLine(command, 0)
}

func checkPowerShellLine(command string, depth int) (bool, string) {
	if depth > maxPSNesting {
		return true, "PowerShell command lines nested too deeply to check"
	}
	_, loose := cmdscan.PowerShellText(command)
	compact := strings.Join(strings.Fields(loose), "")
	for _, s := range psScriptBlockFromString {
		if strings.Contains(compact, s) {
			return true, "PowerShell script block built from a string"
		}
	}
	for _, c := range cmdscan.ParsePowerShell(command) {
		if blocked, reason := checkPowerShellCommand(c); blocked {
			return true, reason
		}
		if inner, ok := psHostCommandLine(c); ok {
			if blocked, reason := checkPowerShellLine(inner, depth+1); blocked {
				return true, reason
			}
		}
	}
	// The loose form joins a computed command word (`&('i'+'ex')`) back
	// into one, so an eval verb it names is a command of its own here.
	for _, c := range cmdscan.ParsePowerShell(loose) {
		if psEvalVerbs[c.Name] || psAliasesEval(c) {
			return true, "PowerShell Invoke-Expression of a string"
		}
	}
	return false, ""
}

// checkPowerShellCommand checks one command for an eval verb, an alias for
// one, an encoded command, or a -Verb start.
func checkPowerShellCommand(c cmdscan.PSCommand) (bool, string) {
	switch {
	case psEvalVerbs[c.Name] || psAliasesEval(c):
		return true, "PowerShell Invoke-Expression of a string"
	case psRunsEncoded(c):
		return true, "PowerShell encoded command"
	case psStartVerbs[c.Name] && slices.ContainsFunc(c.Args, func(a string) bool { return cmdscan.IsPSParamPrefix(a, "verb") }):
		return true, "PowerShell Start-Process with -Verb"
	}
	return false, ""
}

// psAliasesEval reports whether c defines an alias whose value is an eval
// verb (`sal x iex`, `Set-Alias -Value:Invoke-Expression`).
func psAliasesEval(c cmdscan.PSCommand) bool {
	return psAliasVerbs[c.Name] && slices.ContainsFunc(c.Args, func(a string) bool {
		if _, value, ok := strings.Cut(a, ":"); ok && strings.HasPrefix(a, "-") {
			a = value
		}
		return psEvalVerbs[cmdscan.PSCommandName(a)]
	})
}

// psHostWords returns c's words from pwsh or powershell on, run directly or
// as an argument of another program (`cmd /c pwsh ...`, `Start-Process pwsh
// -ArgumentList '-enc ...'`), with each argument split into its own words;
// nil when c runs no host.
func psHostWords(c cmdscan.PSCommand) []string {
	words := []string{c.Name}
	for _, a := range c.Args {
		words = append(words, strings.FieldsFunc(a, func(r rune) bool { return r == ' ' || r == '\t' || r == ',' })...)
	}
	for i, w := range words {
		if psHosts[cmdscan.PSCommandName(w)] {
			return words[i:]
		}
	}
	return nil
}

// psRunsEncoded reports whether c runs pwsh or powershell with an
// -EncodedCommand parameter: any prefix PowerShell accepts for it, or its -ec
// alias.
func psRunsEncoded(c cmdscan.PSCommand) bool {
	return slices.ContainsFunc(psHostWords(c), func(w string) bool {
		return cmdscan.IsPSParamPrefix(w, "encodedcommand") || cmdscan.IsPSParamPrefix(w, "ec")
	})
}

// psHostCommandLine returns the command line c hands to pwsh or powershell:
// the words after its -Command parameter (any prefix, with any `:value`), or
// after the host
// itself when there is none (powershell runs positional words as a command).
func psHostCommandLine(c cmdscan.PSCommand) (string, bool) {
	words := psHostWords(c)
	if len(words) < 2 {
		return "", false
	}
	rest := words[1:]
	if i := slices.IndexFunc(rest, func(w string) bool { return cmdscan.IsPSParamPrefix(w, "command") }); i >= 0 {
		_, value, _ := strings.Cut(rest[i], ":") // `-Command:iex`
		rest = append([]string{value}, rest[i+1:]...)
	}
	return strings.Join(rest, " "), true
}
