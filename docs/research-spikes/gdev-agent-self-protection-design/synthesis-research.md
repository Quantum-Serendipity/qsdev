# Synthesis: gdev Agent Self-Protection Architectural Design

## Purpose

This document synthesizes all 7 research reports from the gdev-agent-self-protection-design spike into a coherent architectural design. It identifies contradictions and tensions between reports, produces a consolidated rule catalog, resolves open questions from research.md, and recommends implementation phases.

Reports synthesized:
1. `threat-model-research.md` -- 12-vector attack taxonomy, 5 rule sets A-E
2. `prempti-patterns-research.md` -- 6 Prempti rule translations, 6 gdev-specific rules, rule format design
3. `fail-policy-research.md` -- severity-tiered failure policy, fail-closed harness
4. `verdict-model-research.md` -- 4-verdict model, deny-overrides combining, 3-tier rule precedence
5. `canonical-path-research.md` -- 9 bypass techniques, 2-tier canonicalization, Bash extraction limits
6. `monitor-mode-research.md` -- per-rule mode, enforce_always, 5-day calibration
7. `escape-hatch-research.md` -- 3-tier bypass policy, chained protection, audit logging

---

## 1. Contradictions and Tensions Between Reports

### 1.1 The "ask" Verdict Paradox

**Tension**: The verdict model report assigns "ask" to 10 rules (43% of the catalog). The escape hatch report says `permissionDecision: "ask"` is unreliable due to two open bugs:
- Bug #39344: ask silently overrides `permissions.deny` rules in settings.json
- Bug #52822: ask/allow may not work reliably in some Claude Code versions

The verdict model report itself documents bug #39344 and says "gdev must never use ask for operations covered by deny rules." But the escape hatch report goes further: it concludes the ask verdict is "currently unsuitable as a self-protection bypass mechanism" and recommends using exit code 2 for ALL hard denials.

**Resolution**: These reports are not contradictory -- they represent an evolution in understanding as the research progressed. The escape hatch report was written last and incorporates lessons from all prior reports. The reconciled position is:

1. **Use exit code 2 (not JSON ask) for all deny verdicts.** This is immune to both bugs and is the most reliable blocking mechanism.
2. **Tier 2 "ask" rules should NOT use Claude Code's `permissionDecision: "ask"` at all.** Instead, they should use exit code 2 with an error message instructing the developer to run `gdev hook bypass-next` in their terminal. This converts the "ask" concept into a separate-terminal workflow that avoids both bugs entirely.
3. **The 10 "ask" rules from the verdict model are reclassified**: they are interactive-bypass rules (Tier 2) that block by default and require an explicit human bypass action, not Claude Code ask prompts.
4. **When bugs #39344 and #52822 are resolved**, gdev can revisit using `permissionDecision: "ask"` for lower-risk Tier 2 rules. Until then, avoid it for anything security-relevant.

### 1.2 Hook Script Consolidation: 2 Scripts vs 5 Rule Sets

**Tension**: The Prempti patterns report recommends consolidating all self-protection rules into 2 hook scripts (one for Bash, one for Write/Edit/Read) to minimize process spawning. The threat model report defines 5 rule sets (A through E), each implying a separate logical grouping. The monitor mode report implies per-rule granularity for mode control.

**Resolution**: The 5 rule sets are *logical groupings*, not physical scripts. The 2-script consolidation from the Prempti patterns report is the correct *deployment architecture*:

1. **`self-protection-bash.sh`** -- Single PreToolUse hook for the Bash matcher. Contains all rules from Rule Sets B, C, and the Bash portions of E. Evaluates all applicable rules per invocation (all-match-then-escalate).
2. **`self-protection-files.sh`** -- Single PreToolUse hook for Write|Edit|Read matcher. Contains all rules from Rule Sets A, and the Write/Edit portions of E.
3. **Rule Set D (SessionStart integrity check)** -- A third script for the SessionStart hook event (different hook event type, cannot be consolidated into PreToolUse scripts).

