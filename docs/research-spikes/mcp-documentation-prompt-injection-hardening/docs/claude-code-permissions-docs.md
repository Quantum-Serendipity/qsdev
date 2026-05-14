# Claude Code: Permissions Documentation

- **Source URL**: https://code.claude.com/docs/en/permissions
- **Retrieved**: 2026-05-14

## Permission System

Claude Code uses a tiered permission system:
- **Read-only** (file reads, grep): No approval required
- **Bash commands** (shell execution): Yes, permanently per project directory and command
- **File modification** (edit/write): Yes, until session end

Rules evaluated in order: **deny -> ask -> allow**. First matching rule wins; deny rules always take precedence.

**Key**: Permission rules are enforced by Claude Code, not by the model. Instructions in prompt or CLAUDE.md shape what Claude tries to do, but don't change what Claude Code allows.

## Permission Modes
- `default`: Standard, prompts on first use
- `acceptEdits`: Auto-accepts file edits and common filesystem commands
- `plan`: Read-only exploration mode
- `auto`: Auto-approves with background safety checks (research preview)
- `dontAsk`: Auto-denies unless pre-approved
- `bypassPermissions`: Skips all prompts (isolated environments only)

## MCP Tool Permissions
- `mcp__puppeteer` matches any tool from a server
- `mcp__puppeteer__*` wildcard for all tools from a server
- `mcp__puppeteer__puppeteer_navigate` matches specific tool

## How Permissions Interact with Sandboxing

Permissions and sandboxing are complementary security layers:
- **Permissions**: Control which tools Claude Code can use and which files/domains it can access. Apply to all tools.
- **Sandboxing**: OS-level enforcement restricting Bash tool's filesystem and network access. Applies only to Bash commands.

Defense-in-depth:
- Permission deny rules block Claude from even attempting restricted resources
- Sandbox restrictions prevent Bash commands from reaching resources outside boundaries, **even if a prompt injection bypasses Claude's decision-making**
- Filesystem restrictions combine sandbox settings with Read/Edit deny rules
- Network restrictions combine WebFetch rules with sandbox domain lists

## Bash Compound Commands
Claude Code is aware of shell operators. A rule like `Bash(safe-cmd *)` won't give permission to run `safe-cmd && other-cmd`. Recognized separators: `&&`, `||`, `;`, `|`, `|&`, `&`, and newlines.

## Settings Precedence
1. **Managed settings**: Cannot be overridden
2. **Command line arguments**: Temporary session overrides
3. **Local project settings**
4. **Shared project settings**
5. **User settings**

If denied at any level, no other level can allow it.

## Managed Settings (Enterprise)
Organizations can deploy managed settings that cannot be overridden:
- `allowManagedMcpServersOnly`: Only allow MCP servers from managed settings
- `allowManagedPermissionRulesOnly`: Only rules in managed settings apply
- `sandbox.network.allowManagedDomainsOnly`: Only managed domain allowlists apply
- `allowManagedHooksOnly`: Only managed hooks are loaded
