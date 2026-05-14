# Research Summary: Claude Code Analysis Tools

## Overview
Survey and analysis of the emerging ecosystem of ~50 tools built to understand, debug, and improve Claude Code workflows. All tools build on the same undocumented JSONL session files in `~/.claude/projects/`. The ecosystem spans lightweight CLI utilities (350 LOC) to full SaaS analytics platforms (ClickHouse + Postgres + React), with most tools created by solo developers in early 2026.

## Topics

- **Session Data Format** — Complete. See [session-data-format-research.md](session-data-format-research.md). Covers ~/.claude/ directory structure, JSONL schema with 5 message types (user, assistant, system, progress, file-history-snapshot), token usage fields, subagent format, file history tracking. No official Anthropic schema documentation exists — format is empirically observed and undocumented.

- **claude-replay** — Complete. See [claude-replay-research.md](claude-replay-research.md). Zero-dependency JavaScript tool (573 stars, MIT) converting Claude Code/Cursor/Codex JSONL into self-contained interactive HTML replays with playback controls, secret redaction, 6 themes, and a web editor. Architecture: 24KB parser normalizes 3 agent formats → compression → vanilla JS HTML template. Unique niche: shareable interactive replays. Risk: bus factor 1, pre-1.0.

- **Claude DevTools** — Complete. See [claude-devtools-research.md](claude-devtools-research.md). Electron + React desktop app (2.7k stars, MIT) with 7-category per-turn token attribution, compaction visualization, recursive subagent tree rendering, SSH remote inspection, and multi-pane layout. Most feature-rich inspection tool. Token attribution is estimated (not API-reported). Risk: Electron binary size, format dependency, bus factor 1.

- **claude-history** — Complete. See [claude-history-research.md](claude-history-research.md). Rust TUI (110 stars, MIT) for fuzzy-searching session history with built-in markdown-rendered viewer. No database — uses binary cache with custom word-prefix matching, recency scoring, and rayon parallelism. Best daily-driver for "find that conversation." Compared to search-sessions (minimal ripgrep CLI), ccrider (Go, SQLite FTS5, MCP), cc-sessions (350 LOC, metadata only), ccsearch (semantic search with local embeddings).

- **Mantra** — Complete. See [mantra-research.md](mantra-research.md). Closed-source Rust+React desktop app with Git time-travel (timestamp→commit matching for code state reconstruction), deterministic sandboxed replay, AI-powered causality mapping, and MCP Hub aggregation. Most architecturally ambitious tool. Signature feature (Git time-travel) is genuinely novel. Risk: closed source, bus factor 1, minimal adoption (196 downloads), aggressive feature sprawl, default-on telemetry with device IDs.

- **Rudel** — Complete. See [rudel-research.md](rudel-research.md). ClickHouse-backed team analytics platform (223 stars, MIT) by ObsessionDB team. Hooks auto-upload full session transcripts. 15-view React dashboard with session archetypes, developer comparison, ROI calculations. Only tool providing organizational-level analytics. Their 1,573-session dataset revealed: 4% skill activation rate, 26% session abandonment within 60s, documentation tasks scored highest success. Risk: uploads full transcripts (source code, secrets, file contents) to remote server; self-hosting requires 3 services.

- **Cross-Cutting Comparison** — Complete. See [comparison-research.md](comparison-research.md). Ecosystem taxonomy, architectural spectrum analysis, head-to-head matrix of 5 tools, privacy spectrum, gap analysis, and recommendations by use case.

## Open Questions

- Will Anthropic ever document the JSONL schema or provide an official analysis API?
- Which tools will survive beyond 6 months given the solo-developer dominance?
- Will the observability space consolidate around native OTel or remain fragmented?

## Conclusions

### The Ecosystem is Real and Growing Fast
~50 tools in 11 categories have emerged in early 2026, driven by a clear user need: Claude Code's terminal output is insufficient for understanding what happened. The JSONL session format, while undocumented, provides enough data for rich analysis.

### Five Questions, No Single Answer
Every tool answers one of five questions: "What happened?" (replay/viewers), "Where was that?" (search), "Why did it go wrong?" (forensic inspection), "How much did it cost?" (usage tracking), "How is my team doing?" (analytics). No tool answers all five well — the ecosystem is complementary.

### Top Picks by Category
- **Sharing sessions**: claude-replay (self-contained HTML, zero deps, multi-format)
- **Finding past sessions**: claude-history (fuzzy search + built-in viewer, Rust TUI)
- **Debugging token usage**: Claude DevTools (7-category attribution, compaction viz)
- **Cost tracking**: ccusage (12k stars, ecosystem standard)
- **Team analytics**: Rudel (only option, but privacy tradeoff is significant)
- **Code forensics**: Mantra (Git time-travel is unique, but closed source)

### Key Risks
1. **Format dependency**: Every tool parses undocumented JSONL. One Claude Code update could break everything.
2. **Solo developer fragility**: Most tools have bus factor 1. Expect high churn within 6 months.
3. **Privacy spectrum**: Tools range from fully local to uploading complete transcripts. Team adoption requires careful evaluation.
4. **No standard observability**: 6+ competing approaches to Claude Code telemetry with no convergence.

### Notable Gaps
No tool yet provides: automated regression detection, cross-machine local-first aggregation, meaningful semantic search, CI/CD integration, collaborative annotation, cost optimization recommendations, or session diffing.
