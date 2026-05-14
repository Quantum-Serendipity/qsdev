# Research Summary: Claude Code Hooks in Practice

## Overview
Survey how teams and individuals actually use Claude Code hooks, skills, and CLAUDE.md patterns for quality enforcement in real-world workflows. Collect community-shared configurations, evaluate their effectiveness, and identify consulting-applicable patterns. The theoretical basis exists in `agentic-workflow-state-of-art` (21 hook events, 4 handler types, Stop hooks as quality gates, deterministic enforcement > advisory instructions). This spike provides the empirical evidence: what configurations do people actually run, what do they enforce, what breaks, and what patterns emerge for consulting-firm use cases (client code review requirements, security scanning, test enforcement)?

## Topics

### Community Hook Configurations (Complete)
- **Status**: Complete
- **Report**: `community-hooks-research.md`
- **Summary**: PostToolUse auto-formatting is the most common hook pattern. `command` handler type dominates. Most configs are individual-level. Two framework ecosystems emerging (TypeScript, Python/UV). Observability tools range from local JSON to OTEL/Grafana.

### CLAUDE.md Patterns in the Wild (Complete)
- **Status**: Complete
- **Report**: `claudemd-patterns-research.md`
- **Summary**: Effective CLAUDE.md files are short (50-100 lines), concrete, and focused on what Claude cannot infer from code. The instruction budget constraint (~150-200 instructions total, ~50 consumed by system prompt) is the fundamental design force. Progressive disclosure pattern emerges: CLAUDE.md (universal) → .claude/rules/ (path-scoped) → skills (on-demand) → hooks (deterministic enforcement). Multiple bug reports document real compliance failures — CLAUDE.md is advisory, hooks are law.

### Consulting-Specific Hook Prototypes (Complete)
- **Status**: Complete
- **Report**: `consulting-hooks-research.md`
- **Summary**: Six complete hook configurations designed for consulting-firm use cases, each with full JSON config, supporting scripts, failure mode analysis, and deployment-level recommendation. Covers: (1) test enforcement via Stop hook, (2) credential/secret scanning via PreToolUse with 12+ regex patterns, (3) destructive command prevention with consulting-specific cross-environment protections, (4) cost alerting via PostToolUse+Stop combination (novel design, fills community gap), (5) SOC 2 session logging via metadata-only audit trail across 4 events, (6) client isolation verification via SessionStart. Three-tier deployment strategy: managed policy for security-critical hooks, user-level for cost/context awareness, project-level for test enforcement. Implementation priority: destructive commands first (trivial, proven), cost alerting last (complex, novel).

