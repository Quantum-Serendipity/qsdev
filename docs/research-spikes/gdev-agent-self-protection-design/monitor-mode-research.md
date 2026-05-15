# Monitor/Shadow Mode Design for gdev Self-Protection Hooks

## Executive Summary

gdev should implement a per-rule monitor mode that evaluates all hook logic normally but overrides the verdict to allow, logging what would have been blocked with full diagnostic context. The design draws on six established security systems (SELinux permissive, AppArmor complain, Windows Defender ASR audit, AWS WAF count, seccomp SECCOMP_RET_LOG, Kubernetes ValidatingAdmissionPolicy warn/audit) that all share the same core pattern: evaluate fully, log the would-block decision, allow the operation to proceed. The key design decisions are per-rule granularity (not global-only), a recommended 5-day calibration period with an auto-prompt at expiry, JSONL logging unified with the enforcement audit trail (distinguished by a `mode` field), and a CLI-driven transition workflow (`gdev hook monitor`, `gdev hook audit`, `gdev hook enforce`).

---

## 1. Mode Semantics

### 1.1 What Monitor Mode Means

When a rule is in monitor mode:

1. **Full evaluation**: The rule evaluates its conditions (path matching, command regex, content inspection) exactly as it would in enforce mode. No logic is skipped.
2. **Override verdict**: Regardless of whether the rule would deny or ask, the hook returns allow (exit 0 with no blocking JSON).
3. **Full logging**: Every would-block or would-ask event is logged with complete context: the rule ID, the original verdict that would have applied, the operation details, the matched pattern, and a human-readable explanation.
4. **Advisory warning**: For would-deny verdicts, a brief non-blocking warning is printed to stderr so the developer sees it in the Claude Code transcript. For would-ask verdicts, a notice is logged but no warning is shown (to avoid noise).
5. **No workflow interruption**: The developer's work is never paused or blocked by a rule in monitor mode.

### 1.2 The Three Rule Modes

| Mode | Verdict Applied | Logged | User Sees |
|------|----------------|--------|-----------|
| **enforce** | deny/ask/allow as evaluated | Yes (all decisions) | Blocked or prompted on deny/ask |
| **monitor** | always allow (override) | Yes (with would-have-been verdict) | Brief advisory warning on would-deny |
| **off** | rule not evaluated | No | Nothing |

### 1.3 Interaction with Severity-Tiered Fail Policy

The fail-policy research (fail-policy-research.md) established a severity-tiered failure policy: fail-closed for security-critical hooks, fail-open for advisory hooks. Monitor mode interacts with this as follows:

- **Monitor mode overrides the fail policy**. When a rule is in monitor mode, even if it is classified as fail-closed (security-critical), it will not block on error. Instead, errors in monitor-mode rules are logged with a `monitor_error` entry and the operation proceeds. The rationale: monitor mode is explicitly a calibration period where the developer has accepted that no blocking will occur.
- **Monitor mode does NOT change the rule's severity tier**. When the rule transitions to enforce mode, it resumes its configured fail policy (fail-closed for security-critical). The severity tier is a property of the rule definition, not the current mode.
- **During monitor mode, if the hook script itself crashes**, the crash is logged as a `monitor_hook_error` event rather than triggering fail-closed behavior. This is valuable for debugging: the developer sees that the hook would have crashed in production, giving them a chance to fix it before enforcement.

### 1.4 Monitor Mode vs Off Mode

A rule that is "off" is not evaluated at all — no patterns are matched, no paths are resolved, no logs are generated. Monitor mode is distinct: the rule runs completely, generating the same performance overhead and the same audit data as enforce mode. The difference is only in the final verdict delivery.

This distinction matters for calibration. A rule in monitor mode reveals its false positive rate, its performance characteristics, and its interaction with real developer workflows. A rule that is "off" reveals nothing.

---

## 2. Prior Art Deep Dive

### 2.1 SELinux Permissive Mode

**Mechanism**: SELinux supports both global permissive mode (`setenforce 0`) and per-domain permissive mode (`semanage permissive -a <domain_t>`). In permissive mode, access decisions are evaluated normally against the loaded policy, but denials generate AVC (Access Vector Cache) log entries rather than blocking the operation.

**Granularity**: Per-domain. Individual process types can be marked permissive while the rest of the system remains in enforcing mode. This is the gold standard for granular monitor mode — you can enforce tested policy for 90% of domains while calibrating new policy for the remaining 10%.

**Logging**: AVC denials are logged to the audit subsystem (`/var/log/audit/audit.log`). The log entries are identical in format to enforcing-mode denials, which means the same tooling (`audit2allow`, `sealert`) can analyze both. The only difference is the `permissive=1` flag in the AVC record.

**Transition workflow**: `setenforce 1` for global, `semanage permissive -d <domain_t>` to remove per-domain permissive. No auto-expiration — permissive mode persists until manually changed or until reboot (for `setenforce`-only changes).

**Duration**: No built-in maximum duration. Permissive mode persists indefinitely. The only automatic reversion is that `setenforce 0` does not survive a reboot if the config file says `SELINUX=enforcing`. There is no reminder, prompt, or timeout mechanism.

