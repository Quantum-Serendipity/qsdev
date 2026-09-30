package claudecode_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"mvdan.cc/sh/v3/syntax"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestHookRegistry_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	got := r.BuildHooksMap(types.WizardAnswers{})
	if got != nil {
		t.Errorf("empty registry should return nil, got %v", got)
	}
}

func TestHookRegistry_RegisterAndRetrieve(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner:         "test-hook",
		Event:         "PreToolUse",
		Matcher:       "Bash",
		Command:       "/bin/true",
		Timeout:       5,
		StatusMessage: "Testing...",
	})

	matchers := r.HooksForEvent("PreToolUse", types.WizardAnswers{})
	if len(matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(matchers))
	}
	if matchers[0].Matcher != "Bash" {
		t.Errorf("matcher = %q, want Bash", matchers[0].Matcher)
	}
	if matchers[0].Hooks[0].Command != "/bin/true" {
		t.Errorf("command = %q, want /bin/true", matchers[0].Hooks[0].Command)
	}
}

func TestHookRegistry_EnabledFuncFilters(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner:       "gated",
		Event:       "PreToolUse",
		Matcher:     "Bash",
		Command:     "/bin/gated",
		Timeout:     5,
		EnabledFunc: func(a types.WizardAnswers) bool { return a.Hooks.SafetyBlock },
	})

	t.Run("disabled when SafetyBlock false", func(t *testing.T) {
		matchers := r.HooksForEvent("PreToolUse", types.WizardAnswers{
			Hooks: types.HookChoices{SafetyBlock: false},
		})
		if len(matchers) != 0 {
			t.Errorf("expected 0 matchers when disabled, got %d", len(matchers))
		}
	})

	t.Run("enabled when SafetyBlock true", func(t *testing.T) {
		matchers := r.HooksForEvent("PreToolUse", types.WizardAnswers{
			Hooks: types.HookChoices{SafetyBlock: true},
		})
		if len(matchers) != 1 {
			t.Errorf("expected 1 matcher when enabled, got %d", len(matchers))
		}
	})
}

func TestHookRegistry_NilEnabledFuncAlwaysEnabled(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner:   "always-on",
		Event:   "PostToolUse",
		Matcher: "*",
		Command: "/bin/always",
		Timeout: 3,
	})

	matchers := r.HooksForEvent("PostToolUse", types.WizardAnswers{})
	if len(matchers) != 1 {
		t.Fatalf("nil EnabledFunc should always be enabled, got %d matchers", len(matchers))
	}
}

func TestHookRegistry_MultipleEventsPartitioned(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner: "pre", Event: "PreToolUse", Matcher: "Bash", Command: "/pre", Timeout: 5,
	})
	r.Register(claudecode.ExportHookDefinition{
		Owner: "post", Event: "PostToolUse", Matcher: "*", Command: "/post", Timeout: 3,
	})

	pre := r.HooksForEvent("PreToolUse", types.WizardAnswers{})
	post := r.HooksForEvent("PostToolUse", types.WizardAnswers{})
	if len(pre) != 1 {
		t.Errorf("PreToolUse: expected 1, got %d", len(pre))
	}
	if len(post) != 1 {
		t.Errorf("PostToolUse: expected 1, got %d", len(post))
	}

	stop := r.HooksForEvent("Stop", types.WizardAnswers{})
	if len(stop) != 0 {
		t.Errorf("Stop: expected 0, got %d", len(stop))
	}
}

func TestHookRegistry_BuildHooksMapPartitions(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner: "a", Event: "PreToolUse", Matcher: "Bash", Command: "/a", Timeout: 5,
	})
	r.Register(claudecode.ExportHookDefinition{
		Owner: "b", Event: "PreToolUse", Matcher: "Write", Command: "/b", Timeout: 5,
	})
	r.Register(claudecode.ExportHookDefinition{
		Owner: "c", Event: "PostToolUse", Matcher: "*", Command: "/c", Timeout: 3,
	})

	m := r.BuildHooksMap(types.WizardAnswers{})
	if m == nil {
		t.Fatal("BuildHooksMap should not return nil with registered hooks")
	}
	if len(m["PreToolUse"]) != 2 {
		t.Errorf("PreToolUse: expected 2 matchers, got %d", len(m["PreToolUse"]))
	}
	if len(m["PostToolUse"]) != 1 {
		t.Errorf("PostToolUse: expected 1 matcher, got %d", len(m["PostToolUse"]))
	}
}

