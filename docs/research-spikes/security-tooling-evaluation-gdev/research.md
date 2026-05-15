# Research Summary: Security Tooling Evaluation for gdev

## Overview

Deep dive analysis of five security and code analysis tools from first principles, evaluating each for integration into the gdev-secure-devenv-bootstrap implementation plan:

1. **[Prempti](https://github.com/falcosecurity/prempti)** — Falco Security's preemptive security tool
2. **[npm-scan](https://github.com/lateos-ai/npm-scan)** — Lateos AI's npm dependency scanner
3. **[reasoning-core](https://github.com/jakubkrzysztofsikora/reasoning-core)** — Reasoning engine for code analysis
4. **[Automating Code Security Reviews](https://cloudberry.engineering/article/automating-code-security-reviews/)** — Cloudberry Engineering's approach to automated security reviews
5. **[Sense](https://github.com/luuuc/sense)** — Structural codebase understanding MCP server (not a security tool)

For each tool: understand how it works at a fundamental level, evaluate whether to include it as a gdev configuration, use it as a default, or borrow concepts/implementation. Focus on developer experience, utility, and capability improvements.

## Topics

### Prempti (falcosecurity/prempti)
- **Status**: Complete
- **Report**: [`prempti-research.md`](prempti-research.md)
- **Summary**: Falco-powered policy and visibility layer for AI coding agents (Rust, Apache-2.0, v0.3.0, 42 stars, created 2026-03-18). Intercepts every Claude Code tool call via PreToolUse hook, evaluates against Falco's rule engine through a Rust interceptor + Unix socket IPC + embedded broker plugin, and returns allow/deny/ask verdicts in real time. Ships 58 default rules + 79 macros covering 7 security domains (working directory boundary, sensitive paths, sandbox disable, threats, MCP/skill content, persistence vectors, self-protection). Runs as a user-space daemon (Falco nodriver + plugin + supervisor) with fail-closed design. Cross-platform (Linux/macOS/Windows) but requires Falco source compilation on macOS/Windows and has no Nix package. Recommended as OPTIONAL configuration option (`gdev enable prempti`) -- too heavy and immature for default but adds audit trail, ask verdicts, monitor mode, and MCP/self-protection rules gdev currently lacks. HIGHLY recommended as concept/implementation source: self-protection rules (block agent from disabling security tools), MCP config poisoning detection, ask verdict pattern for gray-area operations, canonical path resolution for bypass prevention, monitor mode for rule development, and LLM-friendly output conventions. Not recommended as default due to infrastructure weight, 70% rule overlap with gdev's existing 6-layer defenses, fail-closed service crash risk, and experimental status.

### Cloudberry Engineering: Automating Code Security Reviews
- **Status**: Complete
- **Report**: [`cloudberry-security-reviews-research.md`](cloudberry-security-reviews-research.md)
- **Summary**: Six-phase AI security review pipeline (Prep/Map/Hunt/Dedup/Validate/Aggregate) using Semgrep for deterministic attack surface mapping and right-sized models per phase. $3.88/review average, 60% finding discard rate through dedup+validation. Recommended for gdev as: (1) opt-in configuration option for deep security review, (2) concept borrows for separated security context directory, anti-false-positive patterns in security rules, and review benchmarking protocol. Not suitable as default due to cost, latency, and Semgrep dependency.

### reasoning-core (jakubkrzysztofsikora/reasoning-core)
- **Status**: Complete
- **Report**: [`reasoning-core-research.md`](reasoning-core-research.md)
- **Summary**: Local Python sidecar using a 130M-parameter Mamba SSM model + Tree-sitter AST parsing to score code edits for structural regression before LLM CLIs execute them. Integrates via Claude Code PreToolUse hooks (9 layers) and MCP server. Only 15 days old, single contributor, 5 stars, no validated evaluation data (published eval ran stub-mode only). Requires ~2.5GB of Python/PyTorch/model dependencies and adds 5s latency per edit on CPU. Not recommended as configuration option or default for gdev (violates single-binary/zero-prereqs principle, unproven effectiveness, immature). Selectively recommended as concept source: subagent guard pattern (Task hook screening), 6-vector bypass threat model, shadow-mode calibration for hook rollout, escape hatch design (magic comments + CLI bypass), and JSONL audit log schema with decision metadata.

### npm-scan (lateos-ai/npm-scan)
- **Status**: Complete
- **Report**: [`npm-scan-research.md`](npm-scan-research.md)
- **Summary**: Young (6-day-old), solo-maintained npm supply chain scanner claiming AST-level heuristic and behavioral analysis, but source code review reveals most detectors (9 of 11) are single-regex pattern matchers on concatenated file contents -- a significant gap between marketing and implementation. Built from v0.1.0 to v0.9.7 in 4 days (May 9-12, 2026); 4 GitHub stars, 0 forks, ~4,253 monthly npm downloads. Apache-2.0 + Commons Clause license. ATK-002 (obfuscation detector) is genuinely sophisticated with multi-layer detection, context awareness, and decoded previews. The policy-as-code engine is well-designed (context-aware suppressions, unsuppressible safety guards for lifecycle hooks). However, high false-positive risk (ATK-009 flags process.env.CI, ATK-011 flags eslint-plugin-*), no actual AST parsing despite acorn dependency, and no runtime sandbox despite "behavioral" claims. Socket CLI is the established market leader for the same functional slot (70+ risk types, funded team, massive adoption). Not recommended as gdev default or configuration option. Selectively recommended as concept source for 4 patterns: ATK taxonomy structure with NIST 800-161 mappings, policy-as-code YAML format with unsuppressible safety guards, lockfile-triggered pre-commit scanning workflow, and SARIF v2.1 GitHub Security tab integration.

### Sense (luuuc/sense)
- **Status**: Complete
- **Report**: [`sense-research.md`](sense-research.md)
- **Summary**: Go-native MCP server (688 commits, v0.84.3) providing structural codebase understanding to AI agents through 4 tools: symbol graph navigation (`sense_graph`), hybrid semantic+keyword search (`sense_search`), blast radius analysis with BFS confidence decay (`sense_blast`), and convention detection across 9 categories (`sense_conventions`). Uses tree-sitter for 13-language parsing, bundled ONNX for embeddings, SQLite for persistence, and 5 Claude Code lifecycle hooks for deep agent integration. Despite its inclusion in this security tooling evaluation, Sense is NOT a security tool — it solves AI agent navigation efficiency (-47% tool calls, -32% tokens per task). Maturity concerns: 4 stars, single author, O'Saasy license (MIT + SaaS restriction). Recommended as detect-and-offer configuration option in Phase 28 MCP registry — functionally supersedes semble (Unit 11.3) with richer capabilities and zero Python dependency. Also recommended as concept source for silent hook failure pattern, post-tool-use incremental re-indexing, pre-compact context injection, and detect-and-nudge vs hard-block intervention tiers.

### Cross-Tool Comparison & Integration Mapping
- **Status**: Complete
- **Report**: [`cross-tool-comparison-research.md`](cross-tool-comparison-research.md)
- **Summary**: Side-by-side evaluation of all five tools against gdev's architecture, design principles, and phase plan. None recommended as defaults (all too immature/heavy/specialized). Two recommended as optional configuration options (Prempti for runtime policy, Sense for codebase MCP). Twenty-three borrowable design patterns consolidated across 4 categories: security architecture (6), hook & enforcement (7), data & reporting (5), developer experience (5).

## Open Questions

- (Resolved) How does each tool's architecture map to gdev's plugin/configuration model? → See cross-tool comparison Section 3
- (Resolved) Which tools complement vs. overlap with existing gdev security phases? → See cross-tool comparison Section 3
- (Resolved) What's the maintenance burden of each integration path? → See cross-tool comparison Section 2

## Conclusions

**No tool should be a gdev default.** All five are too immature, heavyweight, or specialized to meet gdev's design principles (single binary, zero prerequisites, security by default).

**Two tools warrant optional configuration options:**
1. **Prempti** (`gdev enable prempti`) — Adds audit trail, ask verdicts, MCP poisoning detection, and self-protection rules that gdev currently lacks. Requires Falco as runtime dependency. Medium maintenance burden.
2. **Sense** (`gdev enable sense`) — Replaces semble (Unit 11.3) in Phase 28 MCP registry with richer capabilities and zero Python dependency. Reduces Claude Code token usage by ~32%. Low-medium maintenance burden.

**All five are valuable concept sources** yielding 23 borrowable patterns that can be implemented natively in gdev's Go codebase. The highest-impact patterns are:
- Self-protection rules and MCP poisoning detection (from Prempti)
- Separated security context directory (from Cloudberry)
- Shadow-mode calibration and escape hatch design (from reasoning-core)
- SARIF v2.1 output format (from npm-scan)
- Post-tool-use incremental re-indexing and detect-and-nudge intervention tiers (from Sense)

**The AI agent security tooling space is extremely young** (all tools <3 months old). The design patterns are more valuable than the implementations. Revisit in 6 months for maturity reassessment.
