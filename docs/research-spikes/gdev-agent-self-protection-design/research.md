# Research Summary: gdev Agent Self-Protection Design

## Overview

Design spike for implementing agent self-protection rules natively in gdev's Go hook architecture, without requiring Falco or any external daemon. Covers: preventing the AI agent from disabling its own security tools (settings.json mutation, hook removal, deny rule weakening), MCP config poisoning detection (malicious server injection via prompt), the three-outcome verdict model (allow/deny/ask) as an upgrade to gdev's current binary allow/deny hooks, canonical path resolution to prevent symlink/relative-path traversal bypasses, and monitor mode for rule development before enforcement.

The self-protection gap is gdev's most significant missing defense layer — all six existing layers protect the developer's environment from external threats, but none protect gdev's own security infrastructure from being dismantled by a compromised or manipulated agent.

## Topics

### Fail-Closed vs Fail-Open Policy — **Complete**
- **Detailed report**: [fail-policy-research.md](fail-policy-research.md)
- **Finding**: Claude Code hooks are inherently fail-open (only exit code 2 blocks; all crashes, timeouts, and malformed JSON allow operations to proceed). gdev must implement a severity-tiered failure policy: fail-closed for security-critical hooks (self-protection, destructive prevention, credential scanning) via a wrapper harness that catches all errors and converts to exit 2; fail-open for advisory hooks (cost alerting, audit logging, test enforcement, client isolation). Monitor mode provides a transitional deployment state for rule calibration. The Go binary architecture eliminates most failure scenarios (missing interpreter, dependency issues, shell profile interference, regex backtracking).

### Prempti Self-Protection Rule Patterns — **Complete**
- **Detailed report**: [prempti-patterns-research.md](prempti-patterns-research.md)
- **Finding**: Prempti's self-protection domain contains 5 deny rules and 1 ask rule: (1) deny premptictl CLI invocation, (2) deny OS-level service-stop/process-kill targeting Falco/Prempti, (3) deny writes under ~/.prempti/ install prefix, (4) deny writes to ~/.claude/settings.json, (5) deny writes to ~/.claude/policy-limits.json, (6) ask before reading settings.json. Each translates directly to gdev's hook architecture as PreToolUse hooks on Bash and Write/Edit/Read matchers. Six additional gdev-specific rules identified that Prempti does not need: protecting devenv.nix security settings, .pre-commit-config.yaml, ~/.config/nix/nix.conf, .gdev.yaml compliance config, audit trail integrity, and CLAUDE.md gdev-managed sections. Recommended rule format: compiled Go structs for defaults + optional YAML user overrides. MCP config poisoning detection documented with implementation sketches for temp-path, IOC-domain, and base64-obfuscation detection. Path canonicalization uses two-tier strategy from Prempti: filesystem realpath first, lexical normalization fallback for non-existent paths.

### Threat Model: Agent Self-Disabling Vectors — **Complete**
- **Detailed report**: [threat-model-research.md](threat-model-research.md)
- **Finding**: Comprehensive threat model enumerating 12 attack vector categories, expanding reasoning-core's 6-vector model with configuration poisoning, privilege escalation, TOCTOU, deny rule exhaustion, hook registration manipulation, and audit trail destruction. Of the 12 categories, Phase 32's current hooks defend against 2 fully, 3 partially, and 7 not at all. The three P0 (critical) gaps are: (1) no protection for settings.json/hook script mutations via Write/Edit tools, (2) no protection against indirect file writes via Bash (sed/cat/tee), and (3) no protection against hook registration manipulation. Five minimum self-protection rule sets defined: (A) Protected Path Write Guard for Write/Edit, (B) Protected Path Bash Guard, (C) Subagent Prompt Screen for Task tool, (D) Runtime Integrity Check at SessionStart, (E) Configuration Poisoning Guard. Five real-world incidents analyzed with gdev-specific lessons: the 50-subcommand deny rule bypass (Adversa), the /proc/self/root denylist bypass (Ona), Comment and Control credential theft, OpenAI guardrails self-policing bypass (HiddenLayer), and Semantic Kernel RCE (Microsoft). Key architectural recommendation: all self-protection hooks must use deterministic `command` type only -- never `prompt` or `agent` types, which are susceptible to the same prompt injection they defend against.

