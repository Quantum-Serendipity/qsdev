package aiframework

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestHookChoicesFromSpecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		logic HookLogicID
		want  types.HookChoices
	}{
		{LogicPackageGuard, types.HookChoices{SafetyBlock: true}},
		{LogicCredentialScan, types.HookChoices{CredentialScan: true}},
		{LogicDestructiveBlock, types.HookChoices{DestructivePrevention: true}},
		{LogicAgentSelfProtection, types.HookChoices{SelfProtection: true}},
		{LogicFileBoundary, types.HookChoices{FileBoundary: true}},
		{LogicToolGates, types.HookChoices{ToolGates: true}},
		{HookLogicID("not-a-hook"), types.HookChoices{}},
	}
	for _, tt := range tests {
		t.Run(string(tt.logic), func(t *testing.T) {
			t.Parallel()
			if got := HookChoicesFromSpecs([]HookSpec{{Command: string(tt.logic)}}); got != tt.want {
				t.Errorf("HookChoicesFromSpecs(%q) = %+v, want %+v", tt.logic, got, tt.want)
			}
		})
	}
}

func TestRulePatterns(t *testing.T) {
	t.Parallel()
	got := RulePatterns([]PermissionRule{{Pattern: "Bash(a)"}, {Reason: "empty"}, {Pattern: "Bash(b)"}})
	if want := []string{"Bash(a)", "Bash(b)"}; !slices.Equal(got, want) {
		t.Errorf("RulePatterns() = %v, want %v", got, want)
	}
}
