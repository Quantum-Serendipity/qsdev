# Fail-Closed vs Fail-Open Policy for gdev Hook System

## Executive Summary

gdev should implement a **severity-tiered failure policy**: fail-closed for self-protection and credential-scanning hooks, fail-open for advisory hooks (cost alerting, test enforcement, audit logging), with a mandatory monitor mode for initial deployment. The key insight driving this recommendation is that Claude Code's hook system is **inherently fail-open** -- any hook crash, timeout, or non-exit-2 error code allows the operation to proceed. gdev must therefore implement its own fail-closed wrapper that catches all failure modes and converts them to exit code 2 for security-critical hooks, rather than relying on Claude Code's default behavior.

---

## 1. Failure Scenarios: What Can Go Wrong

### 1.1 Hook Script Crashes

**Scenario**: The hook script encounters a syntax error, runtime panic, missing dependency (e.g., `jq` not in PATH, Python import error), or unhandled exception.

**Claude Code behavior**: The crashed process returns a non-zero exit code (typically exit 1 for unhandled exceptions in Python, exit 127 for missing commands, exit 126 for permission denied). Claude Code treats ALL non-zero exit codes except exit 2 as **non-blocking errors**. The operation proceeds. A brief "hook error" notice appears in the transcript.

**Impact**: The hook provides zero security value. The developer may not notice the error notice. This is the failure mode documented in the Medium article "The Silent Failure Mode in Claude Code Hook Every Dev Should Know About" -- a developer's path-traversal validator used `sys.exit(1)` and "had probably been running exactly like this for days. It was blocking nothing."

**Frequency**: Moderate. Common causes include NixOS path differences (`jq` at a non-standard location), Python version mismatches, corrupted hook files after partial gdev upgrades.

### 1.2 Hook Script Timeouts

**Scenario**: The hook script hangs due to a deadlock, waits on a network resource, or simply takes too long (e.g., credential scan regex hits catastrophic backtracking on a large file).

**Claude Code behavior**: After the configured timeout (default 600 seconds for command hooks), the hook is killed. This produces a **non-blocking error**. The operation proceeds.

**Impact**: Same as crash -- zero security value. Additionally, 600 seconds of blocked developer time before the timeout fires, though for PreToolUse hooks this may be shorter if the developer notices the hang.

**Frequency**: Low for well-written hooks. Higher risk for hooks that shell out to external tools or process very large files.

### 1.3 Hook Script Returns Malformed JSON

**Scenario**: The hook exits 0 but its stdout contains invalid JSON (e.g., shell profile interference printing text before the JSON, or a partial JSON output due to script error).

**Claude Code behavior**: The malformed JSON is treated as plain text context, not as a decision. No deny/allow decision is made. The operation proceeds (equivalent to "allow").

**Impact**: The hook appears to run successfully but makes no decision. Particularly insidious because the developer sees no error indicator -- the hook "succeeded" but enforced nothing.

**Frequency**: Moderate. Shell profile interference (`.bashrc`/`.zshrc` printing text) is a documented issue in Claude Code's hook reference. Also occurs when scripts produce debug output before the JSON response.

### 1.4 Protected Files Become Unreadable

**Scenario**: The hook needs to read a protected-paths configuration file, a client profile YAML, or a deny-patterns list, but the file has incorrect permissions, has been deleted, or the disk has errors.

**Claude Code behavior**: Depends on how the hook handles the error. If the hook crashes (see 1.1), fail-open. If the hook catches the error and returns exit 0 without a deny decision, fail-open. Only if the hook explicitly returns exit 2 or a deny JSON does it fail-closed.

**Impact**: Configuration-dependent hooks silently degrade to no-op. A hook that checks `~/.qsdev/deny-patterns.yaml` and cannot read it has no patterns to match against -- everything passes.

**Frequency**: Low but consequential. Most likely after manual file manipulation or permission changes.

### 1.5 Go Binary Crashes or Is Killed

