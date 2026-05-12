<!-- Source: https://code.claude.com/docs/en/settings -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Settings - Complete Reference

## Settings File Hierarchy & Precedence

### Scope System Overview

| Scope | Location | Who It Affects | Shared with Team | Can Be Overridden |
|-------|----------|----------------|------------------|-------------------|
| **Managed** | Server-managed, plist/registry, or `managed-settings.json` | All users on machine | Yes (deployed by IT) | No - highest precedence |
| **User** | `~/.claude/` directory | You, across all projects | No | Yes, by local/project |
| **Project** | `.claude/` in repository | All collaborators | Yes (committed to git) | Yes, by local |
| **Local** | `.claude/settings.local.json` | You, in this repository only | No (gitignored) | N/A - most specific |

### Precedence Order (Highest to Lowest)

1. **Managed** (cannot be overridden by anything)
2. **Command line arguments** (temporary session overrides)
3. **Local** (overrides project and user settings)
4. **Project** (overrides user settings)
5. **User** (applies when nothing else specifies the setting)

If a permission is allowed in user settings but denied in project settings, the project setting takes precedence and the permission is blocked.

## Settings Array Merging Behavior

- **Most rules:** Arrays concatenate and de-duplicate
- **Permission arrays** (`allow`, `ask`, `deny`): Evaluated in order; deny first, then ask, then allow; first matching rule wins
- **Sandbox filesystem paths**: Arrays merged from all settings scopes (combined, not replaced)

## File Locations

| Feature | User Location | Project Location | Local Location |
|---------|---------------|------------------|----------------|
| **Settings** | `~/.claude/settings.json` | `.claude/settings.json` | `.claude/settings.local.json` |
| **Subagents** | `~/.claude/agents/` | `.claude/agents/` | None |
| **MCP servers** | `~/.claude.json` | `.mcp.json` | `~/.claude.json` (per-project) |
| **CLAUDE.md** | `~/.claude/CLAUDE.md` | `CLAUDE.md` or `.claude/CLAUDE.md` | `CLAUDE.local.md` |

## Key Permission Settings

| Key | Description |
|-----|-------------|
| `permissions.allow` | Array of permission rules to allow tool use |
| `permissions.ask` | Array of permission rules to ask for confirmation |
| `permissions.deny` | Array of permission rules to deny tool use |
| `permissions.defaultMode` | Default permission mode on startup |
| `permissions.disableBypassPermissionsMode` | Set to `"disable"` to prevent bypass mode |
| `allowManagedPermissionRulesOnly` | (Managed only) Block user/project permission rules; enforce managed only |

## Managed Settings Delivery Mechanisms

- **Server-Managed Settings**: Delivered from Anthropic's servers via Claude.ai admin console
- **macOS**: `com.anthropic.claudecode` managed preferences domain (plist)
- **Windows (admin)**: `HKLM\SOFTWARE\Policies\ClaudeCode` registry key
- **Linux/WSL**: `/etc/claude-code/managed-settings.json`

### Drop-In Directory Support

File-based managed settings support `managed-settings.d/` directory:
- `managed-settings.json` is merged first as base
- All `*.json` files in drop-in directory sorted alphabetically and merged on top
- Later files override earlier ones for scalar values
- Arrays concatenated and de-duplicated

## Managed-Only Settings

These settings can only be configured in managed settings and cannot be overridden:

| Setting | Purpose |
|---------|---------|
| `allowManagedPermissionRulesOnly` | Block user/project permission rules; enforce managed only |
| `allowManagedHooksOnly` | Force-load only managed hooks |
| `allowManagedMcpServersOnly` | Enforce admin-defined MCP server allowlist |
| `forceRemoteSettingsRefresh` | Block startup until remote settings fetched |

## MCP Server Settings

| Key | Description |
|-----|-------------|
| `allowedMcpServers` | Allowlist of MCP servers users can configure |
| `deniedMcpServers` | Denylist of MCP servers |
| `allowManagedMcpServersOnly` | Only admin-defined allowlist applies |
| `enableAllProjectMcpServers` | Auto-approve all project MCP servers |

## Sandbox Settings

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "excludedCommands": ["docker *"],
    "filesystem": {
      "allowWrite": ["/tmp/build", "~/.kube"],
      "denyRead": ["~/.aws/credentials"]
    },
    "network": {
      "allowedDomains": ["github.com", "*.npmjs.org"],
      "deniedDomains": ["uploads.github.com"]
    }
  }
}
```
