# rulebricks/claude-code-guardrails

- **Source URL**: https://github.com/rulebricks/claude-code-guardrails
- **Retrieved**: 2026-05-12
- **Note**: External policy-as-a-service guardrails for Claude Code via Rulebricks API.

---

## Architecture

Claude Code -> PreToolUse hook -> Rulebricks API -> allow / deny / ask

## Supported Tool Matchers
- `Bash` — Controls shell command execution
- `Read|Write|Edit` — Manages file operations
- `mcp__*` — Governs MCP server operations

## Configuration

Environment variables in `~/.claude/settings.json`:
```json
{
  "env": {
    "RULEBRICKS_API_KEY": "your-api-key",
    "RULEBRICKS_VERBOSE": "1"
  }
}
```

## Key Advantages

- Policy changes apply instantly across team — no git pull, no restart
- Conditional logic: "allow `rm -rf` on `node_modules`, deny everywhere else"
- Audit log of all blocked operations, filterable by tool type and decision

## Installation

1. Create account at rulebricks.com
2. Fork a template (Bash Guardrails, File Access Policy, or MCP Tool Governance)
3. Customize rules and publish
4. Run `./install.sh` from cloned repository
5. Restart Claude Code

## Removal

```bash
rm ~/.claude/hooks/guardrail.py
```
Then remove PreToolUse hook entry and RULEBRICKS_* env vars from settings.json.
