# reasoning-core HARDENING.md

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/docs/HARDENING.md
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Six Bypass Paths & Mitigations

The threat model identifies six potential vectors for circumventing edit guards:

1. **Bash-based file writes** — Commands like `cat`, `tee`, `sed -i`, or Python file operations. Blocked by L1 (PreToolUse on Bash).
2. **Sidecar process termination** — Using `pkill` to disable the guard, then editing freely. Addressed through hard-deny rules plus sidecar revival.
3. **Settings manipulation** — Removing the matcher from `.claude/settings.json`. Protected by guard-file locking in L2.
4. **Hook script modification** — Replacing the guard script with a no-op. Same lock mechanism as path #3.
5. **Sidecar failure scenarios** — Unrelated crashes or OOM conditions causing fail-open behavior. Mitigated by revival logic and `S2_FAIL_CLOSED=1` configuration.
6. **Task subagent exploitation** — Spawning a subagent with prompts targeting guarded paths, since subagents don't always re-enter parent hooks. Screened by L4 regex checks on mutation verbs and protected paths.

## Core Defense Layers

**L1 (Bash Guard)** uses regex-based pattern matching (~250 lines) with four sequential checks: hard-deny patterns (process kills, settings edits), guarded-path write operations, sidecar PID kills, and generic source-file writes across tracked extensions.

**L2 (Guard-File Lock)** prevents modification of critical infrastructure files including hooks, settings, and core modules by checking the target path against an allow-list before scoring edits.

**L3 (Sidecar Revival)** runs as a PostToolUse hook, detecting kill-like commands and respawning the sidecar if the health check fails. This operates defensively in-depth.

**L4 (Subagent Guard)** applies regex screening to Task tool prompts, looking for mutation verbs paired with references to guarded paths — currently regex-only rather than full semantic scoring.

## Runtime Configuration

The recommended environment includes `S2_FAIL_CLOSED=1` (deny when sidecar is down), `S2_TIMEOUT=60` (accounting for CPU-intensive operations), and `RC_PLAN_BLOCK=1` (escalating plan warnings to blocks). Override toggles like `RC_ALLOW_GUARD_EDIT=1` enable intentional modifications to guard infrastructure but require Claude to be restarted.

## Known Limitations

The document acknowledges five residual gaps: indirect codegen rewrites bypassing regex detection, missing Vue grammar wheels for Python 3.13, absence of plan-code coherence verification, untuned drift calibration thresholds, and partial (regex-only) subagent scoring pending full semantic analysis in phase-3.
