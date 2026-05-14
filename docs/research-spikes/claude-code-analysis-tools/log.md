# Research Log: Claude Code Analysis Tools

## 2026-03-26 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Investigating the ecosystem of tools built to help understand, analyze, and debug Claude Code sessions. Focus areas include semantic search/indexing of sessions, forensic reconstruction of AI workflows, slideshow/video player-style session replay, root cause analysis utilities, and analytics for improving AI-assisted development processes. Many of these tools have been recently posted to and discussed on Hacker News.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-03-26 15:15 — Session Data Format Analysis Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [ClaudeWorld Session Storage](https://claude-world.com/tutorials/s16-session-storage/) → `docs/claude-world-session-storage.md`
  - [LobeHub Claude Code Metadata](https://lobehub.com/skills/marcus-marcus-skills-claude-code-metadata) → `docs/lobehub-claude-code-metadata.md`
  - [DuckDB Log Analysis](https://liambx.com/blog/claude-code-log-analysis-with-duckdb) → `docs/duckdb-log-analysis.md`
  - [KentGigger Conversation History](https://kentgigger.com/posts/claude-code-conversation-history) → `docs/kentgigger-conversation-history.md`
- **Summary**: Documented the full ~/.claude/ directory structure and JSONL session format through direct inspection of live session data. Identified 5 message types (user, assistant, system, progress, file-history-snapshot) with full field schemas. Key findings: sessions stored as JSONL under ~/.claude/projects/<encoded-cwd>/<uuid>.jsonl; each message carries uuid/parentUuid forming a conversation tree; assistant messages include detailed token usage with cache breakdowns; tool results include full file content; subagents get their own JSONL files in a subagents/ subdirectory; stats-cache.json provides daily aggregate metrics. No official Anthropic schema docs exist — format is empirically observed.
- **Next**: Continue with HN survey and tool catalog tasks.

## 2026-03-26 — HN Survey & Tool Catalog Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 18+ web searches, 22 source documents saved to docs/
- **Summary**: Comprehensive survey of Claude Code analysis tools via Hacker News and GitHub. Found 25+ distinct tools across 9 categories: Session Replay (claude-replay, Mantra, Timeline Viewer, claude-code-transcripts, CCViewer), Debugging/Forensics (Claude DevTools, Claude-File-Recovery, Kintsugi), Session Search (claude-history, search-sessions, claude-search, ccrider, cc-sessions), Usage Analytics (Rudel, Sniffly, Subtle, ccusage, tokenusage, cc-toolkit), Observability (claude-code-otel, claude_telemetry, native OTel), JSONL Viewers (claude-JSONL-browser, cclogviewer, claude-code-log, clog), Usage Monitoring (macOS/Windows menu bar apps), Hardware (Clawy), Desktop History (Claude Code History Viewer). All build on local JSONL logs in ~/.claude/projects/. Most mature: claude-replay, Claude DevTools, claude-history, Mantra, Rudel. Full taxonomy in docs/hn-survey-summary.md.
- **Next**: Deep-dive top tools, cross-cutting comparison.

## 2026-03-26 — Expanded Tool Catalog with Full README Fetches
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 30+ web searches, 20 new source documents saved to docs/
- **Summary**: Second comprehensive sweep expanding tool catalog significantly. Fetched and saved full READMEs for all major tools. Key additions/updates to catalog:
  - **Session Replay**: claude-replay (573 stars, zero deps, self-contained HTML), Mantra (Rust+React desktop app with Git time travel, v0.11.0)
  - **Deep Inspection**: claude-devtools (Homebrew-installable desktop app with 7-category token attribution, compaction viz, SSH remote, multi-pane), claude-compaction-viewer (swyx's TUI for inspecting compaction events specifically)
  - **Session Browsing**: Claude Code History Viewer / CCHV (727 stars, Rust+Tauri, multi-provider, server mode), claude-history (Rust TUI with fuzzy search), Agent Sessions (native macOS, 6 agent tools supported)
  - **Transcript Publishing**: claude-code-transcripts (Simon Willison, Python, paginated HTML + Gist publishing), claude-code-viewer (Philipp Spiess, web upload + shareable URLs)
  - **Usage/Cost Analysis**: ccusage (12k stars, TypeScript, daily/monthly/session/blocks), ccost (Rust, multi-currency, zero deps), cccost (fetch() hook for real-time cost), Claude-Code-Usage-Monitor (ML predictions, Rich TUI), claude-dev-insights (29-field plugin with Google Sheets sync)
  - **Observability/Tracing**: claude-code-otel (OTel Collector -> Prometheus+Loki -> Grafana stack), dev-agent-lens (LiteLLM proxy -> Arize/Phoenix), claude_telemetry (claudia CLI drop-in replacement), hooks-multi-agent-observability (12 event types, Vue dashboard), Arize Claude Code Plugin (9 hooks)
  - **Real-Time Monitoring**: claude-code-ui (Durable Streams Kanban board), Agent Flow (VS Code extension, node graph viz), Claude HUD (statusline plugin)
  - **Analytics Platform**: Rudel (ClickHouse-backed, team analytics, self-hostable)
  - **GUI Wrapper**: Opcode (Tauri desktop app, agents, MCP management, checkpoints)
  - **JSONL Conversion**: claude-JSONL-browser (Next.js, JSONL to Markdown), cclogviewer (Go, JSONL to HTML), claude-code-log (Python, HTML+Markdown+TUI)
  - **Misc**: cc-viewer (API request proxy monitoring), claude-code-logger (HTTP proxy for traffic analysis), DuckDB-based ad-hoc SQL analysis approach
  - **Blog Posts**: "I Tested 4 Tools" comparison by gonewx, DuckDB log analysis by Miyagi, Simon Willison transcript tools
- **Next**: Update catalog, write cross-cutting comparison if needed.

## 2026-03-26 — Deep Dive: claude-replay Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GitHub API repo metadata](https://api.github.com/repos/es617/claude-replay) → `docs/claude-replay-source-analysis.md`
  - [Source code: parser.mjs, renderer.mjs, editor-server.mjs, browser.mjs, resolve-session.mjs, secrets.mjs, extract.mjs](https://github.com/es617/claude-replay/tree/main/src) → `docs/claude-replay-source-analysis.md`
  - [Release history](https://api.github.com/repos/es617/claude-replay/releases) → `docs/claude-replay-source-analysis.md`
  - [Issue tracker](https://api.github.com/repos/es617/claude-replay/issues) → `docs/claude-replay-source-analysis.md`
  - Existing docs: `docs/claude-replay-readme.md`, `docs/github-claude-replay-readme.md`, `docs/hn-claude-replay.md`
- **Summary**: Full architectural deep-dive of claude-replay. Read all 8 source files to understand the processing pipeline: format detection → JSONL parsing (3 formats: Claude Code, Cursor, Codex CLI) → turn normalization → filtering → secret redaction (12 regex patterns) → deflate+base64 compression → HTML template injection → self-contained vanilla JS player. Key architectural decisions: zero runtime deps (Node built-ins only), zero output deps (vanilla JS + browser-native DecompressionStream), content deduplication via seenKeys Set to handle Claude Code's streaming JSONL format. Editor server is a full HTTP server with session discovery, search, editing, autosave, CSRF protection. 573 stars, 12 releases in 22 days, single developer (es617, 138 commits). Compared to alternatives: unique niche of shareable interactive replays vs. static viewers (transcripts) and local-only analyzers (DevTools, Mantra). Main limitations: bus factor 1, pre-1.0, no in-player search, large session memory scaling. Full report at claude-replay-research.md.
- **Next**: Continue deep-dives of remaining tools (Claude DevTools, claude-history, Mantra, Rudel).

## 2026-03-26 — Deep Dive: Rudel Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GitHub repo page](https://github.com/obsessiondb/rudel) → `docs/github-rudel.md`, `docs/rudel-readme.md`
  - [HN discussion (144pts, 86 comments)](https://news.ycombinator.com/item?id=47350416) → `docs/hn-rudel.md`
  - [Self-hosting docs](https://raw.githubusercontent.com/obsessiondb/rudel/main/docs/self-hosting.md) → `docs/rudel-self-hosting.md`
  - [Monorepo structure, schema, CLI source, adapter interface, release history](https://github.com/obsessiondb/rudel) → `docs/rudel-architecture-deep-dive.md`
- **Summary**: Full architectural deep-dive of Rudel. Explored the Turborepo monorepo (3 apps, 5 packages), read CLI source code (enable.ts, hook-upload.ts), agent adapter interface (types.ts), ClickHouse schema migrations (3 SQL files), Dockerfile, docker-compose.yml, package.json, CLAUDE.md, and all 15 dashboard page filenames. Key findings: (1) ClickHouse materialized views compute 40+ analytics columns at write time including session archetypes (quick_win/deep_work/struggle/exploration/abandoned/standard) and success scores; (2) pluggable agent-adapters pattern supports Claude Code + Codex with clean extensibility; (3) full session transcripts including source code and secrets are uploaded — the central privacy concern; (4) self-hosting requires 3 services (ClickHouse + Postgres + Bun app) but all have free tiers; (5) v0.1.9 fixed a prompt injection vulnerability in the session classifier; (6) built by ObsessionDB team, partly showcasing their managed ClickHouse product. Unique in ecosystem as the only team-level analytics tool. Full report at rudel-research.md.
- **Next**: Continue deep-dives of remaining tools (Claude DevTools, claude-history, Mantra).

## 2026-03-26 — Deep Dive: claude-history Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GitHub repo page](https://github.com/raine/claude-history) → `docs/claude-history-readme.md`, `docs/github-claude-history.md`
  - [Cargo.toml](https://raw.githubusercontent.com/raine/claude-history/main/Cargo.toml) → `docs/claude-history-cargo-toml.md`
  - [Source code: history/mod.rs, history/cache.rs, history/loader.rs, history/parser.rs, tui/mod.rs, tui/search.rs, cli.rs, config.rs](https://github.com/raine/claude-history/tree/main/src) → `docs/claude-history-src-architecture.md`
  - [GitHub metadata & releases](https://github.com/raine/claude-history/releases) → `docs/claude-history-github-metadata.md`
  - [Nimbalyst session managers comparison](https://nimbalyst.com/blog/best-session-managers-for-claude-code-and-codex) → `docs/nimbalyst-session-managers-comparison.md`
  - Existing comparison docs: `docs/hn-search-sessions.md`, `docs/hn-ccrider.md`, `docs/hn-cc-sessions.md`
- **Summary**: Full architectural deep-dive of claude-history. Read source code for all key modules: history layer (loader, parser, cache, path encoding), TUI layer (app, search, viewer, theme), CLI arguments, and config system. Architecture: ratatui 0.30 + crossterm 0.29 TUI with rayon-parallelized loading and search. No database — uses per-project bincode cache validated on file size + mtime, with negative caching for empty files. Search is a custom word-prefix fuzzy matcher with SIMD-accelerated fast rejection, CJK substring fallback, and recency scoring (3x/2x/1.5x/1x by age). Streaming loader sends batches to TUI for instant interactivity. Built-in viewer with pulldown-cmark markdown rendering, syntect code highlighting, tool display toggling, and thinking block visibility. Compared to 4 alternatives: search-sessions (minimal, ripgrep-based, no viewer), ccrider (Go, SQLite FTS5, MCP server), cc-sessions (minimal ~350 LOC, reads index files only), ccsearch (semantic search via local embeddings). Key limitations: no semantic search, linear scan on every keystroke, memory scales with total session text, no MCP support. 110 stars, v0.1.49, 10 releases in 11 days, single developer. Full report at claude-history-research.md.
- **Next**: Continue deep-dives of remaining tools (Claude DevTools, Mantra).

## 2026-03-26 — Deep Dive: Claude DevTools Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Raw README](https://raw.githubusercontent.com/matt1398/claude-devtools/main/README.md) → `docs/claude-devtools-raw-readme.md`
  - [GitHub repo metadata](https://github.com/matt1398/claude-devtools) → `docs/claude-devtools-github-metadata.md`
  - [Release history](https://github.com/matt1398/claude-devtools/releases) → `docs/claude-devtools-releases.md`
  - [Source architecture & issues](https://github.com/matt1398/claude-devtools) → `docs/claude-devtools-issues-and-architecture.md`
  - [HN discussion full comments](https://news.ycombinator.com/item?id=47004712) → `docs/claude-devtools-hn-comments.md`
  - [SECURITY.md](https://raw.githubusercontent.com/matt1398/claude-devtools/main/SECURITY.md) → inline in architecture doc
  - [electron.vite.config.ts](https://raw.githubusercontent.com/matt1398/claude-devtools/main/electron.vite.config.ts) → inline in architecture doc
  - Existing docs: `docs/claude-devtools-readme.md`, `docs/github-claude-devtools.md`, `docs/hn-claude-devtools.md`
- **Summary**: Full architectural deep-dive of Claude DevTools. Electron + React 18 + Fastify stack with electron-vite build system. Source organized as standard Electron app (main/preload/renderer/shared) with dual entry points (desktop and standalone/Docker). JSONL parsing pipeline: project discovery → stream parsing → context reconstruction (7-category token attribution estimated from message content) → compaction detection (sharp token drops between turns) → subagent resolution (recursive Task tool call → subagent file mapping) → file watching for real-time updates. Key unique capabilities: per-turn token attribution across CLAUDE.md/skills/@-mentions/tool I/O/thinking/teams/user text; compaction visualization showing fill-compress-refill timeline; recursive subagent tree rendering with independent metrics; SSH remote inspection via SFTP; notification triggers with regex matching. 2.7k stars, 184 forks, 10 releases in 5 weeks (v0.4.0–v0.4.9), single maintainer (matt1398). Main limitations: Electron binary size (community Tauri port at 1/10th size, issue #144), large session performance (ongoing optimizations), undocumented JSONL format dependency, estimated rather than actual token attribution. HN reception (69pts/44 comments): praised for filling observability gap, skeptics addressed by creator emphasizing passive viewer philosophy. Full report at claude-devtools-research.md.
- **Next**: Continue deep-dive of remaining tool (Mantra).

## 2026-03-26 — Deep Dive: Mantra Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [GitHub repo page](https://github.com/mantra-hq/mantra-releases) → `docs/github-mantra.md` (existing)
  - [Mantra homepage](https://mantra.gonewx.com) → `docs/mantra-homepage.md` (existing)
  - [Time Travel docs](https://docs.mantra.gonewx.com/features/time-travel) → `docs/mantra-docs-time-travel.md`
  - [Replay Mode docs](https://docs.mantra.gonewx.com/features/replay) → `docs/mantra-docs-replay-mode.md`
  - [Context Causality docs](https://docs.mantra.gonewx.com/features/context-causality) → `docs/mantra-docs-context-causality.md`
  - [MCP Hub docs](https://docs.mantra.gonewx.com/features/mcp-hub) → `docs/mantra-docs-mcp-hub.md`
  - [Pricing page](https://docs.mantra.gonewx.com/about/pricing) → `docs/mantra-docs-pricing.md`
  - [FAQ](https://docs.mantra.gonewx.com/about/faq) → `docs/mantra-docs-faq.md`
  - [Release history](https://github.com/mantra-hq/mantra-releases/releases) → `docs/mantra-releases-history.md`
  - [Creator blog: "I Built a Time Machine"](https://dev.to/gonewx/i-built-a-time-machine-for-ai-coding-sessions-heres-why-e8g) → `docs/mantra-devto-time-machine-blog.md`
  - [Creator blog: "10 days promoting"](https://dev.to/gonewx/i-spent-10-days-promoting-my-indie-dev-tool-heres-what-actually-worked-and-what-completely-3fkd) → `docs/mantra-devto-promotion-blog.md`
  - Existing: `docs/devto-4-tools-comparison.md` (authored by Mantra creator)
- **Summary**: Full deep-dive of Mantra. Closed-source Rust+React desktop app (likely Tauri) with 15 releases in 8 weeks. Unique Git time-travel feature reconstructs code states at each conversation turn via timestamp-to-commit matching — no other tool does this. Also includes deterministic replay (sandboxed re-execution of AI operations), AI-powered context causality mapping (which references influenced which code changes), full MCP Hub aggregation gateway with cross-tool config takeover and per-project permissions, Skills Hub, SSH remote access, and session live streaming. Freemium model: all local features free, optional Sync ($4/mo) and Publish ($8/mo). Key limitations: closed source (binary-only repo, 0 stars), Git dependency for signature feature, solo developer risk, aggressive feature sprawl for one person, default-on telemetry with device ID correlation despite privacy-first marketing. Minimal adoption: 196 downloads in first 10 days, HN Show HN received 2 points, all public content about Mantra traces back to the creator's own blog posts. Most architecturally ambitious tool in ecosystem but least validated by users. Full report at mantra-research.md.
- **Next**: All 5 deep-dives complete. Write cross-cutting comparison.

## 2026-03-26 — Cross-Cutting Comparison & Research Summary Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all 5 deep-dive reports plus the broader catalog into a cross-cutting comparison. Key frameworks developed: (1) Architectural spectrum from 350-LOC CLI to full SaaS platform; (2) Five-question framework — every tool answers "what happened?", "where was that?", "why did it go wrong?", "how much did it cost?", or "how is my team doing?"; (3) Privacy spectrum from fully local to full transcript upload; (4) Gap analysis identifying 7 unaddressed needs (automated regression detection, cross-machine local aggregation, semantic search, CI/CD integration, collaborative annotation, cost optimization recommendations, session diffing). Updated research.md with complete conclusions and top picks per category. All Phase 2 tasks complete.
- **Next**: Spike ready for Phase 3 synthesis/review or completion.

## 2026-03-26 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. Surveyed ~50 tools across 11 categories with 63+ source documents saved. Deep-dived 5 representative tools (claude-replay, Claude DevTools, claude-history, Mantra, Rudel). All 7 research reports pass the depth checklist. Key conclusion: the ecosystem answers 5 distinct questions (replay, search, forensics, cost, team analytics) with no single tool covering all — tools are complementary, not competing. Top picks identified per category. Critical risks: undocumented JSONL format dependency, solo-developer fragility, privacy spectrum from fully local to full transcript upload. Notable gaps remain in automated regression detection, semantic search, CI/CD integration, and session diffing.
