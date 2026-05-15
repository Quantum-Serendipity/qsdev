# reasoning-core enable-in-repo.sh

- **Source**: https://github.com/jakubkrzysztofsikora/reasoning-core/blob/main/scripts/enable-in-repo.sh
- **Retrieved**: 2026-05-15
- **Note**: Content returned via WebFetch AI summary — may not be verbatim

---

## Purpose
Enables reasoning-core hooks for a target repository.

## Steps
1. **Validation**: Checks that `RC_REPO` environment variable is set and points to a valid reasoning-core installation with `src/hooks/` directory present.
2. **Safety checks**: Refuses to overwrite existing `.envrc` or `.claude/settings.local.json` files.
3. **File generation**: Creates two configuration files:
   - `.envrc`: Sets `RC_REPO` path and exports configuration variables including `S2_DEVICE`, `S2_PORT`, `S2_FAIL_CLOSED`, hook policy flags (`RC_PLAN_BLOCK`, `RC_SHADOW_MODE`), and optional iter-3 levers
   - `.claude/settings.local.json`: Registers hook commands for various lifecycle events (PreToolUse, PostToolUse, SessionStart, UserPromptSubmit, PreCompact) and configures the hybrid-reasoner MCP server
4. **Git integration**: Appends `.envrc.local` and `.claude/settings.local.json` to `.gitignore` if the file exists.
5. **User guidance**: Displays instructions for enabling direnv, confirming sidecar health, and optionally activating iter-3 features.

Uses heredocs to generate portable configuration that resolves hook paths through the `$RC_REPO` environment variable.
