# Bug: UserPromptSubmit Hooks from Plugins Match but Never Execute
- **Source**: https://github.com/anthropics/claude-code/issues/10225
- **Retrieved**: 2026-03-27

## Summary
UserPromptSubmit hooks defined in plugin hooks.json files are registered and matched correctly but their commands never execute. Silent failure — no errors logged.

## Environment
- Claude Code version: 2.0.24
- OS: macOS (Darwin 24.6.0)

## Key Finding
| Source | Register | Match | Execute |
|--------|----------|-------|---------|
| Plugin hooks.json | Yes | Yes | NO |
| Manual settings.json | Yes | Yes | Yes |

Only UserPromptSubmit is affected — Stop, PostToolUse, SessionStart work correctly from plugins.

## Workaround
Define hooks directly in ~/.claude/settings.json instead of plugin hooks.json.

## Status: Closed as duplicate of #9708
