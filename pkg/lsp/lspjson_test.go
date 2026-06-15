package lsp

import (
	"reflect"
	"testing"
)

func TestExtensionToLanguage(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("go")
	if !ok {
		t.Fatal("ByEcosystem(\"go\") not found")
	}

	got := cfg.ExtensionToLanguage()
	want := map[string]string{".go": "go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtensionToLanguage() = %v, want %v", got, want)
	}
}

func TestExtensionToLanguageMultiExt(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("javascript")
	if !ok {
		t.Fatal("ByEcosystem(\"javascript\") not found")
	}

	got := cfg.ExtensionToLanguage()
	for _, ext := range cfg.Extensions {
		if got[ext] != "typescript" {
			t.Errorf("ExtensionToLanguage()[%q] = %q, want %q", ext, got[ext], "typescript")
		}
	}
	if len(got) != len(cfg.Extensions) {
		t.Errorf("ExtensionToLanguage() has %d entries, want %d", len(got), len(cfg.Extensions))
	}
}

func TestRuleGlobs(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("go")
	if !ok {
		t.Fatal("ByEcosystem(\"go\") not found")
	}

	globs := cfg.RuleGlobs()
	want := []string{"**/*.go", "**/go.mod", "**/go.sum"}
	if !reflect.DeepEqual(globs, want) {
		t.Errorf("RuleGlobs() = %v, want %v", globs, want)
	}
}

func TestRuleGlobsDeterministic(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("rust")
	if !ok {
		t.Fatal("ByEcosystem(\"rust\") not found")
	}

	first := cfg.RuleGlobs()
	second := cfg.RuleGlobs()
	if !reflect.DeepEqual(first, second) {
		t.Errorf("RuleGlobs() not deterministic: %v vs %v", first, second)
	}
	// Extension glob first (sorted), then rule patterns in order.
	want := []string{"**/*.rs", "**/Cargo.toml", "**/Cargo.lock"}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("RuleGlobs() = %v, want %v", first, want)
	}
}

func TestRuleGlobsNoRulePatterns(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	cfg, ok := r.ByEcosystem("lua")
	if !ok {
		t.Fatal("ByEcosystem(\"lua\") not found")
	}

	globs := cfg.RuleGlobs()
	want := []string{"**/*.lua"}
	if !reflect.DeepEqual(globs, want) {
		t.Errorf("RuleGlobs() = %v, want %v", globs, want)
	}
}
