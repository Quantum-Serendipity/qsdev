<!-- Source: https://raw.githubusercontent.com/rulebricks/claude-code-guardrails/main/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Comparison alternative to Prempti -->

# Claude Code Guardrails (rulebricks)

## What It Does

Claude Code Guardrails is a governance solution that enables teams to enforce security policies on AI tool usage within Claude Code. It controls what operations Claude can execute -- specifically shell commands, file operations, and MCP server calls -- by validating requests against customizable rules before they run.

## How It Works

Core Flow: Claude Code -> PreToolUse hook -> Rulebricks API -> allow / deny / ask

When Claude attempts to use a tool, the PreToolUse hook intercepts the request and sends it to Rulebricks API for validation against published rules. The API returns an allow, deny, or ask decision.

## Key Features

- Policy changes apply instantly across team -- no git pull, no restart
- Complete audit trail logging blocked commands with timestamps and user information
- Conditional logic capabilities (e.g., allowing `rm -rf` only on specific directories)
- Non-technical team members can edit rules without touching code

## Technical Details

- Installation: Shell script auto-detects published rules, configures hooks at `~/.claude/hooks/guardrail.py`
- Configuration: Rules managed through Rulebricks platform with API auth via env vars
- Templates: Bash Command Guardrails, File Access Policy, MCP Tool Governance

## Repo Metadata

- Stars: 67, Forks: 8
- Language: Python
- Created: 2026-01-15, Updated: 2026-05-03
- Requires external Rulebricks API (SaaS dependency)
