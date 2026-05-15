# Claude Code Hooks: Detailed Technical Reference (PreToolUse Decision Control)

- **Source URL**: https://code.claude.com/docs/en/hooks
- **Retrieved**: 2026-05-15
- **Note**: This is a more detailed extraction focused on decision control mechanics, complementing the earlier claude-code-hooks-reference.md

## permissionDecision Field Values

| Value | Behavior |
|-------|----------|
| **allow** | Permits the tool call to execute immediately without further checks |
| **deny** | Blocks the tool call; Claude sees a denial reason and may not retry |
| **ask** | Escalates to a permission dialog, showing the user the tool call for approval |
| **defer** | Allows execution but defers to Claude's normal permission flow for potential user prompts |

## Multiple Hooks & Conflicting Decisions

The documentation does NOT explicitly specify conflict resolution. Key facts:

- All matching hooks run in parallel
- Identical handlers are deduplicated automatically
- **Implied precedence**: deny > ask > defer > allow (inferred from behavior, not documented)

## JSON Response Format for PreToolUse

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Database writes are not allowed",
    "additionalContext": "Optional context for Claude",
    "updatedInput": {
      "command": "npm run lint"
    }
  },
  "continue": true,
  "suppressOutput": false,
  "systemMessage": "Optional warning for user",
  "terminalSequence": "Optional OSC sequence"
}
```

## 'ask' Decision Behavior

- Triggers the standard permission dialog UI
- Shows the user the tool call (tool name and input)
- User sees an "Allow / Deny" prompt
- The hook's `permissionDecisionReason` provides context in the dialog
- Firing `ask` causes a `PermissionRequest` event, which can trigger hooks on that event

## 'defer' Decision

- Allows the tool to execute
- Does not suppress Claude's normal permission flow
- Used when a hook validates format/safety but defers policy to standard permission modes
- Effectively "I have no opinion, let the normal system decide"

## Exit Code 2 vs JSON deny

| Aspect | Exit 2 | JSON deny |
|--------|--------|-----------|
| Exit code required | 2 | 0 |
| JSON processing | Ignored | Parsed |
| Reason shown to Claude | Via stderr | Via permissionDecisionReason |
| Additional context | No | Via additionalContext |
| Input modification | No | Via updatedInput |
| Decision granularity | Binary (block) | Rich (allow/deny/ask/defer) |

**Key**: Exit 2 is the fail-closed mechanism. JSON deny (exit 0 + JSON) is the structured control mechanism.

## Critical Bug: ask Overrides deny Rules (Issue #39344)

A PreToolUse hook returning `permissionDecision: "ask"` silently overrides `permissions.deny` rules in settings.json. This means a hook's ask verdict can bypass hard deny rules, which is a security vulnerability. Status: open as of retrieval date.
