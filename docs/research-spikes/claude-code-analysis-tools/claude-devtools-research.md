# Claude DevTools — Deep-Dive Research Report

## Overview

Claude DevTools is a desktop application that reconstructs and visualizes Claude Code sessions by parsing the raw JSONL logs stored at `~/.claude/`. It is the most feature-rich inspection tool in the Claude Code ecosystem, offering per-turn token attribution across 7 categories, compaction visualization, subagent/team tree rendering, SSH remote session inspection, and a multi-pane layout. It is a passive viewer — it never modifies Claude Code or writes to session files.

- **Repository**: https://github.com/matt1398/claude-devtools
- **Website**: https://claude-dev.tools
- **License**: MIT
- **Stars**: ~2.7k
- **Forks**: 184
- **First release**: v0.4.0, February 14, 2025
- **Latest release**: v0.4.9, March 23, 2025 (10 releases over ~5 weeks)
- **Creator**: matt1398

---

## Architecture

### Tech Stack

Claude DevTools is an **Electron** application with a **React 18** frontend, built using **electron-vite**.

| Layer | Technology |
|-------|-----------|
| Desktop shell | Electron (with auto-updater) |
| Build system | electron-vite (Vite-based, three build targets) |
| Frontend | React 18, TailwindCSS, Zustand (state management) |
| Backend (main process) | Fastify server, SQLite |
| Markdown/code rendering | Syntax highlighting, AST processing, markdown rendering |
| SSH | ssh2 library (optional native bindings with JS fallback) |
| Package manager | pnpm 10.25.0+ |
| Node.js | 20+ |
| Testing | Vitest with coverage |
| Linting | ESLint (12+ plugins), Prettier, TypeScript strict checking |

### Source Code Organization

The codebase follows standard Electron architecture:

```
src/
├── main/           # Electron main process
│   ├── constants/  # Configuration values
│   ├── http/       # HTTP server (standalone/Docker mode)
│   ├── ipc/        # IPC handlers (main ↔ renderer bridge)
│   ├── services/   # Core logic: JSONL parsing, context reconstruction, SSH, file watching
│   ├── types/      # TypeScript type definitions
│   ├── utils/      # Helper functions
│   ├── index.ts    # Electron desktop entry point
│   └── standalone.ts  # Standalone/Docker entry point
├── preload/        # Secure context bridge scripts
├── renderer/       # React UI (Vite-bundled for browser)
└── shared/         # Cross-process types, constants, utilities
```

### Dual Deployment Modes

1. **Electron desktop app**: Native file system watchers with IPC for instant updates. Auto-updater via GitHub Releases API.
2. **Standalone/Docker server**: Fastify HTTP server on port 3456. Uses SSE (Server-Sent Events) instead of IPC for real-time updates (slightly slower). No auto-updater or SSH. Accessed via browser at `http://localhost:3456`.

### How It Reads Session Data

The application reads JSONL files from `~/.claude/projects/<encoded-cwd>/<session-uuid>.jsonl`. The main process services:

1. **Discover projects**: Scan `~/.claude/projects/` for project directories.
2. **Parse JSONL**: Stream-parse each session file line by line. Each line is a JSON message with type (user, assistant, system, progress, file-history-snapshot), uuid/parentUuid forming a conversation tree, and detailed token usage with cache breakdowns.
3. **Reconstruct context**: Walk each turn and reconstruct what occupied the context window — CLAUDE.md injections, skill activations, @-mentions, tool I/O, extended thinking, team overhead, and user prompts. This is estimated from the JSONL data, not directly reported by the API.
4. **Detect compaction**: Identify boundaries where Claude Code silently compressed the conversation (token counts drop sharply between turns).
5. **Resolve subagents**: Follow Task tool calls to link parent sessions to subagent JSONL files in `subagents/` subdirectories. Recursively resolve nested subagents.
6. **File watching**: Monitor session directories for changes, enabling near-real-time updates as active sessions write new JSONL lines.

For SSH remote sessions, an SFTP channel streams the remote `~/.claude/` directory. Each SSH host gets isolated service context with independent caches, file watchers, and parsers. Workspace state (including open tabs) is snapshot to IndexedDB on switch.

### Build Configuration Details

The `electron.vite.config.ts` reveals:
- A custom Rollup plugin stubs out `.node` native addon imports (ssh2, cpu-features) since they have optional native bindings that cannot be bundled but include pure JavaScript fallbacks.
- Production dependencies are bundled into the main process output to avoid pnpm symlink issues with electron-builder's asar packaging.
- Three build targets: main (CJS `.cjs`), preload (CJS `.cjs`), renderer (browser).
- Path aliases (`@main`, `@shared`, `@preload`, `@renderer`) for clean imports.

---

## Key Features — How They Actually Work

### 1. Per-Turn Token Attribution (7 Categories)