**Key lesson for gdev**: Per-domain permissive is the right model — it allows targeted calibration without disabling enforcement for already-validated rules. SELinux's identical log format between permissive and enforcing is excellent: one tool analyzes both.

### 2.2 AppArmor Complain Mode

**Mechanism**: AppArmor profiles operate in either enforce or complain mode. In complain mode, the profile is loaded and evaluated, but violations are logged rather than blocked. The mode is set per-profile using `aa-complain <profile>`.

**Granularity**: Per-profile (equivalent to per-application). Each application's security profile can independently be in enforce or complain mode. There is no global complain toggle — you set each profile individually.

**Logging**: Violations are logged to `/var/log/audit/audit.log` (if auditd is running) or `/var/log/syslog`. The `aa-logprof` tool reads these logs and interactively suggests profile rules to add. This log-to-suggestion pipeline is a key part of the workflow.

**Transition workflow**: 
1. Deploy new profile in complain mode: `aa-complain /path/to/profile`
2. Exercise the application through normal usage
3. Run `aa-logprof` to review logged violations and build rules
4. Iterate until no new violations appear during comprehensive testing
5. Switch to enforce: `aa-enforce /path/to/profile`

**Important caveat**: Deny rules in AppArmor profiles are enforced even in complain mode. This means a developer can have a "mostly-complain" profile that still hard-blocks certain critical operations. This is a useful pattern for gdev: even in monitor mode, certain absolute rules (like "never write malware") could remain enforced.

**Duration**: No auto-expiration. Complain mode persists until manually changed.

**Key lesson for gdev**: The `aa-logprof` interactive review tool is the killer feature. gdev needs an equivalent: `gdev hook audit` that reads monitor-mode logs and presents would-have-blocked events for review. The "deny rules enforce even in complain" pattern is worth adopting for gdev's highest-severity rules.

### 2.3 Windows Defender ASR Audit Mode

**Mechanism**: Each Attack Surface Reduction rule can be independently set to Off (0), Block (1), Audit (2), or Warn (6). Audit mode evaluates the rule and logs what would have been blocked without affecting end users. Warn mode shows a user notification but does not block.

**Granularity**: Per-rule. Each ASR rule (identified by GUID) has its own mode setting. This is finer-grained than SELinux (per-domain) or AppArmor (per-profile). Microsoft recommends enabling all rules in Audit mode simultaneously during the testing phase.

**Logging**: Audit-mode triggers generate Event ID 1122 in the Windows Defender Operational log. Block-mode triggers generate Event ID 1121. The 30-day reporting page in the Defender portal shows both audit and block events, with filtering by rule, device, user, and source application.

**Transition workflow**: Microsoft's recommended deployment follows a ring-based approach:
1. Enable all rules in Audit mode on Ring 1 (champion users/devices)
2. Review the Detections report for 2-4 weeks
3. Build exclusion list for legitimate business processes (false positives)
4. Transition individual rules from Audit to Block
5. Expand to Ring 2, Ring 3, etc.

**Duration**: Microsoft recommends 2-4 weeks of audit data collection before transitioning to Block. No auto-expiration is built in. The admin must manually change each rule's mode.

**Per-rule exclusions**: ASR supports per-rule exclusions, so a rule can block most things while excluding specific known-safe processes. This is the tuning output from the audit phase.

**Key lesson for gdev**: The per-rule granularity, the 2-4 week recommended audit duration, and the ring-based deployment pattern all map directly to gdev's needs. The three-state model (audit/warn/block) is more nuanced than binary monitor/enforce — gdev should consider whether "warn" (show notification but allow) is distinct from "monitor" (log but no notification).

### 2.4 AWS WAF Count Mode

**Mechanism**: Individual WAF rules or entire rule groups can be set to Count action instead of Block. Count evaluates the rule and increments metrics but does not terminate the request. Counted matches appear under `nonTerminatingMatchingRules` in WAF logs.

**Granularity**: Per-rule and per-rule-group. You can override an entire managed rule group to Count mode, or override individual rules within a group. This dual-level granularity (group-level and rule-level) is the most flexible model in the survey.

**Logging**: Counted rules appear in WAF access logs under `nonTerminatingMatchingRules`. They generate CloudWatch metrics (but only Rule/RuleGroup/Region dimensions, not WebACL-level metrics — a notable limitation). CloudWatch Logs Insights can be used for analysis.

**Transition workflow**:
1. Deploy new rule group with OverrideAction set to Count
2. Monitor for 1-2 weeks using CloudWatch metrics and sampled request logs
3. Review for false positives
4. Remove the Count override (rules revert to their configured Block actions)

**Duration**: AWS recommends 1-2 weeks of monitoring. No auto-expiration mechanism.

**Key lesson for gdev**: AWS's log-level distinction between terminating (Block) and non-terminating (Count) matches is a clean model. gdev should adopt the same pattern: enforcement-mode blocks appear as one log entry type, monitor-mode would-blocks appear as another, and both are in the same log stream.

### 2.5 seccomp SECCOMP_RET_LOG

