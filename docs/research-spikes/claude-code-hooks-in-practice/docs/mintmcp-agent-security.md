# Agent-Security: Secrets Scanning for Claude Code and Cursor
- **Source**: https://github.com/mintmcp/agent-security
- **Retrieved**: 2026-03-27
- **Type**: GitHub repository

## Installation Methods

Three deployment approaches:

1. **Claude Code Plugin Marketplace** (easiest): `/plugin marketplace add mintmcp/agent-security` then `/plugin install secrets-scanner@agent-security`
2. **PyPI with Manual Hooks**: `pipx install claude-secret-scan` or `python3 -m pip install --user claude-secret-scan`
3. **Cursor Integration**: Copy configuration to `~/.cursor/hooks.json`

## Configuration for Claude Code

When installing via PyPI, update `~/.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {"hooks": [{"type": "command", "command": "claude-secret-scan --mode=pre"}]}
    ],
    "PreToolUse": [
      {"matcher": "Read|read", "hooks": [{"type": "command", "command": "claude-secret-scan --mode=pre"}]}
    ],
    "PostToolUse": [
      {"matcher": "Read|read", "hooks": [{"type": "command", "command": "claude-secret-scan --mode=post"}]},
      {"matcher": "Bash|bash", "hooks": [{"type": "command", "command": "claude-secret-scan --mode=post"}]}
    ]
  }
}
```

## Hook Behavior

- **Pre-hooks** block execution when credentials are identified
- **Post-hooks** display warnings without blocking
- Hooks execute at: prompt submission, before file reads, and after tool execution

## Technical Architecture

Regex-only pattern matching, no external dependencies. Core scanner in `plugins/secrets_scanner/hooks/secrets_scanner_hook.py`. Detection patterns informed by/adapted from `detect-secrets` (Apache 2.0).

## Design Philosophy

Local-first processing — no code, prompts, or files leave your system. Telemetry disabled by default. Supports opt-in organizational governance through external platforms.