The most distinctive feature. The engine walks each session turn and categorizes token consumption into:

1. **CLAUDE.md files** — broken down by global (`~/.claude/CLAUDE.md`), project, and directory-level
2. **Skill activations** — loaded via slash commands
3. **@-mentioned files** — files explicitly referenced in user prompts
4. **Tool call I/O** — inputs sent to tools and outputs returned (Read file contents, Edit diffs, Bash output)
5. **Extended thinking** — Claude's chain-of-thought reasoning tokens
6. **Team coordination overhead** — TeamCreate, SendMessage, TaskCreate/TaskUpdate messages
7. **User prompt text** — the actual human-written prompts

This is surfaced in three UI elements:
- **Context Badge**: on each assistant response, showing total estimated tokens
- **Token Usage Popover**: percentage breakdown across categories on hover/click
- **Session Context Panel**: full session-level aggregation

**Important caveat**: These are *estimated* attributions. The JSONL logs contain token counts per message but not a native per-category breakdown. The engine infers attribution by analyzing message content structure.

### 2. Compaction Visualization

Claude Code silently compresses conversations when the context window fills. Most tools (and the CLI itself) don't surface this at all. Claude DevTools:

- Detects compaction boundaries by identifying sharp drops in cumulative token counts between turns
- Measures the token delta before and after each compaction event
- Renders a timeline showing how context fills, compresses, and refills over the session lifecycle
- Shows what was in the window at any point and how composition shifted after compaction

This is particularly valuable for understanding why long sessions degrade — you can see which categories of content survived compaction and which were discarded.

### 3. Subagent & Team Trees

Claude Code's distributed execution model (Task tool, TeamCreate, SendMessage) produces interleaved output that is nearly impossible to follow in the terminal. Claude DevTools:

- **Subagent resolution**: Follows Task tool calls to their corresponding subagent JSONL files. Renders each as an expandable inline card with its own tool trace, token metrics, duration, and cost. Nested subagents (agents spawning agents) render as a recursive tree.
- **Teammate visualization**: Detects SendMessage calls with color and summary metadata. Renders as distinct color-coded cards separated from regular user messages. Each teammate identified by name and assigned color.
- **Team lifecycle**: Full visibility of TeamCreate initialization, TaskCreate/TaskUpdate coordination, direct messages and broadcasts, shutdown requests/responses, and TeamDelete teardown.
- **Session summary**: Distinct teammate count vs. subagent count for at-a-glance distribution understanding.

### 4. SSH Remote Sessions

Parses `~/.ssh/config` for host aliases. Supports agent forwarding, private keys, and password auth. Opens SFTP channel to stream remote `~/.claude/` logs. Each SSH host gets isolated service context. Workspace state persisted to IndexedDB across switches.

Known limitation: Issue #96 reports inability to resume remote connections after interruption.

### 5. Multi-Pane Layout

Drag-and-drop tabs between panes, split views for side-by-side session comparison. Behaves like an IDE tab system for AI conversations.

### 6. Rich Tool Call Inspector

Specialized viewers per tool type:
- **Read**: syntax-highlighted code with line numbers
- **Edit**: inline diffs with add/remove highlighting
- **Bash**: command + rendered output
- **Subagent**: full execution tree expandable in-place
- **MCP tools**: input/output rendering (added v0.4.4)

### 7. Custom Notification Triggers

Regex-based rules generating system notifications:
- **Built-in defaults**: `.env` file access, `is_error: true` tool results, >8,000 total tokens per call
- **Custom matching**: regex against `file_path`, `command`, `prompt`, `content`, `thinking`, `text`
- **Noise control**: token thresholds, ignore patterns, repository scoping
- Disabled by default since v0.4.5 (users found default notifications noisy)

### 8. Command Palette & Search

Cmd+K spotlight-style search across all sessions in a project. Results show context snippets with keyword highlighting. Global session search added in v0.4.5, performance optimized in v0.4.8 with cached compiled regexes.

### 9. Session Export

Added in v0.4.5: export sessions as Markdown, JSON, or Plain Text.

---

## Tradeoffs and Limitations

### Architectural Tradeoffs