func TestHookRegistry_BuildHooksMapNilWhenAllDisabled(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner:       "off",
		Event:       "PreToolUse",
		Matcher:     "Bash",
		Command:     "/off",
		Timeout:     5,
		EnabledFunc: func(types.WizardAnswers) bool { return false },
	})

	m := r.BuildHooksMap(types.WizardAnswers{})
	if m != nil {
		t.Errorf("expected nil when all hooks disabled, got %v", m)
	}
}

func TestDefaultHookRegistry_PackageGuardRegistered(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	// Disable LSP enforcement so this test isolates the package-guard matcher;
	// otherwise the always-on lsp-guard adds a second PreToolUse matcher.
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true},
		LSP:   types.LSPSettings{Enforcement: "off"},
	}

	matchers := r.HooksForEvent("PreToolUse", answers)
	if len(matchers) != 1 {
		t.Fatalf("expected 1 PreToolUse matcher, got %d", len(matchers))
	}
	if matchers[0].Matcher != "Bash|PowerShell|Monitor" {
		t.Errorf("matcher = %q, want Bash|PowerShell|Monitor", matchers[0].Matcher)
	}
	if matchers[0].Hooks[0].Timeout != 30 {
		t.Errorf("timeout = %d, want 30", matchers[0].Hooks[0].Timeout)
	}
}

func TestDefaultHookRegistry_AuditLogRegistered(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	answers := types.WizardAnswers{Hooks: types.HookChoices{AuditLog: true}}

	matchers := r.HooksForEvent("PostToolUse", answers)
	if len(matchers) != 1 {
		t.Fatalf("expected 1 PostToolUse matcher, got %d", len(matchers))
	}
	if matchers[0].Matcher != "*" {
		t.Errorf("matcher = %q, want *", matchers[0].Matcher)
	}
}

func TestDefaultHookRegistry_BothEnabled(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true, AuditLog: true},
	}

	m := r.BuildHooksMap(answers)
	if m == nil {
		t.Fatal("expected non-nil hooks map")
	}
	if _, ok := m["PreToolUse"]; !ok {
		t.Error("missing PreToolUse event")
	}
	if _, ok := m["PostToolUse"]; !ok {
		t.Error("missing PostToolUse event")
	}
}

func TestDefaultHookRegistry_BothDisabled(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	// Also disable LSP enforcement: the always-on lsp-guard would otherwise
	// register a PreToolUse matcher even with package-guard and audit-log off.
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: false, AuditLog: false},
		LSP:   types.LSPSettings{Enforcement: "off"},
	}

	m := r.BuildHooksMap(answers)
	if m != nil {
		t.Errorf("expected nil when both disabled, got %v", m)
	}
}

