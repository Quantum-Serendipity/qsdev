package check

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/shebang"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// selfprotectSubcommand is the subcommand the self-protection hook runs; a
// hook whose program does not resolve is critical when it is this one.
const selfprotectSubcommand = "selfprotect"

// CheckClaudeSettingsPosture verifies that .claude/settings.json still
// enforces what qsdev generated for the project: every generated hook
// registration (self-protection, package guard, ...) is present unchanged,
// hooks are not disabled wholesale, every project hook script a registration
// runs exists, the env variables that hand the hooks their policy are
// unchanged, bypassPermissions is not the default mode, and bypass mode stays
// disabled where the tier disables it. An enabled hook that has no policy to
// enforce is reported as a warning. The generated-file check accepts local edits to settings.json (it is
// merged, not overwritten), so without this check deleting the guard hooks or
// switching to bypassPermissions passed `qsdev check`.
//
// Claude Code runs with settings.json overlaid by the uncommitted
// settings.local.json, so the effective settings are checked too: see
// checkLocalOverride.
//
// The expected registrations come from ctx.ExpectedClaudeSettings, the
// settings.json the generator produces for the project's saved answers, not
// from a fixed list.
func CheckClaudeSettingsPosture(ctx CheckContext) []CheckResult {
	data, err := os.ReadFile(filepath.Join(ctx.ProjectRoot, filepath.FromSlash(ClaudeSettingsRelPath)))
	if err != nil {
		return nil // a missing or unreadable file is reported by the deny-rule check
	}
	actual, err := claudesettings.Parse(data)
	if err != nil {
		return []CheckResult{postureResult("claude_settings_parse", StatusFail, SeverityHigh,
			fmt.Sprintf("Could not parse %s: %v", ClaudeSettingsRelPath, err),
			"Fix the JSON syntax in "+ClaudeSettingsRelPath)}
	}

	var expected *claudesettings.Settings
	if len(ctx.ExpectedClaudeSettings) > 0 {
		if parsed, err := claudesettings.Parse(ctx.ExpectedClaudeSettings); err == nil {
			expected = &parsed
		}
	}

	var results []CheckResult
	if actual.DisableAllHooks {
		results = append(results, postureResult("claude_all_hooks_disabled", StatusFail, SeverityCritical,
			fmt.Sprintf("%s sets %s, so no generated hook (self-protection, package guard) runs", ClaudeSettingsRelPath, claudesettings.KeyDisableAllHooks),
			"Remove "+claudesettings.KeyDisableAllHooks+" or run 'qsdev init --update' to restore the generated settings"))
	}
	if actual.DefaultMode == claudesettings.ModeBypassPermissions {
		results = append(results, postureResult("claude_bypass_permissions_mode", StatusFail, SeverityHigh,
			fmt.Sprintf("%s sets permissions.defaultMode to %s, so no permission prompt guards agent actions", ClaudeSettingsRelPath, claudesettings.ModeBypassPermissions),
			"Remove permissions.defaultMode or run 'qsdev init --update' to restore the generated settings"))
	}
	if expected != nil {
		results = append(results, checkDisableBypass(actual, *expected)...)
		results = append(results, checkHookRegistrations(actual, *expected)...)
		results = append(results, checkHookEnv(actual, *expected)...)
	}
	lookPath := ctx.LookPath
	if lookPath == nil {
		lookPath = toolcheck.LookPath
	}
	results = append(results, checkHookScripts(ctx.ProjectRoot, actual, lookPath)...)
	results = append(results, checkHookPrograms(ctx.ProjectRoot, actual, lookPath)...)
	results = append(results, checkHooksWithoutPolicy(ctx.HooksWithoutPolicy)...)
	results = append(results, checkLocalOverride(ctx.ProjectRoot, ctx.ClaudeUserDir, actual, expected)...)

	if len(results) == 0 {
		return []CheckResult{postureResult("claude_settings_posture", StatusPass, SeverityInfo,
			"Generated hook registrations and permission mode are intact in "+ClaudeSettingsRelPath, "")}
	}
	return results
}

