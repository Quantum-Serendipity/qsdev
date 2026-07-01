package projectctx

import (
	"context"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// The five generic prompts.
//
// The research spike that would have pinned the exact prompt set
// (gdev-mcp-security-devenv-tools-implementation) is absent from the repository,
// so this is a principled default: five high-value, end-to-end workflows that
// chain the generic tools and resources this engine exposes. Each prompt names
// the tools to call (its "tool dependencies"), the resources to read (its
// "embedded resources"), and an ordered sequence of steps.
const (
	promptOnboard     = "onboard-project"
	promptDiagnose    = "diagnose-health"
	promptSecurity    = "security-review"
	promptAddDep      = "add-dependency"
	promptConfigureAI = "configure-ai-framework"
)

// Prompts returns the five generic prompt registrations.
func (pc *ProjectContext) Prompts() []spi.PromptRegistration {
	return []spi.PromptRegistration{
		{
			Name:        promptOnboard,
			Description: "Onboard a developer or agent to this project: detect its shape, verify prerequisites, and summarize how it is configured.",
			Arguments: []spi.PromptArgument{
				{Name: "focus", Description: "Optional area to emphasize (e.g. \"build\", \"security\", \"testing\").", Required: false},
			},
			Handler: pc.promptOnboard,
		},
		{
			Name:        promptDiagnose,
			Description: "Diagnose project and toolchain health, interpret failures, and propose concrete remediations.",
			Arguments: []spi.PromptArgument{
				{Name: "verbose", Description: "Set to \"true\" to request full per-tool detail.", Required: false},
			},
			Handler: pc.promptDiagnose,
		},
		{
			Name:        promptSecurity,
			Description: "Review the project's security posture using its configuration, managed security tools, and MCP server inventory.",
			Arguments: []spi.PromptArgument{
				{Name: "scope", Description: "Optional review scope (e.g. \"dependencies\", \"mcp\", \"secrets\").", Required: false},
			},
			Handler: pc.promptSecurity,
		},
		{
			Name:        promptAddDep,
			Description: "Add a dependency safely: choose the right ecosystem workflow, respect the package guard, and commit the lockfile.",
			Arguments: []spi.PromptArgument{
				{Name: "package", Description: "The package/library/crate/module to add.", Required: true},
				{Name: "ecosystem", Description: "Optional ecosystem hint (e.g. \"go\", \"node\", \"python\", \"rust\").", Required: false},
			},
			Handler: pc.promptAddDep,
		},
		{
			Name:        promptConfigureAI,
			Description: "Configure an AI coding framework to use qsdev's universal MCP server and project context.",
			Arguments: []spi.PromptArgument{
				{Name: "framework", Description: "The AI framework to configure (e.g. \"claude-code\", \"cursor\", \"windsurf\", \"copilot\").", Required: true},
			},
			Handler: pc.promptConfigureAI,
		},
	}
}

// userMessages builds a PromptResult from a description and one user message.
func userMessage(description, text string) *spi.PromptResult {
	return &spi.PromptResult{
		Description: description,
		Messages:    []spi.PromptMessage{{Role: spi.PromptRoleUser, Text: text}},
	}
}

// arg returns the prompt argument or a fallback when it is absent/empty.
func arg(args map[string]string, name, fallback string) string {
	if v, ok := args[name]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func (pc *ProjectContext) promptOnboard(_ context.Context, _ *spi.ToolCallContext, req *spi.PromptRequest) (*spi.PromptResult, error) {
	focus := arg(req.Arguments, "focus", "the project as a whole")
	text := fmt.Sprintf(`Help me onboard to %s. Emphasis: %s.

Follow these steps in order:
1. Call the qsdev_detect tool to capture the current project shape.
2. Call qsdev_project_info to summarize languages, frameworks, ecosystems, and AI integration.
3. Read the resource qsdev://project/detection for the full detection detail.
4. Call qsdev_doctor to confirm the system prerequisites are installed.
5. Read qsdev://project/config to see how the project is configured (or note that it is not initialized).

Then produce a concise onboarding briefing: what this project is, how to build and test it, which qsdev tools are managing it, and any prerequisite gaps a newcomer must close first.`, pc.projectRoot, focus)
	return userMessage("Project onboarding briefing", text), nil
}

func (pc *ProjectContext) promptDiagnose(_ context.Context, _ *spi.ToolCallContext, req *spi.PromptRequest) (*spi.PromptResult, error) {
	verbose := arg(req.Arguments, "verbose", "false")
	text := fmt.Sprintf(`Diagnose the health of %s.

Steps:
1. Call qsdev_doctor with verbose=%s and read the pass/fail verdict plus any missing required or optional tools.
2. Call qsdev_mcp_list with health=true to probe the MCP servers this project relies on.
3. Read qsdev://project/state to confirm qsdev's generated files are present and consistent.

For every failure or missing prerequisite, explain the impact and give the exact remediation command (prefer "qsdev devenv setup", "qsdev enable <tool>", or "qsdev devenv add-package <name>"). Finish with a prioritized fix list, most blocking first.`, pc.projectRoot, verbose)
	return userMessage("Project health diagnosis", text), nil
}

func (pc *ProjectContext) promptSecurity(_ context.Context, _ *spi.ToolCallContext, req *spi.PromptRequest) (*spi.PromptResult, error) {
	scope := arg(req.Arguments, "scope", "the whole project")
	text := fmt.Sprintf(`Perform a security-posture review of %s, scoped to: %s.

Steps:
1. Call qsdev_config_show with include_local=true and inspect the security block (level, age gating, script blocking, lock enforcement, vuln scanning).
2. Call qsdev_tool_list and identify which security-category tools are enabled vs available-but-disabled.
3. Read qsdev://project/mcp-servers and flag any MCP server with a low compliance grade or an untrusted source.

Report: (a) the effective security level and any weakening attempted by local overrides, (b) disabled security tools worth enabling and the "qsdev enable <tool>" command for each, (c) MCP supply-chain risks. Keep recommendations concrete and ranked by risk.`, pc.projectRoot, scope)
	return userMessage("Security posture review", text), nil
}

func (pc *ProjectContext) promptAddDep(_ context.Context, _ *spi.ToolCallContext, req *spi.PromptRequest) (*spi.PromptResult, error) {
	pkg := arg(req.Arguments, "package", "<package-name>")
	ecosystem := arg(req.Arguments, "ecosystem", "")
	eco := ecosystem
	if eco == "" {
		eco = "the project's primary ecosystem (call qsdev_project_info to determine it)"
	}
	text := fmt.Sprintf(`Add the dependency %q to this project safely, targeting %s.

Steps:
1. If the ecosystem is unknown, call qsdev_project_info to determine the package manager.
2. Choose the correct in-shell command (go get, pnpm add, cargo add, or editing pyproject.toml). Run it inside the devenv shell so the package guard hook validates age and vulnerabilities.
3. Do NOT use raw "npm install -g", "pip install", or "curl | bash". If the guard blocks the package, surface its reason rather than working around it.
4. After it is added, commit the updated lockfile (go.sum, pnpm-lock.yaml, Cargo.lock, etc.).

Explain each command before running it and confirm the lockfile change is staged for commit.`, pkg, eco)
	return userMessage("Safe dependency addition", text), nil
}

func (pc *ProjectContext) promptConfigureAI(_ context.Context, _ *spi.ToolCallContext, req *spi.PromptRequest) (*spi.PromptResult, error) {
	framework := arg(req.Arguments, "framework", "<framework>")
	text := fmt.Sprintf(`Configure the AI framework %q to use qsdev's universal MCP server for this project.

Steps:
1. Call qsdev_project_info to confirm whether the framework is already detected for this project.
2. Read qsdev://project/mcp-servers to see the MCP servers qsdev can expose.
3. Register the qsdev universal server as a stdio MCP server for %[1]q (command: "qsdev mcp serve"). Use the framework's own MCP configuration mechanism (for Claude Code this is .mcp.json).
4. Verify the framework can call qsdev_project_info and qsdev_doctor through the server.

Produce the exact configuration snippet for %[1]q and the steps to verify the connection, respecting the framework's tool ceiling (request tool pruning if the framework caps the tool count).`, framework)
	return userMessage("AI framework configuration", text), nil
}
