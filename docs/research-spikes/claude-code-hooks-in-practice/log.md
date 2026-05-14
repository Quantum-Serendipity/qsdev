# Research Log: Claude Code Hooks in Practice

## 2026-03-27 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Originated from GAP 3 in `claude-tools-consulting-adoption/gap-analysis-research.md` — the `agentic-workflow-state-of-art` spike covers hooks exhaustively in theory (21 events, 4 handler types, Stop hooks as quality gates) but there is zero research on how real teams/firms actually use hooks for quality enforcement, CI integration, or compliance. This spike closes that gap with empirical evidence. Feeds into the Claude Code Tools CoP talk's hooks preview segment and strengthens the adoption strategy's capability frontier.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-03-27 09:08 — Community Hook Configurations Survey Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 15+ GitHub repositories, 6 HN discussions, multiple blog posts/tutorials (see `community-hooks-research.md` Sources section for full list)
- **Summary**: Comprehensive survey of real-world Claude Code hook usage. Analyzed 15+ repos with published hook configs, 6 HN threads, and multiple blog posts. Key findings: (1) PostToolUse auto-formatting is the single most common hook, (2) PreToolUse safety gates are second, (3) Stop notifications third, (4) `command` handler type overwhelmingly dominates — `prompt`/`agent` types almost unused in community despite being documented, (5) most configs are individual-level not team-enforced, (6) two framework ecosystems emerging (TypeScript: johnlindquist + timoconnellaus; Python/UV: disler), (7) observability tools range from local JSON logs to real-time Vue dashboards to OTEL/Grafana stacks, (8) context preservation across compaction is an active area with multiple competing tools. Saved 20 source docs and wrote detailed report.
- **Next**: Use findings to inform consulting-specific hook prototypes. Survey CLAUDE.md patterns (parallel task).

## 2026-03-27 10:45 — CLAUDE.md Patterns in the Wild Survey Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: Anthropic official docs (best-practices, memory/CLAUDE.md), 8+ real-world CLAUDE.md files (Anthropic's own claude-code-action, Browser Use monorepo, ArthurClune templates, ChrisWiles showcase, coleam00 context-engineering, shanraisshan best-practice), awesome-claude-code (33k stars), HumanLayer blog, multiple GitHub bug reports, Martin Fowler context engineering article. See `claudemd-patterns-research.md` Sources section for full list. 15 source docs saved to `docs/`.
- **Summary**: Comprehensive survey of CLAUDE.md patterns. Key findings: (1) Effective files are 50-100 lines — Anthropic's own is ~60 lines, (2) The instruction budget constraint is the fundamental design force: LLMs can follow ~150-200 instructions, system prompt takes ~50, leaving ~100-150 for everything else, (3) Every low-value instruction actively degrades compliance with high-value ones, (4) 12+ instruction categories identified (commands, architecture, code style, gotchas, testing, git conventions, forbidden actions, etc.), (5) CLAUDE.md is advisory and probabilistic while hooks are deterministic — "the gap between 'usually' and 'always' is where production systems fail", (6) Progressive disclosure pattern emerges: CLAUDE.md (universal) → rules (path-scoped) → skills (on-demand) → hooks (deterministic), (7) Multiple compliance failure bug reports document real instruction non-adherence patterns, (8) Managed policy CLAUDE.md enables org-wide deployment for consulting firms.
- **Next**: Prototype consulting-specific hooks. Compare hooks to alternatives (CI/CD, pre-commit). Evaluate hook reliability/performance.

## 2026-03-27 12:30 — Consulting-Specific Hook Prototypes Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: Official Claude Code hooks reference (full schema for all 21 events), official cost management docs, mintmcp/agent-security (secrets scanning), SOC 2 + AI coding compliance research, ccusage JSONL analysis tool, managed-settings.json enterprise guide. 3 new source docs saved to `docs/`.
- **Summary**: Designed and documented 6 complete hook configurations for consulting-firm use cases: (1) Test enforcement Stop hook with multi-runner detection and stop_hook_active guard, (2) Credential/secret scanning PreToolUse hook with 12+ regex patterns adapted from detect-secrets, (3) Destructive command prevention PreToolUse hook with consulting-specific cross-environment and production deployment protections, (4) Cost alerting PostToolUse+Stop combination — novel design filling a community gap, reads session JSONL for token counting with configurable soft/hard budget thresholds, (5) Session logging for SOC 2 compliance across SessionStart/PostToolUse/Stop/SessionEnd with metadata-only approach (no content logging), (6) Client isolation verification SessionStart hook with client registry JSON. Each hook includes complete JSON config, supporting script, failure mode analysis, deployment level recommendation, and complexity rating. Also documented the three-tier deployment strategy (managed policy → user-level → project-level) and hook interaction patterns.
- **Next**: Evaluate hook reliability and performance. Compare hooks to alternatives (CI/CD, pre-commit).

