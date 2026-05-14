# direnv Integration with Claude Code via Hooks
- **Source**: https://github.com/anthropics/claude-code/issues/42229
- **Retrieved**: 2026-05-14

## The Problem

Claude Code's Bash tool does NOT source `~/.bashrc`, which means:
- `direnv hook bash` installs a PROMPT_COMMAND hook that only fires in interactive shells (never runs in Claude)
- Project-specific tools from `.envrc` files are invisible to Claude
- Switching directories doesn't trigger environment reloading
- Claude inherits the parent shell's environment on startup, but directory changes don't trigger reloading

## Hook-Based Solution

Two components: a hook script and settings configuration.

### Hook Script: `~/.claude/hooks/devbox-and-direnv.sh`
```bash
#!/bin/bash
[ -n "$CLAUDE_ENV_FILE" ] || exit 0

ENV_SNAPSHOT="${CLAUDE_ENV_FILE}.snapshot"
if ! grep -qF "$ENV_SNAPSHOT" "$CLAUDE_ENV_FILE" 2>/dev/null; then
    echo ". \"$ENV_SNAPSHOT\"" >> "$CLAUDE_ENV_FILE"
fi

(direnv export bash 2>/dev/null; echo "true") > "$ENV_SNAPSHOT"
```

### Settings Configuration
```json
{
  "hooks": {
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "bash ~/.claude/hooks/devbox-and-direnv.sh || true" }] }],
    "CwdChanged": [{ "hooks": [{ "type": "command", "command": "bash ~/.claude/hooks/devbox-and-direnv.sh || true" }] }]
  }
}
```

## Key Architecture: Snapshot File Indirection Pattern

1. SessionStart hook: Loads env for the directory where claude was launched
2. CwdChanged hook: Reloads env whenever Claude changes directory
3. Snapshot file (`${CLAUDE_ENV_FILE}.snapshot`): Overwritten on each invocation
4. CLAUDE_ENV_FILE: Gets a single `. "/path/to/snapshot"` line (append-only)

## Environment Variable Caching

Claude Code caches environment variables from previous sessions in `~/.claude/session-envs/`. Cached entries with `;` suffixes can cause syntax errors.

## Critical Insight for gdev

The parent shell's environment IS inherited by Claude Code at startup. So if devenv/direnv sets environment variables before Claude Code launches, Claude Code WILL see them. The hook-based solution is only needed for directory changes DURING a session.
