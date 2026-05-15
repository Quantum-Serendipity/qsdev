# Claude Code Hooks: Complete Technical Reference

- **Source URL**: https://code.claude.com/docs/en/hooks
- **Retrieved**: 2026-05-15

## Exit Code Behavior

### Exit Code Semantics

**Exit 0 (Success)**
- Claude Code parses stdout for JSON output fields
- JSON is only processed on exit 0
- For most events, stdout is written to debug log but not shown in transcript
- Exceptions: UserPromptSubmit, UserPromptExpansion, and SessionStart show stdout as context Claude can see

**Exit 2 (Blocking Error)**
- Claude Code ignores stdout and any JSON in it
- Stderr text is fed back to Claude as an error message
- Effect depends on the event
- This is the ONLY non-zero exit code that blocks actions

**Exit 1 and Other Exit Codes (Non-blocking Errors)**
- Treated as non-blocking errors for most hook events
- Execution continues despite the error
- Transcript shows a hook error notice followed by the first line of stderr
- Full stderr written to debug log
- CRITICAL: Exit 1 does NOT block actions even though it's the conventional Unix failure code

### Exit Code 2 Blocking Behavior per Event

| Hook Event | Can Block? | Exit 2 Behavior |
|---|---|---|
| PreToolUse | Yes | Blocks the tool call |
| PermissionRequest | Yes | Denies the permission |
| UserPromptSubmit | Yes | Blocks prompt processing and erases the prompt |
| Stop | Yes | Prevents Claude from stopping |
| SubagentStop | Yes | Prevents the subagent from stopping |
| PostToolUse | No | Shows stderr to Claude (tool already ran) |
| SessionStart | No | Shows stderr to user only |
| SessionEnd | No | Shows stderr to user only |

## Timeout Configuration

| Hook Type | Default Timeout |
|---|---|
| command | 600 seconds (10 minutes) |
| http | 600 seconds (10 minutes) |
| prompt | 30 seconds |
| agent | 60 seconds |

## Error Handling Behavior

### Command Hook Failures

**Timeout**: Hook execution is canceled after timeout seconds. Produces non-blocking error. Execution continues.

**Connection/Process Spawn Failure**: Hook cannot be executed. Produces non-blocking error.

**Malformed JSON Output on Exit 0**: If stdout cannot be parsed as valid JSON, treated as plain text context. Not processed as decision.

**Invalid JSON Schema on Exit 0**: Non-blocking error. Execution continues.

### Hook Crash Behavior

**Fail-Open (Default)**
- Most hook events produce non-blocking errors on crash
- Execution continues regardless
- Error shown in transcript or debug log
- Claude Code does not abort the session

**Fail-Closed (Blocking Events)**
- Events that support blocking follow the exit code rules above
- A crashing hook behaves like an unhandled exit code
- For blocking events, crashes produce errors but don't prevent the action unless exit code 2 is returned

### CRITICAL: Claude Code hooks are FAIL-OPEN by default
- Non-blocking hook events allow execution to continue even if the hook crashes or times out
- PreToolUse and similar blocking events will only actually block if they exit with code 2 or return appropriate JSON
- Crashes/timeouts on these events still produce errors but don't automatically block (fail-open by default)
- Exception: WorktreeCreate fails entirely on any non-zero exit code

## PreToolUse Decision Control

### Permission Decision Values

| Value | Behavior |
|---|---|
| allow | Permits the tool call to proceed |
| deny | Blocks the tool call. permissionDecisionReason required |
| ask | Escalates to the user with a permission dialog |
| defer | Defers the decision; Claude Code applies other rules |

## JSON Response Format

Exit 0 with JSON on stdout:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow|deny|ask|defer",
    "permissionDecisionReason": "string (required for deny)",
    "updatedInput": {},
    "additionalContext": "string"
  }
}
```