Per-rule mode control (monitor/enforce/off) is implemented *within* the consolidated scripts. Each script loads rule states from `~/.qsdev/hook-state.yaml` and applies mode overrides per-rule after evaluation. The physical deployment is 3 scripts; the logical organization is per-rule.

### 1.3 "enforce_always" vs "Tier 1 never bypassable" -- Same Set?

**Tension**: The monitor mode report defines `enforce_always` for "nuclear" self-protection rules that enforce even in monitor mode, naming SP-4 (settings.json write) and SP-5 (policy-limits.json write) as candidates. The escape hatch report defines Tier 1 (absolute deny, never bypassable) with 8 rules. Are these the same set?

**Resolution**: No -- they overlap but serve different purposes:

- **`enforce_always`** is a *monitor mode* concept: these rules block even when their category is in monitor mode. It prevents the calibration window from being exploitable.
- **Tier 1 (absolute deny)** is a *bypass* concept: these rules have no bypass mechanism (no magic comments, no CLI tokens, no env vars).

All `enforce_always` rules should also be Tier 1 (if a rule enforces even in monitor mode, it should also be non-bypassable). But Tier 1 may contain additional rules that are bypassable in the sense that the developer can edit files outside Claude Code, but which are never bypassable *through* Claude Code.

The consolidated set of rules that are both `enforce_always` AND Tier 1:

| Rule | Why enforce_always | Why Tier 1 |
|------|-------------------|-----------|
| SP-DENY-SETTINGS-WRITE | Disabling hooks during calibration defeats calibration | Root of trust for all hooks |
| SP-DENY-SETTINGS-BASH | Same | Same (Bash vector for same file) |
| SP-DENY-HOOKS-WRITE | Deleting hooks during calibration disables enforcement | Enforcement mechanism itself |
| SP-DENY-HOOKS-BASH | Same | Same (Bash vector) |
| SP-DENY-POLICY-LIMITS | Raising limits during calibration removes constraints | Agent resource constraints |
| SP-DENY-AUDIT-WRITE | Destroying evidence during calibration hides attacks | Evidence integrity |
| SP-DENY-GDEV-CLI | Running `gdev disable hooks` during calibration removes all protection | Administrative control |
| SP-DENY-BYPASS-EXPORT | `export GDEV_HOOK_BYPASS` during calibration disables everything | Meta-protection |
| SP-DENY-BYPASS-CMD | `gdev hook bypass-next` via Bash during calibration creates unauthorized tokens | Meta-protection |

### 1.4 Fail-Closed Exit Code Semantics

**Tension**: The fail-policy report uses `exit 2` as the fail-closed trap fallback. The verdict model report specifies `exit 0 + JSON deny` for structured denials with rich context. The escape hatch report recommends `exit 2` for ALL hard denials.

**Resolution**: Use BOTH, in a layered pattern already described in the verdict model report:

1. **Happy path (rule matched, deny decision made)**: Output structured JSON deny via exit 0. This provides rich `permissionDecisionReason` and `additionalContext` to the agent, guiding it toward compliant behavior.
2. **Error path (hook crashed)**: The `trap ... ERR EXIT` fires exit 2 with a stderr message. This is the fail-closed fallback.
3. **The wrapper pattern**: The fail-closed harness wraps the entire script. If the script reaches a clean deny decision, it disarms the trap (`trap - EXIT`) and outputs JSON deny via exit 0. If the script crashes at any point, the trap fires exit 2.

This means exit 2 is reserved for *errors*, not for *policy denials*. The distinction matters for developer experience: exit 2 denials produce a brief stderr message ("hook error -- blocked for safety"), while exit 0 JSON denials produce structured context that helps the agent understand what was blocked and why.

**Exception**: The escape hatch report recommends exit 2 even for clean denials to avoid depending on JSON `permissionDecision`, which has known bugs. This is valid for self-protection rules specifically (where the ask/deny bug interaction is most dangerous). For self-protection, use exit 2 with a detailed stderr message rather than JSON deny. For other hook categories (destructive prevention, credential scanning), JSON deny with exit 0 is acceptable because these rules do not interact with deny rules in settings.json.

