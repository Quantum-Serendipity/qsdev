<!-- Source: https://dev.to/klement_gunndu/lock-down-claude-code-with-5-permission-patterns-4gcn -->
<!-- Retrieved: 2026-05-12 -->

# Lock Down Claude Code With 5 Permission Patterns

## Pattern 1: Deny-First Rules in settings.json

**Core Principle:** "Claude Code evaluates permission rules in a strict order: deny, then ask, then allow."

**Deny section blocks:**
- Environment files: `.env`, `.env.*`
- Secrets directories: `secrets/**`
- Network tools: `curl`, `wget`
- Destructive git: `git push --force`, `rm -rf`

**Allow section permits safe operations:**
- Build commands: `npm run lint`, `npm run test`
- Version control: `git commit`
- Code quality: `python -m pytest`, `ruff check`

**Critical insight:** A deny rule always supersedes an allow rule regardless of JSON ordering. Additionally, denying the Read tool doesn't prevent `cat .env` in Bash—you must deny both the tool and the Bash equivalent for complete protection.

## Pattern 2: The 4-Layer Settings Hierarchy

1. Managed settings (admin-deployed, cannot override)
2. Command-line arguments
3. Local project settings (`.claude/settings.local.json`, gitignored)
4. Shared project settings (`.claude/settings.json`, committed)
5. User settings (`~/.claude/settings.json`, global)

**Team workflow advantage:** Shared deny rules cannot be overridden by local additions. A developer's personal `.claude/settings.local.json` can extend approvals without weakening team security boundaries.

## Pattern 3: MCP Server and Subagent Controls

**MCP server permission format:**
```
mcp__<server>__<tool>
```

Example:
- Allow: `mcp__filesystem__read_file`, `mcp__github__list_pull_requests`
- Deny: `mcp__filesystem__write_file`, `mcp__github__merge_pull_request`

**Subagent controls** use `Agent(name)` syntax.

## Pattern 4: Sandbox for OS-Level Enforcement

Permission rules prevent Claude from attempting restricted actions; sandbox restrictions block underlying processes even if prompt injection bypasses decision-making.

```json
"sandbox": {
  "enabled": true,
  "filesystem": {
    "denyRead": [".env", ".env.*", "secrets/**"],
    "allowRead": ["src/**", "tests/**", "docs/**"]
  },
  "network": {
    "allowedDomains": ["registry.npmjs.org", "pypi.org"]
  }
}
```

## Pattern 5: Permission Modes for Different Workflows

Six modes match different scenarios. Key for security: `dontAsk` mode auto-denies everything unless pre-approved. `disableBypassPermissionsMode: "disable"` prevents developers from circumventing all configured permissions.

## Production Configuration Example

```json
{
  "permissions": {
    "defaultMode": "acceptEdits",
    "deny": [
      "Read(./.env)", "Read(./.env.*)", "Read(./secrets/**)",
      "Bash(curl *)", "Bash(wget *)",
      "Bash(git push --force *)", "Bash(git push * --force)",
      "Bash(rm -rf *)", "Bash(git checkout .)",
      "Bash(git reset --hard *)",
      "mcp__filesystem__write_file"
    ],
    "allow": [
      "Bash(npm run *)", "Bash(python -m pytest *)",
      "Bash(ruff check *)", "Bash(black *)",
      "Bash(git status)", "Bash(git diff *)",
      "Bash(git add *)", "Bash(git commit *)",
      "Bash(git log *)", "Bash(* --version)",
      "Bash(* --help *)",
      "mcp__github__list_pull_requests",
      "mcp__github__get_pull_request",
      "Agent(Plan)"
    ],
    "disableBypassPermissionsMode": "disable"
  }
}
```

## Security Audit Checklist

1. Create `.claude/settings.json` with environment and secrets denials
2. Add `.claude/settings.local.json` to `.gitignore`
3. Deny force-push in both flag positions
4. Deny Read AND Bash equivalent for sensitive files
5. Enable sandboxing on macOS/Linux
6. Set `disableBypassPermissionsMode` to `"disable"`
7. Review auto-approved commands using `/permissions` after sessions
8. Use `acceptEdits` as default mode

**Risk without configuration:** Default settings provide full filesystem read access, full write access after one approval, unrestricted network access through Bash, no destructive git protections, and permanent allow rules accumulating invisibly.
