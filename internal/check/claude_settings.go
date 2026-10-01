package check

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
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
		results = append(results, postureResult("claude_all_hooks_disabled", StatusFail, SeverityHigh,
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
	results = append(results, checkHookScripts(ctx.ProjectRoot, actual)...)
	lookPath := ctx.LookPath
	if lookPath == nil {
		lookPath = toolcheck.LookPath
	}
	results = append(results, checkHookPrograms(actual, lookPath)...)
	results = append(results, checkHooksWithoutPolicy(ctx.HooksWithoutPolicy)...)
	results = append(results, checkLocalOverride(ctx.ProjectRoot, actual, expected)...)

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
				r := postureResult("claude_hook_missing", StatusFail, severity,
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
// missing: Claude Code treats the failure to run it as a non-blocking error,
// so the guard silently stops applying.
func checkHookScripts(projectRoot string, actual claudesettings.Settings) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(actual.Hooks)) {
		severity := SeverityMedium
		if event == claudesettings.EventPreToolUse {
			severity = SeverityHigh
		}
		for _, ref := range actual.ScriptRefs(event) {
			if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(ref.Script))); err == nil {
				continue
			}
			r := postureResult("claude_hook_script_missing", StatusFail, severity,
				fmt.Sprintf("%s hook %q runs %s, which does not exist", event, ref.Command, ref.Script),
				"Run 'qsdev repair' to restore the hook script")
			r.FilePath = ref.Script
			results = append(results, r)
		}
	}
	return results
}

// checkLocalOverride reports each way the per-machine settings.local.json
// weakens the committed settings.json in the effective settings. Disabling
// hooks or defaulting to bypassPermissions fails: CI never sees the file, but
// the agent on this machine runs without its guardrails. Changing
// disableBypassPermissionsMode or a hook-policy env variable warns. Weakenings
// the committed file already has are reported by the checks above.
func checkLocalOverride(projectRoot string, project claudesettings.Settings, expected *claudesettings.Settings) []CheckResult {
	eff, err := claudesettings.Read(projectRoot)
	if err != nil {
		r := postureResult("claude_settings_parse", StatusFail, SeverityHigh,
			fmt.Sprintf("Could not read the effective Claude settings: %v", err),
			"Fix the JSON syntax in "+claudesettings.LocalRelPath+" or delete it")
		r.FilePath = claudesettings.LocalRelPath
		return []CheckResult{r}
	}
	local := eff.Local
	if local == nil {
		return nil
	}

	var results []CheckResult
	add := func(status CheckStatus, severity CheckSeverity, msg string) {
		r := postureResult("claude_settings_local_override", status, severity,
			claudesettings.LocalRelPath+" "+msg,
			"Remove the override from "+claudesettings.LocalRelPath+", or change the committed policy with 'qsdev init --update'")
		r.FilePath = claudesettings.LocalRelPath
		results = append(results, r)
	}
	if local.DisableAllHooks && !project.DisableAllHooks {
		add(StatusFail, SeverityHigh, fmt.Sprintf("sets %s, so no generated hook (self-protection, package guard) runs on this machine",
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

// checkHookPrograms reports registered hook commands whose program is a bare
// name that does not resolve on PATH: the shell exits 127, which Claude Code
// treats as a non-blocking error, so an unwrapped hook fails open, and a
// generated fail-closed wrapper turns it into a block on every matching tool
// call. Either way the hook never evaluates anything. Paths and command
// words built from an expansion are skipped (project scripts are covered by
// checkHookScripts); the program is only looked up, never run.
func checkHookPrograms(actual claudesettings.Settings, lookPath func(string) (string, error)) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(actual.Hooks)) {
		for _, m := range actual.Hooks[event] {
			for _, h := range m.Hooks {
				program, args, ok := hookProgram(h.Command)
				if !ok {
					continue
				}
				if _, err := lookPath(program); err == nil {
					continue
				}
				severity := SeverityHigh
				if len(args) > 0 && args[0] == selfprotectSubcommand {
					severity = SeverityCritical
				}
				r := postureResult("claude_hook_unresolvable", StatusFail, severity,
					fmt.Sprintf("%s hook %q runs %s, which is not on PATH, so the hook cannot run: unwrapped it exits 127 and Claude Code lets the call through; wrapped fail-closed it blocks every matching call", event, h.Command, program),
					fmt.Sprintf("Install %s on PATH or run 'qsdev init --update' to regenerate the hook", program))
				r.Metadata = map[string]string{"event": event, "program": program}
				results = append(results, r)
			}
		}
	}
	return results
}

// hookProgram returns the command word and arguments of a hook command's
// first simple command when that word is a bare program name resolved on
// PATH; ok is false for paths, expansions and unparseable commands.
func hookProgram(command string) (program string, args []string, ok bool) {
	cmds, err := cmdscan.Parse(command)
	if err != nil {
		return "", nil, false
	}
	for _, c := range cmds {
		if c.Name == "" {
			continue
		}
		if c.HasExpansion || strings.ContainsAny(c.Name, `/\$`) {
			return "", nil, false
		}
		return c.Name, c.Args, true
	}
	return "", nil, false
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
