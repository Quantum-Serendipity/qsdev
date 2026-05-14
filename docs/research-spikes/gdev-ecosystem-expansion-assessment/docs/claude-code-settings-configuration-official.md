# Claude Code Settings Configuration - Official Documentation
- **Source**: https://code.claude.com/docs/en/settings
- **Retrieved**: 2026-05-14

## Settings File Locations & Precedence Hierarchy

| Scope | Location | Shared with Team | Priority |
|-------|----------|------------------|----------|
| **Managed** | Server, plist/registry, or system paths | Yes (IT-deployed) | **Highest (1)** |
| **Command Line** | CLI flags and arguments | N/A | **2** |
| **Local** | `.claude/settings.local.json` (gitignored) | No | **3** |
| **Project** | `.claude/settings.json` (committed to git) | Yes | **4** |
| **User** | `~/.claude/settings.json` | No | **Lowest (5)** |

Precedence: Managed > Command-line args > Local > Project > User

### Managed Settings Delivery Methods

**File-based:**
- macOS: `/Library/Application Support/ClaudeCode/`
- Linux/WSL: `/etc/claude-code/`
- Windows: `C:\Program Files\ClaudeCode/`

**Drop-in directory merging** (systemd convention):
- `managed-settings.json` merged first as base
- `*.json` files in `managed-settings.d/` sorted alphabetically, then merged
- Later files override scalars; arrays concatenated and de-duplicated; objects deep-merged

## Environment Variables (`env` Key)

```json
{
  "env": {
    "VARIABLE_NAME": "value"
  }
}
```

Can be set at any scope (managed, user, project, local). Applied to every session.

## Skills Configuration & Loading

### Skill File Locations

| Location | Scope |
|----------|-------|
| `~/.claude/CLAUDE.md` | User |
| `.claude/CLAUDE.md` | Project |
| `CLAUDE.local.md` | Local |
| `~/.claude/agents/` | User subagents |
| `.claude/agents/` | Project subagents |

### Skill Overrides

```json
{
  "skillOverrides": {
    "legacy-context": "name-only",
    "deploy": "off",
    "custom-skill": "user-invocable-only"
  }
}
```

Visibility values: `"on"`, `"name-only"`, `"user-invocable-only"`, `"off"`

### Context Budgeting

```json
{
  "skillListingBudgetFraction": 0.01,
  "maxSkillDescriptionChars": 1536
}
```

Default skill listing budget: 1% of context window. Default max description: 1536 chars.

### Memory Files (`claudeMd`)

```json
{
  "claudeMd": "Always run make lint before committing.",
  "claudeMdExcludes": ["**/vendor/**/CLAUDE.md"]
}
```

Managed settings only: `claudeMd` for org-wide memory.

## Permission Rules

Rules defined in `permissions` key:

```json
{
  "permissions": {
    "allow": ["Bash(npm run lint)"],
    "ask": ["Bash(git push *)"],
    "deny": ["WebFetch", "Bash(curl *)"]
  }
}
```

Evaluation order: Deny > Ask > Allow (first match wins).
Permission rules MERGE across scopes rather than override.

## Path Resolution

| Prefix | Resolution |
|--------|-----------|
| `/` | Filesystem root |
| `~/` | `$HOME` |
| `./` or none | Project root (project settings) or `~/.claude` (user) |

### Auto Memory Directory

```json
{
  "autoMemoryDirectory": "~/my-memory-dir"
}
```

Accepted from policy and user settings. NOT accepted from project/local settings (security).

## No CLAUDE_HOME Variable

Documentation does NOT mention a `CLAUDE_HOME` environment variable for overriding the base config directory. Configuration is scope-based:
- User scope: `~/.claude/` (always)
- Project scope: `.claude/` in repository root
- Managed: System-level paths (platform-specific)

## Configuration Hierarchy Summary

```
Command-line arguments (highest)
  > Managed settings (server/MDM/file)
    > Local (.claude/settings.local.json)
      > Project (.claude/settings.json)
        > User (~/.claude/settings.json)
```

Permission rules merge across all scopes; deny takes precedence.
