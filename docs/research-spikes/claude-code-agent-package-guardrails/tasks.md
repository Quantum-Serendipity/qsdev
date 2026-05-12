# Tasks: Claude Code Agent Package Guardrails

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed

- [x] **Claude Code hooks mechanism deep dive** — PreToolUse hooks as primary enforcement mechanism.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `hooks-research.md`. Deterministic, pre-execution, can't be bypassed. 8 bypass vectors with mitigations. attach-guard plugin.

- [x] **Permission settings & command allowlists** — Deny-first evaluation, glob patterns, 5-tier settings hierarchy.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `permissions-research.md`. Two historical bypasses documented. Deny rules alone insufficient.

- [x] **MCP server integration for package security** — Socket.dev and Snyk existing servers. Custom server feasible.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `mcp-server-research.md`. MCP alone insufficient — must combine with hooks + permissions.

- [x] **CLAUDE.md instruction-based guardrails** — Advisory layer, ~90%+ compliance, no subagent inheritance.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `claude-md-guardrails-research.md`. Subagent CLAUDE.md regression #40459.

- [x] **Custom skills for secure package installation** — Workflow orchestration layer, not enforcement.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `custom-skills-research.md`. Security Phoenix is best real-world example.

- [x] **Vulnerability database & provenance APIs** — OSV.dev primary (120ms, free), Socket.dev supplementary, Nix gap.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `vulnerability-apis-research.md`. 12 APIs surveyed with comparison table.

- [x] **Cross-reference sibling spikes** — 7 defense categories extracted, mapped to 5 enforcement layers.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `sibling-spike-cross-reference.md`. Age gates, install script sandboxing, lockfile enforcement, Nix protections.

## Phase 2: Synthesis & Architecture

### Pending

### Active

### Completed

- [x] **Unified guardrail architecture specification** — Synthesized all 7 research reports into `unified-architecture.md`. Covers: 5-layer architecture diagram, complete hook script with OSV.dev + age checking, comprehensive deny rules for all package managers, OS/environment configs (.npmrc, pip.conf, nix.conf, Cargo), CLAUDE.md section, /safe-install skill spec, MCP server setup (Socket.dev + Snyk + custom), 3 deployment profiles (individual/team/enterprise), 8 bypass vectors with residual risk assessment.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Report at `unified-architecture.md`. 1,727 lines. Subsumes the reference implementation, deny rules, bypass assessment, and deployment guide tasks — all are sections of the unified document.

- [x] **Reference implementation: PreToolUse hook script** — Included in unified-architecture.md Section 2.3. Complete bash/jq script with install pattern detection, package extraction, OSV.dev CVE querying, npm registry age checking, safety flag rewriting via updatedInput, and structured JSON decision output.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Subsumed into unified architecture.

- [x] **Reference implementation: Permission deny rules** — Included in unified-architecture.md Section 3.1. 48 deny rules covering npm, yarn, pnpm, bun, pip, pip3, uv, poetry, cargo, go, gem, bundle, composer, dotnet, nix-env, nix profile, cachix, apt, apt-get, brew, pacman, plus pipe-to-shell patterns.
  - Outcome: success | Completed: 2026-05-12
  - Notes: Subsumed into unified architecture.

- [x] **Bypass resistance assessment** — Included in unified-architecture.md Section 9.1. All 8 bypass vectors assessed with severity, current mitigation, and residual risk. Additional coverage of historical enforcement bugs (Section 9.6).
  - Outcome: success | Completed: 2026-05-12
  - Notes: Subsumed into unified architecture.

- [x] **Deployment guide** — Included in unified-architecture.md Section 8. Three profiles: individual developer (15-30 min, 3 files), team/project (1-2 hrs, 5+ files with git-committed settings), enterprise (4-8 hrs, managed settings via NixOS module).
  - Outcome: success | Completed: 2026-05-12
  - Notes: Subsumed into unified architecture.
