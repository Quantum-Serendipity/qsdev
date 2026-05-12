# Claude Code Settings Configuration

Source: https://code.claude.com/docs/en/settings.md
Retrieved: 2026-05-12

## Overview
Claude Code uses a hierarchical scope system to manage configuration settings through JSON files, environment variables, and managed policies. Settings control permissions, behavior, integrations, and user preferences.

## Configuration Scopes

| Scope | Location | Who It Affects | Shared with Team |
|-------|----------|----------------|------------------|
| **Managed** | Server-managed, plist/registry, or `managed-settings.json` | All users on machine | Yes (IT-deployed) |
| **User** | `~/.claude/` directory | You, across all projects | No |
| **Project** | `.claude/` in repository | All collaborators | Yes (git-committed) |
| **Local** | `.claude/settings.local.json` | You, in this repo only | No (gitignored) |

### Precedence Order (highest to lowest)
1. Managed settings (cannot be overridden)
2. Command-line arguments
3. Local settings (`.claude/settings.local.json`)
4. Project settings (`.claude/settings.json`)
5. User settings (`~/.claude/settings.json`)

## Settings File Locations

| Feature | User Location | Project Location | Local Location |
|---------|---------------|------------------|----------------|
| **Settings** | `~/.claude/settings.json` | `.claude/settings.json` | `.claude/settings.local.json` |
| **Subagents** | `~/.claude/agents/` | `.claude/agents/` | None |
| **MCP servers** | `~/.claude.json` | `.mcp.json` | `~/.claude.json` |
| **Plugins** | `~/.claude/settings.json` | `.claude/settings.json` | `.claude/settings.local.json` |
| **CLAUDE.md** | `~/.claude/CLAUDE.md` | `CLAUDE.md` or `.claude/CLAUDE.md` | `CLAUDE.local.md` |

**Note:** On Windows, `~/.claude` resolves to `%USERPROFILE%\.claude`

## Key Configuration Settings

### Core Settings
- **`agent`**: Run as named subagent with specific system prompt/tools
- **`model`**: Override default Claude model
- **`availableModels`**: Restrict which models users can select
- **`effortLevel`**: Persist effort level (`"low"`, `"medium"`, `"high"`, `"xhigh"`)
- **`language`**: Configure Claude's response language

### Behavioral Settings
- **`autoMemoryEnabled`**: Enable/disable auto memory (default: `true`)
- **`autoScrollEnabled`**: Follow output to bottom (default: `true`)
- **`awaySummaryEnabled`**: Show session recap after absence
- **`editorMode`**: `"normal"` or `"vim"` key bindings
- **`viewMode`**: Default transcript view (`"default"`, `"verbose"`, `"focus"`)

### Permission Settings
```json
{
  "permissions": {
    "allow": ["Bash(npm run *)", "Read(~/.zshrc)"],
    "ask": ["Bash(git push *)"],
    "deny": ["WebFetch", "Bash(curl *)", "Read(./.env)"],
    "defaultMode": "acceptEdits",
    "additionalDirectories": ["../docs/"]
  }
}
```

### Sandbox Configuration
```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "filesystem": {
      "allowWrite": ["/tmp/build"],
      "denyWrite": ["/etc"],
      "denyRead": ["~/.aws/credentials"]
    },
    "network": {
      "allowedDomains": ["github.com", "*.npmjs.org"],
      "deniedDomains": ["sensitive.internal.com"]
    }
  }
}
```

### Attribution Settings
```json
{
  "attribution": {
    "commit": "🤖 Generated with Claude Code\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
    "pr": "🤖 Generated with Claude Code"
  }
}
```

### Managed Settings Deployment

Managed settings support multiple delivery mechanisms:

1. **Server-Managed Settings**: Delivered via Anthropic's Claude.ai admin console
2. **MDM/OS-Level Policies**:
   - **macOS**: `com.anthropic.claudecode` managed preferences domain (via Jamf, Kandji, etc.)
   - **Windows (Admin)**: `HKLM\SOFTWARE\Policies\ClaudeCode` registry key
   - **Windows (User)**: `HKCU\SOFTWARE\Policies\ClaudeCode` (lowest priority)
3. **File-Based**:
   - **macOS**: `/Library/Application Support/ClaudeCode/`
   - **Linux/WSL**: `/etc/claude-code/`
   - **Windows**: `C:\Program Files\ClaudeCode\`

Files: `managed-settings.json` and `managed-mcp.json`

**Drop-in directory support**: `managed-settings.d/` with alphabetical merge order using numeric prefixes (e.g., `10-telemetry.json`, `20-security.json`)
