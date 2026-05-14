<!-- Source: https://github.com/matt1398/claude-devtools (multiple pages) -->
<!-- Retrieved: 2026-03-26 -->

# Claude DevTools — Source Architecture & Issue Tracker

## Source Code Structure

### src/main/ (Electron Main Process)
- **constants/** — Configuration and constant values
- **http/** — HTTP server (for standalone/Docker mode)
- **ipc/** — Inter-process communication handlers (main ↔ renderer)
- **services/** — Business logic: JSONL parsing, context reconstruction, SSH, file watching
- **types/** — TypeScript type definitions
- **utils/** — Helper functions
- **index.ts** — Electron desktop entry point
- **standalone.ts** — Standalone/Docker server entry point

### src/preload/ (Context Bridge)
- Secure IPC bridge between main and renderer processes

### src/renderer/ (React UI)
- React 18 + Vite + TailwindCSS
- State management: Zustand

### src/shared/ (Cross-process)
- **constants/** — Shared constants
- **types/** — Shared TypeScript interfaces
- **utils/** — Shared utility functions

### Build Configuration (electron.vite.config.ts)
- Three build targets: main (CJS), preload (CJS), renderer (browser)
- Custom Rollup plugin stubs out .node native addon imports (ssh2, cpu-features have optional native bindings)
- Bundles production deps into main process output to avoid pnpm symlink issues with electron-builder asar
- Path aliases: @main, @shared, @preload, @renderer

## Security (SECURITY.md)
- **Zero outbound network calls** to third-party servers
- No telemetry or tracking
- Only network: GitHub Releases API (auto-updater, Electron only), SSH (user-initiated), HTTP server (127.0.0.1/0.0.0.0 when enabled)
- Standalone/Docker: auto-updater and SSH disabled entirely
- Docker `--network none` supported for full isolation
- Input validation with strict path containment checks
- File access constrained to project root and ~/.claude/
- Never writes to session files; Docker uses read-only volume mounts
- Config stored at ~/.claude/claude-devtools-config.json

## Open Issues (as of 2026-03-26)

### Bugs
- **#142** — Settings not applied on startup
- **#138** — Stale session name after /rename
- **#132** — Web interface not working when accessing Docker from another computer on network
- **#96** — Cannot resume remote SSH connection after interruption

### Feature Requests
- **#144** — Tauri version created by community member (1/10th the size of Electron)
- **#140** — Support /rename session command in UI
- **#139** — Native session delete
- **#131** — --append-system-prompt support
- **#130** — Display if skill loaded
- **#124** — Mermaid diagram visualization
- **#107** — Remote access from mobile/LAN (Tailscale, phone)
- **#101** — Writing input messages via UI (CLOSED — out of scope)

### Recently Closed
- **#123** — Windows drive letter casing causing missing sessions (fixed v0.4.9)
- **#121** — Task notifications rendered as styled cards (implemented)
- **#119** — Thinking context not visible (fixed)
- **#95** — Slow & Unresponsive (closed — likely addressed in perf optimizations)

## Notable: Tauri Fork (Issue #144)
Community member created a Tauri-based port with 1/10th the binary size. Indicates Electron bundle size is a known concern.
