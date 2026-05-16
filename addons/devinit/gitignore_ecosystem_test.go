package devinit

import (
	"testing"
)

func TestGitignoreEntriesForLanguages_JavaScript(t *testing.T) {
	entries := gitignoreEntriesForLanguages([]string{"javascript"})

	want := map[string]bool{
		"node_modules/": true,
		"dist/":         true,
		".env":          true,
		".env.*":        true,
		"*.pem":         true,
		"*.key":         true,
	}

	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %v", len(entries), len(want), entries)
	}

	for _, e := range entries {
		if !want[e] {
			t.Errorf("unexpected entry %q", e)
		}
	}
}

func TestGitignoreEntriesForLanguages_MultipleLanguages(t *testing.T) {
	entries := gitignoreEntriesForLanguages([]string{"javascript", "python"})

	required := []string{"node_modules/", "dist/", "__pycache__/", ".venv/", ".env", "*.pem"}
	entrySet := make(map[string]bool)
	for _, e := range entries {
		entrySet[e] = true
	}

	for _, r := range required {
		if !entrySet[r] {
			t.Errorf("missing required entry %q", r)
		}
	}
}

func TestGitignoreEntriesForLanguages_NoDuplicates(t *testing.T) {
	entries := gitignoreEntriesForLanguages([]string{"javascript", "python"})

	seen := make(map[string]int)
	for _, e := range entries {
		seen[e]++
		if seen[e] > 1 {
			t.Errorf("duplicate entry %q", e)
		}
	}
}

func TestGitignoreEntriesForLanguages_UnknownLanguage(t *testing.T) {
	entries := gitignoreEntriesForLanguages([]string{"cobol"})

	// Should still get security entries.
	entrySet := make(map[string]bool)
	for _, e := range entries {
		entrySet[e] = true
	}

	if !entrySet[".env"] {
		t.Error("missing .env for unknown language")
	}
	if !entrySet["*.pem"] {
		t.Error("missing *.pem for unknown language")
	}
}

func TestGitignoreEntriesForLanguages_Empty(t *testing.T) {
	entries := gitignoreEntriesForLanguages(nil)

	// Should still get security entries.
	if len(entries) != len(securityGitignoreEntries) {
		t.Errorf("got %d entries for nil languages, want %d security entries",
			len(entries), len(securityGitignoreEntries))
	}
}
