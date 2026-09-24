package surgery

import (
	"encoding/json"
	"fmt"
)

// SettingsAdditions describes what a tool wants to add to settings.json.
type SettingsAdditions struct {
	AllowRules []string
	DenyRules  []string
	AskRules   []string
}

// SettingsRemovals describes what to remove when disabling a tool.
type SettingsRemovals struct {
	AllowRules []string
	DenyRules  []string
	AskRules   []string
}

// JSONAddSettingsEntries adds entries to the nested permissions.allow,
// permissions.deny and permissions.ask arrays of settings.json (the shape
// Claude Code reads). Deduplicates entries — existing values are not added
// again. Other keys, at the top level and inside "permissions", are preserved.
func JSONAddSettingsEntries(existing []byte, additions SettingsAdditions) ([]byte, error) {
	return editPermissionArrays(existing, map[string][]string{
		"allow": additions.AllowRules,
		"deny":  additions.DenyRules,
		"ask":   additions.AskRules,
	}, addToStringArray)
}

// JSONRemoveSettingsEntries removes tool-specific entries from the nested
// permissions arrays of settings.json. Only removes exact matches of the
// declared rules.
func JSONRemoveSettingsEntries(existing []byte, removals SettingsRemovals) ([]byte, error) {
	return editPermissionArrays(existing, map[string][]string{
		"allow": removals.AllowRules,
		"deny":  removals.DenyRules,
		"ask":   removals.AskRules,
	}, removeFromStringArray)
}

// editPermissionArrays applies edit to each named array inside the nested
// "permissions" object, skipping arrays with no rules. The permissions object
// is only created when an edit actually needs to write into it.
func editPermissionArrays(
	existing []byte,
	rules map[string][]string,
	edit func(raw json.RawMessage, rules []string) (json.RawMessage, error),
) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(existing, &doc); err != nil {
		return nil, fmt.Errorf("parsing settings.json: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("parsing settings.json: top-level value is not an object")
	}

	var perms map[string]json.RawMessage
	if raw, ok := doc["permissions"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &perms); err != nil {
			return nil, fmt.Errorf("parsing settings.json permissions: %w", err)
		}
	}

	changed := false
	for _, key := range []string{"allow", "deny", "ask"} {
		if len(rules[key]) == 0 {
			continue
		}
		raw, present := perms[key]
		updated, err := edit(raw, rules[key])
		if err != nil {
			return nil, fmt.Errorf("editing permissions.%s: %w", key, err)
		}
		if updated == nil && !present {
			continue // nothing to remove from an absent array
		}
		if perms == nil {
			perms = make(map[string]json.RawMessage)
		}
		perms[key] = updated
		changed = true
	}

	if changed {
		permsRaw, err := json.Marshal(perms)
		if err != nil {
			return nil, fmt.Errorf("marshaling settings.json permissions: %w", err)
		}
		doc["permissions"] = permsRaw
	}

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}
	return append(result, '\n'), nil
}

func addToStringArray(raw json.RawMessage, additions []string) (json.RawMessage, error) {
	var arr []string
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("parsing string array: %w", err)
		}
	}

	existing := make(map[string]bool, len(arr))
	for _, v := range arr {
		existing[v] = true
	}

	for _, a := range additions {
		if !existing[a] {
			arr = append(arr, a)
			existing[a] = true
		}
	}

	return json.Marshal(arr)
}

func removeFromStringArray(raw json.RawMessage, removals []string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("parsing string array: %w", err)
	}

	toRemove := make(map[string]bool, len(removals))
	for _, r := range removals {
		toRemove[r] = true
	}

	result := []string{}
	for _, v := range arr {
		if !toRemove[v] {
			result = append(result, v)
		}
	}

	return json.Marshal(result)
}
