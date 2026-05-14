# Bug: Plugin Hooks Not Firing in VSCode Extension
- **Source**: https://github.com/anthropics/claude-code/issues/18547
- **Retrieved**: 2026-03-27

## Summary
Hooks defined in plugin hooks/hooks.json are not loaded by the VS Code extension. Hooks only fire when defined directly in .claude/settings.json or ~/.claude/settings.json. CLI loads plugin hooks correctly — this is VS Code extension-specific.

## Status: OPEN (as of 2026-03-27)
- 6 upvotes
- Labels: bug, area:ide, has repro, platform:macos, stale

## Related Issues (long chain of duplicates/related)
- #10997, #11509, #12649, #11544, #6305
- #16288: Plugin hooks not loaded from external hooks.json
- #16114: Notification hooks not working in VS Code (work in CLI)
- #18900: Plugin hooks loaded but not included in hook registry lookups
- #20062: PreToolUse hooks with additionalContext not working
- #21736: Feature Request: Hooks support in VS Code

## Impact
- Plugin portability broken
- Plugin marketplace hooks may not work in VS Code
- Users must manually copy hooks from plugin to settings.json
- Blocking Supercharge-AI PreToolUse hook implementation
