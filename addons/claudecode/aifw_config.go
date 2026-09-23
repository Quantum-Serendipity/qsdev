package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
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

// Render produces the Claude Code configuration files for the given policy
// using the real generators: GenerateSettings (.claude/settings.json with
// permissions, sandbox, and hooks) and GenerateMcpJson (.mcp.json). The
// policy's allow/deny/ask rules and sandbox constraints are carried onto the
// rendered settings, and URL-based MCP servers are emitted as type/url
// entries. When the policy declares a model preference, it additionally
// verifies the project's context files fit that model's window via
// CalculateContextBudget.
func (a *Adapter) Render(ctx context.Context, input *aiframework.PolicyInput) ([]types.GeneratedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("rendering claude code config: nil policy input")
	}

	answers, cfg, remote := a.policyToInputs(input)

	var files []types.GeneratedFile
	settings, err := a.renderSettings(answers, cfg, input)
	if err != nil {
		return nil, err
	}
	if settings != nil {
		files = append(files, *settings)
	}

	mcp, err := GenerateMcpJson(answers, cfg)
	if err != nil {
		return nil, fmt.Errorf("generating .mcp.json: %w", err)
	}
	mcp, err = addRemoteMCPServers(mcp, remote)
	if err != nil {
		return nil, err
	}
	if mcp != nil {
		files = append(files, *mcp)
	}

	if err := checkContextBudget(input); err != nil {
		return nil, err
	}
	return files, nil
}

// renderSettings generates settings.json and merges the policy's ask rules and
// sandbox constraints onto it, so nothing the policy declares is dropped.
func (a *Adapter) renderSettings(answers types.WizardAnswers, cfg Config, input *aiframework.PolicyInput) (*types.GeneratedFile, error) {
	settings, err := GenerateSettings(answers, a.registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("generating settings.json: %w", err)
	}
	if settings == nil {
		return nil, nil
	}
	if input.Permissions != nil {
		settings, err = mergeRulesIntoSettings(settings, nil, nil, aiframework.RulePatterns(input.Permissions.AskRules))
		if err != nil {
			return nil, err
		}
	}
	if input.Sandbox != nil {
		settings, err = applySandboxPolicy(settings, input.Sandbox)
		if err != nil {
			return nil, err
		}
	}
	return settings, nil
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

// Format reports the on-disk format of the rendered configuration files.
func (a *Adapter) Format() string { return "json" }

// policyToInputs translates the framework-agnostic PolicyInput into the
// wizard answers and addon config that GenerateSettings/GenerateMcpJson
// consume. It also returns the URL-based MCP servers, which the addon config
// cannot express and Render emits directly. The always-on Claude Code hooks
// (self-protection, and package-guard when no primary hook is chosen) are
// applied whatever hooks the policy lists.
func (a *Adapter) policyToInputs(input *aiframework.PolicyInput) (types.WizardAnswers, Config, []aiframework.MCPServerSpec) {
	cfg := a.cfg
	cfg.MCPServers = slices.Clone(cfg.MCPServers)
	answers := types.WizardAnswers{ProjectRoot: input.ProjectRoot, ClaudeCode: true}

	if input.Permissions != nil {
		answers.PermissionLevel = input.Permissions.Preset
		if input.Permissions.Preset != "" {
			cfg.DefaultPermissions = PermissionPreset(input.Permissions.Preset)
		}
		cfg.ExtraAllowPatterns = append(slices.Clone(cfg.ExtraAllowPatterns), aiframework.RulePatterns(input.Permissions.AllowRules)...)
		cfg.ExtraDenyPatterns = append(slices.Clone(cfg.ExtraDenyPatterns), aiframework.RulePatterns(input.Permissions.DenyRules)...)
	}
	if input.Hooks != nil {
		answers.Hooks = aiframework.HookChoicesFromSpecs(input.Hooks.Hooks)
	}
	answers.ApplyClaudeHookDefaults()
	if input.Sandbox != nil {
		cfg.SandboxEnabled = true
	}

	var remote []aiframework.MCPServerSpec
	for _, s := range input.MCPServers {
		switch {
		case s.Command != "":
			cfg.MCPServers = append(cfg.MCPServers, MCPServerConfig{
				Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env,
			})
		case s.URL != "":
			remote = append(remote, s)
		default:
			answers.MCPServers = append(answers.MCPServers, s.Name)
		}
	}
	return answers, cfg, remote
}

// addRemoteMCPServers adds URL-based (streamable HTTP or SSE) MCP servers to
// the rendered .mcp.json, creating the file when no other server produced
// one. A remote server overrides a same-named catalog entry, because the
// policy supplied its endpoint explicitly.
func addRemoteMCPServers(mcp *types.GeneratedFile, remote []aiframework.MCPServerSpec) (*types.GeneratedFile, error) {
	if len(remote) == 0 {
		return mcp, nil
	}
	out := types.GeneratedFile{Path: ".mcp.json", Mode: fileutil.ModeReadWrite, Strategy: types.ThreeWayMerge}
	var doc McpJSON
	if mcp != nil {
		out = *mcp
		if err := json.Unmarshal(mcp.Content, &doc); err != nil {
			return nil, fmt.Errorf("parsing rendered .mcp.json: %w", err)
		}
	}
	if doc.MCPServers == nil {
		doc.MCPServers = make(map[string]MCPServerEntry, len(remote))
	}
	for _, s := range remote {
		entryType, err := remoteTransportType(s)
		if err != nil {
			return nil, err
		}
		doc.MCPServers[s.Name] = MCPServerEntry{Type: entryType, URL: s.URL, Env: s.Env}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling .mcp.json: %w", err)
	}
	out.Content = append(data, '\n')
	return &out, nil
}

// remoteTransportType maps a URL-based server's transport onto the .mcp.json
// "type" field. The zero-value transport (stdio) cannot carry a URL, so a
// URL-only spec is treated as streamable HTTP, the current MCP default.
func remoteTransportType(s aiframework.MCPServerSpec) (string, error) {
	switch s.Transport {
	case aiframework.TransportSSE:
		return "sse", nil
	case aiframework.TransportStreamableHTTP, aiframework.TransportStdio:
		return "http", nil
	default:
		return "", fmt.Errorf("MCP server %q: unsupported transport %s", s.Name, s.Transport)
	}
}

// checkContextBudget verifies, when a model preference is declared, that the
// project's existing context files fit the model's window, delegating to the
// real CalculateContextBudget measurement and its 5% threshold validation.
func checkContextBudget(input *aiframework.PolicyInput) error {
	if input.Model == nil || input.ProjectRoot == "" {
		return nil
	}
	budget, err := CalculateContextBudget(input.ProjectRoot, input.Model.PreferredModel)
	if err != nil {
		return fmt.Errorf("calculating context budget: %w", err)
	}
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("rendered project exceeds context budget: %w", err)
	}
	return nil
}