func TestBuildHookStatuses(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true, AuditLog: false},
	}

	statuses := claudecode.ExportBuildHookStatuses(r, answers)
	if len(statuses) != 17 {
		t.Fatalf("expected 17 statuses, got %d", len(statuses))
	}

	if statuses[0].Name != "self-protection" || statuses[0].Configured {
		t.Errorf("statuses[0]: want self-protection/disabled (no ClaudeCode), got %s/%v", statuses[0].Name, statuses[0].Configured)
	}
	if statuses[1].Name != "package-guard" || !statuses[1].Configured {
		t.Errorf("statuses[1]: want package-guard/enabled, got %s/%v", statuses[1].Name, statuses[1].Configured)
	}
	if statuses[2].Name != "credential-scan" || statuses[2].Configured {
		t.Errorf("statuses[2]: want credential-scan/disabled, got %s/%v", statuses[2].Name, statuses[2].Configured)
	}
	if statuses[3].Name != "destructive-prevention" || statuses[3].Configured {
		t.Errorf("statuses[3]: want destructive-prevention/disabled, got %s/%v", statuses[3].Name, statuses[3].Configured)
	}
	if statuses[4].Name != "file-boundary" || statuses[4].Configured {
		t.Errorf("statuses[4]: want file-boundary/disabled, got %s/%v", statuses[4].Name, statuses[4].Configured)
	}
	if statuses[5].Name != "tool-gates" || statuses[5].Configured {
		t.Errorf("statuses[5]: want tool-gates/disabled, got %s/%v", statuses[5].Name, statuses[5].Configured)
	}
	for i := 6; i <= 11; i++ {
		if statuses[i].Name != "soc2-audit" || statuses[i].Configured {
			t.Errorf("statuses[%d]: want soc2-audit/disabled, got %s/%v", i, statuses[i].Name, statuses[i].Configured)
		}
	}
	if statuses[12].Name != "semble" || statuses[12].Configured {
		t.Errorf("statuses[12]: want semble/disabled, got %s/%v", statuses[12].Name, statuses[12].Configured)
	}
	if statuses[13].Name != "audit-log" || statuses[13].Configured {
		t.Errorf("statuses[13]: want audit-log/disabled, got %s/%v", statuses[13].Name, statuses[13].Configured)
	}
	for i := 14; i <= 15; i++ {
		if statuses[i].Name != "security-enforcement" || statuses[i].Configured {
			t.Errorf("statuses[%d]: want security-enforcement/disabled, got %s/%v", i, statuses[i].Name, statuses[i].Configured)
		}
	}
	// lsp-guard is registered last and enabled by default (LSP enforcement
	// defaults to "block" when unset).
	if statuses[16].Name != "lsp-guard" || !statuses[16].Configured {
		t.Errorf("statuses[16]: want lsp-guard/enabled, got %s/%v", statuses[16].Name, statuses[16].Configured)
	}
}

func TestSOC2Audit_SuppressesBasicAuditLog(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()

	t.Run("soc2 enabled suppresses audit-log", func(t *testing.T) {
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{AuditLog: true, SOC2Audit: true},
		}
		m := r.BuildHooksMap(answers)
		postToolUse := m["PostToolUse"]
		for _, matcher := range postToolUse {
			for _, h := range matcher.Hooks {
				if h.Command == `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/audit-log.sh` {
					t.Error("basic audit-log should be suppressed when SOC2Audit is enabled")
				}
			}
		}
		if _, ok := m["SessionStart"]; !ok {
			t.Error("SOC2 audit should register SessionStart event")
		}
		if _, ok := m["Stop"]; !ok {
			t.Error("SOC2 audit should register Stop event")
		}
		if _, ok := m["SessionEnd"]; !ok {
			t.Error("SOC2 audit should register SessionEnd event")
		}
	})

	t.Run("soc2 disabled allows audit-log", func(t *testing.T) {
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{AuditLog: true, SOC2Audit: false},
		}
		m := r.BuildHooksMap(answers)
		found := false
		for _, matcher := range m["PostToolUse"] {
			for _, h := range matcher.Hooks {
				if h.Command == `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/audit-log.sh` {
					found = true
				}
			}
		}
		if !found {
			t.Error("basic audit-log should be active when SOC2Audit is disabled")
		}
		if _, ok := m["SessionStart"]; ok {
			t.Error("SOC2 audit should not register when SOC2Audit is disabled")
		}
	})
}

