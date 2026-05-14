<!-- Source: https://github.com/raine/claude-history/tree/main/src (multiple files) -->
<!-- Retrieved: 2026-03-26 -->

# claude-history Source Architecture

## Directory Structure (src/)

```
src/
├── history/
│   ├── mod.rs      — Module root: Conversation/Project/LoaderMessage structs, path helpers
│   ├── cache.rs    — Per-project binary cache (bincode format, atomic writes, negative caching)
│   ├── loader.rs   — Streaming loader with rayon parallelism, cache-aware parsing
│   ├── parser.rs   — JSONL parser: message extraction, metadata, token usage, 40+ tests
│   └── path.rs     — Path encoding/decoding utilities
├── tui/
│   ├── mod.rs      — Re-exports: Action, run_single_file, run_with_loader, render_conversation
│   ├── app.rs      — Main TUI application loop (ratatui + crossterm)
│   ├── export.rs   — Export conversations to files
│   ├── search.rs   — Fuzzy search engine: word-prefix matching, CJK support, recency scoring
│   ├── theme.rs    — Light/dark theme detection via terminal-light
│   ├── ui.rs       — UI layout and rendering
│   └── viewer.rs   — Conversation viewer with markdown rendering, tool display modes
├── claude.rs       — Claude CLI integration (resume, fork)
├── cli.rs          — clap argument definitions
├── config.rs       — TOML config loading, key binding parsing
├── debug.rs        — Debug output utilities
├── debug_log.rs    — Debug logging
├── display.rs      — Display formatting
├── error.rs        — Error types (AppError, Result)
├── main.rs         — Entry point, argument resolution, dispatch
├── markdown.rs     — Markdown rendering (pulldown-cmark + syntect)
├── pager.rs        — Pager integration (less)
├── syntax.rs       — Syntax highlighting
├── tool_format.rs  — Tool output formatting
└── update.rs       — Self-update functionality
```

## Key Dependencies

- **ratatui 0.30** + **crossterm 0.29** — TUI framework + terminal backend
- **rayon 1.10** — Parallel iteration for loading and search
- **serde/serde_json** — JSON parsing
- **bincode 1.3** — Binary serialization for cache
- **pulldown-cmark 0.13** — Markdown to terminal rendering
- **syntect 5.2** — Syntax highlighting in code blocks
- **clap 4.5** — CLI argument parsing
- **arboard 3.4** — Clipboard access
- **terminal-light 1.0** — Light/dark theme detection
- **indicatif 0.17** — Progress indicators during loading
- **tempfile 3** — Atomic cache writes
- **chrono 0.4** — Timestamp handling

## Conversation Struct (from history/mod.rs)

```rust
pub struct Conversation {
    pub path: PathBuf,
    pub index: usize,
    pub timestamp: DateTime<Local>,
    pub preview: String,
    pub preview_first: String,    // First 3 messages
    pub preview_last: String,     // Last 3 messages
    pub full_text: String,
    pub search_text_lower: String, // Pre-normalized for search
    pub project_name: Option<String>,
    pub project_path: Option<PathBuf>,
    pub cwd: Option<PathBuf>,
    pub message_count: usize,
    pub parse_errors: Vec<ParseError>,
    pub summary: Option<String>,
    pub custom_title: Option<String>,
    pub model: Option<String>,
    pub total_tokens: u64,
    pub duration_minutes: Option<u64>,
}
```

## Search Algorithm (from tui/search.rs)

1. Query is normalized: lowercase, underscores/hyphens/slashes become spaces, CJK punctuation becomes spaces
2. Query split into words (AND logic - all must match)
3. Fast rejection: substring check via `contains()` (SIMD-accelerated memchr)
4. Word-prefix matching: each query word must appear at a word boundary in the text
5. CJK fallback: substring matching for queries containing CJK characters (no word boundaries in CJK)
6. Recency scoring: 3x for today, 2x for this week, 1.5x for this month, 1x for older
7. Parallel scoring via rayon

## Cache System (from history/cache.rs)

- Per-project binary cache files at `~/.cache/claude-history/projects/`
- Format: magic bytes + schema version + HashMap<filename, CacheEntry> in bincode
- Validation: file size + modification time comparison
- Negative caching: empty entries for files that parse to zero conversations
- Atomic writes via tempfile to prevent corruption
- Cache miss triggers full JSONL parse; cache hit restores Conversation from cached fields

## Loader (from history/loader.rs)

- Streaming architecture: background thread sends LoaderMessage variants to TUI
- Per-project parallelism with rayon
- Files partitioned into cache hits/misses; only misses parsed
- Deterministic ordering maintained
- Fatal vs. non-fatal error distinction
