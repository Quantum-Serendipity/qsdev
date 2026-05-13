package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestInferEnabledTools_NilMap_AttachGuard(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "attach-guard", Category: CategorySecurity, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Hooks:        types.HookChoices{SafetyBlock: true},
	}

	InferEnabledTools(answers, reg)

	if answers.EnabledTools == nil {
		t.Fatal("expected EnabledTools to be initialized")
	}
	if !answers.EnabledTools["attach-guard"] {
		t.Fatal("expected attach-guard to be enabled when SafetyBlock is true")
	}
}

func TestInferEnabledTools_NilMap_AttachGuardDisabled(t *testing.T) {
	reg := newTestRegistry(
		Tool{Name: "attach-guard", Category: CategorySecurity, Default: OptIn},
	)

	answers := &types.WizardAnswers{
		EnabledTools: nil,
		Hooks:        types.HookChoices{SafetyBlock: false},
	}

	InferEnabledTools(answers, reg)

	if answers.EnabledTools["attach-guard"] {
		t.Fatal("expected attach-guard to not be enabled when SafetyBlock is false")
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
		Tool{Name: "attach-guard", Category: CategorySecurity, Default: OptIn},
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
		"attach-guard":        true,
		"agent-postmortem":    true,
		"version-sentinel":    false,
		"semble":              true,
		"trail-of-bits-skills": true,
		"always-on-tool":      true,
	}

	for name, want := range checks {
		got := answers.EnabledTools[name]
		if got != want {
			t.Errorf("tool %q: expected %v, got %v", name, want, got)
		}
	}
}