func checkDisableBypass(actual, expected claudesettings.Settings) []CheckResult {
	want := expected.DisableBypassPermissionsMode
	if want == "" || actual.DisableBypassPermissionsMode == want {
		return nil
	}
	return []CheckResult{postureResult("claude_disable_bypass_missing", StatusFail, SeverityHigh,
		fmt.Sprintf("%s no longer sets permissions.disableBypassPermissionsMode to %q, which the project's tier requires", ClaudeSettingsRelPath, want),
		"Run 'qsdev init --update' to restore the generated settings")}
}

func checkHookRegistrations(actual, expected claudesettings.Settings) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(expected.Hooks)) {
		severity := SeverityMedium
		if event == claudesettings.EventPreToolUse {
			severity = SeverityHigh
		}
		for _, m := range expected.Hooks[event] {
			for _, h := range m.Hooks {
				if h.Command == "" || actual.Registered(event, m.Matcher, h) {
					continue
				}
				sev := severity
				if event == claudesettings.EventPreToolUse && claudesettings.IsFailClosed(h.Command) {
					// A generated guard (self-protection, package guard, ...)
					// left unregistered is off, as a gutted guard script is.
					sev = SeverityCritical
				}
				r := postureResult("claude_hook_missing", StatusFail, sev,
					fmt.Sprintf("Generated %s hook %q (matcher %q) is not registered in %s", event, h.Command, m.Matcher, ClaudeSettingsRelPath),
					"Run 'qsdev init --update' to restore the generated hook registrations")
				r.Metadata = map[string]string{"event": event, "matcher": m.Matcher, "command": h.Command}
				results = append(results, r)
			}
		}
	}
	return results
}

// checkHookEnv reports every "env" variable the generator sets from the
// committed hook policy (e.g. the tool-gates allow and deny lists) that is
// missing or holds another value on disk: the hook reads its policy only from
// there, so an edited variable silently changes what it enforces.
func checkHookEnv(actual, expected claudesettings.Settings) []CheckResult {
	var results []CheckResult
	for _, key := range slices.Sorted(maps.Keys(expected.Env)) {
		want := expected.Env[key]
		got, ok := actual.Env[key]
		if ok && got == want {
			continue
		}
		msg := fmt.Sprintf("%s env %s is missing; the committed hook policy sets it to %q", ClaudeSettingsRelPath, key, want)
		if ok {
			msg = fmt.Sprintf("%s env %s is %q; the committed hook policy sets it to %q", ClaudeSettingsRelPath, key, got, want)
		}
		r := postureResult("claude_hook_env_changed", StatusFail, SeverityHigh, msg,
			"Run 'qsdev init --update' to restore the generated hook policy, or change it in .qsdev.yaml")
		r.Metadata = map[string]string{"variable": key}
		results = append(results, r)
	}
	return results
}

// checkHooksWithoutPolicy warns about each enabled hook that has no policy
// to enforce, so it runs on every call yet restricts nothing.
func checkHooksWithoutPolicy(hooks []HookWithoutPolicy) []CheckResult {
	results := make([]CheckResult, 0, len(hooks))
	for _, h := range hooks {
		r := postureResult("claude_hook_no_policy", StatusWarn, SeverityMedium,
			fmt.Sprintf("Claude Code hook %q is enabled (no policy): it runs on every matching tool call but restricts nothing", h.Name),
			fmt.Sprintf("Set %s in .qsdev.yaml and run 'qsdev init --update', or disable the hook", h.PolicyKey))
		r.Metadata = map[string]string{"hook": h.Name, "policy_key": h.PolicyKey}
		results = append(results, r)
	}
	return results
}