**Scenario**: If gdev wraps hooks in a Go binary (the planned architecture), the binary itself could crash (nil pointer dereference, OOM kill) or be killed by the user or the agent.

**Claude Code behavior**: Same as hook crash (1.1) -- the process returns a non-zero exit code, treated as non-blocking.

**Impact**: All hook logic fails simultaneously. Unlike individual script crashes, a Go binary crash disables ALL rules evaluated by that binary in a single failure.

**Frequency**: Low for well-tested Go code, but OOM kills are possible if the binary loads large configuration files or processes large tool inputs.

### 1.6 Claude Code Version Regression (v2.0.27-v2.0.31)

**Scenario**: The developer is running a Claude Code version in the documented regression range where hooks silently do not fire at all.

**Claude Code behavior**: The hook system is completely bypassed. No hooks fire. No errors are produced. No indication that hooks are not working.

**Impact**: Total security bypass. All six defense layers that depend on hooks (destructive prevention, credential scanning, audit logging, client isolation, cost alerting, test enforcement) are disabled.

**Frequency**: Documented historical issue. Could recur in future Claude Code versions. `gdev doctor` already plans to detect this (Phase 32, Unit 32.7).

### 1.7 Agent Engineers a Failure

**Scenario**: A compromised or manipulated agent intentionally causes hook failures to bypass security. Attack vectors include:
- Writing a file that causes catastrophic regex backtracking in the credential scanner (timeout bypass)
- Modifying `~/.bashrc` to print text that corrupts JSON output (malformed JSON bypass)
- Using the Bash tool to `chmod 000` the hook scripts or their configuration files (permission denied crash)
- Using the Bash tool to fill `/tmp` or the home directory, causing write failures in audit hooks

**Claude Code behavior**: All of these produce non-blocking errors under the default fail-open model. The operation proceeds.

**Impact**: This is the most dangerous failure mode because it is adversarial. A fail-open default means the agent has a direct path to disabling security: make the hooks fail, then proceed with the blocked action.

**Frequency**: The adversarial scenario is the entire reason self-protection rules exist. If an agent is prompt-injected or manipulated, this is the expected attack path.

---

## 2. Analysis of Each Approach

### 2.1 Fail-Closed (Deny All on Error)

**Definition**: When any hook encounters an error (crash, timeout, malformed output), the tool call is blocked as if the hook returned a deny decision.

**How to implement in Claude Code**: Since Claude Code is inherently fail-open, gdev must implement fail-closed at the hook script level:
```bash
#!/usr/bin/env bash
# Fail-closed wrapper pattern
set -euo pipefail
trap 'echo "Hook internal error — blocking for safety" >&2; exit 2' ERR

# ... actual hook logic ...

# If we reach here without producing a decision, default to deny
echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Hook completed without explicit allow — fail-closed default"}}' 
```

For Python hooks:
```python
import sys
try:
    # ... actual hook logic ...
    pass
except Exception as e:
    print(f"Hook internal error: {e}", file=sys.stderr)
    sys.exit(2)  # Fail-closed: block on any error
```

**Safety implications**:
- Prevents the adversarial bypass scenario (1.7). An agent cannot disable security by crashing hooks.
- Prevents the silent failure scenario (1.1, 1.2, 1.3). Broken hooks block rather than pass.
- Consistent with OWASP's "Fail Securely" principle: "a failure will follow the same execution path as disallowing the operation."
- Consistent with Prempti's design: "If Falco/plugin is unreachable, all tool calls are denied."

**Developer experience impact**:
- **Risk of total paralysis.** If a hook is broken (bad upgrade, missing dependency, corrupted config), the developer cannot use Claude Code at all until the hook is fixed. Every tool call is blocked.
- In a consulting context, this means: developer arrives at a client site, `gdev enable hooks` was run months ago, a Claude Code upgrade changed hook behavior, and now every action is blocked. The developer must diagnose and fix the hook system before doing any work.
- Prempti's documentation acknowledges this: "If the Falco service crashes, ALL Claude Code tool calls are denied until manual recovery. For a developer in the middle of work, this is disruptive."
- Requires excellent hook reliability and fast recovery mechanisms to be viable.

