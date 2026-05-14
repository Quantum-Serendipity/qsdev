# Bug: Hooks Configured but Not Executing Despite Being Detected
- **Source**: https://github.com/anthropics/claude-code/issues/5093
- **Retrieved**: 2026-03-27

## Summary
Hooks configured in ~/.claude.json are detected by the system but never execute. Root cause: wrong config file path.

## Root Cause
Hooks must be in:
- ~/.claude/settings.json (global)
- .claude/settings.json (project)
- .claude/settings.local.json (project, gitignored)

NOT in ~/.claude.json — this is a common user error.

## Related Duplicates
- #2891: Hooks not executing despite following documentation
- #3828: Hooks consistently ignored since 1.0.54
- #3706: Unexpected Execution Halt: No Hook Commands Processed

## Status: Closed as duplicate
