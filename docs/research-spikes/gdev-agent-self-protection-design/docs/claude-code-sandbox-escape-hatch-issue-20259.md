<!-- Source: https://github.com/anthropics/claude-code/issues/20259 -->
<!-- Retrieved: 2026-05-15 -->

# Issue #20259: Make Sandbox Escape Hatch Opt-In and Add Audit Logging

## Issue Description

This GitHub issue proposes security improvements to Claude Code's sandbox escape mechanism.

### Current Problem

The `dangerouslyDisableSandbox` escape hatch is **enabled by default** (`allowUnsandboxedCommands: true`), creating several security concerns:

1. **User awareness gap**: Users enabling sandboxing may not realize Claude can automatically bypass it
2. **Approval fatigue risk**: Bypass requests use the "normal permissions flow" that users click through repeatedly, potentially without proper scrutiny
3. **No audit trail**: Zero logging when `dangerouslyDisableSandbox` is used
4. **Enterprise policy friction**: Organizations must explicitly set `allowUnsandboxedCommands: false` on every installation

### Proposed Solutions

#### 1. Change Default to Opt-In
```json
{
  "sandbox": {
    "allowUnsandboxedCommands": false  // Changed from true
  }
}
```

Users wanting automatic sandbox escaping must explicitly enable it.

#### 2. Add Distinct Approval UI for Sandbox Escapes

When `dangerouslyDisableSandbox` is requested, present a differentiated prompt that:
- Clearly states "This command will run OUTSIDE the sandbox"
- Shows what sandbox restrictions are being bypassed
- Requires distinct confirmation (not just normal "allow" button)

#### 3. Add Audit Logging for Sandbox Escapes

Log all `dangerouslyDisableSandbox` usages with:
- **Timestamp**: When the escape was requested
- **Command executed**: The exact command running unsandboxed
- **Reason for escape**: The sandbox failure that triggered it
- **User decision**: Whether approved or denied

### Current Workaround

Set `"allowUnsandboxedCommands": false` in settings.json, but this:
- Requires users to know the setting exists
- Must be configured per installation
- Provides no audit trail

### Use Case Scenario

A security-conscious developer enables sandboxing, but later when a build script fails due to sandbox restrictions, Claude automatically retries with `dangerouslyDisableSandbox`. The developer clicks "allow" on what appears to be a normal permission prompt, and the command runs with full system access—with no audit record of when/why the bypass occurred.

**With proposed changes**: Either the bypass cannot occur (default off), or a distinct alarming prompt explains the sandbox will be bypassed with full audit logging.

### Key Design Principle

Defense-in-depth configurations should be **secure by default**. Users should opt into reduced security, not opt out of it.

### Issue Status

**Closed as not planned** (with labels: area:core, area:security, area:tools, enhancement, has repro, stale)
