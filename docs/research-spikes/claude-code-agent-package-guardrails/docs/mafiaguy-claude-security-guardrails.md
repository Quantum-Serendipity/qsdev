# mafiaguy/claude-security-guardrails

- **Source URL**: https://github.com/mafiaguy/claude-security-guardrails
- **Retrieved**: 2026-05-12
- **Note**: PreToolUse/PostToolUse hooks with React dashboard for monitoring.

---

## Architecture

- **PreToolUse Hook**: Blocking mechanism. Blocks writes containing secrets, SQL injection, eval(), or other vulnerabilities before execution.
- **PostToolUse Hook**: Reporting layer. Scans after file writes, generates findings, persists to dashboard.

## Detected Threat Categories (60+ patterns)

**Secrets (10 patterns):** AWS keys, GitHub tokens, private keys, database connection strings, Slack tokens, API keys, JWT tokens, hardcoded passwords, generic secrets.

**OWASP vulnerabilities (13 patterns):** SQL injection, command injection, XSS, SSRF, path traversal, CORS misconfiguration.

**Dangerous commands (30+ patterns):** filesystem destruction (`rm -rf /`), disk wiping, fork bombs, git disasters (`git push --force main`), remote code execution (`curl | bash`), secrets exposure, database destruction, data exfiltration.

**Dependencies:** Checks for 16 known vulnerable packages with CVEs and unsafe version ranges.

**Code patterns (11 patterns):** `eval()`, `new Function()`, MD5 for passwords, disabled TLS, insecure randomness, sensitive data logging.

## Configuration

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Write|Edit|Bash",
      "hooks": [{
        "type": "command",
        "command": "node .claude-hooks/pre-tool-use.js"
      }]
    }]
  }
}
```

Safety levels configurable: `critical`, `high`, `strict`.

## React Dashboard

- SecurityScore (0-100 gauge)
- SeverityChart (bar chart)
- CategoryBreakdown (donut chart)
- ActivityLog (real-time hook events)
- FindingsTable (detailed vulnerability listings)

## Deployment Options

- Local per-project
- Global (`~/.claude/hooks/` + `~/.claude/settings.json`)
- Enterprise managed (MDM/policy files)
- Centralized dashboard (Docker)