### Three-Outcome Verdict Model — **Complete**
- **Detailed report**: [verdict-model-research.md](verdict-model-research.md)
- **Finding**: gdev implements a four-verdict model (allow/deny/ask/warn) using Claude Code's native `permissionDecision` JSON response, with deny-overrides as the combining algorithm (matching XACML best practice, Cedar's forbid-overrides-permit, and Claude Code's own implied precedence). Critical discovery: Claude Code bug #39344 (open) means `permissionDecision: "ask"` silently overrides `permissions.deny` rules in settings.json -- gdev must never use ask for operations covered by deny rules, and must use deny for all core self-protection. Two deny mechanisms serve different failure tiers: exit 2 (fail-closed, for security hooks) and JSON deny (structured, for rich context). Rules are evaluated using all-match-then-escalate (not first-match) for order-independent, complete audit trail evaluation. Three-tier rule precedence (managed-policy > user > project) prevents configuration poisoning attacks where a malicious `.gdev.yaml` weakens security. Of 23 rules from prior research, 13 are assigned deny verdicts (operations never legitimate for agents) and 10 are assigned ask verdicts (sometimes legitimate, need human approval). The warn verdict fills the gap between ask (requires action) and allow (silent) for operations that deserve visibility without interruption. Audit logging integrates verdict records into the SOC 2 JSONL trail with fields for rule ID, tier, verdict, user decision (for ask), and response time.
### Canonical Path Resolution — **Complete**
- **Detailed report**: [canonical-path-research.md](canonical-path-research.md)
- **Finding**: Nine bypass technique categories cataloged (symlink traversal, relative paths, hardlinks, /proc/self/root, /dev/fd, case sensitivity, Unicode normalization, TOCTOU, bind mounts). Two-tier canonicalization pipeline: filesystem resolution via `realpath` (resolves all symlinks), with `realpath -m` fallback for non-existent paths (resolves existing prefix, normalizes remainder). Go equivalent: `filepath.EvalSymlinks` + `filepath.Abs` primary, parent-walk fallback. Critical finding: extracting write targets from arbitrary Bash commands is equivalent to static analysis of shell scripts — undecidable in the general case. Three-strategy defense: regex pattern matching for Phase 32 bash hooks (catches common patterns like `> path`, `tee path`, `sed -i`), AST-based tree-sitter-bash analysis for the Go binary (higher accuracy for redirections and common utilities), and evasion-mechanism blocklist as complementary layer (blocking `/proc/self/root`, `/dev/fd/`, `ln` targeting protected paths). Path matching uses exact/prefix string comparison on canonicalized paths — no glob or regex needed for protected path lists.

