# Tasks: Claude Code Hooks in Practice

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Evaluate hook reliability & performance** — Test latency impact of different handler types, silent failure modes, error handling behavior. Open question from agentic-workflow-state-of-art spike.
  - Priority: medium
  - Estimate: small
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Report: `hook-reliability-research.md`
  - Notes: Analyzed 20+ GitHub issues across 5 bug categories. Command hooks are moderately reliable in CLI; VS Code extension is fundamentally broken for plugin hooks. Version regressions are the biggest risk. Prompt/agent handlers have near-zero adoption. Two distinct exit code systems cause silent failures. PreToolUse blocks trigger non-deterministic model behavior. API is unstable (new events every release). 15 source docs saved.

- [x] **Prototype consulting-specific hooks** — Build and test hook configurations for consulting use cases: test enforcement (Stop hook), client credential scanning (PreToolUse), cost alerting (PostToolUse token tracking), session logging for compliance
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Report: `consulting-hooks-research.md`
  - Notes: 6 complete hook configurations designed with full JSON, supporting scripts, failure modes, and deployment recommendations. Covers test enforcement, credential scanning, destructive command prevention, cost alerting (novel — fills community gap), SOC 2 session logging, and client isolation verification. Three-tier deployment strategy documented (managed → user → project). Cost alerting is the most complex (no community precedent); destructive commands the simplest (community-proven patterns). 3 new source docs saved.

- [x] **Compare hooks to alternatives** — Hooks vs. pre-commit hooks, CI/CD checks, code review. When is a Claude Code hook the right enforcement point vs. existing infrastructure? Consulting-specific answer.
  - Priority: medium
  - Estimate: small
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Report: `hooks-vs-alternatives-research.md`
  - Notes: Compared Claude Code hooks against 5 alternative enforcement points (pre-commit, CI/CD, IDE, code review, CLAUDE.md). Key finding: hooks are the only mechanism providing in-flight correction during AI code generation. Built a 4-step decision framework for consulting firms. Defense-in-depth architecture with 5 layers. 17 web sources analyzed.

- [x] **Survey CLAUDE.md patterns in the wild** — Find public repos with substantial CLAUDE.md files. What conventions emerge? What do teams standardize vs. leave individual? How do project-level vs. user-level settings interact?
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Report: `claudemd-patterns-research.md`
  - Notes: 8+ real-world CLAUDE.md files analyzed (including Anthropic's own), 15 source docs saved. Key finding: instruction budget constraint (~150-200 total, ~50 consumed by system prompt) is the fundamental design force. Progressive disclosure pattern: CLAUDE.md → rules → skills → hooks. Effective files are 50-100 lines. CLAUDE.md is advisory; hooks are deterministic enforcement.

- [x] **Survey community hook configurations** — Search GitHub repos, blog posts, HN discussions, and Claude Code community for shared hook configs. What do people actually enforce? What handler types dominate?
  - Priority: high
  - Estimate: medium
  - Started: 2026-03-27
  - Completed: 2026-03-27
  - Outcome: success
  - Report: `community-hooks-research.md`
  - Notes: 15+ repos analyzed, 20 source docs saved to docs/. Four dominant categories: auto-formatting, safety gates, notifications, observability. `command` handler type overwhelmingly dominates. `prompt`/`agent` types almost unused in community.
