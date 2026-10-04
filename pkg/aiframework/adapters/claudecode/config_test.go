package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// TestAdapterGenerate_AlwaysRendersPackageGuard verifies the package guard is
// registered as a PreToolUse hook whatever hooks a framework spec lists: a
// spec cannot record a safety-block opt-out, so omitting package-guard never
// drops it.
func TestAdapterGenerate_AlwaysRendersPackageGuard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		hooks *aiframework.HookConfiguration
	}{
		{name: "no hook configuration"},
		{name: "empty spec list", hooks: &aiframework.HookConfiguration{}},
		{name: "only credential scan", hooks: specs(aiframework.LogicCredentialScan)},
		{name: "only destructive block", hooks: specs(aiframework.LogicDestructiveBlock)},
		{name: "file boundary and tool gates", hooks: specs(aiframework.LogicFileBoundary, aiframework.LogicToolGates)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			settings := renderSettings(t, &aiframework.PolicyInput{
				ProjectRoot: t.TempDir(),
				Permissions: &aiframework.PermissionPolicy{Preset: "standard"},
				Hooks:       tt.hooks,
			})
			var cmds []string
			for _, m := range settings.Hooks["PreToolUse"] {
				for _, h := range m.Hooks {
					cmds = append(cmds, h.Command)
				}
			}
			if !containsSubstring(cmds, string(aiframework.LogicPackageGuard)) {
				t.Errorf("PreToolUse hooks lack package-guard: %v", cmds)
			}
		})
	}
}

// specs returns a hook configuration listing one spec per logic ID.
func specs(ids ...aiframework.HookLogicID) *aiframework.HookConfiguration {
	cfg := &aiframework.HookConfiguration{}
	for _, id := range ids {
		cfg.Hooks = append(cfg.Hooks, aiframework.HookSpec{Command: string(id)})
	}
	return cfg
}
