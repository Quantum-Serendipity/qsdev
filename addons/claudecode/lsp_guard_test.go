package claudecode

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
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

// runLspGuard pipes a Grep payload through lsp-first-guard.sh at the block
// tier and reports "deny" or "allow".
func runLspGuard(t *testing.T, input map[string]any) string {
	t.Helper()
	for _, bin := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available; skipping lsp-first-guard test", bin)
		}
	}
	payload, err := json.Marshal(map[string]any{"tool_name": "Grep", "tool_input": input})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", filepath.Join("templates", "hooks", "lsp-first-guard.sh"), "block")
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}
	if strings.Contains(string(out), `"permissionDecision":"deny"`) {
		return "deny"
	}
	return "allow"
}

// TestLspFirstGuardHook_Classification pins which Grep searches are redirected
// to the LSP tool (W050): code symbols are, while searches restricted to
// non-code files (Grep's type filter, brace globs, extension-less build and
// CI files) and ordinary capitalised words are not.
func TestLspFirstGuardHook_Classification(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input map[string]any
		want  string
	}{
		{"camelCase symbol", map[string]any{"pattern": "getUserById"}, "deny"},
		{"PascalCase symbol", map[string]any{"pattern": "UserService"}, "deny"},
		{"dotted access", map[string]any{"pattern": "router.refresh"}, "deny"},
		{"snake_case symbol", map[string]any{"pattern": "handle_user_request"}, "deny"},
		{"symbol in go files", map[string]any{"pattern": "UserService", "glob": "**/*.go"}, "deny"},
		{"type json", map[string]any{"pattern": "compilerOptions", "type": "json"}, "allow"},
		{"type md", map[string]any{"pattern": "QuickStart", "type": "md"}, "allow"},
		{"type go", map[string]any{"pattern": "UserService", "type": "go"}, "deny"},
		{"brace glob docs", map[string]any{"pattern": "InstallGuide", "glob": "**/*.{md,mdx}"}, "allow"},
		{"brace glob code", map[string]any{"pattern": "UserService", "glob": "**/*.{ts,tsx}"}, "deny"},
		{"plain word Copyright", map[string]any{"pattern": "Copyright"}, "allow"},
		{"plain word Deprecated", map[string]any{"pattern": "Deprecated"}, "allow"},
		{"plain word Installation", map[string]any{"pattern": "Installation"}, "allow"},
		{"workflows dir", map[string]any{"pattern": "runsOn", "path": ".github/workflows"}, "allow"},
		{"Makefile", map[string]any{"pattern": "buildAll", "path": "Makefile"}, "allow"},
		{"Dockerfile", map[string]any{"pattern": "baseImage", "path": "Dockerfile"}, "allow"},
		{"go.mod", map[string]any{"pattern": "golangOrg", "path": "go.mod"}, "allow"},
		{"devenv.nix", map[string]any{"pattern": "enterShell", "path": "devenv.nix"}, "allow"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := runLspGuard(t, tc.input); got != tc.want {
				t.Errorf("decision = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLspFirstGuardHarness runs the script's standalone shell harness so its
// cases are part of `go test`.
func TestLspFirstGuardHarness(t *testing.T) {
	t.Parallel()
	for _, bin := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available; skipping lsp-first-guard harness", bin)
		}
	}
	out, err := exec.Command("bash", filepath.Join("templates", "hooks", "lsp-first-guard_test.sh")).CombinedOutput()
	if err != nil {
		t.Fatalf("harness failed: %v\n%s", err, out)
	}
}