// checkHookScripts reports registered hook commands whose project script is
// missing, and scripts a hook starts as a program that cannot run: their
// interpreter does not resolve or they are not executable. Claude Code
// treats each failure as a non-blocking error, so the guard silently stops
// applying. Interpreters are only looked up,
// never run.
func checkHookScripts(projectRoot string, actual claudesettings.Settings, lookPath func(string) (string, error)) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(actual.Hooks)) {
		severity := SeverityMedium
		if event == claudesettings.EventPreToolUse {
			severity = SeverityHigh
		}
		for _, ref := range actual.ScriptRefs(event) {
			if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(ref.Script))); err != nil {
				r := postureResult("claude_hook_script_missing", StatusFail, severity,
					fmt.Sprintf("%s hook %q runs %s, which does not exist", event, ref.Command, ref.Script),
					"Run 'qsdev repair' to restore the hook script")
				r.FilePath = ref.Script
				results = append(results, r)
				continue
			}
			if !slices.Contains(programScripts(ref.Command), ref.Script) {
				continue
			}
			problem := scriptRunProblem(projectRoot, filepath.Join(projectRoot, filepath.FromSlash(ref.Script)), ref.Script, lookPath)
			if problem == nil {
				continue
			}
			r := postureResult("claude_hook_unresolvable", StatusFail, severity,
				fmt.Sprintf("%s hook %q runs %s, which %s, so the hook cannot run", event, ref.Command, ref.Script, problem.reason),
				problem.remediation)
			r.FilePath = ref.Script
			r.Metadata = map[string]string{"event": event, "program": problem.program}
			results = append(results, r)
		}
	}
	return results
}

// programScripts returns the project hook scripts command starts as a
// program: the command word (directly or through wrappers such as exec, env
// or timeout, see cmdscan.ProgramWordIndex), the command after the "--" of
// `<app> sandbox exec ... -- <command>`, or one of those inside a `sh -c`
// script. A script passed to an interpreter (`sh <script>`,
// `python3 -- <script>`) needs neither an exec bit nor a resolvable shebang.
// An unparseable command, or one nested too deep, counts every script it
// names.
func programScripts(command string) []string {
	return nestedProgramScripts(command, 0)
}

// maxScriptNesting bounds how many `sh -c` levels programScripts parses.
const maxScriptNesting = 4

func nestedProgramScripts(command string, depth int) []string {
	cmds, err := cmdscan.Parse(command)
	if err != nil || depth > maxScriptNesting {
		return claudesettings.ScriptRe.FindAllString(command, -1)
	}
	var scripts []string
	for _, c := range cmds {
		if c.Name != "" {
			scripts = append(scripts, runScripts(append([]string{c.Name}, c.Args...), depth)...)
		}
	}
	return scripts
}

// runScripts returns the project hook scripts the words of one simple
// command start as a program (see programScripts).
func runScripts(words []string, depth int) []string {
	p := cmdscan.ProgramWordIndex(words)
	if p < 0 {
		return nil
	}
	run := words[p:]
	scripts := claudesettings.ScriptRe.FindAllString(run[0], -1)
	if script, ok := cmdscan.ShellScript(run); ok {
		scripts = append(scripts, nestedProgramScripts(script, depth+1)...)
	}
	if rest, ok := sandboxExecCommand(run); ok {
		scripts = append(scripts, runScripts(rest, depth)...)
	}
	return scripts
}

// sandboxExecCommand returns the command `<app> sandbox exec [options] --
// <command>` runs, when run is that form.
func sandboxExecCommand(run []string) ([]string, bool) {
	if len(run) < 3 || cmdscan.ProgramName(run[0]) != branding.Get().AppName || run[1] != "sandbox" || run[2] != "exec" {
		return nil, false
	}
	i := slices.Index(run[3:], "--")
	if i < 0 {
		return nil, false
	}
	return run[3+i+1:], true
}

// runProblem is why the shell cannot start a hook script or program, and the
// file that fails: the script itself when it cannot be read or executed,
// otherwise the interpreter its interpreter line names.
type runProblem struct {
	program, reason, remediation string
}