**Mechanism**: Added in Linux 4.14, `SECCOMP_RET_LOG` is a seccomp filter return action that allows the system call to execute while logging the filter's decision. It occupies a middle position in the action precedence hierarchy — higher than `SECCOMP_RET_ALLOW` (silent allow) but lower than enforcement actions (`SECCOMP_RET_KILL_THREAD`, `SECCOMP_RET_TRAP`, `SECCOMP_RET_ERRNO`).

**Granularity**: Per-syscall within a BPF filter. Each syscall in the filter can independently return `SECCOMP_RET_LOG` (monitor) or `SECCOMP_RET_ERRNO` (enforce).

**Logging**: Logged via the kernel audit subsystem. The `actions_logged` sysctl (`/proc/sys/kernel/seccomp/actions_logged`) controls which actions are logged, giving the administrator global control over log volume.

**Duration**: No auto-expiration. The filter remains in effect until the process exits.

**Key lesson for gdev**: The `SECCOMP_FILTER_FLAG_LOG` flag is a useful concept — it says "log everything that is not a clean allow." This maps to gdev's audit trail design: in monitor mode, log every evaluation that would have resulted in deny or ask, but do not log clean allows (to reduce noise). The `actions_logged` sysctl is an admin-level volume control — gdev could offer a similar `GDEV_MONITOR_LOG_LEVEL` to control verbosity.

### 2.6 Kubernetes ValidatingAdmissionPolicy

**Mechanism**: Each ValidatingAdmissionPolicyBinding specifies `validationActions` — one or more of Deny, Warn, and Audit. Warn returns HTTP warning headers to the client. Audit records the failure in the Kubernetes audit log. Both allow the request to proceed.

**Granularity**: Per-binding (which maps to per-scope, since bindings target specific namespaces/resources). The same policy can have different actions in different environments: Deny in production, Warn+Audit in staging.

**Notable constraint**: Deny and Warn cannot be combined (duplicative). Warn+Audit can be combined (warn user AND log for compliance).

**Key lesson for gdev**: The Kubernetes three-action model (Deny/Warn/Audit) maps perfectly to gdev's needs. Deny = enforce mode (block), Warn = monitor mode with advisory warnings, Audit = monitor mode with logging only (no user-visible feedback). gdev should support at least Warn and Audit variants of monitor mode. The per-binding-per-scope pattern (same rule, different modes in different environments) is relevant if gdev ever supports per-project monitor/enforce configuration.

### 2.7 Cross-System Comparison

| System | Granularity | Log Format | Duration | Auto-Expire? | Transition Command |
|--------|------------|------------|----------|-------------|-------------------|
| SELinux | Per-domain | AVC audit log (permissive=1 flag) | Indefinite | No (reboot reverts setenforce) | `semanage permissive -d <domain>` |
| AppArmor | Per-profile | syslog/audit.log | Indefinite | No | `aa-enforce <profile>` |
| Defender ASR | Per-rule | Event ID 1122 (audit) vs 1121 (block) | 2-4 weeks recommended | No | Change rule action 2->1 |
| AWS WAF | Per-rule / per-rule-group | nonTerminatingMatchingRules in access log | 1-2 weeks recommended | No | Remove Count override |
| seccomp | Per-syscall | Kernel audit | Until process exit | No | N/A (compile-time) |
| K8s ValidatingAdmission | Per-binding (per-scope) | Audit annotation | Indefinite | No | Change validationActions |

**Universal findings**:
1. **No system auto-expires its monitor mode.** Every system requires manual promotion to enforcement. This is consistent across kernel, network, endpoint, and cloud security.
2. **Every system supports per-entity granularity.** None of the six systems is global-only. The minimum granularity is per-profile/per-domain, with ASR and WAF achieving per-rule granularity.
3. **All systems log monitor-mode events to the same log stream as enforcement events**, distinguished by a flag or event type. None uses a separate log file for monitor vs enforce events.
4. **Recommended monitoring durations range from 1-4 weeks** where specified (ASR: 2-4 weeks, WAF: 1-2 weeks). Other systems give no specific guidance.
5. **All systems evaluate fully in monitor mode.** No system skips logic or takes shortcuts — the monitor evaluation is identical to enforcement evaluation.

---

## 3. Granularity Recommendation

### 3.1 Options Evaluated

| Approach | Description | Pros | Cons |
|----------|-------------|------|------|
| **Global** | All rules in same mode | Simple to implement, simple UX | Can't enforce validated rules while calibrating new ones |
| **Per-rule** | Each rule has its own mode | Maximum flexibility, matches ASR/WAF | More complex configuration, more commands to manage |
| **Per-severity** | Mode determined by severity tier | Reasonable default (critical=enforce, medium=monitor) | Inflexible — a critical rule that hasn't been validated yet is forced into enforce |
| **Per-category** | Mode determined by category | Intuitive grouping (self-protection=enforce, advisory=monitor) | Same inflexibility as per-severity |

### 3.2 Recommendation: Per-Rule with Category Defaults

gdev should implement **per-rule mode control** with **category-level defaults**:

1. **Each rule has a `mode` field**: `enforce`, `monitor`, or `off`.
2. **Categories define defaults**: When a new rule is added to a category, it inherits the category's default mode.
3. **Explicit per-rule overrides**: A developer can set any individual rule to any mode, overriding the category default.
4. **Global override**: `gdev hook monitor --all` sets all rules to monitor mode (for initial deployment). `gdev hook enforce --all` promotes all rules to enforce mode.

