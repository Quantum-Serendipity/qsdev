<!-- Source: https://snyk.io/articles/snyk-mcp-cheat-sheet/ -->
<!-- Retrieved: 2026-05-12 -->

# Snyk MCP Cheat Sheet

## Installation & Setup

The Snyk MCP server comes bundled with the Snyk CLI (v1.1298.0 or later).

Launch: `snyk mcp -t <transport type>`

JSON configuration:
```json
"servers": {
   "SnykMCP": {
       "command": "snyk",
       "args": ["mcp", "-t", "stdio"],
   }
}
```

Alternative with npx: `npx -y snyk@latest mcp -t stdio`

## Transport Options

- **stdio** (recommended): Standard input/output for fast, lightweight local communication
- **SSE**: Server-Sent Events via HTTP layer for local operation

## Authentication

- Login via browser automatically when needed
- Manually invoke `/snyk_auth`
- If already authenticated with Snyk CLI, no additional steps required
- For restricted environments, set `SNYK_TOKEN=<TOKEN>` environment variable

## Core MCP Tools

### Security Scanning
- `snyk_sca_scan` — Detects open source dependency vulnerabilities
- `snyk_code_scan` — Assesses proprietary code security flaws
- `snyk_iac_scan` — Examines Infrastructure as Code configurations
- `snyk_container_scan` — Identifies container image vulnerabilities

### Supply Chain Tools
- `snyk_sbom_scan` — Analyzes existing Software Bill of Materials files
- `snyk_aibom` — Creates AI Bill of Materials documentation

### Utilities
- `snyk_version` — Displays version information
- `snyk_logout` — Terminates sessions
- `snyk_trust` — Trusts folders before scanning (usually automatic)
- `snyk_auth` — Manual authentication

## Recommended Prompts

Basic: "Scan this project for code security & dependency vulnerabilities and security issues"

Explicit: "Scan this project for code security & dependency vulnerabilities and security issues with Snyk"

## System Rules for AI Agents

Configure your assistant to automatically:
1. Run Snyk Code scanning for new first-party code
2. Execute SCA tool for dependencies or updates
3. Attempt fixes based on Snyk results
4. Rescan to verify fixes and catch new issues
5. Repeat until no vulnerabilities remain

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Compatibility | Ensure CLI v1.1298.0+ installed |
| Direct scan fails | Test scanning directly from terminal first |
| Browser login blocked | Use SNYK_TOKEN environment variable |
| Multi-org accounts | Set `SNYK_CFG_ORG=<YOUR_ORG_ID>` |
| Verbose diagnostics | Run with `-d` flag: `snyk mcp -t stdio -d` |
| Folder trust issues | Disable with: `snyk mcp -t stdio --disable-trust` |
| Large responses | Save to temp file using `-o` flag |

## Supported AI Platforms

Amazon Q, Claude Code, Cursor, Gemini Code Assist, GitHub Copilot, JetBrains AI Assistant, Qodo, Windsurf, and others.

## Documentation

Official Snyk Studio docs: https://docs.snyk.io/integrations/developer-guardrails-for-agentic-workflows
