package claudecode

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// integrationRegistry builds an ecosystem registry containing a minimal mock
// module for each named language. The full Generate() pipeline tolerates
// languages that are absent from the registry (GenerateClaudeMd skips them),
// but registering matching mocks mirrors a real detected project.
func integrationRegistry(t *testing.T, names ...string) *ecosystem.Registry {
	t.Helper()
	reg := ecosystem.NewRegistry()
	for _, n := range names {
		mod := &ecosystem.MockModule{NameVal: n, DisplayNameVal: n, TierVal: 1}
		if err := reg.Register(mod); err != nil {
			t.Fatalf("registering mock %q: %v", n, err)
		}
	}
	return reg
}

// standardAnswers builds wizard answers that resolve to the Standard tier (the
// gate behind which the LSP plugin and rules are generated). The Tier field is
// the explicit selector consumed by tier.Resolve.
func standardAnswers(langs ...string) types.WizardAnswers {
	return types.WizardAnswers{
		ProjectName: "lsp-itest",
		Tier:        "standard",
		Languages:   languages(langs...),
	}
}

// generateStandard runs the full ClaudeCodeGenerator.Generate at Standard tier
// for the given languages and returns the generated files.
func generateStandard(t *testing.T, langs ...string) []types.GeneratedFile {
	t.Helper()
	reg := integrationRegistry(t, langs...)
	gen := NewClaudeCodeGenerator(reg, Config{})
	files, err := gen.Generate(standardAnswers(langs...))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return files
}

// filePaths collects all generated file paths into a set.
func filePaths(files []types.GeneratedFile) map[string]bool {
	set := make(map[string]bool, len(files))
	for _, f := range files {
		set[f.Path] = true
	}
	return set
}

// requirePresent fails if any of the given paths is missing from files.
func requirePresent(t *testing.T, files []types.GeneratedFile, paths ...string) {
	t.Helper()
	set := filePaths(files)
	for _, p := range paths {
		if !set[p] {
			t.Errorf("expected generated file %q to be present", p)
		}
	}
}

// requireAbsent fails if any of the given paths is present in files.
func requireAbsent(t *testing.T, files []types.GeneratedFile, paths ...string) {
	t.Helper()
	set := filePaths(files)
	for _, p := range paths {
		if set[p] {
			t.Errorf("expected generated file %q to be absent", p)
		}
	}
}

const settingsJSONPath = ".claude/settings.json"

// settingsHasGrepPreToolUse parses the generated .claude/settings.json and
// reports whether its PreToolUse hooks contain a matcher equal to "Grep".
func settingsHasGrepPreToolUse(t *testing.T, files []types.GeneratedFile) bool {
	t.Helper()
	f := findGeneratedFile(t, files, settingsJSONPath)
	var settings SettingsJSON
	if err := json.Unmarshal(f.Content, &settings); err != nil {
		t.Fatalf("unmarshaling settings.json: %v", err)
	}
	for _, m := range settings.Hooks["PreToolUse"] {
		if m.Matcher == "Grep" {
			return true
		}
	}
	return false
}

// TestFullLSPPipeline_GoOnly drives the real Generate() for a Go-only Standard
// tier project and asserts the LSP artifacts appear together.
func TestFullLSPPipeline_GoOnly(t *testing.T) {
	t.Parallel()
	files := generateStandard(t, "go")

	requirePresent(t, files,
		lspJSONPath,
		".claude/skills/qsdev-lsp/.claude-plugin/plugin.json",
		".claude/hooks/lsp-first-guard.sh",
		".claude/rules/lsp-navigation.md",
		".claude/rules/go-lsp.md",
	)
	requireAbsent(t, files, ".claude/rules/typescript-lsp.md")

	entries := lspEntries(t, files)
	for _, want := range []string{"go", "nix"} {
		if _, ok := entries[want]; !ok {
			t.Errorf(".lsp.json missing %q entry; got keys %v", want, keysOf(entries))
		}
	}

	guard := findGeneratedFile(t, files, ".claude/hooks/lsp-first-guard.sh")
	if guard.Mode != 0o755 {
		t.Errorf("lsp-first-guard.sh mode = %o, want %o", guard.Mode, 0o755)
	}

	if !settingsHasGrepPreToolUse(t, files) {
		t.Error("settings.json PreToolUse hooks missing a \"Grep\" matcher")
	}
}

