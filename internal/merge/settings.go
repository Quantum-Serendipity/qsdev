package merge

import (
	"encoding/json"
	"fmt"
)

// settingsJSON mirrors the claudecode.SettingsJSON structure.
type settingsJSON struct {
	Permissions permissions              `json:"permissions"`
	Sandbox     *sandboxConfig           `json:"sandbox,omitempty"`
	Hooks       map[string][]hookMatcher `json:"hooks,omitempty"`
}

type permissions struct {
	DefaultMode                  string   `json:"defaultMode,omitempty"`
	DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
	Allow                        []string `json:"allow"`
	Deny                         []string `json:"deny"`
	Ask                          []string `json:"ask,omitempty"`
}

type sandboxConfig struct {
	WriteDeny  []string `json:"writeDeny,omitempty"`
	WriteAllow []string `json:"writeAllow,omitempty"`
	ReadDeny   []string `json:"readDeny,omitempty"`
	NetAllow   []string `json:"netAllow,omitempty"`
}

type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

type hookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// MergeSettings performs a three-way merge of settings.json content.
// base: original generated content (from last generation — may be nil for first update)
// theirs: current on-disk content (may have user modifications)
// ours: newly generated content
func MergeSettings(base, theirs, ours []byte) ([]byte, error) {
	var baseParsed, theirsParsed, oursParsed settingsJSON

	if len(base) > 0 {
		if err := json.Unmarshal(base, &baseParsed); err != nil {
			return nil, fmt.Errorf("parsing base settings: %w", err)
		}
	}
	if len(theirs) == 0 {
		return nil, fmt.Errorf("current settings.json is empty; cannot merge")
	}
	if err := json.Unmarshal(theirs, &theirsParsed); err != nil {
		return nil, fmt.Errorf("parsing theirs settings: %w", err)
	}
	if err := json.Unmarshal(ours, &oursParsed); err != nil {
		return nil, fmt.Errorf("parsing ours settings: %w", err)
	}

	// Capture extra top-level keys from theirs by unmarshaling into a raw map.
	var theirsRaw map[string]json.RawMessage
	if err := json.Unmarshal(theirs, &theirsRaw); err != nil {
		return nil, fmt.Errorf("parsing theirs settings (raw): %w", err)
	}

	var result settingsJSON

	// permissions.allow: union of ours + user-added (theirs minus base).
	userAddedAllow := diffStrings(theirsParsed.Permissions.Allow, baseParsed.Permissions.Allow)
	result.Permissions.Allow = unionStrings(oursParsed.Permissions.Allow, userAddedAllow)

	// permissions.deny: same algorithm.
	userAddedDeny := diffStrings(theirsParsed.Permissions.Deny, baseParsed.Permissions.Deny)
	result.Permissions.Deny = unionStrings(oursParsed.Permissions.Deny, userAddedDeny)

	// permissions.ask: same algorithm.
	userAddedAsk := diffStrings(theirsParsed.Permissions.Ask, baseParsed.Permissions.Ask)
	result.Permissions.Ask = unionStrings(oursParsed.Permissions.Ask, userAddedAsk)
	if len(result.Permissions.Ask) == 0 {
		result.Permissions.Ask = nil
	}

	// Policy fields always from ours.
	result.Permissions.DefaultMode = oursParsed.Permissions.DefaultMode
	result.Permissions.DisableBypassPermissionsMode = oursParsed.Permissions.DisableBypassPermissionsMode

	// Hooks merge.
	result.Hooks = mergeHooks(baseParsed.Hooks, theirsParsed.Hooks, oursParsed.Hooks)

	// Sandbox merge.
	result.Sandbox = mergeSandbox(theirsParsed.Sandbox, oursParsed.Sandbox)

	// Marshal the typed result.
	typedBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged settings: %w", err)
	}

	// Overlay typed result onto theirs raw map to preserve unknown keys.
	var typedRaw map[string]json.RawMessage
	if err := json.Unmarshal(typedBytes, &typedRaw); err != nil {
		return nil, fmt.Errorf("re-parsing merged settings: %w", err)
	}

	// Start with extra keys from theirs, then overlay our merged keys.
	merged := make(map[string]json.RawMessage)
	for k, v := range theirsRaw {
		merged[k] = v
	}
	for k, v := range typedRaw {
		merged[k] = v
	}

	// Remove keys that are zero-valued in the typed result but present from theirs.
	// Specifically, if hooks is empty/null in the typed result, remove it.
	if result.Hooks == nil {
		delete(merged, "hooks")
	}
	if result.Sandbox == nil {
		delete(merged, "sandbox")
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling final settings: %w", err)
	}
	return append(out, '\n'), nil
}

// mergeHooks performs a three-way merge of hook maps.
func mergeHooks(base, theirs, ours map[string][]hookMatcher) map[string][]hookMatcher {
	if len(ours) == 0 && len(theirs) == 0 {
		return nil
	}

	result := make(map[string][]hookMatcher)

	// Start with all hooks from ours.
	for event, matchers := range ours {
		result[event] = append([]hookMatcher(nil), matchers...)
	}

	// For each event in theirs, check for user-added matchers.
	for event, theirsMatchers := range theirs {
		baseMatchers := base[event]
		for _, tm := range theirsMatchers {
			_, inBase := findMatcher(baseMatchers, tm.Matcher)
			_, inOurs := findMatcher(ours[event], tm.Matcher)
			if !inBase {
				// User-added matcher — add if not already present.
				if _, alreadyInResult := findMatcher(result[event], tm.Matcher); !alreadyInResult {
					result[event] = append(result[event], tm)
				}
			}
			// If in base and in ours → ours version already in result (updated by generator).
			// If in base but NOT in ours → generator removed it → don't add.
			_ = inOurs // clarity: we rely on ours being the starting point
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// mergeSandbox merges two sandbox configs by unioning all arrays.
func mergeSandbox(theirs, ours *sandboxConfig) *sandboxConfig {
	if theirs == nil && ours == nil {
		return nil
	}
	if theirs == nil {
		return ours
	}
	if ours == nil {
		return theirs
	}
	return &sandboxConfig{
		WriteDeny:  unionStrings(ours.WriteDeny, theirs.WriteDeny),
		WriteAllow: unionStrings(ours.WriteAllow, theirs.WriteAllow),
		ReadDeny:   unionStrings(ours.ReadDeny, theirs.ReadDeny),
		NetAllow:   unionStrings(ours.NetAllow, theirs.NetAllow),
	}
}

// findMatcher searches for a hookMatcher by matcher string in a slice.
func findMatcher(matchers []hookMatcher, matcher string) (hookMatcher, bool) {
	for _, m := range matchers {
		if m.Matcher == matcher {
			return m, true
		}
	}
	return hookMatcher{}, false
}

