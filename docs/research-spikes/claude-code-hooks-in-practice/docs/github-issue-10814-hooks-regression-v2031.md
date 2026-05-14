# Bug: Hooks Broken Again in v2.0.31 (Regression After v2.0.30 Fix)
- **Source**: https://github.com/anthropics/claude-code/issues/10814
- **Retrieved**: 2026-03-27

## Regression Timeline
| Version | Status |
|---------|--------|
| v2.0.25 | Working |
| v2.0.27 | Broken (original issue #10399) |
| v2.0.28-29 | Still broken |
| v2.0.30 | Fixed (#10401, Oct 31) |
| v2.0.31 | Broken again (Nov 1) |

## Impact
ALL hook types failed: PreCompact, SessionStart, PreToolUse, PostToolUse, UserPromptSubmit. Silent failures — no debug log entries.

## Root Cause
v2.0.31 changelog mentions no hook changes — unintentional regression, likely accidental revert of v2.0.30 fix.

## Anthropic Response
Collaborator @bcherny requested reproduction steps. Issue eventually closed as NOT_PLANNED (auto-close). Users confirmed hooks remained broken in later versions.

## Related Issues
- #10399: Original report (v2.0.27 broke hooks)
- #10401: Fix confirmed in v2.0.30
- #10450: Windows hook failures
- #11394: Only Notification hooks load from settings.json (v2.0.37)

## Status: Closed as NOT_PLANNED
