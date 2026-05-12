<!-- Source: https://code.claude.com/docs/en/hooks-guide -->
<!-- Retrieved: 2026-05-12 -->

# Automate workflows with hooks

> Run shell commands automatically when Claude Code edits files, finishes tasks, or needs input.

Hooks are user-defined shell commands that execute at specific points in Claude Code's lifecycle. They provide deterministic control over Claude Code's behavior, ensuring certain actions always happen rather than relying on the LLM to choose to run them.

For decisions that require judgment, you can also use prompt-based hooks or agent-based hooks that use a Claude model to evaluate conditions.

## Hook Types

| Type | Description |
|------|-------------|
| `command` | Runs a shell command |
| `http` | POST event data to a URL |
| `mcp_tool` | Call a tool on an already-connected MCP server |
| `prompt` | Single-turn LLM evaluation |
| `agent` | Multi-turn verification with tool access (experimental) |

## Hook Events (All 24+)

| Event | When it fires | Can block? |
|-------|---------------|------------|
| SessionStart | When a session begins or resumes | No |
| Setup | When you start Claude Code with --init-only, or with --init or --maintenance in -p mode | No |
| UserPromptSubmit | When you submit a prompt, before Claude processes it | No |
| UserPromptExpansion | When a user-typed command expands into a prompt | Yes |
| **PreToolUse** | **Before a tool call executes** | **Yes** |
| PermissionRequest | When a permission dialog appears | Yes (allow/deny/ask) |
| PermissionDenied | When a tool call is denied by the auto mode classifier | No |
| PostToolUse | After a tool call succeeds | No |
| PostToolUseFailure | After a tool call fails | No |
| PostToolBatch | After a full batch of parallel tool calls resolves | No |
| Notification | When Claude Code sends a notification | No |
| SubagentStart | When a subagent is spawned | No |
| SubagentStop | When a subagent finishes | No |
| TaskCreated | When a task is being created | No |
| TaskCompleted | When a task is being marked as completed | No |
| Stop | When Claude finishes responding | No |
| StopFailure | When the turn ends due to an API error | No |
| TeammateIdle | When an agent team teammate is about to go idle | No |
| InstructionsLoaded | When a CLAUDE.md or .claude/rules/*.md file is loaded | No |
| ConfigChange | When a configuration file changes during a session | Yes |
| CwdChanged | When the working directory changes | No |
| FileChanged | When a watched file changes on disk | No |
| WorktreeCreate | When a worktree is being created | No |
| WorktreeRemove | When a worktree is being removed | No |
| PreCompact | Before context compaction | No |
| PostCompact | After context compaction completes | No |
| Elicitation | When an MCP server requests user input | No |
| ElicitationResult | After a user responds to an MCP elicitation | No |
| SessionEnd | When a session terminates | No |

## PreToolUse — The Key Hook for Package Security

PreToolUse hooks run after Claude creates tool parameters but before the tool executes. This is the only hook that can block actions via exit code 2.

### Hook Input (stdin JSON)

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

### Exit Code Behavior

* **Exit 0**: the action proceeds. Stdout added to Claude's context.
* **Exit 2**: the action is blocked. Stderr becomes Claude's feedback.
* **Any other exit code**: the action proceeds (with error notice in transcript).

### Structured JSON Output

For more control, exit 0 and print JSON to stdout:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Use rg instead of grep for better performance"
  }
}
```

Permission decision values:
* `"allow"`: skip the interactive permission prompt (deny rules still apply)
* `"deny"`: cancel the tool call and send the reason to Claude
* `"ask"`: show the permission prompt to the user as normal
* `"defer"`: available in non-interactive mode with -p flag

**Important**: Returning `"allow"` skips the interactive prompt but does not override permission rules. Deny rules from any settings scope always take precedence over hook approvals.

### Matchers

Without a matcher, a hook fires on every occurrence of its event. The matcher field filters by tool name for PreToolUse/PostToolUse hooks.

Example: Match only Bash tool calls:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "path/to/validation-script.sh"
          }
        ]
      }
    ]
  }
}
```

## Practical Examples

### Block edits to protected files

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

### Combine multiple hooks

When multiple hooks match the same event, every hook runs in parallel. For PreToolUse permission decisions, the most restrictive answer wins: deny overrides ask, which overrides allow.

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r .tool_input.command >> ~/.claude/bash.log"
          },
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/block-rm-rf.sh"
          }
        ]
      }
    ]
  }
}
```

## MCP Tool Hooks

A hook can directly call a tool on an already-connected MCP server using `"type": "mcp_tool"`. This allows hooks to delegate decisions to MCP server tools.

## Configuration Location

Hooks are configured in settings files:
- `~/.claude/settings.json` (global/user)
- `.claude/settings.json` (project)
- `.claude/settings.local.json` (local project, gitignored)

## Environment Variables Available to Hooks

- `CLAUDE_PROJECT_DIR` — the project root directory
- `CLAUDE_ENV_FILE` — path to environment file for Bash preamble
- Standard session info via stdin JSON (session_id, cwd, etc.)

## PermissionRequest Hook

Can auto-approve or auto-deny specific permission prompts:

```json
{
  "hooks": {
    "PermissionRequest": [
      {
        "matcher": "ExitPlanMode",
        "hooks": [
          {
            "type": "command",
            "command": "echo '{\"hookSpecificOutput\": {\"hookEventName\": \"PermissionRequest\", \"decision\": {\"behavior\": \"allow\"}}}'"
          }
        ]
      }
    ]
  }
}
```
