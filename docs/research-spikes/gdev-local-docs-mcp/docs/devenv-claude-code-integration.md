<!-- Source: https://devenv.sh/integrations/claude-code/ -->
<!-- Retrieved: 2026-05-14 -->

# devenv Claude Code Integration Documentation

## Overview
The devenv integration with Claude Code establishes an automated development workflow through configuration in `devenv.nix`. The system enables AI-assisted development while maintaining reproducible environments.

## .mcp.json Generation

Devenv automatically generates a `.mcp.json` file based on MCP server configurations defined in `devenv.nix`. This file allows Claude Code to connect to specified servers without manual setup. "When MCP servers are configured, devenv generates a `.mcp.json` file that Claude Code uses to connect to these servers."

## Configuration Discovery

Claude Code discovers devenv configuration through two mechanisms:

1. **Global Configuration**: Users create `~/.claude/CLAUDE.md` containing instructions to use devenv for running commands, ensuring all tools and dependencies are available.

2. **Project-Level Configuration**: Settings defined in `devenv.nix` under the `claude.code` namespace are automatically applied to the project environment.

## MCP Server Configuration Patterns

Two server types are supported:

**Stdio servers** execute commands communicating via stdin/stdout, with properties for `command`, `args`, and environment variables. Example: "The devenv MCP server uses type 'stdio' with command 'devenv' and args ['mcp']."

**HTTP servers** connect to remote endpoints with configurable `url` and optional authentication `headers` for services like Linear or GitHub.

## Integration with devenv.nix

The `claude.code` section in `devenv.nix` controls all Claude Code functionality:

- **`enable`**: Activates the integration
- **`hooks`**: Defines pre/post-tool execution actions
- **`commands`**: Creates project-specific slash commands
- **`agents`**: Configures specialized sub-agents with restricted tool access
- **`mcpServers`**: Registers MCP server definitions

This approach ensures Claude's capabilities remain scoped to project-defined tools and dependencies, maintaining security and reproducibility.
