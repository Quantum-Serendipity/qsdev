package answers_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestEnforceInvariants verifies the settings no input path may leave off.
// With Claude Code configured, self-protection is forced on and the safety
// block mirrors the recorded opt-out, whatever the answers file says; without
// it the hook choices are left alone. The tier is always recorded, inferred
// from the legacy fields when empty and kept when explicit. Default MCP
// servers never make an inferred tier full.
func TestEnforceInvariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		answers  types.WizardAnswers
		wantSP   bool
		wantSB   bool
		wantTier string
	}{
		{"claude code forces self-protection and safety block", types.WizardAnswers{
			ClaudeCode: true, Tier: "standard",
		}, true, true, "standard"},
		{"answers edit cannot switch self-protection off", types.WizardAnswers{
			ClaudeCode: true, Tier: "full", Hooks: types.HookChoices{SelfProtection: false, SafetyBlock: true},
		}, true, true, "full"},
		{"answers edit cannot switch safety block off", types.WizardAnswers{
			ClaudeCode: true, Tier: "standard", Hooks: types.HookChoices{SafetyBlock: false},
		}, true, true, "standard"},
		{"opt-out switches safety block off", types.WizardAnswers{
			ClaudeCode: true, Tier: "standard", Hooks: types.HookChoices{SafetyBlockOptOut: true},
		}, true, false, "standard"},
		{"opt-out wins over a recorded safety block", types.WizardAnswers{
			ClaudeCode: true, Tier: "standard", Hooks: types.HookChoices{SafetyBlock: true, SafetyBlockOptOut: true},
		}, true, false, "standard"},
		{"no claude code leaves hooks off", types.WizardAnswers{
			Tier: "standard",
		}, false, false, "standard"},
		{"no claude code keeps explicit hooks", types.WizardAnswers{
			Tier: "standard", Hooks: types.HookChoices{SelfProtection: true, SafetyBlock: true, SafetyBlockOptOut: true},
		}, true, true, "standard"},
		{"explicit tier kept", types.WizardAnswers{Tier: "full"}, false, false, "full"},
		{"legacy default MCP servers", types.WizardAnswers{
			ClaudeCode: true, PermissionLevel: "standard",
			MCPServers: []string{"context7", "github", "socket", "semble"},
		}, true, true, "standard"},
		{"legacy supply-chain-only", types.WizardAnswers{
			ClaudeCode: true, PermissionLevel: "supply-chain-only",
		}, true, true, "supply-chain-only"},
		{"legacy non-default MCP server", types.WizardAnswers{
			ClaudeCode: true, PermissionLevel: "standard", MCPServers: []string{"custom-db"},
		}, true, true, "full"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := tt.answers
			answers.EnforceInvariants(&a)
			if a.Hooks.SelfProtection != tt.wantSP {
				t.Errorf("Hooks.SelfProtection = %v, want %v", a.Hooks.SelfProtection, tt.wantSP)
			}
			if a.Hooks.SafetyBlock != tt.wantSB {
				t.Errorf("Hooks.SafetyBlock = %v, want %v", a.Hooks.SafetyBlock, tt.wantSB)
			}
			if a.Hooks.SafetyBlockOptOut != tt.answers.Hooks.SafetyBlockOptOut {
				t.Errorf("Hooks.SafetyBlockOptOut changed to %v", a.Hooks.SafetyBlockOptOut)
			}
			if a.Tier != tt.wantTier {
				t.Errorf("Tier = %q, want %q", a.Tier, tt.wantTier)
			}
		})
	}
}
