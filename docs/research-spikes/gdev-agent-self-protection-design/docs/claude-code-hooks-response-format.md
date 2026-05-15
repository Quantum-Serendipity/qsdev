<!-- Source: https://code.claude.com/docs/en/hooks -->
<!-- Retrieved: 2026-05-15 -->

# Claude Code Hook Response Formats and Decision Control

## JSON Output Structure

Hooks communicate decisions through JSON output on exit code 0.

### PreToolUse - Rich Permission Control

Most flexible decision system. Returns hookSpecificOutput with permission decision:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow|deny|ask|defer",
    "permissionDecisionReason": "explanation",
    "updatedInput": { "command": "modified command" },
    "additionalContext": "context for Claude"
  }
}
```

Permission Decision Values:
- "allow" — Tool call proceeds without user prompt
- "deny" — Tool call is blocked
- "ask" — User sees permission dialog
- "defer" — No hook decision; use default permission mode

### How Ask Verdict Works

When a PreToolUse hook returns permissionDecision: "ask":
1. Hook exits 0 with ask decision
2. Claude Code triggers the permission dialog
3. User sees the tool name, parameters, and hook's permissionDecisionReason
4. User clicks Allow or Deny
5. Tool call proceeds or is blocked based on user choice

### Permission Mode Interaction

| Mode | Behavior |
|------|----------|
| default | Hook can allow/deny/ask; user sees dialog if hook says ask |
| auto | Hooks can override denial with "ask" or "allow" |
| dontAsk | Hook "ask" is converted to "allow" (no dialog shown) |
| bypassPermissions | Hooks "ask" becomes "allow" automatically |

### PermissionRequest Hook Event

When a permission dialog appears, hooks can auto-approve on behalf of the user:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow|deny",
      "updatedInput": { "command": "npm run lint" },
      "addPermissionRule": {
        "matcher": "Bash(npm run *)",
        "decision": "allow"
      }
    }
  }
}
```

### Exit Code Behavior

- Exit 0: Parses JSON for decisions; plain stdout is context
- Exit 2: Blocks action, stderr shown as error (no JSON parsing)
- Other codes: Non-blocking error, first line of stderr shown

## Critical Note

Exit code 2 is the most reliable blocking mechanism — it stops the tool call before permission rules are evaluated, so the block applies even when an allow rule would otherwise let the call proceed. Hook decisions do not bypass permission rules: deny and ask rules are evaluated regardless of what a PreToolUse hook returns.
