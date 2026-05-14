# Deep Dive: claude-replay

## Summary

claude-replay is a zero-dependency JavaScript tool that converts AI coding agent session transcripts (Claude Code, Cursor, Codex CLI) into self-contained, interactive HTML replays. Created by es617, it has reached 573 stars and 12 releases in under a month, making it the most popular tool in the session replay category. The architecture is clean: a JSONL parser extracts structured turns from three different agent formats, a renderer compresses and embeds them into a vanilla-JS HTML template, and an optional local editor server provides a GUI for browsing, editing, and exporting sessions. The output is a single HTML file with no external dependencies, suitable for embedding in blogs, sharing via email, or hosting anywhere.

## Architecture

### Processing Pipeline

```
JSONL Session File(s)
        │
        ▼
┌─────────────────┐
│  Format Detect   │  detectFormatFromText() peeks at first line
│  (parser.mjs)    │  → claude-code | cursor | codex | replay
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Parse → Turns   │  Format-specific parsers normalize to Turn[]
│  (parser.mjs)    │  Each Turn = user_text + AssistantBlock[]
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Filter/Pace     │  --turns, --from/--to, --timing paced
│  (parser.mjs)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Redact Secrets  │  12 regex patterns + custom --redact rules
│  (secrets.mjs)   │  Recursive walk of entire turn tree
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Render HTML     │  deflate + base64 compress JSON (~60-70%)
│  (renderer.mjs)  │  Inject into player.html template
└────────┬────────┘
         │
         ▼
   Self-contained HTML
   (vanilla JS player)
```

### Key Source Files

| File | Size | Role |
|------|------|------|
| `src/parser.mjs` | 24 KB | Core JSONL parser — handles 3 agent formats + re-import |
| `src/editor-server.mjs` | 28 KB | Local HTTP server for web editor GUI |
| `src/renderer.mjs` | 7 KB | HTML generation with compression and template injection |
| `src/browser.mjs` | 5 KB | Browser-compatible export layer for web usage |
| `src/themes.mjs` | 6 KB | 6 built-in themes + custom theme support |
| `src/resolve-session.mjs` | 4 KB | Session ID resolution across Claude/Cursor/Codex dirs |
| `src/secrets.mjs` | 2 KB | Automatic secret detection and redaction |
| `src/extract.mjs` | 3 KB | Reverse operation: extract data from generated HTML |
| `template/player.html` | 86 KB | Vanilla JS player template |
| `template/editor.html` | 65 KB | Web editor template |

### Tech Stack

- **Language**: Pure JavaScript (ES modules, `.mjs` extension throughout)
- **Runtime**: Node.js >= 18
- **Runtime dependencies**: Zero. Uses only Node.js built-ins (`fs`, `path`, `os`, `zlib`, `http`, `crypto`, `child_process`)
- **Dev dependencies**: esbuild (template minification), Playwright (E2E tests)
- **Output**: Vanilla JavaScript + CSS in a single HTML file. Uses browser-native `DecompressionStream` API for runtime decompression. No React, no frameworks, no CDN links.

### Parser Internals

The parser is the most sophisticated component. It handles three fundamentally different JSONL formats:

**Claude Code**: Standard `{type: "user"|"assistant", message: {role, content}, timestamp}` entries. The parser:
1. Strips system tags (`<system-reminder>`, `<ide_opened_file>`, `<local-command-caveat>`, `<command-message>`, etc.) via `cleanSystemTags()`
2. Groups consecutive user messages into a single turn
3. Collects assistant blocks (text, thinking, tool_use) with deduplication via content hashing into a `seenKeys` Set — this handles Claude Code's streaming format where the same block may appear in multiple JSONL entries as it's being generated
4. Matches tool results to tool uses by `tool_use_id`
5. Merges orphan assistant blocks (those not preceded by a user message) into the previous turn

