<!-- Source: https://raw.githubusercontent.com/falcosecurity/prempti/main/hooks/claude-code/src/main.rs -->
<!-- Retrieved: 2026-05-15 -->

# Claude Code Interceptor Architecture

## Overview
The interceptor bridges Claude Code's PreToolUse hook and the Prempti plugin broker. It reads the hook JSON from stdin, wraps it in a wire-protocol envelope, sends it to the broker, and maps the broker's verdict back to Claude Code's hook response format.

## Input Processing

The interceptor reads up to 64KB from stdin containing the hook invocation. It performs minimal parsing -- extracting only the `tool_use_id` field for correlation purposes while preserving all other content as raw JSON to forward to the broker.

## Wire Protocol Request Structure

The `Request` struct wraps the incoming hook data:
- **version**: Protocol version (currently 1)
- **id**: The tool use ID for correlation
- **agent_name**: Hardcoded as "claude_code"
- **agent_pid**: Optional PID of the parent process (retrieved via platform-specific lookup)
- **event**: The complete hook JSON as `RawValue`

## Socket Communication

**Connection**: The interceptor connects via Unix domain socket (or Windows AF_UNIX equivalent) at a path determined by:
1. `PREMPTI_SOCKET` environment variable (if set), or
2. Platform default: `$HOME/.prempti/run/broker.sock` (Unix) or `%LOCALAPPDATA%/prempti/run/broker.sock` (Windows)

**Protocol**: The serialized request is transmitted with a trailing newline. On Unix, the interceptor calls `shutdown(Write)` to signal EOF, but gracefully tolerates "peer already disconnected" errors.

**Timeout Handling**: Configurable via `PREMPTI_TIMEOUT_MS` (default 5000ms, range 100-30000ms). Both read and write timeouts are set immediately after connection.

## Response Processing

The broker returns a single-line JSON response containing:
- **id**: Must match the request ID
- **decision**: One of "allow", "deny", or "ask"
- **reason**: Optional explanation

## Verdict Output Format

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "<allow|deny|ask>",
    "permissionDecisionReason": "<reason string>"
  }
}
```

## Error Handling Strategy

**Input Errors (exit code 2)**: Malformed JSON, oversized input, invalid UTF-8, or empty stdin.

**Broker Errors (fail-closed)**: Connection failures, timeouts, response validation failures -- all result in deny verdict with descriptive reason, exit code 0.

**Serialization Fallback**: If JSON serialization fails, writes a hardcoded deny literal.
