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

// JSONAddSettingsEntries adds entries to settings.json permission arrays.
// Deduplicates entries — existing values are not added again.
func JSONAddSettingsEntries(existing []byte, additions SettingsAdditions) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(existing, &doc); err != nil {
		return nil, fmt.Errorf("parsing settings.json: %w", err)
	}

	if len(additions.AllowRules) > 0 {
		updated, err := addToStringArray(doc["permissions.allow"], additions.AllowRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.allow"] = updated
	}

	if len(additions.DenyRules) > 0 {
		updated, err := addToStringArray(doc["permissions.deny"], additions.DenyRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.deny"] = updated
	}

	if len(additions.AskRules) > 0 {
		updated, err := addToStringArray(doc["permissions.ask"], additions.AskRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.ask"] = updated
	}

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}
	return append(result, '\n'), nil
}

// JSONRemoveSettingsEntries removes tool-specific entries from settings.json.
// Only removes exact matches of the declared rules.
func JSONRemoveSettingsEntries(existing []byte, removals SettingsRemovals) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(existing, &doc); err != nil {
		return nil, fmt.Errorf("parsing settings.json: %w", err)
	}

	if len(removals.AllowRules) > 0 {
		updated, err := removeFromStringArray(doc["permissions.allow"], removals.AllowRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.allow"] = updated
	}

	if len(removals.DenyRules) > 0 {
		updated, err := removeFromStringArray(doc["permissions.deny"], removals.DenyRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.deny"] = updated
	}

	if len(removals.AskRules) > 0 {
		updated, err := removeFromStringArray(doc["permissions.ask"], removals.AskRules)
		if err != nil {
			return nil, err
		}
		doc["permissions.ask"] = updated
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
		return raw, nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("parsing string array: %w", err)
	}

	toRemove := make(map[string]bool, len(removals))
	for _, r := range removals {
		toRemove[r] = true
	}

	var result []string
	for _, v := range arr {
		if !toRemove[v] {
			result = append(result, v)
		}
	}

	return json.Marshal(result)
}
