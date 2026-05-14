# Claude Code Hooks Collection - karanb192/claude-code-hooks
- **Source**: https://github.com/karanb192/claude-code-hooks
- **Retrieved**: 2026-03-27

## Overview
A growing collection of useful Claude Code hooks. Copy, paste, customize. Ready-to-use hooks for Claude Code that enhance safety, automation, and notifications.

## Available Hooks

### Pre-Tool-Use Hooks

#### block-dangerous-commands
- **Matcher:** Bash
- **Purpose:** Blocks dangerous shell commands (rm -rf ~, fork bombs, curl|sh)
- **Configuration:**
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "node ~/.claude/hooks/block-dangerous-commands.js"
          }
        ]
      }
    ]
  }
}
```

#### protect-secrets
- **Matcher:** Read|Edit|Write|Bash
- **Purpose:** Prevents reading/modifying/exfiltrating sensitive files
- **Scope:** Works across multiple file operation types

### Post-Tool-Use Hooks

#### auto-stage
- **Matcher:** Edit|Write
- **Purpose:** Automatically git stages files after Claude modifies them
- **Benefit:** Streamlines version control workflows

### Notification Hooks

#### notify-permission
- **Matcher:** permission_prompt|idle_prompt
- **Purpose:** Sends Slack alerts when Claude needs input
- **Integration:** External notification service (Slack)

## Safety Levels Configuration

Hooks support three configurable security tiers:

| Level | Coverage | Use Case |
|-------|----------|----------|
| critical | Only catastrophic operations | Maximum flexibility |
| high | Catastrophic + risky operations | **Recommended** |
| strict | Catastrophic + risky + cautionary | Maximum safety |

Setup: Modify the `SAFETY_LEVEL` constant in each hook file.

## Utility Tools

### event-logger
- **Language:** Python
- **Purpose:** Logs all hook events to inspect payload structures
- **Use:** Debugging and discovering available event data before writing custom hooks

## Testing
- Command: `npm test`
- 262 passing tests across the collection

## Contributing Ideas
The repository identifies potential additions including branch protection, auto-formatting, cost tracking, Discord notifications, and rate limiting functionality.

**License:** MIT
