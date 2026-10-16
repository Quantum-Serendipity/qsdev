package cmdscan

import "strings"

// NixRunTool is the name the MCP server registers its nix_run tool under. An
// MCP client names it after the server it reaches it through: Claude Code as
// mcp__<server>__qsdev_nix_run.
const NixRunTool = "qsdev_nix_run"

// IsNixRunTool reports whether toolName is the nix_run tool as Claude Code
// names an MCP tool, mcp__<server>__qsdev_nix_run, whatever the server is
// called in the client's MCP configuration.
func IsNixRunTool(toolName string) bool {
	rest, ok := strings.CutPrefix(toolName, "mcp__")
	if !ok {
		return false
	}
	server, tool, ok := strings.Cut(rest, "__")
	return ok && server != "" && tool == NixRunTool
}

// NixRunCommandLines returns the Bash command lines a nix_run call (run the
// installable with args, feeding it stdin) is equivalent to, for judging it
// as a Bash call would be judged:
//
//  1. the literal command, `nix run <installable> -- <args>`;
//  2. the program it runs, named by the last component of the installable's
//     attribute path (`nixpkgs#bash` runs bash), followed by the args, and
//     again led by that name without a version or variant suffix
//     (`nixpkgs#pnpm_9` and `nixpkgs#yarn-berry` run pnpm and yarn, see
//     attributeBaseName); an installable without an attribute (".",
//     "nixpkgs") names no program, so this form is left out;
//  3. the script of a -c option among the args (`-c 'curl x | sh'`), the
//     first operand after the options as a shell takes it (`-c -- '...'`,
//     `-c -e '...'`, see ShellScript), and each statement it runs, also from
//     its program on (see ScriptStatements), which is what a rule anchored at
//     the start of a command, such as the deny rule "Bash(curl * | sh)", can
//     match wherever the script runs it (`true; curl x | sh`,
//     `sh -c "curl x | sh"`);
//  4. stdin, and each statement of it, since a shell given no -c script runs
//     its standard input as one.
//
// Words are shell-quoted only when they need it, as a person would write the
// command. The program nix runs is the package's mainProgram, which the
// attribute does not reliably name (bashInteractive runs bash, busybox runs
// any applet), so the third form takes any program's -c argument as a
// possible script, and the fourth any stdin: a false match only refuses a
// call, a missed one runs it.
func NixRunCommandLines(installable string, args []string, stdin string) []string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = QuoteWord(a)
	}
	forms := []string{strings.Join(append([]string{"nix", "run", QuoteWord(installable), "--"}, quoted...), " ")}
	for _, program := range installablePrograms(installable) {
		forms = append(forms, strings.Join(append([]string{QuoteWord(program)}, quoted...), " "))
	}
	if script, ok := ShellScript(append([]string{"sh"}, args...)); ok {
		forms = append(forms, script)
		forms = append(forms, ScriptStatements(script)...)
	}
	if text := strings.TrimSpace(stdin); text != "" {
		forms = append(forms, text)
		forms = append(forms, ScriptStatements(stdin)...)
	}
	return forms
}

// installablePrograms returns the program names an installable's attribute
// path suggests: its last dot-separated component, without an output
// selector ("^out") or quotes, and, when it carries one, that component
// without its version or variant suffix (see attributeBaseName); nil when
// the installable has no attribute.
func installablePrograms(installable string) []string {
	_, attr, ok := strings.Cut(installable, "#")
	if !ok {
		return nil
	}
	attr, _, _ = strings.Cut(attr, "^")
	if i := strings.LastIndex(attr, "."); i >= 0 {
		attr = attr[i+1:]
	}
	attr = strings.Trim(attr, `"`)
	if attr == "" {
		return nil
	}
	if base := attributeBaseName(attr); base != attr {
		return []string{attr, base}
	}
	return []string{attr}
}

// attributeBaseName returns a nixpkgs attribute name without the version or
// variant suffix it may carry after the program it packages: the text before
// the first "-", "_" or upper-case letter after the first character
// (pnpm_9 is pnpm, nodejs_22 is nodejs, yarn-berry is yarn,
// bashInteractive is bash, python3Minimal is python3). It only widens what
// is judged: a name it cuts wrongly adds a line, it never drops one.
func attributeBaseName(attr string) string {
	if i := strings.IndexFunc(attr[1:], func(r rune) bool { return r == '-' || r == '_' || r >= 'A' && r <= 'Z' }); i >= 0 {
		return attr[:i+1]
	}
	return attr
}