### 1.5 Process-Kill Relevance

**Tension**: The threat model rates process termination as P2 (medium priority) because gdev hooks are short-lived processes. The Prempti patterns report translates Prempti's service-stop rules but notes gdev "doesn't run a daemon like Prempti/Falco." Yet SP-2 (deny process-kill targeting gdev infrastructure) is included in the core rule set.

**Resolution**: SP-2 as written in the Prempti patterns report is correct but should be scoped narrowly. The realistic attack surface for gdev is:
- `pkill pre-commit` during git hook execution
- `nix profile remove gdev` to uninstall gdev
- `rm`/`mv`/`chmod` targeting hook scripts (already covered by SP-3)

The process-kill rules should focus on `pkill pre-commit`, `pkill gitleaks`, `pkill ripsecrets` (tools that run as subprocesses during git operations), and Nix package removal. The service-stop patterns from Prempti (systemctl, launchctl, taskkill) are irrelevant for gdev and should be omitted.

### 1.6 Subagent Prompt Screen Reliability

**Tension**: The threat model (Rule Set C) proposes regex screening of Task tool prompts for mutation verbs paired with protected path references. The canonical path report concludes that "extracting write targets from arbitrary [text] is equivalent to static analysis" and is undecidable. The verdict model assigns the subagent prompt screen an "ask" verdict acknowledging high false positive rates.

**Resolution**: Subagent prompt screening is a speed bump with known limitations. The design is sound as one layer in a defense-in-depth strategy:
1. **Prompt screen (speed bump)**: Catches naive/direct attacks ("edit ~/.claude/settings.json and remove hooks")
2. **Subagent inherits parent hooks**: Even if the prompt screen misses a sophisticated request, the subagent's actual Write/Edit/Bash tool calls will trigger the same self-protection hooks that protect against the parent agent
3. **Verdict**: Use exit 2 with a message asking the developer to verify (effectively a Tier 2 interactive bypass), not `permissionDecision: "ask"` (which is buggy)

The subagent prompt screen should be implemented but should NOT be relied upon as a primary defense. It is supplementary to the Write/Edit/Bash hooks that fire on the subagent's actual tool calls.

---

## 2. Consolidated Rule Catalog

### 2.1 Rule Naming Convention

Rules use a hierarchical ID: `SP-{NN}` for core self-protection, `MCP-{NN}` for MCP poisoning detection, `CFG-{NN}` for configuration guards.

### 2.2 Complete Rule Table