const (
	remediateInterpreter = "Install the script's interpreter on PATH, make the script executable, or run 'qsdev repair' to restore it"
	remediateCRLF        = "Convert the script to LF line endings (and add a '.gitattributes' rule such as '.claude/hooks/** text eol=lf'), or run 'qsdev repair' to restore it"
)

// scriptRunProblem returns why the shell cannot start the file at path
// (named label in findings) as a program, or nil when it can. A relative
// interpreter resolves against projectRoot, the hook's working directory. On Windows Claude
// Code runs hooks through Git Bash, which has no exec bit, tolerates CRLF
// interpreter lines and maps absolute interpreters such as /usr/bin/python3
// into its own layer, so only env lookups are checked there.
func scriptRunProblem(projectRoot, path, label string, lookPath func(string) (string, error)) *runProblem {
	line, err := shebang.Read(path)
	if err != nil {
		return &runProblem{label, fmt.Sprintf("cannot be read: %v", err), remediateInterpreter}
	}
	posix := runtime.GOOS != "windows"
	named := cmp.Or(line.Program(), line.Interpreter)
	switch {
	case posix && !canExecute(path):
		return &runProblem{label, "is not executable for this user (the shell exits 126)", remediateInterpreter}
	case line.Interpreter == "":
		return nil // no #! line: the shell runs it as a shell script
	case posix && strings.ContainsRune(line.Interpreter+line.Arg, '\r'):
		return &runProblem{named, "has a CRLF interpreter line, so the kernel looks for an interpreter or argument ending in a carriage return", remediateCRLF}
	case line.ViaEnv():
		if posix {
			// The kernel must find env itself before env looks up the program.
			if p := absInterpreterProblem(projectRoot, line.Interpreter); p != nil {
				return p
			}
		}
		prog, ok := line.EnvProgram(runtime.GOOS)
		if !ok {
			return &runProblem{line.Interpreter, fmt.Sprintf("has interpreter line %q, from which env runs no program", line.Interpreter+" "+line.Arg), remediateInterpreter}
		}
		if _, err := lookPath(prog); err != nil {
			return &runProblem{prog, fmt.Sprintf("needs interpreter %q, which is not on PATH (the shell exits 127)", prog), remediateInterpreter}
		}
	case posix:
		return absInterpreterProblem(projectRoot, line.Interpreter)
	}
	return nil
}

// absInterpreterProblem returns why the kernel cannot run interp, a path
// relative to projectRoot (the hook's working directory) unless absolute, as
// an interpreter, or nil.
func absInterpreterProblem(projectRoot, interp string) *runProblem {
	p := interp
	if !filepath.IsAbs(p) {
		p = filepath.Join(projectRoot, p)
	}
	fi, err := os.Stat(p)
	if err != nil {
		return &runProblem{interp, fmt.Sprintf("needs interpreter %s, which does not exist", interp), remediateInterpreter}
	}
	if fi.IsDir() || !canExecute(p) {
		return &runProblem{interp, fmt.Sprintf("needs interpreter %s, which is not an executable file for this user", interp), remediateInterpreter}
	}
	return nil
}

