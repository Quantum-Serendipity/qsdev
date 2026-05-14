# Feature Request: Silent Option for Async Hooks
- **Source**: https://github.com/anthropics/claude-code/issues/32954
- **Retrieved**: 2026-03-27

## Summary
Request for "silent": true option on async hooks to suppress status line messages. Every async hook completion displays "hook success: Success" messages, creating visual noise for background telemetry.

## Key Performance Data Point
Making hooks synchronous would add ~50-200ms latency per hook fire (reading stdin, parsing JSON, opening SQLite, writing data). Async is correct for passive telemetry.

## Use Case
agent-stalker plugin registers 11 async hooks across all major event types for observability. All produce no user-visible output but each generates a status message.

## Status: Closed as duplicate of #31595
