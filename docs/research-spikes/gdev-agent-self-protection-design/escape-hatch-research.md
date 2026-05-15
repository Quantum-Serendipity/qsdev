# Escape Hatch and Bypass Mechanisms for gdev Self-Protection Hooks

## Research Question

How should gdev design legitimate override mechanisms for its self-protection hooks that are accessible to human developers but not exploitable by AI agents, with mandatory audit logging for all bypass events?

## Executive Summary

This report designs a complete bypass system for gdev's hook architecture, addressing the fundamental tension: developers need escape hatches for legitimate workflow needs, but every bypass mechanism accessible to a human via the terminal is also accessible to an AI agent operating through that same terminal. The key insight is that **the agent's constraint is not capability but visibility** — Claude Code's permission system already creates a human-in-the-loop checkpoint for every Bash command. A bypass mechanism that requires a Bash command is inherently protected by Claude Code's permission prompt, creating a two-step approval chain.

The recommended architecture uses a **tiered bypass policy** with three levels: (1) absolute-deny rules that are never bypassable, (2) interactive-bypass rules that require a `gdev hook bypass-next` CLI command (itself protected by Claude Code's permission prompt), and (3) standard-bypass rules that accept magic comments. Self-protection rules for settings.json, hook scripts, and audit trail integrity belong in tier 1 (absolute deny). Configuration changes to devenv.nix, .pre-commit-config.yaml, and .mcp.json belong in tier 2 (interactive bypass). Standard consulting rules (destructive prevention, credential scanning) belong in tier 3 (magic comments).

**Critical finding**: Claude Code's `permissionDecision: "ask"` JSON response has two known bugs — it silently overrides `permissions.deny` rules (#39344) and fails to suppress native prompts (#52822). gdev must use exit code 2 for all hard-deny verdicts and cannot rely on JSON-based ask verdicts for security enforcement until these bugs are resolved. This means the ask verdict is currently unsuitable as a self-protection bypass mechanism.

---

## 1. Bypass Mechanism Enumeration

### 1.1 Magic Comments in Code

**Mechanism**: The developer includes a comment like `# gdev-allow-destructive` in a command or file content. The hook script checks for this string and skips enforcement.

**Current gdev usage**: Phase 32 already defines `# gdev-allow-destructive` for the destructive prevention hook and `# gdev-allow-credential` for the credential scanning hook.

**Human accessibility**: High. Developer adds a comment to their command — minimal friction, zero workflow interruption.

**Agent exploitability**: **Trivially exploitable**. The agent can embed `# gdev-allow-destructive` in any Bash command it generates. This requires no special privileges, no separate tool call, and no human interaction. The agent simply appends the comment to its command string.

**Analysis**: Magic comments are the `git --no-verify` of gdev's hook system. They provide convenience at the cost of security. For non-self-protection rules (destructive prevention, credential scanning), this tradeoff is acceptable because:
1. The developer can see the comment in Claude Code's tool use display and reject it via the permission prompt
2. The rules protect against *mistakes*, not deliberate attacks — if the agent deliberately bypasses destructive prevention, it is already compromised
3. The audit trail records the bypass (the command with the comment is logged)

For self-protection rules, magic comments are **unacceptable** because the agent's goal in bypassing self-protection is to remove the very mechanisms that would catch subsequent malicious actions. The comment `# gdev-allow-self-modify` in a command that disables hooks would be visible in the permission prompt, but approval fatigue means the developer might not notice it among dozens of routine approvals.

**Recommendation**: Allow for tier 3 (standard consulting rules). Prohibit for tiers 1 and 2 (self-protection and configuration rules).

### 1.2 CLI Command (`gdev hook bypass-next`)

**Mechanism**: The developer runs `gdev hook bypass-next <rule-id>` before the operation. This generates a one-time token (stored in a temp file with a short TTL) that the next hook invocation checks and consumes.

**Human accessibility**: Moderate. Requires a separate command before the protected operation. Two-step process adds friction.

**Agent exploitability**: **Exploitable, but detectable**. The agent can run `gdev hook bypass-next` via the Bash tool. However:
1. Claude Code's permission system will show the developer a prompt: "Run: gdev hook bypass-next sp-deny-settings-write" — this is conspicuous
2. The bypass command itself can be a protected operation (self-referential protection)
3. The audit trail records both the bypass request and the subsequent operation

**Chained protection opportunity**: If `gdev hook bypass-next` is itself recognized by the self-protection Bash guard, the agent cannot run it without first bypassing the Bash guard — creating an infinite regression that can only be broken by human intervention. This is the key insight for tier 2 bypass design.

**Token design**:
```
File: ~/.qsdev/bypass-tokens/<rule-id>-<random>.token
Content: {"rule_id": "sp-ask-devenv-nix", "created": "2026-05-15T14:30:00Z", 
          "expires": "2026-05-15T14:35:00Z", "creator_pid": 12345,
          "reason": "Updating devenv.nix to add new tool"}
TTL: 5 minutes (configurable, default 300 seconds)
Single-use: consumed on first check, then deleted
```

**Recommendation**: Use for tier 2 (interactive bypass) rules. The Bash guard should recognize `gdev hook bypass-next` as a self-modification command and block it via exit code 2, creating the chained-protection pattern. The developer must approve the bypass command via Claude Code's permission prompt (the human-in-the-loop checkpoint).

### 1.3 Environment Variable (`GDEV_HOOK_BYPASS=1`)

**Mechanism**: The developer sets `GDEV_HOOK_BYPASS=1` in their shell environment before starting Claude Code. All hooks check this variable and skip enforcement.

**Human accessibility**: Easy for developers who know it exists. Must be set before Claude Code starts (or in a separate terminal).

**Agent exploitability**: **Exploitable via `export`**. The agent can run `export GDEV_HOOK_BYPASS=1` via the Bash tool. However, environment variable changes within a Bash tool call do not persist across tool calls in Claude Code — each Bash invocation gets a fresh shell. The agent would need to prepend `GDEV_HOOK_BYPASS=1` to every subsequent command, which is detectable.

**Mitigation**: The Bash guard should block `export GDEV_HOOK_BYPASS` as a self-protection rule. The variable should only be checked from the process environment inherited at Claude Code startup, not from within-session `export` commands.

**Design flaw**: A blanket bypass variable that disables ALL hooks is too coarse. It provides no granularity — the developer cannot bypass one specific rule while keeping others enforced.

**Recommendation**: **Remove `GDEV_HOOK_BYPASS=1` as a mechanism**. Replace with specific per-rule environment variables (e.g., `GDEV_BYPASS_DESTRUCTIVE_PREVENTION=1`) that must be set before Claude Code starts. The Bash guard blocks all `export GDEV_BYPASS_*` commands. For self-protection rules (tiers 1-2), no environment variable bypass should exist.

### 1.4 Interactive Prompt (gdev asks the developer directly)

**Mechanism**: When a hook fires, instead of blocking, it outputs a `permissionDecision: "ask"` verdict. Claude Code shows the developer a permission dialog explaining what the agent is attempting and why it is flagged.

**Human accessibility**: Seamless — the developer sees a dialog in their normal workflow and clicks Allow or Deny.

**Agent exploitability**: **Not directly exploitable** — the agent cannot click the Allow button. However:
1. **Approval fatigue**: After dozens of prompts, developers click Allow reflexively
2. **Bug #39344**: `permissionDecision: "ask"` silently overrides `permissions.deny` rules, so an ask verdict from a hook can actually *weaken* security by overriding a deny rule
3. **Bug #52822**: `permissionDecision: "allow"` and possibly `"ask"` are broken in v2.1.119 (regression from v2.1.59)
4. **Permission mode bypass**: In `dontAsk` or `bypassPermissions` mode, ask verdicts are automatically converted to allow — no human sees the prompt

**Critical design constraint**: The ask verdict is the *conceptually correct* mechanism for tier 2 rules, but Claude Code's current implementation makes it unreliable:
- It may silently override deny rules (#39344)
- It may not work at all in some versions (#52822)
- It is automatically approved in `dontAsk`/`bypassPermissions` modes

**Recommendation**: Do not use `permissionDecision: "ask"` for security-critical rules until bugs #39344 and #52822 are resolved. For tier 2 rules, use the CLI bypass mechanism (1.2) instead, which leverages Claude Code's *native* permission prompt (the one that fires for every Bash command) rather than the hook-injected ask verdict. The native permission prompt is battle-tested and not affected by these bugs.

### 1.5 Time-Limited Token (`gdev hook grant --duration 5m`)

**Mechanism**: The developer runs `gdev hook grant --rule sp-ask-devenv-nix --duration 5m`, which creates a time-limited bypass token. For the next 5 minutes, the specified rule allows operations without blocking.

**Human accessibility**: Good for batch operations. Developer grants a window, does multiple protected operations, window closes automatically.

**Agent exploitability**: Same as CLI command (1.2) — the agent can run it, but it appears in Claude Code's permission prompt. Additionally, time-limited tokens are more dangerous than single-use tokens because they allow unlimited operations within the window.

**Design considerations**:
- TTL should be short (1-10 minutes) to limit exposure window
- Token should be rule-specific (not blanket bypass)
- Token creation must be audited
- Token should include the creator's PID for forensic correlation

**Recommendation**: Implement as a variant of the CLI bypass mechanism. `gdev hook bypass-next` creates a single-use token; `gdev hook grant --duration 5m` creates a time-limited token. Both are protected by the same chained-protection mechanism. Time-limited grants are useful for batch devenv.nix modifications or .pre-commit-config.yaml updates where multiple Write/Edit operations are needed in quick succession.

### 1.6 Out-of-Band Channel (separate terminal, GUI notification)

**Mechanism**: gdev displays a bypass code in a separate terminal, GUI notification, or system tray widget. The developer enters this code in the Claude Code session to authorize the bypass.

**Human accessibility**: Low — requires the developer to notice and interact with a separate UI element. Interrupts flow significantly.

**Agent exploitability**: **Impossible** — the agent has no access to the separate terminal or GUI notification. This is the only mechanism that provides true cryptographic separation between human and agent.

**Design considerations**:
- Requires a persistent gdev process (daemon or tray widget) to display the code
- Adds significant infrastructure complexity
- Poor developer experience for routine bypasses
- Appropriate for extremely high-risk operations only

**Recommendation**: Not recommended for initial implementation. The chained-protection mechanism (CLI bypass via Claude Code's permission prompt) provides sufficient human-in-the-loop verification without the infrastructure overhead. Consider for future hardening if the threat model escalates.

---

## 2. Self-Protection Bypass Paradox

The core design problem: if bypass mechanisms exist for self-protection rules, the agent could use those same mechanisms to disable self-protection, creating a circular vulnerability. The solution is a **tiered bypass policy** where the most critical rules have no bypass at all.

### 2.1 Tiered Bypass Policy

#### Tier 1: Absolute Deny (NEVER bypassable)

These rules protect the integrity of the security infrastructure itself. No bypass mechanism exists — not magic comments, not CLI commands, not environment variables. The only way to override them is to modify the hook scripts directly (which is itself protected by these rules) or to uninstall gdev entirely.

**Rules in this tier**:
- **SP-DENY-SETTINGS-WRITE**: Block Write/Edit to `~/.claude/settings.json`, `~/.claude/settings.local.json`, `~/.claude/policy-limits.json`
- **SP-DENY-SETTINGS-BASH**: Block Bash commands writing to the above paths
- **SP-DENY-HOOKS-WRITE**: Block Write/Edit to `~/.qsdev/hooks/*`
- **SP-DENY-HOOKS-BASH**: Block Bash commands modifying `~/.qsdev/hooks/*`
- **SP-DENY-AUDIT-WRITE**: Block Write/Edit/Bash targeting `~/.qsdev/audit/*`
- **SP-DENY-GDEV-CLI**: Block agent from invoking `gdev enable hooks`, `gdev disable hooks`, or any gdev command that modifies security configuration
- **SP-DENY-BYPASS-EXPORT**: Block `export GDEV_BYPASS_*` and `export GDEV_HOOK_BYPASS`
- **SP-DENY-BYPASS-CMD**: Block agent from running `gdev hook bypass-next` (this creates the chained protection — the bypass command is itself blocked)

**Rationale**: These rules form the "root of trust" for gdev's security system. If any of them can be bypassed, the entire system can be dismantled. No legitimate developer workflow requires an AI agent to modify Claude Code settings or gdev hook scripts. When a developer needs to modify these files, they do it directly in their editor, not through Claude Code.

**How the developer overrides if needed**: The developer opens `~/.claude/settings.json` or `~/.qsdev/hooks/` in their text editor (not Claude Code) and makes changes directly. This is outside the agent's reach entirely. Alternatively, the developer runs `gdev enable hooks` or `gdev disable hooks` directly in their terminal (not through Claude Code's Bash tool). Since these commands modify their own hooks, they bypass the hook system by definition — the hooks only fire for Claude Code tool calls, not for direct terminal commands.

**Escape hatch of last resort**: `GDEV_SELF_PROTECTION=off` as a pre-session environment variable (set before launching Claude Code, not via `export` during a session). This disables all self-protection hooks for that session. The Bash guard blocks any attempt to `export` this variable mid-session. This is the nuclear option for developers who need to run gdev maintenance commands through Claude Code.

#### Tier 2: Interactive Bypass (CLI command with chained protection)

These rules protect configuration files that the developer legitimately needs to modify through Claude Code (because the AI agent is helping them write devenv.nix, update pre-commit config, etc.), but modifications should require explicit human approval.

**Rules in this tier**:
- **SP-ASK-DEVENV-NIX**: Require approval for edits to devenv.nix that weaken security settings
- **SP-ASK-PRECOMMIT-CONFIG**: Require approval for edits to .pre-commit-config.yaml
- **SP-ASK-MCP-CONFIG**: Require approval for edits to .mcp.json
- **SP-ASK-GDEV-YAML**: Require approval for edits to .gdev.yaml that change compliance level
- **SP-ASK-CLAUDE-MD**: Require approval for edits to CLAUDE.md that remove gdev-managed sections
- **SP-ASK-CLAUDE-COMMANDS**: Require approval for writes to .claude/commands/ or .claude/rules/

**Bypass mechanism**: `gdev hook bypass-next <rule-id> --reason "description"`. Creates a single-use token consumed by the next hook invocation for that rule. The token expires after 5 minutes if unused.

**Chained protection**: The `gdev hook bypass-next` command is itself blocked by tier 1 rule SP-DENY-BYPASS-CMD. Wait — this creates a deadlock: the bypass command is blocked, so tier 2 rules can never be bypassed.

**Resolution of the deadlock**: SP-DENY-BYPASS-CMD blocks the agent from running `gdev hook bypass-next` through the Bash tool. But the developer can run it directly in their terminal (outside Claude Code). The workflow is:
1. Agent attempts to edit devenv.nix → hook fires → returns exit code 2 with message: "Self-protection rule SP-ASK-DEVENV-NIX: Editing devenv.nix security settings requires approval. Run `gdev hook bypass-next sp-ask-devenv-nix` in your terminal to authorize."
2. Developer reads the block message in Claude Code output
3. Developer switches to their terminal and runs `gdev hook bypass-next sp-ask-devenv-nix --reason "adding new tool to devenv"`
4. Developer returns to Claude Code and tells the agent to retry
5. On retry, the hook finds the valid bypass token, consumes it, allows the operation, and logs the bypass

This workflow is the out-of-band channel (mechanism 1.6) implemented without a daemon — the "separate terminal" is the developer's regular terminal. The agent cannot access it because Claude Code's Bash tool only executes commands through Claude Code's tool pipeline, not in the developer's terminal.

**Alternative for less security-sensitive tier 2 rules**: For rules like SP-ASK-PRECOMMIT-CONFIG that are more advisory than protective, gdev could use Claude Code's native permission prompt instead of the separate-terminal workflow. The hook would output exit code 2 with a clear message, the agent would report the block to the developer, and the developer could tell the agent to proceed (which the agent can do by including `# gdev-allow-precommit` in its next attempt). This is effectively demoting the rule to tier 3 for convenience, which may be acceptable for lower-risk configuration files.

#### Tier 3: Standard Bypass (magic comments)

These rules protect against mistakes and bad practices, not against deliberate attacks. The bypass mechanism is a magic comment in the command or file content.

**Rules in this tier**:
- **DESTRUCTIVE-PREVENTION**: Bypass via `# gdev-allow-destructive`
- **CREDENTIAL-SCAN**: Bypass via `# gdev-allow-credential`
- **COST-ALERT**: No bypass needed (advisory only, never blocks)
- **CLIENT-ISOLATION**: Bypass via `GDEV_SKIP_ISOLATION_CHECK=1` (pre-session env var)
- **TEST-ENFORCEMENT**: No bypass needed (advisory only, never blocks)

**Rationale**: These rules protect against the agent making mistakes (deleting production infrastructure, committing secrets). If the agent deliberately adds a bypass comment, the developer can see it in Claude Code's tool use display. The audit trail records the bypass. The risk/friction tradeoff favors low-friction bypasses for these rules.

### 2.2 Tier Assignment Rationale

| Rule | Tier | Rationale |
|------|------|-----------|
| Settings.json write protection | 1 (absolute) | Root of trust — all other hooks depend on this |
| Hook script write protection | 1 (absolute) | Root of trust — hooks are the enforcement mechanism |
| Audit trail protection | 1 (absolute) | Evidence integrity — cannot be sacrificed |
| gdev CLI invocation | 1 (absolute) | Administrative control — only humans should manage gdev |
| Bypass mechanism protection | 1 (absolute) | Meta-protection — bypass of bypass is the end of security |
| devenv.nix security settings | 2 (interactive) | Legitimate edit target, but security-critical sections need approval |
| .pre-commit-config.yaml | 2 (interactive) | Legitimate edit target, security implications |
| .mcp.json | 2 (interactive) | MCP poisoning is a documented attack vector |
| .gdev.yaml compliance level | 2 (interactive) | Changing compliance level is a policy decision |
| CLAUDE.md gdev sections | 2 (interactive) | Instruction injection via CLAUDE.md is documented |
| .claude/commands/ and .claude/rules/ | 2 (interactive) | Skill injection is a documented attack vector |
| Destructive command prevention | 3 (standard) | Protects against mistakes, not attacks |
| Credential scanning | 3 (standard) | Protects against mistakes, not attacks |
| Cost alerting | N/A | Advisory, never blocks |
| Client isolation | N/A | Advisory, never blocks |
| Test enforcement | N/A | Advisory, never blocks |

---

## 3. Mandatory Audit Logging

### 3.1 Bypass Audit Record Schema

Every bypass event must produce an audit record, regardless of bypass mechanism used. The record captures the full context needed for forensic analysis and compliance review.

```jsonl
{
  "timestamp": "2026-05-15T14:32:17Z",
  "event_type": "hook_bypass",
  "session_id": "abc123def456",
  "rule_id": "sp-ask-devenv-nix",
  "rule_tier": 2,
  "bypass_method": "cli_token",
  "bypass_token_id": "tok_7f8a9b2c",
  "bypass_reason": "Adding new tool to devenv",
  "tool_name": "Edit",
  "tool_input_summary": {
    "file_path": "/home/dev/project/devenv.nix",
    "operation": "edit"
  },
  "command_summary": null,
  "actor": "human_approved",
  "creator_pid": 12345,
  "token_age_seconds": 23,
  "token_ttl_seconds": 300,
  "outcome": "allowed_via_bypass"
}
```

### 3.2 Field Definitions

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string (RFC3339 UTC) | When the bypass was consumed |
| `event_type` | string | Always `"hook_bypass"` for bypass events |
| `session_id` | string | Claude Code session identifier |
| `rule_id` | string | The self-protection rule that was bypassed |
| `rule_tier` | int | 1, 2, or 3 — the tier of the bypassed rule |
| `bypass_method` | string | One of: `magic_comment`, `cli_token`, `time_grant`, `env_var`, `pre_session_env` |
| `bypass_token_id` | string | For CLI/time-grant: the token identifier. For magic comments: null |
| `bypass_reason` | string | Human-provided reason (required for CLI bypass, optional for magic comments) |
| `tool_name` | string | The Claude Code tool that triggered the hook (Bash, Write, Edit, Read, Task) |
| `tool_input_summary` | object | Sanitized summary of tool input (file paths, first 80 chars of command) |
| `command_summary` | string | For Bash: first 80 chars of command, redacted for secrets. For other tools: null |
| `actor` | string | One of: `human_approved` (CLI bypass), `comment_bypass` (magic comment), `env_bypass` (env var) |
| `creator_pid` | int | PID of the process that created the bypass token (for CLI/time-grant) |
| `token_age_seconds` | int | How old the token was when consumed (for CLI/time-grant) |
| `token_ttl_seconds` | int | The token's configured TTL (for CLI/time-grant) |
| `outcome` | string | `allowed_via_bypass` or `bypass_expired` or `bypass_invalid` |

### 3.3 Log Destination

Bypass events are logged to **two locations**:

1. **Primary**: The existing SOC 2 audit trail at `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl` — same file as all other tool call audit records. Bypass events are interleaved chronologically with the tool calls they authorize, providing a complete timeline.

2. **Secondary**: A dedicated bypass log at `~/.qsdev/audit/bypasses.jsonl` — an append-only file containing ONLY bypass events across all sessions. This makes it easy to review all bypasses without scanning session logs. Useful for periodic compliance reviews ("show me every bypass in the last 30 days").

The secondary log is a convenience view, not the source of truth. The primary session log is authoritative.

### 3.4 Alerting

Bypass events should trigger **tiered alerting** proportional to the risk:

| Bypass Tier | Alert Level | Mechanism |
|-------------|------------|-----------|
| Tier 1 bypass attempt (should be impossible) | CRITICAL | Write to stderr immediately + bypass log + `gdev doctor` flags anomaly |
| Tier 2 bypass consumed | INFO | Write to bypass log only; developer initiated, expected |
| Tier 3 bypass (magic comment) | DEBUG | Write to session audit log only; routine, expected |
| Any bypass with empty reason | WARN | Write to bypass log with "no reason provided" flag |

**Future enhancement**: If gdev gains notification integration (Slack, email), tier 1 bypass attempts should trigger an immediate alert to the team lead. A tier 1 bypass attempt means either (a) a bug in gdev's protection, (b) a determined attacker, or (c) an environment misconfiguration. All three warrant human investigation.

### 3.5 Log Integrity

The bypass log inherits protection from the audit trail self-protection rules (tier 1):
- SP-DENY-AUDIT-WRITE blocks Write/Edit/Bash targeting `~/.qsdev/audit/*`
- The agent cannot delete, truncate, or modify bypass logs
- The developer can review bypass logs via `gdev audit bypasses` command

For additional integrity, the bypass log should include a running hash chain:
```jsonl
{"seq": 1, "prev_hash": "0000000000", "hash": "a1b2c3d4e5", ...event fields...}
{"seq": 2, "prev_hash": "a1b2c3d4e5", "hash": "f6g7h8i9j0", ...event fields...}
```

Each record's `hash` is computed over the record content + `prev_hash`. This creates a tamper-evident chain — deleting or modifying any record breaks the chain. `gdev doctor` can verify chain integrity.

---

## 4. Prior Art: Security Override Design Patterns

### 4.1 SELinux: Permissive Mode and audit2allow

**Design**: SELinux operates in three modes: Enforcing (denies and logs violations), Permissive (logs but does not deny), and Disabled. The `setenforce 0` command switches to permissive mode temporarily (until reboot). `audit2allow` converts logged denials into policy rules.

**Key patterns for gdev**:
- **Per-domain permissive mode**: SELinux can set individual domains to permissive while the system remains enforcing. This is the analog of gdev's per-rule bypass — granular override without disabling the entire system.
- **Temporary permissive mode**: Best practice is to stay in permissive mode only during active troubleshooting, then return to enforcing. The analog is gdev's time-limited bypass tokens with short TTLs.
- **audit2allow as policy development**: Denials in permissive mode inform policy refinement. The analog is gdev's monitor mode — log what would be blocked to calibrate rules before enforcement.
- **deny rules survive permissive mode**: In AppArmor (a comparable system), explicit deny rules are enforced even in complain (permissive) mode. gdev's tier 1 rules should follow this pattern — they fire even if gdev is in monitor mode.

**What gdev should NOT copy**: `setenforce 0` as a blanket system-wide override. gdev's `GDEV_HOOK_BYPASS=1` is the equivalent and should be removed in favor of per-rule overrides.

### 4.2 sudo: Time-Limited Elevation with Audit

**Design**: sudo grants per-command privilege elevation. After authentication, sudo caches credentials for `timestamp_timeout` minutes (default 15). Every sudo invocation is logged to a system log with the user, command, timestamp, and outcome.

**Key patterns for gdev**:
- **Per-command authorization**: Each privileged operation requires explicit approval, not a blanket "elevate and stay elevated." gdev's `bypass-next` (single-use token) follows this pattern.
- **Credential caching with timeout**: sudo's `timestamp_timeout` is the analog of gdev's time-limited grant tokens. The timeout limits the exposure window.
- **Mandatory logging**: sudo logs every invocation regardless of success/failure. The IO log feature provides a full recording of what happened during elevated sessions. gdev's bypass audit log serves the same purpose.
- **NOPASSWD as anti-pattern**: sudo's `NOPASSWD` option disables the authentication requirement — convenient but eliminates the human verification checkpoint. gdev should not provide a NOPASSWD equivalent for tier 1-2 rules.
- **Logs stored separately from the system they monitor**: sudo IO logs should be stored on a different system to prevent the elevated user from tampering. gdev cannot achieve this for local files, but the hash chain provides tamper detection.

### 4.3 AppArmor: Complain Mode and Deny Rule Persistence

**Design**: AppArmor profiles operate in Enforce mode (violations are blocked and logged), Complain mode (violations are logged but not blocked), and Audit mode (allowed operations are also logged). Critically, **deny rules in profiles are enforced even in complain mode**.

**Key patterns for gdev**:
- **Deny rules are non-negotiable**: AppArmor's design decision that explicit deny rules fire even in complain mode is the strongest analog for gdev's tier 1 rules. Even when gdev is in monitor mode, tier 1 absolute-deny rules should still block.
- **Mode switching is a privileged operation**: Switching from enforce to complain requires root access. gdev's tier 1 rules protect the mechanism that would switch enforcement modes.
- **Complain mode for policy development**: AppArmor's complain mode lets administrators see what would be blocked before enforcing. gdev's monitor mode serves the same purpose.

### 4.4 Git --no-verify: The Anti-Pattern

**Design**: `git commit --no-verify` bypasses all pre-commit and pre-push hooks. It is a blanket, unaudited, all-or-nothing bypass.

**Why gdev explicitly avoids this pattern**:
1. **No granularity**: --no-verify bypasses ALL hooks, not specific ones. gdev's per-rule bypass is the deliberate alternative.
2. **No audit trail**: --no-verify leaves no record that hooks were bypassed. gdev's mandatory bypass logging ensures every override is recorded.
3. **No friction differentiation**: --no-verify is equally easy for a typo fix and for a production-breaking change. gdev's tiered bypass ensures friction is proportional to risk.
4. **Habitual use**: Developers who learn --no-verify often use it routinely, defeating the purpose of hooks entirely. gdev's CLI bypass mechanism adds enough friction to prevent habituation.
5. **Agent-exploitable**: An agent can trivially add --no-verify to git commands. gdev's Bash guard blocks `git commit --no-verify` and `git push --no-verify` as part of the destructive prevention rules.

### 4.5 Firewall Temporary Exceptions

**Design**: Enterprise firewalls support temporary rule exceptions with TTL. A security engineer can create an exception that auto-expires after a defined period (hours to days). The exception is logged, audited, and reviewed.

**Key patterns for gdev**:
- **Auto-expiration**: Temporary exceptions auto-expire, preventing forgotten overrides from becoming permanent security holes. gdev's bypass tokens have TTLs (default 5 minutes for single-use, configurable for time-limited grants).
- **Override hierarchy**: Firewalls have explicit priority rules — upper levels override lower levels. gdev's tier system is the analog: tier 1 overrides everything (no bypass), tier 2 overrides tier 3 (requires more friction).
- **"Temporary exceptions become permanent"**: This is the known failure mode of firewall overrides. gdev mitigates this by making tokens single-use or time-limited, with no mechanism for permanent bypass of tier 1-2 rules.

### 4.6 Claude Code's dangerouslyDisableSandbox (Anti-Pattern)

**Design**: Claude Code's sandbox escape hatch is enabled by default, uses the normal permission prompt (no differentiated UI), and produces no audit trail. Issue #20259 proposed making it opt-in with distinct UI and audit logging but was closed as "not planned."

**Lessons for gdev**:
1. **Secure by default**: gdev's self-protection is on by default. Bypass requires affirmative action.
2. **Differentiated UI**: gdev's bypass messages clearly explain what is being bypassed and why it is flagged, not just a generic "Allow?" prompt.
3. **Mandatory audit**: gdev logs every bypass. Claude Code's sandbox escape logs nothing.
4. **Not the agent's choice**: gdev's tier 1-2 bypasses require human action outside the agent's reach. Claude Code's sandbox escape can be triggered by the agent and auto-approved in certain permission modes.

---

## 5. Agent-Resistant Bypass Design

### 5.1 Claude Code's Permission System as a Defense Layer

The fundamental insight: **Claude Code already implements human-in-the-loop verification for every Bash command**. In default permission mode, the developer sees a prompt for every command the agent wants to run. This means:

1. Any bypass mechanism that requires a Bash command is implicitly protected by Claude Code's permission prompt
2. The developer sees `gdev hook bypass-next sp-deny-settings-write` in the prompt and must approve it
3. In `auto` mode, Claude Code's server-side classifier screens commands — if `gdev hook bypass-next` is classified as sensitive, it still prompts

**Limitation**: In `bypassPermissions` mode, no prompts are shown, and the agent can run any command. Developers using `bypassPermissions` have explicitly opted out of human-in-the-loop verification. gdev cannot protect against this — it is equivalent to running with root access and no sudo password.

**Recommendation**: Document that `bypassPermissions` mode is incompatible with gdev's self-protection guarantees. `gdev doctor` should warn if `bypassPermissions` is detected in the permission configuration.

### 5.2 Chained Protection: The Two-Step Approval Pattern

The key innovation for agent-resistant bypass:

```
Step 1: Agent attempts protected operation
        → Hook fires → Blocks with exit code 2
        → Message: "Run `gdev hook bypass-next <rule>` in your terminal to authorize"

Step 2: Developer runs bypass command in their terminal (outside Claude Code)
        → Bypass token created
        → Developer tells agent to retry

Step 3: Agent retries the operation
        → Hook fires → Finds valid bypass token → Allows → Logs bypass event
```

The agent cannot execute Step 2 because:
- `gdev hook bypass-next` via Claude Code's Bash tool is itself blocked by SP-DENY-BYPASS-CMD (tier 1 rule)
- Even if SP-DENY-BYPASS-CMD did not exist, the developer would see the bypass command in Claude Code's permission prompt
- The developer's separate terminal is outside the agent's control entirely

This creates a **cryptographic separation** between the agent's execution environment (Claude Code's tool pipeline) and the developer's direct terminal access. The agent cannot bridge this gap because Claude Code does not provide a tool for executing commands in the developer's terminal — it only has the Bash tool, which executes in a sandboxed subprocess.

### 5.3 Challenge-Response (TOTP/Display Code)

**Concept**: gdev displays a random code in the terminal when a bypass is needed. The developer must type this code to authorize.

**Analysis**: This is a stronger variant of the out-of-band channel. However:
- The challenge code is displayed in Claude Code's output (the hook's stderr message)
- The agent can read Claude Code's output (it sees the full conversation)
- If the response is entered via a Bash command, the agent could type it

**Problem**: In Claude Code's architecture, the agent sees everything the developer sees. There is no private channel between gdev and the developer that excludes the agent. The agent reads the challenge code from the hook's error message, then types the response via the Bash tool.

**Mitigation**: Display the challenge code via a separate mechanism — a desktop notification (via `notify-send` on Linux), a separate terminal window, or a file in a directory the agent cannot read. However, all of these add significant complexity.

**Recommendation**: Not recommended for initial implementation. The chained protection pattern (5.2) provides sufficient agent resistance without the complexity of challenge-response. The separate-terminal workflow is effectively a challenge-response where the "challenge" is "go to your terminal and run this command" and the "response" is the bypass token.

### 5.4 Is the Ask Verdict Sufficient as a Bypass Mechanism?

**Conceptual answer**: Yes — if a rule uses the ask verdict, the developer approves or denies in the permission dialog, and no separate bypass mechanism is needed.

**Practical answer**: No — for three reasons:

1. **Bug #39344**: ask verdicts silently override deny rules, potentially weakening security
2. **Bug #52822**: ask verdicts may not work reliably in all Claude Code versions
3. **Permission mode bypass**: In `dontAsk` and `bypassPermissions` modes, ask verdicts are auto-approved with no human interaction

For tier 3 rules (standard consulting), the ask verdict is acceptable because these rules protect against mistakes, and the bugs represent edge cases that are unlikely to cause harm in practice.

For tier 2 rules (interactive bypass), the ask verdict is NOT sufficient because the bugs could allow an agent to bypass configuration protection silently. The CLI bypass mechanism provides a more reliable human-in-the-loop checkpoint.

For tier 1 rules (absolute deny), no verdict other than deny is acceptable. These rules use exit code 2 unconditionally.

---

## 6. Recommended Bypass Architecture

### 6.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                  gdev Hook System                        │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐ │
│  │   Tier 1    │  │    Tier 2    │  │     Tier 3     │ │
│  │ ABSOLUTE    │  │ INTERACTIVE  │  │   STANDARD     │ │
│  │ DENY        │  │ BYPASS       │  │   BYPASS       │ │
│  │             │  │              │  │                │ │
│  │ exit 2      │  │ exit 2       │  │ exit 2         │ │
│  │ always      │  │ unless valid │  │ unless magic   │ │
│  │             │  │ bypass token │  │ comment found  │ │
│  │ No bypass.  │  │              │  │                │ │
│  │ Period.     │  │ Token from:  │  │ Comments:      │ │
│  │             │  │ gdev hook    │  │ # gdev-allow-  │ │
│  │             │  │ bypass-next  │  │ destructive    │ │
│  │             │  │ (run in      │  │ # gdev-allow-  │ │
│  │             │  │  terminal)   │  │ credential     │ │
│  └─────────────┘  └──────────────┘  └────────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Mandatory Audit Layer                │   │
│  │  Every bypass attempt → session log + bypass log  │   │
│  │  Tier 1 attempts → CRITICAL alert                 │   │
│  │  Hash chain for tamper detection                  │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### 6.2 Integration with Verdict Model

| Tier | Hook Exit | JSON Output | Claude Code Behavior |
|------|-----------|-------------|---------------------|
| 1 (absolute deny) | Exit 2 | stderr: block message | Tool call blocked unconditionally |
| 2 (interactive, no token) | Exit 2 | stderr: block message + bypass instructions | Tool call blocked; message tells developer how to create token |
| 2 (interactive, valid token) | Exit 0 | No JSON (allow) | Tool call proceeds; token consumed; bypass logged |
| 3 (standard, no comment) | Exit 2 | stderr: block message + comment instructions | Tool call blocked; message tells developer to add comment |
| 3 (standard, with comment) | Exit 0 | No JSON (allow) | Tool call proceeds; bypass logged |

**Why exit code 2 for all denials**: Exit code 2 blocks the tool call *before* Claude Code's permission rules are evaluated. This means the block applies even if `permissions.allow` rules would otherwise permit the operation. It is immune to the bug where ask verdicts override deny rules (#39344). It is the most reliable blocking mechanism available.

**Why no JSON for allows**: When a bypass token or magic comment is valid, the hook simply exits 0 with no output, which means "defer to default behavior." This avoids depending on the JSON `permissionDecision` field, which has known reliability issues (#52822).

### 6.3 Agent Exploitation Prevention

| Attack | Defense | How It Works |
|--------|---------|-------------|
| Agent adds `# gdev-allow-destructive` to command | Visible in permission prompt | Developer sees the comment and can reject |
| Agent runs `gdev hook bypass-next` via Bash | Blocked by SP-DENY-BYPASS-CMD (tier 1) | Exit code 2 unconditionally |
| Agent runs `export GDEV_HOOK_BYPASS=1` | Blocked by SP-DENY-BYPASS-EXPORT (tier 1) | Exit code 2 unconditionally |
| Agent modifies settings.json to remove hooks | Blocked by SP-DENY-SETTINGS-WRITE (tier 1) | Exit code 2 unconditionally |
| Agent uses `sed -i` on settings.json | Blocked by SP-DENY-SETTINGS-BASH (tier 1) | Exit code 2 unconditionally |
| Agent creates symlink to bypass path check | Canonicalized via `realpath` before matching | Symlink resolved to real path |
| Agent instructs subagent to bypass | Subagent inherits parent hooks; SP rules fire | Same tier 1 blocks apply |
| Agent sets `GDEV_SELF_PROTECTION=off` via export | Blocked by SP-DENY-BYPASS-EXPORT pattern | Catches all `export GDEV_*` bypass patterns |
| Developer uses `bypassPermissions` mode | No defense — documented incompatibility | `gdev doctor` warns |

### 6.4 Developer Experience Assessment

| Scenario | Friction Level | Justified? |
|----------|---------------|-----------|
| Normal development (no bypass needed) | Zero | Yes — hooks fire invisibly |
| Destructive command (tier 3) | Minimal — add comment to command | Yes — low risk, high frequency |
| Credential in example file (tier 3) | Minimal — add comment to content | Yes — documentation is common |
| Editing devenv.nix security settings (tier 2) | Moderate — switch to terminal, run command, switch back | Yes — security config changes are rare |
| Editing .mcp.json (tier 2) | Moderate — same as above | Yes — MCP config changes are rare |
| Modifying gdev hooks (tier 1) | High — must use external editor, not Claude Code | Yes — this should be rare and deliberate |
| Disabling all self-protection (last resort) | High — must restart Claude Code with env var | Yes — this is the nuclear option |

The friction gradient matches the risk gradient: routine operations have zero friction, rare security-sensitive operations have moderate friction, and actions that could compromise the security system have maximum friction.

### 6.5 Implementation Priorities

1. **Immediate (ship with self-protection)**: Tier 1 absolute-deny rules with exit code 2. Tier 3 magic comment bypasses (already designed in Phase 32). Mandatory bypass audit logging in session log.
2. **Near-term (next iteration)**: Tier 2 CLI bypass mechanism with token system. Dedicated bypass log (`bypasses.jsonl`). `gdev doctor` check for `bypassPermissions` mode.
3. **Future (hardening)**: Hash chain for bypass log integrity. Notification integration for tier 1 bypass attempts. Time-limited grant tokens. `GDEV_SELF_PROTECTION=off` pre-session override.

---

## Depth Checklist

- [x] **Underlying mechanism explained** — six bypass mechanisms analyzed with precise mechanics (how tokens work, how chained protection creates deadlocks, how exit code 2 differs from JSON verdicts)
- [x] **Key tradeoffs and limitations identified** — ask verdict bugs (#39344, #52822), bypassPermissions incompatibility, approval fatigue, friction vs convenience tradeoffs per tier
- [x] **Compared to alternatives** — SELinux (per-domain permissive), sudo (time-limited elevation), AppArmor (deny in complain mode), git --no-verify (anti-pattern), firewall TTL exceptions, Claude Code sandbox escape (anti-pattern)
- [x] **Failure modes and edge cases** — bypassPermissions mode defeat, token expiry race conditions, chained protection deadlock resolution, hash chain breaks as tamper signal
- [x] **Concrete examples found** — Claude Code issues #39344 and #52822, Claude Code issue #20259 (sandbox escape), Prempti self-protection rules, reasoning-core escape hatches, Phase 32 magic comment design
- [x] **Report is standalone-readable** — contains complete bypass architecture design, tier assignments, audit schema, implementation priorities, and developer experience assessment

---

## Sources

### Internal (from prior research)
- `threat-model-research.md` — 12-vector attack taxonomy, bypass as attack vector
- `prempti-patterns-research.md` — Prempti self-protection rules, bypass comment design
- `fail-policy-research.md` — Fail-closed vs fail-open, severity-tiered policy
- `research-spikes/security-tooling-evaluation-gdev/reasoning-core-research.md` — reasoning-core escape hatches (magic comments, CLI bypass, env var)
- `research-spikes/security-tooling-evaluation-gdev/prempti-research.md` — Prempti's deny/ask verdict model
- `implementation-plans/gdev-secure-devenv-bootstrap/phases/32-managed-hook-policy-consulting-enforcement.md` — Phase 32 hook architecture, magic comment design

### External (saved to docs/)
- `docs/claude-code-hooks-response-format.md` — Hook JSON response format, permission decision values, ask/deny/allow behavior
- `docs/claude-code-hook-verdict-bug-52822.md` — permissionDecision "allow" fails to suppress native prompt (regression)
- `docs/claude-code-issue-39344-ask-overrides-deny.md` — ask verdict silently overrides deny rules (security vulnerability)
- `docs/claude-code-sandbox-escape-hatch-issue-20259.md` — Sandbox escape hatch design: opt-in, audit logging, differentiated UI
- `docs/owasp-logging-cheat-sheet.md` — OWASP audit logging requirements: fields, tamper protection, override logging
- `docs/prefactor-audit-trails-ai-agents.md` — AI agent audit trail best practices: immutable storage, agent identities, compliance mapping
- `docs/graphite-git-no-verify.md` — git --no-verify: risks, legitimate uses, why bypassing should be exceptional
