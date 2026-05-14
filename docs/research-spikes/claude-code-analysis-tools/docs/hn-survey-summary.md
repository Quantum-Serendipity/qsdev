<!-- Generated: 2026-03-26 -->
<!-- Based on: Hacker News searches, GitHub READMEs, DEV Community articles -->

# Hacker News Survey: Claude Code Analysis, Debugging, Replay & Understanding Tools

## Overview

A comprehensive survey of community-built tools for analyzing, debugging, replaying, and understanding Claude Code sessions. All tools leverage Claude Code's local JSONL session logs stored in `~/.claude/projects/`. This ecosystem has exploded since mid-2025, with dozens of tools appearing on Hacker News as Show HN posts.

---

## Category 1: Session Replay & Visualization

### claude-replay
- **What it does**: Converts Claude Code (also Cursor, Codex) session JSONL logs into self-contained, interactive HTML replays with video-like playback controls
- **GitHub**: https://github.com/es617/claude-replay
- **HN**: https://news.ycombinator.com/item?id=47276604 (~March 2026)
- **Key features**: Single HTML file output, speed control (0.5x-5x), bookmarks/chapters, secret redaction, live monitoring mode, multiple themes
- **Install**: `npm install -g claude-replay` or `npx claude-replay`
- **Maturity**: High — active development, npm package, Docker support, web editor
- **Saved to**: `docs/hn-claude-replay.md`, `docs/github-claude-replay-readme.md`

### Mantra
- **What it does**: Desktop app for "Time Travel" through AI coding sessions — scrub through timelines like video, view code diffs at each step
- **GitHub**: https://github.com/mantra-hq/mantra-releases
- **Website**: mantra.gonewx.com
- **Referenced in**: DEV Community comparison article, claude-replay HN discussion
- **Key features**: Multi-tool support (Claude Code, Cursor, Codex, Gemini CLI), project organization, MCP hub, skills hub, content redaction, privacy-first (all local)
- **Platforms**: macOS 12+, Windows 10+, Linux (Ubuntu 20.04+)
- **Maturity**: High — cross-platform desktop app, multi-agent support
- **Saved to**: `docs/github-mantra.md`, `docs/devto-4-tools-comparison.md`

### Claude Code Timeline Viewer (Simon Willison)
- **What it does**: Browser-based interactive timeline for exploring session .jsonl files with filtering, search, and detail views
- **URL**: https://tools.simonwillison.net/claude-code-timeline
- **Key features**: Drag-and-drop file loading, color-coded event badges, filtering by type/role/content, image gallery extraction, shareable URL state, timezone switching
- **Install**: None — runs in browser
- **Maturity**: Medium — single-page tool by a well-known developer
- **Saved to**: `docs/simon-willison-timeline-viewer.md`

### claude-code-transcripts (Simon Willison)
- **What it does**: Converts Claude Code sessions to clean, paginated HTML transcripts with timeline index
- **GitHub**: https://github.com/simonw/claude-code-transcripts
- **Key features**: Interactive session picker grouped by repo, GitHub Gist export, batch conversion of all sessions
- **Install**: `uv tool install claude-code-transcripts`
- **Maturity**: Medium — by Simon Willison, Python/uv ecosystem
- **Saved to**: `docs/github-claude-code-transcripts.md`

### CCViewer
- **What it does**: Single-page HTML app for visualizing Claude Code sessions with thinking, tool calls, diffs
- **HN**: https://news.ycombinator.com/item?id=46545981 (~January 2026)
- **URL**: https://rcanand.gumroad.com/l/ccviewer
- **Key features**: Search/sort/filter, markdown export, fully local
- **Limitation**: Chromium-only (no Safari/Firefox)
- **Saved to**: `docs/hn-ccviewer.md`

---

## Category 2: Session Debugging & Forensics

### Claude DevTools
- **What it does**: Desktop app for deep inspection of Claude Code sessions — token attribution, compaction detection, subagent trees, team visualization
- **GitHub**: https://github.com/matt1398/claude-devtools
- **HN**: https://news.ycombinator.com/item?id=47004712 (~February 2026, 69 points)
- **Key features**: Per-turn token breakdown across 7 categories, compaction visualization, regex-based alerts, tool call inspector with inline diffs, SSH remote session support, multi-pane layout
- **Install**: `brew install --cask claude-devtools` (macOS), also Linux/Windows/Docker
- **Philosophy**: "Terminal tells you nothing. This shows you everything."
- **Maturity**: High — Electron app, cross-platform, Homebrew cask, actively developed
- **Saved to**: `docs/hn-claude-devtools.md`, `docs/github-claude-devtools.md`