**Category default modes for initial deployment**:

| Category | Default Mode | Rationale |
|----------|-------------|-----------|
| self-protection | **monitor** | These are new rules with no production history; calibrate before enforcing |
| destructive-prevention | **enforce** | Already deployed and validated in Phase 32 |
| credential-scanning | **enforce** | Already deployed and validated in Phase 32 |
| mcp-poisoning | **monitor** | New rules, unknown false positive rate |
| configuration-guard | **monitor** | New rules, high ask-verdict frequency expected |
| advisory (cost, audit, test) | **enforce** | Already advisory/non-blocking by design |

This means on first deployment of self-protection hooks, only the new untested rules are in monitor mode. Existing Phase 32 hooks continue to enforce. As the developer reviews monitor-mode logs and confirms zero false positives, they promote individual rules to enforce.

### 3.3 Rationale: Why Not Global-Only?

A global monitor mode would force all rules into monitor simultaneously, including the already-validated destructive-prevention and credential-scanning hooks. This creates a security window where known-good protections are suspended while unknown-quantity new rules are being calibrated. Per-rule mode avoids this: enforce what is proven, monitor what is new.

SELinux's per-domain permissive mode exists for exactly this reason. Android's SELinux documentation explicitly states: "Per-domain permissive mode enables incremental application of SELinux to an ever-increasing portion of the system and policy development for new services (while keeping the rest of the system enforcing)."

### 3.4 AppArmor's "Deny Rules Enforce Even in Complain" Pattern

AppArmor enforces explicit deny rules even when a profile is in complain mode. gdev should adopt an equivalent: certain absolute rules should enforce even when their category is in monitor mode. Candidates:

- **SP-4: Deny writes to `~/.claude/settings.json`** — Allowing this in monitor mode defeats the purpose of self-protection; the agent could disable all hooks during the calibration period.
- **SP-5: Deny writes to `~/.claude/policy-limits.json`** — Same rationale.

These are the "nuclear" self-protection rules where even a single successful bypass during calibration would undermine the entire security architecture. They should have an `enforce_always: true` flag that prevents monitor mode from overriding their verdict.

Implementation: The mode override logic checks `enforce_always` before applying the monitor-mode allow:

```go
func applyMode(rule Rule, verdict Verdict) Verdict {
    if rule.EnforceAlways && verdict == Deny {
        return Deny // Never override to allow
    }
    if rule.Mode == Monitor {
        return Allow // Override to allow, but log
    }
    return verdict // Enforce mode: return original verdict
}
```

---

## 4. Transition Workflow

### 4.1 Enabling Monitor Mode

**Initial deployment** (all new self-protection rules start in monitor mode):

```bash
gdev enable hooks --self-protection
# Output:
# Deploying self-protection hooks...
#   sp-deny-gdev-cli ................... monitor (new rule)
#   sp-deny-process-kill ............... monitor (new rule)
#   sp-deny-qsdev-write ............... monitor (new rule)
#   sp-deny-settings-write ............ enforce (enforce_always)
#   sp-deny-policy-limits-write ....... enforce (enforce_always)
#   sp-ask-read-settings .............. monitor (new rule)
#   sp-deny-audit-trail-destroy ....... monitor (new rule)
#   sp-bash-guard-protected-paths ..... monitor (new rule)
#   sp-subagent-prompt-screen ......... monitor (new rule)
#
# 7 rules in monitor mode. Run `gdev hook audit` to review logged events.
# Monitor mode will prompt for review after 5 days.
```

**Per-rule mode change**:

```bash
gdev hook monitor sp-deny-gdev-cli
# Output: Rule sp-deny-gdev-cli set to monitor mode.

gdev hook enforce sp-deny-gdev-cli
# Output: Rule sp-deny-gdev-cli set to enforce mode.

gdev hook off sp-deny-gdev-cli
# Output: Rule sp-deny-gdev-cli disabled.
# Warning: Disabling self-protection rules reduces security.
```

**Category-level mode change**:

```bash
gdev hook monitor --category self-protection
# Output: 9 self-protection rules set to monitor mode.
# Warning: sp-deny-settings-write and sp-deny-policy-limits-write have enforce_always=true
#          and will remain in enforce mode.

gdev hook enforce --category self-protection
# Output: 9 self-protection rules set to enforce mode.
```

**Global mode change**:

```bash
gdev hook monitor --all
# Output: 15 rules set to monitor mode.
# Warning: 2 rules with enforce_always=true will remain in enforce mode.

gdev hook enforce --all
# Output: 15 rules set to enforce mode.
```

### 4.2 Reviewing Monitor-Mode Logs

The `gdev hook audit` command is the equivalent of AppArmor's `aa-logprof` — it reads monitor-mode log entries and presents them for review:

