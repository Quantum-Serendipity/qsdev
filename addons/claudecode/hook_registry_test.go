package claudecode_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
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

func TestHookDeploymentTier_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier claudecode.HookDeploymentTier
		want string
	}{
		{claudecode.ExportTierProject, "project"},
		{claudecode.ExportTierTeam, "team"},
		{claudecode.ExportTierOrg, "org"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("Tier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestHookRegistry_TierFiltering(t *testing.T) {
	t.Parallel()
	r := claudecode.ExportNewHookRegistry()
	r.Register(claudecode.ExportHookDefinition{
		Owner: "project-hook", Event: "PreToolUse", Matcher: "Bash",
		Command: "/project", Timeout: 5, Tier: claudecode.ExportTierProject,
	})
	r.Register(claudecode.ExportHookDefinition{
		Owner: "org-hook", Event: "PreToolUse", Matcher: "Bash",
		Command: "/org", Timeout: 5, Tier: claudecode.ExportTierOrg,
	})

	answers := types.WizardAnswers{}

	t.Run("all tiers", func(t *testing.T) {
		m := r.BuildHooksMap(answers)
		if len(m["PreToolUse"]) != 2 {
			t.Errorf("all tiers: expected 2 matchers, got %d", len(m["PreToolUse"]))
		}
	})

	t.Run("project tier only", func(t *testing.T) {
		tier := claudecode.ExportTierProject
		m := r.BuildHooksMapForTier(answers, &tier)
		if len(m["PreToolUse"]) != 1 {
			t.Fatalf("project tier: expected 1 matcher, got %d", len(m["PreToolUse"]))
		}
		if m["PreToolUse"][0].Hooks[0].Command != "/project" {
			t.Errorf("wrong command: %s", m["PreToolUse"][0].Hooks[0].Command)
		}
	})

	t.Run("org tier only", func(t *testing.T) {
		tier := claudecode.ExportTierOrg
		m := r.BuildHooksMapForTier(answers, &tier)
		if len(m["PreToolUse"]) != 1 {
			t.Fatalf("org tier: expected 1 matcher, got %d", len(m["PreToolUse"]))
		}
		if m["PreToolUse"][0].Hooks[0].Command != "/org" {
			t.Errorf("wrong command: %s", m["PreToolUse"][0].Hooks[0].Command)
		}
	})

	t.Run("team tier empty", func(t *testing.T) {
		tier := claudecode.ExportTierTeam
		m := r.BuildHooksMapForTier(answers, &tier)
		if m != nil {
			t.Errorf("team tier: expected nil, got %v", m)
		}
	})
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

// TestWrapHooksForSandbox_UsesAppName verifies the sandbox wrapper invokes the
// branded binary: a downstream build whose binary is not "qsdev" must not emit
// hook commands that fail with command-not-found (a non-blocking hook error,
// so every guard would fail open).
func TestWrapHooksForSandbox_UsesAppName(t *testing.T) {
	t.Parallel()
	for _, app := range []string{"qsdev", "acme"} {
		t.Run(app, func(t *testing.T) {
			t.Parallel()
			r := claudecode.ExportDefaultHookRegistry()
			hooks := r.BuildHooksMap(types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true, AuditLog: true}})
			wrapped := claudecode.ExportWrapHooksForSandbox(hooks, r, types.WizardAnswers{}, app)
			if len(wrapped) == 0 {
				t.Fatal("expected hooks to wrap")
			}
			for event, matchers := range wrapped {
				for _, m := range matchers {
					for _, h := range m.Hooks {
						if !strings.HasPrefix(h.Command, app+" sandbox exec --category ") {
							t.Errorf("%s hook command %q does not start with %q", event, h.Command, app+" sandbox exec")
						}
					}
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