**Precedents**:
- AWS WAF default behavior: fail-closed. But AWS also provides a fail-open toggle because the fail-closed default caused 30 minutes of outage during an AZ failure.
- Docker seccomp: fail-closed (SCMP_ACT_ERRNO default action). Unallowed syscalls are denied.
- gVisor: fail-closed at the extreme (SCMP_ACT_KILL -- process is terminated on unallowed syscall).
- Firewalls: industry standard is fail-closed, paired with redundancy/failover.

### 2.2 Fail-Open (Allow All on Error)

**Definition**: When any hook encounters an error, the tool call proceeds as if no hook existed.

**How it works in Claude Code**: This is the **default behavior**. No additional implementation needed. Any non-exit-2 error code allows the operation to proceed.

**Security implications**:
- **Defeats the purpose of self-protection rules.** If the agent can bypass security by engineering a crash, self-protection is an illusion. The AWS WAF article documents this exact attack: "if they suspect some target applications are using fail-open then they might launch an attack when there is an AWS availability zone outage."
- Silent degradation: hooks fail without visible indication. The Medium article documents a hook that "had probably been running exactly like this for days. It was blocking nothing."
- Inconsistent with OWASP fail-securely principle.
- Consistent with reasoning-core's shadow-mode default (logs but does not block).

**Developer experience impact**:
- No paralysis risk. Broken hooks never prevent work.
- Developer may not realize hooks are broken, reducing trust in the security system.
- Good for non-critical advisory hooks where blocking would be disproportionate.

**Precedents**:
- SELinux permissive mode: logs violations without blocking. Used for debugging and initial deployment, but not recommended for production.
- AppArmor complain mode: same -- logs without enforcement.
- reasoning-core: fail-open/shadow by default.
- Claude Code itself: fail-open is the built-in default for all hook events.

### 2.3 Hybrid: Severity-Tiered Failure Policy

**Definition**: Different failure modes for different rule categories. Security-critical hooks fail-closed; advisory hooks fail-open.

**Tier mapping for gdev's Phase 32 hooks**:

| Hook | Category | Failure Mode | Rationale |
|------|----------|-------------|-----------|
| Destructive prevention | Security-critical | **Fail-closed** | Blocking `terraform destroy` is the whole point; a broken hook must not silently allow it |
| Credential scanning | Security-critical | **Fail-closed** | Writing secrets to disk is irreversible (enters git history); must block on error |
| Self-protection rules | Security-critical | **Fail-closed** | The agent disabling security is the primary threat; fail-open defeats the defense |
| Client isolation | Advisory | **Fail-open** | Warning about wrong AWS profile is valuable but blocking all work is disproportionate |
| Cost alerting | Advisory | **Fail-open** | Cost visibility is nice-to-have; blocking developer work over a cost hook crash is absurd |
| Test enforcement | Advisory | **Fail-open** | Already advisory by design (always exits 0); a crash should not change that |
| SOC 2 audit logging | Advisory | **Fail-open** | Audit logging should never block operations; log the logging failure separately |

**Safety implications**:
- Security-critical operations are protected even when hooks fail.
- Advisory operations never cause paralysis.
- The agent cannot bypass self-protection by crashing hooks.
- Consistent with the cross-tool comparison finding: "Detect-and-nudge vs hard-block" -- tiered intervention where only critical violations are hard-blocked.

**Developer experience impact**:
- Partial paralysis is possible if a security-critical hook is broken. But the blast radius is limited to the tool calls that match that hook (e.g., only Bash commands are blocked by destructive prevention, not Read or Write).
- Advisory hooks never disrupt workflow.
- The developer gets clear error messages distinguishing "blocked for safety (fix hook)" from "hook error (continuing anyway)."

