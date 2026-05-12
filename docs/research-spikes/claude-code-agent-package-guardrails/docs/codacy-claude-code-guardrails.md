# Codacy: Deterministic Security Guardrails for Claude Code

- **Source URL**: https://blog.codacy.com/equipping-claude-code-with-deterministic-security-guardrails
- **Retrieved**: 2026-05-12
- **Note**: Codacy's MCP-based integration for real-time security analysis during code generation.

---

## Approach

Integrates via MCP (Model Context Protocol), not hooks. Acts as real-time security analysis layer during code generation.

## Security Enforcement

1. **During-Generation Analysis**: Every line analyzed as generated; AI agent made aware immediately of issues.
2. **Automatic Iteration**: Claude proposes and applies fixes when issues detected.
3. **Dependency Scanning**: Trivy-based vulnerability analysis run IMMEDIATELY after any dependency installation.

## Configuration

CLAUDE.md mandates:
- Run `codacy_cli_analyze` for each edited file
- Execute Trivy scanning after dependency installation
- Enforce company standards and team coding rules

## Tools Exposed

23 tools via `/mcp` command, covering:
- Near real-time scanning during file modification
- Automatic issue correction
- New dependency vulnerability analysis
- Organizational standards enforcement

## Practical Result

Caught 6 security issues instantly with SLA-based prioritization.

## Key Distinction

This is an MCP server integration, not a hooks-based approach. Relies on CLAUDE.md instructions and MCP tool availability rather than deterministic PreToolUse blocking. The agent must choose to call the tools — they are not automatically invoked.
