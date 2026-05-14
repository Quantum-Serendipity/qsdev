<!-- Source: https://raw.githubusercontent.com/matt1398/claude-devtools/main/README.md -->
<!-- Retrieved: 2026-03-26 -->

# claude-devtools — Full Raw README

A desktop app that reconstructs exactly what Claude Code did — every file path, every tool call, every token — from the raw session logs already on your machine.

**Tagline**: "Terminal tells you nothing. This shows you everything."

**Website**: https://claude-dev.tools

**Product Hunt**: Featured product (post_id=1080673)

100% free, open source. No API keys. No configuration. Just download, open, and see everything Claude Code did.

---

## Installation

### Homebrew (macOS)

```bash
brew install --cask claude-devtools
```

### Direct Download

| Platform | Download | Notes |
|----------|----------|-------|
| **macOS** (Apple Silicon) | `.dmg` arm64 asset | Drag to Applications. On first launch: right-click → Open |
| **macOS** (Intel) | `.dmg` x64 asset | Drag to Applications. On first launch: right-click → Open |
| **Linux** | `.AppImage` / `.deb` / `.rpm` / `.pacman` | Choose the package format for your distro |
| **Windows** | `.exe` | Standard installer. May trigger SmartScreen |
| **Docker** | `docker compose up` | Open `http://localhost:3456` |

The app reads session logs from `~/.claude/` — the data is already on your machine. No setup, no API keys, no login.

---

## Why This Exists

### Claude Code stopped telling you what it's doing.

Recent Claude Code updates replaced detailed tool output with opaque summaries. `Read 3 files`. `Searched for 1 pattern`. `Edited 2 files`. No paths, no content, no line numbers. The context usage indicator became a three-segment progress bar with no breakdown. To get the details back, the only option is `--verbose` — which dumps raw JSON, internal system prompts, and thousands of lines of noise into your terminal.

**There is no middle ground in the CLI.** You either see too little or too much.

claude-devtools restores the information that was taken away — structured, searchable, and without a single modification to Claude Code itself. It reads the raw session logs from `~/.claude/` and reconstructs the full execution trace: every file path that was read, every regex that was searched, every diff that was applied, every token that was consumed — organized into a visual interface you can actually reason about.

### The wrapper problem.

There are many GUI wrappers for Claude Code — Conductor, Craft Agents, Vibe Kanban, 1Code, ccswitch, and others. The creator tried them all. None of them solved the actual problem:

**They wrap Claude Code.** They inject their own prompts, add their own abstractions, and change how Claude behaves. If you love the terminal, you don't want that.

**They only show their own sessions.** Run something in the terminal? It doesn't exist in their UI. You can only see what was executed through *their* tool.

**You can't debug what went wrong.** A session failed — but why? The context filled up too fast — but what consumed it? A subagent spawned 5 child agents — but what did they do?

**You can't monitor what matters.** Want to know when Claude reads `.env`? When a single tool call exceeds 4K tokens? When a teammate sends a shutdown request? You'd have to wire up hooks manually.

**claude-devtools takes a different approach.** It doesn't wrap or modify Claude Code at all. It reads the session logs that already exist on your machine and turns them into a rich, interactive interface — regardless of whether the session ran in the terminal, in an IDE, or through another tool.

---

## Key Features

### Visible Context Reconstruction

Claude Code doesn't expose what's actually in the context window. claude-devtools reverse-engineers it.

The engine walks each turn of the session and reconstructs the full set of context injections — **CLAUDE.md files** (broken down by global, project, and directory-level), **skill activations**, **@-mentioned files**, **tool call inputs and outputs**, **extended thinking**, **team coordination overhead**, and **user prompt text**.

The result is a per-turn breakdown of estimated token attribution across 7 categories, surfaced in three places: a **Context Badge** on each assistant response, a **Token Usage popover** with percentage breakdowns, and a dedicated **Session Context Panel**.

### Compaction Visualization

When Claude Code hits its context limit, it silently compresses your conversation and continues. Most tools don't even notice this happened.

claude-devtools detects these compaction boundaries, measures the token delta before and after, and visualizes how your context fills, compresses, and refills over the course of a session. You can see exactly what was in the window at any point, and how the composition shifted after each compaction event.

### Custom Notification Triggers

Define rules for when you want to receive **system notifications**. Match on regex patterns, assign colors, and filter your inbox by trigger.

- **Built-in defaults**: `.env File Access Alert`, `Tool Result Error` (`is_error: true`), and `High Token Usage` (default: 8,000 total tokens).
- **Custom matching**: use regex against specific fields like `file_path`, `command`, `prompt`, `content`, `thinking`, or `text`.
- **Sensitive-file monitoring**: create alerts for `.env`, `secrets`, payment/billing/stripe paths, or any project-specific pattern.
- **Noise control**: choose input/output/total token thresholds, add ignore patterns, and scope triggers to selected repositories.

### Rich Tool Call Inspector

Every tool call is paired with its result in an expandable card. Specialized viewers render each tool natively:
- **Read** calls show syntax-highlighted code with line numbers
- **Edit** calls show inline diffs with added/removed highlighting
- **Bash** calls show command output
- **Subagent** calls show the full execution tree, expandable in-place