### Claude-File-Recovery
- **What it does**: Recovers files from Claude Code session history — extracts any file that Claude read, edited, or wrote, including earlier versions at specific points in time
- **GitHub**: https://github.com/hjtenklooster/claude-file-recovery
- **HN**: https://news.ycombinator.com/item?id=47182387 (~March 2026, 99 points)
- **Install**: `pip install claude-file-recovery`
- **Key limitation**: Claude Code auto-deletes logs after 30 days by default
- **Maturity**: Medium — Python CLI/TUI, pip installable
- **Saved to**: `docs/hn-claude-file-recovery.md`

### Kintsugi (by Sonar)
- **What it does**: Desktop app for reviewing Claude Code sessions with code quality focus — PR-style review of AI-generated code with integrated SonarQube analysis
- **HN**: https://news.ycombinator.com/item?id=47006289 (~February 2026)
- **URL**: https://events.sonarsource.com/kintsugi/
- **Key features**: Multi-agent orchestration, PR-style code review with comments, plan review with inline comments, SonarQube integration
- **Limitation**: macOS-only, requires Java 17+
- **Maturity**: Early — prototype from Sonar, built with Claude Code itself
- **Saved to**: `docs/hn-kintsugi.md`

---

## Category 3: Session Search & Navigation

### claude-history
- **What it does**: Terminal UI for fuzzy-searching all Claude Code conversation history with built-in viewer
- **GitHub**: https://github.com/raine/claude-history
- **HN**: https://news.ycombinator.com/item?id=47110768
- **Key features**: Fuzzy multi-word search, tool output indexing (searches inside bash/grep results), vim-style navigation, resume/fork sessions, thinking blocks toggle, markdown rendering
- **Install**: `brew install raine/claude-history/claude-history` or `cargo install claude-history`
- **Maturity**: High — Rust, 13K LOC, Homebrew + Cargo, crates.io published
- **Saved to**: `docs/github-claude-history.md`

### search-sessions
- **What it does**: Fast CLI for full-text search across all Claude Code session history (<300ms)
- **GitHub**: https://github.com/sinzin91/search-sessions
- **HN**: https://news.ycombinator.com/item?id=47128630 (~March 2026)
- **Key features**: Quick index search (~18ms) and deep search via ripgrep (~280ms), session UUID output for `claude --resume`
- **Install**: Homebrew or Cargo
- **Philosophy**: "No database, no indexing step, no dependencies"
- **Maturity**: Medium — Rust, simple and focused
- **Saved to**: `docs/hn-search-sessions.md`

### claude-search
- **What it does**: Grep-like CLI for searching Claude Code session history with date filtering and code extraction
- **GitHub**: https://github.com/pi-netizen/claude-search
- **HN**: https://news.ycombinator.com/item?id=47176556 (~March 2026)
- **Key features**: Date filtering (`--since`), code extraction (`--code-only`), project scoping, session reopening, extended thinking visibility
- **Philosophy**: "No server, no API calls, no sync"
- **Saved to**: `docs/hn-claude-search.md`

### ccrider
- **What it does**: Session browser/search/resume tool with TUI, CLI, and MCP server interfaces
- **GitHub**: https://github.com/neilberkman/ccrider
- **HN**: https://news.ycombinator.com/item?id=46512501 (~January 2026, 19 points)
- **Key features**: Three interfaces (TUI/CLI/MCP), SQLite-backed, full-text search, markdown export, session resumption
- **Install**: Homebrew or source compilation (Go)
- **Maturity**: Medium — Go, SQLite, Homebrew
- **Saved to**: `docs/hn-ccrider.md`

### cc-sessions
- **What it does**: Fast CLI to list all Claude Code sessions across all projects with fzf picker
- **GitHub**: https://github.com/chronologos/cc-sessions
- **HN**: https://news.ycombinator.com/item?id=46805870 (~February 2026)
- **Key features**: Parallel directory scanning (Rust + rayon), fzf picker with preview, fork mode
- **Technical**: ~350 lines of Rust, reads sessions-index.json (not transcripts)
- **Saved to**: `docs/hn-cc-sessions.md`

---

## Category 4: Usage Analytics & Monitoring

### Rudel
- **What it does**: Analytics platform for Claude Code sessions — dashboard with token consumption, duration, activity trends, model usage
- **GitHub**: https://github.com/obsessiondb/rudel
- **HN**: https://news.ycombinator.com/item?id=47350416 (~March 2026, 144 points, 86 comments)
- **Key features**: Team analytics, auto-upload via hooks, batch upload, ClickHouse backend
- **Dataset insights**: 1,573 sessions, 15M+ tokens — skills used in only 4% of sessions, 26% abandoned within 60 seconds
- **Privacy note**: Hosted service; self-hosting via Docker available
- **Maturity**: High — most HN traction (144 points), team features, ClickHouse
- **Saved to**: `docs/hn-rudel.md`, `docs/github-rudel.md`

