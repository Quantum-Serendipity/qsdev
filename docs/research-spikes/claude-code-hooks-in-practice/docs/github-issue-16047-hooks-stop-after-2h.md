# Bug: Hooks Stop Executing After ~2.5 Hours in Session
- **Source**: https://github.com/anthropics/claude-code/issues/16047
- **Retrieved**: 2026-03-27

## Summary
Claude Code hooks silently stop executing after approximately 2.5 hours in the same session, with no errors logged.

## Environment
- OS: Windows 11
- Shell: Git Bash (MINGW64)
- Claude Code Version: Latest (as of 2026-01-02)
- Hooks Config: Valid `.claude/hooks.json` (2683 bytes, 8 hooks configured)

## What Happens
1. Hooks execute successfully at session start
2. After ~2.5 hours, hooks silently stop firing
3. No error messages or warnings appear
4. The last log entry shows: "No stdin data, allowing operation" — suggesting hooks are invoked but not receiving tool use data

## Evidence
- 16:27-16:28: Hooks executing normally
- 16:50:47: Last hook entry (suspicious: "No stdin data")
- 19:15: Edit operation performed — NO hook fired (2 hours 25 minute gap)

## Root Cause Found
The issue was resolved by the user who discovered:
- Problem: `~/.claude/hooks.log` had grown to ~48GB, causing all hooks to fail silently when trying to write logs
- Solution: Delete the massive log file
- Result: Hooks immediately started working again

## Status: CLOSED (COMPLETED)
