# Bug: UserPromptSubmit Hooks Not Working from Subdirectories
- **Source**: https://github.com/anthropics/claude-code/issues/8810
- **Retrieved**: 2026-03-27

## Summary
UserPromptSubmit hooks defined in ~/.claude/settings.json fail to execute when Claude Code is started from subdirectories. PreToolUse hooks work from any directory.

## Root Cause
Using relative paths or ~ in hook commands. The ~ home directory expansion may not work correctly when Claude Code is launched from subdirectories.

## Fix
Always use absolute paths in hook commands.

## Status: Closed as duplicate of #7873
