# Three-Outcome Verdict Model (Allow/Deny/Ask) for gdev Hooks

## Research Question

How should gdev's self-protection hooks implement a three-outcome verdict system (allow/deny/ask), including verdict semantics, escalation priority when multiple rules match, rule matching architecture, Claude Code hook protocol integration, per-rule verdict assignment, and audit logging?

## Executive Summary

gdev's hook system should implement a **four-verdict model** (allow, deny, ask, warn) using Claude Code's native `permissionDecision` JSON response for structured control and exit code 2 as a fail-closed fallback. The combining algorithm is **deny-overrides** (matching Cedar, XACML best practice, and Claude Code's own implied precedence): any deny from any rule wins, ask escalates only when no rule denies, and warn proceeds with an advisory message. A critical bug in Claude Code (issue #39344, open) means `permissionDecision: "ask"` can silently override `permissions.deny` rules -- gdev must never rely on the ask verdict for operations that have a deny rule in settings.json.

The evaluation pipeline uses **all-match-then-escalate** (not first-match): every rule evaluates against the operation, verdicts are collected, and the most restrictive verdict wins. This matches XACML deny-overrides, Cedar's order-independent evaluation, and Claude Code's parallel hook execution model. Rules are organized into three tiers with strict precedence: managed-policy rules (gdev-compiled, non-overridable) > user rules (YAML overrides) > project rules (per-repo). Self-protection deny rules are non-overridable by design -- no user or project rule can downgrade a managed-policy deny to ask or allow.

Of the 23 rules identified across prior research (6 Prempti translations + 6 gdev-specific + 11 from threat model rule sets A-E), 13 are assigned deny verdicts and 10 are assigned ask verdicts. The assignment principle is: **deny** for operations that are never legitimate when performed by an agent (modifying settings.json, deleting hook scripts, killing security processes), **ask** for operations that are sometimes legitimate but carry security risk (editing devenv.nix, modifying .pre-commit-config.yaml, reading settings.json for reconnaissance).

---

## 1. Verdict Semantics

### 1.1 The Four Verdicts

gdev implements four verdict outcomes, extending Claude Code's three-option `permissionDecision` with an advisory "warn" verdict:

| Verdict | Meaning | User Experience | Agent Impact | Audit |
|---------|---------|-----------------|--------------|-------|
| **allow** | Operation proceeds | No visible indication | Agent continues normally | Minimal (rule ID logged if debug) |
| **deny** | Operation blocked | Error message in transcript explaining what was blocked and why | Agent sees the denial reason and adapts | Full (rule ID, operation details, deny reason) |
| **ask** | Operation paused for human approval | Permission dialog with context; user chooses Allow/Deny | Agent waits for user decision, then proceeds or adapts | Full (rule ID, operation details, user decision, response time) |
| **warn** | Operation proceeds with advisory | Warning message in transcript | Agent sees the warning as context | Full (rule ID, operation details, warning text) |

#### allow

The operation passes all rule checks and proceeds without interruption. This is the default verdict when no rule matches the operation. Logging is minimal -- only in debug mode to avoid noise. Implementation: hook exits 0 with no JSON output (implicit allow) or with `{"hookSpecificOutput":{"permissionDecision":"allow"}}`.

#### deny

The operation is hard-blocked. The agent receives a structured explanation that names the violated rule, explains what was blocked, and suggests legitimate alternatives (e.g., "Use `gdev enable hooks` to reconfigure hooks" or "Use `gdev hook bypass-next` for a one-time override"). The deny message is crafted for LLM consumption -- it guides the agent toward compliant behavior rather than just saying "no."

Implementation: two mechanisms available, chosen by failure tier:

| Mechanism | When to Use | Behavior on Hook Error |
|-----------|-------------|----------------------|
| Exit 2 + stderr | Fail-closed tier (self-protection, destructive prevention, credential scan) | Hook crash = deny (fail-closed) |
| Exit 0 + JSON deny | Structured control with rich context | Hook crash = allow (fail-open, Claude Code default) |

**Recommendation for gdev self-protection**: Use the **fail-closed wrapper pattern** from the fail-policy research. The wrapper catches all errors and falls through to exit 2. For the happy path (rule evaluated, deny decision made), output JSON deny via exit 0 to provide the rich `permissionDecisionReason` and `additionalContext` fields. For error paths (rule evaluation crashed), the trap fires exit 2 with a stderr message.

```bash
#!/usr/bin/env bash
set -euo pipefail
trap 'echo "gdev self-protection hook error -- blocking for safety. Run gdev doctor." >&2; exit 2' ERR EXIT

# ... rule evaluation logic ...

if [[ "$VERDICT" == "deny" ]]; then
    trap - EXIT  # Disarm the fail-closed trap
    jq -n --arg reason "$REASON" --arg context "$CONTEXT" '{
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": $reason,
            "additionalContext": $context
        }
    }'
    exit 0
fi

trap - EXIT  # Disarm the fail-closed trap
exit 0  # allow
```

#### ask

The operation is paused and the user is presented with a permission dialog. The dialog shows the tool name, tool input, and the hook's `permissionDecisionReason` as context. The user can approve (allow) or deny. Either outcome is logged.

Implementation: exit 0 with JSON:
```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "ask",
        "permissionDecisionReason": "The agent is requesting to edit .pre-commit-config.yaml, which contains security hooks. Confirm this is intentional."
    }
}
```

**Critical limitation (issue #39344)**: Claude Code has an open bug where a hook returning `permissionDecision: "ask"` silently overrides `permissions.deny` rules in settings.json. This means if an operation matches both a gdev hook returning "ask" AND a `permissions.deny` rule in settings.json, the deny rule is bypassed and the command executes without any prompt. **gdev must never use "ask" for operations that are also covered by deny rules in settings.json.** The ask verdict is safe only for operations that do not have a corresponding deny rule.

**Approval fatigue mitigation**: The ask verdict should be reserved for genuinely ambiguous operations. If more than ~5 asks fire per session, developers will habituate to approving everything, defeating the purpose. The per-rule verdict assignment (Section 5) is calibrated to keep ask verdicts rare -- most security-critical operations get deny, not ask.

#### warn (allow-with-warning)

The operation proceeds but the user sees an advisory message. This is not a Claude Code `permissionDecision` value -- it is implemented via the `systemMessage` field or by writing to stderr on exit 0:

```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "allow",
        "additionalContext": "Warning: this command modifies package manager security settings (.npmrc). The change will be logged."
    },
    "systemMessage": "gdev advisory: .npmrc modification detected. Security settings may be affected."
}
```

Alternatively, for simpler implementation:
```bash
echo "gdev advisory: .npmrc modification detected. Security settings may be affected." >&2
exit 0  # allow (non-blocking)
```

The warn verdict is useful for operations that are legitimate but deserve visibility: editing `.npmrc` (might remove `ignore-scripts=true`), modifying `CLAUDE.md` (might remove gdev sections), changing cost configuration. It occupies a space below "ask" (no user action required) but above "allow" (something noteworthy happened).

### 1.2 Verdict vs Exit Code Mapping

| gdev Verdict | Claude Code Mechanism | Exit Code | JSON Required? |
|-------------|----------------------|-----------|---------------|
| allow | No output or `permissionDecision: "allow"` | 0 | No |
| deny (structured) | `permissionDecision: "deny"` | 0 | Yes |
| deny (fail-closed) | stderr message | 2 | No |
| ask | `permissionDecision: "ask"` | 0 | Yes |
| warn | `permissionDecision: "allow"` + `additionalContext` / stderr | 0 | Optional |
| error (security tier) | Trap fires | 2 | No |
| error (advisory tier) | Non-zero (not 2) | 1 | No |

### 1.3 The `defer` Decision

Claude Code also supports `permissionDecision: "defer"`, which means "I have no opinion; let Claude Code's normal permission system decide." gdev should use `defer` as the **default verdict when no rule matches** an operation. This is equivalent to "not applicable" in XACML -- the hook has nothing to say about this operation, so it defers to Claude Code's built-in permission system.

In practice, gdev hooks that return no JSON output (exit 0 with empty stdout) already behave as `defer` -- Claude Code proceeds with its normal permission flow. Explicit `defer` is only needed when the hook wants to provide `additionalContext` to Claude without making a permission decision.

---

## 2. Escalation Priority (Verdict Combining)

### 2.1 The Combining Algorithm

When multiple rules match the same operation, their verdicts must be combined into a single decision. gdev uses **deny-overrides** (also called "most-restrictive-wins"):

```
deny > ask > warn > allow > defer
```

This is the standard combining algorithm for security-critical systems:

| Framework | Combining Algorithm | Equivalent |
|-----------|-------------------|------------|
| XACML | deny-overrides | deny > permit > not-applicable > indeterminate |
| Cedar (AWS) | forbid-overrides-permit | forbid > permit > default-deny |
| Claude Code (implied) | deny > defer > ask > allow | Same direction, different names |
| Prempti | deny > ask (2 outcomes only) | Subset of gdev's model |
| OPA | User-defined | Typically deny-overrides |

The algorithm is:

1. Collect verdicts from all matching rules
2. If any verdict is **deny**, the combined verdict is **deny** (with the reason from the highest-priority deny rule)
3. Else if any verdict is **ask**, the combined verdict is **ask** (with the reason from the highest-priority ask rule)
4. Else if any verdict is **warn**, the combined verdict is **warn** (all warning messages concatenated)
5. Else the combined verdict is **allow**

This is order-independent -- the same result regardless of which rule evaluates first. This matches Cedar's design principle and Claude Code's parallel hook execution.

### 2.2 Inter-Tier Precedence

gdev has three rule tiers with strict precedence:

| Tier | Source | Modifiable By | Override Behavior |
|------|--------|---------------|-------------------|
| **Managed-policy** | Compiled into gdev binary | Nobody (recompile required) | Cannot be overridden by user or project rules |
| **User** | `~/.qsdev/rules/self-protection.yaml` | Developer | Can override other user rules; cannot override managed-policy |
| **Project** | `.gdev.yaml` or project-level config | Project team | Can add rules; cannot override managed-policy or user rules |

The combining algorithm operates within a strict tier hierarchy:

1. **Managed-policy rules evaluate first.** If any managed-policy rule returns deny, the operation is blocked. No user or project rule can downgrade this to ask or allow.
2. **User rules evaluate second.** User rules can add deny/ask verdicts beyond managed-policy rules. A user rule can disable a non-security user rule (e.g., disable the "ask before reading settings.json" rule), but cannot disable a managed-policy deny rule.
3. **Project rules evaluate last.** Project rules can add additional protections but cannot weaken managed-policy or user rules.

This prevents a configuration poisoning attack where a malicious `.gdev.yaml` in a cloned repository downgrades self-protection rules from deny to allow.

### 2.3 Conflict Resolution: Self-Protection Deny vs User-Configured Allow

**Scenario**: A managed-policy rule denies writes to `~/.claude/settings.json`. A user has configured a rule in `~/.qsdev/rules/self-protection.yaml` that allows writes to `~/.claude/settings.json`.

**Resolution**: The managed-policy deny always wins. The user rule is ignored for this operation. This is by design -- the user cannot accidentally (or intentionally, if prompted by a compromised agent) weaken core self-protection rules.

**Escape hatch**: If the developer genuinely needs to modify settings.json while Claude Code is running:
1. `gdev hook bypass-next` -- one-time bypass with mandatory audit logging (the bypass is consumed on the next tool call)
2. `GDEV_HOOK_BYPASS=self-protection` -- session-level bypass via environment variable set before starting Claude Code (cannot be set by the agent, since the Bash guard blocks `export GDEV_HOOK_BYPASS`)
3. Edit the file directly in a terminal outside Claude Code (hooks only fire on Claude Code tool calls)

### 2.4 Conflict Resolution: Multiple Rules with Same Verdict

When multiple rules return the same verdict (e.g., two rules both deny), the combined message should aggregate the reasons:

```
BLOCKED by gdev self-protection (2 rules matched):
  1. sp-deny-settings-write: Cannot modify Claude Code settings (~/.claude/settings.json)
  2. sp-deny-qsdev-write: Cannot write to gdev installation directory (~/.qsdev/)
  
  Use 'gdev enable hooks' to reconfigure hooks, or 'gdev hook bypass-next' for a one-time override.
```

For ask verdicts from multiple rules, the permission dialog should show all reasons so the user can make an informed decision.

---

## 3. Rule Matching Architecture

### 3.1 Evaluation Pipeline

```
Tool Call
    |
    v
[1. Extract Operation Context]
    - tool_name (Bash, Write, Edit, Read, Task)
    - tool_input (command, file_path, content, prompt)
    - canonicalized paths (realpath resolution)
    |
    v
[2. Select Applicable Rules]
    - Filter by hook event (PreToolUse)
    - Filter by matcher (tool_name match)
    - Result: subset of rules that apply to this operation
    |
    v
[3. Evaluate All Applicable Rules]          <-- ALL rules, not first-match
    - Each rule evaluates independently
    - Each produces a verdict (allow/deny/ask/warn)
    - Rules within a tier run in parallel (no ordering dependency)
    |
    v
[4. Combine Verdicts]
    - Apply deny-overrides within each tier
    - Apply inter-tier precedence (managed > user > project)
    - Produce single combined verdict + aggregated reasons
    |
    v
[5. Execute Verdict]
    - deny: output JSON deny or exit 2
    - ask: output JSON ask
    - warn: output JSON allow + systemMessage / stderr
    - allow: exit 0 (no output)
    |
    v
[6. Audit Log]
    - Log: rule IDs matched, individual verdicts, combined verdict, operation context
    - For ask: log user's decision when it arrives (via PostToolUse or separate mechanism)
```

### 3.2 Why All-Match-Then-Escalate (Not First-Match)

| Approach | Pros | Cons |
|----------|------|------|
| **First-match** | Fast (stops at first hit), simple mental model | Order-dependent (reordering rules changes behavior), misses rules that should have fired, incomplete audit trail |
| **All-match-then-escalate** | Order-independent, complete audit trail, defense-in-depth (every applicable rule fires) | Slightly slower (evaluates all rules), more complex combining logic |

gdev uses all-match because:

1. **Order independence matches Claude Code's parallel execution.** Claude Code runs all matching hooks in parallel. gdev's rule evaluation within a single hook script should follow the same principle.
2. **Complete audit trail.** When a deny fires, the audit log should record ALL rules that matched, not just the first one. This is critical for debugging false positives and for SOC 2 evidence.
3. **Defense in depth.** If a bug in one rule causes it to miss a match, other rules covering the same operation still fire. First-match puts all trust in the first matching rule.
4. **Consistent with XACML deny-overrides and Cedar.** Both evaluate all applicable policies before combining.

### 3.3 Short-Circuit Optimization

While the logical model is all-match, the implementation can short-circuit for performance:

- **Within the managed-policy tier**: If a deny is found, skip remaining user and project rule evaluation (the deny cannot be overridden).
- **Within a tier**: Evaluate all rules (no short-circuit) for complete audit logging.
- **Total rules per operation**: Expected ~5-10 applicable rules per tool call. At <1ms per rule evaluation, total evaluation time is <10ms. Short-circuiting is an optimization, not a necessity.

### 3.4 Rule Structure

Each rule in the evaluation pipeline has:

```go
type EvaluatedRule struct {
    // Identity
    ID          string          // "sp-deny-settings-write"
    Tier        RuleTier        // ManagedPolicy | User | Project
    Description string          // Human-readable explanation
    
    // Matching
    HookEvent   string          // "PreToolUse"
    ToolMatcher string          // "Bash", "Write|Edit", "Read", "Task"
    Condition   RuleCondition   // Path match, command match, content match
    
    // Verdict
    DefaultVerdict Verdict      // deny | ask | warn
    Message        string       // LLM-friendly explanation
    
    // Metadata
    Enabled     bool            // Can be disabled by user override (for user/project tier only)
    Category    string          // "self-protection", "mcp-guard", "config-poisoning"
    FailureTier FailurePolicy   // FailClosed | FailOpen
}
```

---

## 4. Claude Code Hook Protocol Integration

### 4.1 The Four permissionDecision Values

Claude Code's `hookSpecificOutput.permissionDecision` field accepts four values:

| Value | gdev Usage | Behavior |
|-------|-----------|----------|
| `"allow"` | Explicit allow (rare -- typically implicit via exit 0) | Bypasses all further permission checks |
| `"deny"` | Hard block with structured reason | Claude sees `permissionDecisionReason`, cannot retry |
| `"ask"` | Human-in-the-loop escalation | Triggers permission dialog; user sees context from `permissionDecisionReason` |
| `"defer"` | Default when no rule matches | Defers to Claude Code's normal permission flow |

### 4.2 JSON Response Formats for Each Verdict

**deny (structured)**:
```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "deny",
        "permissionDecisionReason": "gdev self-protection: Cannot modify Claude Code settings (~/.claude/settings.json). Hook registrations and security configuration must not be modified by the agent. Use 'gdev enable hooks' to reconfigure, or 'gdev hook bypass-next' for a one-time override with audit logging.",
        "additionalContext": "Rule: sp-deny-settings-write | Category: self-protection | Tier: managed-policy"
    }
}
```

**ask**:
```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "ask",
        "permissionDecisionReason": "The agent is requesting to edit .pre-commit-config.yaml. This file contains security hooks (gitleaks, ripsecrets). Removing these hooks would disable credential scanning at git commit time. Is this edit intentional?"
    }
}
```

**warn (allow with advisory)**:
```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "allow",
        "additionalContext": "gdev advisory: This command modifies .npmrc. If ignore-scripts is changed to false, npm will execute arbitrary scripts during install. The change has been logged."
    },
    "systemMessage": "gdev advisory: .npmrc modification detected."
}
```

**defer (no opinion)**:
```json
{
    "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "defer"
    }
}
```

Or simply: `exit 0` with no stdout output (equivalent to defer).

### 4.3 Exit Code Integration

| Scenario | Exit Code | Stdout | Stderr | Claude Code Behavior |
|----------|-----------|--------|--------|---------------------|
| Rule matches, deny verdict | 0 | JSON deny | (empty) | Blocks tool call, shows reason to agent |
| Rule matches, ask verdict | 0 | JSON ask | (empty) | Triggers permission dialog |
| Rule matches, warn verdict | 0 | JSON allow + context | Advisory message | Proceeds, agent sees context |
| No rule matches | 0 | (empty) | (empty) | Proceeds normally (defer) |
| Hook evaluation error, security tier | 2 | (ignored) | Error message | Blocks tool call (fail-closed) |
| Hook evaluation error, advisory tier | 1 | (ignored) | Error message | Proceeds (fail-open) |

### 4.4 The ask Override Bug (Issue #39344) -- Design Implications

**Bug**: A PreToolUse hook returning `permissionDecision: "ask"` silently overrides `permissions.deny` rules in `settings.json`. The deny rule is bypassed -- the command executes without prompt or denial.

**Impact on gdev**: gdev uses `permissions.deny` rules (via Phase 4 settings.json management) AND PreToolUse hooks. If a gdev hook returns "ask" for an operation that also matches a `permissions.deny` entry, the deny is silently bypassed.

**Design mitigation**:
1. **Never use "ask" for operations covered by deny rules.** If an operation is dangerous enough to have a `permissions.deny` rule, gdev's hook should return "deny" (not "ask") for the same operation.
2. **Self-protection rules always use deny.** The operations blocked by self-protection (settings.json writes, hook script modifications) should never be "ask" -- they should be unconditional deny. This avoids the bug entirely for the most critical rules.
3. **Use "ask" only for operations NOT in deny lists.** The ask verdict is safe for operations like editing `.pre-commit-config.yaml`, modifying `devenv.nix`, or reading `settings.json` -- these are not covered by `permissions.deny` rules.
4. **Track bug resolution.** If/when #39344 is fixed, the mitigation can be relaxed. Until then, treat "ask" as potentially unsafe when deny rules are also in play.
5. **Test for the bug in `gdev doctor`.** Add a synthetic test that verifies a deny rule is not overridden by a hook's ask verdict.

### 4.5 How "ask" Works in Practice

When a hook returns `permissionDecision: "ask"`:

1. Claude Code pauses the tool execution
2. A permission dialog appears in the user's terminal
3. The dialog shows:
   - The tool name and its input (e.g., "Edit file: ~/.pre-commit-config.yaml")
   - The hook's `permissionDecisionReason` as context
   - Allow / Deny buttons
4. The user chooses Allow or Deny
5. If Allow: the tool call proceeds normally
6. If Deny: the tool call is blocked, and Claude sees a denial message
7. The `ask` also triggers a `PermissionRequest` hook event, which other hooks can intercept

**Context text**: Yes, a hook CAN provide context text that Claude Code shows to the user. The `permissionDecisionReason` field is shown in the permission dialog. gdev should craft these messages for human consumption -- clear, concise, explaining the risk.

---

## 5. Per-Rule Verdict Assignment

### 5.1 Assignment Principle

| Verdict | Criterion | Examples |
|---------|-----------|---------|
| **deny** | Operation is NEVER legitimate when performed by the agent | Modifying settings.json, deleting hook scripts, killing security processes |
| **ask** | Operation is SOMETIMES legitimate but carries security risk | Editing devenv.nix, modifying .pre-commit-config.yaml, registering MCP servers |
| **warn** | Operation is usually legitimate but deserves visibility | Editing .npmrc, modifying CLAUDE.md |
| **allow** | Operation is unambiguously safe | Default for non-matching operations |

The key question for each rule: **Is there ANY legitimate reason for the agent to do this?**
- If no: **deny**. The agent can never have a good reason to delete hook scripts or modify settings.json.
- If yes, but rarely: **ask**. The agent might legitimately need to edit devenv.nix, but changes to security-critical sections should be human-approved.
- If yes, usually: **warn**. The agent regularly edits CLAUDE.md, but removing gdev sections deserves a note.

### 5.2 Complete Rule Verdict Table

#### Prempti Translations (SP-1 through SP-6)

| ID | Rule | Default Verdict | Rationale |
|----|------|----------------|-----------|
| SP-1 | Deny gdev CLI invocation via Bash | **deny** | The agent never needs to invoke `gdev` commands. All gdev management is human-initiated. |
| SP-2 | Deny process-kill targeting gdev/hook infrastructure | **deny** | The agent never needs to kill security processes. Legitimate process management uses different tools. |
| SP-3 | Deny writes under `~/.qsdev/` (gdev install prefix) | **deny** | The agent never needs to modify gdev's installed files. All modifications go through `gdev` commands. |
| SP-4 | Deny writes to `~/.claude/settings.json` | **deny** | The agent should never modify the user-level settings file. Hook registration, deny rules, and permissions are managed by gdev or the human. |
| SP-5 | Deny writes to `~/.claude/policy-limits.json` | **deny** | Policy limits constrain agent resource consumption. The agent should never modify its own constraints. |
| SP-6 | Ask before reading `~/.claude/settings.json` | **ask** | Reading settings is not directly harmful but provides reconnaissance (hook names, deny patterns). Human should decide if this is appropriate. Can be disabled by user override if too noisy. |

#### gdev-Specific Rules (GSP-1 through GSP-6)

| ID | Rule | Default Verdict | Rationale |
|----|------|----------------|-----------|
| GSP-1 | Ask before editing devenv.nix security settings | **ask** | devenv.nix is a legitimate edit target, but changes that remove security tools or disable sandbox should be human-approved. Content inspection required (check for `sandbox = false`, removal of security tool references). |
| GSP-2 | Ask before editing .pre-commit-config.yaml | **ask** | Pre-commit config changes have high security impact (removing gitleaks/ripsecrets). The file is rarely edited, so ask-fatigue is low. |
| GSP-3 | Deny writes to `~/.config/nix/nix.conf` | **deny** | User-level Nix configuration affects all Nix operations. Adding untrusted substituters or disabling sandboxing is a systemic risk. The agent never needs to modify nix.conf. |
| GSP-4 | Ask before editing .gdev.yaml compliance/tools | **ask** | .gdev.yaml changes (compliance level downgrade, tool disabling) should be intentional. The agent might legitimately add new configuration keys, so blanket deny is too aggressive. Content inspection: ask only when compliance_level or enabled_tools change. |
| GSP-5 | Deny writes to `~/.qsdev/audit/` (audit trail) | **deny** | Audit logs must be tamper-resistant. Already covered by SP-3 (deny writes to ~/.qsdev/) but explicitly reinforced for Bash tool attempts targeting audit files. |
| GSP-6 | Ask before editing CLAUDE.md gdev sections | **ask** | CLAUDE.md is frequently edited. Only trigger ask when the edit removes gdev section markers (`<!-- gdev-managed -->`). If the edit does not touch gdev sections, allow silently. Content inspection on Edit tool: check if `old_string` contains gdev markers but `new_string` does not. |

#### Threat Model Rule Sets A-E

**Rule Set A: Protected Path Write Guard** (PreToolUse -- Write, Edit)

| Protected Path | Verdict | Rationale |
|---------------|---------|-----------|
| `~/.claude/settings.json` | **deny** | See SP-4 |
| `~/.claude/settings.local.json` | **deny** | Local overrides can weaken global settings |
| `~/.claude/policy-limits.json` | **deny** | See SP-5 |
| `~/.qsdev/hooks/*` | **deny** | See SP-3 |
| `~/.qsdev/cost-config.yaml` | **deny** | Raising cost thresholds to infinity effectively disables cost alerting |
| `~/.qsdev/audit/**` | **deny** | See GSP-5 |
| `.claude/settings.json` (project, gdev sections) | **ask** | Project settings may need legitimate edits; only gdev-managed sections need protection |
| `.mcp.json` | **ask** | MCP config changes are high-risk but sometimes legitimate (adding a development MCP server) |
| `CLAUDE.md` (gdev sections) | **ask** | See GSP-6 |
| `.claude/commands/*` | **ask** | Skill file injection is a known attack vector, but agents legitimately create commands |
| `.claude/rules/*` | **ask** | Rule file injection can override security instructions |
| `.claude/agents/*` | **ask** | Agent configuration can weaken security |

**Rule Set B: Protected Path Bash Guard** (PreToolUse -- Bash)

| Pattern Category | Verdict | Rationale |
|-----------------|---------|-----------|
| File write operations targeting protected paths (`sed -i`, `cat >`, `tee`, `echo >`, `cp`, `mv`, `dd`, `perl -pi`, `python -c`, `ruby -e`, `awk >`) | **deny** | Indirect write to protected files is equivalent to direct Write/Edit |
| Deletion operations targeting protected paths (`rm`, `unlink`, `truncate`) | **deny** | File deletion of security infrastructure is never legitimate for the agent |
| Permission operations targeting protected paths (`chmod`, `chown`) | **deny** | Permission changes can enable subsequent attacks (e.g., `chmod 000` makes hook non-executable, triggering fail-open) |
| Redirect operations targeting protected paths (`>`, `>>`) | **deny** | Catch-all for file write redirects |
| Bypass mechanism abuse (`export GDEV_HOOK_BYPASS`, `export GDEV_SKIP_ISOLATION_CHECK`) | **deny** | Agent should never set bypass environment variables |
| History destruction (`export HISTSIZE=0`, `history -c`) | **deny** | Audit trail preservation |
| Pre-commit hook disabling (`git config core.hooksPath`) | **deny** | Disabling pre-commit removes the git-commit-time security layer |
| Obfuscation patterns (`base64 -d \| bash`, `base64 -d \| sh`) | **deny** | Encoded command execution is a known attack vector |
| Process killing of security infrastructure (`pkill -f gdev`, `pkill -f falco`) | **deny** | See SP-2 |

**Rule Set C: Subagent Prompt Screen** (PreToolUse -- Task)

| Pattern | Verdict | Rationale |
|---------|---------|-----------|
| Mutation verbs + protected path references in Task prompt | **ask** | False positive rate is high for natural language screening. Ask verdict lets the human decide. A deny would prevent legitimate prompts that mention security topics. |

**Rule Set D: Runtime Integrity Check** (SessionStart)

| Check | Verdict on Failure | Rationale |
|-------|-------------------|-----------|
| Missing hook scripts | **warn** | Alert the developer; do not block session start (they need the session to fix it) |
| Modified hook checksums | **warn** | Tamper detection; warn and suggest `gdev doctor --fix` |
| Missing hook entries in settings.json | **warn** | Hook registration may have been removed; suggest `gdev enable hooks` |

**Rule Set E: Configuration Poisoning Guard** (PreToolUse -- Bash, Write, Edit)

| Pattern | Verdict | Rationale |
|---------|---------|-----------|
| .npmrc: `ignore-scripts=false` | **ask** | Removing ignore-scripts enables arbitrary code execution during `npm install`. Sometimes needed for legitimate development, but should be human-approved. |
| .npmrc: `min-release-age=0` | **warn** | Removing the age gate increases supply chain risk but may be needed for bleeding-edge development. Advisory only. |
| MCP from temp directory (`/tmp/...--stdio`) | **deny** | MCP server in temp directory is a strong malicious indicator. No legitimate use case. |
| MCP with IOC domain in URL | **deny** | Known malicious hosting domains. No legitimate use case. |
| MCP with base64-encoded commands | **ask** | Obfuscation is suspicious but base64 has legitimate uses (e.g., encoding JSON). Human should review. |
| `claude mcp add` / `claude mcp install` | **ask** | Agent self-registering MCP servers is suspicious but may be legitimate in development contexts. |
| `npx -y` with MCP/skill keywords | **ask** | Auto-accept npm execution with MCP keywords is risky but has legitimate uses. |
| Write to `.claude/commands/` | **ask** | Already covered in Rule Set A. |

### 5.3 Verdict Distribution Summary

| Verdict | Count | Percentage |
|---------|-------|------------|
| deny | 13 rules | 57% |
| ask | 10 rules | 43% |
| warn | 3 checks (Rule Set D) | Integrity checks only |

The deny-heavy distribution reflects the threat model: most self-protection rules guard operations that the agent should never perform. The ask rules cover the gray zone where the agent might have legitimate reasons.

### 5.4 Should Self-Protection Rules Allow Override via Ask?

**No, with one exception.** Managed-policy deny rules (SP-1 through SP-5, GSP-3, GSP-5, and all Rule Set B denials) should be non-overridable. The escape hatches are:

1. `gdev hook bypass-next` -- one-time bypass with audit logging
2. `GDEV_HOOK_BYPASS` environment variable -- session-level bypass set before Claude Code starts
3. Direct file editing outside Claude Code

The one exception is SP-6 (ask before reading settings.json): this can be disabled by user override in `~/.qsdev/rules/self-protection.yaml` because some developers find it too noisy and the risk (reconnaissance, not mutation) is lower.

---

## 6. Audit Logging Integration

### 6.1 Verdict Audit Record Schema

Every rule evaluation that produces a non-allow verdict should generate an audit record:

```json
{
    "timestamp": "2026-05-15T14:23:01Z",
    "session_id": "abc123",
    "event_type": "verdict",
    "rule_id": "sp-deny-settings-write",
    "rule_category": "self-protection",
    "rule_tier": "managed-policy",
    "tool_name": "Write",
    "tool_input_summary": {
        "file_path": "~/.claude/settings.json",
        "content_length": 1247
    },
    "verdict": "deny",
    "reason": "Cannot modify Claude Code settings",
    "combined_verdict": "deny",
    "all_matching_rules": ["sp-deny-settings-write", "sp-deny-qsdev-write"],
    "user_decision": null,
    "response_time_ms": null
}
```

For ask verdicts that are resolved by the user:

```json
{
    "timestamp": "2026-05-15T14:25:12Z",
    "session_id": "abc123",
    "event_type": "verdict_resolved",
    "rule_id": "gsp-2-precommit-config",
    "verdict": "ask",
    "user_decision": "allow",
    "response_time_ms": 3200,
    "reason": "User approved edit to .pre-commit-config.yaml"
}
```

### 6.2 What Gets Logged

| Verdict | Logged? | Fields |
|---------|---------|--------|
| allow | No (unless debug) | Rule ID only |
| deny | Yes | Full record: rule ID, category, tier, tool, input summary, reason |
| ask | Yes (twice) | Initial ask record + resolution record (user decision, response time) |
| warn | Yes | Full record with advisory message |
| error | Yes | Full record with error details, stack trace if available |

### 6.3 What Is NOT Logged (Privacy)

Following the SOC 2 audit trail design from Phase 32:
- File contents (never)
- Command output (never)
- Full command strings (truncated to 80 chars, secrets redacted)
- User prompt text (never)
- Conversation context (never)

### 6.4 Log Format and Location

Verdict audit records are appended to the same JSONL audit trail used by the SOC 2 audit hook (Unit 32.4):

```
~/.qsdev/audit/sessions/<date>/<session-id>.jsonl
```

This keeps all session audit data in one place. Verdict records are distinguished by `"event_type": "verdict"` or `"event_type": "verdict_resolved"`.

### 6.5 Audit Trail for Bypass Operations

When a bypass mechanism is used (`gdev hook bypass-next` or `GDEV_HOOK_BYPASS` env var), the audit record includes:

```json
{
    "timestamp": "2026-05-15T14:30:00Z",
    "session_id": "abc123",
    "event_type": "verdict_bypassed",
    "bypass_mechanism": "gdev hook bypass-next",
    "original_verdict": "deny",
    "rule_id": "sp-deny-settings-write",
    "tool_name": "Write",
    "tool_input_summary": {"file_path": "~/.claude/settings.json"},
    "reason": "Developer bypass (one-time)"
}
```

Bypass events are **always logged regardless of bypass**. This is the tamper-evident property: even when security is intentionally relaxed, the relaxation itself is recorded.

---

## 7. Comparison to Existing Systems

### 7.1 Verdict Model Comparison

| System | Verdicts | Combining Algorithm | Human-in-the-Loop | Order-Independent |
|--------|----------|--------------------|--------------------|-------------------|
| **gdev** | allow, deny, ask, warn | deny-overrides | Yes (ask verdict) | Yes |
| **Prempti** | allow, deny, ask | deny > ask > allow | Yes (ask verdict) | Yes (Falco engine) |
| **Cedar/AWS** | permit, forbid | forbid-overrides-permit | No | Yes |
| **XACML** | permit, deny, N/A, indeterminate | deny-overrides (configurable) | No | Configurable |
| **Microsoft AGT** | allow, deny | priority-based | No | Priority-dependent |
| **OPA** | User-defined | User-defined | Possible (undefined = review) | User-defined |
| **Claude Code native** | allow, deny, ask, defer | deny > defer > ask > allow (implied) | Yes (ask) | Yes (parallel) |

gdev's model is closest to Prempti's (both have the ask escalation) and Cedar's (both use forbid/deny-overrides with order independence). The warn verdict is unique to gdev and fills the gap between "ask" (requires action) and "allow" (silent).

### 7.2 Key Design Differences from Prempti

1. **Four verdicts vs three**: gdev adds "warn" for advisory-only operations. Prempti has only deny and ask (with allow as the implicit default).
2. **Tiered rule precedence**: gdev has managed-policy > user > project tiers. Prempti has a single rule set.
3. **Fail-closed wrapper**: gdev implements fail-closed at the hook level (exit 2 trap). Prempti implements fail-closed at the Falco level (daemon unavailable = all blocked).
4. **Per-rule failure policy**: gdev's tiered failure policy (fail-closed for security, fail-open for advisory) is more granular than Prempti's all-or-nothing fail-closed.

---

## Depth Checklist

- [x] **Underlying mechanism explained** -- Complete specification of four verdict semantics, two deny mechanisms (exit 2 vs JSON), JSON response formats, Claude Code protocol integration
- [x] **Key tradeoffs and limitations identified** -- Issue #39344 (ask overrides deny), approval fatigue risk, short-circuit vs complete evaluation, fail-closed vs structured deny
- [x] **Compared to alternatives** -- XACML combining algorithms, Cedar forbid-overrides-permit, OPA flexible decisions, Microsoft AGT binary model, Prempti three-verdict model, Claude Code native four-value system
- [x] **Failure modes and edge cases** -- Bug #39344 silent override, hook crash behavior per failure tier, multiple rules with same verdict (aggregation), managed-policy vs user rule conflict, bypass mechanism audit
- [x] **Concrete examples found** -- JSON response formats for every verdict, bash implementation sketch with fail-closed wrapper, complete rule verdict table (23 rules), audit record schema
- [x] **Report is standalone-readable** -- Complete specification sufficient for implementing the verdict model without consulting other reports

---

## Sources

### Internal (from prior research in this spike)
| File | Content |
|------|---------|
| `prempti-patterns-research.md` | 6 Prempti self-protection rules, gdev translations, rule format design |
| `threat-model-research.md` | 12 attack vectors, 5 rule sets A-E, defense coverage matrix |
| `fail-policy-research.md` | Severity-tiered failure policy, fail-closed wrapper pattern |

### External (saved to docs/)
| File | Content |
|------|---------|
| `docs/claude-code-hooks-reference.md` | Exit code semantics, timeout defaults, error handling |
| `docs/claude-code-hooks-reference-detailed.md` | permissionDecision values, JSON format, ask behavior, defer semantics |
| `docs/claude-code-hook-development-skill.md` | Hook development reference from Claude Code repo, parallel execution, matcher patterns |
| `docs/claude-code-issue-39344-ask-overrides-deny.md` | Critical bug: ask verdict silently overrides deny rules |
| `docs/claudefast-hooks-lifecycle-guide.md` | 12 hook lifecycle events, decision gaps |
| `docs/microsoft-agent-governance-toolkit-policy-model.md` | Binary verdict model, priority-based rules, Cedar/OPA/YAML support |
| `docs/aws-cedar-verified-permissions-terminology.md` | Cedar forbid-overrides-permit, order-independent evaluation |
| `docs/xacml-combining-algorithms-reference.md` | Deny-overrides, permit-overrides, first-applicable algorithms |
