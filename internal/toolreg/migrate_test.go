package toolreg

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestInferEnabledTools_NilMap_AttachGuardFollowsHooksAnswer verifies that
// inference reports what the answers configure: attach-guard follows the
// hooks answer. Keeping it on is EnforceAlwaysOn's job, not inference's, so
// status and list never credit a guard that is not configured.
func TestInferEnabledTools_NilMap_AttachGuardFollowsHooksAnswer(t *testing.T) {
	t.Parallel()
	for _, safetyBlock := range []bool{false, true} {
		reg := newTestRegistry(Tool{Name: ToolAttachGuard, Category: CategorySecurity, Default: AlwaysOn})
		answers := &types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{SafetyBlock: safetyBlock}}

		InferEnabledTools(answers, reg)

		if got := answers.EnabledTools[ToolAttachGuard]; got != safetyBlock {
			t.Errorf("SafetyBlock=%v: EnabledTools[attach-guard] = %v, want %v", safetyBlock, got, safetyBlock)
		}
		if answers.Hooks.SafetyBlock != safetyBlock {
			t.Errorf("SafetyBlock=%v: inference changed the hooks answer", safetyBlock)
		}
	}
}

func TestInferEnabledTools_NilMap_AgentPostmortem(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "agent-postmortem", Category: CategoryAIAgent, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		AgentTools:   types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["agent-postmortem"] {
		t.Fatal("expected agent-postmortem to be enabled when PostmortemEnabled is true")
	}
}

func TestInferEnabledTools_NilMap_VersionSentinel(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "version-sentinel", Category: CategoryAIAgent, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		AgentTools:   types.AgentToolsAnswers{VersionSentinel: true},
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["version-sentinel"] {
		t.Fatal("expected version-sentinel to be enabled when VersionSentinel is true")
	}
}

func TestInferEnabledTools_NilMap_Semble(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "semble", Category: CategoryAIAgent, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		AgentTools:   types.AgentToolsAnswers{SembleEnabled: true},
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["semble"] {
		t.Fatal("expected semble to be enabled when SembleEnabled is true")
	}
}

func TestInferEnabledTools_NilMap_TrailOfBitsSkills(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "trail-of-bits-skills", Category: CategorySecurity, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Skills:       []string{"code-review", "security-review", "testing"},
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["trail-of-bits-skills"] {
		t.Fatal("expected trail-of-bits-skills to be enabled when skills contain security-review")
	}
}

func TestInferEnabledTools_NilMap_TrailOfBitsSkills_NotPresent(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "trail-of-bits-skills", Category: CategorySecurity, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Skills:       []string{"code-review", "testing"},
	}

	InferEnabledTools(answers, reg)

	if answers.EnabledTools["trail-of-bits-skills"] {
		t.Fatal("expected trail-of-bits-skills to not be enabled without security-review skill")
	}
}

func TestInferEnabledTools_NilMap_UnknownTool_AlwaysOn(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "new-tool", Category: CategoryDevEx, Default: AlwaysOn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["new-tool"] {
		t.Fatal("expected unknown AlwaysOn tool to be enabled by default")
	}
}

func TestInferEnabledTools_NilMap_UnknownTool_OnWhenDetected(t *testing.T) {
	reg := newTestRegistry(
		Tool{
			Name:     "go-scanner",
			Category: CategoryDevEx,
			Default:  OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasGoMod
			},
		},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Detected:     types.DetectedProject{HasGoMod: true},
	}

	InferEnabledTools(answers, reg)

	if !answers.EnabledTools["go-scanner"] {
		t.Fatal("expected OnWhenDetected tool to be enabled when detected")
	}
}

func TestInferEnabledTools_NilMap_UnknownTool_OnWhenDetected_NotDetected(t *testing.T) {
	reg := newTestRegistry(
		Tool{
			Name:     "go-scanner",
			Category: CategoryDevEx,
			Default:  OnWhenDetected,
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasGoMod
			},
		},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Detected:     types.DetectedProject{HasGoMod: false},
	}

	InferEnabledTools(answers, reg)

	if answers.EnabledTools["go-scanner"] {
		t.Fatal("expected OnWhenDetected tool to not be enabled when not detected")
	}
}

