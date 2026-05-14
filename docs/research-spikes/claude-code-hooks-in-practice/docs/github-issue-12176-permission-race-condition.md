# Bug: PermissionRequest Hook Race Condition — Dialog Shows Despite "allow"
- **Source**: https://github.com/anthropics/claude-code/issues/12176
- **Retrieved**: 2026-03-27

## Summary
Race condition where PermissionRequest hooks returning "allow" still display permission dialogs if hook takes >1-2 seconds. Permission dialog is added to UI state BEFORE hook results are awaited.

## Root Cause
The permission system adds dialogs to UI state immediately, then runs hooks asynchronously in parallel. If the hook completes fast (<1.5s), the dialog is removed before rendering. If slow (>2s), the user sees the dialog despite the hook approving.

## Impact
- PermissionRequest hooks cannot be trusted for automated approvals
- CI/CD workflows fail unpredictably
- Non-deterministic behavior confuses users

## Workaround
Use permissions.allow rules or fast command hooks instead of prompt-type PermissionRequest hooks.

## Related
Issue #9575 explains Notification hooks have ~25% fire rate due to similar timing issues.

## Status: CLOSED (resolved)
