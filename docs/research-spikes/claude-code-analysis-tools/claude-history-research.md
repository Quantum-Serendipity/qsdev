# claude-history Deep Dive

## Overview

[claude-history](https://github.com/raine/claude-history) is a Rust terminal UI tool for fuzzy-searching and browsing Claude Code conversation history. Written by GitHub user `raine`, it provides an fzf-like interactive experience over the JSONL session files stored in `~/.claude/projects/`. As of March 2026 it has 110 stars, 10 forks, and is at version 0.1.49 with MIT license.

The tool occupies the "session search and browsing" niche in the Claude Code analysis tool ecosystem — its primary value proposition is answering "what did I work on?" and "where was that conversation?" across all projects, then enabling resume/fork directly from the results.

## Architecture

### Technology Stack

claude-history is built on the standard Rust TUI stack:

- **ratatui 0.30** — The TUI rendering framework (successor to tui-rs). Provides layout, widgets, and the rendering abstraction layer.
- **crossterm 0.29** — The terminal backend. Handles raw mode, event polling, cursor control, and cross-platform terminal I/O.
- **rayon 1.10** — Data-parallel iteration. Used for both conversation loading (parsing JSONL files in parallel) and search scoring (scoring all conversations in parallel).
- **pulldown-cmark 0.13** — Markdown parsing for in-terminal rendering of assistant responses.
- **syntect 5.2** — Syntax highlighting within code blocks in the conversation viewer.
- **bincode 1.3** — Binary serialization for the per-project cache system.
- **clap 4.5** — CLI argument parsing with derive macros.
- **terminal-light 1.0** — Automatic light/dark theme detection from terminal background.
- **arboard 3.4** — Clipboard access for copy operations.

Total dependency count is 20 direct crates. The project uses Rust edition 2024.

### Source Organization (~2,000+ LOC estimated)

```
src/
├── history/          # Data layer
│   ├── mod.rs        — Core types: Conversation, Project, LoaderMessage, ParseError
│   ├── cache.rs      — Per-project binary cache (bincode, atomic writes)
│   ├── loader.rs     — Streaming parallel loader with cache-aware partitioning
│   ├── parser.rs     — JSONL message parser (40+ unit tests)
│   └── path.rs       — Claude's path encoding/decoding (/ → -)
├── tui/              # Presentation layer
│   ├── app.rs        — Main event loop and state machine
│   ├── search.rs     — Fuzzy search engine with recency scoring
│   ├── viewer.rs     — Conversation viewer with markdown rendering
│   ├── ui.rs         — Layout and widget rendering
│   ├── theme.rs      — Terminal theme detection
│   └── export.rs     — File export functionality
├── main.rs           — Entry point, setting resolution, dispatch
├── cli.rs            — Argument definitions
├── config.rs         — TOML config loading, custom key binding parser
├── claude.rs         — Claude CLI integration (resume, fork via exec)
├── markdown.rs       — Markdown → terminal rendering pipeline
├── syntax.rs         — Syntax highlighting integration
├── tool_format.rs    — Tool output formatting/truncation
├── pager.rs          — less integration
├── display.rs        — Display formatting utilities
├── error.rs          — Error types
├── update.rs         — Self-update mechanism
└── debug*.rs         — Debug logging
```

The architecture is a clean two-layer design: `history/` handles all data concerns (discovering, parsing, caching, loading JSONL files) while `tui/` handles all presentation concerns (rendering, interaction, search). The `main.rs` orchestrates between them.

### How It Indexes: Cache-Accelerated Parallel Loading

claude-history does **not** use a database. Instead, it employs a cache-accelerated approach that reads raw JSONL files:

1. **Discovery**: On startup, it scans `~/.claude/projects/` for all project directories and their `.jsonl` files.

2. **Cache check**: For each project, it reads a per-project binary cache file from `~/.cache/claude-history/projects/`. Cache entries are validated against file modification time and size — if both match, the cached `Conversation` is used without re-parsing the JSONL.

3. **Parallel parsing**: Cache misses are parsed in parallel via rayon. The parser extracts: message content (user + assistant text), tool output content, timestamps, model name, token usage, custom titles, summaries, and working directory.

4. **Text pre-normalization**: During parsing, the full text of each conversation is lowercased and normalized (underscores, hyphens, slashes become spaces; CJK punctuation becomes spaces). This `search_text_lower` field is stored in the cache so normalization cost is paid once, not on every startup.

5. **Negative caching**: Files that parse to zero conversations (empty or warmup-only) get "empty" cache entries so they are not re-parsed on subsequent runs.

6. **Streaming to TUI**: The loader runs in a background thread, sending `LoaderMessage::Batch(Vec<Conversation>)` messages to the TUI as each project completes. This allows the UI to become interactive before all projects are loaded.

7. **Cache writes**: Updated caches are written atomically via `tempfile` to prevent corruption from concurrent access or crashes.

The cache format uses magic bytes and a schema version number. Version mismatches cause a full cache rebuild — this handles upgrades transparently.

### Search Algorithm

The search engine is a custom word-prefix fuzzy matcher, not a full-text search index:

1. **Query normalization**: The query is lowercased, underscores/hyphens/slashes become spaces, CJK punctuation becomes spaces.
2. **Word splitting**: Query is split on whitespace. **All words must match** (AND logic).
3. **Fast rejection**: For each query word, a simple `contains()` substring check is performed first. Rust's `str::contains()` uses SIMD-accelerated `memchr` internally, making this very fast.
4. **Word-prefix matching**: Each query word must appear at a word boundary (start of string or preceded by whitespace). This means typing "hard" matches "hardened" but not "sharder". This is the key UX insight — prefix matching at word boundaries gives fzf-like behavior without the complexity of a Levenshtein distance scorer.
5. **CJK fallback**: If query words contain CJK ideographs (Unicode range U+4E00–U+9FFF), substring matching is used instead of word-prefix matching, since CJK text lacks whitespace word boundaries. CJK matches get a 0.5x score penalty vs. prefix matches.
6. **Recency scoring**: Matched conversations are scored by recency: 3x multiplier for today, 2x for this week, 1.5x for this month, 1x for older. This ensures recent conversations bubble up for ambiguous queries.
7. **Parallel scoring**: The entire search is parallelized across conversations via rayon.

This approach trades off recall (you cannot find "runtime" by searching "ime") for speed and relevance. The design is optimized for interactive use where you remember approximate words from a conversation.

## Key Features

### Interactive TUI (List Mode)

Running `claude-history` opens a full-screen TUI showing all conversations sorted by recency. Key capabilities:

- **Fuzzy search**: Type to filter. Match context is highlighted when the search matches content not visible in the preview.
- **Scope toggle**: Tab key switches between all-projects (default) and current-workspace-only views. `-L/--local` flag starts in local mode.
- **Preview**: Each conversation shows a preview of first or last 3 messages. `--last` (default) vs `--first` controls which end.
- **Metadata display**: Timestamp (hybrid relative/absolute format), project name, message count, model name, total tokens, duration.
- **Vim-style navigation**: j/k or arrow keys, Ctrl+P/Ctrl+N.
- **UUID lookup**: If the search query is a UUID, it jumps directly to that session across all projects.

### Conversation Viewer

Pressing Enter opens the full conversation with:

- **Markdown rendering**: Assistant responses rendered as formatted markdown with syntax-highlighted code blocks (via pulldown-cmark + syntect).
- **Tool output modes**: `t` key cycles through Hidden/Truncated/Full display of tool calls. Truncated mode shows the tool header plus first few lines — a practical default that avoids huge `cat` outputs overwhelming the view.
- **Thinking blocks**: `T` key toggles extended thinking/reasoning step visibility.
- **Subagent display**: Subagent messages appear dimmed with `↳` prefix.
- **In-conversation search**: `/` opens search within the current conversation. `n`/`N` navigate matches.
- **Message navigation**: `J`/`K` jump between messages (not just scroll lines). Teal `▌` gutter marker shows current message.
- **Clipboard**: Copy support via arboard.

### Session Management

- **Resume**: `Ctrl+R` (or `--resume` flag) hands off to `claude --resume <id>`, replacing the current process via Unix `exec()`.
- **Fork**: `Ctrl+F` creates a new session branching from the selected one. For cross-project forks, it copies the `.jsonl` file and subagent directory to the target project.
- **Delete**: `Ctrl+X` deletes a session (with confirmation).
- **Export**: `e` key exports the conversation to a file.
- **Key rebinding**: All action keys configurable in `~/.config/claude-history/config.toml`.

### Plain Output Mode

`--plain` outputs raw text without TUI formatting, suitable for piping to other tools or LLMs. `--render` renders a specific JSONL file in ledger format and exits.

### Direct File Input

`claude-history /path/to/conversation.jsonl` bypasses the list interface and opens the viewer directly on a specific file.

## Configuration

Configuration lives at `~/.config/claude-history/config.toml` with three sections:

- **`[display]`**: `no_tools`, `last`, `show_thinking`, `plain`, `pager`
- **`[resume]`**: `default_args` — additional arguments passed to `claude --resume` (e.g., `["--dangerously-skip-permissions"]`)
- **`[keys]`**: `resume`, `fork`, `delete` — custom key bindings in "Ctrl+R" format

CLI flags override config values, which override defaults. The resolution uses explicit enable/disable flag pairs (e.g., `--show-tools` vs `--no-tools`) with a priority hierarchy.

## Tradeoffs and Limitations

### Strengths

- **No external database**: No SQLite, no Tantivy index, no embedding model. Reads JSONL directly with a binary cache for speed. This means zero setup, zero storage overhead beyond the cache, and no sync issues.
- **Instant interactivity**: Streaming loader lets you start searching before all projects are loaded.
- **Search quality**: Word-prefix matching with recency scoring hits the sweet spot for "I vaguely remember" queries. The AND logic and boundary matching prevent false positives.
- **Complete viewer**: Not just search — the built-in viewer with markdown rendering, syntax highlighting, and tool display modes means you rarely need to open files externally.
- **CJK support**: Thoughtful handling of CJK text and punctuation as word boundaries, with dedicated test coverage.
- **Cross-platform**: Works on macOS, Linux, and Windows (Windows compilation fixed in v0.1.48).

### Limitations

- **No semantic search**: Cannot find conceptually similar conversations by meaning. If you search "error handling" you will not find a conversation that discusses "exception management" unless those exact words appear. The tool trades semantic recall for speed and simplicity.
- **No full-text index**: Search scans all normalized text on every keystroke (albeit in parallel with SIMD-accelerated substring checks). For very large session collections (thousands of sessions, millions of lines), this could become sluggish compared to tools with pre-built inverted indices.
- **Memory usage on large histories**: Every conversation's full normalized text is held in memory (the `search_text_lower` field). For heavy Claude Code users with hundreds of long sessions, this could consume significant RAM. The `Conversation` struct stores `full_text`, `preview_first`, `preview_last`, and `search_text_lower` — four copies of overlapping content per session.
- **No MCP server**: Unlike ccrider, claude-history cannot be queried by Claude itself via MCP. It is purely a human-facing tool.
- **No analytics**: Provides no cost tracking, token usage aggregation, or trend analysis. It is focused solely on search and viewing.
- **Cache invalidation**: The cache validates on file size + mtime. If a file is modified without changing either (unlikely but possible with fast appends), stale cache data would be served.
- **Single-machine scope**: Reads only local `~/.claude/` files. No multi-machine aggregation or cloud sync.

### Performance Characteristics

- **Startup time**: Cold start (no cache) requires parsing all JSONL files, which is parallelized but proportional to total session data size. Hot start (all cached) should be fast since it just reads bincode files. The streaming loader mitigates cold-start UX by showing results incrementally.
- **Search latency**: Per-keystroke full scan of all normalized text. For moderate session counts (<500), this should be sub-frame. For very large collections, rayon parallelism helps but the linear scan could become noticeable.
- **Disk usage**: Cache files are bincode-serialized conversation metadata. Should be significantly smaller than raw JSONL since they exclude the raw JSON structure, but they do include full text content.

## Maturity Assessment

### Development Velocity

Extremely active: 10 releases in 11 days (March 13–24, 2026), going from v0.1.40 to v0.1.49. This suggests rapid iteration, likely driven by personal use and community feedback. The version numbers (0.1.x) appropriately signal pre-1.0 status.

### Code Quality Indicators

- 40+ unit tests in the parser module alone, with dedicated test coverage for CJK handling, UUID detection, and search scoring.
- Clean two-layer architecture (history/ data layer, tui/ presentation layer).
- Atomic cache writes prevent corruption.
- Graceful error handling with fatal vs. non-fatal error distinction.
- `deny_unknown_fields` on config parsing catches typos.

### Distribution

- **Homebrew tap**: `brew install raine/claude-history/claude-history` — mature distribution for macOS/Linux.
- **crates.io**: `cargo install claude-history` — standard Rust distribution.
- **Install script**: Quick curl-pipe-bash installer.
- No pre-built binaries on GitHub Releases (only source-based installation).

### Community

- 110 stars, 10 forks — modest but growing. Consistent with a focused utility tool.
- Primarily a single-developer project (raine).
- 0 open issues as of March 2026 — either low adoption or diligent maintenance.

## Comparison: Session Search Tools

### claude-history vs. search-sessions

| Dimension | claude-history | search-sessions |
|-----------|---------------|-----------------|
| **Language** | Rust | Rust |
| **Architecture** | TUI app with built-in viewer | CLI tool (no TUI beyond search) |
| **Search method** | Custom word-prefix fuzzy match with recency scoring | Index search (~18ms) + deep search via ripgrep (~280ms) |
| **Indexing** | Binary cache of parsed metadata | No index, no database — raw file search |
| **Viewer** | Full markdown-rendered conversation viewer | None — just finds sessions for `claude --resume` |
| **Database** | None (binary cache only) | None |
| **Resume** | Ctrl+R from TUI | Via session UUID output |
| **Fork** | Ctrl+F with cross-project copy | Not supported |
| **Stars** | 110 | ~4 (based on HN post) |
| **Installation** | Homebrew, Cargo, script | Homebrew, Cargo |

**Key difference**: claude-history is a comprehensive browsing and viewing experience. search-sessions is a minimal search utility that finds sessions and gets out of the way. search-sessions claims faster search by using ripgrep under the hood for deep search, while claude-history's custom search engine runs entirely in-process.

### claude-history vs. ccrider

| Dimension | claude-history | ccrider |
|-----------|---------------|---------|
| **Language** | Rust | Go |
| **Architecture** | TUI with streaming loader | TUI + CLI + MCP server (triple interface) |
| **Search method** | Custom fuzzy word-prefix | SQLite FTS5 full-text search |
| **Database** | None (binary cache) | SQLite with FTS5 |
| **MCP support** | No | Yes — Claude can query your session history |
| **Viewer** | Full markdown viewer with syntax highlighting | Session browser with markdown export |
| **Incremental updates** | Cache invalidation on file change | Detects new sessions, imports without re-processing |
| **Stars** | 110 | ~19 (based on HN post) |

**Key difference**: ccrider uses a persistent SQLite database with FTS5 for proper full-text search, which scales better for very large session collections but requires an explicit index. ccrider's MCP server is a unique capability — Claude itself can search your past sessions, enabling "what approach did we use last time?" queries within a session. claude-history's binary cache is simpler but means search is always a full scan.

### claude-history vs. cc-sessions

| Dimension | claude-history | cc-sessions |
|-----------|---------------|-------------|
| **Language** | Rust | Rust |
| **Architecture** | Full TUI with viewer | Minimal CLI (~350 LOC) |
| **Search method** | Fuzzy word-prefix | fzf picker (external dependency) |
| **Data source** | Parses JSONL files directly | Reads sessions-index.json metadata |
| **Viewer** | Built-in markdown viewer | None — preview via fzf |
| **Stars** | 110 | Small (HN: ~few) |

**Key difference**: cc-sessions is extremely minimal — it reads Claude's `sessions-index.json` rather than parsing JSONL files, which means it gets session metadata without any parsing overhead. But it cannot search within conversation content, only by session metadata. claude-history parses full JSONL content and makes all of it searchable.

### claude-history vs. ccsearch (madzarm)

| Dimension | claude-history | ccsearch |
|-----------|---------------|----------|
| **Search method** | Custom word-prefix fuzzy | SQLite FTS5 + local embedding model (all-MiniLM-L6-v2) |
| **Semantic search** | No | Yes — 80MB local embedding model |
| **Ranking** | Recency-weighted prefix score | Reciprocal Rank Fusion of keyword + semantic |
| **Dependencies** | Pure Rust, no ML runtime | Requires embedding model download |

**Key difference**: ccsearch is the only tool in this category with semantic search capability, at the cost of an 80MB model download and more complex architecture. claude-history prioritizes simplicity and speed over semantic understanding.

### Summary of Positioning

claude-history occupies the "opinionated integrated experience" position: it is both the search tool and the viewer, with thoughtful UX details (recency scoring, tool display modes, CJK support, theme detection). Its lack of a database is both its strength (simplicity, zero setup) and its limitation (linear scan, no semantic search, no MCP). For the typical Claude Code user with dozens to low-hundreds of sessions, it is likely the most pleasant daily-driver tool for "find that conversation."

## Failure Modes and Edge Cases

- **Corrupted JSONL**: The parser captures parse errors with context (2 lines before/after the error) and stores them in `Conversation.parse_errors`. Corrupted lines are skipped, not fatal. This is robust.
- **Very large session collections**: Linear scan with parallel scoring. At thousands of sessions with multi-MB each, search latency and memory usage could degrade. No built-in pagination or lazy loading of conversation text.
- **Cache corruption**: Binary cache uses magic bytes + version validation. Corrupted or version-mismatched caches cause full rebuild, not errors. Atomic writes via tempfile prevent partial writes.
- **Missing ~/.claude/**: The `get_claude_projects_root()` function sends a `LoaderMessage::Fatal` if the projects directory does not exist, which the TUI should handle gracefully.
- **Cross-project fork edge case**: Forking copies JSONL files and subagent directories. If the source project has been deleted or the JSONL is missing, the fork operation would fail.
- **Warmup messages**: The parser filters out "warmup" messages (first message saying "Warmup") and `/clear` metadata, preventing noise in search results.
- **Rapid session growth**: If Claude Code is actively writing to a JSONL file during search, the cache will be stale until the next launch (cache validates on file size + mtime at startup, not during the session).

## Real-World Usage and Community Reception

claude-history was not found in a dedicated Show HN submission, but it appeared in several community discussions about Claude Code session search tools. It was mentioned in the expanded catalog session of this research spike as one of the "most mature" tools in the session search category.

The tool appears on the "Top 50 Claude Skills and Github Repos" lists and "Best Claude Skills GitHub Repos" roundups, suggesting community recognition. The 110-star count is modest but places it among the more popular focused utilities (compared to broad-appeal tools like ccusage at 12k+ stars or claude-replay at 573 stars).

The rapid release cadence (10 releases in 11 days) suggests active personal use driving development — the author is likely eating their own dog food and fixing issues as they encounter them.

## Sources

- [GitHub: raine/claude-history](https://github.com/raine/claude-history) — repository, README, releases
- [Cargo.toml](https://raw.githubusercontent.com/raine/claude-history/main/Cargo.toml) → `docs/claude-history-cargo-toml.md`
- [Source architecture analysis](https://github.com/raine/claude-history/tree/main/src) → `docs/claude-history-src-architecture.md`
- [GitHub metadata](https://github.com/raine/claude-history) → `docs/claude-history-github-metadata.md`
- [search-sessions HN thread](https://news.ycombinator.com/item?id=47128630) → `docs/hn-search-sessions.md`
- [ccrider HN thread](https://news.ycombinator.com/item?id=46512501) → `docs/hn-ccrider.md`
- [cc-sessions HN thread](https://news.ycombinator.com/item?id=46805870) → `docs/hn-cc-sessions.md`
- [Nimbalyst session managers comparison](https://nimbalyst.com/blog/best-session-managers-for-claude-code-and-codex) → `docs/nimbalyst-session-managers-comparison.md`