```bash
gdev hook audit
# Output:
# Monitor mode events (last 5 days):
#
# ┌─ sp-deny-gdev-cli ────────────────────────────────────────┐
# │ 0 would-block events                                      │
# │ Status: Clean — safe to promote to enforce                 │
# └────────────────────────────────────────────────────────────┘
#
# ┌─ sp-bash-guard-protected-paths ────────────────────────────┐
# │ 3 would-block events                                       │
# │                                                            │
# │ 1. 2026-05-15 14:23:01 — Bash: cat ~/.qsdev/hooks/destr.. │
# │    Pattern: read from ~/.qsdev/hooks/                      │
# │    Assessment: FALSE POSITIVE — reading hook for debugging  │
# │                                                            │
# │ 2. 2026-05-16 09:11:45 — Bash: ls ~/.qsdev/               │
# │    Pattern: command references ~/.qsdev/                    │
# │    Assessment: FALSE POSITIVE — listing directory           │
# │                                                            │
# │ 3. 2026-05-17 11:02:33 — Bash: gdev doctor                │
# │    Pattern: command contains "gdev"                         │
# │    Assessment: FALSE POSITIVE — legitimate gdev CLI use     │
# │                                                            │
# │ Status: 3 false positives found — refine rules before      │
# │         promoting to enforce                                │
# └────────────────────────────────────────────────────────────┘
#
# Summary: 7 rules in monitor mode
#   4 clean (0 events) — safe to promote
#   2 with events — review needed
#   1 with false positives — rule refinement needed
#
# Actions:
#   gdev hook enforce <rule-id>    Promote a clean rule
#   gdev hook enforce --clean      Promote all rules with 0 events
#   gdev hook audit --detail <id>  See full details for a rule
```

**Detailed view**:

```bash
gdev hook audit --detail sp-bash-guard-protected-paths
# Output:
# Rule: sp-bash-guard-protected-paths
# Mode: monitor (since 2026-05-15)
# Events: 3 would-block
#
# Event 1/3:
#   Time:     2026-05-15 14:23:01
#   Tool:     Bash
#   Command:  cat ~/.qsdev/hooks/destructive-prevention.sh
#   Rule:     sp-bash-guard-protected-paths
#   Pattern:  (cat|less|head|tail).*\.qsdev/hooks/
#   Verdict:  would-deny
#   Reason:   "Bash command reads from gdev hooks directory"
#   Context:  Session abc123, user colin
#
# [1/3] Is this a legitimate operation? (y=false positive, n=true positive, s=skip)
```

### 4.3 Promoting to Enforce Mode

**Promote individual rule**:

```bash
gdev hook enforce sp-deny-gdev-cli
# Output: Rule sp-deny-gdev-cli promoted to enforce mode.
#         This rule will now block matching operations.
```

**Bulk promote clean rules** (the "safe bet" operation):

```bash
gdev hook enforce --clean
# Output: Promoting 4 rules with 0 monitor-mode events:
#   sp-deny-gdev-cli .................. enforce ✓
#   sp-deny-process-kill .............. enforce ✓
#   sp-deny-qsdev-write .............. enforce ✓
#   sp-deny-audit-trail-destroy ....... enforce ✓
#
# 3 rules remain in monitor mode (have events to review).
```

### 4.4 Recommended Calibration Period

**Recommendation: 5 working days (1 week)**

Rationale:
- Microsoft ASR recommends 2-4 weeks, but ASR rules cover a much larger attack surface across an entire enterprise. gdev's self-protection rules are a small, focused set.
- AWS WAF recommends 1-2 weeks for managed rule groups. gdev's rules are simpler than WAF rule groups.
- 5 working days captures a representative sample of the developer's workflow — different tasks, different Claude Code tool patterns, different project contexts.
- Shorter than 5 days risks missing infrequent operations (weekly standup automation, end-of-sprint cleanup, etc.).
- Longer than 2 weeks delays security value without proportionate benefit for a small rule set.

**The calibration period is a recommendation, not an enforcement.** The developer can promote rules to enforce at any time, including immediately after deployment. The 5-day recommendation is communicated via:
- A message at deployment time: "Monitor mode will prompt for review after 5 days."
- A reminder after the period: "5 days of monitor data collected. Run `gdev hook audit` to review."

### 4.5 Auto-Suggest Enforcement

After the calibration period, if a rule has zero would-block events:

```
gdev: Rule sp-deny-gdev-cli has been in monitor mode for 5 days with
      0 would-block events. Promote to enforce mode?
      Run: gdev hook enforce sp-deny-gdev-cli
      Or:  gdev hook enforce --clean  (promotes all clean rules)
```

This is a suggestion, not an automatic transition. The developer must explicitly promote. The rationale:
- Automatic promotion could surprise the developer with unexpected blocks.
- Zero events could mean the rule is never triggered (no security value) rather than that it has no false positives — the developer should consciously review this.
- Automatic transitions in security tools are a known source of incidents (see: AWS WAF propagation timing issues where "new rule group rules might be in effect in one area while still allowed in another").

---

## 5. Log Format

### 5.1 Log Entry Design

Monitor-mode events are logged to the same JSONL audit trail as enforcement events (`~/.qsdev/audit/sessions/<date>/<session-id>.jsonl`). They are distinguished by the `mode` and `effective_verdict` fields.

**Monitor-mode would-deny entry**:

