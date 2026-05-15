# Cross-Tool Comparison & gdev Integration Mapping

## Executive Summary

None of the five evaluated tools are suitable as gdev defaults — all are too immature, heavyweight, or specialized. Two are recommended as optional configuration options (Prempti for runtime policy enforcement, Sense for codebase navigation MCP). One article (Cloudberry) provides a replicable architecture for an opt-in deep security review feature. The remaining two (npm-scan, reasoning-core) are concept-source-only. Across all five, 23 borrowable design patterns were identified that can strengthen gdev's existing architecture without adding external dependencies.

---

## 1. Side-by-Side Comparison

| Dimension | Prempti | npm-scan | reasoning-core | Cloudberry | Sense |
|-----------|---------|----------|---------------|------------|-------|
| **Category** | AI agent policy layer | npm supply chain scanner | Code edit risk scoring | AI security review pipeline | Codebase understanding MCP |
| **Language** | Rust | Node.js | Python | Article (no tool) | Go |
| **Age** | 2 months | 6 days | 15 days | Article (2026-05-14) | ~1 year (688 commits) |
| **Stars** | 42 | 4 | 5 | N/A | 4 |
| **Contributors** | 2 | 1 | 1 | N/A | 1 |
| **License** | Apache-2.0 | Apache-2.0 + Commons Clause | MIT | N/A | MIT + SaaS restriction |
| **Dep weight** | Heavy (Falco binary + plugin + supervisor) | Moderate (Node.js + sql.js WASM) | Heavy (~2.5GB Python/PyTorch) | N/A (methodology) | Moderate (60MB Go binary, 100-200MB index) |
| **Security-relevant** | Yes (direct) | Yes (direct) | Indirect (code quality) | Yes (direct) | No (navigation efficiency) |
| **gdev overlap** | 70% with existing 6-layer defense | Partial (OSV Scanner, Socket CLI cover same slot) | Partial (hook architecture only) | Complementary (deep review is new capability) | Complementary (replaces semble) |

## 2. Integration Recommendations

### Tier 1: Recommended as Optional Configuration + Concept Source

#### Prempti → `gdev enable prempti`
- **What it adds over existing gdev defenses**: Audit trail of all agent actions, ask verdict for gray-area operations, MCP config poisoning detection, self-protection rules (block agent from disabling security tools), monitor mode for rule development
- **gdev phases affected**: Phase 6-8 (Claude Code addon hook rules), Phase 32 (advanced security hardening)
- **Integration path**: Add ecosystem module that downloads prempti release binary, generates Falco config from gdev's security profile, registers PreToolUse hook. Requires Falco as a runtime dependency.
- **Maintenance burden**: Medium — Prempti is under active development by the Falco/Sysdig team (CNCF graduated project backing), but API stability is unproven at v0.3.0
- **Risk**: Fail-closed design means Falco daemon crash blocks all Claude Code tool execution. No Nix package exists yet.

#### Sense → `gdev enable sense` (Phase 28 MCP Registry)
- **What it adds**: Symbol graph navigation, blast radius analysis, convention detection, semantic search — all via MCP tools that reduce Claude Code token usage by ~32%
- **gdev phases affected**: Phase 28 (MCP server registry), Unit 11.3 (supersedes semble)
- **Integration path**: Add MCP registry entry with detect-and-offer policy. Download Go binary, configure `.mcp.json`, register lifecycle hooks for incremental indexing.
- **Maintenance burden**: Low-medium — single author concern, but Go binary is self-contained with no external dependencies
- **Risk**: O'Saasy license restricts SaaS usage (not relevant for gdev's local-only model). Single-author bus factor. 60MB binary + 100-200MB per-project index may concern disk-constrained environments.

### Tier 2: Recommended as Concept Source Only (No Direct Integration)

#### Cloudberry Article → Concept borrows for gdev security review features
- **Borrowable patterns** (3):
  1. **Separated security context directory** (`.security/architecture.md`, `.security/attack-surface.md`) — distinct from codegen context, prevents trust/distrust confusion
  2. **Anti-false-positive patterns** in security rules — document known-safe patterns per project to reduce noise
  3. **Review benchmarking protocol** — track precision/recall on known-vulnerability test sets to measure security review quality over time
