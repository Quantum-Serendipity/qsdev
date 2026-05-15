# Sense internal/setup/setup.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/setup/setup.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Package Purpose
The `setup` package generates configuration files for AI tools like Claude Code, Cursor, and Codex CLI. It enables these tools to discover and use Sense without manual configuration.

## Key Types

**Options struct**: Controls which tools to configure, with fields for tool selection and current-only mode.

**ToolResult struct**: Summarises what was written for a single tool with tool identifier and file paths.

**Result struct**: Aggregates results across all configured tools.

## Main Functions

- `Run()`: Detects installed tools and writes integration files
- `resolveTools()`: Determines which tools to configure based on detection or options
- `configureTool()`: Routes to tool-specific configuration functions
- `configureClaudeCode()`: Sets up MCP JSON, Claude settings, Markdown files, and skill/agent directories
- `configureCursor()`: Configures MCP and cursor rules files
- `configureCodexCLI()`: Sets up MCP and agents documentation
- `writeMCPJSON()`: Creates or merges the Sense MCP server entry into .mcp.json
- `writeClaudeSettings()`: Merges hooks and permissions into Claude settings
- `ParseTools()`: Converts comma-separated tool names into Tool values

## Configuration Details

The setup uses idempotent operations:
- JSON files are deep-merged
- Markdown uses marker comments
- Skill files are overwritten
- All write operations check for errors and return boolean indicators of file modifications

## What It Configures for Claude Code
1. `.mcp.json` — MCP server entry for sense
2. `.claude/settings.local.json` — hooks (pre-tool-use, pre-compact, subagent-start, session-start) and permissions
3. Markdown instructions files
4. Skill/agent directory files