```json
{
  "timestamp": "2026-05-15T14:23:01.234Z",
  "event_type": "hook_evaluation",
  "session_id": "abc123def456",
  "mode": "monitor",
  "rule_id": "sp-bash-guard-protected-paths",
  "rule_category": "self-protection",
  "hook_event": "PreToolUse",
  "tool_name": "Bash",
  "evaluated_verdict": "deny",
  "effective_verdict": "allow",
  "reason": "Bash command reads from gdev hooks directory",
  "matched_pattern": "(cat|less|head|tail).*\\.qsdev/hooks/",
  "operation": {
    "tool": "Bash",
    "command": "cat ~/.qsdev/hooks/destructive-prevention.sh",
    "file_path": null,
    "resolved_path": "/home/colin/.qsdev/hooks/destructive-prevention.sh"
  },
  "evaluation_time_ms": 3,
  "user": "colin",
  "cwd": "/home/colin/Repos/client-project"
}
```

**Enforcement-mode deny entry** (for comparison):

```json
{
  "timestamp": "2026-05-15T14:23:01.234Z",
  "event_type": "hook_evaluation",
  "session_id": "abc123def456",
  "mode": "enforce",
  "rule_id": "sp-bash-guard-protected-paths",
  "rule_category": "self-protection",
  "hook_event": "PreToolUse",
  "tool_name": "Bash",
  "evaluated_verdict": "deny",
  "effective_verdict": "deny",
  "reason": "Bash command targets gdev hooks directory",
  "matched_pattern": "(sed -i|cat.*>|tee|cp|mv).*\\.qsdev/hooks/",
  "operation": {
    "tool": "Bash",
    "command": "sed -i 's/exit 2/exit 0/' ~/.qsdev/hooks/destructive-prevention.sh",
    "file_path": null,
    "resolved_path": "/home/colin/.qsdev/hooks/destructive-prevention.sh"
  },
  "evaluation_time_ms": 3,
  "user": "colin",
  "cwd": "/home/colin/Repos/client-project"
}
```

### 5.2 Key Fields

| Field | Type | Purpose |
|-------|------|---------|
| `mode` | `enforce` \| `monitor` | Which mode the rule was in when evaluated |
| `evaluated_verdict` | `deny` \| `ask` \| `allow` | What the rule would have returned |
| `effective_verdict` | `deny` \| `ask` \| `allow` | What was actually applied (same as evaluated in enforce mode; always `allow` in monitor mode) |
| `rule_id` | string | The specific rule that matched |
| `rule_category` | string | The category (self-protection, destructive-prevention, etc.) |
| `matched_pattern` | string | The regex or path pattern that triggered the match |
| `operation` | object | Full details of the tool call (tool name, command/path, resolved path) |
| `evaluation_time_ms` | number | How long the rule evaluation took (for performance monitoring) |

### 5.3 Distinguishing Monitor from Enforce in Logs

The log design follows the universal pattern from the prior art survey: **same log stream, same format, different flag**. Specifically:

- SELinux uses `permissive=1` in AVC entries
- Defender ASR uses Event ID 1122 (audit) vs 1121 (block)
- AWS WAF uses `nonTerminatingMatchingRules` vs `terminatingRule`
- gdev uses `mode: "monitor"` + `effective_verdict: "allow"` vs `mode: "enforce"` + `effective_verdict: "deny"`

The `effective_verdict` field is the primary filter. To find all would-have-blocked events:

```bash
# All events where the rule would have blocked but didn't (monitor mode)
jq 'select(.mode == "monitor" and .evaluated_verdict == "deny")' \
  ~/.qsdev/audit/sessions/2026-05-15/*.jsonl

# All actually-blocked events (enforce mode)
jq 'select(.mode == "enforce" and .effective_verdict == "deny")' \
  ~/.qsdev/audit/sessions/2026-05-15/*.jsonl

# All events for a specific rule across all modes
jq 'select(.rule_id == "sp-bash-guard-protected-paths")' \
  ~/.qsdev/audit/sessions/2026-05-15/*.jsonl
```

### 5.4 Log Location

Monitor-mode events are stored in the same location as all other hook evaluation events: `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl`. There is no separate monitor-mode log file.

Rationale: A single audit trail provides a complete picture of all hook activity. Splitting into separate files would require merging for analysis, adds configuration complexity, and diverges from the universal pattern (all six prior art systems use the same log stream for monitor and enforce events).

### 5.5 Monitor-Mode Hook Error Entry

When a hook script crashes or times out during monitor-mode evaluation:

```json
{
  "timestamp": "2026-05-15T14:23:01.234Z",
  "event_type": "hook_error",
  "session_id": "abc123def456",
  "mode": "monitor",
  "rule_id": "sp-bash-guard-protected-paths",
  "rule_category": "self-protection",
  "hook_event": "PreToolUse",
  "tool_name": "Bash",
  "evaluated_verdict": "error",
  "effective_verdict": "allow",
  "error": {
    "type": "hook_crash",
    "exit_code": 127,
    "stderr": "/usr/bin/env: 'jq': No such file or directory",
    "duration_ms": 12
  },
  "operation": {
    "tool": "Bash",
    "command": "echo hello"
  }
}
```

This is valuable because it surfaces hook reliability issues during the calibration period, before they become fail-closed blocks in enforce mode.

---

## 6. Maximum Duration Guard

