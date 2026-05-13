# Semble - MCP Configuration Examples

- **Source**: https://github.com/MinishLab/semble
- **Retrieved**: 2026-05-12

## Claude Code
```
claude mcp add semble -s user -- uvx --from "semble[mcp]" semble
```

## Codex
Add to `~/.codex/config.toml`:
```toml
[mcp_servers.semble]
command = "uvx"
args = ["--from", "semble[mcp]", "semble"]
```

## OpenCode
Add to `~/.opencode/config.json`:
```json
{
  "mcp": {
    "semble": {
      "type": "local",
      "command": ["uvx", "--from", "semble[mcp]", "semble"]
    }
  }
}
```

## Cursor
Add to `~/.cursor/mcp.json` or `.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "semble": {
      "command": "uvx",
      "args": ["--from", "semble[mcp]", "semble"]
    }
  }
}
```

## Bash / AGENTS.md Integration
```bash
semble search "authentication flow" ./my-project
semble search "save_pretrained" ./my-project
semble find-related src/auth.py 42 ./my-project
```

## Claude Code Sub-Agent Init
```bash
semble init
# Creates .claude/agents/semble-search.md
```