| ID | Description | Hook Type | Matcher | Default Verdict | Bypass Tier | Monitor Behavior | Fail Policy |
|----|-------------|-----------|---------|----------------|-------------|------------------|-------------|
| **SP-01** | Deny gdev CLI invocation via Bash | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-02** | Deny process-kill targeting security tools (pre-commit, gitleaks, ripsecrets) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-03** | Deny Write/Edit to `~/.qsdev/` (gdev install prefix) | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-04** | Deny Write/Edit to `~/.claude/settings.json` and `settings.local.json` | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-05** | Deny Write/Edit to `~/.claude/policy-limits.json` | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-06** | Deny Bash commands writing to `~/.claude/settings*` (sed, cat, tee, cp, mv, etc.) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-07** | Deny Bash commands writing to `~/.qsdev/` (all write/delete/permission operations) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-08** | Deny `export GDEV_HOOK_BYPASS`, `export GDEV_BYPASS_*`, `export GDEV_SELF_PROTECTION` | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-09** | Deny `gdev hook bypass-next` via Bash tool (chained protection) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-10** | Deny `export HISTSIZE=0`, `history -c` (audit trail preservation) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-11** | Deny `git config core.hooksPath` (pre-commit hook disabling) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-12** | Deny obfuscation patterns: `base64 -d \| bash`, `base64 --decode \| sh` | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-13** | Deny `nix profile remove.*gdev` (Nix package removal) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-14** | Deny Write/Edit to `~/.config/nix/nix.conf` | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **SP-15** | Screen Task tool prompts for mutation verbs + protected path references | PreToolUse | Task | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **SP-16** | Block Read of `~/.claude/settings.json` (reconnaissance prevention) | PreToolUse | Read | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-01** | Block edits to devenv.nix that weaken security settings | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-02** | Block edits to .pre-commit-config.yaml | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-03** | Block edits to .gdev.yaml that change compliance_level or disable tools | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-04** | Block edits to CLAUDE.md that remove gdev-managed sections | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-05** | Block writes to `.claude/commands/`, `.claude/rules/`, `.claude/agents/` | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-06** | Block edits to `.mcp.json` | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **CFG-07** | Block .npmrc changes setting `ignore-scripts=false` | PreToolUse | Write\|Edit\|Bash | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **MCP-01** | Block MCP config with command from temp directory | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **MCP-02** | Block MCP config with IOC domain in server URL | PreToolUse | Write\|Edit | deny | 1 (absolute) | enforce_always | fail-closed |
| **MCP-03** | Block MCP server execution from temp directory (Bash) | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **MCP-04** | Block npm/pip install from IOC domain | PreToolUse | Bash | deny | 1 (absolute) | enforce_always | fail-closed |
| **MCP-05** | Block MCP config with base64-encoded commands | PreToolUse | Write\|Edit | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **MCP-06** | Block `claude mcp add` / `claude mcp install` | PreToolUse | Bash | deny (with bypass instruction) | 2 (interactive) | standard | fail-closed |
| **INT-01** | SessionStart: verify hook scripts exist and are executable | SessionStart | (all) | warn | N/A | standard | fail-open |
| **INT-02** | SessionStart: verify settings.json contains expected hook entries | SessionStart | (all) | warn | N/A | standard | fail-open |
| **INT-03** | SessionStart: verify hook script checksums match expected values | SessionStart | (all) | warn | N/A | standard | fail-open |

### 2.3 Rule Counts by Category

| Category | Deny (Tier 1) | Deny (Tier 2) | Warn | Total |
|----------|--------------|---------------|------|-------|
| Self-protection (SP-*) | 14 | 2 | 0 | 16 |
| Configuration guard (CFG-*) | 0 | 7 | 0 | 7 |
| MCP poisoning (MCP-*) | 4 | 2 | 0 | 6 |
| Integrity checks (INT-*) | 0 | 0 | 3 | 3 |
| **Total** | **18** | **11** | **3** | **32** |

### 2.4 Design Decision: All Security Rules Default to Deny

A major departure from the verdict model report's original assignment of 10 "ask" rules: after the escape hatch research revealed that Claude Code's `permissionDecision: "ask"` has two critical bugs, ALL security-relevant rules now default to `deny` with exit code 2. Rules that were originally "ask" are now Tier 2 interactive-bypass denials -- they block with a message telling the developer how to create a bypass token in their terminal. This is functionally equivalent to "ask" but does not depend on Claude Code's buggy ask infrastructure.

The "warn" verdict is reserved for integrity checks (INT-*) at SessionStart, which are advisory and should never block session creation.

---

## 3. Architectural Recommendations

### 3.1 Hook Architecture: 3 Physical Scripts + 1 Future Binary

**Phase 32 deployment** (bash scripts):

| Script | Hook Event | Matcher | Rules |
|--------|-----------|---------|-------|
| `self-protection-bash.sh` | PreToolUse | Bash | SP-01, SP-02, SP-06 through SP-13, SP-15 (Task in same script via matcher check), MCP-03, MCP-04, MCP-06 |
| `self-protection-files.sh` | PreToolUse | Write\|Edit\|Read | SP-03, SP-04, SP-05, SP-14, SP-16, CFG-01 through CFG-07, MCP-01, MCP-02, MCP-05 |
| `self-protection-integrity.sh` | SessionStart | (all) | INT-01, INT-02, INT-03 |

**Future Go binary** (`~/.qsdev/bin/gdev-hooks`): All 3 scripts consolidated into a single compiled binary. The binary eliminates shell/Python dependency issues, enables tree-sitter-bash AST analysis for Bash commands, and provides the cleanest fail-closed implementation.