func TestInferEnabledTools_NilMap_UnknownTool_OptIn(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "opt-in-tool", Category: CategoryDevEx, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
	}

	InferEnabledTools(answers, reg)

	if answers.EnabledTools["opt-in-tool"] {
		t.Fatal("expected OptIn tool to not be enabled by default")
	}
}

func TestInferEnabledTools_ExistingMap_Preserved(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "attach-guard", Category: CategorySecurity, Default: AlwaysOn},
		Tool{Name: "semble", Category: CategoryAIAgent, Default: OptIn},
	)

	existing := map[string]bool{
		"attach-guard": false,
		"semble":       true,
	}
	answers := &types.WizardAnswers{
		EnabledTools: existing,
		Hooks:        types.HookChoices{SafetyBlock: true},
		AgentTools:   types.AgentToolsAnswers{SembleEnabled: false},
	}

	InferEnabledTools(answers, reg)

	// Existing map should be completely untouched.
	if answers.EnabledTools["attach-guard"] {
		t.Fatal("expected existing attach-guard=false to be preserved")
	}
	if !answers.EnabledTools["semble"] {
		t.Fatal("expected existing semble=true to be preserved")
	}
	if len(answers.EnabledTools) != 2 {
		t.Fatalf("expected map length to remain 2, got %d", len(answers.EnabledTools))
	}
}

func TestInferEnabledTools_NilMap_MultipleTools(t *testing.T) {
	reg := newTestRegistry(
		// attach-guard is inferred from its default policy (always-on in the
		// catalog), not from the hooks answer.
		Tool{Name: "attach-guard", Category: CategorySecurity, Default: AlwaysOn},
		Tool{Name: "agent-postmortem", Category: CategoryAIAgent, Default: OptIn},
		Tool{Name: "version-sentinel", Category: CategoryAIAgent, Default: OptIn},
		Tool{Name: "semble", Category: CategoryAIAgent, Default: OptIn},
		Tool{Name: "trail-of-bits-skills", Category: CategorySecurity, Default: OptIn},
		Tool{Name: "always-on-tool", Category: CategoryDevEx, Default: AlwaysOn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Hooks:        types.HookChoices{SafetyBlock: true},
		AgentTools: types.AgentToolsAnswers{
			PostmortemEnabled: true,
			VersionSentinel:   false,
			SembleEnabled:     true,
		},
		Skills: []string{"security-review"},
	}

	InferEnabledTools(answers, reg)

	checks := map[string]bool{
		"attach-guard":         true,
		"agent-postmortem":     true,
		"version-sentinel":     false,
		"semble":               true,
		"trail-of-bits-skills": true,
		"always-on-tool":       true,
	}

	for name, want := range checks {
		got := answers.EnabledTools[name]
		if got != want {
			t.Errorf("tool %q: expected %v, got %v", name, want, got)
		}
	}
}

// catalogRegistry returns the registry built from the embedded catalog, so
// enforcement is exercised against the real toggle_field declarations.
func catalogRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := BuildFromCatalogE()
	if err != nil {
		t.Fatalf("building registry from catalog: %v", err)
	}
	return reg
}

func TestEnforceAlwaysOn_OverridesHooksAnswer(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	tests := []struct {
		name    string
		enabled map[string]bool
	}{
		{name: "no enabled tools", enabled: nil},
		{name: "empty enabled tools", enabled: map[string]bool{}},
		{name: "attach-guard enabled", enabled: map[string]bool{ToolAttachGuard: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{
				ClaudeCode:   true,
				EnabledTools: tt.enabled,
				Hooks:        types.HookChoices{SafetyBlock: false},
			}

			overridden := EnforceAlwaysOn(answers, reg)

			if !answers.EnabledTools[ToolAttachGuard] {
				t.Error("EnabledTools[attach-guard] = false, want true")
			}
			if !answers.Hooks.SafetyBlock {
				t.Error("Hooks.SafetyBlock = false, want true")
			}
			if !slices.Contains(overridden, ToolAttachGuard) {
				t.Errorf("overridden = %v, want it to list %s", overridden, ToolAttachGuard)
			}
		})
	}
}

