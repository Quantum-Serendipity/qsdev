package claudecode

import (
	"context"

	claudecodeaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func (a *Adapter) SupportedEvents() []aiframework.HookEvent {
	return []aiframework.HookEvent{
		aiframework.EventPreToolUse,
		aiframework.EventPostToolUse,
		aiframework.EventSessionStart,
		aiframework.EventSessionEnd,
	}
}

func (a *Adapter) Protocol() aiframework.HookProtocol {
	return aiframework.HookProtocol{
		InputFormat:     aiframework.InputJSONStdin,
		ResponseFormat:  aiframework.ResponseExitCode,
		EnforcementMode: aiframework.EnforcementHardDeny,
	}
}

func (a *Adapter) Deploy(_ context.Context, hooks []aiframework.HookPolicy) ([]types.GeneratedFile, error) {
	choices := hookPoliciesToChoices(hooks)
	answers := types.WizardAnswers{Hooks: choices}
	return claudecodeaddon.GenerateHookFiles(answers)
}

func (a *Adapter) Undeploy(_ context.Context, _ string) error {
	return nil
}

func hookPoliciesToChoices(policies []aiframework.HookPolicy) types.HookChoices {
	var c types.HookChoices
	for _, p := range policies {
		switch p.Logic {
		case aiframework.LogicPackageGuard:
			c.SafetyBlock = true
		case aiframework.LogicCredentialScan:
			c.CredentialScan = true
		case aiframework.LogicDestructiveBlock:
			c.DestructivePrevention = true
		case aiframework.LogicFileBoundary:
			c.FileBoundary = true
		case aiframework.LogicToolGates:
			c.ToolGates = true
		}
	}
	return c
}