## 2026-03-27 14:15 — Hooks vs. Alternatives Comparison Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Sources**: 17 web sources (blog posts, enterprise governance articles, tool comparisons, practitioner guides) + prior spike research (community-hooks-research.md, claudemd-patterns-research.md, compliance failures, instruction budget). Key sources: DEV.to Git Hooks + Claude Code article, Liam ERD Lefthook enforcement, Xygeni/TruffleSecurity pre-commit bypass analysis, Chris Richardson GenAI guardrails, Cycode/Codacy IDE-level security tools, CodeRabbit AI review, AgentRuleGen cross-tool comparison. Full source list saved in `docs/web-search-hooks-vs-alternatives.md`.
- **Summary**: Deep comparison of Claude Code hooks against 5 enforcement alternatives (pre-commit hooks, CI/CD, IDE tooling, code review, CLAUDE.md). Key findings: (1) Hooks are the only mechanism providing in-flight correction during AI code generation — no other layer operates at tool-use-time, (2) "Git hooks protect you from yourself, Claude Code hooks protect you from your AI agent" — different threat models, same codebase, (3) Pre-commit hooks are trivially bypassable (--no-verify), but Claude Code hooks can block the bypass command itself, (4) CI/CD is the authoritative enforcement point but has the slowest feedback loop — hooks serve as pre-flight optimization, (5) IDE tooling doesn't reach Claude Code terminal-mode file writes — hooks fill this gap, (6) Code review and hooks serve complementary functions with almost no overlap (mechanical vs. semantic), (7) CLAUDE.md is for advisory guidance that shapes reasoning; hooks are for deterministic requirements that must always be met. Built a 4-step decision framework for consulting firms: classify requirement → map to enforcement point(s) → apply consulting multipliers → execute decision matrix. Defense-in-depth architecture with 5 layers. 6 worked examples covering common consulting scenarios.
- **Next**: Evaluate hook reliability/performance.

## 2026-03-27 16:00 — Hook Reliability, Performance, and Failure Modes Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 15+ GitHub issues (anthropics/claude-code), official hooks reference (detailed), CHANGELOG.md, CVE-2025-59536/CVE-2026-21852 security reports, community blog posts and tutorials. 15 new source docs saved to `docs/`.
- **Summary**: Comprehensive analysis of hook reliability across 4 handler types, 5 bug categories, and 20+ GitHub issues. Key findings: (1) `command` hooks are moderately reliable in CLI but VS Code extension is fundamentally broken for plugin hooks (#18547, OPEN), (2) Version regressions are the biggest risk — hooks were completely broken and re-broken across v2.0.27-v2.0.31 in a 5-day window, (3) `prompt` and `agent` handler types have near-zero community adoption due to latency, cost, and non-deterministic behavior, (4) Two distinct exit code systems exist for PreToolUse (exit 2 vs JSON permissionDecision with exit 0) and confusing them causes silent enforcement failures, (5) PreToolUse blocks cause non-deterministic Claude behavior — model sometimes stops instead of fixing and retrying, (6) PermissionRequest hooks have a race condition with UI dialogs (#12176), (7) SessionStart hooks don't work for new conversations (#10373, 17+ upvotes), (8) The hooks API is unstable — new events in nearly every release, 13 releases in 3 weeks of March 2026, (9) Security improved significantly (trust dialogs, credential scrubbing, CVE fixes) but permission interaction bugs persist, (10) One developer runs 95 hooks without latency issues if each completes <200ms — performance is per-hook, not per-count.
- **Next**: Update research.md with reliability findings. Mark task complete.

## 2026-03-27 — Spike Synthesis Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: All 5 tasks completed in a single session. Depth checklist passed: mechanisms ✓ (hook execution model, handler types, exit codes, matchers, configuration scopes), tradeoffs ✓ (advisory vs deterministic, command vs prompt/agent, per-hook performance), alternatives ✓ (5 enforcement mechanisms compared with decision framework), failure modes ✓ (version regressions, VS Code broken, silent exit-code failures, PreToolUse non-determinism, SessionStart bug), examples ✓ (6 complete consulting hook configs, 15+ community repos, 50+ source docs), standalone ✓. Key cross-cutting findings: (1) `command` hooks work — stick to them, keep <200ms, (2) hooks are the only in-flight AI enforcement mechanism, (3) the community adoption gap (no cost alerting, no compliance logging, no client isolation) is the CoP talk's opportunity, (4) maturity risk requires version pinning for consulting deployment.
- **Next**: Spike complete. Feeds into `claude-tools-consulting-adoption` gap analysis (GAP 3 resolved) and the eventual `implementation-plans/claude-tools-cop-talk/` hooks preview segment.

## 2026-03-27 — Spike Completed
- **Type**: decision
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. 5 tasks completed, 5 research reports produced, 50+ source docs saved, all depth checklist items satisfied. Key conclusions: (1) CLAUDE.md + hooks forms a complete advisory→deterministic enforcement spectrum, (2) "anything that would cause client escalation belongs in a hook, not CLAUDE.md", (3) `command` handlers work reliably if kept <200ms — `prompt`/`agent` handlers are unusable in production, (4) hooks are the only in-flight AI enforcement mechanism, (5) the 6 consulting hook prototypes (especially cost alerting and SOC 2 logging) fill gaps no public configuration addresses, making the CoP talk hooks segment genuinely novel, (6) maturity is the biggest risk — pin versions and regression-test hook configs.