- **gdev phases affected**: Phase 14 (Claude Code skills — security reviewer agent), existing `.claude/` directory structure
- **Not worth direct integration because**: The full 6-phase pipeline requires Semgrep as a hard dependency, costs $3.88/review, takes 9.4 min per run, and violates gdev's zero-prerequisites principle

#### reasoning-core → Concept borrows for hook architecture
- **Borrowable patterns** (5):
  1. **Subagent guard pattern** — Task hook (L4) that screens mutation verbs in subagent-spawned operations
  2. **6-vector bypass threat model** — Enumerated attack surface for agent self-disabling guardrails (prompt injection, tool aliasing, permission escalation, config tampering, context overflow, social engineering)
  3. **Shadow-mode calibration** — Run hooks in monitor-only mode during initial rollout, collect metrics before enforcing
  4. **Escape hatch design** — Magic comments (`# reasoning-core: skip`) and CLI bypass (`--no-guard`) for legitimate override, logged for audit
  5. **JSONL audit log schema** — Structured decision metadata (timestamp, file, risk_vector, decision, override_reason)
- **gdev phases affected**: Phase 6-8 (hook architecture), Phase 32 (advanced security)
- **Not worth direct integration because**: 15-day-old project, no validated evaluation data, ~2.5GB Python/PyTorch dependency stack, supports only 5/27 gdev ecosystems

#### npm-scan → Concept borrows for policy engine
- **Borrowable patterns** (4):
  1. **ATK taxonomy with NIST 800-161 mappings** — Structured attack-type classification with standards cross-references
  2. **Policy-as-code YAML format** — Context-aware suppressions with unsuppressible safety guards for lifecycle hooks
  3. **Lockfile-triggered pre-commit scanning** — Scan only when dependency manifests change, not on every commit
  4. **SARIF v2.1 GitHub Security tab integration** — Standard output format for findings that renders natively in GitHub
- **gdev phases affected**: Phase 15 (health/compliance reporting — SARIF output), existing pre-commit hook generation
- **Not worth direct integration because**: 6-day-old project, most detectors are single-regex pattern matchers (not the AST-level analysis claimed), Socket CLI covers the same functional slot with genuine depth (70+ risk types, funded team, massive adoption)

## 3. Mapping to gdev-secure-devenv-bootstrap Phases

| gdev Phase | Tool | Integration Type | Specific Contribution |
|-----------|------|-----------------|----------------------|
| Phase 6-8 (Claude Code addon) | Prempti | Config option + concept borrow | Self-protection rules, ask verdict, MCP poisoning detection |
| Phase 6-8 (Claude Code addon) | reasoning-core | Concept borrow | Subagent guard pattern, bypass threat model, escape hatch design |
| Phase 14 (Claude Code skills) | Cloudberry | Concept borrow | `.security/` context directory, anti-false-positive patterns |
| Phase 15 (Health/compliance) | npm-scan | Concept borrow | SARIF v2.1 output format, ATK taxonomy structure |
| Phase 28 (MCP server registry) | Sense | Config option | Replaces semble (Unit 11.3), better capabilities, Go-native |
| Phase 32 (Advanced security) | Prempti | Config option | Full Falco-backed policy enforcement, audit trail |
| Phase 32 (Advanced security) | reasoning-core | Concept borrow | Shadow-mode calibration, JSONL audit log schema |

## 4. Consolidated Borrowable Patterns (23 total)

### Security Architecture Patterns
1. **Self-protection rules** (Prempti) — Block agent from disabling its own security tools
2. **MCP config poisoning detection** (Prempti) — Detect malicious MCP server configurations injected via prompt
3. **Ask verdict for gray-area operations** (Prempti) — Three-outcome model (allow/deny/ask) instead of binary allow/deny
4. **Canonical path resolution** (Prempti) — Resolve symlinks and relative paths before rule evaluation to prevent traversal bypass
5. **6-vector bypass threat model** (reasoning-core) — Comprehensive attack surface enumeration for agent guardrails
6. **Separated security context** (Cloudberry) — `.security/` directory with architecture docs distinct from codegen context