### Team & Subagent Visualization

Claude Code now spawns subagents via the Task tool and coordinates entire teams via `TeamCreate`, `SendMessage`, and `TaskUpdate`. In the terminal, all of this collapses into an unreadable stream. claude-devtools untangles it.

- **Subagent sessions** are resolved from Task tool calls and rendered as expandable inline cards — each with its own tool trace, token metrics, duration, and cost. Nested subagents (agents spawning agents) render as a recursive tree.
- **Teammate messages** — sent via `SendMessage` with color and summary metadata — are detected and rendered as distinct color-coded cards. Each teammate is identified by name and assigned color.
- **Team lifecycle** is fully visible: `TeamCreate` initialization, `TaskCreate`/`TaskUpdate` coordination, `SendMessage` direct messages and broadcasts, shutdown requests and responses, and `TeamDelete` teardown.
- **Session summary** shows distinct teammate count separately from subagent count.

### Command Palette & Cross-Session Search

Hit **Cmd+K** for a Spotlight-style command palette. Search across all sessions in a project — results show context snippets with highlighted keywords. Navigate directly to the exact message.

### SSH Remote Sessions

Connect to any remote machine over SSH and inspect Claude Code sessions running there — same interface, no compromise.

claude-devtools parses your `~/.ssh/config` for host aliases, supports agent forwarding, private keys, and password auth, then opens an SFTP channel to stream session logs from the remote `~/.claude/` directory. Each SSH host gets its own isolated service context with independent caches, file watchers, and parsers. Switching between local and remote workspaces is instant — the app snapshots your current state to IndexedDB before the switch and restores it when you return, tabs and all.

### Multi-Pane Layout

Open multiple sessions side-by-side. Drag-and-drop tabs between panes, split views, and compare sessions in parallel.

---

## What the CLI Hides vs. What claude-devtools Shows

| What you see in the terminal | What claude-devtools shows you |
|------------------------------|-------------------------------|
| `Read 3 files` | Exact file paths, syntax-highlighted content with line numbers |
| `Searched for 1 pattern` | The regex pattern, every matching file, and the matched lines |
| `Edited 2 files` | Inline diffs with added/removed highlighting per file |
| A three-segment context bar | Per-turn token attribution across 7 categories + compaction visualization |
| Subagent output interleaved with the main thread | Isolated execution trees per agent, expandable inline with own metrics |
| Teammate messages buried in session logs | Color-coded teammate cards with name, message, and full team lifecycle visibility |
| Critical events mixed into normal output | Trigger-filtered notification inbox for .env access, errors, high token usage |
| `--verbose` JSON dump | Structured, filterable, navigable interface |

---

## Docker / Standalone Deployment

Run claude-devtools without Electron — in Docker, on a remote server, or anywhere Node.js runs.

### Quick Start (Docker Compose)

```bash
docker compose up
```

Open `http://localhost:3456` in your browser.

### Quick Start (Docker)

```bash
docker build -t claude-devtools .
docker run -p 3456:3456 -v ~/.claude:/data/.claude:ro claude-devtools
```

### Quick Start (Node.js)

```bash
pnpm install
pnpm standalone:build
node dist-standalone/index.cjs
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_ROOT` | `~/.claude` | Path to the `.claude` data directory |
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `3456` | Listen port |
| `CORS_ORIGIN` | `*` (standalone) | CORS origin policy |

### Notes

- **Real-time updates may be slower than Electron.** The Electron app uses native file system watchers with IPC for instant updates. The Docker/standalone server uses SSE (Server-Sent Events) over HTTP, which may introduce slight delays when sessions are actively being written to.
- **Custom Claude root path.** If your `.claude` directory is not at `~/.claude`, update the volume mount.

### Security-Focused Deployment

The standalone server has **zero** outbound network calls. For maximum isolation:

```bash
docker run --network none -p 3456:3456 -v ~/.claude:/data/.claude:ro claude-devtools
```

---

## Development

**Prerequisites:** Node.js 20+, pnpm 10+

```bash
git clone https://github.com/matt1398/claude-devtools.git
cd claude-devtools
pnpm install
pnpm dev
```

The app auto-discovers your Claude Code projects from `~/.claude/`.

### Build for Distribution

```bash
pnpm dist:mac:arm64  # macOS Apple Silicon (.dmg)
pnpm dist:mac:x64    # macOS Intel (.dmg)
pnpm dist:win        # Windows (.exe)
pnpm dist:linux      # Linux (AppImage/.deb/.rpm/.pacman)
pnpm dist            # macOS + Windows + Linux
```

### Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Development with hot reload |
| `pnpm build` | Production build |
| `pnpm typecheck` | TypeScript type checking |
| `pnpm lint:fix` | Lint and auto-fix |
| `pnpm test` | Run all tests |
| `pnpm test:watch` | Watch mode |
| `pnpm test:coverage` | Coverage report |
| `pnpm check` | Full quality gate (types + lint + test + build) |

---

## Security

IPC handlers validate all inputs with strict path containment checks. File reads are constrained to the project root and `~/.claude`. Sensitive credential paths are blocked. See SECURITY.md for details.

## License

MIT