**Precedents**:
- Microsoft Agent Governance Toolkit: circuit breakers for cascading failures, kill switches for critical violations, but SLO-based (not all-or-nothing).
- AWS WAF: provides per-rule-group fail-open configuration, allowing critical rules to fail-closed while less important rules fail-open.
- Enterprise firewall deployments: fail-closed on the firewall itself, fail-open on monitoring/logging/IDS layers.

### 2.4 Shadow/Monitor Mode (Log-Only)

**Definition**: All hooks log what they would have done but never block anything, regardless of configuration. Used during initial deployment, after upgrades, and for rule development.

**How to implement**: A gdev-level flag (`gdev hooks --mode=monitor` or `GDEV_HOOK_MODE=monitor`) that overrides all hooks to:
1. Run the full hook logic (pattern matching, path checking, etc.)
2. Log the decision that would have been made (allow/deny/ask) to the audit trail
3. Always return exit 0 (allow) regardless of the decision

**Value**:
- Enables calibration before enforcement. SELinux's permissive mode and AppArmor's complain mode exist precisely for this purpose.
- Reduces "hooks that break my workflow" rejection risk during initial rollout.
- Provides data on false positive rates before enforcement.
- reasoning-core uses this as its default mode specifically because thresholds need per-project tuning.

**Limitations**:
- Provides zero security during the monitor period. If the monitor period is indefinite (developer never switches to enforce), the hooks are theater.
- Must be a transitional mode, not a permanent configuration. gdev should prompt/remind the developer to switch to enforcement after a calibration period.

---

## 3. Industry Practices Survey

### 3.1 Kernel-Level Security (Strictest: Always Fail-Closed)

| System | Default Failure Mode | Override Available? |
|--------|---------------------|-------------------|
| seccomp (Docker default profile) | SCMP_ACT_ERRNO (deny) | Allowlist-only: specify allowed syscalls |
| seccomp (gVisor) | SCMP_ACT_KILL (terminate process) | No -- even stricter than deny |
| SELinux (enforcing mode) | Deny + log (AVC denial) | Permissive mode for debugging |
| AppArmor (enforce mode) | Deny + log | Complain mode for debugging |

**Pattern**: Kernel-level security is always fail-closed in production, with a debug/permissive mode for initial setup. The fail-closed default is non-negotiable because kernel-level bypasses have catastrophic consequences (container escape, privilege escalation).

### 3.2 Network Security (Fail-Closed Default, Fail-Open Available)

| System | Default | Override? | Notes |
|--------|---------|-----------|-------|
| Firewalls (iptables/nftables) | Default DROP (fail-closed) | Can set to ACCEPT | Industry standard is DROP-by-default |
| AWS WAF on ALB | Fail-closed (500 on WAF failure) | Fail-open toggle | Fail-open introduced after availability incidents |
| Cisco IPS/IDS | Fail-closed | Fail-open mode available | Hardware bypass available for inline deployments |
| Trellix Network Security | Fail-closed | Fail-open configurable | Per-sensor configuration |

**Pattern**: Network security defaults to fail-closed but universally provides a fail-open escape hatch. The escape hatch exists because network security failures cause total service outages, and the business cost of an outage often exceeds the security risk of brief fail-open windows. Redundancy (failover to backup appliance) is the preferred solution over fail-open.

### 3.3 Application-Level Security (Fail-Secure by Principle)

| System | Approach | Notes |
|--------|----------|-------|
| OWASP Fail Securely | "Failure follows the disallow path" | Security controls must return deny on exception |
| AuthZed (authorization) | Fail-closed recommended | "Authorization frameworks must default to denial" |
| OAuth/OIDC flows | Fail-closed | Token validation failure = deny access |

**Pattern**: Application-level security follows OWASP's principle: any exception in a security control must deny access, not grant it. The canonical example: initialize `isAdmin = false` before the check, so any exception leaves the user unauthorized.

### 3.4 AI Agent Security (Emerging, No Consensus)

