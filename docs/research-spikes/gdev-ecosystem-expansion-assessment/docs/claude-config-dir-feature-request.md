# CLAUDE_CONFIG_DIR Feature Request
- **Source**: https://github.com/anthropics/claude-code/issues/25762
- **Retrieved**: 2026-05-14

## Request

Environment variable to control `.claude/` config directory location instead of hardcoded `~/.claude/`.

## Current Status

- Open feature request (enhancement)
- NOT officially supported
- No assignees

## Key Finding

`CLAUDE_CONFIG_DIR` is NOT an officially supported variable. Yaw Terminal uses it in their implementation, but it has known bugs:
- Issue #3833: Still creates local .claude/ directories
- Issue #4739: /ide command fails when set
- Issue #30538: VS Code extension ignores it

The variable exists in the codebase but is undocumented and partially implemented.