### Sniffly (by Chip Huyen)
- **What it does**: Claude Code analytics dashboard with usage stats and error analysis
- **GitHub**: https://github.com/chiphuyen/sniffly
- **HN**: https://news.ycombinator.com/item?id=45081711 (~September 2025, 41 points)
- **Maturity**: Medium — by a well-known ML practitioner
- **Saved to**: `docs/hn-sniffly.md`

### Subtle
- **What it does**: Local, open-source analytics web app for Claude Code session logs
- **GitHub**: https://github.com/itsderek23/subtle
- **HN**: https://news.ycombinator.com/item?id=46590647 (~January 2026)
- **Key features**: Usage over time (AI vs tool time), Git commit tracking, execution traces, session filtering
- **Install**: `pip install subtle-claude-code`
- **Privacy**: All processing local, no telemetry
- **Saved to**: `docs/hn-subtle.md`, `docs/github-subtle.md`

### ccusage
- **What it does**: CLI tool for analyzing Claude Code token usage from local JSONL files
- **GitHub**: https://github.com/ryoppippi/ccusage
- **Website**: https://ccusage.com
- **HN**: https://news.ycombinator.com/item?id=44610925 (~July 2025)
- **Maturity**: Established — one of the earlier tools in this space
- **Saved to**: `docs/hn-ccusage.md`

### tokenusage
- **What it does**: Rust CLI that tracks Claude Code/Codex tokens — 214x faster than ccusage
- **HN**: https://news.ycombinator.com/item?id=47262484 (~March 2026)
- **Benchmark**: 0.08s vs ccusage's 17.15s on 1,521 JSONL files (2.2 GB)
- **Note**: HN page could not be fetched due to rate limiting

### cc-toolkit
- **What it does**: Suite of 41 zero-dependency browser-based tools for Claude Code usage analysis
- **URL**: https://yurukusa.github.io/cc-toolkit/
- **HN**: https://news.ycombinator.com/item?id=47208594 (~March 2026)
- **Key tools**: cc-wrapped (yearly summary), cc-session-stats, cc-agent-load, cc-ghost-log, cc-impact, cc-peak, cc-burnout, cc-predict
- **Saved to**: `docs/hn-cc-toolkit.md`

---

## Category 5: Observability & Telemetry

### claude-code-otel
- **What it does**: Full OpenTelemetry observability stack for Claude Code with Grafana dashboards
- **GitHub**: https://github.com/ColeMurray/claude-code-otel
- **HN**: https://news.ycombinator.com/item?id=45325410 (~September 2025)
- **Architecture**: Claude Code -> OTel Collector -> Prometheus + Loki -> Grafana
- **Key features**: Cost tracking by model, DAU/WAU/MAU, tool success rates, API latency, productivity metrics
- **Maturity**: Medium — Docker Compose stack, MIT license
- **Saved to**: `docs/github-claude-code-otel.md`

### claude_telemetry (claudia)
- **What it does**: Drop-in CLI wrapper that sends structured OpenTelemetry traces to any backend (Logfire, Sentry, Honeycomb, Datadog)
- **GitHub**: https://github.com/TechNickAI/claude_telemetry
- **Key features**: Zero behavior change (swaps `claude` for `claudia`), per-execution and per-tool-call traces, cost tracking, multi-backend
- **Install**: `pip install claude_telemetry`
- **Maturity**: Medium — Python, pip installable
- **Saved to**: `docs/github-claude-telemetry.md`

### Native OpenTelemetry (built into Claude Code)
- Claude Code now supports OpenTelemetry natively via env vars: `CLAUDE_CODE_ENABLE_TELEMETRY=1`
- Exports metrics as time series and events via logs/events protocol
- Discussed in HN posts: item 45325410, item 45197359

---

## Category 6: JSONL Log Viewers (Lightweight)

### claude-JSONL-browser
- **What it does**: Web app that converts JSONL logs to human-readable Markdown with file explorer
- **GitHub**: https://github.com/withLinda/claude-JSONL-browser
- **Live demo**: jsonl.withlinda.dev
- **Key features**: Multi-file management, search across conversations, export to Markdown, model change tracking
- **Stack**: Next.js 15, TypeScript, Tailwind CSS
- **Saved to**: `docs/github-claude-jsonl-browser.md`

### cclogviewer
- **What it does**: Converts JSONL to interactive HTML with hierarchical display
- **GitHub**: https://github.com/Brads3290/cclogviewer
- **Key features**: Expandable sections, nested Task tool support, token tracking, syntax highlighting
- **Install**: `go install github.com/brads3290/cclogviewer/cmd/cclogviewer@latest`
- **Saved to**: `docs/github-cclogviewer.md`