### 3.2 Rule Evaluation Pipeline

```
Tool Call arrives
    |
    v
[1. Read stdin JSON]
    - Extract tool_name, tool_input
    |
    v
[2. Canonicalize Paths]
    - For Write/Edit/Read: realpath on tool_input.file_path
    - For Bash: expand ~/HOME in command, extract redirect targets
    - Two-tier: realpath (filesystem) -> realpath -m (lexical fallback)
    |
    v
[3. Load Rule States]
    - Read ~/.qsdev/hook-state.yaml for per-rule mode (enforce/monitor/off)
    - Skip rules with mode=off
    |
    v
[4. Evaluate ALL Applicable Rules]
    - Each rule evaluates independently
    - Each produces a verdict (deny/allow/warn)
    - enforce_always rules block regardless of mode
    - Monitor-mode rules evaluate fully but record "would-deny"
    |
    v
[5. Combine Verdicts (deny-overrides)]
    - Any deny -> combined verdict = deny
    - No deny, any warn -> combined verdict = warn
    - All allow -> combined verdict = allow
    - Managed-policy deny cannot be overridden by user/project rules
    |
    v
[6. Apply Mode Override]
    - If combined verdict = deny AND all matching deny rules are in monitor mode
      AND none have enforce_always: override to allow, log as would-deny
    - If any enforce_always rule denied: verdict stays deny
    |
    v
[7. Emit Verdict]
    - Deny: exit 2 with stderr message (for Tier 1 and Tier 2 rules)
    - Warn: exit 0 with stderr advisory
    - Allow: exit 0 with no output
    |
    v
[8. Audit Log]
    - Append JSONL record to ~/.qsdev/audit/sessions/<date>/<session>.jsonl
    - Record: rule_id, mode, evaluated_verdict, effective_verdict, operation details
```

### 3.3 Deployment Strategy

**Phase 1 -- Bash scripts with Phase 32 infrastructure:**
- Deploy as 3 additional hook scripts alongside the existing 6 Phase 32 hooks
- Use the existing `gdev enable hooks` / `gdev disable hooks` infrastructure
- Register in `~/.claude/settings.json` under `[gdev-managed-policy]` section markers
- All rules start in monitor mode except `enforce_always` rules

**Phase 2 -- Go binary consolidation (future):**
- Consolidate all hook scripts (Phase 32 originals + self-protection) into a single Go binary
- The binary receives the hook event type and matcher as CLI arguments
- All rule evaluation, path canonicalization, and verdict combining happens in Go
- Eliminates shell/Python dependency issues, enables tree-sitter-bash AST analysis
- Rule definitions compiled into the binary as Go structs

### 3.4 Configuration Model

**Compiled defaults (non-overridable):**
- All Tier 1 rules with their conditions, verdicts, and enforcement behavior
- Path canonicalization logic
- Fail-closed harness
- Deny-overrides combining algorithm

**YAML user overrides (`~/.qsdev/rules/self-protection.yaml`):**
- Disable specific Tier 2 rules (e.g., disable SP-16 read-settings if too noisy)
- Add custom protected paths (e.g., project-specific infrastructure directories)
- Change Tier 2 rule mode (enforce/monitor/off)
- Cannot override Tier 1 rules or change enforce_always behavior

**YAML project overrides (`.gdev.yaml`):**
- Add project-specific configuration guard rules
- Tier 2 and Tier 3 only -- cannot weaken managed-policy rules

**Mode state (`~/.qsdev/hook-state.yaml`):**
- Per-rule mode (enforce/monitor/off) with timestamps
- Monitor event counters
- Managed by `gdev hook monitor/enforce/off` commands
- Protected by SP-07 (deny writes to ~/.qsdev/)

### 3.5 Audit Trail Design

**Single unified JSONL trail:** All hook events (enforcement, monitor, bypass, error) write to the same `~/.qsdev/audit/sessions/<date>/<session-id>.jsonl` file used by the SOC 2 audit hook (Phase 32, Unit 32.4). Event types are distinguished by the `event_type` field:

