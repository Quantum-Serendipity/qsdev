<!-- Source: https://github.com/snyk/agent-scan -->
<!-- Retrieved: 2026-05-12 -->

# Agent Scan: Security Scanner for AI Agents & MCP Servers

## What It Does

Agent Scan is a security tool that discovers and examines agent components on your machine. It scans for vulnerabilities including prompt injections and vulnerabilities in agents, MCP servers, and skills. Auto-discovers configurations for Claude, Cursor, Windsurf, Gemini CLI, and other AI agents.

## Security Threats Detected (15+)

**MCP Servers:**
- Prompt injection
- Tool poisoning
- Tool shadowing
- Toxic flows

**Agent Skills:**
- Prompt injection
- Malware payloads
- Untrusted content
- Credential mishandling
- Hardcoded secrets

## How It Works

### Scan Mode (default)
CLI searches local agent configuration files, connects to MCP servers to retrieve tool descriptions, then validates components through both local checks and the Agent Scan API.

### Background Mode
Monitors machines at regular intervals (enterprise/MDM deployments).

### Critical Security Note
"Scanning an MCP config executes the commands defined in it." By default, the tool requires interactive user consent before starting each stdio MCP server, displaying the exact command and arguments for review. The `--dangerously-run-mcp-servers` flag bypasses this prompt for CI/CD use.

## Quick Start

```bash
# Sign up at Snyk and obtain API token from https://app.snyk.io/account
export SNYK_TOKEN=your-api-token-here
# Install uv package manager if needed
uvx snyk-agent-scan@latest
```

## CLI Commands

### scan (default)
```bash
snyk-agent-scan scan [CONFIG_FILE...]
  --checks-per-server NUM
  --server-timeout SECONDS
  --dangerously-run-mcp-servers
  --no-skills
```

### inspect
Lists tools/prompts/resources without verification:
```bash
snyk-agent-scan inspect [CONFIG_FILE...]
```

## Usage Examples

```bash
# Full machine scan
uvx snyk-agent-scan@latest

# Specific config
uvx snyk-agent-scan@latest ~/.vscode/mcp.json

# Single skill
uvx snyk-agent-scan@latest ~/path/to/SKILL.md

# Directory scan
uvx snyk-agent-scan@latest ~/.claude/skills
```

## Supported Agents

Coverage varies by OS (macOS, Linux, Windows):
- Windsurf, Cursor, VS Code, Claude Desktop, Claude Code
- Gemini CLI, OpenClaw, Amp, Kiro, Amazon Q, and others

## Data Privacy

Agent Scan does not store or log any usage data from MCP tool calls. Skills, tool names, and descriptions are shared with Snyk's verification service.