| Decision | Benefit | Cost |
|----------|---------|------|
| Electron | Cross-platform, rich UI, native file watchers | Large binary size (~10x a Tauri equivalent per issue #144), high memory usage |
| Passive viewer (read-only) | Zero risk to sessions, no modification to Claude Code | Cannot send messages, control sessions, or interact with the agent |
| Local-only data | No API keys, zero network calls, full privacy | Cannot access billing data, API-level metrics, or cross-device sessions |
| Estimated token attribution | Provides insight unavailable anywhere else | Estimates may not perfectly match actual API token counts |
| Undocumented JSONL format | Only data source available | Format changes could break parsing (mitigated by VS Code extension reading same files) |

### Known Limitations

1. **Performance on large sessions**: Issue #95 reported "Slow & Unresponsive" behavior. Performance optimizations have been ongoing (v0.4.6 fixed regressions, v0.4.8 optimized search, v0.4.9 optimized message categorization with cached regexes). Large sessions with many tool calls and subagents may still strain the Electron renderer.

2. **Binary size**: Electron app is large. A community member created a Tauri port at 1/10th the size (issue #144). This is inherent to Electron.

3. **Format dependency**: Relies on undocumented `.jsonl` format from `~/.claude/`. kzahel on HN noted this is an "undocumented API surface representing a risk" and that the CLI modifies files during runtime for cleanup/migration. matt1398's defense: the official VS Code extension reads these same files, and adding handlers for format changes is "trivial."

4. **Docker/standalone mode limitations**: Real-time updates are slower than Electron (SSE vs. native file watchers). SSH feature unavailable. Cross-network access has bugs (issue #132: web interface not working from another computer on Docker network).

5. **SSH remote recovery**: Cannot resume remote connection after interruption (issue #96). Must reconnect manually.

6. **Settings persistence**: Bug where settings are not applied on startup (issue #142).

7. **Session rename tracking**: UI shows stale session names after Claude Code's `/rename` command (issue #138).

8. **Windows/WSL quirks**: Drive letter casing inconsistency caused missing sessions (fixed in v0.4.9). WSL path handling has required ongoing fixes.

9. **No thinking content initially**: Extended thinking was not visible at launch (issue #119, fixed by v0.4.8/v0.4.9).

10. **Token attribution is estimated**: The 7-category breakdown is inferred from JSONL content structure, not reported by the API. Accuracy depends on correctly identifying which content belongs to which category.

---

## Maturity Assessment

### Quantitative

| Metric | Value | Assessment |
|--------|-------|------------|
| Stars | ~2,700 | Strong for a 5-week-old niche tool |
| Forks | 184 | Active community interest |
| Open issues | 11 | Manageable backlog |
| Open PRs | 11 | Active contribution |
| Releases | 10 (v0.4.0–v0.4.9) | Rapid iteration |
| Time span | Feb 14 – Mar 23, 2025 (5 weeks) | Very young project |
| Release cadence | Multiple per week initially, slowing to biweekly | Typical early-stage curve |
| Platforms | macOS (ARM+Intel), Linux (4 formats), Windows, Docker | Excellent breadth |
| Homebrew | Yes (`brew install --cask claude-devtools`) | Low-friction macOS install |
| Product Hunt | Featured (post_id=1080673) | Marketing effort |
| HN reception | 69 points, 44 comments | Solid Show HN performance |

### Qualitative

- **Solo creator project** (matt1398) with community contributions
- **v0.4.x version** suggests the author considers it pre-1.0 / beta quality
- **v0.4.7 reverted several PRs** — indicates scope management challenges with community contributions
- **Active maintainer**: matt1398 responded to nearly every HN comment and engages on GitHub issues
- **CI/CD**: GitHub Actions CI pipeline, automated builds for all platforms
- **Code quality**: TypeScript strict mode, ESLint with 12+ plugins, Vitest test suite with coverage
- **Security**: Documented in SECURITY.md, path containment checks, blocked credential paths, read-only Docker mounts

### Risk Factors

- Single maintainer: bus factor of 1
- Pre-1.0: API/UI breaking changes likely
- Depends on undocumented format: single point of failure if Anthropic changes JSONL structure
- Electron: large binary, high memory, potential perf issues at scale

---

## Comparison: What Makes It Unique

### vs. Simple Viewers (claude-replay, cclogviewer, claude-JSONL-browser)

Simple viewers render JSONL sessions as readable HTML/markdown. Claude DevTools adds:
- **Token attribution**: No other tool breaks down context consumption into 7 categories per turn
- **Compaction visualization**: Most tools don't detect or surface compaction events at all
- **Subagent tree resolution**: Simple viewers show flat message lists; Claude DevTools recursively resolves subagent sessions into expandable trees
- **SSH remote inspection**: Unique to Claude DevTools
- **Multi-pane layout**: IDE-style session comparison
- **Notification triggers**: Proactive monitoring, not just passive viewing

### vs. CCHV (Claude Code History Viewer)

CCHV is a Rust+Tauri app (much smaller binary) focused on browsing and searching sessions across multiple AI providers. Claude DevTools goes deeper on *inspection* — token attribution, compaction viz, team lifecycle. CCHV goes wider on *breadth* — multi-provider support, server mode.

### vs. Mantra

Mantra is a Rust+React desktop app with Git time-travel (checkout at any point in session). Claude DevTools has no Git integration but provides deeper token-level analysis and compaction visibility.

### vs. Rudel

Rudel is a ClickHouse-backed team analytics platform. Claude DevTools is individual-focused inspection. They serve different audiences: Rudel for engineering managers tracking team AI usage patterns; Claude DevTools for individual developers debugging specific sessions.

### vs. ccusage/ccost

Cost tracking tools provide aggregate usage/spend data. Claude DevTools provides per-turn token attribution — understanding *what* consumed context, not just *how much* was spent.

### Unique Insights It Provides

1. **What's eating your context window**: The 7-category attribution answers "why did my session run out of context?" — was it massive tool outputs? Bloated CLAUDE.md? Too many @-mentions?
2. **When and how compaction happened**: Critical for understanding session quality degradation in long conversations.
3. **Subagent accountability**: Which subagent consumed the most tokens? Which one failed? How deep is the agent tree?
4. **Security monitoring**: Real-time alerts for .env access, credential paths, and error patterns.

---

## Failure Modes

### Large Sessions
Sessions with hundreds of turns, many subagents, and large tool outputs (e.g., full file reads) can cause UI sluggishness. The Electron renderer must parse and render all of this in a web view. Ongoing optimizations (regex caching, lazy rendering) help but may not fully solve for extreme sessions.

### Missing or Corrupted Data
- If JSONL lines are malformed (e.g., process killed mid-write), parsing may skip or error on those lines.
- If Claude Code changes the JSONL format without notice, parsing breaks until handlers are updated.
- Subagent resolution depends on correct Task tool call → subagent file mapping. If subagent files are missing (e.g., deleted or corrupted), the tree will be incomplete.
- Issue #123: Windows drive letter casing inconsistency caused sessions to be invisible (fixed).

### SSH Edge Cases
- Connection interruption requires full reconnection (issue #96).
- Docker cross-network access has bugs (issue #132).
- Remote `~/.claude/` directory must be accessible via SFTP (non-standard paths require manual config).

### Format Evolution Risk
The single biggest failure mode: Anthropic modifies the JSONL structure. The creator's defense is that the VS Code extension reads the same files, providing some stability guarantee. kzahel's approach of Zod schema validation is more defensive. The actual risk is moderate — Anthropic has incentive to keep the format stable for their own extension, but they don't document or guarantee it.

---

## Real-World Reception

### HN Discussion Summary (69 points, 44 comments)

**Positive**:
- "Anthropic should hire this person" (igravious)
- Multiple users acknowledge the observability gap this fills
- Creator's technical depth in responses impressed commenters

**Skeptical**:
- "Just use --verbose" → Creator: "floods terminal with noise, defeats purpose"
- "Just look at the diffs" → Creator: "standard observability, not micromanagement"
- "Won't this break with format changes?" → Creator: "VS Code extension reads same files"
- Config file count criticized → Defended as standard modern dev setup

**Concerns raised**:
- ToS risk if using alternative tools with Claude Code subscription (azuanrb)
- Undocumented JSONL format as fragile dependency (kzahel)
- Claude Code itself mocked as "liability" for frequent changes (quikoa)

**Alternative suggestions**: OpenCode, Pi Code Agent, claude-trace, Braintrust plugin, OpenTelemetry-based approaches.

**Creator's positioning**: Not a wrapper or replacement — a passive viewer for debugging and observability. Works with every session regardless of execution context. Zero network calls.

---

## Sources

| Document | Location |
|----------|----------|
| Raw README (full) | `docs/claude-devtools-raw-readme.md` |
| GitHub metadata | `docs/claude-devtools-github-metadata.md` |
| Release history | `docs/claude-devtools-releases.md` |
| Architecture & issues | `docs/claude-devtools-issues-and-architecture.md` |
| HN discussion (full comments) | `docs/claude-devtools-hn-comments.md` |
| Earlier README summary | `docs/claude-devtools-readme.md` |
| Earlier GitHub summary | `docs/github-claude-devtools.md` |
| Earlier HN summary | `docs/hn-claude-devtools.md` |

---

## Depth Checklist

- [x] **Underlying mechanism explained**: Full architecture (Electron + React + Fastify), JSONL parsing pipeline, context reconstruction engine, dual deployment modes
- [x] **Key tradeoffs and limitations identified**: Electron size/perf vs. rich UI, estimated vs. actual token attribution, undocumented format dependency, read-only passive design
- [x] **Compared to alternatives**: vs. simple viewers, CCHV, Mantra, Rudel, ccusage — unique positioning on token attribution and compaction viz
- [x] **Failure modes and edge cases**: Large session perf, format evolution, SSH interruption, Windows path quirks, corrupted JSONL, missing subagent files
- [x] **Concrete examples**: 10 releases analyzed, 11 open issues cataloged, 44 HN comments reviewed, specific bug reports and feature requests
- [x] **Standalone-readable**: Sufficient for decisions without consulting original sources