| Event Type | When | Key Fields |
|-----------|------|------------|
| `hook_evaluation` | Every non-allow verdict | mode, rule_id, evaluated_verdict, effective_verdict, matched_pattern |
| `hook_bypass` | Bypass token consumed or magic comment used | bypass_method, bypass_reason, original_verdict, token_age |
| `hook_error` | Hook script crash or timeout | error type, exit code, stderr excerpt |
| `hook_integrity` | SessionStart integrity check | check_name, status (pass/warn/fail) |

**Secondary bypass log:** `~/.qsdev/audit/bypasses.jsonl` contains only bypass events across all sessions. This is a convenience view for compliance review, not the source of truth.

**Tamper-evident hash chain** (future): Each bypass record includes a SHA-256 hash linking it to the previous record. `gdev doctor` can verify chain integrity.

---

## 4. Resolved Open Questions

### From research.md Open Questions:

**Q1: Should the fail-closed harness be implemented in shell/Python wrapper scripts (Phase 32 timeline) or deferred to the Go binary?**

**Resolved**: Implement in bash scripts NOW (Phase 32 timeline). The fail-closed wrapper pattern is straightforward in bash (`trap ... ERR EXIT` + `exit 2`). The Go binary is cleaner but further out. Self-protection cannot wait for the Go binary -- the gap exists today. The bash implementation provides immediate protection; the Go binary provides better failure characteristics later.

**Q2: What should the consecutive-failure threshold be before switching from fail-closed to degraded mode?**

**Resolved**: Do NOT implement automatic degradation. A fail-closed hook that is persistently broken should remain broken (blocking) until the developer fixes it. Automatic degradation creates an exploitable attack vector (crash the hook 3 times, then proceed freely). The developer's escalation path is manual: `gdev doctor` -> `gdev doctor --fix` -> `GDEV_HOOK_BYPASS=<rule>` -> `gdev disable hooks`.

**Q3: Should monitor mode have a maximum duration?**

**Resolved**: No hard maximum. Soft reminders at 5 days (initial calibration period), escalating at 10 and 20 days. `gdev doctor` flags rules in monitor mode for >20 days as a health issue. Never auto-promote to enforcement. This matches the universal pattern across all 6 surveyed security systems (SELinux, AppArmor, Defender ASR, AWS WAF, seccomp, Kubernetes) -- none auto-expire their monitor mode.

**Q4: Should project-level `.claude/settings.json` writes be permitted while user-level is denied?**

**Resolved**: Yes. Project-level `.claude/settings.json` gets Tier 2 (interactive bypass, CFG-05 covers `.claude/commands/`, `.claude/rules/`, `.claude/agents/`). User-level `~/.claude/settings.json` gets Tier 1 (absolute deny, SP-04). The threat model validates this: project settings are a legitimate edit target (the agent helps configure the project), but user settings contain the hook registrations that protect all projects.

**Q5: Should the ask verdict for reading settings.json be included?**

**Resolved**: Yes, as SP-16 with Tier 2 interactive bypass. It CAN be disabled by user override in `~/.qsdev/rules/self-protection.yaml` for developers who find it too noisy. The risk (reconnaissance -- agent learns hook names, deny patterns) is real but lower than mutation risks. Default: enabled in monitor mode, developer promotes to enforce if desired.

**Q6: What is Claude Code's exact behavior when a hook script is missing or non-executable?**

**Partially resolved**: The fail-policy research established that Claude Code is inherently fail-open for all non-exit-2 errors, including missing scripts (exit 127) and permission denied (exit 126). The compensating control is INT-01 (SessionStart integrity check that warns if hook scripts are missing). Full empirical testing has not been performed, but the fail-closed harness makes this less critical -- if a hook can launch at all, the trap ensures it blocks on error.

**Q7: How should bypass mechanisms be designed for human access but not agent access?**