func TestEnforceAlwaysOn_RespectsExplicitDisabled(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{
		ClaudeCode:   true,
		EnabledTools: map[string]bool{ToolAttachGuard: false},
		Hooks:        types.HookChoices{SafetyBlock: false},
	}

	overridden := EnforceAlwaysOn(answers, reg)

	if enabled, ok := answers.EnabledTools[ToolAttachGuard]; !ok || enabled {
		t.Errorf("EnabledTools[attach-guard] = %v (present %v), want explicit false", enabled, ok)
	}
	if answers.Hooks.SafetyBlock {
		t.Error("Hooks.SafetyBlock = true, want the explicit opt-out left untouched")
	}
	if slices.Contains(overridden, ToolAttachGuard) {
		t.Errorf("overridden = %v, want no %s", overridden, ToolAttachGuard)
	}
}

// TestEnforceAlwaysOn_EnabledMeansBacked verifies, for every always-on
// catalog tool, that a tool EnforceAlwaysOn records as enabled has what
// backs it switched on (its toggle_field, skill_name and mcp_server_name), so it is
// generated: "enabled" is never a label without the configuration.
func TestEnforceAlwaysOn_EnabledMeansBacked(t *testing.T) {
	t.Parallel()
	defs := catalog.MustDefault().Tools()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{ClaudeCode: true}

	EnforceAlwaysOn(answers, reg)

	for name, def := range defs {
		if def.DefaultPolicy != "always-on" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if !answers.EnabledTools[name] {
				t.Fatalf("EnabledTools[%s] = false, want true", name)
			}
			if def.ToggleField != "" && !toggleFields[def.ToggleField].get(answers) {
				t.Errorf("toggle %s is off", def.ToggleField)
			}
			if def.SkillName != "" && !slices.Contains(answers.Skills, def.SkillName) {
				t.Errorf("skill %s missing from %v", def.SkillName, answers.Skills)
			}
			if def.MCPServerName != "" && !slices.Contains(answers.MCPServers, def.MCPServerName) {
				t.Errorf("MCP server %s missing from %v", def.MCPServerName, answers.MCPServers)
			}
		})
	}
}

// TestEnforceAlwaysOn_OverrideReportsOnlyExplicitOff verifies that only a
// backing toggle switched off is reported: a toggle already on is not, and
// a skill or MCP server missing from its list is added silently (U28-WS1:
// an always-on MCP tool recorded as enabled has its server configured).
func TestEnforceAlwaysOn_OverrideReportsOnlyExplicitOff(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{
		ClaudeCode: true,
		Hooks:      types.HookChoices{SafetyBlock: true},
		AgentTools: types.AgentToolsAnswers{PostmortemEnabled: false},
	}

	overridden := EnforceAlwaysOn(answers, reg)

	if !slices.Equal(overridden, []string{ToolAgentPostmortem}) {
		t.Errorf("overridden = %v, want [%s]", overridden, ToolAgentPostmortem)
	}
	if !slices.Contains(answers.Skills, "security-review-owasp") {
		t.Errorf("Skills = %v, want the trail-of-bits-skills skill added", answers.Skills)
	}
	if want := catalog.MustDefault().AlwaysOnMCPServers(); !slices.Equal(slices.Sorted(slices.Values(answers.MCPServers)), want) {
		t.Errorf("MCPServers = %v, want the always-on tools' servers %v", answers.MCPServers, want)
	}
}