| System | Default | Notes |
|--------|---------|-------|
| Prempti (Falco-based) | **Fail-closed** | "If Falco/plugin is unreachable, all tool calls are denied" |
| reasoning-core | **Fail-open/shadow** | Shadow mode default; logs but never blocks |
| Claude Code hooks | **Fail-open** | Only exit code 2 blocks; all other errors allow |
| Microsoft Agent Governance Toolkit | **Circuit-breaker** | Hybrid: SLO monitoring with kill switches |

**Pattern**: No industry consensus. Security-focused tools (Prempti) choose fail-closed. Calibration-focused tools (reasoning-core) choose fail-open. Claude Code itself is inherently fail-open. Microsoft's toolkit introduces circuit-breaker patterns as a middle ground.

---

## 4. Recommendation for gdev

### 4.1 Default Policy: Severity-Tiered Failure with Fail-Closed Self-Protection

gdev should implement **hybrid approach 2.3** with the following specifics:

#### Tier 1: Fail-Closed (Security-Critical)

These hooks MUST block on error. Implementation: wrap in a fail-closed harness that catches all exceptions and converts to exit 2.

- **Self-protection rules** (block agent from modifying `.claude/settings.json`, hook scripts, deny rules)
- **Destructive command prevention** (block `terraform destroy`, `rm -rf /`, force push to main)
- **Credential scanning** (block writing secrets to disk)

**Rationale**: These hooks protect against irreversible actions (secrets in git history, infrastructure deletion, security system dismantlement). The cost of a false allow exceeds the cost of temporary paralysis.

#### Tier 2: Fail-Open (Advisory)

These hooks should continue on error. Implementation: standard Claude Code behavior (any crash = non-blocking).

- **Cost alerting** (PostToolUse, never blocks anyway)
- **Test enforcement** (Stop hook, never blocks anyway)
- **SOC 2 audit logging** (metadata logging, never blocks anyway)
- **Client isolation verification** (SessionStart warning, never blocks anyway)

**Rationale**: These hooks are advisory by design -- they already always exit 0. A crash in an advisory hook should not change the hook's fundamental nature. Blocking the developer because the cost alerting script has a bug is disproportionate.

#### Tier 3: Configurable (Project-Specific Rules)

For rules defined in `.gdev.yaml` or per-project configuration, the project owner chooses:
- `failure_mode: closed` -- fail-closed (recommended for regulated environments)
- `failure_mode: open` -- fail-open (recommended for initial deployment)
- `failure_mode: monitor` -- log-only (recommended for rule development)

### 4.2 The Fail-Closed Harness

Because Claude Code is inherently fail-open, gdev must implement fail-closed behavior at the hook level. The recommended pattern:

**For shell hooks** (destructive prevention):
```bash
#!/usr/bin/env bash
set -euo pipefail

# Fail-closed: any unhandled error blocks the operation
trap 'echo "gdev hook internal error — blocking for safety. Run gdev doctor to diagnose." >&2; exit 2' ERR EXIT

# ... actual hook logic ...

# Explicit success: must reach this point to allow
exit 0
```

The `trap ... EXIT` ensures that even if the script exits normally without producing a deny/allow decision, exit 2 fires. The actual hook logic must explicitly `trap - EXIT` and `exit 0` at the end for a clean allow.

**For Python hooks** (credential scanning):
```python
import sys

def main():
    try:
        # ... actual hook logic ...
        # Must explicitly allow or deny
        sys.exit(0)  # allow
    except SystemExit:
        raise  # Don't catch explicit sys.exit()
    except Exception as e:
        print(f"gdev hook internal error: {e}", file=sys.stderr)
        print("Blocking for safety. Run gdev doctor to diagnose.", file=sys.stderr)
        sys.exit(2)  # fail-closed

if __name__ == '__main__':
    main()
```

