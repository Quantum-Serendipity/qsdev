# Claude Code Issue #39344: PreToolUse hook permissionDecision 'ask' silently overrides permissions.deny rules

- **Source URL**: https://github.com/anthropics/claude-code/issues/39344
- **Retrieved**: 2026-05-15

## Summary

A critical security vulnerability where a `PreToolUse` hook returning `permissionDecision: "ask"` silently bypasses `permissions.deny` rules in `settings.json`, allowing denied commands to execute without prompts or denial.

## The Bug

**What happens:**
- When a command matches a `permissions.deny` rule AND a PreToolUse hook returns `permissionDecision: "ask"`, the deny rule is completely ignored
- The command executes without any prompt or denial
- This is a silent security bypass

**Severity:** Critical -- deny rules should be non-overridable

## Reproduction Steps

### Step 1: Baseline (deny rule alone - works correctly)
```json
{
  "permissions": {
    "deny": ["Bash(printf REPRO42*)"]
  }
}
```
Result: Command `printf REPRO42` is **DENIED** (correct)

### Step 2: Bug (deny rule + hook returning "ask")
```json
{
  "permissions": {
    "deny": ["Bash(printf REPRO42*)"]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{
          "type": "command",
          "command": "bash /tmp/repro-hook.sh"
        }]
      }
    ]
  }
}
```

Hook returns:
```json
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask"}}
```

Result: Command `printf REPRO42` **EXECUTES** (deny rule bypassed -- BUG)

### Step 3: Control (hook returning "deny" - works correctly)
Changing hook to return `"deny"` instead of `"ask"` correctly **DENIES** the command.

## Expected Behavior

Permission evaluation order should be:
1. **Check `permissions.deny`** -- if matched, **DENY** (non-overridable, final decision)
2. Run hooks -- apply hook decisions
3. Check `permissions.allow` / `permissions.ask`

A hook's `"ask"` decision should only escalate allowed commands to prompts -- it should **never** override deny rules.

## Resolution Status

The issue appears to be OPEN -- no resolution or fix is documented. The issue was marked with labels: `area:hooks`, `area:permissions`, `area:security`, `bug`, and `has repro`, indicating it was properly categorized but not yet closed at time of retrieval.

## gdev Implications

This is critical for gdev's verdict model design: gdev's self-protection hooks should NEVER use `permissionDecision: "ask"` for operations that are also covered by `permissions.deny` rules, because the ask verdict may silently override the deny rule. gdev should use its own deny (exit 2) for hard blocks, and only use "ask" for operations not covered by deny rules.