func TestHooksCmd_ListJSON(t *testing.T) {
	t.Parallel()
	cmd := claudecode.ExportHooksCmd()
	cmd.SetArgs([]string{"list", "--json"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// The command will fail because there's no project root, but we can
	// test that the command structure is correct by checking it exists.
	if cmd.Name() != "hooks" {
		t.Errorf("command name = %q, want hooks", cmd.Name())
	}
	sub, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("finding list subcommand: %v", err)
	}
	if sub.Name() != "list" {
		t.Errorf("subcommand name = %q, want list", sub.Name())
	}
}

func TestDefaultHookRegistry_AllTemplatesExist(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportDefaultHookRegistry()
	defs := r.Definitions()

	const prefix = `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/`
	seen := make(map[string]bool)

	for _, d := range defs {
		idx := strings.Index(d.Command, prefix)
		if idx < 0 {
			continue
		}
		rest := d.Command[idx+len(prefix):]
		scriptName := strings.Fields(rest)[0]

		if seen[scriptName] {
			continue
		}
		seen[scriptName] = true

		templatePath := "templates/hooks/" + scriptName
		_, err := claudecode.ExportTemplateFS.ReadFile(templatePath)
		if err != nil {
			t.Errorf("hook %q (event=%s) references template %q that does not exist: %v",
				d.Owner, d.Event, templatePath, err)
		}
	}

	if len(seen) == 0 {
		t.Error("no hook templates found to validate")
	}
}

func TestSecretPatterns_MatchPythonHook(t *testing.T) {
	t.Parallel()

	pyContent, err := claudecode.ExportTemplateFS.ReadFile("templates/hooks/scan-secrets.py")
	if err != nil {
		t.Fatalf("reading scan-secrets.py: %v", err)
	}
	content := string(pyContent)

	all := append(append([]string{}, claudecode.ExportDefaultSecretPatterns...), claudecode.ExportConfigSecretPatterns...)
	for i, goPattern := range all {
		if !strings.Contains(content, goPattern) {
			t.Errorf("Go pattern [%d] %q not found in scan-secrets.py (patterns may be out of sync)", i, goPattern)
		}
	}
}

// TestEmittedCommand_SandboxUsesAppName verifies the sandbox prefix invokes
// the branded binary: a downstream build whose binary is not "qsdev" must not
// emit hook commands that fail with command-not-found.
func TestEmittedCommand_SandboxUsesAppName(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true, AuditLog: true, SandboxEnabled: true}}
	for _, app := range []string{"qsdev", "acme"} {
		t.Run(app, func(t *testing.T) {
			t.Parallel()
			for _, d := range claudecode.ExportDefaultHookRegistry().Definitions() {
				d.FailClosed = false
				cmd := claudecode.ExportEmittedCommand(d, answers, app)
				if !strings.HasPrefix(cmd, app+" sandbox exec --category ") {
					t.Errorf("%s/%s command %q does not start with %q", d.Owner, d.Event, cmd, app+" sandbox exec")
				}
			}
		})
	}
}

// lspGuardCommand returns the lsp-guard PreToolUse command in the settings.json
// Generate emits, or "" when the hook is absent, along with whether the hook
// script itself was generated.
func lspGuardCommand(t *testing.T, answers types.WizardAnswers, cfg claudecode.Config) (string, bool) {
	t.Helper()
	files, err := claudecode.NewClaudeCodeGenerator(newTestRegistry(t, goMock()), cfg).Generate(answers)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var cmd string
	var script bool
	for _, f := range files {
		switch f.Path {
		case ".claude/hooks/lsp-first-guard.sh":
			script = true
		case ".claude/settings.json":
			var s claudecode.SettingsJSON
			if err := json.Unmarshal(f.Content, &s); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			for _, m := range s.Hooks["PreToolUse"] {
				if m.Matcher == "Grep" {
					cmd = m.Hooks[0].Command
				}
			}
		}
	}
	return cmd, script
}

