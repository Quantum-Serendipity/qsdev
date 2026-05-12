<!-- Source: https://code.claude.com/docs/en/permissions -->
<!-- Retrieved: 2026-05-12 -->

# Configure permissions

> Control what Claude Code can access and do with fine-grained permission rules, modes, and managed policies.

## Permission system

Claude Code uses a tiered permission system:

| Tool type         | Example          | Approval required | "Yes, don't ask again" behavior               |
|-------------------|------------------|-------------------|-----------------------------------------------|
| Read-only         | File reads, Grep | No                | N/A                                           |
| Bash commands     | Shell execution  | Yes               | Permanently per project directory and command  |
| File modification | Edit/write files | Yes               | Until session end                             |

## Rule Evaluation Order

Rules are evaluated in order: **deny -> ask -> allow**. The first matching rule wins, so deny rules always take precedence.

Permission rules are enforced by Claude Code, not by the model. Instructions in your prompt or CLAUDE.md shape what Claude tries to do, but they don't change what Claude Code allows.

## Permission modes

| Mode                | Description |
|---------------------|-------------|
| `default`           | Standard behavior: prompts for permission on first use |
| `acceptEdits`       | Auto-accepts file edits and common filesystem commands |
| `plan`              | Plan Mode: reads files, runs read-only shell commands, no edits |
| `auto`              | Auto-approves with background safety checks (research preview) |
| `dontAsk`           | Auto-denies tools unless pre-approved via permissions |
| `bypassPermissions` | Skips all permission prompts (isolated environments only) |

## Permission rule syntax

Format: `Tool` or `Tool(specifier)`

### Bash Rules with Wildcards

Bash rules support glob patterns with `*` at any position:

| Rule | Effect |
|------|--------|
| `Bash(npm run build)` | Matches exact command `npm run build` |
| `Bash(npm run test *)` | Matches commands starting with `npm run test` |
| `Bash(npm *)` | Matches any command starting with `npm ` |
| `Bash(* install)` | Matches any command ending with ` install` |
| `Bash(git * main)` | Matches commands like `git checkout main` |

**Compound commands**: Claude Code is aware of shell operators. A rule like `Bash(safe-cmd *)` won't give permission to run `safe-cmd && other-cmd`. Each subcommand must match independently.

**Process wrappers**: Claude Code strips `timeout`, `time`, `nice`, `nohup`, and `stdbuf` before matching.

### MCP Tool Rules

- `mcp__puppeteer` matches any tool provided by the `puppeteer` server
- `mcp__puppeteer__*` wildcard matches all tools from the server
- `mcp__puppeteer__puppeteer_navigate` matches a specific tool

## Example Configuration

```json
{
  "permissions": {
    "allow": [
      "Bash(npm run *)",
      "Bash(git commit *)",
      "Bash(* --version)",
      "Bash(* --help *)"
    ],
    "deny": [
      "Bash(git push *)"
    ]
  }
}
```

## Extending Permissions with Hooks

PreToolUse hooks run before the permission prompt. The hook output can deny the tool call, force a prompt, or skip the prompt.

**Key interactions**:
- Hook decisions do NOT bypass permission rules
- Deny rules always take precedence over hook "allow" responses
- A blocking hook (exit 2) takes precedence over allow rules
- A hook that exits 2 stops the tool call BEFORE permission rules are evaluated

**Pattern**: Allow all Bash via allow rules, then use PreToolUse hook to reject specific commands:
- Add `"Bash"` to allow list
- Register PreToolUse hook that rejects dangerous commands

## Managed Settings (Enterprise)

Administrators can deploy managed settings that cannot be overridden:

| Setting | Description |
|---------|-------------|
| `allowManagedPermissionRulesOnly` | Prevents user/project settings from defining permission rules |
| `allowManagedMcpServersOnly` | Only managed MCP servers are respected |
| `allowManagedHooksOnly` | Only managed hooks are loaded |
| `disableBypassPermissionsMode` | Prevent bypass mode |
| `disableAutoMode` | Prevent auto mode |

## Settings Precedence

1. Managed settings (highest, cannot be overridden)
2. Command line arguments
3. Local project settings (`.claude/settings.local.json`)
4. Shared project settings (`.claude/settings.json`)
5. User settings (`~/.claude/settings.json`)

If a tool is denied at any level, no other level can allow it.

## Sandboxing Interaction

Permissions and sandboxing are complementary:
- **Permissions**: control which tools Claude Code can use
- **Sandboxing**: OS-level enforcement restricting Bash filesystem and network access
- Both should be used for defense-in-depth
