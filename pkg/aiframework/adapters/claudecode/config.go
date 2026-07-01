package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Capabilities reports that the Claude Code renderer covers permissions, MCP
// servers, hooks, sandboxing, and ignore patterns (all expressed through
// .claude/settings.json and .mcp.json).
func (a *Adapter) Capabilities() aiframework.ConfigCapabilities {
	return aiframework.ConfigCapabilities{
		RendersPermissions: true,
		RendersMCP:         true,
		RendersHooks:       true,
		RendersSandbox:     true,
		RendersIgnore:      true,
	}
}

// Format reports the on-disk format of the rendered configuration files.
func (a *Adapter) Format() string { return "json" }

// Render produces the Claude Code configuration files for the given policy by
// delegating to the real generators: GenerateSettings (.claude/settings.json
// with permissions, sandbox, and hooks) and GenerateMcpJson (.mcp.json). When
// the policy declares a model preference, it additionally verifies the
// project's context files fit that model's window via CalculateContextBudget.
func (a *Adapter) Render(ctx context.Context, input *aiframework.PolicyInput) ([]types.GeneratedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("rendering claude code config: nil policy input")
	}

	answers, cfg := a.buildInputs(input)

	var files []types.GeneratedFile
	settings, err := ccaddon.GenerateSettings(answers, a.registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("generating settings.json: %w", err)
	}
	if settings != nil {
		files = append(files, *settings)
	}

	mcp, err := ccaddon.GenerateMcpJson(answers, cfg)
	if err != nil {
		return nil, fmt.Errorf("generating .mcp.json: %w", err)
	}
	if mcp != nil {
		files = append(files, *mcp)
	}

	if err := a.checkContextBudget(input); err != nil {
		return nil, err
	}
	return files, nil
}

// Validate checks that every rendered JSON file is well-formed, reporting an
// error-severity issue for any malformed content.
func (a *Adapter) Validate(_ context.Context, files []types.GeneratedFile) []aiframework.ValidationIssue {
	var issues []aiframework.ValidationIssue
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".json") && !json.Valid(f.Content) {
			issues = append(issues, aiframework.ValidationIssue{
				Path:     f.Path,
				Message:  "invalid JSON",
				Severity: aiframework.SeverityError,
			})
		}
	}
	return issues
}

// buildInputs translates the framework-agnostic PolicyInput into the wizard
// answers and addon config that GenerateSettings/GenerateMcpJson consume.
func (a *Adapter) buildInputs(input *aiframework.PolicyInput) (types.WizardAnswers, ccaddon.Config) {
	cfg := a.cfg
	cfg.MCPServers = slices.Clone(cfg.MCPServers)
	answers := types.WizardAnswers{ProjectRoot: input.ProjectRoot, ClaudeCode: true}

	if input.Permissions != nil {
		answers.PermissionLevel = input.Permissions.Preset
		if input.Permissions.Preset != "" {
			cfg.DefaultPermissions = ccaddon.PermissionPreset(input.Permissions.Preset)
		}
		cfg.ExtraAllowPatterns = append(slices.Clone(cfg.ExtraAllowPatterns), rulePatterns(input.Permissions.AllowRules)...)
		cfg.ExtraDenyPatterns = append(slices.Clone(cfg.ExtraDenyPatterns), rulePatterns(input.Permissions.DenyRules)...)
	}
	if input.Hooks != nil {
		answers.Hooks = hookSpecsToChoices(input.Hooks.Hooks)
	}
	for _, s := range input.MCPServers {
		if s.Command != "" {
			cfg.MCPServers = append(cfg.MCPServers, ccaddon.MCPServerConfig{
				Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env,
			})
		} else {
			answers.MCPServers = append(answers.MCPServers, s.Name)
		}
	}
	return answers, cfg
}

// checkContextBudget verifies, when a model preference is declared, that the
// project's existing context files fit the model's window, delegating to the
// real CalculateContextBudget measurement and its 5% threshold validation.
func (a *Adapter) checkContextBudget(input *aiframework.PolicyInput) error {
	if input.Model == nil || input.ProjectRoot == "" {
		return nil
	}
	budget, err := ccaddon.CalculateContextBudget(input.ProjectRoot, input.Model.PreferredModel)
	if err != nil {
		return fmt.Errorf("calculating context budget: %w", err)
	}
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("rendered project exceeds context budget: %w", err)
	}
	return nil
}