// TestEnforceAlwaysOn_ClaudeCodeToolsNeedClaudeCode is the --devenv-only
// regression: without Claude Code, the always-on tools that configure it are
// neither recorded nor backed, while project tools stay enforced.
func TestEnforceAlwaysOn_ClaudeCodeToolsNeedClaudeCode(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{ClaudeCode: false}

	overridden := EnforceAlwaysOn(answers, reg)

	for _, name := range []string{ToolAttachGuard, ToolTrailOfBitsSkills, ToolAgentPostmortem} {
		if _, set := answers.EnabledTools[name]; set {
			t.Errorf("EnabledTools[%s] recorded without Claude Code", name)
		}
	}
	if answers.Hooks.SafetyBlock || len(answers.Skills) != 0 || len(overridden) != 0 {
		t.Errorf("SafetyBlock = %v, Skills = %v, overridden = %v; want nothing backed",
			answers.Hooks.SafetyBlock, answers.Skills, overridden)
	}
	if !answers.EnabledTools["branch-naming"] {
		t.Error("EnabledTools[branch-naming] = false, want the project tool enforced")
	}
}

// TestSeedAlwaysOn_BacksWithoutRecording verifies that seeding switches the
// backings on without recording tools, so a committed opt-out can still be
// adopted, and that enforcement afterwards reports nothing.
func TestSeedAlwaysOn_BacksWithoutRecording(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{ClaudeCode: true}

	SeedAlwaysOn(answers, reg)

	if !answers.Hooks.SafetyBlock || !answers.AgentTools.PostmortemEnabled {
		t.Errorf("Hooks = %+v, AgentTools = %+v; want the backings on", answers.Hooks, answers.AgentTools)
	}
	if len(answers.EnabledTools) != 0 {
		t.Errorf("EnabledTools = %v, want none recorded", answers.EnabledTools)
	}
	if overridden := EnforceAlwaysOn(answers, reg); len(overridden) != 0 {
		t.Errorf("overridden after seeding = %v, want none", overridden)
	}
}

// TestMergeInferredTools_EnforcesAlwaysOn is the U28-V01 regression: a
// re-init whose hooks answer dropped the safety block keeps attach-guard and
// reports the restored toggle so the caller can warn.
func TestMergeInferredTools_EnforcesAlwaysOn(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{
		ClaudeCode:   true,
		EnabledTools: map[string]bool{ToolAttachGuard: true},
		Hooks:        types.HookChoices{SafetyBlock: false},
		AgentTools:   types.AgentToolsAnswers{PostmortemEnabled: true},
	}

	overridden := MergeInferredTools(answers, reg)

	if !answers.Hooks.SafetyBlock || !answers.EnabledTools[ToolAttachGuard] {
		t.Errorf("SafetyBlock = %v, EnabledTools[attach-guard] = %v; want both true",
			answers.Hooks.SafetyBlock, answers.EnabledTools[ToolAttachGuard])
	}
	if !slices.Equal(overridden, []string{ToolAttachGuard}) {
		t.Errorf("overridden = %v, want [%s]", overridden, ToolAttachGuard)
	}
}

// TestInferTools_DoesNotEnforce verifies the reporting inference used by
// status and list: a safety block that is off is neither credited nor
// switched on.
func TestInferTools_DoesNotEnforce(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	answers := &types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{SafetyBlock: false}}

	InferTools(answers, reg)

	if answers.EnabledTools[ToolAttachGuard] || answers.Hooks.SafetyBlock {
		t.Errorf("EnabledTools[attach-guard] = %v, SafetyBlock = %v; want both false",
			answers.EnabledTools[ToolAttachGuard], answers.Hooks.SafetyBlock)
	}
}

