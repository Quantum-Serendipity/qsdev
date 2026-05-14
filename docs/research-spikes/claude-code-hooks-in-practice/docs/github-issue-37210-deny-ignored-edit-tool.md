# Bug: PreToolUse permissionDecision "deny" Ignored for Edit Tool
- **Source**: https://github.com/anthropics/claude-code/issues/37210
- **Retrieved**: 2026-03-27

## Summary
PreToolUse hooks returning permissionDecision "deny" were ignored for the Edit tool. Root cause: user error in hook implementation, not a Claude Code bug.

## CRITICAL FINDING: Two Exit Code Systems

There are TWO ways to deny in PreToolUse hooks, and they work DIFFERENTLY:

### Method 1: Exit Code 2 (simple block)
- Exit code 2 = tool call blocked
- stderr fed back to Claude as error message
- No JSON processing needed
- Claude sees it as a "blocked action"

### Method 2: JSON permissionDecision (structured denial)
- Exit code MUST be 0
- JSON output with hookSpecificOutput wrapper required
- permissionDecision: "deny" in the JSON
- Must include hookSpecificOutput wrapper or decision is ignored

### Common Mistake
Using exit code 2 WITH permissionDecision JSON. Exit code 2 = hook crash/error in the JSON pathway. Claude Code ignores JSON output when exit code is non-zero.

## Correct Format
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "Blocked by policy"
  }
}
```
With exit code 0.

## Related Issues
- #31592: Distinguish hook denial from hook error
- #33106: deny not enforced for MCP server tool calls
- #36286: PermissionDecision ignored in VS Code Extension
- #36059: permissionDecision no longer overrides ask rules

## Status: Closed as NOT PLANNED (user-side fix)