**For the planned Go binary**: The Go binary wrapping all hooks provides the cleanest fail-closed implementation:
```go
func runHookWithPolicy(hook Hook, input []byte, policy FailurePolicy) HookResult {
    result, err := hook.Evaluate(input)
    if err != nil {
        if policy == FailClosed {
            return HookResult{
                Decision: Deny,
                Reason:   fmt.Sprintf("Hook error (fail-closed): %v. Run gdev doctor.", err),
            }
        }
        // FailOpen: log the error, return allow
        log.Warnf("Hook %s error (fail-open): %v", hook.ID, err)
        return HookResult{Decision: Allow}
    }
    return result
}
```

### 4.3 Why Not Pure Fail-Closed?

Prempti's pure fail-closed approach is wrong for gdev because:

1. **gdev is a developer tool, not a production security gateway.** A firewall that blocks all traffic for 30 minutes is an operational incident. A developer tool that blocks all tool calls for 30 minutes is a lost morning of work for a consultant billing by the hour.

2. **The failure blast radius is too large.** Prempti's fail-closed blocks ALL tool calls when Falco crashes. gdev's hooks are per-tool-type (Bash, Write, Edit), so a crash in the destructive-prevention hook blocks only Bash commands, not Write or Read. But even this partial paralysis is unacceptable for non-security hooks.

3. **Advisory hooks should never block.** Cost alerting, audit logging, and test enforcement exist to inform, not to gate. Converting an advisory hook to a blocking hook on crash changes its fundamental contract with the developer.

4. **The consulting context demands reliability.** Developers at client sites need tools that work. A tool that intermittently paralyzes their workflow will be disabled (`gdev disable hooks`) rather than debugged, which defeats all security value.

### 4.4 Why Not Pure Fail-Open?

Claude Code's default fail-open is wrong for security hooks because:

1. **It defeats self-protection.** If the agent can bypass security by crashing hooks, then the self-protection rules are security theater. An adversarial agent will discover that `chmod 000 ~/.qsdev/hooks/destructive-prevention.sh` followed by `terraform destroy` works.

2. **Silent failure is worse than noisy failure.** The Medium article demonstrates that fail-open hooks can run for days providing zero security value without anyone noticing. At least fail-closed failures are immediately visible.

3. **Irreversible actions demand fail-closed.** Writing a secret to disk (enters git history), deleting infrastructure, and disabling security controls are all irreversible or expensive to reverse. The OWASP principle applies: the failure path must follow the deny path.

4. **The AWS WAF lesson applies directly.** AWS's default fail-closed for WAF exists because they recognized that fail-open WAF creates an exploitable attack vector: trigger a WAF failure, then send malicious requests during the failure window. The same logic applies to AI agent hooks.

---

## 5. Failure Detection and Recovery Mechanism

### 5.1 Detection: How gdev Knows Hooks Are Failing

**Approach 1: Health check in `gdev doctor`**

`gdev doctor` should run a synthetic hook invocation for each security-critical hook:
```
gdev doctor
  ...
  Hook health:
    destructive-prevention.sh .......... pass (12ms)
    credential-scan.py ................. pass (45ms)
    self-protection.sh ................. FAIL: exit code 127 (jq not found)
    audit-log.py ....................... pass (23ms)
    client-isolation.sh ................ pass (8ms)
  ...
```

Each hook is invoked with a synthetic test input (a known-safe Bash command, a known-clean Write content) and must produce a valid allow response. If it crashes, times out, or returns malformed JSON, `gdev doctor` reports the failure with diagnostic information.

**Approach 2: Runtime failure counter**

The Go binary tracks consecutive failures per hook. After N consecutive failures (configurable, default 3), it:
1. Logs a warning to the audit trail
2. Prints a visible warning to the developer's terminal
3. Continues with the configured failure policy (closed or open)

This detects hooks that are persistently broken (not one-off transients).

**Approach 3: Startup validation on `gdev enable hooks`**

When hooks are first enabled, gdev runs the full health check suite. If any security-critical hook fails, `gdev enable hooks` warns but still enables (to avoid blocking initial setup). It schedules a follow-up check by adding a note to `gdev doctor` output.