// TestLSPGuard_TierAndEnforcement verifies the lsp-first-guard is only
// installed at tiers that generate the LSP plugin it redirects to, and that the
// configured enforcement tier (including the Config override) is baked into the
// hook command rather than depending on the devenv shell's environment.
func TestLSPGuard_TierAndEnforcement(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		answers    types.WizardAnswers
		cfg        claudecode.Config
		wantCmd    string
		wantScript bool
	}{
		{
			name:    "supply-chain-only has no LSP plugin, so no guard",
			answers: types.WizardAnswers{Tier: "supply-chain-only", Languages: []types.LanguageChoice{{Name: "go"}}},
		},
		{
			name:       "standard defaults to block",
			answers:    types.WizardAnswers{Tier: "standard", Languages: []types.LanguageChoice{{Name: "go"}}},
			wantCmd:    `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/lsp-first-guard.sh block`,
			wantScript: true,
		},
		{
			name: "answers warn is baked in",
			answers: types.WizardAnswers{Tier: "standard", Languages: []types.LanguageChoice{{Name: "go"}},
				LSP: types.LSPSettings{Enforcement: "warn"}},
			wantCmd:    `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/lsp-first-guard.sh warn`,
			wantScript: true,
		},
		{
			name:       "config override reaches the command",
			answers:    types.WizardAnswers{Tier: "standard", Languages: []types.LanguageChoice{{Name: "go"}}},
			cfg:        claudecode.NewConfig(claudecode.WithLSPEnforcement("warn")),
			wantCmd:    `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/lsp-first-guard.sh warn`,
			wantScript: true,
		},
		{
			name: "off removes the guard",
			answers: types.WizardAnswers{Tier: "full", Languages: []types.LanguageChoice{{Name: "go"}},
				LSP: types.LSPSettings{Enforcement: "off"}},
		},
		{
			// `sandbox exec --` execs its arguments directly, so the tier must
			// be a script argument (an env-assignment prefix would be run as
			// the program name), and the linter category must still resolve.
			name: "sandbox wrapping keeps tier and category",
			answers: types.WizardAnswers{Tier: "standard", Languages: []types.LanguageChoice{{Name: "go"}},
				LSP: types.LSPSettings{Enforcement: "warn"}, Hooks: types.HookChoices{SandboxEnabled: true}},
			wantCmd:    `qsdev sandbox exec --category linter -- "${CLAUDE_PROJECT_DIR}"/.claude/hooks/lsp-first-guard.sh warn`,
			wantScript: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, script := lspGuardCommand(t, tc.answers, tc.cfg)
			if cmd != tc.wantCmd {
				t.Errorf("Grep hook command = %q, want %q", cmd, tc.wantCmd)
			}
			if script != tc.wantScript {
				t.Errorf("lsp-first-guard.sh generated = %v, want %v", script, tc.wantScript)
			}
		})
	}
}

