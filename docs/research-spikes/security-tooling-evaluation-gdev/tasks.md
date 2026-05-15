# Tasks: Security Tooling Evaluation for gdev

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Define research question and tasks** — Confirm scope and create Phase 1 task breakdown
  - Outcome: success
  - Completed: 2026-05-15
  - Notes: Research question confirmed. Six investigation tasks created for Phase 2.

## Phase 2: Research & Investigation

### Pending

### Active

### Completed
- [x] **Cross-tool comparison & gdev integration mapping** — Compare all five tools side-by-side, map capabilities to gdev-secure-devenv-bootstrap phases, recommend integration strategy for each
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: None recommended as default. 2 recommended as optional configs (Prempti, Sense). All 5 yield concept borrows (23 total patterns). Report at `cross-tool-comparison-research.md`.
- [x] **Deep dive: npm-scan (lateos-ai/npm-scan)** — Fetch repo, understand architecture/mechanisms, evaluate integration fit for gdev
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: Young (6-day-old) solo-maintained npm supply chain scanner with ambitious feature surface but shallow detection depth. Most detectors are single-regex pattern matchers on concatenated source — NOT the AST-level/behavioral analysis claimed in README. 4 stars, 0 forks, ~4,253 monthly downloads. Built v0.1.0 to v0.9.7 in 4 days. NOT recommended as default or config option. SELECTIVELY recommended as concept source: ATK taxonomy structure with NIST mappings, policy-as-code YAML format (allowlists, context-aware suppressions, unsuppressible safety guards), lockfile-triggered pre-commit scanning pattern, and SARIF v2.1 GitHub integration. Socket CLI is the better recommendation for the same functional slot. Report at `npm-scan-research.md`.
- [x] **Deep dive: Prempti (falcosecurity/prempti)** — Fetch repo, understand architecture/mechanisms, evaluate integration fit for gdev
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: Falco-powered policy and visibility layer for AI coding agents (Rust, Apache-2.0, v0.3.0, 42 stars, 2 months old). Intercepts every Claude Code tool call via PreToolUse hook, evaluates against Falco rule engine, returns allow/deny/ask verdicts. Ships 58 default rules + 79 macros covering 7 security domains. Runs as user-space daemon (Falco nodriver + plugin + supervisor). Recommended as OPTIONAL config option (`gdev enable prempti`) — too heavy/immature for default but adds audit trail, ask verdicts, monitor mode, and MCP/self-protection rules gdev lacks. HIGHLY recommended as concept source for 6 borrowable patterns. Not recommended as default (infrastructure weight, 70% rule overlap, fail-closed risk, no Nix package). Report at `prempti-research.md`.
- [x] **Deep dive: reasoning-core** — Fetch repo, understand architecture/mechanisms, evaluate integration fit for gdev
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: Local Mamba SSM sidecar that scores code edits before LLM CLIs execute them. 15-day-old project, 1 contributor, 5 stars, no validated eval data, heavyweight Python/PyTorch stack. NOT recommended as config option or default (violates single-binary, zero-prereqs, unproven). SELECTIVELY recommended as concept source: subagent guard pattern, hook bypass threat model, shadow-mode calibration, escape hatch design, audit log schema. Report at `reasoning-core-research.md`.
- [x] **Deep dive: Cloudberry automated security reviews** — Fetch article, extract concepts/patterns, evaluate applicability to gdev
  - Priority: high
  - Estimate: small
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: Article describes a 6-phase AI security review pipeline (Prep→Map→Hunt→Dedup→Validate→Aggregate) using right-sized models. Key patterns: separated security context, Semgrep-based attack surface mapping, dedup-before-validate. Recommended as opt-in config option for gdev + 3 concept borrows for existing security infrastructure. Report at `cloudberry-security-reviews-research.md`.
- [x] **Deep dive: Sense (luuuc/sense)** — Fetch repo, understand architecture/mechanisms, evaluate integration fit for gdev
  - Priority: high
  - Estimate: medium
  - Started: 2026-05-15
  - Completed: 2026-05-15
  - Outcome: success
  - Notes: Go-native MCP server for structural codebase understanding (symbol graphs, blast radius, semantic search, convention detection). NOT a security tool — solves AI agent navigation efficiency. 688 commits, v0.84.3, but only 4 stars (single author, O'Saasy license). Recommended as detect-and-offer config option in Phase 28 MCP registry — functionally superior to semble (Unit 11.3) with zero Python dependency. Also recommended as concept source: silent hook failure pattern, post-tool-use incremental re-indexing, pre-compact context injection, detect-and-nudge vs hard-block distinction. Report at `sense-research.md`.