### 6.1 Research Finding: No Established Tool Auto-Expires

None of the six surveyed security systems implements automatic expiration of their monitor/permissive/audit mode:

- **SELinux**: Permissive mode persists indefinitely. The `setenforce 0` command reverts on reboot (because the config file wins), but `semanage permissive -a` persists across reboots.
- **AppArmor**: Complain mode persists indefinitely.
- **Defender ASR**: Audit mode persists indefinitely. No timeout mechanism.
- **AWS WAF**: Count mode persists indefinitely. No timeout.
- **seccomp**: `SECCOMP_RET_LOG` persists until process exit.
- **Kubernetes**: Warn/Audit mode persists indefinitely.

### 6.2 Recommendation: Soft Reminder, Not Hard Expiration

gdev should implement a **soft reminder** rather than a hard expiration:

**After the calibration period (default 5 working days)**:
- gdev prints a reminder at session start: "N rules have been in monitor mode for X days. Run `gdev hook audit` to review."
- The reminder appears once per session, not on every tool call.
- The reminder is a CLI message, not a blocking prompt.

**After 2x the calibration period (default 10 working days)**:
- The reminder escalates in prominence: "Warning: N rules have been in monitor mode for X days without review. These rules provide no security protection. Run `gdev hook audit` or `gdev hook enforce --clean`."

**After 4x the calibration period (default 20 working days)**:
- `gdev doctor` flags it as a health issue: "Monitor mode stale: N self-protection rules have been in monitor mode for X days. This provides no security value."

**Never auto-promote**: gdev should never automatically transition rules from monitor to enforce. The rationale:
1. No established security tool does this — it is unprecedented.
2. Automatic enforcement could surprise the developer with unexpected blocks mid-workflow.
3. "Zero events" during monitoring could mean the rule covers a rare code path, not that it has no false positives.
4. The developer should consciously opt into enforcement as a positive action.

**Never auto-disable**: gdev should also never automatically turn off stale monitor-mode rules. A rule in monitor mode at least generates audit data; turning it off generates nothing. The escalating reminders create sufficient pressure without removing any functionality.

### 6.3 Configuration

The calibration period and reminder schedule should be configurable:

```yaml
# ~/.qsdev/config.yaml (or .gdev.yaml)
monitor_mode:
  calibration_days: 5         # Days before first review prompt
  reminder_interval_days: 5   # Days between subsequent reminders
  escalate_after_days: 20     # Days before gdev doctor flags as stale
```

---

## 7. Implementation Architecture

### 7.1 Mode Storage

Rule modes are stored in `~/.qsdev/hook-state.yaml`:

```yaml
# ~/.qsdev/hook-state.yaml
# Auto-generated by gdev. Do not edit directly.
rules:
  sp-deny-gdev-cli:
    mode: monitor
    mode_since: "2026-05-15T10:00:00Z"
    monitor_events: 0
  sp-deny-settings-write:
    mode: enforce
    enforce_always: true
  sp-bash-guard-protected-paths:
    mode: monitor
    mode_since: "2026-05-15T10:00:00Z"
    monitor_events: 3
```

This file is separate from the rule definitions (which are compiled into the Go binary or loaded from `~/.qsdev/rules/`). The state file tracks only runtime mode and statistics.

**Self-protection**: The `hook-state.yaml` file itself is under `~/.qsdev/` and is therefore protected by the `sp-deny-qsdev-write` rule. An agent cannot modify rule modes by editing this file directly.

### 7.2 Hook Script Mode Override

The Go binary (or shell hook script) reads the mode from `hook-state.yaml` and applies the override:

```go
func evaluateRule(rule Rule, input ToolInput, state RuleState) HookResult {
    // 1. Full evaluation regardless of mode
    verdict, reason, pattern := rule.Evaluate(input)
    
    // 2. Log the evaluation
    logEntry := AuditEntry{
        Mode:             state.Mode,
        RuleID:           rule.ID,
        EvaluatedVerdict: verdict,
        Reason:           reason,
        MatchedPattern:   pattern,
        Operation:        input,
    }
    
    // 3. Apply mode override
    if state.Mode == Monitor {
        if rule.EnforceAlways && verdict == Deny {
            // enforce_always rules are never overridden
            logEntry.EffectiveVerdict = Deny
        } else {
            logEntry.EffectiveVerdict = Allow
            if verdict == Deny {
                // Print advisory warning
                fmt.Fprintf(os.Stderr, "gdev monitor: would-block by %s: %s\n", rule.ID, reason)
                state.MonitorEvents++
            }
        }
    } else {
        logEntry.EffectiveVerdict = verdict
    }
    
    // 4. Write to audit trail
    writeAuditLog(logEntry)
    
    return HookResult{Verdict: logEntry.EffectiveVerdict}
}
```

### 7.3 SessionStart Reminder Check

The SessionStart hook checks for stale monitor-mode rules:

```go
func sessionStartCheck(state HookState, config MonitorConfig) {
    staleRules := []string{}
    for id, rs := range state.Rules {
        if rs.Mode == Monitor {
            daysInMonitor := time.Since(rs.ModeSince).Hours() / 24
            if daysInMonitor >= float64(config.CalibrationDays) {
                staleRules = append(staleRules, id)
            }
        }
    }
    if len(staleRules) > 0 {
        fmt.Fprintf(os.Stderr, 
            "gdev: %d rules in monitor mode for >%d days. Run `gdev hook audit`.\n",
            len(staleRules), config.CalibrationDays)
    }
}
```