**Resolved**: The escape hatch report fully answers this with the 3-tier bypass architecture:
- Tier 1: No bypass mechanism at all. Developer edits files outside Claude Code.
- Tier 2: `gdev hook bypass-next` run in a separate terminal (blocked via Bash tool by SP-09, forcing out-of-band human action).
- Tier 3: Magic comments (`# gdev-allow-destructive`) visible in Claude Code's permission prompt.
- `GDEV_HOOK_BYPASS=1` is replaced with per-rule `GDEV_BYPASS_<RULE>=1` env vars, set before Claude Code starts.

**Q8: Should hook scripts be deployed with immutable file attributes or via NixOS home-manager?**

**Remains open**: Both approaches have merit. `chattr +i` requires root for initial setup. NixOS home-manager deployment means hooks are Nix store symlinks (read-only by Nix design). The canonical path report notes that if hooks are Nix-managed symlinks, the path matching must account for Nix store paths (match against both raw and canonicalized paths). This is an implementation detail for the deployment phase, not a design question. Recommend home-manager for NixOS deployments, checksum verification via INT-03 for non-NixOS.

**Q9: Should gdev implement a "new repository" warning when CLAUDE.md contains security-related instructions?**

**Remains open**: This is a valuable feature but is out of scope for self-protection. It belongs in a separate "repository trust" or "CLAUDE.md sanitization" feature. The CFG-04 rule (block edits to CLAUDE.md that remove gdev sections) provides partial protection against the reverse scenario (agent removing gdev instructions), but does not address a poisoned repo's CLAUDE.md injecting anti-security instructions.

---

## 5. Implementation Plan Recommendations

### Recommended New Phases for gdev Implementation Plan

#### Phase 33: Self-Protection Core (Tier 1 Rules)

**Scope**: Implement the 14 Tier 1 (absolute deny) SP-* rules plus the 4 Tier 1 MCP-* rules as 2 bash hook scripts, the fail-closed harness, path canonicalization, and the `enforce_always` mechanism.

**Units**:
- 33.1: Self-protection Bash guard script (SP-01, SP-02, SP-06 through SP-13)
- 33.2: Self-protection file guard script (SP-03, SP-04, SP-05, SP-14)
- 33.3: MCP poisoning detection (MCP-01 through MCP-04, integrated into 33.1/33.2)
- 33.4: Fail-closed harness wrapper for all security-critical hooks (retrofit Phase 32 hooks)
- 33.5: Path canonicalization (`canonicalize_path()` function, integrated into both scripts)
- 33.6: Deploy self-protection hooks via `gdev enable hooks --self-protection`

**Dependencies**: Phase 32 complete (hook deployment infrastructure).
**Priority**: P0 -- this closes the most critical gap identified in the threat model.

#### Phase 34: Self-Protection Interactive Bypass and Monitor Mode

**Scope**: Implement Tier 2 interactive-bypass rules (SP-15, SP-16, CFG-01 through CFG-07, MCP-05, MCP-06), the `gdev hook bypass-next` CLI command, the bypass token system, per-rule monitor mode, and the audit integration.

**Units**:
- 34.1: Tier 2 rule evaluation in self-protection scripts (content inspection for devenv.nix, .pre-commit-config.yaml, CLAUDE.md section markers)
- 34.2: `gdev hook bypass-next` CLI command with token generation (single-use, 5-minute TTL)
- 34.3: Token consumption logic in hook scripts (check `~/.qsdev/bypass-tokens/`, consume on match)
- 34.4: Per-rule monitor mode (`gdev hook monitor/enforce/off` commands, `hook-state.yaml`)
- 34.5: `gdev hook audit` command (review monitor-mode events, promote clean rules)
- 34.6: Verdict audit logging integration (extend SOC 2 JSONL trail with verdict records)
- 34.7: SessionStart integrity check script (INT-01, INT-02, INT-03)

**Dependencies**: Phase 33 complete.
**Priority**: P1 -- provides the calibration workflow and configuration protection.

#### Phase 35: Self-Protection Hardening

**Scope**: Go binary consolidation, tree-sitter-bash AST analysis, YAML user overrides, bypass log hash chain, `gdev doctor` self-protection health checks.

