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
// protected path in the command text. A nested session can also escape this
// session's permission rules (--dangerously-skip-permissions, --add-dir), or
// load another directory's project settings when started from outside the
// project. SP-008 guards those.

// hookDroppingFlags are the Claude Code options that start a session without
// the hooks the settings files register: --bare skips hooks, --safe-mode
// disables every customization including hooks, and --restricted ignores the
// user, project and local settings files.
var hookDroppingFlags = map[string]bool{"--bare": true, "--safe-mode": true, "--restricted": true}

// settingsFlags are the Claude Code options whose value selects the settings a
// session loads.
var settingsFlags = map[string]bool{"--settings": true, "--setting-sources": true}

// permissionEscalatingFlags are the Claude Code options that start a session
// with fewer permission checks than this one: the two skip-permissions
// options turn them off, and --add-dir grants access to directories outside
// the session's own.
var permissionEscalatingFlags = map[string]bool{
	"--dangerously-skip-permissions":       true,
	"--allow-dangerously-skip-permissions": true,
	"--add-dir":                            true,
}

// permissionModeFlag selects the permission mode of a session; with the mode
// bypassPermissions (any case) it skips every permission check.
const (
	permissionModeFlag     = "--permission-mode"
	bypassPermissionsValue = "bypassPermissions"
)

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

// maxSettingsScriptDepth bounds how deep settingsOverride follows shell scripts
// nested in `sh -c` or eval.
const maxSettingsScriptDepth = 3

