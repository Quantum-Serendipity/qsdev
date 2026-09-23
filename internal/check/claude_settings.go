package check

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
)

// bypassPermissionsMode is the Claude Code permission mode that skips every
// permission prompt.
const bypassPermissionsMode = "bypassPermissions"

// preToolUseEvent is the hook event the guard hooks (self-protection,
// package guard) are registered under.
const preToolUseEvent = "PreToolUse"

// hookScriptRe matches a project hook script referenced by a hook command,
// e.g. "${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py.
var hookScriptRe = regexp.MustCompile(`\.claude/hooks/[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*`)

// claudeSettingsPosture is the part of .claude/settings.json that decides
// whether the generated guardrails are in force.
type claudeSettingsPosture struct {
	Deny                         []string
	DefaultMode                  string
	DisableBypassPermissionsMode string
	DisableAllHooks              bool
	Hooks                        map[string][]hookMatcher
}

// hookMatcher is one matcher entry of a hook event.
type hookMatcher struct {
	Matcher string
	Hooks   []hookEntry
}

// hookEntry is one registered hook. If is the optional permission-rule
// condition that limits when the hook runs.
type hookEntry struct {
	Type    string
	Command string
	If      string
}

// parseSettingsPosture reads the posture keys of a settings.json document.
// Keys are matched exactly, as Claude Code reads them: encoding/json matches
// struct fields case-insensitively, so a decoy "Hooks" or "Permissions" key
// next to the real one could otherwise decide what the check sees. Duplicate
// keys resolve to the last value, as in JavaScript's JSON.parse.
func parseSettingsPosture(data []byte) (claudeSettingsPosture, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return claudeSettingsPosture{}, err
	}
	var s claudeSettingsPosture
	perms, _ := root["permissions"].(map[string]any)
	s.DefaultMode, _ = perms["defaultMode"].(string)
	s.DisableBypassPermissionsMode, _ = perms["disableBypassPermissionsMode"].(string)
	s.DisableAllHooks, _ = root["disableAllHooks"].(bool)
	deny, _ := perms["deny"].([]any)
	for _, rule := range deny {
		if r, ok := rule.(string); ok {
			s.Deny = append(s.Deny, r)
		}
	}

	events, _ := root["hooks"].(map[string]any)
	s.Hooks = make(map[string][]hookMatcher, len(events))
	for event, v := range events {
		entries, _ := v.([]any)
		for _, e := range entries {
			em, _ := e.(map[string]any)
			var m hookMatcher
			m.Matcher, _ = em["matcher"].(string)
			hooks, _ := em["hooks"].([]any)
			for _, h := range hooks {
				hm, _ := h.(map[string]any)
				var he hookEntry
				he.Type, _ = hm["type"].(string)
				he.Command, _ = hm["command"].(string)
				he.If, _ = hm["if"].(string)
				m.Hooks = append(m.Hooks, he)
			}
			s.Hooks[event] = append(s.Hooks[event], m)
		}
	}
	return s, nil
}

// registered reports whether want is registered for event under matcher with
// the same type and run condition (an added "if" narrows when a guard runs).
func (s claudeSettingsPosture) registered(event, matcher string, want hookEntry) bool {
	for _, m := range s.Hooks[event] {
		if m.Matcher != matcher {
			continue
		}
		if slices.Contains(m.Hooks, want) {
			return true
		}
	}
	return false
}

