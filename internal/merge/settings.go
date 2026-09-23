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
	Enabled    bool               `json:"enabled"`
	Filesystem *sandboxFilesystem `json:"filesystem,omitempty"`
	Network    *sandboxNetwork    `json:"network,omitempty"`
}

type sandboxFilesystem struct {
	AllowWrite []string `json:"allowWrite,omitempty"`
	DenyWrite  []string `json:"denyWrite,omitempty"`
	DenyRead   []string `json:"denyRead,omitempty"`
}

type sandboxNetwork struct {
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

type hookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

// hookEntry exposes the fields the merge needs to identify a hook while
// keeping the entry's original JSON, so hook types with fields this package
// does not model (e.g. "prompt" hooks, "url"/"headers" on http hooks) are
// written back verbatim instead of being reshaped into a command hook.
type hookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
	URL           string `json:"url,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`

	raw json.RawMessage
}

// hookEntryFields breaks the UnmarshalJSON/MarshalJSON recursion.
type hookEntryFields hookEntry

// UnmarshalJSON decodes the identifying fields and retains the raw entry.
func (h *hookEntry) UnmarshalJSON(data []byte) error {
	var f hookEntryFields
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing hook entry: %w", err)
	}
	*h = hookEntry(f)
	h.raw = append(json.RawMessage(nil), data...)
	return nil
}

// MarshalJSON emits the original entry JSON when available.
func (h hookEntry) MarshalJSON() ([]byte, error) {
	if len(h.raw) > 0 {
		return h.raw, nil
	}
	return json.Marshal(hookEntryFields(h))
}

// key identifies a hook by what it runs, so a generated hook whose options
// (e.g. timeout) changed is still recognised as the same hook.
func (h hookEntry) key() string {
	return h.Type + "\x00" + h.Command + "\x00" + h.Prompt + "\x00" + h.URL
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
	result.Sandbox = mergeSandbox(baseParsed.Sandbox, theirsParsed.Sandbox, oursParsed.Sandbox)

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

	// The typed "permissions" object models only allow/deny/ask and the policy
	// fields, so overlaying it wholesale drops unmodeled nested children (e.g.
	// permissions.additionalDirectories). Deep-merge the typed permissions back
	// over theirs' raw permissions so those survive. Modeled arrays (allow/deny)
	// are always emitted by the typed result and therefore remain authoritative.
	if permMerged, err := deepMergeObject("permissions", theirsRaw["permissions"], typedRaw["permissions"]); err != nil {
		return nil, err
	} else if permMerged != nil {
		merged["permissions"] = permMerged
	}

	// Likewise "sandbox": its unmodeled children (e.g. excludedCommands,
	// network.allowUnixSockets) must survive, while the modeled fields are
	// decided entirely by the typed result.
	sandboxMerged, err := mergeSandboxRaw(theirsRaw["sandbox"], result.Sandbox)
	if err != nil {
		return nil, err
	}
	if sandboxMerged == nil {
		delete(merged, "sandbox")
	} else {
		merged["sandbox"] = sandboxMerged
	}

	// Remove hooks when the merged result has none.
	if result.Hooks == nil {
		delete(merged, "hooks")
	}

	out, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshaling final settings: %w", err)
	}
	return CanonicalJSON(out)
}

// deepMergeObject deep-merges the typed (merged) object for the top-level key
// over theirs' raw object so unknown nested keys survive. It returns nil when
// there is nothing typed to overlay (leaving any theirs value untouched).
func deepMergeObject(key string, theirsRaw, typedRaw json.RawMessage) (json.RawMessage, error) {
	if len(typedRaw) == 0 {
		return nil, nil
	}
	var theirsObj, typedObj map[string]any
	if len(theirsRaw) > 0 {
		if err := json.Unmarshal(theirsRaw, &theirsObj); err != nil {
			return nil, fmt.Errorf("parsing theirs %s: %w", key, err)
		}
	}
	if err := json.Unmarshal(typedRaw, &typedObj); err != nil {
		return nil, fmt.Errorf("parsing merged %s: %w", key, err)
	}
	out, err := json.Marshal(DeepMergeJSON(theirsObj, typedObj))
	if err != nil {
		return nil, fmt.Errorf("marshaling merged %s: %w", key, err)
	}
	return out, nil
}

