# Bug: WorktreeCreate/WorktreeRemove Hooks Silent Hang
- **Source**: https://github.com/anthropics/claude-code/issues/27467
- **Retrieved**: 2026-03-27

## Summary
WorktreeCreate hooks silently hang indefinitely when the hook command produces extra stdout beyond the expected worktree path. No error message or timeout — Claude just freezes.

## Environment
- Claude Code v2.1.50
- Linux (WSL2)

## Root Cause
When `git worktree add` prints status info to stdout (e.g., "HEAD is now at 1424fd1 ..."), Claude Code concatenates this with the echoed worktree path, cannot parse it, and hangs indefinitely.

## Related Issues
- #21992: Shell profile echo statements pollute hook stdout
- #27562: `claude --tmux --worktree` creates worktree but Claude never starts
- #27963: WorktreeCreate hook exits status 1 causes silent hang
- #27989: WorktreeCreate no-op hook causes silent hang
- #34457: Hooks with shell commands cause 5+ minute hangs on Windows

## Additional Problems Found
- Inline commands don't reliably receive stdin (shell profile sourcing consumes stdin before command runs)
- No timeout/error recovery mechanism for hung hooks

## Workaround
Redirect git output to stderr: `git worktree add ... >&2`

## Status: Closed as NOT_PLANNED (stale)
