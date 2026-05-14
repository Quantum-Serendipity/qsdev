# Bug: Hooks Not Loading — /hooks Shows "No hooks configured"
- **Source**: https://github.com/anthropics/claude-code/issues/11544
- **Retrieved**: 2026-03-27

## Summary
Hooks defined in settings.json are not loaded by Claude Code v2.0.37+. The /hooks command displays "No hooks configured yet" despite valid JSON. Debug logs show "Found 0 hook matchers in settings" even when hooks are properly defined.

## Environment
- Claude Code Version: 2.0.37+ (confirmed in v2.0.46)
- Platform: macOS
- Installation: npm-global, Homebrew

## Key Observations
- JSON syntax is valid (verified with python3 -m json.tool)
- Other settings load correctly (mcpServers, permissions)
- Settings file IS being monitored (shown in debug logs)
- Hook commands work when executed manually
- THIS IS A REGRESSION — worked in previous versions
- Confirmed across multiple macOS installations

## Impact
Complete hook system failure — no hooks fire at all.

## Status: CLOSED
