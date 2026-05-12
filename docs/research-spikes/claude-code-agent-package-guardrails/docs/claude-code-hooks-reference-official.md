# Claude Code Hooks Reference (Official Documentation)

- **Source URL**: https://code.claude.com/docs/en/hooks
- **Retrieved**: 2026-05-12
- **Note**: Content extracted from official Anthropic documentation. Redirected from docs.anthropic.com.

---

## Overview

Hooks are user-defined shell commands, HTTP endpoints, or LLM prompts that execute automatically at specific points in Claude Code's lifecycle. They enable workflow automation, security enforcement, and contextual environment management.

## Hook Lifecycle

Hooks fire at three cadences:
- **Once per session**: `SessionStart`, `SessionEnd`
- **Once per turn**: `UserPromptSubmit`, `Stop`, `StopFailure`
- **Every tool call**: `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PermissionRequest`, `PermissionDenied`, `PostToolBatch`

Additionally: `Setup`, `UserPromptExpansion`, `SubagentStart`, `SubagentStop`, `TaskCreated`, `TaskCompleted`, `TeammateIdle`, `PreCompact`, `PostCompact`, `Notification`, `ConfigChange`, `CwdChanged`, `FileChanged`, `WorktreeCreate`, `WorktreeRemove`, `InstructionsLoaded`, `Elicitation`, `ElicitationResult`

## PreToolUse

**When it fires**: After Claude creates tool parameters and before processing the tool call

**Matcher values** (tool_name):
- `Bash` - Shell commands
- `Edit` - File editing
- `Write` - File writing
- `Read` - File reading
- `Glob` - File globbing
- `Grep` - Pattern searching
- `Agent` - Subagent spawning
- `WebFetch` - Web content fetching
- `WebSearch` - Web searching
- `AskUserQuestion` - User questions
- `ExitPlanMode` - Plan approval
- `mcp__<server>__<tool>` - MCP tools

### Tool Input Schemas

**Bash**:
```json
{
  "command": "npm test",
  "description": "Run test suite",
  "timeout": 120000,
  "run_in_background": false
}
```

**Write**:
```json
{
  "file_path": "/path/to/file.txt",
  "content": "file content"
}
```

**Edit**:
```json
{
  "file_path": "/path/to/file.txt",
  "old_string": "original text",
  "new_string": "replacement text",
  "replace_all": false
}
```

### PreToolUse Decision Control

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny|allow|ask|defer",
    "permissionDecisionReason": "Destructive command blocked",
    "updatedInput": {
      "command": "npm run lint"
    },
    "additionalContext": "Why this was modified"
  }
}
```

**permissionDecision values**:
- `"allow"` - Proceed with tool call (but deny rules still override)
- `"deny"` - Block the tool call
- `"ask"` - Escalate to user for permission
- `"defer"` - Defer decision to next model call (non-interactive mode only)

**updatedInput**: Modify tool input before execution (e.g., sanitize Bash commands, rewrite to safer version)

## PostToolUse

**When it fires**: After a tool call succeeds

**Input schema**:
```json
{
  "session_id": "abc123",
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/file.ts",
    "content": "..."
  },
  "tool_response": {
    "success": true,
    "message": "File written successfully"
  }
}
```

**Decision control**:
```json
{
  "decision": "block",
  "reason": "Test suite must pass before proceeding",
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "File was generated. Run `bun generate` to refresh."
  }
}
```

## Hook Handler Types

### 1. Command Hooks
```json
{
  "type": "command",
  "command": "/path/to/script.sh",
  "args": [],
  "async": false,
  "asyncRewake": false,
  "shell": "bash",
  "timeout": 600,
  "if": "Bash(git *)",
  "statusMessage": "Validating..."
}
```

### 2. HTTP Hooks
```json
{
  "type": "http",
  "url": "http://localhost:8080/hooks/pre-tool-use",
  "headers": { "Authorization": "Bearer $MY_TOKEN" },
  "allowedEnvVars": ["MY_TOKEN"],
  "timeout": 30
}
```

### 3. MCP Tool Hooks
```json
{
  "type": "mcp_tool",
  "server": "my_server",
  "tool": "validate_input",
  "input": { "file_path": "${tool_input.file_path}" },
  "timeout": 60
}
```

### 4. Prompt Hooks
```json
{
  "type": "prompt",
  "prompt": "Is this safe to run? Command: $ARGUMENTS",
  "model": "claude-opus",
  "timeout": 30
}
```

### 5. Agent Hooks
```json
{
  "type": "agent",
  "prompt": "Validate tool call for security: $ARGUMENTS",
  "timeout": 60
}
```

## Exit Codes

- **Exit 0**: Success. Action proceeds. stdout parsed for JSON output.
- **Exit 2**: Blocking error. stderr fed back to Claude. Blocks tool call for PreToolUse.
- **Any other exit code**: Non-blocking error. Action continues. stderr in debug log.

## The `if` Field (v2.1.85+)

Uses permission rule syntax to filter hooks by tool name AND arguments:
```json
{
  "type": "command",
  "if": "Bash(git push *)",
  "command": "/path/to/pre-push-checks.sh"
}
```

Only spawns the hook process when the tool call matches the pattern. For compound commands like `npm test && git push`, evaluates each subcommand.

## Matcher Patterns

| Pattern Type | Example | Matches |
|---|---|---|
| `"*"`, `""`, omitted | fires on every occurrence | all |
| Letters, digits, `_`, `\|` | `Bash` or `Edit\|Write` | exact matches |
| Any other character | `mcp__memory__.*` | regex patterns |

## Settings File Locations

| Location | Scope | Shareable |
|---|---|---|
| `~/.claude/settings.json` | All projects | No |
| `.claude/settings.json` | Single project | Yes |
| `.claude/settings.local.json` | Single project | No |
| Managed policy settings | Organization-wide | Yes |

## Hook Deduplication and Ordering

- All matching hooks run in **parallel**
- Identical handlers deduplicated automatically
- When several hooks return `additionalContext`, Claude receives all values
- For PreToolUse permission decisions, the **most restrictive answer wins**: deny > ask > allow

## Security Considerations

1. PreToolUse hooks fire **before** any permission-mode check. A hook that returns `deny` blocks the tool **even in `bypassPermissions` mode**.
2. A hook returning `"allow"` does NOT bypass deny rules from settings. Hooks can tighten restrictions but not loosen them.
3. `allowManagedHooksOnly` in managed settings blocks all user/project hooks.
4. Exec form (with `args`) is safer than shell form for untrusted paths.
5. Only exit code 2 blocks; exit code 1 is a non-blocking error.

## Environment Variables

- `CLAUDE_PROJECT_DIR`: Project root
- `CLAUDE_PLUGIN_ROOT`: Plugin installation directory
- `CLAUDE_PLUGIN_DATA`: Plugin persistent data directory
- `CLAUDE_EFFORT`: Effort level
- `CLAUDE_ENV_FILE`: (SessionStart, CwdChanged, FileChanged) file path to persist env vars
