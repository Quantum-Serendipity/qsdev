# Tasks: Claude Code Analysis Tools

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Define research question and scope** — Frame the core question and confirm with user
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: Research question: "What tools exist for analyzing, replaying, and understanding Claude Code sessions, and how do they compare in approach, capability, and maturity?"

## Phase 2: Research & Investigation

### Pending

### Active

### Completed
- [x] **Deep-dive top tools** — Detailed analysis of 5 tools: claude-replay, Claude DevTools, claude-history, Mantra, Rudel
  - Priority: high
  - Estimate: large
  - Started: 2026-03-26
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: All 5 deep-dives complete. Reports: claude-replay-research.md, claude-devtools-research.md, claude-history-research.md, mantra-research.md, rudel-research.md

- [x] **Compare approaches** — Cross-cutting comparison: problems solved, gaps, complementary vs overlapping
  - Priority: medium
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: Full comparison at comparison-research.md. Covers architectural spectrum, 5-question framework, head-to-head matrix, privacy spectrum, ecosystem patterns, gap analysis, and recommendations by use case.

- [x] **Survey HN posts and discussions** — Find recent Hacker News submissions about Claude Code analysis/debugging/replay tools. Save sources to docs/
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: Ran 18+ web searches across HN and GitHub. Found 25+ distinct tools. Saved 22 source documents to docs/. Summary at docs/hn-survey-summary.md.

- [x] **Catalog known tools** — Build taxonomy of tools: what each does, approach (semantic search, replay, analytics), repo/status
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: Organized tools into 9 categories in docs/hn-survey-summary.md: Session Replay, Debugging/Forensics, Session Search, Usage Analytics, Observability/Telemetry, JSONL Viewers, Usage Monitoring, Hardware, Desktop History Viewers. Expanded significantly in second session with 30+ additional web searches finding ~20 additional tools not in initial catalog (observability proxies, cost trackers, VS Code extensions, statusline plugins, analytics platforms).
- [x] **Analyze Claude Code session data format** — Understand what data Claude Code exposes (~/.claude/ structure, JSONL sessions, etc.) that these tools build on
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-26
  - Notes: Full report at session-data-format-research.md. Documented directory structure, all 5 JSONL message types, token usage fields, subagent format, file history, and aggregate stats. Based on direct inspection of live data plus 4 web sources saved to docs/.
