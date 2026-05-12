<!-- Source: https://code.claude.com/docs/en/hooks-guide -->
<!-- Retrieved: 2026-05-12 -->

# Automate workflows with hooks - Claude Code Official Documentation

> Run shell commands automatically when Claude Code edits files, finishes tasks, or needs input.

Hooks are user-defined shell commands that execute at specific points in Claude Code's lifecycle. They provide deterministic control over Claude Code's behavior, ensuring certain actions always happen rather than relying on the LLM to choose to run them.

## Hook Types

- `"type": "command"`: runs a shell command
- `"type": "http"`: POST event data to a URL
- `"type": "mcp_tool"`: call a tool on an already-connected MCP server
- `"type": "prompt"`: single-turn LLM evaluation
- `"type": "agent"`: multi-turn verification with tool access (experimental)

## PreToolUse Hooks

PreToolUse runs before a tool call executes. Can block it.

### Input JSON (example for Bash)

```json
{
  "session_id": "abc123",
  "cwd": "/Users/sarah/myproject",
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "npm test"
  }
}
```

### Exit Codes

- **Exit 0**: action proceeds. Stdout added to context for some events.
- **Exit 2**: action is blocked. Stderr becomes feedback to Claude.
- **Any other exit code**: action proceeds, error notice shown.

### Structured JSON Output

Exit 0 with JSON for fine-grained control:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Use rg instead of grep for better performance"
  }
}
```

permissionDecision values:
- `"allow"`: skip interactive prompt (deny/ask rules still apply)
- `"deny"`: cancel tool call, send reason to Claude
- `"ask"`: show permission prompt to user
- `"defer"`: available in non-interactive mode only

### Critical: Hook vs Permission Rule Precedence

Hook `"allow"` does NOT override permission rules. If a deny rule matches, the call is blocked even when hook returns "allow". If an ask rule matches, user is still prompted.

But: A blocking hook (exit 2) DOES take precedence over allow rules. A hook that exits with code 2 stops the tool call before permission rules are evaluated.

Summary: hooks can block what rules allow, but hooks cannot allow what rules deny.

## Compound Command Handling

Claude Code is aware of shell operators. A rule like `Bash(safe-cmd *)` won't give permission to run `safe-cmd && other-cmd`. Recognized separators: `&&`, `||`, `;`, `|`, `|&`, `&`, and newlines. A rule must match each subcommand independently.

## The `if` Field (v2.1.85+)

Filters hooks by tool name AND arguments using permission rule syntax:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "if": "Bash(git *)",
            "command": ".claude/hooks/check-git-policy.sh"
          }
        ]
      }
    ]
  }
}
```

The hook only spawns when a subcommand matches `git *`, or when command is too complex to parse.

## Multiple Hook Combination

When multiple hooks match, all run in parallel. For PreToolUse permission decisions, the most restrictive answer wins: deny > ask > allow.

## Hook Location Scope

| Location | Scope |
|---|---|
| `~/.claude/settings.json` | All your projects |
| `.claude/settings.json` | Single project, shareable |
| `.claude/settings.local.json` | Single project, gitignored |
| Managed policy settings | Organization-wide |
| Plugin hooks/hooks.json | When plugin is enabled |

## Block Edits to Protected Files (Example)

```bash
#!/bin/bash
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

PROTECTED_PATTERNS=(".env" "package-lock.json" ".git/")

for pattern in "${PROTECTED_PATTERNS[@]}"; do
  if [[ "$FILE_PATH" == *"$pattern"* ]]; then
    echo "Blocked: $FILE_PATH matches protected pattern '$pattern'" >&2
    exit 2
  fi
done

exit 0
```

## Process Wrappers (stripped before matching)

`timeout`, `time`, `nice`, `nohup`, `stdbuf`, bare `xargs`

NOT stripped: `direnv exec`, `devbox run`, `mise exec`, `npx`, `docker exec`

## Prompt-Based Hooks

For decisions requiring judgment, `type: "prompt"` sends data to a Claude model (Haiku by default). Model returns `"ok": true/false` with optional reason.

## Hook Lifecycle Events (Complete List)

SessionStart, Setup, UserPromptSubmit, UserPromptExpansion, PreToolUse, PermissionRequest, PermissionDenied, PostToolUse, PostToolUseFailure, PostToolBatch, Notification, SubagentStart, SubagentStop, TaskCreated, TaskCompleted, Stop, StopFailure, TeammateIdle, InstructionsLoaded, ConfigChange, CwdChanged, FileChanged, WorktreeCreate, WorktreeRemove, PreCompact, PostCompact, Elicitation, ElicitationResult, SessionEnd
