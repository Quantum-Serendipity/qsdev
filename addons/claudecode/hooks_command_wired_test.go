package claudecode

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestHookWired_MatchesGeneratedWiring guards F100: every hook the generator
// wires into settings.json, plain or sandbox-wrapped, is recognized as wired,
// and a foreign wrapper around the same command is not.
func TestHookWired_MatchesGeneratedWiring(t *testing.T) {
	t.Parallel()
	all := types.HookChoices{
		AutoFormat: true, SafetyBlock: true, PreCommit: true, AuditLog: true,
		CredentialScan: true, DestructivePrevention: true, SOC2Audit: true,
		FileBoundary: true, ToolGates: true, SecurityEnforcement: true, SelfProtection: true,
	}
	for _, sandbox := range []bool{false, true} {
		answers := types.WizardAnswers{ClaudeCode: true, Hooks: all}
		answers.Hooks.SandboxEnabled = sandbox
		deployed := buildHooks(answers)
		registry := defaultHookRegistry()
		for _, def := range registry.Definitions() {
			if def.EnabledFunc != nil && !def.EnabledFunc(answers) {
				continue
			}
			if !hookWired(def, def.commandFor(answers), deployed[def.Event]) {
				t.Errorf("sandbox=%v: %s (%s) generated but not recognized as wired", sandbox, def.Owner, def.Command)
			}
		}
	}

	def := defaultHookRegistry().Definitions()[1]
	foreign := []HookMatcher{{Matcher: def.Matcher, Hooks: []HookEntry{{Type: "command", Command: "true -- " + def.Command}}}}
	if hookWired(def, def.Command, foreign) {
		t.Errorf("a foreign wrapper around %s must not count as wired", def.Command)
	}
}
