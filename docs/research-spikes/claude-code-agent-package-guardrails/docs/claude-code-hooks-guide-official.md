# Automate Workflows with Hooks (Official Guide)

- **Source URL**: https://code.claude.com/docs/en/hooks-guide
- **Retrieved**: 2026-05-12
- **Note**: Official Anthropic guide. Practical examples and configuration patterns.

---

## Overview

Hooks are user-defined shell commands that execute at specific points in Claude Code's lifecycle. They provide deterministic control over Claude Code's behavior, ensuring certain actions always happen rather than relying on the LLM to choose to run them.

## Hook Types Available

- `"type": "command"` — Run a shell command (most common)
- `"type": "http"` — POST event data to a URL
- `"type": "mcp_tool"` — Call a tool on a connected MCP server
- `"type": "prompt"` — Single-turn LLM evaluation
- `"type": "agent"` — Multi-turn verification with tool access (experimental)

## Key Example: Block Edits to Protected Files

```bash
#!/bin/bash
# protect-files.sh
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

Configuration:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/protect-files.sh"
          }
        ]
      }
    ]
  }
}
```

## How Multiple Hooks Combine

When multiple hooks match the same event, every hook runs to completion in parallel. One hook returning `deny` does not stop sibling hooks from executing. After all hooks finish, the most restrictive answer wins: `deny` overrides `ask`, which overrides `allow`.

## The `if` Field (v2.1.85+)

Uses permission rule syntax to filter hooks by tool name AND arguments:
```json
{
  "type": "command",
  "if": "Bash(git push *)",
  "command": "/path/to/pre-push-checks.sh"
}
```

For compound commands like `npm test && git push`, Claude Code evaluates each subcommand and fires the hook because `git push` matches.

## Prompt-Based Hooks

For decisions requiring judgment, `type: "prompt"` sends hook input to a Claude model:
- `"ok": true` — action proceeds
- `"ok": false` — for PreToolUse, the tool call is denied; reason returned to Claude

## Agent-Based Hooks (Experimental)

Spawn a subagent with Read, Grep, Glob tools for multi-turn verification. Same ok/reason response format as prompt hooks. Default 60s timeout, up to 50 tool-use turns.

## Hooks and Permission Modes

**Critical**: PreToolUse hooks fire **before** any permission-mode check. A hook returning `deny` blocks the tool even in `bypassPermissions` mode or with `--dangerously-skip-permissions`. This lets you enforce policy that users cannot bypass.

The reverse is NOT true: `"allow"` from a hook does not bypass deny rules from settings.

## Limitations

- Command hooks communicate through stdout, stderr, and exit codes only
- Hook timeout is 10 minutes by default
- PostToolUse hooks cannot undo actions (tool already executed)
- PermissionRequest hooks do not fire in non-interactive mode (`-p`)
- When multiple PreToolUse hooks return `updatedInput`, the last to finish wins (non-deterministic)
- Claude can create files via Bash tool, bypassing Edit/Write matchers — use Stop hooks or also match Bash

## Troubleshooting

- Stop hook infinite loops: check `stop_hook_active` field and exit early if true
- JSON validation failures: shell profile echo statements pollute stdout; guard with `[[ $- == *i* ]]`
- Hook not firing: check matcher case sensitivity, correct event type
- Debug: `claude --debug-file /tmp/claude.log` or `/debug` mid-session
