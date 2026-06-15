package claudecode

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const lspGuardScriptPath = ".claude/hooks/lsp-first-guard.sh"

// TestLspGuardHook_GeneratedByDefault verifies that with default answers (no
// explicit enforcement tier, which resolves to "block") the lsp-first-guard
// script is emitted with executable mode.
func TestLspGuardHook_GeneratedByDefault(t *testing.T) {
	t.Parallel()

	files, err := GenerateHookFiles(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateHookFiles: %v", err)
	}

	f := findGeneratedFile(t, files, lspGuardScriptPath)
	if f.Mode != 0o755 {
		t.Errorf("mode = %o, want %o", f.Mode, 0o755)
	}
	if len(f.Content) == 0 {
		t.Error("script content is empty")
	}
	if f.Owner != "lsp-guard" {
		t.Errorf("owner = %q, want %q", f.Owner, "lsp-guard")
	}
}

// TestLspGuardHook_NotGeneratedWhenOff verifies the script is absent from the
// generated files when enforcement is explicitly disabled.
func TestLspGuardHook_NotGeneratedWhenOff(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		LSP: types.LSPSettings{Enforcement: "off"},
	}
	files, err := GenerateHookFiles(answers)
	if err != nil {
		t.Fatalf("GenerateHookFiles: %v", err)
	}

	for _, f := range files {
		if f.Path == lspGuardScriptPath {
			t.Fatalf("lsp-first-guard.sh should be absent when enforcement is off, got %+v", f)
		}
	}
}

// TestLspGuardHook_SettingsRegistration verifies the PreToolUse hooks include a
// Grep matcher pointing at the guard script under default answers, and that the
// matcher disappears when enforcement is off.
func TestLspGuardHook_SettingsRegistration(t *testing.T) {
	t.Parallel()

	t.Run("default registers Grep matcher", func(t *testing.T) {
		t.Parallel()
		hooks := buildHooks(types.WizardAnswers{})
		if !hasGrepLspMatcher(hooks["PreToolUse"]) {
			t.Errorf("PreToolUse hooks missing Grep matcher for lsp-first-guard.sh: %+v", hooks["PreToolUse"])
		}
	})

	t.Run("off omits Grep matcher", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			LSP: types.LSPSettings{Enforcement: "off"},
		}
		hooks := buildHooks(answers)
		if hasGrepLspMatcher(hooks["PreToolUse"]) {
			t.Errorf("PreToolUse hooks should not contain the lsp-first-guard Grep matcher when enforcement is off: %+v", hooks["PreToolUse"])
		}
	})
}

// hasGrepLspMatcher reports whether the matchers contain a Grep matcher whose
// command references the lsp-first-guard script.
func hasGrepLspMatcher(matchers []HookMatcher) bool {
	for _, m := range matchers {
		if m.Matcher != "Grep" {
			continue
		}
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, "lsp-first-guard.sh") {
				return true
			}
		}
	}
	return false
}