// checkLocalOverride reports each way the per-machine settings (the
// project's settings.local.json and, when userDir is set, the user
// settings.json beneath it) weaken the committed settings.json in the
// effective settings. Disabling hooks fails at critical, as a gutted guard
// script does: it switches off self-protection and the package guard
// together. Defaulting to bypassPermissions fails at high. CI never sees
// these files, but the agent on this machine runs without its guardrails.
// Changing disableBypassPermissionsMode or a hook-policy env variable warns.
// Weakenings the committed file already has are reported by the checks above.
func checkLocalOverride(projectRoot, userDir string, project claudesettings.Settings, expected *claudesettings.Settings) []CheckResult {
	eff, err := claudesettings.ReadWith(projectRoot, claudesettings.ReadOptions{UserDir: userDir})
	if err != nil {
		r := postureResult("claude_settings_parse", StatusFail, SeverityHigh,
			fmt.Sprintf("Could not read the effective Claude settings: %v", err),
			"Fix the JSON syntax in the named settings file or delete it")
		r.FilePath = claudesettings.LocalRelPath
		return []CheckResult{r}
	}

	var results []CheckResult
	if user := eff.User; user != nil && !project.DisableAllHooks && user.DisableAllHooks {
		r := postureResult("claude_settings_user_override", StatusFail, SeverityCritical,
			fmt.Sprintf("%s sets %s, so no generated hook (self-protection, package guard) runs on this machine",
				claudesettings.UserLabel, claudesettings.KeyDisableAllHooks),
			"Remove "+claudesettings.KeyDisableAllHooks+" from "+claudesettings.UserLabel)
		r.Metadata = map[string]string{"source": claudesettings.UserLabel}
		results = append(results, r)
	}
	local := eff.Local
	if local == nil {
		return results
	}
	add := func(status CheckStatus, severity CheckSeverity, msg string) {
		r := postureResult("claude_settings_local_override", status, severity,
			claudesettings.LocalRelPath+" "+msg,
			"Remove the override from "+claudesettings.LocalRelPath+", or change the committed policy with 'qsdev init --update'")
		r.FilePath = claudesettings.LocalRelPath
		results = append(results, r)
	}
	if local.DisableAllHooks && !project.DisableAllHooks {
		add(StatusFail, SeverityCritical, fmt.Sprintf("sets %s, so no generated hook (self-protection, package guard) runs on this machine",
			claudesettings.KeyDisableAllHooks))
	}
	if local.DefaultMode == claudesettings.ModeBypassPermissions && project.DefaultMode != claudesettings.ModeBypassPermissions {
		add(StatusFail, SeverityHigh, fmt.Sprintf("sets %s.%s to %s, so no permission prompt guards agent actions on this machine",
			claudesettings.KeyPermissions, claudesettings.KeyDefaultMode, claudesettings.ModeBypassPermissions))
	}

	// Without the generated settings, the committed value is the baseline;
	// the hook-policy env variables are known only from the generated ones.
	wantBypass, wantEnv := project.DisableBypassPermissionsMode, map[string]string(nil)
	if expected != nil {
		wantBypass, wantEnv = expected.DisableBypassPermissionsMode, expected.Env
	}
	if wantBypass != "" && local.DisableBypassPermissionsMode != "" && eff.DisableBypassPermissionsMode != wantBypass {
		add(StatusWarn, SeverityMedium, fmt.Sprintf("sets %s.%s to %q; the committed settings require %q",
			claudesettings.KeyPermissions, claudesettings.KeyDisableBypassPermissionsMode, eff.DisableBypassPermissionsMode, wantBypass))
	}
	for _, key := range slices.Sorted(maps.Keys(wantEnv)) {
		if got, ok := local.Env[key]; ok && got != wantEnv[key] {
			add(StatusWarn, SeverityMedium, fmt.Sprintf("sets %s %s to %q; the committed hook policy sets it to %q",
				claudesettings.KeyEnv, key, got, wantEnv[key]))
		}
	}
	return results
}

