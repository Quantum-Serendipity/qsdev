# Claude Code Settings Reference

- **Source URL**: https://code.claude.com/docs/en/settings
- **Retrieved**: 2026-05-12
- **Note**: Official documentation on settings hierarchy, managed settings, and hook interaction across scopes.

---

## Settings File Hierarchy

| Scope | Location | Who Affected | Shared? |
|---|---|---|---|
| Managed | Server/MDM/system-level | All users on machine | Yes (deployed by IT) |
| Project | `.claude/settings.json` | All collaborators | Yes (committed to git) |
| User | `~/.claude/settings.json` | You, across all projects | No |
| Local | `.claude/settings.local.json` | You, in this repo only | No (gitignored) |

## Scope Precedence (Highest to Lowest)

1. **Managed** (highest) — cannot be overridden
2. **Command line arguments**
3. **Local**
4. **Project**
5. **User** (lowest)

## Hook-Specific Managed Settings

| Setting | Effect |
|---|---|
| `allowManagedHooksOnly` | Only managed/SDK/force-enabled plugin hooks load. User, project, and plugin hooks blocked. |
| `allowedHttpHookUrls` | Allowlist of URL patterns for HTTP hooks. Arrays merge across scopes. |
| `httpHookAllowedEnvVars` | Allowlist of env vars HTTP hooks may interpolate. Arrays merge. |
| `disableAllHooks` | Completely disables all hooks and custom status lines. |

## When `allowManagedHooksOnly` is true:
- Managed hooks: loaded
- SDK hooks: loaded
- Force-enabled plugin hooks: loaded
- User hooks: **BLOCKED**
- Project hooks: **BLOCKED**
- Other plugin hooks: **BLOCKED**

## Managed Settings Delivery

**Linux/WSL**: `/etc/claude-code/managed-settings.json`
**macOS**: `/Library/Application Support/ClaudeCode/managed-settings.json`

Drop-in directory support:
```
/etc/claude-code/
├── managed-settings.json
└── managed-settings.d/
    ├── 10-telemetry.json
    ├── 20-security.json
    └── 30-permissions.json
```

Merge: base first, then drop-in files alphabetically. Arrays concatenated and de-duplicated. Objects deep-merged.

## Permission Rules Interaction

- Project settings override user settings for permissions
- Managed `allowManagedPermissionRulesOnly` prevents user/project from defining permission rules
- Deny rules always take precedence over hook approvals
- PreToolUse hooks fire before permission checks — deny from hook blocks even in bypassPermissions mode

## Key Implication for Package Guardrails

Project-level `.claude/settings.json` hooks apply to all collaborators and can be committed to version control. For enterprise enforcement, managed settings with `allowManagedHooksOnly` ensures only admin-approved hooks run — users cannot disable or override them.