// TestEnforceAlwaysOn_TierGatesAgentConfig verifies that below the standard
// tier, which generates no skills or agent-tool files, the always-on agent
// tools and skills are neither recorded nor backed, so none is reported as
// enabled without being generated, while the settings-backed attach-guard
// stays enforced at every tier.
func TestEnforceAlwaysOn_TierGatesAgentConfig(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	var agentConfig []string
	for _, tool := range reg.All() {
		if tool.Default == AlwaysOn && (tool.IsAgentTool() || tool.SkillBacked) {
			agentConfig = append(agentConfig, tool.Name)
		}
	}
	if !slices.Contains(agentConfig, ToolTrailOfBitsSkills) {
		t.Fatalf("always-on agent configuration %v lacks %s", agentConfig, ToolTrailOfBitsSkills)
	}
	tests := []struct {
		tier        string
		wantEnforce bool
	}{
		{tier: "supply-chain-only", wantEnforce: false},
		{tier: "standard", wantEnforce: true},
		{tier: "full", wantEnforce: true},
	}
	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			t.Parallel()
			answers := &types.WizardAnswers{ClaudeCode: true, Tier: tt.tier}

			EnforceAlwaysOn(answers, reg)

			if !answers.EnabledTools[ToolAttachGuard] || !answers.Hooks.SafetyBlock {
				t.Errorf("attach-guard not enforced: EnabledTools = %v, SafetyBlock = %v", answers.EnabledTools, answers.Hooks.SafetyBlock)
			}
			for _, name := range agentConfig {
				if got := answers.EnabledTools[name]; got != tt.wantEnforce {
					t.Errorf("EnabledTools[%s] = %v, want %v", name, got, tt.wantEnforce)
				}
			}
			if got := slices.Contains(answers.Skills, "security-review-owasp"); got != tt.wantEnforce {
				t.Errorf("Skills = %v, want security-review-owasp present = %v", answers.Skills, tt.wantEnforce)
			}
		})
	}
}

// TestWarnSafetyBlockOptOut verifies the opt-out warning is written only when
// Claude Code is on and the answers record a safety-block opt-out, at every
// tier, and that it names the way back.
func TestWarnSafetyBlockOptOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    bool
	}{
		{name: "claude off", answers: types.WizardAnswers{Hooks: types.HookChoices{SafetyBlockOptOut: true}}},
		{name: "not opted out", answers: types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{SafetyBlock: true}}},
		{name: "opted out", answers: types.WizardAnswers{ClaudeCode: true, Hooks: types.HookChoices{SafetyBlockOptOut: true}}, want: true},
		{
			name:    "opted out at supply-chain-only tier",
			answers: types.WizardAnswers{ClaudeCode: true, Tier: "supply-chain-only", Hooks: types.HookChoices{SafetyBlockOptOut: true}},
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			WarnSafetyBlockOptOut(&buf, &tt.answers)
			got := buf.String()
			if (got != "") != tt.want {
				t.Fatalf("WarnSafetyBlockOptOut wrote %q, want a warning = %v", got, tt.want)
			}
			if tt.want && !strings.Contains(got, "qsdev enable "+ToolAttachGuard) {
				t.Errorf("warning %q does not name `qsdev enable %s`", got, ToolAttachGuard)
			}
		})
	}
}

// TestEnforceAlwaysOn_MCPServerRespectsClientPolicy verifies enforcement
// adds an always-on MCP tool's server only where the client MCP policy
// permits it (U28-WS1).
func TestEnforceAlwaysOn_MCPServerRespectsClientPolicy(t *testing.T) {
	t.Parallel()
	reg := catalogRegistry(t)
	servers := catalog.MustDefault().AlwaysOnMCPServers()
	blocked := servers[0]
	answers := &types.WizardAnswers{ClaudeCode: true, Tier: "standard",
		MCPPolicy: types.MCPPolicy{Blocked: []string{blocked}}}

	EnforceAlwaysOn(answers, reg)

	if slices.Contains(answers.MCPServers, blocked) {
		t.Errorf("MCPServers = %v, want the blocked %q left out", answers.MCPServers, blocked)
	}
	for _, s := range servers[1:] {
		if !slices.Contains(answers.MCPServers, s) {
			t.Errorf("MCPServers = %v, want permitted %q added", answers.MCPServers, s)
		}
	}
}
