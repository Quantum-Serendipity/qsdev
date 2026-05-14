# Claude Code Hooks Reference — Complete Documentation
- **Source**: https://code.claude.com/docs/en/hooks
- **Retrieved**: 2026-03-27
- **Type**: Official documentation (full reference)

## Overview

Hooks are user-defined shell commands, HTTP endpoints, LLM prompts, or agents that execute automatically at specific points in Claude Code's lifecycle.

## Hook Lifecycle

```
SessionStart → UserPromptSubmit → [Agentic Loop] → Stop/StopFailure → SessionEnd

Agentic Loop:
  PreToolUse → PermissionRequest → PostToolUse/PostToolUseFailure
  SubagentStart/Stop
  TaskCreated/TaskCompleted
  TeammateIdle
```

## Configuration Structure

```json
{
  "hooks": {
    "EVENT_NAME": [
      {
        "matcher": "pattern",
        "hooks": [
          {
            "type": "command|http|prompt|agent",
            "timeout": 600,
            "statusMessage": "Custom message",
            "once": false
          }
        ]
      }
    ]
  }
}
```

## Handler Types

### Command Hook
```json
{"type": "command", "command": "script.sh", "async": false, "shell": "bash", "timeout": 600}
```

### HTTP Hook
```json
{"type": "http", "url": "http://localhost:8080/hook", "headers": {"Authorization": "Bearer $MY_TOKEN"}, "allowedEnvVars": ["MY_TOKEN"], "timeout": 30}
```

### Prompt Hook
```json
{"type": "prompt", "prompt": "Analyze this: $ARGUMENTS", "model": "claude-opus", "timeout": 30}
```

### Agent Hook
```json
{"type": "agent", "prompt": "Verify this condition: $ARGUMENTS", "timeout": 60}
```

## Exit Code Behavior

| Exit Code | Meaning | JSON Processing |
|-----------|---------|-----------------|
| 0 | Success | Parse stdout for JSON output |
| 2 | Blocking error | Ignore stdout; use stderr as feedback |
| Other | Non-blocking error | Show stderr in verbose mode |

**Can block:** PreToolUse, PermissionRequest, UserPromptSubmit, Stop, SubagentStop, TeammateIdle, TaskCreated, TaskCompleted, ConfigChange, Elicitation, ElicitationResult, WorktreeCreate

**Cannot block:** PostToolUse, PostToolUseFailure, Notification, SubagentStart, SessionStart, SessionEnd, CwdChanged, FileChanged, PreCompact, PostCompact, WorktreeRemove, StopFailure, InstructionsLoaded

## Common Input Fields (All Events)

```json
{
  "session_id": "string",
  "transcript_path": "/path/to/transcript.jsonl",
  "cwd": "/current/working/directory",
  "permission_mode": "default|plan|acceptEdits|auto|dontAsk|bypassPermissions",
  "hook_event_name": "EventName"
}
```

## Key Event Schemas

### SessionStart
Matchers: startup, resume, clear, compact
Output: additionalContext, environment variable persistence via CLAUDE_ENV_FILE

### PreToolUse
Matchers: Tool name (Bash, Edit|Write, mcp__*, etc.)
Input includes: tool_name, tool_input (tool-specific fields), tool_use_id
Output: hookSpecificOutput.permissionDecision (allow|deny|ask), updatedInput, additionalContext

### PostToolUse
Matchers: Tool name
Input includes: tool_name, tool_input, tool_response, tool_use_id
Output: decision (block), additionalContext

### Stop
No matchers
Input: stop_hook_active, last_assistant_message
Output: decision (block), reason

## Hook Matcher Reference

| Event | Matcher Field | Example Values |
|-------|---------------|-----------------|
| PreToolUse/PostToolUse | tool_name | Bash, Edit\|Write, mcp__.* |
| SessionStart | source | startup, resume, clear, compact |
| SessionEnd | reason | clear, resume, logout, prompt_input_exit |
| Notification | notification_type | permission_prompt, idle_prompt |
| SubagentStart/Stop | agent_type | Bash, Explore, Plan |
| PreCompact/PostCompact | trigger | manual, auto |
| ConfigChange | source | user_settings, project_settings |
| FileChanged | basename | .envrc, .env |
| StopFailure | error | rate_limit, authentication_failed |

## Environment Variables

- $CLAUDE_PROJECT_DIR — project root
- $CLAUDE_PLUGIN_ROOT — plugin root (changes on updates)
- $CLAUDE_PLUGIN_DATA — plugin persistent data
- $CLAUDE_ENV_FILE — path to write persistent env vars

## Disable Hooks

```json
{"disableAllHooks": true}
```

Managed hooks cannot be disabled at user/project level.