// TestHooksWithoutPolicy covers reporting a hook that is enabled but has no
// policy to enforce (W046): tool-gates without allow or deny lists allows
// every tool, so it must not read as an active control.
func TestHooksWithoutPolicy(t *testing.T) {
	t.Parallel()
	gates := types.HookChoices{ToolGates: true}
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    []claudecode.HookWithoutPolicy
	}{
		{
			name:    "tool-gates without policy",
			answers: types.WizardAnswers{Hooks: gates},
			want:    []claudecode.HookWithoutPolicy{{Name: "tool-gates", PolicyKey: "hooks.tool_gates"}},
		},
		{
			name: "tool-gates with deny list",
			answers: types.WizardAnswers{Hooks: gates, HookPolicy: types.HooksConfig{
				ToolGates: types.ToolGatesConfig{Denied: []string{"Bash"}},
			}},
		},
		{
			name: "tool-gates with allow list",
			answers: types.WizardAnswers{Hooks: gates, HookPolicy: types.HooksConfig{
				ToolGates: types.ToolGatesConfig{Allowed: []string{"Read"}},
			}},
		},
		{name: "tool-gates disabled", answers: types.WizardAnswers{}},
		{name: "hooks without a policy key", answers: types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true, FileBoundary: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := claudecode.HooksWithoutPolicy(tt.answers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HooksWithoutPolicy = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// allHooksAnswers enables every hook the registry knows, so each definition
// is emitted.
func allHooksAnswers(sandbox bool) types.WizardAnswers {
	return types.WizardAnswers{
		ClaudeCode: true,
		Tier:       "full",
		Languages:  []types.LanguageChoice{{Name: "go"}},
		Hooks: types.HookChoices{
			AutoFormat: true, SafetyBlock: true, PreCommit: true, AuditLog: true,
			CredentialScan: true, DestructivePrevention: true, SOC2Audit: true,
			FileBoundary: true, ToolGates: true, SecurityEnforcement: true,
			SelfProtection: true, SandboxEnabled: sandbox,
		},
		AgentTools: types.AgentToolsAnswers{SembleEnabled: true},
	}
}

// runSh runs command under the system `sh` with env, returning its exit code,
// stdout and stderr. It skips the test when no `sh` is available.
func runSh(t *testing.T, command string, env []string) (int, string, string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	cmd := exec.Command(sh, "-c", command)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, stdout.String(), stderr.String()
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	default:
		t.Fatalf("running sh: %v", err)
		return 0, "", ""
	}
}

// TestFailClosedCommand_Exit127Blocks guards U17-01: a hook whose interpreter
// or script cannot be found exits 127, which Claude Code treats as a
// non-blocking error; the wrapper must turn it into a block.
func TestFailClosedCommand_Exit127Blocks(t *testing.T) {
	t.Parallel()
	cmd := claudecode.ExportFailClosedCommand("package-guard", `"$D"/missing.py`)
	rc, _, stderr := runSh(t, cmd, []string{"PATH=/nonexistent", "D=" + t.TempDir()})
	if rc != 2 {
		t.Errorf("rc = %d, want 2 (stderr %q)", rc, stderr)
	}
	if !strings.Contains(stderr, "could not run") {
		t.Errorf("stderr %q lacks %q", stderr, "could not run")
	}
}

// writeStub writes content to the non-executable file dir/name and returns
// its path.
func writeStub(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// shStub writes a shell script stub and returns a command running it through
// sh, so the test never execs a file it just wrote (ETXTBSY under parallel
// forks).
func shStub(t *testing.T, dir, name, content string) string {
	t.Helper()
	return `sh "` + writeStub(t, dir, name, content) + `"`
}

// TestFailClosedCommand_ExitCodes pins the wrapper's exit-code mapping: 0
// passes through with stdout (so a JSON permissionDecision still works), 2
// keeps the hook's own reason, and anything else becomes a reasoned block.
func TestFailClosedCommand_ExitCodes(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("stub hooks are POSIX shell scripts")
	}
	dir := t.TempDir()
	const decision = `{"hookSpecificOutput":{"permissionDecision":"allow"}}`
	tests := []struct {
		name        string
		command     func(t *testing.T) string
		wantRC      int
		wantStdout  string
		wantStderr  string
		couldNotRun bool
	}{
		{
			name: "exit 0 passes stdout through",
			command: func(t *testing.T) string {
				return shStub(t, dir, "ok.sh", "#!/bin/sh\nprintf '%s' '"+decision+"'\nexit 0\n")
			},
			wantRC: 0, wantStdout: decision,
		},
		{
			name: "exit 1 blocks",
			command: func(t *testing.T) string {
				return shStub(t, dir, "one.sh", "#!/bin/sh\nexit 1\n")
			},
			wantRC: 2, couldNotRun: true,
		},
		{
			name: "exit 2 keeps hook reason",
			command: func(t *testing.T) string {
				return shStub(t, dir, "two.sh", "#!/bin/sh\necho 'blocked by policy' >&2\nexit 2\n")
			},
			wantRC: 2, wantStderr: "blocked by policy",
		},
		{
			name: "exit 126 not executable blocks",
			command: func(t *testing.T) string {
				return writeStub(t, dir, "noexec.sh", "#!/bin/sh\nexit 0\n")
			},
			wantRC: 2, couldNotRun: true,
		},
		{
			name:    "exit 127 missing blocks",
			command: func(*testing.T) string { return filepath.Join(dir, "missing.py") },
			wantRC:  2, couldNotRun: true,
		},
		{
			name: "uncaught python exception blocks",
			command: func(t *testing.T) string {
				py, err := exec.LookPath("python3")
				if err != nil {
					t.Skip("python3 not found")
				}
				script := writeStub(t, dir, "raise.py", "raise RuntimeError('boom')\n")
				return `"` + py + `" "` + script + `"`
			},
			wantRC: 2, couldNotRun: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := claudecode.ExportFailClosedCommand("test-hook", tc.command(t))
			rc, stdout, stderr := runSh(t, cmd, []string{"PATH=" + os.Getenv("PATH")})
			if rc != tc.wantRC {
				t.Errorf("rc = %d, want %d (stderr %q)", rc, tc.wantRC, stderr)
			}
			if stdout != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tc.wantStdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr %q lacks hook reason %q", stderr, tc.wantStderr)
			}
			if got := strings.Contains(stderr, "could not run"); got != tc.couldNotRun {
				t.Errorf("stderr %q: contains 'could not run' = %v, want %v", stderr, got, tc.couldNotRun)
			}
		})
	}
}

