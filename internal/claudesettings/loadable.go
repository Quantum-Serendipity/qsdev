package claudesettings

import (
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// This file models when Claude Code refuses a whole settings file: over a
// value its settings schema rejects (see settingsSchemaProblem), or over its
// hooks. A PreToolUse or PermissionRequest hook it cannot load may be the
// one guarding the permissions declared beside it, so Claude Code (as of
// 2.1.280) applies nothing from that file, neither its hooks nor its deny
// rules, until the entry is fixed or removed. Bad entries of other events
// are dropped one by one and the rest of the file still applies.

// EventPermissionRequest is the other hook event whose unloadable entries
// make Claude Code refuse the whole settings file (see guardEvents).
const EventPermissionRequest = "PermissionRequest"

// guardEvents are the hook events whose entries Claude Code must load for it
// to apply the settings file they sit in.
var guardEvents = []string{EventPreToolUse, EventPermissionRequest}

// hookEvents are the hook event names Claude Code 2.1.280 accepts. A key of
// "hooks" outside this list is ignored, unless it holds guard-event hooks,
// in which case the file is refused. An event added by a later release reads
// as unknown here, which can only refuse more, never credit more.
var hookEvents = []string{
	"PreToolUse", "PostToolUse", "PostToolUseFailure", "PostToolBatch", "Notification",
	"UserPromptSubmit", "UserPromptExpansion", "SessionStart", "SessionEnd", "Stop",
	"StopFailure", "SubagentStart", "SubagentStop", "PreCompact", "PostCompact",
	"PreModelSwitch", "PostModelSwitch", "PermissionRequest", "PermissionDenied", "Setup",
	"TeammateIdle", "TaskCreated", "TaskCompleted", "Elicitation", "ElicitationResult",
	"ConfigChange", "WorktreeCreate", "WorktreeRemove", "InstructionsLoaded", "CwdChanged",
	"FileChanged", "DirectoryAdded", "MessageDisplay",
}

// field is one key of a schema: the check of its value, what that check
// wants (worded for settingsSchemaProblem), and whether the key must be
// present.
type field struct {
	valid    func(any) bool
	want     string
	required bool
}

func optional(valid func(any) bool) field { return field{valid: valid} }
func required(valid func(any) bool) field { return field{valid: valid, required: true} }

// commonHookFields are the keys every hook type accepts.
var commonHookFields = map[string]field{
	"if":            optional(isString),
	"timeout":       optional(isPositiveNumber),
	"statusMessage": optional(isString),
	"once":          optional(isBool),
}

// hookTypes maps each hook type Claude Code accepts to its schema beyond
// commonHookFields. Keys outside the schema are accepted and ignored, and so
// is any value of "cloud", which Claude Code reads leniently.
var hookTypes = map[string]map[string]field{
	HookTypeCommand: {
		"command":       required(isString),
		"args":          optional(isStringArray),
		"shell":         optional(isOneOf("bash", "powershell")),
		"async":         optional(isBool),
		"asyncRewake":   optional(isBool),
		"rewakeMessage": optional(isNonEmptyString),
		"rewakeSummary": optional(isNonEmptyString),
	},
	"prompt": {
		"prompt":          required(isString),
		"model":           optional(isString),
		"continueOnBlock": optional(isBool),
	},
	"agent": {
		"prompt": required(isString),
		"model":  optional(isString),
	},
	"http": {
		"url":            required(isURL),
		"headers":        optional(isStringMap),
		"allowedEnvVars": optional(isStringArray),
	},
	"mcp_tool": {
		"server": required(isString),
		"tool":   required(isString),
		"input":  optional(isObject),
	},
}

func isString(v any) bool { _, ok := v.(string); return ok }

func isNonEmptyString(v any) bool { s, ok := v.(string); return ok && s != "" }

func isBool(v any) bool { _, ok := v.(bool); return ok }

func isPositiveNumber(v any) bool { n, ok := v.(float64); return ok && n > 0 }

// maxSafeInteger is JavaScript's Number.MAX_SAFE_INTEGER, the largest value
// Claude Code's schema accepts as an integer.
const maxSafeInteger = 1<<53 - 1

// isPositiveInt reports whether v is a whole number from 1 to maxSafeInteger.
func isPositiveInt(v any) bool {
	n, ok := v.(float64)
	return ok && n >= 1 && n <= maxSafeInteger && n == math.Trunc(n)
}

func isArray(v any) bool { _, ok := v.([]any); return ok }

func isObject(v any) bool { _, ok := v.(map[string]any); return ok }

func isStringArray(v any) bool {
	a, ok := v.([]any)
	return ok && !slices.ContainsFunc(a, func(e any) bool { return !isString(e) })
}

func isStringMap(v any) bool {
	m, ok := v.(map[string]any)
	return ok && !slices.ContainsFunc(slices.Collect(maps.Values(m)), func(e any) bool { return !isString(e) })
}

func isOneOf(values ...string) func(any) bool {
	return func(v any) bool { s, ok := v.(string); return ok && slices.Contains(values, s) }
}

// isURL approximates the WHATWG URL parse Claude Code applies: an absolute
// URL with a scheme.
func isURL(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && u.Scheme != ""
}

// hookProblem returns why Claude Code cannot load the hook entry v, or ""
// when it can.
func hookProblem(v any) string {
	h, ok := v.(map[string]any)
	if !ok {
		return "hook entry is not an object"
	}
	typ, ok := h["type"].(string)
	if !ok {
		return `hook entry has no string "type"`
	}
	fields, ok := hookTypes[typ]
	if !ok {
		return fmt.Sprintf("unknown hook type %q", typ)
	}
	for _, key := range slices.Sorted(maps.Keys(h)) {
		f, known := fields[key]
		if !known {
			f, known = commonHookFields[key]
		}
		if known && !f.valid(h[key]) {
			return fmt.Sprintf("%s hook has an invalid %q", typ, key)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if _, present := h[key]; fields[key].required && !present {
			return fmt.Sprintf("%s hook has no %q", typ, key)
		}
	}
	return ""
}

// unloadableHooks returns why Claude Code applies none of a settings file
// whose "hooks" key holds raw (present says the key exists), or "" when the
// hooks do not stop the file loading. It follows Claude Code's own checks:
// a guard-event entry that is not an object, whose "hooks" is not an array,
// whose matcher is not a string or that holds a hook it cannot load (see
// hookProblem); a guard event whose value is not an array; and guard-event
// hooks found where a matcher or event was expected.
func unloadableHooks(raw any, present bool) string {
	if !present {
		return ""
	}
	events, ok := raw.(map[string]any)
	if !ok {
		if list, isList := raw.([]any); isList && slices.ContainsFunc(list, mentionsGuardHooks) {
			return `"hooks" is an array of matchers, not an object mapping events to them`
		}
		return ""
	}
	if matcherLike(events) {
		return `"hooks" is a single matcher, not an object mapping events to matchers`
	}
	for _, event := range slices.Sorted(maps.Keys(events)) {
		v := events[event]
		list, isList := v.([]any)
		switch {
		case !slices.Contains(hookEvents, event):
			if holdsGuardHooks(v, 3, !isList, false, nil) {
				return fmt.Sprintf("hooks.%s is not a hook event but holds %s/%s hooks", event, EventPreToolUse, EventPermissionRequest)
			}
		case !isList:
			if slices.Contains(guardEvents, event) && v != nil || holdsGuardHooks(v, 3, false, false, nil) {
				return fmt.Sprintf("hooks.%s is not an array of matchers", event)
			}
		default:
			for i, entry := range list {
				if p := matcherEntryProblem(entry, slices.Contains(guardEvents, event)); p != "" {
					return "hooks." + event + "." + strconv.Itoa(i) + ": " + p
				}
			}
		}
	}
	return ""
}

// matcherEntryProblem returns why Claude Code cannot load the matcher entry v
// of an event's list, when that refuses the whole file, or "". guard says the
// event is a guard event; for any other event a bad entry is only dropped.
func matcherEntryProblem(v any, guard bool) string {
	if holdsGuardHooks(v, 3, false, false, nil) {
		return fmt.Sprintf("holds %s/%s hooks where a matcher was expected", EventPreToolUse, EventPermissionRequest)
	}
	if !guard {
		return ""
	}
	m, ok := v.(map[string]any)
	if !ok {
		return "matcher entry is not an object"
	}
	hooks, ok := m["hooks"].([]any)
	if !ok {
		return `matcher entry's "hooks" is not an array`
	}
	for i, h := range hooks {
		if p := hookProblem(h); p != "" {
			return "hooks." + strconv.Itoa(i) + ": " + p
		}
	}
	if raw, present := m["matcher"]; present && !isString(raw) {
		return `"matcher" is not a string`
	}
	return ""
}

// hasGuardEventKey reports whether v is an object with a guard-event key
// holding anything but null or an empty array.
func hasGuardEventKey(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	for _, event := range guardEvents {
		if val, present := m[event]; present && val != nil {
			if list, isList := val.([]any); !isList || len(list) > 0 {
				return true
			}
		}
	}
	return false
}

// matcherLike reports whether v is shaped like a matcher entry: an object
// with a non-empty "hooks" array, or with a "hooks" object next to a
// "matcher" key or itself holding a string "type". An object keyed by event
// names without a "matcher" is an event map, not a matcher.
func matcherLike(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	_, hasMatcher := m["matcher"]
	if !hasMatcher && slices.ContainsFunc(slices.Collect(maps.Keys(m)), func(k string) bool { return slices.Contains(hookEvents, k) }) {
		return false
	}
	switch hooks := m["hooks"].(type) {
	case []any:
		return len(hooks) > 0
	case map[string]any:
		return hasMatcher || isString(hooks["type"])
	}
	return false
}

// mentionsGuardHooks reports whether v is matcher-like, holds a guard-event
// key, or is an array with such an element at any depth.
func mentionsGuardHooks(v any) bool {
	if matcherLike(v) || hasGuardEventKey(v) {
		return true
	}
	list, ok := v.([]any)
	return ok && slices.ContainsFunc(list, mentionsGuardHooks)
}

// holdsGuardHooks reports whether v holds guard-event hooks within depth
// levels: an object with a guard-event key, or, while matchers counts, a
// matcher-like object. Nesting under an event-name key stops matchers
// counting, a typed hook entry inside a "hooks" array is not searched, and
// neither is the value of an object key listed in unscanned.
func holdsGuardHooks(v any, depth int, matchers, inHooksList bool, unscanned []string) bool {
	if hasGuardEventKey(v) || matchers && matcherLike(v) {
		return true
	}
	if depth == 0 {
		return false
	}
	switch val := v.(type) {
	case []any:
		return slices.ContainsFunc(val, func(e any) bool { return holdsGuardHooks(e, depth-1, matchers, inHooksList, unscanned) })
	case map[string]any:
		if inHooksList && isString(val["type"]) {
			return false
		}
		for k, e := range val {
			if slices.Contains(unscanned, k) {
				continue
			}
			_, isList := e.([]any)
			if holdsGuardHooks(e, depth-1, matchers && !slices.Contains(hookEvents, k), k == "hooks" && isList, unscanned) {
				return true
			}
		}
	}
	return false
}

// unscannedSettingsKeys are the keys whose values Claude Code never searches
// for guard hooks declared outside "hooks": their values are server, plugin
// or environment maps, not hook configuration.
var unscannedSettingsKeys = []string{
	"mcpServers", "managedMcpServers", "lspServers", "pluginConfigs", "enabledPlugins",
	"extraKnownMarketplaces", "env", "skillOverrides", "modelSettings",
}

// droppedBeforeScanKeys are top-level keys Claude Code removes before that
// search: "isolation" is honoured only from managed settings, and
// "additionalMarketplaces" is an alias it folds into "extraKnownMarketplaces"
// (or drops when that key is set).
var droppedBeforeScanKeys = []string{"isolation", "additionalMarketplaces"}

// permissionRuleKeys are the permissions lists from which Claude Code drops
// every non-string entry before that search, so none of them can hold hooks.
var permissionRuleKeys = []string{"allow", KeyDeny, "ask"}

// guardHooksOutsideHooks returns why Claude Code applies none of the settings
// document root because it declares PreToolUse/PermissionRequest hooks
// somewhere other than "hooks" (at the top level, or within three levels of
// any other key not in unscannedSettingsKeys), or "".
func guardHooksOutsideHooks(root map[string]any) string {
	if hasGuardEventKey(root) {
		return fmt.Sprintf(`%s/%s hooks are declared at the top level, outside "hooks"`, EventPreToolUse, EventPermissionRequest)
	}
	for _, key := range slices.Sorted(maps.Keys(root)) {
		if key == KeyHooks || slices.Contains(unscannedSettingsKeys, key) || slices.Contains(droppedBeforeScanKeys, key) {
			continue
		}
		v := root[key]
		if perms, ok := v.(map[string]any); ok && key == KeyPermissions {
			v = withoutRuleLists(perms)
		}
		if holdsGuardHooks(v, 3, !slices.Contains(hookEvents, key), false, unscannedSettingsKeys) {
			return fmt.Sprintf(`%s holds %s/%s hooks outside "hooks"`, key, EventPreToolUse, EventPermissionRequest)
		}
	}
	return ""
}

// withoutRuleLists returns a copy of perms without the permissionRuleKeys
// lists, which Claude Code strips of every non-string entry.
func withoutRuleLists(perms map[string]any) map[string]any {
	out := maps.Clone(perms)
	for _, k := range permissionRuleKeys {
		if _, isList := out[k].([]any); isList {
			delete(out, k)
		}
	}
	return out
}

// Value schemas of settings keys: what Claude Code's settings schema wants.
var (
	aBool        = typed("a boolean", isBool)
	aString      = typed("a string", isString)
	anArray      = typed("an array", isArray)
	aStringArray = typed("an array of strings", isStringArray)
	anObject     = typed("an object", isObject)
	aPositiveInt = typed("a positive whole number", isPositiveInt)
)

// typed is the schema of an optional key whose value valid accepts, as
// want describes.
func typed(want string, valid func(any) bool) field { return field{valid: valid, want: want} }

// oneOf is the schema of a key whose value must be one of values.
func oneOf(values ...string) field {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	want := quoted[0]
	if len(values) > 1 {
		want = "one of " + strings.Join(quoted, ", ")
	}
	return typed(want, isOneOf(values...))
}

// settingsFields and permissionsFields are the top-level and "permissions"
// keys whose bad value makes Claude Code 2.1.280 refuse the whole settings
// file (project, local and user files alike). They were read from the
// settings schema in that release's bundle: run `strings -n 8` on the
// claude binary, find the object schema beginning `u({$schema:o()` (its
// permissions schema begins `u({allow:C(`, and keys contributed by feature
// modules come from the `zt={autoMode:` table, all built unconditionally),
// and split it into keys at the top nesting level. A key is listed when its
// schema is a plain boolean (H()), string (o()), array of strings (C(o())),
// literal enum (V([...])) or positive integer (k().int().positive()) with
// .optional() and no .catch(), which would drop a bad value instead.
//
// Left out, because a bad value does not refuse the file or because modelling
// it would need more than these checks: keys Claude Code catches (theme,
// effortLevel, ...); keys it deletes or repairs before validating
// (managedMcpServers and isolation, which only managed settings honour;
// extraKnownMarketplaces and its aliases, allowedMcpServers, deniedMcpServers,
// the marketplace lists, modelPicker, modelPricing, crossSessionInbound,
// remoteControl); keys behind a feature flag spread in only when it is on;
// "hooks", which unloadableHooks models; and numeric ranges, records, nested
// objects and unions. Leaving a key out keeps it loadable, as before; listing
// a lenient key would refuse a file Claude Code loads, which only withholds
// credit. Re-extract when the supported Claude Code release changes.
var settingsFields = map[string]field{
	KeyEnv:                        anObject,
	KeyPermissions:                anObject,
	"autoUpdatesChannel":          oneOf("latest", "stable", "rc"),
	"daemonColdStart":             oneOf("transient", "ask"),
	"defaultShell":                oneOf("bash", "powershell"),
	"defaultView":                 oneOf("chat", "transcript"),
	"disableAutoMode":             oneOf("disable"),
	"disableDeepLinkRegistration": oneOf("disable"),
	"feedbackDrafts":              oneOf("notify", "quiet", "off"),
	"managedSourcesBehavior":      oneOf("first-wins", "merge"),
	"parentSettingsBehavior":      oneOf("first-wins", "merge"),
	"totalTokensReminder":         oneOf("off", "infinite", "fixed", "countdown", "padded-countdown"),
	"tui":                         oneOf("default", "fullscreen"),
	"workflowSizeGuideline":       oneOf("unrestricted", "small", "medium", "large"),

	"agentPushNotifEnabled":             aBool,
	"allowAllClaudeAiMcps":              aBool,
	"allowManagedHooksOnly":             aBool,
	"allowManagedMcpServersOnly":        aBool,
	"allowManagedPermissionRulesOnly":   aBool,
	"alwaysThinkingEnabled":             aBool,
	"autoCompactEnabled":                aBool,
	"autoContinueAtUsageLimit":          aBool,
	"autoDreamEnabled":                  aBool,
	"autoMemoryEnabled":                 aBool,
	"autoScrollEnabled":                 aBool,
	"autoUploadSessions":                aBool,
	"awaySummaryEnabled":                aBool,
	"axScreenReader":                    aBool,
	"bashEditDiffEnabled":               aBool,
	"channelsEnabled":                   aBool,
	"disableAgentView":                  aBool,
	KeyDisableAllHooks:                  aBool,
	"disableArtifact":                   aBool,
	"disableBundledSkills":              aBool,
	"disableClaudeAiConnectors":         aBool,
	"disableCommandPluginSources":       aBool,
	"disableRemoteControl":              aBool,
	"disableSkillShellExecution":        aBool,
	"disableWorkflows":                  aBool,
	"doneMeansMerged":                   aBool,
	"emojiCompletionEnabled":            aBool,
	"enableAllProjectMcpServers":        aBool,
	"enableArtifact":                    aBool,
	"enableWorkflows":                   aBool,
	"enforceAvailableModels":            aBool,
	"fastMode":                          aBool,
	"fastModePerSessionOptIn":           aBool,
	"fileCheckpointingEnabled":          aBool,
	"forceRemoteSettingsRefresh":        aBool,
	"includeCoAuthoredBy":               aBool,
	"includeGitInstructions":            aBool,
	"inputNeededNotifEnabled":           aBool,
	"isolatePeerMachines":               aBool,
	"precomputeCompactionEnabled":       aBool,
	"prefersReducedMotion":              aBool,
	"promptSuggestionEnabled":           aBool,
	"remoteControlAtStartup":            aBool,
	"respectGitignore":                  aBool,
	"respondToBashCommands":             aBool,
	"showClearContextOnPlanAccept":      aBool,
	"showMessageTimestamps":             aBool,
	"showThinkingSummaries":             aBool,
	"showTurnDuration":                  aBool,
	"skipAutoPermissionPrompt":          aBool,
	"skipDangerousModePermissionPrompt": aBool,
	"skipWebFetchPreflight":             aBool,
	"skipWorkflowUsageWarning":          aBool,
	"spinnerTipsEnabled":                aBool,
	"switchModelsOnFlag":                aBool,
	"syncClaudeAiPlugins":               aBool,
	"syncClaudeAiSkills":                aBool,
	"syntaxHighlightingDisabled":        aBool,
	"terminalProgressBarEnabled":        aBool,
	"terminalTitleFromRename":           aBool,
	"todoFeatureEnabled":                aBool,
	"totalTokensReminderAfterUserTurn":  aBool,
	"useAutoModeDuringPlan":             aBool,
	"verbose":                           aBool,
	"voiceEnabled":                      aBool,
	"wheelScrollAccelerationEnabled":    aBool,
	"workflowKeywordTriggerEnabled":     aBool,
	"wslInheritsWindowsSettings":        aBool,

	"$schema":                aString,
	"advisorModel":           aString,
	"agent":                  aString,
	"apiKeyHelper":           aString,
	"autoMemoryDirectory":    aString,
	"awsAuthRefresh":         aString,
	"awsCredentialExport":    aString,
	"claudeMd":               aString,
	"gcpAuthRefresh":         aString,
	"language":               aString,
	"minimumVersion":         aString,
	"model":                  aString,
	"otelHeadersHelper":      aString,
	"outputStyle":            aString,
	"plansDirectory":         aString,
	"pluginTrustMessage":     aString,
	"prUrlTemplate":          aString,
	"processWrapper":         aString,
	"proxyAuthHelper":        aString,
	"requiredMaximumVersion": aString,
	"requiredMinimumVersion": aString,
	"timeZone":               aString,

	"allowedHttpHookUrls":          aStringArray,
	"availableModels":              aStringArray,
	"claudeMdExcludes":             aStringArray,
	"companyAnnouncements":         aStringArray,
	"disabledMcpjsonServers":       aStringArray,
	"enabledMcpjsonServers":        aStringArray,
	"fallbackModel":                aStringArray,
	"httpHookAllowedEnvVars":       aStringArray,
	"pluginSuggestionMarketplaces": aStringArray,

	"cleanupPeriodDays":         aPositiveInt,
	"skillListingMaxDescChars":  aPositiveInt,
	"totalTokensReminderBudget": aPositiveInt,
}

// permissionsFields are the "permissions" keys (see settingsFields). The
// rule lists need only be arrays: Claude Code drops each entry that is not a
// valid rule before it validates the file. "manual" is accepted as an alias
// of the "default" mode.
var permissionsFields = map[string]field{
	"allow":                               anArray,
	KeyDeny:                               anArray,
	"ask":                                 anArray,
	KeyDefaultMode:                        oneOf("acceptEdits", "auto", ModeBypassPermissions, "default", "dontAsk", "plan", "manual"),
	KeyDisableBypassPermissionsMode:       oneOf("disable"),
	"disableAutoMode":                     oneOf("disable"),
	"blockReadsOutsideWorkingDirectories": aBool,
	"additionalDirectories":               aStringArray,
}

// settingsSchemaProblem returns why Claude Code's settings schema rejects
// root, so that it applies none of the file, or "". It checks the keys of
// settingsFields and permissionsFields; any other key is accepted.
func settingsSchemaProblem(root map[string]any) string {
	if p := schemaProblem(root, settingsFields, strconv.Quote); p != "" {
		return p
	}
	perms, _ := root[KeyPermissions].(map[string]any)
	return schemaProblem(perms, permissionsFields, func(k string) string { return KeyPermissions + "." + k })
}

// schemaProblem returns why the first key of obj, in sorted order, whose
// value fails its schema in fields does so, naming the key with name, or "".
func schemaProblem(obj map[string]any, fields map[string]field, name func(string) string) string {
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if v, present := obj[key]; present && !fields[key].valid(v) {
			return name(key) + " is not " + fields[key].want
		}
	}
	return ""
}
