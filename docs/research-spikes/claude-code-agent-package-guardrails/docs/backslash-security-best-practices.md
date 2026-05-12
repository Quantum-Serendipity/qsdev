<!-- Source: https://www.backslash.security/blog/claude-code-security-best-practices -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Security Best Practices (Backslash Security)

## Threat Model

Four primary attack vectors:
1. **Command Injection** - Malicious inputs convincing Claude to run destructive commands
2. **Data Exfiltration** - Unauthorized access to `.env` files, credentials, and secrets
3. **Persistence** - Compromised hooks or MCP servers reintroducing malicious code
4. **Safeguard Bypass** - Unsafe defaults like auto-approving servers

## Core Configuration: managed-settings.json

Located at `/Library/Application Support/ClaudeCode/managed-settings.json` (macOS) or `/etc/claude-code/` (Linux).

### Critical Settings

| Setting | Recommendation |
|---------|---|
| `cleanupPeriodDays` | 7-14 days - Minimizes transcript exposure |
| `disableAllHooks` | true - Blocks all pre/post-tool scripts |

## MCP Server Security

**Dangerous:** `{ "enableAllProjectMcpServers": true }` - enables any discovered server without verification.

**Secure:** Explicitly whitelist only trusted servers:
```json
{ "enabledMcpServers": ["github", "memory"] }
```

## Permission Architecture

### Denylist (permissions.deny)
"Nuclear shield" blocking dangerous operations:
```json
{ "permissions": { "deny": ["WebFetch", "Bash(curl:*)", "Read(./secrets/**)"] } }
```

### Allowlist (permissions.allow)
Include only 100% harmless commands:
```json
{ "permissions": { "allow": ["Bash(echo Hello)"] } }
```

### Default Behavior
Set default mode to "ask" - Claude must always request permission for unmatched commands.

## Recommendations

1. Use allowlists as primary defense, denylists as additional layers
2. Sandbox Claude Code in VMs or containers
3. Never run as root
4. Use vault systems instead of plaintext `.env` files
5. Audit configurations monthly

## Core Philosophy

"Treat Claude like you would an untrusted but powerful intern." Provide minimum necessary permissions, implement sandboxing, and conduct regular audits.
