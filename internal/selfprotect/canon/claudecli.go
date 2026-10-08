package canon

import (
	"path"
	"slices"
	"strings"
)

// ClaudePackage is the npm package of the Claude Code CLI, which npx, bunx
// and `pnpm dlx` run by name.
const ClaudePackage = "@anthropic-ai/claude-code"

// IsClaudeCLI reports whether a command word runs the Claude Code CLI: the
// claude binary (any directory, a Windows extension, a Nix wrapper's
// .claude-wrapped), a release of the native installer
// (…/claude/versions/<version>, which the claude launcher links to), its
// npm package or the package's entry script.
func IsClaudeCLI(word string) bool {
	if IsClaudePackage(word) || IsClaudeEntryScript(word) {
		return true
	}
	dir, base := path.Split(strings.ToLower(strings.ReplaceAll(word, `\`, "/")))
	if strings.HasSuffix(dir, "/claude/versions/") && base != "" {
		return true
	}
	for _, ext := range []string{".exe", ".cmd", ".ps1"} {
		base = strings.TrimSuffix(base, ext)
	}
	for strings.HasPrefix(base, ".") && strings.HasSuffix(base, "-wrapped") {
		base = strings.TrimSuffix(strings.TrimPrefix(base, "."), "-wrapped")
	}
	return base == "claude"
}

// IsClaudePackage reports whether word names the Claude Code npm package, at
// any version: a package runner (npx, bunx, pnpm dlx) given it runs Claude
// Code, whatever option precedes it.
func IsClaudePackage(word string) bool {
	return strings.HasPrefix(strings.TrimPrefix(word, "npm:"), ClaudePackage)
}

// IsClaudeEntryScript reports whether word is a script of the Claude Code npm
// package (…/@anthropic-ai/claude-code/cli.js), which an interpreter such as
// node runs as the CLI.
func IsClaudeEntryScript(word string) bool {
	dir, base := path.Split(strings.ToLower(strings.ReplaceAll(word, `\`, "/")))
	return strings.HasSuffix("/"+dir, "/"+ClaudePackage+"/") && slices.Contains([]string{".js", ".mjs", ".cjs"}, path.Ext(base))
}

// IsClaudeProcess reports whether a running process is the Claude Code CLI,
// from its executable's path (exe, "" when unknown) and its argv: the
// executable or argv[0] is the CLI (IsClaudeCLI), or an interpreter runs the
// package's entry script. A process that is the CLI need not carry any
// AgentEnvMarkers in the environment it started with: the CLI sets them for
// the shells it starts.
func IsClaudeProcess(exe string, argv []string) bool {
	if exe != "" && IsClaudeCLI(exe) || len(argv) > 0 && IsClaudeCLI(argv[0]) {
		return true
	}
	return len(argv) > 1 && slices.ContainsFunc(argv[1:], IsClaudeEntryScript)
}