// checkHookPrograms reports registered hook commands whose program cannot
// run: a bare name that does not resolve on PATH, or a path (literal, or
// under ${CLAUDE_PROJECT_DIR}) that does not exist or cannot be executed.
// The shell exits 127 or 126, which Claude Code treats as a non-blocking
// error, so an unwrapped hook fails open, and a generated fail-closed wrapper
// turns it into a block on every matching tool call. Either way the hook
// never evaluates anything. Every program the hook runs whenever it runs
// is checked (see hookPrograms): shell builtins such as cd and source run
// no program, project hook scripts are covered by checkHookScripts, and a
// command word built from any other expansion cannot be known. Programs are
// only looked up, never run.
func checkHookPrograms(projectRoot string, actual claudesettings.Settings, lookPath func(string) (string, error)) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(actual.Hooks)) {
		for _, m := range actual.Hooks[event] {
			for _, h := range m.Hooks {
				for _, run := range hookPrograms(projectRoot, h.Command) {
					problem := programProblem(projectRoot, run.program, lookPath)
					if problem == nil {
						continue
					}
					severity := SeverityHigh
					if len(run.args) > 0 && run.args[0] == selfprotectSubcommand {
						severity = SeverityCritical
					}
					r := postureResult("claude_hook_unresolvable", StatusFail, severity,
						fmt.Sprintf("%s hook %q runs %s, which %s, so the hook cannot run: unwrapped it fails and Claude Code lets the call through; wrapped fail-closed it blocks every matching call", event, h.Command, run.program, problem.reason),
						problem.remediation)
					r.Metadata = map[string]string{"event": event, "program": problem.program}
					results = append(results, r)
				}
			}
		}
	}
	return results
}