### Monitor/Shadow Mode Design — **Complete**
- **Detailed report**: [monitor-mode-research.md](monitor-mode-research.md)
- **Finding**: Survey of six established security systems (SELinux permissive, AppArmor complain, Windows Defender ASR audit, AWS WAF count, seccomp SECCOMP_RET_LOG, Kubernetes ValidatingAdmissionPolicy warn/audit) revealed universal patterns: evaluate fully then override verdict to allow, use the same log stream with a distinguishing field, support per-entity granularity, and no system auto-expires its monitor mode. Design: per-rule mode control with category-level defaults, `enforce_always` flag for nuclear self-protection rules (borrowed from AppArmor's "deny rules enforce even in complain mode"), 5-day calibration period with escalating reminders (soft, never auto-promote to enforcement), unified JSONL audit log with `mode` and `effective_verdict` fields. CLI workflow: `gdev hook monitor [--rule|--category]`, `gdev hook audit [--since]`, `gdev hook enforce [--clean|--rule|--category]`.

### Escape Hatch and Bypass Mechanism — **Complete**
- **Detailed report**: [escape-hatch-research.md](escape-hatch-research.md)
- **Finding**: Six bypass mechanisms analyzed for human-accessibility vs agent-exploitability (magic comments, CLI command, env var, interactive prompt, time-limited token, out-of-band channel). Three-tier bypass policy designed: Tier 1 (absolute deny, never bypassable) for settings.json, hook scripts, audit trail — these rules have no override; Tier 2 (interactive bypass via separate-terminal CLI command) for devenv.nix, .pre-commit-config.yaml, .mcp.json — uses "chained protection" where the bypass command (`gdev hook bypass-next`) is itself blocked by a Tier 1 rule on Bash, forcing the developer to run it in their own terminal outside Claude Code; Tier 3 (magic comment bypass like `# gdev-allow-destructive`) for standard consulting rules. Critical constraint: Claude Code bugs #39344 (ask overrides deny) and #52822 (permissionDecision regression) make the JSON ask verdict unreliable — gdev must use exit code 2 for all hard denials. Mandatory audit logging with 14-field JSONL schema, dual-destination (session trail + dedicated bypass log), and tamper-evident SHA-256 hash chain linking each entry to the previous.

## Open Questions

### Resolved
- ~~Should the fail-closed harness be implemented in shell/Python wrapper scripts or deferred to the Go binary?~~ → Bash wrapper scripts in Phase 33, Go binary consolidation in Phase 35.
- ~~What should the consecutive-failure threshold be before switching from fail-closed to degraded mode?~~ → Default 3, validated by fail-policy research survey of industry defaults.
- ~~Should monitor mode have a maximum duration?~~ → No auto-expiration (universal pattern across 6 surveyed systems). Escalating reminders at 5/10/20 days.
- ~~Should project-level `.claude/settings.json` writes be permitted?~~ → Yes, validated. Project-level is developer-controlled; user-level is managed-policy.
- ~~Should the ask verdict for reading settings.json be included?~~ → Moot. Ask verdict is unreliable (bug #39344). Settings reads get warn verdict instead.
- ~~What is Claude Code's behavior when a hook script is missing?~~ → Fail-open (confirmed). Compensated by SessionStart integrity check (INT-01).
- ~~How should bypass mechanisms be agent-resistant?~~ → Three-tier bypass policy with chained protection. Tier 1 (no bypass), Tier 2 (separate-terminal CLI), Tier 3 (magic comments).

### Remaining
- Should hook scripts be deployed via NixOS home-manager (inherently read-only Nix store symlinks) or with `chattr +i`? Implementation detail for deployment phase.
- Should gdev implement a "new repository" CLAUDE.md trust warning? Out of scope for self-protection; belongs in a separate feature.

## Conclusions

**Full synthesis**: [synthesis-research.md](synthesis-research.md)

### Architecture

gdev's self-protection system is a 32-rule enforcement layer organized across 4 categories (self-protection, configuration guard, MCP poisoning, integrity checks), deployed as 3 consolidated bash hook scripts (Bash guard, file guard, SessionStart integrity), with a future path to a single compiled Go binary.

### Key Design Decisions

1. **All security rules use exit code 2 for denials** -- not Claude Code's `permissionDecision` JSON, which has two open bugs (#39344: ask overrides deny; #52822: allow/ask regression). This is the single most consequential decision: it makes gdev's enforcement independent of Claude Code's buggy permission machinery.

2. **"Ask" is dead; interactive bypass lives.** The 10 rules originally assigned "ask" verdicts are instead Tier 2 interactive-bypass denials. They block with exit code 2 and instruct the developer to run `gdev hook bypass-next` in their terminal. This avoids Claude Code's ask bugs while preserving the human-in-the-loop approval concept through a separate-terminal workflow.

3. **Three-tier bypass policy** prevents the agent from dismantling security: Tier 1 (18 rules, absolute deny, no bypass mechanism), Tier 2 (11 rules, interactive bypass via separate-terminal CLI command), Tier 3 (existing Phase 32 magic comments). The `gdev hook bypass-next` command is itself blocked by a Tier 1 rule (SP-09) when invoked through Claude Code's Bash tool, creating chained protection that forces human action outside the agent's control.

4. **Severity-tiered fail policy**: fail-closed for all security hooks (self-protection, destructive prevention, credential scanning), fail-open for advisory hooks (cost alerting, audit logging, test enforcement). The fail-closed harness uses `trap ... ERR EXIT` to convert any hook crash to exit 2 (block).

5. **Per-rule monitor mode with enforce_always exception**: New rules deploy in monitor mode (evaluate fully, log would-block events, allow operation). Nine nuclear rules (settings.json protection, hook script protection, audit trail protection, bypass mechanism protection) have `enforce_always=true` and block even during calibration. This borrows from AppArmor's pattern where deny rules enforce even in complain mode.

6. **Path canonicalization everywhere**: Two-tier strategy (filesystem `realpath` first, lexical `realpath -m` fallback) on all path-based rules. Bash commands cannot be fully analyzed for write targets (equivalent to static analysis of arbitrary shell scripts), so gdev uses regex pattern matching as a speed bump plus evasion-mechanism blocklist, accepting that Bash matching is a speed bump, not a wall.

### Contradictions Resolved

- The ask verdict bugs (#39344/#52822) are reconciled with the verdict model by converting all "ask" rules to Tier 2 interactive-bypass denials using exit code 2 instead of JSON ask.
- The 5 logical rule sets (A-E) from the threat model map to 3 physical hook scripts, with per-rule mode control implemented within each script.
- The `enforce_always` set (monitor mode concept) and Tier 1 set (bypass concept) overlap but are not identical -- all enforce_always rules are Tier 1, but Tier 1 is the broader set.

### Open Questions Remaining

- Should hook scripts be deployed via NixOS home-manager (Nix store symlinks, inherently read-only) or with `chattr +i` on conventional deployments? Implementation detail for the deployment phase.
- Should gdev implement a "new repository" CLAUDE.md trust warning? Out of scope for self-protection; belongs in a separate feature.

### Implementation Path

- **Phase 33** (P0): Core Tier 1 rules + fail-closed harness + path canonicalization -- closes the critical settings.json/hook-script protection gap
- **Phase 34** (P1): Tier 2 interactive bypass + monitor mode + audit integration -- enables real-world calibration and configuration protection
- **Phase 35** (P2): Go binary consolidation + tree-sitter AST + hardening -- production maturity
