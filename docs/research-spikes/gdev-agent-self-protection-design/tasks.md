# Tasks: gdev Agent Self-Protection Design

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Threat model: agent self-disabling vectors** — Enumerate all paths an agent could use to disable gdev's six defense layers.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [threat-model-research.md](threat-model-research.md)
  - Key findings: 12 attack vector categories. Phase 32 defends 2 fully, 3 partially, 7 not at all. 3 P0 gaps. 5 minimum rule sets (A-E). 5 real-world incidents analyzed.
- [x] **Prempti self-protection rule patterns** — Extract concrete rule definitions from Prempti and translate to gdev's hook architecture.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [prempti-patterns-research.md](prempti-patterns-research.md)
  - Key findings: 6 Prempti rules (5 deny, 1 ask) translated to 2 consolidated gdev hook scripts. 6 additional gdev-specific rules. Declarative rule format designed. MCP poisoning detection documented.
- [x] **Fail-closed vs fail-open policy** — Research tradeoffs and recommend a default failure mode.
  - Priority: high
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [fail-policy-research.md](fail-policy-research.md)
  - Key finding: Claude Code hooks are inherently fail-open. Severity-tiered failure policy recommended.
- [x] **Three-outcome verdict model design** — Design the allow/deny/ask/warn verdict system for gdev hooks.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [verdict-model-research.md](verdict-model-research.md)
  - Key findings: Four-verdict model. Critical bug #39344: ask overrides deny. Deny-overrides combining. 23 rules assigned verdicts (13 deny, 10 ask). Three-tier precedence.
- [x] **Canonical path resolution strategy** — Path canonicalization to prevent symlink/traversal bypasses.
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [canonical-path-research.md](canonical-path-research.md)
  - Key findings: 9 bypass techniques cataloged. Two-tier canonicalization pipeline. Bash write targets undecidable — three-strategy defense designed.
- [x] **Monitor/shadow mode design** — Logging-only enforcement mode for rule calibration.
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [monitor-mode-research.md](monitor-mode-research.md)
  - Key findings: 6 security systems surveyed. Per-rule mode control with `enforce_always` for nuclear rules. 5-day calibration period. Unified JSONL log.
- [x] **Escape hatch and bypass mechanism** — Legitimate override mechanisms with mandatory audit logging.
  - Priority: medium
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [escape-hatch-research.md](escape-hatch-research.md)
  - Key findings: Three-tier bypass policy. Chained protection pattern. Bugs #39344/#52822 make ask unreliable — exit code 2 required. 14-field JSONL audit with hash chain.

## Phase 2: Synthesis & Review

### Pending

### Active

### Completed
- [x] **Cross-report synthesis** — Synthesize all 7 research reports into coherent conclusions in research.md. Resolve contradictions, consolidate open questions, draft architectural recommendations.
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Report: [synthesis-research.md](synthesis-research.md)
  - Key findings: 4 contradictions identified and resolved. 32-rule consolidated catalog. 3 implementation phases recommended (33-35). 7 of 9 open questions resolved.
- [x] **Depth checklist review** — Run depth checklist against each report. Identify and fill gaps.
  - Priority: high
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Key findings: All 7 reports pass all 6 depth checklist items. No gaps requiring additional research.

### Completed
