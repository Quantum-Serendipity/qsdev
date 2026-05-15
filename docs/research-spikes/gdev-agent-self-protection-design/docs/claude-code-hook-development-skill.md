# Claude Code Hook Development Reference (SKILL.md)

- **Source URL**: https://raw.githubusercontent.com/anthropics/claude-code/main/plugins/plugin-dev/skills/hook-development/SKILL.md
- **Retrieved**: 2026-05-15

## Core Hook Types

Claude Code supports two primary hook implementations:

**Prompt-Based Hooks** use LLM reasoning for context-aware decisions.

**Command Hooks** execute bash scripts for deterministic validations, file operations, and external integrations.

## Configuration Formats

### Plugin hooks.json Structure
```json
{
  "description": "Optional explanation",
  "hooks": {
    "PreToolUse": [...],
    "Stop": [...]
  }
}
```

### Settings Format
User settings in `.claude/settings.json` use a direct format without wrapping:
```json
{
  "PreToolUse": [...],
  "Stop": [...]
}
```

## Hook Output Specifications

### Standard Response Format
```json
{
  "continue": true,
  "suppressOutput": false,
  "systemMessage": "Message for Claude"
}
```

### PreToolUse Specific Output
```json
{
  "hookSpecificOutput": {
    "permissionDecision": "allow|deny|ask",
    "updatedInput": {"field": "modified_value"}
  },
  "systemMessage": "Explanation"
}
```

### Stop/SubagentStop Responses
```json
{
  "decision": "approve|block",
  "reason": "Detailed explanation",
  "systemMessage": "Additional context"
}
```

## Hook Input Specifications

All hooks receive JSON via stdin:
```json
{
  "session_id": "abc123",
  "transcript_path": "/path/to/transcript.txt",
  "cwd": "/current/working/dir",
  "permission_mode": "ask|allow",
  "hook_event_name": "PreToolUse"
}
```

Event-specific fields:
- PreToolUse/PostToolUse: `tool_name`, `tool_input`, `tool_result`
- UserPromptSubmit: `user_prompt`
- Stop/SubagentStop: `reason`

## Exit Code Behavior

- `0`: Success; stdout displays in transcript
- `2`: Blocking error; stderr feeds back to Claude
- Other codes: Non-blocking errors

## Available Hook Events

- **PreToolUse**: Before any tool runs (approve, deny, modify)
- **PostToolUse**: After tool completion (react, log)
- **Stop**: Validates main agent completion
- **SubagentStop**: Validates subagent task completion
- **UserPromptSubmit**: Process user prompts
- **SessionStart**: Load context/environment
- **SessionEnd**: Cleanup and state preservation
- **PreCompact**: Preserve info before compaction
- **Notification**: React to notifications

## Environment Variables

- `$CLAUDE_PROJECT_DIR`: Project root path
- `$CLAUDE_PLUGIN_ROOT`: Plugin directory
- `$CLAUDE_ENV_FILE`: SessionStart only; persists environment variables
- `$CLAUDE_CODE_REMOTE`: Remote context indicator

## Tool Matcher Patterns

- `"Write"`: Exact tool match
- `"Read|Write|Edit"`: Multiple tools
- `"*"`: All tools
- `"mcp__.*"`: All MCP tools via regex
- Case-sensitive matching

## Execution Model

All matching hooks run in parallel, with non-deterministic ordering. Design hooks for independence rather than sequential execution.

## Permission Decision Precedence

When multiple hooks return different decisions, the precedence is: deny > defer > ask > allow. This means a deny from any hook will override ask or allow from other hooks.