// CheckClaudeSettingsPosture verifies that .claude/settings.json still
// enforces what qsdev generated for the project: every generated hook
// registration (self-protection, package guard, ...) is present unchanged,
// hooks are not disabled wholesale, every project hook script a registration
// runs exists, bypassPermissions is not
// the default mode, and bypass mode stays disabled where the tier disables
// it. The generated-file check accepts local edits to settings.json (it is
// merged, not overwritten), so without this check deleting the guard hooks or
// switching to bypassPermissions passed `qsdev check`.
//
// The expected registrations come from ctx.ExpectedClaudeSettings, the
// settings.json the generator produces for the project's saved answers, not
// from a fixed list.
func CheckClaudeSettingsPosture(ctx CheckContext) []CheckResult {
	data, err := os.ReadFile(filepath.Join(ctx.ProjectRoot, filepath.FromSlash(ClaudeSettingsRelPath)))
	if err != nil {
		return nil // a missing or unreadable file is reported by the deny-rule check
	}
	actual, err := parseSettingsPosture(data)
	if err != nil {
		return []CheckResult{postureResult("claude_settings_parse", StatusFail, SeverityHigh,
			fmt.Sprintf("Could not parse %s: %v", ClaudeSettingsRelPath, err),
			"Fix the JSON syntax in "+ClaudeSettingsRelPath)}
	}

	var expected *claudeSettingsPosture
	if len(ctx.ExpectedClaudeSettings) > 0 {
		if parsed, err := parseSettingsPosture(ctx.ExpectedClaudeSettings); err == nil {
			expected = &parsed
		}
	}

	var results []CheckResult
	if actual.DisableAllHooks {
		results = append(results, postureResult("claude_all_hooks_disabled", StatusFail, SeverityHigh,
			fmt.Sprintf("%s sets disableAllHooks, so no generated hook (self-protection, package guard) runs", ClaudeSettingsRelPath),
			"Remove disableAllHooks or run 'qsdev init --update' to restore the generated settings"))
	}
	if actual.DefaultMode == bypassPermissionsMode {
		results = append(results, postureResult("claude_bypass_permissions_mode", StatusFail, SeverityHigh,
			fmt.Sprintf("%s sets permissions.defaultMode to %s, so no permission prompt guards agent actions", ClaudeSettingsRelPath, bypassPermissionsMode),
			"Remove permissions.defaultMode or run 'qsdev init --update' to restore the generated settings"))
	}
	if expected != nil {
		results = append(results, checkDisableBypass(actual, *expected)...)
		results = append(results, checkHookRegistrations(actual, *expected)...)
	}
	results = append(results, checkHookScripts(ctx.ProjectRoot, actual)...)

	if len(results) == 0 {
		return []CheckResult{postureResult("claude_settings_posture", StatusPass, SeverityInfo,
			"Generated hook registrations and permission mode are intact in "+ClaudeSettingsRelPath, "")}
	}
	return results
}

func checkDisableBypass(actual, expected claudeSettingsPosture) []CheckResult {
	want := expected.DisableBypassPermissionsMode
	if want == "" || actual.DisableBypassPermissionsMode == want {
		return nil
	}
	return []CheckResult{postureResult("claude_disable_bypass_missing", StatusFail, SeverityHigh,
		fmt.Sprintf("%s no longer sets permissions.disableBypassPermissionsMode to %q, which the project's tier requires", ClaudeSettingsRelPath, want),
		"Run 'qsdev init --update' to restore the generated settings")}
}

func checkHookRegistrations(actual, expected claudeSettingsPosture) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(expected.Hooks)) {
		severity := SeverityMedium
		if event == preToolUseEvent {
			severity = SeverityHigh
		}
		for _, m := range expected.Hooks[event] {
			for _, h := range m.Hooks {
				if h.Command == "" || actual.registered(event, m.Matcher, h) {
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

// checkHookScripts reports registered hook commands whose project script is
// missing: Claude Code treats the failure to run it as a non-blocking error,
// so the guard silently stops applying.
func checkHookScripts(projectRoot string, actual claudeSettingsPosture) []CheckResult {
	var results []CheckResult
	for _, event := range slices.Sorted(maps.Keys(actual.Hooks)) {
		severity := SeverityMedium
		if event == preToolUseEvent {
			severity = SeverityHigh
		}
		for _, m := range actual.Hooks[event] {
			for _, h := range m.Hooks {
				for _, script := range hookScriptRe.FindAllString(h.Command, -1) {
					if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(script))); err == nil {
						continue
					}
					r := postureResult("claude_hook_script_missing", StatusFail, severity,
						fmt.Sprintf("%s hook %q runs %s, which does not exist", event, h.Command, script),
						"Run 'qsdev repair' to restore the hook script")
					r.FilePath = script
					results = append(results, r)
				}
			}
		}
	}
	return results
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
