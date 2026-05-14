# Yaw Mode Technical Implementation Details
- **Source**: https://yaw.sh/blog/claude-code-yaw-mode
- **Retrieved**: 2026-05-14

## 1. Overlay Mechanism

Yaw Mode uses a **temporary directory overlay** rather than environment variables or CLI flags:

> "Creates a temp directory under your platform's tmp root, namespaced by process id + pty id so no two overlays can ever collide."

It then sets `CLAUDE_CONFIG_DIR=<overlay>` to redirect Claude Code's configuration source.

## 2. Augment vs Fresh Mode - Technical Differences

**Augment (default):** Layers the Yaw bundle atop your existing `~/.claude/` configuration. Your skills, agents, and settings participate alongside Yaw's additions.

**Fresh:** Ignores your `~/.claude/` for skills, agents, and CLAUDE.md, using only the Yaw bundle. Both modes route conversation transcripts through your home directory.

Both modes set `$YAW_MODE` environment variable to either "augment" or "fresh" for runtime detection.

## 3. Files and Directories Modified

The overlay creates/links:
- **Hardlinked files:** `settings.json`, `.credentials.json`, `history.jsonl`, `.claude.json`
- **Symlinked/junctioned directories:** `projects/`, `sessions/`, `plans/`, `file-history/`
- **Merged file:** CLAUDE.md with additions under heading "## Yaw Mode - added instructions"
- **Backup directory:** `~/.yaw-claude-json-backups/` (keeps 7 timestamped snapshots)

## 4. Environment Variables

- `CLAUDE_CONFIG_DIR=<overlay>` - Points Claude Code to temporary overlay
- `$YAW_MODE` - Set to "augment" or "fresh" for session identification

## 5. ~/.claude/ Manipulation

The overlay **preserves** your home directory through hardlinks and symlinks. Writes flow directly through to your real `~/.claude/`. Upon session exit, the overlay syncs `.claude.json` back with a stale-snapshot guard using additive merging.

## 6. Skill Layering Method

Skills load via symlinks/junctions in the overlay. The bundle ships seven skills and three sub-agents whose bodies load on-demand rather than remaining in the persistent system prompt.

## 7. No devenv/Nix/Shell Integration

The document contains no mentions of devenv, Nix, or shell integration mechanisms.