**Cursor**: Similar structure but uses `{role, message: {content}}` format. Normalized to Claude Code shape during parsing. Post-processing heuristic: in each turn, all text blocks except the last are reclassified as "thinking" (Cursor doesn't distinguish thinking from response text in its logs).

**Codex CLI**: Event-based format with `task_started`/`task_complete` boundaries. Requires extensive normalization:
- `exec_command` → mapped to "Bash" tool name, `cmd` → `command`
- `apply_patch` → parsed into Edit/Write operations via dedicated patch parser that handles Add File / Update File / context lines
- `commentary` phase → thinking blocks, `final_answer` phase → text blocks
- Handles sessions ending without `task_complete` event

### Normalized Data Model

All three formats are normalized to:

```
Turn {
  index: number          // 1-based sequential
  user_text: string      // cleaned user message text
  blocks: AssistantBlock[]
  timestamp: string      // ISO 8601
  system_events?: string[] // background task notifications
}

AssistantBlock {
  kind: "text" | "thinking" | "tool_use"
  text: string
  tool_call: {
    tool_use_id: string
    name: string         // e.g. "Bash", "Read", "Edit", "Write"
    input: object
    result: string | null
    resultTimestamp: string | null
    is_error: boolean
  } | null
  timestamp: string | null
}
```

### Rendering Pipeline

The renderer (`renderer.mjs`) takes turns and produces a self-contained HTML file:

1. **Serialization**: Turns → JSON with selective field preservation
2. **Redaction**: Optional automatic secret detection (12 regex patterns) + custom `--redact` rules applied recursively to all strings in the turn tree
3. **Compression**: `zlib.deflateSync()` → base64 encoding, achieving ~60-70% size reduction
4. **Template injection**: Placeholders in `player.html` replaced with data blobs, theme CSS, speed settings, visibility toggles, metadata. Careful ordering prevents conversation content from matching placeholder strings.
5. **Player**: At runtime, browser decompresses using native `DecompressionStream` API. All rendering is vanilla JavaScript — no frameworks.

### Editor Server

The editor server (`editor-server.mjs`, 28 KB) is a standalone Node.js HTTP server providing:

- **Session discovery**: Walks `~/.claude/projects/`, `~/.cursor/projects/`, `~/.codex/sessions/` to find all sessions
- **Session search**: Full-text search across session content with snippet extraction
- **Turn editing**: Modify user text in turns before export
- **Live preview**: Real-time HTML rendering as edits are made
- **Autosave**: Throttled (max 1 save per 2 seconds) to `~/.claude-replay/autosave/`
- **Import/Export**: Can import previously-generated HTML replays and re-export
- **Security**: CSRF protection via Origin header checking, path traversal prevention, 10 MB request body limit

### Extract (Reverse Operation)

The `extract.mjs` module reverses the rendering: given a generated HTML file, it finds the compressed data blobs (matching `await <fn>("...")`), decompresses them (handles both compressed and `--no-compress` raw JSON), and returns the original turns and bookmarks. This enables round-tripping: generate → edit → re-generate.

## Key Features

### Playback & Navigation
- Interactive playback with speed control (0.5x to 5x)
- Block-by-block step-through (arrow keys / H/L vim bindings)
- Turn-level jumping (Shift+arrows, turn skip buttons)
- Thinking/tool block jumping (T / Shift+T)
- Progress bar with time tracking
- Collapsible thinking blocks and tool call sections
- Bookmarks/chapters for marking important moments

### Timing Modes
- **auto**: Uses real timestamps if available, falls back to paced
- **real**: Preserves original transcript timestamps (shows actual time gaps)
- **paced**: Synthetic timing based on content length (30ms per character, clamped to 1-10 seconds per block)

### Multi-Format Support
- Claude Code JSONL (primary target)
- Cursor IDE transcripts (with thinking heuristic)
- Codex CLI event logs (with tool name normalization)
- Round-trip: can re-import its own HTML output

### Session Management
- Auto-discovery of sessions from standard directories
- Session ID lookup (partial UUID matching for Codex)
- Session chaining: combine up to 20 sessions into one replay, chronologically sorted
- Turn filtering: by index range, exclusion list, or timestamp window

### Security & Privacy
- Automatic secret redaction (12 pattern categories — see Architecture section)
- Custom redaction rules via `--redact "text"` CLI flag
- Visual redaction rule configuration in web editor
- Editor runs on 127.0.0.1 only, never modifies original session files
- CSRF protection on editor endpoints

### Theming
- 6 built-in themes: tokyo-night (default), monokai, solarized-dark, github-light, dracula, bubbles
- Custom theme JSON files
- Terminal-style bottom-to-top scrolling option

### Deployment Options
- CLI: `claude-replay session.jsonl -o replay.html`
- npx: `npx claude-replay` (no install)
- Docker: containerized with read-only volume mount
- Web editor: `claude-replay` (no args, launches browser GUI)
- Live mode: `--serve --watch` for real-time session monitoring
- Embedding: self-contained HTML works in `<iframe>`

## Tradeoffs and Limitations

### Strengths
1. **Zero runtime dependencies** — both the tool itself (Node.js built-ins only) and the output (vanilla JS HTML) have no external deps. This is a deliberate and well-executed design choice.
2. **Multi-format support** — Claude Code, Cursor, and Codex CLI parsing in one tool. The Codex parser is particularly impressive given the complexity of Codex's event-based format.
3. **Self-contained output** — a single HTML file that works offline, in email, in iframes. No server needed to view.
4. **Comprehensive secret redaction** — 12 regex pattern categories with recursive object walking. Addresses a real concern (Issue #1 was about PII in compressed blobs).
5. **Rapid iteration** — 12 releases in 22 days shows responsive development. Issues are addressed quickly (most closed within days).
6. **Good test coverage** — 10 test files + E2E tests + fixture files for all 3 formats + format validation module.

### Limitations
1. **Single developer** — es617 has 138 of 139 total commits. Bus factor of 1. No indication of organizational backing.
2. **Very new** — created 2026-03-02, less than a month old. Rapid release cadence is impressive but also means the API is still evolving (already at v0.6.x with breaking changes between majors).
3. **JavaScript only** — while zero-dep is admirable, the tool requires Node.js 18+. Can't be used in environments without Node.
4. **10 MB editor body limit** — the editor server caps request bodies at 10 MB. Very large sessions (which can be 100+ MB with full tool results) may hit this limit during editing, though CLI-based generation reads from filesystem directly.
5. **Output size scales with session** — while compression achieves 60-70% reduction, a large session with many tool results still produces a large HTML file. No streaming or pagination in the player; the entire session is loaded into memory at once.
6. **No search within replays** — the player provides navigation (playback, stepping, bookmarks) but not text search across the session content.
7. **Cursor thinking heuristic** — since Cursor doesn't distinguish thinking from response text, the tool uses a heuristic (all but last text block = thinking). This will misclassify in some cases.
8. **Browser compatibility** — uses `DecompressionStream` API which is available in modern browsers but not in older ones. `--no-compress` fallback exists but produces larger files.

### Known Issues
- **Issue #1 (closed)**: PII persisted in compressed data blob even after JS-level turn filtering. Fixed by ensuring redaction happens before compression.
- **Issue #4 (closed)**: `extract` command failed on its own minified output due to regex expecting unminified function names. Fixed.
- **Issue #6 (open)**: Request for VS Code GitHub Copilot Chat parsing — indicates demand for more format support.
- **Issue #8 (closed)**: Windows path separator bug in editor. Fixed, but indicates cross-platform testing gaps that may recur.

## Maturity Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| **Code quality** | High | Clean ES module architecture, consistent patterns, comprehensive test suite, good separation of concerns |
| **Feature completeness** | High | Covers the replay use case thoroughly: parsing, filtering, timing, theming, redaction, editing, exporting |
| **Stability** | Medium | Rapid iteration (12 releases in 22 days) — functional but API still evolving. Version 0.x signals pre-1.0 |
| **Community** | Low | 573 stars but only 1 external contributor. Bus factor = 1. No governance, no contributing guide |
| **Documentation** | Medium-High | Good README with examples, but no architectural docs. Changelog exists but is summary-level |
| **Cross-platform** | Medium | Windows path bug was fixed but late. Primary development appears Linux/macOS |
| **Security posture** | Medium-High | Secret redaction, CSRF protection, path traversal prevention, localhost-only server. Issue #1 showed initial gap but was addressed |
| **Maintenance** | High (current) | Very active as of March 2026. Risk is sustainability given single developer |

### Stars/Activity Context
- 573 stars in ~24 days is strong growth
- 138 commits from a single developer in 24 days = ~5.75 commits/day average
- 13 of 14 issues closed, most within 1-3 days
- Active response to community feedback (Cursor support, Docker, Codex support all added in response to requests)

## Comparison to Alternatives in Replay/Visualization Category

### vs. claude-code-transcripts (Simon Willison)
- **Approach**: Python-based, generates paginated HTML with Gist publishing. Static output, no playback controls.
- **claude-replay advantage**: Interactive playback, timing simulation, multi-format support, web editor.
- **transcripts advantage**: Simpler output (just formatted conversation), Python ecosystem (wider install base), Gist integration for sharing.
- **Use case split**: claude-replay for demos/presentations where step-through matters; transcripts for archival/reference where you want to read the whole conversation.

### vs. Mantra
- **Approach**: Rust + React desktop app with Git time travel, showing actual file diffs alongside the conversation.
- **Mantra advantage**: Deep Git integration (see file state at each point), native desktop app with full system access, richer visualization of code changes.
- **claude-replay advantage**: Zero-install (npx), self-contained shareable HTML output, multi-format support. Mantra requires install and can't produce shareable artifacts the same way.
- **Use case split**: Mantra for deep post-hoc analysis of what the agent did to your codebase; claude-replay for sharing sessions with others.

### vs. CCViewer / cclogviewer / claude-code-log
- **Approach**: Various JSONL-to-HTML/Markdown converters. Static output, no playback.
- **claude-replay advantage**: Interactive playback, compression, redaction, theming, editing. More polished and featureful.
- **These tools' advantage**: Simpler, often single-file scripts. Good for quick one-off conversion without the playback overhead.

### vs. Claude DevTools
- **Approach**: Desktop app with token attribution, compaction visualization, multi-pane inspection.
- **DevTools advantage**: Deeper analytical capabilities (7-category token breakdown, compaction events, cost analysis). Built for understanding token usage, not sharing sessions.
- **claude-replay advantage**: Shareable output, playback simulation, multi-format support. Built for communication, not analysis.
- **Use case split**: DevTools for debugging/optimizing your own sessions; claude-replay for showing sessions to others.

### Unique Position
claude-replay occupies a distinct niche: **shareable, interactive session replays**. Most other tools are either viewers (static output) or analyzers (local-only deep inspection). The combination of interactive playback + self-contained HTML output + zero dependencies is unique in the ecosystem. The closest alternative for sharing is claude-code-transcripts, but it lacks the playback/timing dimension.

## Failure Modes

### When It Breaks
1. **Malformed JSONL**: The parser uses `try/catch` on each line and silently skips unparseable lines. This is resilient but means silently dropped data if a session has corruption.
2. **Very large sessions**: No streaming in the parser or player. A session with thousands of turns and full file contents in tool results could produce HTML files of 50+ MB that browsers struggle with.
3. **Missing timestamps**: Falls back to paced timing, but "real" timing mode will produce nonsensical output if timestamps are absent or malformed.
4. **Concurrent session writes**: Reading a JSONL file while Claude Code is actively writing to it may produce incomplete final entries. The `--serve --watch` mode handles this but standalone generation doesn't explicitly guard against it.
5. **DecompressionStream unavailability**: Older browsers (pre-2022) don't support this API. The `--no-compress` flag works around this but users have to know to use it.
6. **Content matching placeholders**: The renderer guards against conversation content matching template placeholders (like `__TURNS_DATA__`) by using careful replacement ordering, but this is a known fragile pattern.

### Edge Cases Documented in Tests
- Codex sessions ending without `task_complete` event
- Apply_patch with Add File vs Update File
- System tags in user messages (task notifications, IDE context)
- Empty turns from slash commands
- Orphan assistant blocks without preceding user message
- Consecutive user messages (CLI command sequences)
- Tool results with `<tool_use_error>` wrappers

## Real-World Usage

### From HN Discussion (47276604)
The creator's motivation: "I got tired of sharing AI demos with terminal screenshots or screen recordings." The HN discussion showed genuine enthusiasm, with users identifying these use cases:
- **Team onboarding**: New engineers watch how experienced devs use Claude Code
- **Prompting technique demos**: Share effective prompt patterns interactively
- **Hardware project workflows**: Embedded developers showing Claude Code controlling UART peripherals
- **Educational contexts**: Non-technical stakeholders stepping through AI reasoning
- **Bug reports**: Interactive transcripts instead of static logs

Feature requests from the community (many subsequently implemented):
- Cursor IDE support (added in v0.3/v0.4)
- Keyboard shortcuts for thinking/tool block jumping (added)
- Turn skip buttons (added in v0.6.1)
- Session search/discovery (added via web editor)
- Slack integration (not yet implemented)

### Deployment Patterns
The tool is used in three main modes:
1. **CLI generation**: Power users pipe sessions through the CLI for blog posts and documentation
2. **Web editor**: Teams use the editor for browsing, curating, and exporting sessions
3. **Live monitoring**: `--serve --watch` for watching remote agent sessions in real-time

## Sources

- [GitHub README](https://github.com/es617/claude-replay) → `docs/claude-replay-readme.md`, `docs/github-claude-replay-readme.md`
- [HN Discussion](https://news.ycombinator.com/item?id=47276604) → `docs/hn-claude-replay.md`
- [Source code analysis](https://github.com/es617/claude-replay/tree/main/src) → `docs/claude-replay-source-analysis.md`
- Direct source code review of: `parser.mjs`, `renderer.mjs`, `editor-server.mjs`, `browser.mjs`, `resolve-session.mjs`, `secrets.mjs`, `extract.mjs`, `package.json`, `CHANGELOG.md`
