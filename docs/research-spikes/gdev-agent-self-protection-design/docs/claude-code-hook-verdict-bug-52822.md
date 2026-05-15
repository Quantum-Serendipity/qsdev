<!-- Source: https://github.com/anthropics/claude-code/issues/52822 -->
<!-- Retrieved: 2026-05-15 -->

# PreToolUse Hook permissionDecision Bug (Issue #52822)

## Issue Summary

Bug #52822 reports that PreToolUse hooks returning permissionDecision: "allow" fail to suppress the native permission prompt in Claude Code v2.1.119, despite logging successful JSON parsing.

## Key Findings

### The Problem
- A PreToolUse hook exits with code 0 and emits valid JSON with permissionDecision: "allow"
- Claude Code logs: "Successfully parsed and validated hook JSON output"
- But the native UI prompt still appears, requiring manual Yes/No confirmation
- This is a regression from v2.1.59 where it worked correctly

### Evidence
From debug logs: permissionDecisionMs shows delay of 60+ seconds (until manual prompt confirmation), not the hook response time (<1s), proving the hook decision is ignored.

### Impact
- Programmatic approval workflows cannot work reliably — hooks that should gate tool execution are ignored
- Audit/compliance logging via hooks cannot prevent execution — decision is made after hook processing
- Human-in-the-loop via hooks is compromised — cannot enforce "ask" verdict programmatically

### ask vs allow Verdicts
The issue only explicitly tests "allow" verdict (broken). The "ask" verdict status is unclear but likely also affected since the hook/permission-prompt systems appear decoupled in this version.

### Secondary Finding: --permission-mode bypassPermissions
When bypassPermissions CLI flag is set, the hook is not invoked at all (contradicts SDK docs).

## Related Issues
- #28812 — Same feature, confirmed working v2.1.59
- #41615 — Similar scope but limited to sensitive paths
- #13339 — VS Code extension ignores permissionDecision: "ask"

## Implications for gdev

This bug means gdev CANNOT rely solely on JSON-based permissionDecision for security enforcement. The exit code 2 mechanism (hard block) remains reliable. For deny verdicts, gdev should use exit code 2 (not JSON deny). For ask verdicts, the JSON approach may be unreliable in affected Claude Code versions — gdev needs a fallback strategy.