### claude-code-log
- **What it does**: Python CLI with TUI for converting JSONL to HTML/Markdown with interactive browsing
- **GitHub**: https://github.com/daaain/claude-code-log
- **Key features**: TUI with session summaries, runtime JS filtering in HTML output, date range filtering, responsive design
- **Install**: `pip install claude-code-log`
- **Saved to**: `docs/github-claude-code-log.md`

### clog
- **What it does**: Zero-install web-based viewer with real-time file monitoring
- **GitHub**: https://github.com/HillviewCap/clog
- **Key features**: Auto-refresh on file changes, parent-child message threading, cross-platform browser support
- **Stack**: Vanilla JS, HTML5, Tailwind CSS
- **Saved to**: `docs/github-clog.md`

---

## Category 7: Usage Monitoring (Real-Time)

### Claude Code Usage Monitor (macOS menu bar)
- **HN**: https://news.ycombinator.com/item?id=44317012 (~June 2025)
- **What it does**: Real-time usage tracking to dodge usage cut-offs

### SessionWatcher (macOS menu bar)
- **HN**: https://news.ycombinator.com/item?id=45344681 (~September 2025)
- **What it does**: macOS menu bar app to monitor Claude Code usage

### Claude Code Usage Monitor (Windows)
- **GitHub**: https://github.com/CodeZeno/Claude-Code-Usage-Monitor
- **HN**: https://news.ycombinator.com/item?id=47194211 (~March 2026)
- **What it does**: Native Windows system tray app in Rust
- **Saved to**: `docs/hn-claude-code-usage-monitor-windows.md`

---

## Category 8: Hardware / Novel Interfaces

### Clawy
- **What it does**: Physical companion device (M5StickC Plus 2, ~$20) with retro JRPG-style animations that reacts to Claude Code activity
- **GitHub**: https://github.com/marcvermeeren/clawy
- **Website**: clawy.lol
- **HN**: https://news.ycombinator.com/item?id=47061181 (~February 2026)
- **Key features**: Uses Claude Code hook system over local WiFi, approve/deny via physical buttons, nothing leaves your network
- **Saved to**: `docs/hn-clawy.md`

---

## Category 9: Desktop History Viewers

### Claude Code History Viewer (macOS)
- **GitHub**: https://github.com/jhlee0409/claude-code-history-viewer
- **HN**: https://news.ycombinator.com/item?id=44459376 (~July 2025)
- **Stack**: Tauri + React + Rust
- **Saved to**: `docs/hn-claude-code-history-viewer-macos.md`

---

## Additional Tools Mentioned in Discussions (Not Directly Surveyed)

- **AgentsView.io** — Similar analytics platform to Rudel (mentioned in Rudel discussion)
- **K9 Audit** — Local-first causal auditing with hash chains (mentioned in Rudel discussion)
- **Linko** — MITM proxy for inspecting Claude Code API traffic, macOS only (mentioned in Rudel discussion)
- **unfucked.ai** — Tracks all file writes across agents (mentioned in Claude-File-Recovery discussion)
- **agentlore** — Multi-agent log aggregation (mentioned in claude-replay discussion)
- **Agent Flow** — Real-time visualization of agent orchestration as node graph (GitHub: patoles/agent-flow)

---

## Ecosystem Patterns

### Common Foundation
All tools build on Claude Code's local JSONL session logs in `~/.claude/projects/`. This is an undocumented format, but developers note confidence because Claude Code's official VS Code extension reads the same files.

### Key Limitation
Claude Code auto-deletes session logs after 30 days by default. Several tool authors recommend increasing this retention period.

### Language Distribution
- **Rust**: claude-history, search-sessions, cc-sessions, tokenusage, Windows usage monitor — chosen for speed on large JSONL datasets
- **Python**: claude-code-log, subtle, claude-file-recovery, claude_telemetry, claude-code-transcripts
- **TypeScript/JavaScript**: claude-devtools (Electron), clog, claude-JSONL-browser (Next.js), Rudel
- **Go**: ccrider, cclogviewer
- **Browser-only**: Claude Code Timeline Viewer, cc-toolkit, CCViewer

### Privacy Spectrum
- **Fully local**: Most tools (claude-devtools, subtle, claude-history, search-sessions, Mantra, etc.)
- **Self-hostable**: Rudel (Docker), claude-code-otel (Docker Compose)
- **Cloud/hosted**: Rudel (hosted option), Sniffly

### Maturity Tiers
- **Most mature**: claude-replay, claude-devtools, claude-history, Mantra, Rudel
- **Solid utilities**: search-sessions, ccrider, cc-sessions, ccusage, claude-code-transcripts
- **Early/experimental**: Kintsugi, Clawy, cc-toolkit, tokenusage