// TestFailClosedCommand_POSIXOnly holds the wrapper to the POSIX sh subset:
// settings.json is shared across the team's OSes and Claude Code may run it
// under dash or Git Bash.
func TestFailClosedCommand_POSIXOnly(t *testing.T) {
	t.Parallel()
	tmpl := claudecode.ExportFailClosedCommand("owner", "CMD")
	bashisms := map[string]*regexp.Regexp{
		"[[":        regexp.MustCompile(`\[\[`),
		"$((":       regexp.MustCompile(`\$\(\(`),
		"function":  regexp.MustCompile(`\bfunction\b`),
		"source":    regexp.MustCompile(`\bsource\b`),
		"local":     regexp.MustCompile(`\blocal\b`),
		"array":     regexp.MustCompile(`\w+=\(`),
		"&>":        regexp.MustCompile(`&>`),
		"pipefail":  regexp.MustCompile(`pipefail`),
		"$'string'": regexp.MustCompile(`\$'`),
	}
	for name, re := range bashisms {
		if re.MatchString(tmpl) {
			t.Errorf("fail-closed template uses bash-only %s: %q", name, tmpl)
		}
	}
	for _, tc := range []struct {
		inner  string
		wantRC int
	}{{"true", 0}, {"false", 2}, {"exit 2", 2}} {
		rc, _, _ := runSh(t, claudecode.ExportFailClosedCommand("owner", tc.inner), []string{"PATH=" + os.Getenv("PATH")})
		if rc != tc.wantRC {
			t.Errorf("sh: %q wrapped rc = %d, want %d", tc.inner, rc, tc.wantRC)
		}
	}
}

// TestBuildHooks_SecurityHooksFailClosed verifies exactly the security
// PreToolUse hooks are wrapped, and that the wrapper is outermost (around the
// sandbox prefix), in the settings Generate writes.
func TestBuildHooks_SecurityHooksFailClosed(t *testing.T) {
	t.Parallel()
	wantWrapped := []string{
		"credential-scan/PreToolUse",
		"destructive-prevention/PreToolUse",
		"file-boundary/PreToolUse",
		"package-guard/PreToolUse",
		"security-enforcement/PreToolUse",
		"self-protection/PreToolUse",
		"tool-gates/PreToolUse",
	}
	for _, sandbox := range []bool{false, true} {
		answers := allHooksAnswers(sandbox)
		built := claudecode.ExportBuildHooks(answers)
		var wrapped []string
		for _, d := range claudecode.ExportDefaultHookRegistry().Definitions() {
			if d.EnabledFunc != nil && !d.EnabledFunc(answers) {
				continue
			}
			emitted := claudecode.ExportEmittedCommand(d, answers, "qsdev")
			plain := d
			plain.FailClosed = false
			inner := claudecode.ExportEmittedCommand(plain, answers, "qsdev")
			if sandbox && !strings.HasPrefix(inner, "qsdev sandbox exec --category ") {
				t.Errorf("sandbox: %s/%s inner command %q lacks the sandbox prefix", d.Owner, d.Event, inner)
			}
			if d.FailClosed {
				wrapped = append(wrapped, d.Owner+"/"+d.Event)
				if !strings.HasPrefix(emitted, inner+" || ") || emitted != claudecode.ExportFailClosedCommand(d.Owner, inner) {
					t.Errorf("sandbox=%v: %s/%s emitted %q, want %q wrapped outermost", sandbox, d.Owner, d.Event, emitted, inner)
				}
			} else if emitted != inner {
				t.Errorf("sandbox=%v: %s/%s must not be fail-closed wrapped, got %q", sandbox, d.Owner, d.Event, emitted)
			}
			if !builtHas(built[d.Event], d.Matcher, emitted) {
				t.Errorf("sandbox=%v: %s/%s command %q not in built hooks", sandbox, d.Owner, d.Event, emitted)
			}
		}
		slices.Sort(wrapped)
		if !slices.Equal(wrapped, wantWrapped) {
			t.Errorf("sandbox=%v: fail-closed hooks = %v, want %v", sandbox, wrapped, wantWrapped)
		}
	}
}