### 5.2 Alerting: How the Developer Knows

**For fail-closed blocks due to hook errors**:
```
BLOCKED by gdev (hook error, fail-closed):
  Hook: destructive-prevention.sh
  Error: exit code 127 — /usr/bin/env: 'jq': No such file or directory
  
  This security hook failed and blocked the operation for safety.
  To diagnose: gdev doctor
  To temporarily bypass: GDEV_HOOK_BYPASS=destructive-prevention gdev ...
  To disable all hooks: gdev disable hooks
```

The message must:
1. Clearly state this is a hook error, not a policy deny
2. Provide the specific error (exit code, stderr first line)
3. Provide three escalation paths: diagnose, bypass single hook, disable all

**For fail-open errors on advisory hooks**:
```
  gdev hook warning: audit-log.py failed (exit 1: FileNotFoundError)
  Advisory hook — operation continues. Run gdev doctor to fix.
```

Brief, non-blocking, but visible.

### 5.3 Recovery: How to Fix Broken Hooks

**Automated recovery**:
1. `gdev doctor --fix`: Re-installs hook scripts from the embedded binary. This fixes corrupted scripts, permission changes, and missing files. Does NOT fix missing system dependencies (jq, python3).
2. `gdev enable hooks --reinstall`: Full re-deployment of all hooks. Regenerates configuration files.

**Manual recovery**:
1. `GDEV_HOOK_BYPASS=<hook-id>`: Environment variable that disables a specific hook for a single session. Logged to audit trail.
2. `gdev hooks --mode=monitor`: Switches all hooks to monitor mode (log-only). Useful when hooks are causing persistent issues and the developer needs to work while diagnosing.
3. `gdev disable hooks`: Nuclear option. Removes all gdev-managed hooks.

**Escalation path** (from least to most disruptive):
1. `gdev doctor` -- diagnose the problem
2. `gdev doctor --fix` -- reinstall scripts
3. `GDEV_HOOK_BYPASS=<hook-id>` -- bypass one broken hook for this session
4. `gdev hooks --mode=monitor` -- switch to log-only
5. `gdev disable hooks` -- remove all hooks

### 5.4 Preventing Failures

**Design hooks for resilience**:
1. **No external dependencies beyond POSIX and the language runtime.** The destructive-prevention hook uses `grep` (POSIX), not `jq`. The credential scanner uses Python stdlib `re` and `json`, not third-party packages.
2. **Bounded execution time.** No network calls. No subprocess chains. All pattern matching is in-process. Timeout configured per hook (50ms for shell, 100ms for Python).
3. **Graceful degradation for config files.** If a configuration file (deny patterns, client profile) is unreadable, security-critical hooks should deny (fail-closed), not skip the check.
4. **Self-contained binary.** The planned Go binary eliminates shell/Python dependency issues entirely. All hook logic compiled into a single binary with no runtime dependencies.

---

## 6. The Go Binary Advantage

The planned Go hook binary (`~/.qsdev/bin/gdev-hooks`) resolves most failure scenarios:

| Failure Scenario | Shell/Python Hooks | Go Binary |
|-----------------|-------------------|-----------|
| Missing interpreter | Hook fails (exit 127) | N/A -- self-contained |
| Missing dependency (jq, etc.) | Hook fails | N/A -- compiled in |
| Python version mismatch | Hook fails | N/A -- no Python |
| Shell profile interference | Corrupts JSON output | N/A -- no shell |
| Script corruption | Unpredictable behavior | Binary integrity check (checksum) |
| Unhandled exception | Exit 1 (fail-open!) | Explicit error handling in Go |
| Timeout (regex backtracking) | Possible with complex regex | Go regex (RE2) has no backtracking |

The Go binary is the strongest argument for fail-closed in security-critical hooks: when the hook is a single compiled binary with no external dependencies, the failure modes narrow dramatically. The main remaining risks are OOM kill and binary corruption, both of which are rare and detectable.