// reClaudeWord finds the Claude Code CLI in text that cannot be parsed: claude
// as a word or path base, or the claude-code package, but not the .claude
// directory or a longer name that embeds it.
var reClaudeWord = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_.-])claude(-code)?([^A-Za-z0-9_.-]|$)`)

// settingsOverride reports why a shell command changes the settings, and so the
// hooks, of a Claude Code session: it writes through $CLAUDE_CONFIG_DIR, or
// starts Claude Code with a hook-dropping option, a settings override, a
// permission-escalating option or a settings environment override, or from a
// working directory that is not the session's or inside it, or runs a
// subcommand that writes settings. Claude Code started through a wrapper
// (env, sudo, xargs, npx) or a shell script is followed; a command that
// cannot be parsed and names Claude Code with an override marker fails
// closed. Returns "" when it does none of these.
func settingsOverride(ctx *EvalContext) string {
	text := looseText(ctx.Command)
	if reason := configDirWrite(ctx, text); reason != "" {
		return reason
	}
	if isPowerShell(ctx) {
		if reason := psLaunchDirOverride(ctx); reason != "" {
			return reason
		}
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		if reClaudeWord.MatchString(text) && mentionsOverride(text) {
			return "the command starts Claude Code with a settings override but cannot be parsed"
		}
		return ""
	}
	cmds := make([]cmdscan.Command, len(scs))
	for i, sc := range scs {
		cmds[i] = sc.Command
	}
	dirs := mainShellDirs(ctx, scs)
	return claudeLaunchOverride(cmds, text, 0, func(i int) string { return launchDirReason(ctx.CWD, dirs[i]) })
}

// claudeLaunchOverride checks each command that runs Claude Code, following
// shell scripts, against the settings overrides and the permission-escalating
// options, and judges the directory it runs in. text is the whole line, whose
// override markers count against a launch whose arguments are not all visible
// (an expansion, xargs). dirReason reports why the command at an index runs
// outside the session's working directory, or "".
func claudeLaunchOverride(cmds []cmdscan.Command, text string, depth int, dirReason func(int) string) string {
	envSet := ""
	for _, c := range cmds {
		if v := envAssignment(c, isOverrideEnv); v != "" {
			envSet = v
		}
	}
	for idx, c := range cmds {
		words := append([]string{c.Name}, c.Args...)
		launchDir := func() string {
			if cmdscan.Program(words).DirChanged {
				return dirChangedReason
			}
			return dirReason(idx)
		}
		if script, ok := cmdscan.ShellScript(words); ok {
			if reason := scriptLaunchOverride(script, depth, launchDir()); reason != "" {
				return reason
			}
		}
		if reason := unknownProgramOverride(c); reason != "" {
			return reason
		}
		i := slices.IndexFunc(words, canon.IsClaudeCLI)
		if i < 0 || !canon.IsClaudePackage(words[i]) && !canon.IsClaudeEntryScript(words[i]) && !slices.Contains(cmdscan.CommandWordIndexes(words), i) {
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
		if printsOnly(words[i+1:]) {
			continue
		}
		if reason := launchDir(); reason != "" {
			return reason
		}
	}
	return ""
}

// scriptLaunchOverride checks a script run through `sh -c` or eval. outerDir
// is why the command running the script runs outside the session's working
// directory, or ""; a launch after a cd, pushd or popd inside the script
// fails closed, since the script's directory changes are not followed.
func scriptLaunchOverride(script string, depth int, outerDir string) string {
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
	firstCD := slices.IndexFunc(sub, func(c cmdscan.Command) bool { return cmdscan.ChangesDir(c.Name) })
	return claudeLaunchOverride(sub, script, depth+1, func(i int) string {
		if outerDir == "" && firstCD >= 0 && i > firstCD {
			return scriptDirReason
		}
		return outerDir
	})
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
		if permissionEscalatingFlags[name] {
			return "Claude Code started with " + name + ", which escapes this session's permission rules"
		}
		if !settingsFlags[name] && name != permissionModeFlag {
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				return name + " without a value"
			}
			i++
			value = args[i]
		}
		if reason := valuedFlagOverride(name, value); reason != "" {
			return reason
		}
	}
	return ""
}

// valuedFlagOverride judges the value of a settings option or of
// --permission-mode.
func valuedFlagOverride(name, value string) string {
	switch name {
	case "--settings":
		return inlineSettingsOverride(value)
	case permissionModeFlag:
		if strings.EqualFold(strings.TrimSpace(value), bypassPermissionsValue) {
			return "Claude Code started with " + permissionModeFlag + " " + bypassPermissionsValue + ", which skips every permission check"
		}
		return ""
	default:
		return settingSourcesOverride(value)
	}
}

// claudeInfoFlags are the Claude Code options that print the version or the
// help and exit, loading no settings and running no hooks.
var claudeInfoFlags = map[string]bool{"--version": true, "-v": true, "--help": true, "-h": true}

// printsOnly reports whether a Claude Code argument list, before any "--",
// holds an option that only prints (claudeInfoFlags), so where the launch
// runs does not matter.
func printsOnly(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if claudeInfoFlags[a] {
			return true
		}
	}
	return false
}

// launchDirReason reports why a Claude Code launch at sc runs in a working
// directory other than the session's: a session started elsewhere loads that
// directory's project settings, and with them other hooks or none. A
// directory that cannot be resolved fails closed; with no session cwd known,
// only that is judged.
func launchDirReason(sessionCWD string, sc scannedCommand) string {
	switch {
	case sc.cwdUnknown:
		return "Claude Code started after a directory change this check cannot resolve, so it may load another directory's project settings and hooks"
	case sessionCWD != "" && !withinDir(sc.cwd, sessionCWD):
		return "Claude Code started from " + sc.cwd + ", outside the session's working directory, loads that directory's project settings and hooks"
	}
	return ""
}

// dirChangedReason is the reason for a launch whose wrapper runs it in
// another directory (`env -C /tmp claude`).
const dirChangedReason = "Claude Code started by a wrapper that changes its working directory, so it loads another directory's project settings and hooks"

// scriptDirReason is the reason for a launch after a cd inside a shell
// script, whose directory changes this check does not follow.
const scriptDirReason = "Claude Code started after a directory change inside a shell script, so it may load another directory's project settings and hooks"

// psLaunchDirOverride judges the working directory of each Claude Code launch
// on a PowerShell line, tracked through Set-Location, Push-Location and
// Pop-Location (see psScanned).
func psLaunchDirOverride(ctx *EvalContext) string {
	for _, sc := range ctx.psScanned() {
		if !canon.IsClaudeCLI(sc.Name) {
			continue
		}
		if reason := launchDirReason(ctx.CWD, sc); reason != "" {
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
	if _, ok := settings[skipDangerousPromptKey]; ok {
		return "--settings sets " + skipDangerousPromptKey + ", which escapes this session's permission rules"
	}
	if raw, ok := settings[permissionsKey]; ok {
		return permissionsEscalation(raw)
	}
	return ""
}

// Inline settings keys that grant what the permission-escalating options do:
// permissions.defaultMode bypassPermissions is --permission-mode
// bypassPermissions, permissions.additionalDirectories is --add-dir, and
// skipDangerousModePermissionPrompt drops the prompt before a bypass.
const (
	permissionsKey         = "permissions"
	defaultModeKey         = "defaultMode"
	additionalDirsKey      = "additionalDirectories"
	skipDangerousPromptKey = "skipDangerousModePermissionPrompt"
)

// permissionsEscalation judges the permissions block of inline --settings:
// a bypassPermissions default mode (any case) or additional directories
// escape this session's permission rules. A block that is not an object, or
// a default mode that is not a string, fails closed.
func permissionsEscalation(raw json.RawMessage) string {
	var perms map[string]json.RawMessage
	if err := json.Unmarshal(raw, &perms); err != nil {
		return "--settings sets permissions that cannot be read"
	}
	if _, ok := perms[additionalDirsKey]; ok {
		return "--settings sets permissions." + additionalDirsKey + ", which escapes this session's permission rules"
	}
	mode, ok := perms[defaultModeKey]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(mode, &s); err != nil || strings.EqualFold(strings.TrimSpace(s), bypassPermissionsValue) {
		return "--settings sets permissions." + defaultModeKey + " " + bypassPermissionsValue + ", which skips every permission check"
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

func isOverrideEnv(name string) bool {
	return slices.Contains(launchOverrideEnv, name)
}

func isOverrideFlag(word string) bool {
	name, _, _ := strings.Cut(word, "=")
	return hookDroppingFlags[name] || settingsFlags[name] || permissionEscalatingFlags[name] || name == permissionModeFlag
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