// mergeHooks performs a three-way merge of hook maps. Generated matchers and
// entries come from ours; for every matcher in theirs, the entries the user
// added (theirs minus base, by hook identity) are kept, whether they sit under
// a user-added matcher or under a generated one.
func mergeHooks(base, theirs, ours map[string][]hookMatcher) map[string][]hookMatcher {
	if len(ours) == 0 && len(theirs) == 0 {
		return nil
	}

	result := make(map[string][]hookMatcher)

	// Start with all hooks from ours (copied so appends never alias ours).
	for event, matchers := range ours {
		for _, m := range matchers {
			result[event] = append(result[event], hookMatcher{
				Matcher: m.Matcher,
				Hooks:   append([]hookEntry(nil), m.Hooks...),
			})
		}
	}

	for event, theirsMatchers := range theirs {
		for _, tm := range theirsMatchers {
			userEntries := tm.Hooks
			if bm, inBase := findMatcher(base[event], tm.Matcher); inBase {
				// Entries generated last time are owned by the generator: ours
				// decides whether they stay.
				userEntries = entriesNotIn(tm.Hooks, bm.Hooks)
				if len(userEntries) == 0 {
					continue
				}
			}

			idx := indexMatcher(result[event], tm.Matcher)
			if idx < 0 {
				result[event] = append(result[event], hookMatcher{
					Matcher: tm.Matcher,
					Hooks:   append([]hookEntry(nil), userEntries...),
				})
				continue
			}
			for _, e := range userEntries {
				if !containsEntry(result[event][idx].Hooks, e) {
					result[event][idx].Hooks = append(result[event][idx].Hooks, e)
				}
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// entriesNotIn returns the entries of a whose identity is not present in b.
func entriesNotIn(a, b []hookEntry) []hookEntry {
	var out []hookEntry
	for _, e := range a {
		if !containsEntry(b, e) {
			out = append(out, e)
		}
	}
	return out
}

// containsEntry reports whether entries holds a hook with e's identity.
func containsEntry(entries []hookEntry, e hookEntry) bool {
	for _, x := range entries {
		if x.key() == e.key() {
			return true
		}
	}
	return false
}

// mergeSandbox merges sandbox configs. Deny arrays are unioned so neither a
// generated nor a user deny is ever lost. Allow arrays use the permissions
// algorithm — ours plus user additions (theirs minus base) — so an allow
// entry the generator stops emitting is removed rather than kept forever.
func mergeSandbox(base, theirs, ours *sandboxConfig) *sandboxConfig {
	if theirs == nil && ours == nil {
		return nil
	}
	if theirs == nil {
		return ours
	}
	var b, o sandboxConfig
	if base != nil {
		b = *base
	}
	if ours != nil {
		o = *ours
	}
	result := &sandboxConfig{
		// A generated "enabled" the generator no longer emits is dropped; one
		// the user set is kept.
		Enabled: o.Enabled || (theirs.Enabled && !b.Enabled),
	}
	if o.Filesystem != nil || theirs.Filesystem != nil {
		of, tf, bf := derefOr(o.Filesystem), derefOr(theirs.Filesystem), derefOr(b.Filesystem)
		result.Filesystem = &sandboxFilesystem{
			// Allow lists keep only the user's additions (theirs minus base),
			// so an allow entry the generator dropped is really removed.
			AllowWrite: unionStrings(of.AllowWrite, diffStrings(tf.AllowWrite, bf.AllowWrite)),
			// Deny lists never lose an entry, generated or user-added.
			DenyWrite: unionStrings(of.DenyWrite, tf.DenyWrite),
			DenyRead:  unionStrings(of.DenyRead, tf.DenyRead),
		}
	}
	if o.Network != nil || theirs.Network != nil {
		on, tn, bn := derefOr(o.Network), derefOr(theirs.Network), derefOr(b.Network)
		result.Network = &sandboxNetwork{
			AllowedDomains: unionStrings(on.AllowedDomains, diffStrings(tn.AllowedDomains, bn.AllowedDomains)),
		}
	}
	if !result.Enabled && result.Filesystem.empty() && result.Network.empty() {
		return nil
	}
	return result
}

// empty reports whether f holds no paths.
func (f *sandboxFilesystem) empty() bool {
	return f == nil || (len(f.AllowWrite) == 0 && len(f.DenyWrite) == 0 && len(f.DenyRead) == 0)
}

// empty reports whether n allows no domains.
func (n *sandboxNetwork) empty() bool {
	return n == nil || len(n.AllowedDomains) == 0
}

// derefOr returns *p, or the zero value when p is nil.
func derefOr[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// sandboxModeledKeys are the sandbox fields decided by mergeSandbox, as paths
// into the sandbox object.
var sandboxModeledKeys = [][]string{
	{"enabled"},
	{"filesystem", "allowWrite"}, {"filesystem", "denyWrite"}, {"filesystem", "denyRead"},
	{"network", "allowedDomains"},
}

// mergeSandboxRaw overlays the merged typed sandbox onto theirs' raw sandbox
// object so keys this package does not model (e.g. excludedCommands,
// network.allowUnixSockets) survive, while modeled fields are decided entirely
// by the typed result (a field it dropped is removed, not resurrected from
// theirs). It returns nil when the result has no keys at all.
func mergeSandboxRaw(theirsRaw json.RawMessage, typed *sandboxConfig) (json.RawMessage, error) {
	out := map[string]any{}
	if len(theirsRaw) > 0 && string(theirsRaw) != "null" {
		if err := json.Unmarshal(theirsRaw, &out); err != nil {
			return nil, fmt.Errorf("parsing theirs sandbox: %w", err)
		}
	}
	for _, keyPath := range sandboxModeledKeys {
		deleteNested(out, keyPath)
	}
	if typed != nil {
		typedBytes, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshaling merged sandbox: %w", err)
		}
		var typedMap map[string]any
		if err := json.Unmarshal(typedBytes, &typedMap); err != nil {
			return nil, fmt.Errorf("re-parsing merged sandbox: %w", err)
		}
		out = DeepMergeJSON(out, typedMap)
	}
	for _, key := range []string{"filesystem", "network"} {
		if child, ok := out[key].(map[string]any); ok && len(child) == 0 {
			delete(out, key)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	merged, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged sandbox: %w", err)
	}
	return merged, nil
}

// deleteNested removes the value at keyPath from m, if present.
func deleteNested(m map[string]any, keyPath []string) {
	for _, k := range keyPath[:len(keyPath)-1] {
		child, ok := m[k].(map[string]any)
		if !ok {
			return
		}
		m = child
	}
	delete(m, keyPath[len(keyPath)-1])
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

// indexMatcher returns the index of the matcher with the given string, or -1.
func indexMatcher(matchers []hookMatcher, matcher string) int {
	for i, m := range matchers {
		if m.Matcher == matcher {
			return i
		}
	}
	return -1
}
