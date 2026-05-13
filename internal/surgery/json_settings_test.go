package surgery

import (
	"encoding/json"
	"testing"
)

func TestJSONAddSettingsEntries_AddsToAllArrays(t *testing.T) {
	existing := []byte(`{}`)
	additions := SettingsAdditions{
		AllowRules: []string{"Bash(npm run *)"},
		DenyRules:  []string{"Bash(rm -rf /*)"},
		AskRules:   []string{"Bash(git push *)"},
	}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(npm run *)"})
	assertStringArray(t, doc["permissions.deny"], []string{"Bash(rm -rf /*)"})
	assertStringArray(t, doc["permissions.ask"], []string{"Bash(git push *)"})
}

func TestJSONAddSettingsEntries_AddsToExistingArrays(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"],
  "permissions.deny": ["Bash(shutdown)"]
}`)
	additions := SettingsAdditions{
		AllowRules: []string{"Bash(npm run *)"},
		DenyRules:  []string{"Bash(rm -rf /*)"},
	}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)", "Bash(npm run *)"})
	assertStringArray(t, doc["permissions.deny"], []string{"Bash(shutdown)", "Bash(rm -rf /*)"})
}

func TestJSONAddSettingsEntries_Deduplicates(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(npm run *)", "Bash(echo *)"]
}`)
	additions := SettingsAdditions{
		AllowRules: []string{"Bash(npm run *)", "Bash(new-command)"},
	}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// "Bash(npm run *)" should not be duplicated.
	assertStringArray(t, doc["permissions.allow"], []string{"Bash(npm run *)", "Bash(echo *)", "Bash(new-command)"})
}

func TestJSONAddSettingsEntries_DeduplicatesMultipleSameRule(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	// Additions list itself has duplicates.
	additions := SettingsAdditions{
		AllowRules: []string{"Bash(new)", "Bash(new)"},
	}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)", "Bash(new)"})
}

func TestJSONAddSettingsEntries_PreservesOtherKeys(t *testing.T) {
	existing := []byte(`{
  "model": "claude-sonnet-4-20250514",
  "permissions.allow": ["Bash(echo *)"]
}`)
	additions := SettingsAdditions{
		AllowRules: []string{"Bash(npm run *)"},
	}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := doc["model"]; !ok {
		t.Error("model key should be preserved")
	}

	var model string
	if err := json.Unmarshal(doc["model"], &model); err != nil {
		t.Fatalf("failed to unmarshal model: %v", err)
	}
	if model != "claude-sonnet-4-20250514" {
		t.Errorf("model value changed: got %q", model)
	}
}

func TestJSONAddSettingsEntries_EmptyAdditions(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	additions := SettingsAdditions{} // No rules to add.

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Existing data should be preserved.
	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)"})
}

func TestJSONAddSettingsEntries_InvalidJSON(t *testing.T) {
	existing := []byte(`{invalid}`)
	additions := SettingsAdditions{AllowRules: []string{"rule"}}

	_, err := JSONAddSettingsEntries(existing, additions)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONAddSettingsEntries_TrailingNewline(t *testing.T) {
	existing := []byte(`{}`)
	additions := SettingsAdditions{AllowRules: []string{"rule"}}

	result, err := JSONAddSettingsEntries(existing, additions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(result)
	if got[len(got)-1] != '\n' {
		t.Error("result should end with a trailing newline")
	}
}

func TestJSONRemoveSettingsEntries_RemovesExactMatches(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(npm run *)", "Bash(echo *)", "Bash(go test *)"],
  "permissions.deny": ["Bash(rm -rf /*)", "Bash(shutdown)"]
}`)
	removals := SettingsRemovals{
		AllowRules: []string{"Bash(npm run *)"},
		DenyRules:  []string{"Bash(rm -rf /*)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)", "Bash(go test *)"})
	assertStringArray(t, doc["permissions.deny"], []string{"Bash(shutdown)"})
}

func TestJSONRemoveSettingsEntries_PreservesNonMatchingEntries(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)", "Bash(npm run *)", "Bash(go test *)"]
}`)
	removals := SettingsRemovals{
		AllowRules: []string{"Bash(npm run *)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)", "Bash(go test *)"})
}

func TestJSONRemoveSettingsEntries_RemoveAll(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	removals := SettingsRemovals{
		AllowRules: []string{"Bash(echo *)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// The key should still exist but with an empty array (null in Go).
	var arr []string
	if err := json.Unmarshal(doc["permissions.allow"], &arr); err != nil {
		t.Fatalf("failed to unmarshal allow array: %v", err)
	}
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %v", arr)
	}
}

func TestJSONRemoveSettingsEntries_NoMatchingRules(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	removals := SettingsRemovals{
		AllowRules: []string{"Bash(nonexistent)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)"})
}

func TestJSONRemoveSettingsEntries_EmptyRemovals(t *testing.T) {
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	removals := SettingsRemovals{}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)"})
}

func TestJSONRemoveSettingsEntries_MissingArrayKey(t *testing.T) {
	// Remove from a key that doesn't exist in the document.
	existing := []byte(`{
  "permissions.allow": ["Bash(echo *)"]
}`)
	removals := SettingsRemovals{
		DenyRules: []string{"Bash(rm *)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Allow should be unchanged.
	assertStringArray(t, doc["permissions.allow"], []string{"Bash(echo *)"})
}

func TestJSONRemoveSettingsEntries_PreservesOtherKeys(t *testing.T) {
	existing := []byte(`{
  "model": "claude-sonnet-4-20250514",
  "permissions.allow": ["Bash(echo *)", "Bash(npm run *)"]
}`)
	removals := SettingsRemovals{
		AllowRules: []string{"Bash(npm run *)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if _, ok := doc["model"]; !ok {
		t.Error("model key should be preserved")
	}
}

func TestJSONRemoveSettingsEntries_InvalidJSON(t *testing.T) {
	existing := []byte(`{invalid}`)
	removals := SettingsRemovals{AllowRules: []string{"rule"}}

	_, err := JSONRemoveSettingsEntries(existing, removals)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONRemoveSettingsEntries_AskRules(t *testing.T) {
	existing := []byte(`{
  "permissions.ask": ["Bash(git push *)", "Bash(git rebase *)"]
}`)
	removals := SettingsRemovals{
		AskRules: []string{"Bash(git push *)"},
	}

	result, err := JSONRemoveSettingsEntries(existing, removals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertStringArray(t, doc["permissions.ask"], []string{"Bash(git rebase *)"})
}

// assertStringArray checks that a JSON raw message contains the expected string array.
func assertStringArray(t *testing.T, raw json.RawMessage, expected []string) {
	t.Helper()
	if raw == nil {
		if len(expected) > 0 {
			t.Errorf("expected %v but key is missing", expected)
		}
		return
	}

	var got []string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("failed to unmarshal array: %v", err)
	}

	if len(got) != len(expected) {
		t.Errorf("expected %d elements %v, got %d elements %v", len(expected), expected, len(got), got)
		return
	}

	for i, v := range expected {
		if got[i] != v {
			t.Errorf("element [%d]: expected %q, got %q", i, v, got[i])
		}
	}
}
