package claudecode_test

import (
	"bytes"
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
	answers := types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true}}

	matchers := r.HooksForEvent("PreToolUse", answers)
	if len(matchers) != 1 {
		t.Fatalf("expected 1 PreToolUse matcher, got %d", len(matchers))
	}
	if matchers[0].Matcher != "Bash" {
		t.Errorf("matcher = %q, want Bash", matchers[0].Matcher)
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
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: false, AuditLog: false},
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
	if len(statuses) != 14 {
		t.Fatalf("expected 14 statuses, got %d", len(statuses))
	}

	if statuses[0].Name != "self-protection" || statuses[0].Enabled {
		t.Errorf("statuses[0]: want self-protection/disabled (no ClaudeCode), got %s/%v", statuses[0].Name, statuses[0].Enabled)
	}
	if statuses[1].Name != "package-guard" || !statuses[1].Enabled {
		t.Errorf("statuses[1]: want package-guard/enabled, got %s/%v", statuses[1].Name, statuses[1].Enabled)
	}
	if statuses[2].Name != "credential-scan" || statuses[2].Enabled {
		t.Errorf("statuses[2]: want credential-scan/disabled, got %s/%v", statuses[2].Name, statuses[2].Enabled)
	}
	if statuses[3].Name != "destructive-prevention" || statuses[3].Enabled {
		t.Errorf("statuses[3]: want destructive-prevention/disabled, got %s/%v", statuses[3].Name, statuses[3].Enabled)
	}
	if statuses[4].Name != "file-boundary" || statuses[4].Enabled {
		t.Errorf("statuses[4]: want file-boundary/disabled, got %s/%v", statuses[4].Name, statuses[4].Enabled)
	}
	if statuses[5].Name != "tool-gates" || statuses[5].Enabled {
		t.Errorf("statuses[5]: want tool-gates/disabled, got %s/%v", statuses[5].Name, statuses[5].Enabled)
	}
	for i := 6; i <= 9; i++ {
		if statuses[i].Name != "soc2-audit" || statuses[i].Enabled {
			t.Errorf("statuses[%d]: want soc2-audit/disabled, got %s/%v", i, statuses[i].Name, statuses[i].Enabled)
		}
	}
	if statuses[10].Name != "semble" || statuses[10].Enabled {
		t.Errorf("statuses[10]: want semble/disabled, got %s/%v", statuses[10].Name, statuses[10].Enabled)
	}
	if statuses[11].Name != "audit-log" || statuses[11].Enabled {
		t.Errorf("statuses[11]: want audit-log/disabled, got %s/%v", statuses[11].Name, statuses[11].Enabled)
	}
	for i := 12; i <= 13; i++ {
		if statuses[i].Name != "security-enforcement" || statuses[i].Enabled {
			t.Errorf("statuses[%d]: want security-enforcement/disabled, got %s/%v", i, statuses[i].Name, statuses[i].Enabled)
		}
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

	for i, goPattern := range claudecode.ExportDefaultSecretPatterns {
		if !strings.Contains(content, goPattern) {
			t.Errorf("Go pattern [%d] %q not found in scan-secrets.py (patterns may be out of sync)", i, goPattern)
		}
	}
}
