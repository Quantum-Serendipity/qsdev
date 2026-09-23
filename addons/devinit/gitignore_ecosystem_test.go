package devinit

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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

// TestEcosystemGitignoreEntries_KeysAreRegisteredModules guards against keys
// that never match: entries are looked up by WizardAnswers.Languages[].Name,
// which is the ecosystem module's Name() (e.g. "go", not "golang").
func TestEcosystemGitignoreEntries_KeysAreRegisteredModules(t *testing.T) {
	t.Parallel()
	registry := ecosystem.DefaultRegistry()
	for key := range ecosystemGitignoreEntries {
		if _, ok := registry.ByName(key); !ok {
			t.Errorf("ecosystemGitignoreEntries key %q is not a registered ecosystem module name (registered: %v)",
				key, registry.Names())
		}
	}
}

// TestGitignoreEntriesForLanguages_EcosystemSpecific checks that registered
// module names pick up their ecosystem entries and that entries which would
// ignore files an ecosystem needs committed are absent or re-included.
func TestGitignoreEntriesForLanguages_EcosystemSpecific(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		lang    string
		want    []string
		notWant []string
		// ordered pairs: first must appear before second.
		before [][2]string
	}{
		{
			name:    "go uses module name and keeps vendor committed",
			lang:    "go",
			want:    []string{"*.exe"},
			notWant: []string{"vendor/"},
		},
		{
			name:    "terraform ignores state, variables and cache but keeps the lock file",
			lang:    "terraform",
			want:    []string{".terraform/", "*.tfstate", "*.tfstate.*", "*.tfvars", "*.tfvars.json", "crash.log"},
			notWant: []string{".terraform.lock.hcl", "*.hcl"},
		},
		{
			name: "java re-includes build wrapper jars after *.jar",
			lang: "java",
			want: []string{"*.jar", "!gradle/wrapper/gradle-wrapper.jar", "!.mvn/wrapper/maven-wrapper.jar"},
			before: [][2]string{
				{"*.jar", "!gradle/wrapper/gradle-wrapper.jar"},
				{"*.jar", "!.mvn/wrapper/maven-wrapper.jar"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entries := gitignoreEntriesForLanguages([]string{tt.lang})
			index := make(map[string]int, len(entries))
			for i, e := range entries {
				index[e] = i
			}
			for _, w := range tt.want {
				if _, ok := index[w]; !ok {
					t.Errorf("missing entry %q in %v", w, entries)
				}
			}
			for _, nw := range tt.notWant {
				if _, ok := index[nw]; ok {
					t.Errorf("unexpected entry %q in %v", nw, entries)
				}
			}
			for _, pair := range tt.before {
				if index[pair[0]] >= index[pair[1]] {
					t.Errorf("%q must precede %q in %v", pair[0], pair[1], entries)
				}
			}
		})
	}
}
