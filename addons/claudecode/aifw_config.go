package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func (a *Adapter) Capabilities() aiframework.ConfigCapabilities {
	return aiframework.ConfigCapabilities{
		RendersPermissions: true,
		RendersMCP:         true,
		RendersHooks:       true,
		RendersSandbox:     true,
		RendersIgnore:      true,
	}
}

func (a *Adapter) Render(_ context.Context, input *aiframework.PolicyInput) ([]types.GeneratedFile, error) {
	answers := policyToAnswers(input)
	cfg := a.cfg

	if input.Permissions != nil && input.Permissions.Preset != "" {
		cfg.DefaultPermissions = PermissionPreset(input.Permissions.Preset)
	}

	settings, err := GenerateSettings(answers, a.registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("generating settings: %w", err)
	}

	var files []types.GeneratedFile
	if settings != nil {
		files = append(files, *settings)
	}
	return files, nil
}

func (a *Adapter) Validate(_ context.Context, files []types.GeneratedFile) []aiframework.ValidationIssue {
	var issues []aiframework.ValidationIssue
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".json") {
			if !json.Valid(f.Content) {
				issues = append(issues, aiframework.ValidationIssue{
					Path:     f.Path,
					Message:  "invalid JSON",
					Severity: aiframework.SeverityError,
				})
			}
		}
	}
	return issues
}

func (a *Adapter) Format() string { return "json" }

func policyToAnswers(input *aiframework.PolicyInput) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot: input.ProjectRoot,
		ClaudeCode:  true,
	}
	if input.Permissions != nil {
		answers.PermissionLevel = input.Permissions.Preset
	}
	if input.Hooks != nil {
		answers.Hooks = hookSpecsToChoices(input.Hooks.Hooks)
	}
	for _, s := range input.MCPServers {
		answers.MCPServers = append(answers.MCPServers, s.Name)
	}
	return answers
}

func hookSpecsToChoices(specs []aiframework.HookSpec) types.HookChoices {
	var c types.HookChoices
	for _, s := range specs {
		switch s.Command {
		case string(aiframework.LogicPackageGuard):
			c.SafetyBlock = true
		case string(aiframework.LogicCredentialScan):
			c.CredentialScan = true
		case string(aiframework.LogicDestructiveBlock):
			c.DestructivePrevention = true
		case string(aiframework.LogicFileBoundary):
			c.FileBoundary = true
		case string(aiframework.LogicToolGates):
			c.ToolGates = true
		}
	}
	return c
}
