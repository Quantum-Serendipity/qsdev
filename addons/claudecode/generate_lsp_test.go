package claudecode

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/lsp"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// findGeneratedFile returns the GeneratedFile with the given path, or fails.
func findGeneratedFile(t *testing.T, files []types.GeneratedFile, path string) types.GeneratedFile {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("no generated file at path %q", path)
	return types.GeneratedFile{}
}

// lspEntries unmarshals the .lsp.json content into its language-keyed map.
func lspEntries(t *testing.T, files []types.GeneratedFile) map[string]lspJSONEntry {
	t.Helper()
	f := findGeneratedFile(t, files, lspJSONPath)
	var entries map[string]lspJSONEntry
	if err := json.Unmarshal(f.Content, &entries); err != nil {
		t.Fatalf("unmarshaling .lsp.json: %v", err)
	}
	return entries
}

func languages(names ...string) []types.LanguageChoice {
	out := make([]types.LanguageChoice, 0, len(names))
	for _, n := range names {
		out = append(out, types.LanguageChoice{Name: n})
	}
	return out
}

func TestGenerateLspPlugin_GoProject(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{Languages: languages("go")}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	entries := lspEntries(t, files)
	goEntry, ok := entries["go"]
	if !ok {
		t.Fatalf("missing \"go\" entry; got keys %v", keysOf(entries))
	}
	if _, ok := entries["nix"]; !ok {
		t.Fatalf("missing \"nix\" entry; got keys %v", keysOf(entries))
	}
	if goEntry.Command != "gopls" {
		t.Errorf("go command = %q, want %q", goEntry.Command, "gopls")
	}
	if goEntry.ExtensionToLanguage[".go"] != "go" {
		t.Errorf("extensionToLanguage[.go] = %q, want %q", goEntry.ExtensionToLanguage[".go"], "go")
	}

	lspFile := findGeneratedFile(t, files, lspJSONPath)
	if bytes.Contains(lspFile.Content, []byte("filePatterns")) {
		t.Errorf(".lsp.json must not contain \"filePatterns\":\n%s", lspFile.Content)
	}
}

func TestGenerateLspPlugin_Polyglot(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{Languages: languages("go", "javascript", "python", "rust")}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	entries := lspEntries(t, files)
	// go + typescript + python + rust + nix = 5 keys.
	if len(entries) != 5 {
		t.Errorf("entry count = %d, want 5; keys %v", len(entries), keysOf(entries))
	}
	for _, want := range []string{"go", "typescript", "python", "rust", "nix"} {
		if _, ok := entries[want]; !ok {
			t.Errorf("missing %q entry; got keys %v", want, keysOf(entries))
		}
	}
}

func TestGenerateLspPlugin_NixdAlways(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	entries := lspEntries(t, files)
	if len(entries) != 1 {
		t.Errorf("entry count = %d, want 1; keys %v", len(entries), keysOf(entries))
	}
	if _, ok := entries["nix"]; !ok {
		t.Errorf("missing \"nix\" entry; got keys %v", keysOf(entries))
	}
}

func TestGenerateLspPlugin_OptInExcluded(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{Languages: languages("kotlin")}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	entries := lspEntries(t, files)
	if _, ok := entries["kotlin"]; ok {
		t.Errorf("kotlin is default-off and must not be auto-included; keys %v", keysOf(entries))
	}
	if _, ok := entries["nix"]; !ok {
		t.Errorf("missing \"nix\" entry; got keys %v", keysOf(entries))
	}
}

func TestGenerateLspPlugin_Schema(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{Languages: languages("go", "python")}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	entries := lspEntries(t, files)
	for id, entry := range entries {
		if entry.Command == "" {
			t.Errorf("entry %q has empty command", id)
		}
		if len(entry.ExtensionToLanguage) == 0 {
			t.Errorf("entry %q has empty extensionToLanguage", id)
		}
	}

	lspFile := findGeneratedFile(t, files, lspJSONPath)
	if bytes.Contains(lspFile.Content, []byte("filePatterns")) {
		t.Errorf(".lsp.json must not contain \"filePatterns\":\n%s", lspFile.Content)
	}
}

func TestGenerateLspPlugin_Files(t *testing.T) {
	t.Parallel()
	reg := lsp.NewRegistry()
	answers := types.WizardAnswers{Languages: languages("go")}

	files, err := GenerateLspPlugin(answers, reg)
	if err != nil {
		t.Fatalf("GenerateLspPlugin: %v", err)
	}

	manifest := findGeneratedFile(t, files, ".claude/skills/qsdev-lsp/.claude-plugin/plugin.json")
	lspFile := findGeneratedFile(t, files, ".claude/skills/qsdev-lsp/.lsp.json")

	for _, f := range []types.GeneratedFile{manifest, lspFile} {
		if f.Owner != "lsp-config" {
			t.Errorf("file %q Owner = %q, want %q", f.Path, f.Owner, "lsp-config")
		}
		if f.Strategy != types.Overwrite {
			t.Errorf("file %q Strategy = %v, want %v", f.Path, f.Strategy, types.Overwrite)
		}
	}

	var m lspPluginManifest
	if err := json.Unmarshal(manifest.Content, &m); err != nil {
		t.Fatalf("unmarshaling plugin.json: %v", err)
	}
	if m.Name != "qsdev-lsp" {
		t.Errorf("manifest name = %q, want %q", m.Name, "qsdev-lsp")
	}
}

// keysOf returns the keys of an entry map for diagnostic messages.
func keysOf(m map[string]lspJSONEntry) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
