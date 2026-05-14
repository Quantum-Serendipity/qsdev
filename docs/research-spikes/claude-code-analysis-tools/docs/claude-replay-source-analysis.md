<!-- Source: https://github.com/es617/claude-replay (source code analysis) -->
<!-- Retrieved: 2026-03-26 -->

# claude-replay Source Code Analysis

## Repository Metadata (from GitHub API)
- **Stars**: 573 | **Forks**: 33 | **Open Issues**: 1
- **Language**: JavaScript | **Size**: 8,090 KB
- **Created**: 2026-03-02 | **Last Push**: 2026-03-24
- **License**: MIT
- **Node.js**: >=18 | **Version**: 0.6.1
- **Contributors**: es617 (138 commits), ShriPunta (1 commit) — essentially solo project
- **Topics**: agents, claude-code, codex, cursor
- **DevDependencies**: esbuild, @playwright/test (no runtime deps)

## Release History (12 releases in ~3 weeks)
| Tag | Date | Key Changes |
|-----|------|-------------|
| v0.1.1 | 2026-03-06 | Initial release |
| v0.1.2 | 2026-03-07 | Bug fixes |
| v0.2.0 | 2026-03-08 | Diff view, parser improvements |
| v0.3.0 | 2026-03-11 | Extract subcommand, keyboard shortcuts, redaction |
| v0.4.0 | 2026-03-13 | Web editor UI, Codex CLI support, session ID lookup |
| v0.4.1 | 2026-03-14 | Bug fixes |
| v0.5.0 | 2026-03-14 | Major UI overhaul: 3-panel editor, session chaining (up to 20) |
| v0.5.1 | 2026-03-16 | Fixes |
| v0.5.2 | 2026-03-17 | Fixes |
| v0.5.3 | 2026-03-18 | Fixes |
| v0.6.0 | 2026-03-20 | HTTP serve mode, file watching, static website |
| v0.6.1 | 2026-03-24 | Turn skip buttons, responsive design, mtime sorting |

## File Structure
```
bin/claude-replay.mjs          - CLI entry point
src/parser.mjs        (24KB)   - JSONL transcript parser (Claude Code, Cursor, Codex)
src/renderer.mjs       (7KB)   - HTML generation with compression
src/editor-server.mjs (28KB)   - Local HTTP server for web editor
src/browser.mjs        (5KB)   - Browser-compatible export layer
src/resolve-session.mjs (4KB)  - Session ID → file path resolution
src/secrets.mjs        (2KB)   - Secret detection and redaction
src/themes.mjs         (6KB)   - Theme definitions and CSS generation
src/extract.mjs        (3KB)   - Extract data from generated HTML replays
template/player.html  (86KB)   - HTML player template (vanilla JS)
template/editor.html  (65KB)   - HTML editor template
```

## Parser Architecture (parser.mjs — the core)

### Format Detection
`detectFormatFromText()` peeks at first JSONL line to identify format:
- `type: "user"|"assistant"` → Claude Code
- `role: "user"|"assistant"` → Cursor
- `type: "session_meta"` → Codex CLI
- `user_text + blocks` → Replay (re-import)

### Claude Code / Cursor Parsing Pipeline
1. `parseJsonl()` — reads lines, filters to user/assistant entries, normalizes Cursor format to Claude Code shape
2. Main loop in `parseTranscriptFromText()`:
   - Finds user messages, extracts text via `extractText()`
   - `cleanSystemTags()` strips `<system-reminder>`, `<ide_opened_file>`, `<local-command-caveat>`, `<command-message>`, etc.
   - Absorbs consecutive non-tool-result user messages into same turn
   - `collectAssistantBlocks()` gathers text/thinking/tool_use blocks with deduplication via `seenKeys` Set
   - `attachToolResults()` matches tool_result blocks to tool_use blocks by `tool_use_id`
   - Orphan assistant blocks merge into previous turn
3. Post-processing: filters empty turns, re-indexes, heuristic Cursor thinking detection (all but last text block per turn → thinking)

### Codex CLI Parsing
Separate `parseCodexTranscript()` handles event-based format:
- `task_started`/`task_complete` events define turn boundaries
- `exec_command` → normalized to "Bash" tool name
- `apply_patch` → parsed into Edit/Write operations via `parseCodexPatch()`
- Handles sessions ending without `task_complete`

### Key Data Structure: Turn
```javascript
{
  index: number,           // 1-based
  user_text: string,       // cleaned user message
  blocks: AssistantBlock[], // text, thinking, tool_use blocks
  timestamp: string,       // ISO 8601
  system_events?: string[] // background task notifications
}
```

## Renderer Architecture (renderer.mjs)

1. `turnsToJsonData()` — serializes turns, optionally applying `redactSecrets()` + custom redaction rules
2. `compressForEmbed()` — deflate + base64 encoding (~60-70% size reduction)
3. `render()` — reads player.html template, replaces placeholders:
   - `__THEME_CSS__` → generated CSS from theme
   - `__TURNS_DATA__` → compressed JSON blob
   - `__BOOKMARKS_DATA__` → compressed bookmarks
   - `__SPEED__`, `__SHOW_THINKING__`, `__SHOW_TOOL_CALLS__`, etc.
   - Careful ordering to prevent content-in-conversation matching placeholders

Player uses vanilla JavaScript + browser-native `DecompressionStream` API. Zero external frameworks.

## Editor Server (editor-server.mjs)

Full local HTTP server with:
- Session discovery across Claude Code, Cursor, Codex directories
- In-memory session store with autosave to `~/.claude-replay/autosave/`
- 10 MB request body limit
- CSRF protection via Origin header checking
- Path traversal prevention
- API endpoints: /api/sessions, /api/search, /api/themes, /api/browse, /api/load, /api/import, /api/edit, /api/export-data, /api/preview, /api/export, /api/reset

## Secret Redaction (secrets.mjs)

Detects and replaces:
- Private keys (RSA, EC, DSA, OPENSSH)
- AWS access key IDs (AKIA...)
- Anthropic API keys (sk-ant-...)
- Generic sk-/key- prefixed secrets
- Bearer tokens, JWTs
- Connection strings (mongodb://, postgres://, etc.)
- Key=value patterns (api_key=, auth_token=, etc.)
- Environment variables (PASSWORD=, TOKEN=, etc.)
- Standalone hex tokens (40+ chars)

Recursive redaction via `redactObject()` walks entire turn tree.

## Test Suite
- 10 test files covering parser, renderer, CLI, editor-server, extract, secrets, themes, session resolution, concatenation, format validation
- 7 fixture JSONL files (Claude Code, Cursor, Codex, edge cases)
- Playwright E2E tests
- Both agent and human smoke test documentation
- `validate-format.mjs` (10KB) — dedicated format validation

## Issues History (14 total, 13 closed, 1 open)
- #1: PII persists in compressed data blob after JS-level turn filtering (CLOSED — security fix)
- #4: extract fails on own output — minified names regex issue (CLOSED)
- #6: Adding VSCode Github Copilot Chat parsing and rendering (OPEN — feature request)
- #8: Windows path separator bug in editor (CLOSED)
- Other issues: feature requests (time-warp, turn jumping, sorting, Docker, etc.)