---

## 8. Developer Experience Summary

### 8.1 Lifecycle

```
[deploy] ──> [monitor] ──> [review] ──> [refine] ──> [enforce]
   │              │             │            │             │
   │              │             │            │             │
   gdev enable    rules         gdev hook    adjust        gdev hook
   hooks          evaluate      audit        patterns      enforce
                  fully,                     and re-deploy
                  log only
```

### 8.2 Command Reference

| Command | Effect |
|---------|--------|
| `gdev hook status` | Show all rules with current mode, event counts, time in mode |
| `gdev hook monitor <rule-id>` | Set one rule to monitor mode |
| `gdev hook monitor --category <cat>` | Set all rules in category to monitor mode |
| `gdev hook monitor --all` | Set all rules to monitor mode (respects enforce_always) |
| `gdev hook enforce <rule-id>` | Promote one rule to enforce mode |
| `gdev hook enforce --clean` | Promote all rules with 0 monitor events |
| `gdev hook enforce --category <cat>` | Promote all rules in category |
| `gdev hook enforce --all` | Promote all rules to enforce |
| `gdev hook off <rule-id>` | Disable one rule (warning for self-protection) |
| `gdev hook audit` | Interactive review of monitor-mode events |
| `gdev hook audit --detail <rule-id>` | Detailed view of events for one rule |
| `gdev hook audit --json` | Export monitor-mode events as JSON (for scripting) |
| `gdev hook audit --since <date>` | Filter events by date |

### 8.3 Configuration File

```yaml
# .gdev.yaml (project-level)
hooks:
  self_protection:
    default_mode: monitor  # or enforce
    calibration_days: 5
    enforce_always:
      - sp-deny-settings-write
      - sp-deny-policy-limits-write
  
  # Per-rule overrides
  rules:
    sp-deny-gdev-cli:
      mode: enforce  # Override category default
    sp-ask-read-settings:
      mode: off  # Disable this rule (too noisy)
```

---

## Depth Checklist

- [x] **Underlying mechanism explained** — Full semantics of monitor mode (evaluate fully, override verdict, log with context, show advisory warning). Interaction with severity-tiered fail policy documented. The `enforce_always` exception pattern defined.
- [x] **Key tradeoffs and limitations identified** — Monitor mode provides zero security; indefinite monitor is security theater; auto-promote risks surprise blocks; `enforce_always` rules create a hybrid that may confuse developers; per-rule granularity adds UX complexity.
- [x] **Compared to alternatives** — Six established security systems surveyed (SELinux, AppArmor, Defender ASR, AWS WAF, seccomp, Kubernetes). Cross-system comparison table. Universal patterns extracted.
- [x] **Failure modes and edge cases** — Hook crashes during monitor mode (log as error, continue allowing); stale monitor mode (escalating reminders); false sense of security during calibration; `enforce_always` rules that cannot be monitored; agent exploiting monitor mode window to attack before enforcement.
- [x] **Concrete examples found** — SELinux per-domain permissive (`semanage permissive -a`), AppArmor `aa-logprof` workflow, ASR Event ID 1121/1122, AWS WAF nonTerminatingMatchingRules, seccomp `SECCOMP_RET_LOG`, Kubernetes `validationActions: [Warn, Audit]`. Palantir's 2-4 week ASR audit recommendation. Full gdev CLI examples with output.
- [x] **Report is standalone-readable** — Contains complete design specification for monitor mode: semantics, prior art, granularity, transition workflow, log format, duration guard, implementation architecture, and CLI reference. Sufficient for implementation without consulting other sources.

---

## Sources

### Internal (from prior spikes)
- `fail-policy-research.md` — Severity-tiered fail policy, monitor mode as transitional state
- `prempti-patterns-research.md` — Prempti's rule architecture, Falco-based evaluation
- `threat-model-research.md` — 12 attack vectors, defense coverage matrix
- `research-spikes/security-tooling-evaluation-gdev/reasoning-core-research.md` — Shadow mode calibration, audit trail schema

### External (saved to docs/)
- `docs/selinux-permissive-mode-gentoo-wiki.md` — SELinux per-domain permissive mode, `semanage permissive`, AVC logging
- `docs/datadog-container-security-apparmor-selinux.md` — AppArmor enforce/complain, SELinux enforcing/permissive modes
- `docs/microsoft-asr-audit-mode-deployment.md` — ASR per-rule audit mode, Event IDs 1121/1122, ring-based deployment, 2-4 week recommendation
- `docs/aws-waf-count-mode-testing.md` — WAF count mode, nonTerminatingMatchingRules, 1-2 week recommendation
- `docs/seccomp-ret-log-manpage.md` — SECCOMP_RET_LOG, SECCOMP_FILTER_FLAG_LOG, actions_logged sysctl
- `docs/kubernetes-validating-admission-policy-modes.md` — Deny/Warn/Audit validation actions, per-binding configuration