**Units**:
- 35.1: Go hook binary (`gdev-hooks`) consolidating all hook logic
- 35.2: Tree-sitter-bash integration for AST-based redirect extraction
- 35.3: YAML user override loader (`~/.qsdev/rules/self-protection.yaml`)
- 35.4: Bypass log with SHA-256 hash chain (`~/.qsdev/audit/bypasses.jsonl`)
- 35.5: `gdev doctor` self-protection health checks (hook integrity, monitor mode staleness, bypassPermissions detection)
- 35.6: `GDEV_SELF_PROTECTION=off` pre-session escape hatch

**Dependencies**: Phase 34 complete.
**Priority**: P2 -- hardening for production maturity.

### Phase Ordering Rationale

Phase 33 first because it closes the P0 gaps (settings.json/hook script protection, Bash guard). These are the rules that prevent the agent from dismantling the security system entirely.

Phase 34 second because interactive bypass and monitor mode are required for real-world usability. Without bypass mechanisms, Tier 2 rules cannot be deployed (they would block legitimate devenv.nix edits with no override). Without monitor mode, calibrating false positives requires production incidents.

Phase 35 last because it is optimization and hardening. The bash scripts from Phases 33-34 provide functional protection; the Go binary provides better performance, reliability, and analysis depth.

---

## 6. Key Architectural Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Verdict delivery mechanism | Exit code 2 for all security denials | Avoids Claude Code bugs #39344 and #52822; most reliable blocking mechanism |
| "Ask" implementation | Block with message + out-of-band CLI bypass | Claude Code's `permissionDecision: "ask"` is buggy; separate-terminal workflow is more reliable |
| Hook consolidation | 3 scripts (Bash, Files, SessionStart) | Minimizes process spawning while separating hook event types |
| Rule evaluation | All-match-then-escalate with deny-overrides | Order-independent, complete audit trail, defense-in-depth |
| Fail policy | Severity-tiered (fail-closed for security, fail-open for advisory) | Balances protection against paralysis; matches enterprise firewall patterns |
| Monitor mode | Per-rule with enforce_always exception | Enables incremental enforcement; prevents calibration window exploitation |
| Bypass tiers | 3 tiers (absolute/interactive/magic-comment) | Friction proportional to risk; agent cannot bypass Tier 1-2 |
| Path canonicalization | Two-tier realpath (filesystem first, lexical fallback) | Handles symlinks, relative paths, non-existent files; adapted from Prempti |
| Configuration model | Compiled defaults + YAML user overrides | Type-safe defaults, user-extensible, no config-file-missing failure mode |
| Audit trail | Unified JSONL with mode/verdict fields | Single source of truth; matches universal pattern across 6 surveyed systems |
| Hook type | `command` only, never `prompt`/`agent` | Deterministic evaluation immune to prompt injection |
| Self-protection scope | 32 rules across 4 categories | Comprehensive coverage of 12 attack vectors from threat model |

---

## Depth Checklist

- [x] Underlying mechanism explained -- complete architectural design with evaluation pipeline, verdict delivery, and deployment strategy
- [x] Key tradeoffs identified -- ask-verdict bugs vs UX, fail-closed paralysis vs fail-open bypass, monitor window vs security gap, rule count vs false positive fatigue
- [x] Compared to alternatives -- Prempti (Falco daemon), reasoning-core (shadow mode), OWASP framework, Cedar/XACML (combining algorithms), SELinux/AppArmor/ASR/WAF/seccomp/K8s (monitor modes)
- [x] Failure modes described -- Claude Code bugs #39344/#52822, hook crash fail-open default, calibration window exploitation, Bash obfuscation bypasses, hardlink evasion
- [x] Concrete examples found -- implementation pseudocode for all components, complete rule catalog with 32 rules, JSON schema for audit records
- [x] Standalone-readable -- sufficient for implementing self-protection without re-reading individual research reports

## Sources

All 7 research reports from this spike, plus Phase 32 implementation plan. See individual reports for external source citations.