---

## 7. Comparison to Prior Art

| Tool | Default | Self-Protection | Advisory | Monitor Mode |
|------|---------|----------------|----------|-------------|
| **Prempti** | Fail-closed (all) | Fail-closed | Fail-closed | Yes (configurable) |
| **reasoning-core** | Fail-open/shadow | N/A (no self-protection) | Fail-open | Yes (default) |
| **Claude Code** | Fail-open (all) | N/A | N/A | N/A |
| **gdev (recommended)** | **Tiered** | **Fail-closed** | **Fail-open** | **Yes (transitional)** |
| **SELinux** | Enforcing (fail-closed) | N/A | N/A | Permissive mode |
| **AppArmor** | Enforce (fail-closed) | N/A | N/A | Complain mode |
| **AWS WAF** | Fail-closed | N/A | N/A | N/A (fail-open toggle) |
| **seccomp** | SCMP_ACT_ERRNO (deny) | N/A | N/A | Audit mode |

gdev's tiered approach is closest to enterprise firewall deployments (fail-closed on the firewall, fail-open on monitoring/IDS) and to SELinux/AppArmor's enforcing/permissive dual modes. No existing AI agent security tool implements severity-tiered failure policy -- this is a novel design for the space.

---

## Depth Checklist

- [x] Underlying mechanism explained -- Claude Code's exit code semantics, the fail-open default, how each failure scenario manifests at the hook level
- [x] Key tradeoffs and limitations identified -- paralysis risk vs security bypass, developer experience vs protection, the consulting context constraint
- [x] Compared to alternatives -- Prempti (fail-closed), reasoning-core (fail-open), Claude Code (fail-open), SELinux/AppArmor (enforcing/permissive), seccomp (default deny), AWS WAF (fail-closed with toggle), firewalls (fail-closed with failover)
- [x] Failure modes and edge cases described -- 7 failure scenarios with Claude Code behavior, frequency, and impact for each
- [x] Concrete examples found -- Medium article on silent hook failure, AWS WAF DDoS bypass attack, OWASP isAdmin=true anti-pattern, go binary fail-closed harness code
- [x] Report is standalone-readable -- sufficient for implementation decisions without consulting original sources

---

## Sources

| File | Content |
|------|---------|
| `docs/claude-code-hooks-reference.md` | Claude Code hooks exit code semantics, timeout defaults, error handling behavior |
| `docs/medium-silent-hook-failure-mode.md` | Real-world case study of silent fail-open in Claude Code hooks |
| `docs/aws-waf-fail-open-fail-closed.md` | AWS WAF fail-closed default, fail-open toggle, DDoS bypass attack vector |
| `docs/owasp-fail-securely.md` | OWASP fail-securely principle, isAdmin anti-pattern |
| `docs/authzed-fail-open-fail-closed.md` | AuthZed analysis of fail-open vs fail-closed in authorization systems |
| `docs/microsoft-agent-governance-toolkit.md` | Microsoft Agent Governance Toolkit: circuit breakers, kill switches, SLO enforcement |
| `docs/gvisor-security-model.md` | gVisor security model: SCMP_ACT_KILL, no-passthrough design |
| `docs/datadog-container-security-apparmor-selinux.md` | Container security: AppArmor enforce/complain, SELinux enforcing/permissive modes |
| `security-tooling-evaluation-gdev/prempti-research.md` | Prempti fail-closed design: all tool calls denied when Falco unreachable |
| `security-tooling-evaluation-gdev/reasoning-core-research.md` | reasoning-core fail-open/shadow default mode |
| `security-tooling-evaluation-gdev/cross-tool-comparison-research.md` | Cross-tool comparison of failure modes |
| `implementation-plans/gdev-secure-devenv-bootstrap/phases/32-managed-hook-policy-consulting-enforcement.md` | gdev Phase 32 hook architecture, exit code handling, known issues |