// programProblem returns why the shell cannot run program, a bare name or a
// path, or nil when it can.
func programProblem(projectRoot, program string, lookPath func(string) (string, error)) *runProblem {
	if !strings.ContainsAny(program, `/\`) {
		if _, err := lookPath(program); err != nil {
			return &runProblem{program, "is not on PATH (the shell exits 127)",
				fmt.Sprintf("Install %s on PATH or run 'qsdev init --update' to regenerate the hook", program)}
		}
		return nil
	}
	path := filepath.FromSlash(program)
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}
	remediation := fmt.Sprintf("Restore %s or run 'qsdev init --update' to regenerate the hook", program)
	info, err := os.Stat(path)
	if err != nil {
		return &runProblem{program, "does not exist (the shell exits 127)", remediation}
	}
	if info.IsDir() {
		return &runProblem{program, "is a directory (the shell exits 126)", remediation}
	}
	return scriptRunProblem(projectRoot, path, program, lookPath)
}

// hookRun is one program a hook command runs, with its arguments.
type hookRun struct {
	program string
	args    []string
}

// hookPrograms returns the programs a hook command runs whenever it runs, in
// order, following wrappers (exec, env, timeout, see cmdscan.Program). Shell
// builtins run no program and are taken to succeed, except the status tests
// (statusTests, `command -v`); a program's exit status is unknown, so it is
// a test too. A statement is skipped when it runs only conditionally: behind
// ||, in a branch, loop or function (cmdscan.Guarded), or in an && list
// after a test (`test -x prog && prog`, `[ -f .env ] && . ./.env`), but
// `cd "$dir" && prog` and `cd "$dir" || exit 1; prog` run prog. Scanning
// stops at a statement that may end the hook (exit, return, exec of a
// command) unless it runs only when a builtin taken to succeed fails, so
// `command -v prog || exit 0; prog` checks nothing. A leading unquoted ~ or
// ~/ expands to the home directory, and ${CLAUDE_PROJECT_DIR} renders as
// projectRoot. A program that is a project hook script (see
// checkHookScripts), is built from any other expansion (including ~user),
// or cannot be named is left out, as is every program of an unparseable
// command. On Windows a POSIX absolute path is left out, as Git Bash maps
// it into its own layer.
func hookPrograms(projectRoot, command string) []hookRun {
	cmds, err := cmdscan.ParseWithVars(command, map[string]string{claudeProjectDirVar: projectRoot})
	if err != nil {
		return nil
	}
	var runs []hookRun
	tested := false // a statement of the current && list has a status that tests something
	for _, c := range cmds {
		if c.Guard == cmdscan.Unguarded {
			tested = false
		}
		if c.Name == "" {
			continue
		}
		st := classifyStatement(projectRoot, c)
		conditional := c.Guard == cmdscan.Guarded || c.Guard == cmdscan.GuardedByAnd && tested
		if !conditional && st.run != nil {
			runs = append(runs, *st.run)
		}
		if st.ends && (!conditional || tested) {
			break
		}
		tested = tested || st.tests
	}
	return runs
}

// hookStatement is what one simple command of a hook command does, for
// hookPrograms.
type hookStatement struct {
	run   *hookRun // the program to check, when it can be named
	tests bool     // its exit status tests something, so && may guard a branch
	ends  bool     // it can end the hook: exit, return, or exec of a command
}

// classifyStatement classifies the simple command c of a hook command.
func classifyStatement(projectRoot string, c cmdscan.Command) hookStatement {
	words := append([]string{c.Name}, c.Args...)
	run := cmdscan.Program(words)
	if run.Index < 0 {
		return hookStatement{tests: run.LookupOnly || run.CommandString, ends: run.Exec}
	}
	word := words[run.Index]
	expanded := c.NameHasExpansion
	if run.Index > 0 {
		expanded = slices.Contains(c.ExpandedArgs, word)
	}
	if !expanded && run.ShellRuns && cmdscan.IsShellBuiltin(word) {
		return hookStatement{tests: statusTests[word], ends: word == "exit" || word == "return"}
	}
	st := hookStatement{tests: true, ends: run.Exec}
	if program, ok := namedProgram(projectRoot, word, expanded, slices.Contains(c.TildeWords, run.Index)); ok {
		st.run = &hookRun{program: program, args: words[run.Index+1:]}
	}
	return st
}

// namedProgram returns the program a hook's program word names, or false
// when it cannot be known or is left to checkHookScripts (see
// hookPrograms). tilde says the word starts with a ~ the shell expands.
func namedProgram(projectRoot, word string, expanded, tilde bool) (string, bool) {
	if expanded && !strings.HasPrefix(word, projectRoot+"/") || strings.ContainsRune(word, '$') {
		return "", false
	}
	if tilde && !expanded {
		var ok bool
		if word, ok = homeProgram(word); !ok {
			return "", false
		}
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(word, "/") {
		return "", false
	}
	if isProjectHookScript(projectRoot, word) {
		return "", false
	}
	return word, true
}

// statusTests are the builtins whose exit status tests something (`test -x
// prog && prog`), so a command after them in an && list may never run. Other
// builtins, such as cd, are taken to succeed.
var statusTests = map[string]bool{"test": true, "[": true, "false": true, "type": true, "hash": true}

// homeProgram expands the leading ~ or ~/ of a program word the way the
// shell does, to the home directory. ok is false for ~user, whose account
// lookup is not reproduced, and when the home directory is unknown.
func homeProgram(word string) (string, bool) {
	if word != "~" && !strings.HasPrefix(word, "~/") {
		return "", false
	}
	expanded, err := canon.ExpandTilde(word)
	if err != nil {
		return "", false
	}
	return expanded, true
}

// claudeProjectDirVar is the variable Claude Code sets to the project
// directory for hook commands.
const claudeProjectDirVar = "CLAUDE_PROJECT_DIR"

// isProjectHookScript reports whether the program word names a project hook
// script under .claude/hooks/ (claudesettings.ScriptRe), whether through
// ${CLAUDE_PROJECT_DIR} or relative to the project directory.
func isProjectHookScript(projectRoot, word string) bool {
	rel := strings.TrimPrefix(strings.TrimPrefix(word, projectRoot+"/"), "./")
	return claudesettings.ScriptRe.FindString(rel) == rel
}

func postureResult(name string, status CheckStatus, severity CheckSeverity, message, remediation string) CheckResult {
	return CheckResult{
		Category:    CategorySecurityHarden,
		Name:        name,
		Status:      status,
		Severity:    severity,
		Message:     message,
		Remediation: remediation,
		FilePath:    ClaudeSettingsRelPath,
	}
}
