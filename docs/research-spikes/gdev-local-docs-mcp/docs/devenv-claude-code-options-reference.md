<!-- Source: https://devenv.sh/reference/options/ -->
<!-- Retrieved: 2026-05-14 -->

# Claude.code Configuration Options in devenv.nix

## Core Settings

- **claude.code.enable** - Master toggle for Claude Code integration
- **claude.code.model** - Specifies which Claude model to use
- **claude.code.forceLoginMethod** - Controls authentication approach
- **claude.code.apiKeyHelper** - Configures API key management
- **claude.code.cleanupPeriodDays** - Sets retention period for temporary files

## MCP Servers

- **claude.code.mcpServers** - Container for Model Context Protocol configurations
  - **claude.code.mcpServers.<name>.type** - Server protocol type (stdio or http)
  - **claude.code.mcpServers.<name>.command** - Executable command (for stdio)
  - **claude.code.mcpServers.<name>.args** - Command-line arguments (for stdio)
  - **claude.code.mcpServers.<name>.env** - Environment variables
  - **claude.code.mcpServers.<name>.url** - Server endpoint (for http)
  - **claude.code.mcpServers.<name>.headers** - HTTP headers (for http)

## Hooks

- **claude.code.hooks** - Git and custom event handlers
- **claude.code.hooks.<name>.enable** - Individual hook activation
- **claude.code.hooks.<name>.command** - Hook execution command
- **claude.code.hooks.<name>.hookType** - Event classification
- **claude.code.hooks.<name>.matcher** - Pattern matching rules

## Permissions

- **claude.code.permissions** - Access control framework
- **claude.code.permissions.defaultMode** - Default policy stance
- **claude.code.permissions.disableBypassPermissionsMode** - Locks override capabilities

## Additional Options

- **claude.code.commands** - Custom CLI commands
- **claude.code.env** - Environment variable mappings
- **claude.code.agents** - AI agent definitions with model, description, and tool specifications

## Key Integration Pattern

devenv generates .mcp.json from claude.code.mcpServers configuration. This means gdev can either:
1. Generate devenv.nix with claude.code.mcpServers entries (devenv generates .mcp.json)
2. Generate .mcp.json directly (bypassing devenv's Claude integration)

Option 1 is preferred when the project uses devenv, as it keeps all configuration in one place.
