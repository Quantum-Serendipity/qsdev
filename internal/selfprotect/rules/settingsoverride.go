package rules

import (
	"encoding/json"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// A Bash call cannot change the environment of the running Claude Code
// process or of the hooks it spawns, so exporting a variable there disables
// nothing. What a Bash call CAN do is start another Claude Code session that
// loads different settings, and with them no hooks, or write the settings of a
// relocated configuration directory through $CLAUDE_CONFIG_DIR, which names no
// protected path in the command text. SP-008 guards those.

// hookDroppingFlags are the Claude Code options that start a session without
// the hooks the settings files register: --bare skips hooks, --safe-mode
// disables every customization including hooks, and --restricted ignores the
// user, project and local settings files.
var hookDroppingFlags = map[string]bool{"--bare": true, "--safe-mode": true, "--restricted": true}

// settingsFlags are the Claude Code options whose value selects the settings a
// session loads.
var settingsFlags = map[string]bool{"--settings": true, "--setting-sources": true}

// settingsOverrideEnv are the environment variables that change which settings
// a Claude Code session loads or whether it runs hooks: CLAUDE_CONFIG_DIR
// relocates the user settings, and CLAUDE_CODE_SIMPLE / CLAUDE_CODE_SAFE_MODE
// are what --bare and --safe-mode set.
var settingsOverrideEnv = []string{canon.ClaudeConfigDirEnv, "CLAUDE_CODE_SIMPLE", "CLAUDE_CODE_SAFE_MODE"}

// launchOverrideEnv are the variables that, set for a Claude Code launch,
// change the settings it loads: settingsOverrideEnv plus HOME, under which the
// default ~/.claude configuration directory lives (HOME=/tmp/empty drops
// every user-level hook). HOME is kept out of mentionsOverride, since $HOME
// appears on ordinary command lines.
var launchOverrideEnv = append([]string{"HOME"}, settingsOverrideEnv...)

// hookSettingsKeys are the settings keys that decide which hooks run and in
// what environment: disableAllHooks turns every hook off, hooks registers hook
// commands, and env sets variables (PATH among them) for the session and every
// hook it spawns.
var hookSettingsKeys = []string{"disableAllHooks", "hooks", "env"}

// requiredSettingSources are the settings sources that register hooks; a
// --setting-sources list must keep all of them.
var requiredSettingSources = []string{"user", "project", "local"}

// settingsWritingSubcommands are Claude Code subcommands that write its
// settings files on the agent's behalf, where SP-001 cannot see the file:
// `claude import` copies another agent's configuration into Claude Code. Their
// --dry-run form writes nothing.
var settingsWritingSubcommands = map[string]bool{"import": true}

// claudePackage is the npm package of the Claude Code CLI, which npx, bunx and
// `pnpm dlx` run by name.
const claudePackage = "@anthropic-ai/claude-code"

// maxSettingsScriptDepth bounds how deep settingsOverride follows shell scripts
// nested in `sh -c` or eval.
const maxSettingsScriptDepth = 3

// reClaudeWord finds the Claude Code CLI in text that cannot be parsed: claude
// as a word or path base, or the claude-code package, but not the .claude
// directory or a longer name that embeds it.
var reClaudeWord = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_.-])claude(-code)?([^A-Za-z0-9_.-]|$)`)

// settingsOverride reports why a shell command changes the settings, and so the
// hooks, of a Claude Code session: it writes through $CLAUDE_CONFIG_DIR, or
// starts Claude Code with a hook-dropping option, a settings override or a
// settings environment override, or runs a subcommand that writes settings.
// Claude Code started through a wrapper (env, sudo, xargs, npx) or a shell
// script is followed; a command that cannot be parsed and names Claude Code
// with an override marker fails closed. Returns "" when it does none of these.
func settingsOverride(ctx *EvalContext) string {
	text := looseText(ctx.Command)
	if reason := configDirWrite(ctx, text); reason != "" {
		return reason
	}
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		if reClaudeWord.MatchString(text) && mentionsOverride(text) {
			return "the command starts Claude Code with a settings override but cannot be parsed"
		}
		return ""
	}
	return claudeLaunchOverride(cmds, text, 0)
}

// claudeLaunchOverride checks each command that runs Claude Code, following
// shell scripts, against the settings overrides. text is the whole line, whose
// override markers count against a launch whose arguments are not all visible
// (an expansion, xargs).
func claudeLaunchOverride(cmds []cmdscan.Command, text string, depth int) string {
	envSet := ""
	for _, c := range cmds {
		if v := envAssignment(c, isOverrideEnv); v != "" {
			envSet = v
		}
	}
	for _, c := range cmds {
		words := append([]string{c.Name}, c.Args...)
		if script, ok := shellScript(words); ok {
			if reason := scriptLaunchOverride(script, depth); reason != "" {
				return reason
			}
		}
		if reason := unknownProgramOverride(c); reason != "" {
			return reason
		}
		i := slices.IndexFunc(words, isClaudeCLI)
		if i < 0 || !isClaudePackage(words[i]) && !isClaudeEntryScript(words[i]) && !slices.Contains(commandWordIndexes(words), i) {
			continue
		}
		if v := wrapperEnvAssignment(words[:i]); v != "" {
			envSet = v
		}
		if envSet != "" {
			return "Claude Code started with " + envSet + " set, which changes the settings and hooks it loads"
		}
		if reason := claudeArgsOverride(words[i+1:]); reason != "" {
			return reason
		}
		if (c.HasExpansion || i > 0 && path.Base(words[0]) == "xargs") && mentionsOverride(text) {
			return "Claude Code started with arguments this check cannot see, on a line that carries a settings override"
		}
	}
	return ""
}

// scriptLaunchOverride checks a script run through `sh -c` or eval.
func scriptLaunchOverride(script string, depth int) string {
	if !reClaudeWord.MatchString(script) {
		return ""
	}
	if depth >= maxSettingsScriptDepth {
		return "Claude Code runs inside too many nested shell scripts to be checked"
	}
	sub, err := cmdscan.Parse(script)
	if err != nil {
		if mentionsOverride(script) {
			return "a shell script starts Claude Code with a settings override but cannot be parsed"
		}
		return ""
	}
	return claudeLaunchOverride(sub, script, depth+1)
}

// unknownProgramOverride reports a command whose program is only known after
// expansion (`c=claude; $c --bare`) and that passes a hook-dropping or
// settings option: the program may be Claude Code.
func unknownProgramOverride(c cmdscan.Command) string {
	if !c.HasExpansion || (c.Name != "" && !strings.HasSuffix(c.Name, "/")) {
		return ""
	}
	for _, a := range c.Args {
		if isOverrideFlag(a) {
			return "a command whose program is only known after expansion passes " + a
		}
	}
	return ""
}

// claudeArgsOverride reports why a Claude Code argument list changes the
// settings the session loads, or "".
func claudeArgsOverride(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return ""
		}
		if i == 0 && settingsWritingSubcommands[a] && !slices.Contains(args, "--dry-run") {
			return "claude " + a + " writes Claude Code settings"
		}
		name, value, hasValue := strings.Cut(a, "=")
		if hookDroppingFlags[name] {
			return "Claude Code started with " + name + ", which runs without the configured hooks"
		}
		if !settingsFlags[name] {
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				return name + " without a value"
			}
			i++
			value = args[i]
		}
		reason := settingSourcesOverride(value)
		if name == "--settings" {
			reason = inlineSettingsOverride(value)
		}
		if reason != "" {
			return reason
		}
	}
	return ""
}

// inlineSettingsOverride judges a --settings value. Inline JSON may set any key
// but those that decide which hooks run (hookSettingsKeys). A settings file is
// refused: its content can change between this check and the launch.
func inlineSettingsOverride(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "{") {
		return "--settings names a settings file, which cannot be inspected before Claude Code loads it; pass the settings as inline JSON"
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &settings); err != nil {
		return "--settings value is not valid JSON"
	}
	for _, k := range hookSettingsKeys {
		if _, ok := settings[k]; ok {
			return "--settings sets " + k + ", which changes the hooks the session runs"
		}
	}
	return ""
}

// settingSourcesOverride reports a --setting-sources list that drops a source
// hooks are registered in.
func settingSourcesOverride(value string) string {
	var sources []string
	for s := range strings.SplitSeq(value, ",") {
		sources = append(sources, strings.ToLower(strings.TrimSpace(s)))
	}
	for _, want := range requiredSettingSources {
		if !slices.Contains(sources, want) {
			return "--setting-sources omits the " + want + " settings, which register hooks"
		}
	}
	return ""
}

// configDirWrite reports a command that mentions CLAUDE_CONFIG_DIR and writes
// through an expansion, or to a relative path after a cd that cannot be
// resolved: the target may be a settings file of the relocated configuration
// directory, which the command text does not name.
func configDirWrite(ctx *EvalContext, text string) string {
	if !strings.Contains(text, canon.ClaudeConfigDirEnv) {
		return ""
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return "the command references " + canon.ClaudeConfigDirEnv + " but cannot be parsed"
	}
	for _, sc := range scs {
		expandedWrite := sc.HasExpansion && (isMutating(sc) || len(sc.WriteRedirects) > 0)
		if expandedWrite || (sc.cwdUnknown && relativeWriteTarget(sc)) {
			return "write through " + canon.ClaudeConfigDirEnv + ", which can name the Claude Code settings"
		}
	}
	return ""
}

// wrapperEnvAssignment returns a settings override variable that a wrapper
// sets for the program it runs (`sudo CLAUDE_CONFIG_DIR=/x claude`).
func wrapperEnvAssignment(words []string) string {
	for _, w := range words {
		if name, _, ok := strings.Cut(w, "="); ok && isOverrideEnv(name) {
			return name
		}
	}
	return ""
}

// isClaudeCLI reports whether a command word runs the Claude Code CLI: the
// claude binary (any directory, a Windows extension), a release of the native
// installer (…/claude/versions/<version>, which the claude launcher links
// to), its npm package or the package's entry script.
func isClaudeCLI(word string) bool {
	if isClaudePackage(word) || isClaudeEntryScript(word) {
		return true
	}
	dir, base := path.Split(strings.ToLower(strings.ReplaceAll(word, `\`, "/")))
	if strings.HasSuffix(dir, "/claude/versions/") && base != "" {
		return true
	}
	for _, ext := range []string{".exe", ".cmd", ".ps1"} {
		base = strings.TrimSuffix(base, ext)
	}
	return base == "claude"
}

// isClaudePackage reports whether word names the Claude Code npm package, at
// any version: a package runner (npx, bunx, pnpm dlx) given it runs Claude
// Code, whatever option precedes it.
func isClaudePackage(word string) bool {
	return strings.HasPrefix(strings.TrimPrefix(word, "npm:"), claudePackage)
}

// isClaudeEntryScript reports whether word is a script of the Claude Code npm
// package (…/@anthropic-ai/claude-code/cli.js), which an interpreter such as
// node runs as the CLI.
func isClaudeEntryScript(word string) bool {
	dir, base := path.Split(strings.ToLower(strings.ReplaceAll(word, `\`, "/")))
	return strings.HasSuffix("/"+dir, "/"+claudePackage+"/") && slices.Contains([]string{".js", ".mjs", ".cjs"}, path.Ext(base))
}

func isOverrideEnv(name string) bool {
	return slices.Contains(launchOverrideEnv, name)
}

func isOverrideFlag(word string) bool {
	name, _, _ := strings.Cut(word, "=")
	return hookDroppingFlags[name] || settingsFlags[name]
}

// reLongOption finds long options anywhere in text, including inside an
// assignment (`F=--bare`).
var reLongOption = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(--[a-z][a-z-]*)`)

// mentionsOverride reports whether text carries a settings override: a
// hook-dropping or settings option, or a settings override variable.
func mentionsOverride(text string) bool {
	if slices.ContainsFunc(settingsOverrideEnv, func(v string) bool { return strings.Contains(text, v) }) {
		return true
	}
	for _, m := range reLongOption.FindAllStringSubmatch(text, -1) {
		if isOverrideFlag(m[1]) {
			return true
		}
	}
	return false
}