// TestFullLSPPipeline_Polyglot drives Generate() for a Go+JS+Python+Rust
// project and asserts the full polyglot set of LSP entries, rules and the hook.
func TestFullLSPPipeline_Polyglot(t *testing.T) {
	t.Parallel()
	files := generateStandard(t, "go", "javascript", "python", "rust")

	entries := lspEntries(t, files)
	wantKeys := []string{"go", "typescript", "python", "rust", "nix"}
	if len(entries) != len(wantKeys) {
		t.Errorf(".lsp.json entry count = %d, want %d; keys %v", len(entries), len(wantKeys), keysOf(entries))
	}
	for _, want := range wantKeys {
		if _, ok := entries[want]; !ok {
			t.Errorf(".lsp.json missing %q entry; got keys %v", want, keysOf(entries))
		}
	}

	requirePresent(t, files,
		".claude/rules/lsp-navigation.md",
		".claude/rules/nix-lsp.md",
		".claude/rules/go-lsp.md",
		".claude/rules/typescript-lsp.md",
		".claude/rules/python-lsp.md",
		".claude/rules/rust-lsp.md",
		".claude/hooks/lsp-first-guard.sh",
	)
}

// TestFullLSPPipeline_NoLanguages drives Generate() with no detected languages.
// The .lsp.json should still be present with only the always-on "nix" entry,
// the navigation and nix rules present, and no per-language rule.
func TestFullLSPPipeline_NoLanguages(t *testing.T) {
	t.Parallel()
	reg := integrationRegistry(t)
	gen := NewClaudeCodeGenerator(reg, Config{})
	files, err := gen.Generate(standardAnswers())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	entries := lspEntries(t, files)
	if len(entries) != 1 {
		t.Errorf(".lsp.json entry count = %d, want 1; keys %v", len(entries), keysOf(entries))
	}
	if _, ok := entries["nix"]; !ok {
		t.Errorf(".lsp.json missing \"nix\" entry; got keys %v", keysOf(entries))
	}

	requirePresent(t, files,
		".claude/rules/lsp-navigation.md",
		".claude/rules/nix-lsp.md",
	)
	requireAbsent(t, files, ".claude/rules/go-lsp.md")
}

// TestFullLSPPipeline_DefaultOffExcluded drives Generate() for a Kotlin-only
// project. Kotlin's LSP server is default-off, so it must not be auto-included
// in .lsp.json, while the always-on nix entry remains.
func TestFullLSPPipeline_DefaultOffExcluded(t *testing.T) {
	t.Parallel()
	files := generateStandard(t, "kotlin")

	entries := lspEntries(t, files)
	if _, ok := entries["kotlin"]; ok {
		t.Errorf("kotlin is default-off and must not be auto-included; keys %v", keysOf(entries))
	}
	if _, ok := entries["nix"]; !ok {
		t.Errorf(".lsp.json missing \"nix\" entry; got keys %v", keysOf(entries))
	}
}

// TestRuleFilePathScoping asserts that every per-language LSP rule generated for
// a polyglot project carries a paths: frontmatter block and the expected glob.
func TestRuleFilePathScoping(t *testing.T) {
	t.Parallel()
	files := generateStandard(t, "go", "javascript", "python", "rust", "java")

	cases := []struct {
		path string
		glob string
	}{
		{".claude/rules/go-lsp.md", "**/*.go"},
		{".claude/rules/typescript-lsp.md", "**/*.ts"},
		{".claude/rules/python-lsp.md", "**/*.py"},
		{".claude/rules/rust-lsp.md", "**/*.rs"},
		{".claude/rules/java-lsp.md", "**/*.java"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			f := findGeneratedFile(t, files, tc.path)
			content := string(f.Content)
			if !strings.HasPrefix(content, "---") {
				t.Errorf("%s: content should begin with frontmatter delimiter ---, got %.10q", tc.path, content)
			}
			if !strings.Contains(content, "paths:") {
				t.Errorf("%s: content should contain a paths: frontmatter block", tc.path)
			}
			if !strings.Contains(content, tc.glob) {
				t.Errorf("%s: content should contain glob %q", tc.path, tc.glob)
			}
		})
	}
}

// TestNoConflictWithConventionRules asserts that the convention rule and the LSP
// rule for the same language coexist at distinct paths with no collision.
func TestNoConflictWithConventionRules(t *testing.T) {
	t.Parallel()
	files := generateStandard(t, "go")

	requirePresent(t, files,
		".claude/rules/go-conventions.md",
		".claude/rules/go-lsp.md",
	)

	// Each path must be produced exactly once: no two files share a path.
	counts := map[string]int{}
	for _, f := range files {
		counts[f.Path]++
	}
	if c := counts[".claude/rules/go-conventions.md"]; c != 1 {
		t.Errorf("go-conventions.md generated %d times, want 1", c)
	}
	if c := counts[".claude/rules/go-lsp.md"]; c != 1 {
		t.Errorf("go-lsp.md generated %d times, want 1", c)
	}
}
