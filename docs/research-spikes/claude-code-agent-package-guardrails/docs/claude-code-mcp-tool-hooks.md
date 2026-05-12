<!-- Source: https://code.claude.com/docs/en/hooks (MCP tool hooks section) -->
<!-- Retrieved: 2026-05-12 -->

# MCP Tool Hooks in Claude Code

## Overview

MCP tool hooks (`type: "mcp_tool"`) allow you to call tools on already-connected MCP servers as part of your hook handlers. They're available on every hook event once Claude Code has connected to your MCP servers.

## Configuration Fields

| Field    | Required | Description |
|----------|----------|-------------|
| `server` | yes      | Name of a configured MCP server (must already be connected) |
| `tool`   | yes      | Name of the tool to call on that server |
| `input`  | no       | Arguments passed to the tool. Supports `${path}` substitution |

## How MCP Tool Output Works

The tool's text content is treated like command-hook stdout:
- If it parses as valid JSON output, it is processed as a decision
- Otherwise it is shown as plain text
- If the named server is not connected, or the tool returns `isError: true`, the hook produces a **non-blocking error** and execution continues

## Example: Post-Tool Security Validation

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "mcp_tool",
            "server": "my_server",
            "tool": "security_scan",
            "input": { "file_path": "${tool_input.file_path}" }
          }
        ]
      }
    ]
  }
}
```

## Using MCP Tool Hooks with PreToolUse

You CAN use MCP tool hooks as PreToolUse hooks to validate before allowing actions:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "mcp_tool",
            "server": "validation_server",
            "tool": "validate_command",
            "input": { "command": "${tool_input.command}" }
          }
        ]
      }
    ]
  }
}
```

The MCP tool should return JSON with `permissionDecision: "deny"` to block, or omit decision fields to allow.

## Input Path Substitution

String values in `input` support `${path}` substitution:
- `${tool_input.file_path}` — file path from Write/Edit/Read
- `${tool_input.command}` — command from Bash
- Any other field from the JSON input structure

## Error Handling

MCP tool hooks are **non-blocking on failure**:
- Server not connected → non-blocking error, execution continues
- Tool returns `isError: true` → non-blocking error, execution continues
- Valid response → processed according to decision fields

**Critical limitation**: To enforce blocking behavior on failure, wrap the MCP tool call in a command hook that examines the output and exits with code 2 if needed.

## Timing Note

SessionStart and Setup typically fire before servers finish connecting, so hooks on those events should expect the "not connected" error on first run. Use PreToolUse or PostToolUse for reliable post-connection validation.