### Hook & Enforcement Patterns
7. **Monitor mode** (Prempti) — Log-only mode for rule development before enforcement
8. **Shadow-mode calibration** (reasoning-core) — Collect metrics in non-blocking mode before enabling enforcement
9. **Subagent guard pattern** (reasoning-core) — Task-hook screening of mutation verbs in subagent-spawned operations
10. **Escape hatch design** (reasoning-core) — Magic comments + CLI bypass with mandatory audit logging
11. **LLM-friendly output** (Prempti) — Blocked-action responses formatted so the agent can adapt its approach
12. **Silent hook failure** (Sense) — Hooks fail silently rather than blocking agent operation (appropriate for non-security hooks)
13. **Detect-and-nudge vs hard-block** (Sense) — Tiered intervention: suggest first, enforce only for critical violations

### Data & Reporting Patterns
14. **JSONL audit log schema** (reasoning-core) — Structured decision metadata (timestamp, file, risk_vector, decision)
15. **ATK taxonomy with NIST mappings** (npm-scan) — Attack-type classification with standards cross-references
16. **SARIF v2.1 output** (npm-scan) — Standard findings format for GitHub Security tab integration
17. **Anti-false-positive documentation** (Cloudberry) — Per-project known-safe patterns that reduce noise
18. **Review benchmarking protocol** (Cloudberry) — Track precision/recall on known-vulnerability test sets

### Developer Experience Patterns
19. **Policy-as-code YAML** (npm-scan) — Context-aware suppressions with unsuppressible safety guards
20. **Lockfile-triggered scanning** (npm-scan) — Scan only when dependency manifests change
21. **Post-tool-use incremental re-indexing** (Sense) — Re-index only changed files after agent writes
22. **Pre-compact context injection** (Sense) — Inject critical context before Claude Code compacts conversation
23. **Idempotent setup with deep-merge** (Sense) — Configuration that can be re-applied without destroying user changes

## 5. Key Themes Across All Five Tools

### Theme 1: The AI Agent Security Space Is Extremely Young
Every tool evaluated (except the Cloudberry article) is less than 3 months old. Prempti (42 stars) is the most mature. This means:
- No tool is production-hardened enough to be a gdev default
- The design patterns are more valuable than the implementations
- The space is moving fast — revisit in 6 months

### Theme 2: Hook Architecture Is the Critical Integration Point
All three tools that integrate with Claude Code (Prempti, reasoning-core, Sense) use the same mechanism: PreToolUse/PostToolUse hooks. This validates gdev's existing hook-centric architecture. The variation is in verdict models (binary vs ternary), failure modes (fail-closed vs fail-open), and scope (security-only vs multi-purpose).

### Theme 3: Concept Borrowing Beats Direct Integration
For all five tools, the borrowable patterns are more valuable to gdev than the tool themselves. gdev's design principles (single binary, zero prerequisites, curate don't reinvent) filter out every evaluated tool as a default. But the patterns — self-protection rules, ask verdicts, shadow-mode calibration, separated security context — can be implemented natively in gdev's Go codebase with no external dependencies.

### Theme 4: MCP Is the New Extension Point
Sense demonstrates that MCP servers are becoming the standard way to extend AI agent capabilities. gdev's Phase 28 MCP server registry is well-positioned. The key insight from Sense: MCP tools should be detect-and-offer (suggest to user when relevant), not force-installed.

### Theme 5: None of These Replace Socket CLI or OSV Scanner
For the specific problem of dependency vulnerability scanning (gdev's existing Layer 4), none of the evaluated tools compete with Socket CLI (supply chain) or OSV Scanner (known CVEs). npm-scan explicitly tries to compete but falls far short. The evaluated tools operate at different layers: agent behavior (Prempti), code quality (reasoning-core), review depth (Cloudberry), and navigation (Sense).

## 6. Depth Checklist

- [x] Underlying mechanisms explained — each tool's architecture, pipeline, and integration points detailed in individual reports
- [x] Key tradeoffs and limitations identified — maturity, dependency weight, overlap with existing gdev defenses, license restrictions
- [x] Compared to alternatives — each tool compared to established alternatives (Socket CLI vs npm-scan, gdev hooks vs Prempti, semble vs Sense)
- [x] Failure modes and edge cases described — fail-closed risks (Prempti), false positives (npm-scan ATK-009/011), unvalidated claims (reasoning-core eval)
- [x] Concrete examples found — specific rules, patterns, CLI commands, and integration configurations documented
- [x] Report is standalone-readable — sufficient for making integration decisions without consulting individual tool reports