### Hook Reliability, Performance, and Failure Modes (Complete)
- **Status**: Complete
- **Report**: `hook-reliability-research.md`
- **Summary**: Analysis of 20+ GitHub issues reveals hooks are "rapidly maturing beta" — moderately reliable for `command` handlers in CLI, but with significant caveats. Version regressions broke hooks entirely across v2.0.27-v2.0.31. VS Code extension plugin hooks are fundamentally broken (#18547 OPEN). `prompt`/`agent` handlers have near-zero adoption due to 1-60+ second latency and non-deterministic evaluation. Two distinct exit-code systems (hooks vs. permission matchers) cause silent enforcement failures when confused. PreToolUse blocks trigger non-deterministic model behavior (sometimes stops instead of retrying). SessionStart hooks don't fire for new conversations (#10373, 17+ upvotes). One developer runs 95 hooks without latency issues when each completes <200ms — performance is per-hook, not per-count. 13 releases in 3 weeks of March 2026; new events nearly every release. The API is actively unstable.

### Hooks vs. Alternative Enforcement Mechanisms (Complete)
- **Status**: Complete
- **Report**: `hooks-vs-alternatives-research.md`
- **Summary**: Claude Code hooks are the only enforcement mechanism that operates during AI code generation (tool-use-time), providing in-flight correction that no other layer offers. Pre-commit hooks cover all code but are trivially bypassed (--no-verify); CI/CD is authoritative but slow; IDE tooling doesn't reach terminal-mode Claude; code review handles semantic quality hooks cannot check. Defense-in-depth requires all five layers. For consulting firms, hooks via managed policy travel with the consultant (not the client repo), providing a consistent quality floor across engagements regardless of client infrastructure. A 4-step decision framework maps quality requirements to enforcement points: classify requirement, map to enforcement point(s), apply consulting multipliers, execute decision matrix.

## Open Questions

1. Quantitative compliance benchmarks at different CLAUDE.md lengths are absent — the ~150-200 limit is cited but not empirically published.
2. How well do path-scoped rules in `.claude/rules/` activate in practice? No community reports yet.
3. What's the enterprise adoption rate for managed policy CLAUDE.md?
4. Does instruction ordering in CLAUDE.md show primacy bias (earlier instructions followed more reliably)?
5. No established pattern for CI failures to automatically update hook configurations — automated hook generation from CI failure patterns would close this gap.
6. Cross-tool governance diverges when teams use both Claude Code and Cursor — no unified governance layer exists above individual tools (Codacy/Cycode via MCP are closest).
7. No published quantitative data on reduction in CI failure rates or review cycle times when Claude Code hooks are deployed.

## Conclusions

### The Enforcement Spectrum Is Real
The CLAUDE.md + hooks ecosystem forms a complete enforcement spectrum: CLAUDE.md provides guidance (probabilistic, context-dependent), while hooks provide enforcement (deterministic, zero-exception). The progressive disclosure architecture (CLAUDE.md → rules → skills → hooks) maps instruction budget to actual need, and the managed policy layer enables org-wide deployment without relying on individual developer compliance.

### The Consulting Rule
Anything that would cause client escalation if missed belongs in a hook, not CLAUDE.md. The three-tier deployment strategy (managed policy for security, user-level for cost awareness, project-level for test enforcement) matches how consulting firms operate: security hooks travel with the consultant, test hooks travel with the client repo.

### Hooks Are the Only In-Flight Enforcement
No other mechanism (pre-commit, CI/CD, IDE, code review) operates during AI code generation. Hooks provide in-flight correction at tool-use-time. For consulting firms, hooks via managed policy provide a consistent quality floor across engagements regardless of client infrastructure. Defense-in-depth requires all five layers.

### Maturity Is the Biggest Risk
The hooks system is "rapidly maturing beta" — `command` handlers work reliably if kept fast (<200ms), but version regressions, VS Code incompatibilities, and an actively changing API mean consulting-firm deployment must pin Claude Code versions and regression-test hook configs. `prompt`/`agent` handlers are theoretically powerful but practically unusable due to latency and non-determinism. Stick to `command` handlers for production.

### Community Adoption Gap = Presentation Opportunity
The community overwhelmingly uses simple `command` hooks for formatting and safety gates. The six consulting-specific configurations designed in this spike (especially cost alerting and SOC 2 logging) are novel — filling gaps no public configuration addresses. This makes the hooks segment of the CoP talk genuinely additive rather than a repeat of what attendees can find online.

### Depth Checklist
- [x] Underlying mechanisms explained — hook execution model, handler types, exit codes, matcher system, configuration scopes
- [x] Key tradeoffs identified — CLAUDE.md advisory vs hooks deterministic, command vs prompt/agent handlers, per-hook vs per-count performance
- [x] Compared to alternatives — 5 enforcement mechanisms compared, 4-step decision framework, defense-in-depth architecture
- [x] Failure modes described — version regressions, VS Code broken, silent exit-code failures, PreToolUse non-determinism, SessionStart bug
- [x] Concrete examples — 6 complete JSON hook configs with supporting scripts, 15+ community repos analyzed, 50+ source docs saved
- [x] Standalone-readable — sufficient for implementation plan creation without consulting original sources