// TestFailClosedHooks_SimpleCommands holds every fail-closed hook's inner
// command (with and without the sandbox prefix) to a single simple command.
// A list or pipeline (`a; b`, `a && b`, `a | b`, `a &`) exits with its last
// part's status, so the wrapper could not catch an earlier part's failure.
func TestFailClosedHooks_SimpleCommands(t *testing.T) {
	t.Parallel()
	for _, sandbox := range []bool{false, true} {
		answers := allHooksAnswers(sandbox)
		for _, d := range claudecode.ExportDefaultHookRegistry().Definitions() {
			if !d.FailClosed {
				continue
			}
			plain := d
			plain.FailClosed = false
			inner := claudecode.ExportEmittedCommand(plain, answers, "qsdev")
			file, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(inner), "")
			if err != nil {
				t.Errorf("sandbox=%v: %s/%s command %q does not parse as POSIX sh: %v", sandbox, d.Owner, d.Event, inner, err)
				continue
			}
			if len(file.Stmts) != 1 {
				t.Errorf("sandbox=%v: %s/%s command %q is a list of %d statements", sandbox, d.Owner, d.Event, inner, len(file.Stmts))
				continue
			}
			st := file.Stmts[0]
			if _, ok := st.Cmd.(*syntax.CallExpr); !ok || st.Background || st.Negated {
				t.Errorf("sandbox=%v: %s/%s command %q is not a single simple command", sandbox, d.Owner, d.Event, inner)
			}
		}
	}
}

func builtHas(matchers []claudecode.HookMatcher, matcher, command string) bool {
	for _, m := range matchers {
		if m.Matcher != matcher {
			continue
		}
		for _, h := range m.Hooks {
			if h.Command == command {
				return true
			}
		}
	}
	return false
}

// TestHookRegistry_FailClosedTimeouts: a fail-closed hook needs time to reach
// its own verdict; Claude Code treats a timed-out hook as non-blocking. Each
// fail-closed Python hook's internal deadline must fire at least 2s before
// its registered timeout, so the hook blocks before Claude Code gives up.
func TestHookRegistry_FailClosedTimeouts(t *testing.T) {
	t.Parallel()
	for _, d := range claudecode.ExportDefaultHookRegistry().Definitions() {
		if d.FailClosed && d.Timeout < 10 {
			t.Errorf("%s/%s: fail-closed hook timeout %ds, want >= 10", d.Owner, d.Event, d.Timeout)
		}
	}
	for script, timeout := range failClosedPythonHooks(t) {
		if deadline := hookDeadlineS(t, script); deadline <= 0 || deadline+2 > timeout {
			t.Errorf("%s: _HOOK_DEADLINE_S = %d, want 0 < deadline <= timeout-2 (%d)", script, deadline, timeout-2)
		}
	}
}

// TestSelfprotectHookTimeoutCoversDeadline verifies Claude Code's timeout for
// the self-protection hook leaves room for the hook's own evaluation deadline
// plus process start-up: if Claude Code's timeout fired first it would allow
// the call instead of receiving the hook's fail-closed deny.
func TestSelfprotectHookTimeoutCoversDeadline(t *testing.T) {
	t.Parallel()

	for _, def := range claudecode.ExportDefaultHookRegistry().Definitions() {
		if def.Owner != "self-protection" {
			continue
		}
		if def.Timeout < 10 {
			t.Errorf("self-protection Timeout = %ds, want at least 10s", def.Timeout)
		}
		if got, need := time.Duration(def.Timeout)*time.Second, hookio.EvalDeadline+2*time.Second; got < need {
			t.Errorf("self-protection Timeout = %v, want at least EvalDeadline+2s = %v", got, need)
		}
		return
	}
	t.Fatal("no self-protection hook registered")
}
