# mcph Config Schema v1
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/schemas/mcph.config.v1.json
- **Retrieved**: 2026-05-14

## Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://raw.githubusercontent.com/YawLabs/mcph/main/schemas/mcph.config.v1.json",
  "title": "mcph CLI configuration (.mcph/config.json)",
  "description": "Configuration file for the @yawlabs/mcph CLI. Resolved in precedence order: <project>/.mcph/config.local.json > <project>/.mcph/config.json > ~/.mcph/config.json. Env vars MCPH_TOKEN and MCPH_URL override file values. JSONC permitted.",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "$schema": { "type": "string", "format": "uri" },
    "version": { "type": "integer", "minimum": 1 },
    "token": {
      "type": "string",
      "pattern": "^mcp_pat_[A-Za-z0-9_-]+$",
      "description": "Personal access token for mcp.hosting. SECURITY: must NOT appear in committed project-scope config. Only valid in ~/.mcph/config.json or .mcph/config.local.json."
    },
    "apiBase": {
      "type": "string",
      "format": "uri",
      "pattern": "^https?://",
      "description": "Base URL of the mcp.hosting API. Default: https://mcp.hosting"
    },
    "servers": {
      "type": "array",
      "items": { "type": "string", "minLength": 1 },
      "uniqueItems": true,
      "description": "Allow-list of server namespaces to activate. Most-specific scope wins."
    },
    "blocked": {
      "type": "array",
      "items": { "type": "string", "minLength": 1 },
      "uniqueItems": true,
      "description": "Deny-list of server namespaces. Merged as union across all scopes (additive)."
    }
  }
}
```

## Config Precedence
1. `<project>/.mcph/config.local.json` (machine-local, gitignored)
2. `<project>/.mcph/config.json` (team-shared, committed)
3. `~/.mcph/config.json` (personal default)

Environment variables override file values:
- `MCPH_TOKEN` overrides `token`
- `MCPH_URL` overrides `apiBase`
