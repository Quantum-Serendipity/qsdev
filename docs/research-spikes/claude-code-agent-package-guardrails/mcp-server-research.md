# MCP Server Integration for Package Security

## Executive Summary

MCP (Model Context Protocol) servers provide a viable and powerful mechanism for wrapping package installation with pre-flight security checks in Claude Code. Two production-ready security MCP servers already exist (Socket.dev and Snyk), and the official TypeScript SDK makes building a custom server straightforward. However, MCP servers alone cannot prevent agents from bypassing them via raw Bash commands — they must be paired with PreToolUse hooks and/or permission deny rules to form a complete enforcement layer. The most robust architecture combines a custom MCP server (for security validation logic) with a PreToolUse hook (for interception and enforcement) and permission rules (for defense-in-depth).

## 1. How MCP Servers Are Configured in Claude Code

### Configuration Methods

Claude Code supports three ways to add MCP servers:

1. **CLI command**: `claude mcp add --transport <type> <name> [options] -- <command> [args]`
2. **Project `.mcp.json`**: Shared with team via version control
3. **Direct JSON editing**: In `~/.claude.json`, `~/.claude/settings.json`, `.claude/settings.local.json`, or `~/.claude/mcp_servers.json`

### Transport Types

| Transport | Status | Use Case |
|-----------|--------|----------|
| **stdio** | Active, recommended for local | Server runs as child process, communicates via stdin/stdout |
| **HTTP (Streamable HTTP)** | Active, recommended for remote | HTTP-based, supports reconnection with exponential backoff |
| **SSE** | Deprecated | Server-Sent Events; legacy, being replaced by Streamable HTTP |

### Configuration Structure

```json
{
  "mcpServers": {
    "my-security-server": {
      "command": "node",
      "args": ["./security-server.js"],
      "env": {
        "API_KEY": "${SECURITY_API_KEY}"
      }
    }
  }
}
```

For remote servers:
```json
{
  "mcpServers": {
    "socket-mcp": {
      "type": "http",
      "url": "https://mcp.socket.dev/"
    }
  }
}
```

### Scoping

| Scope | Loads in | Shared | Stored in |
|-------|----------|--------|-----------|
| Local (default) | Current project only | No | `~/.claude.json` |
| Project | Current project only | Yes | `.mcp.json` in project root |
| User | All projects | No | `~/.claude.json` |

### Tool Discovery

- Claude Code discovers tools from connected MCP servers at startup
- Setting `alwaysLoad: true` ensures tools are always in context (useful for a small number of critical tools)
- Otherwise, tools are discovered via tool search when relevant
- MCP tools appear as `mcp__<serverName>__<toolName>` in the tool system
- Dynamic tool updates supported via `list_changed` notifications

### Key Environment

- `CLAUDE_PROJECT_DIR` is set in the spawned server's environment
- `MCP_TIMEOUT` controls startup timeout (default varies)
- `MAX_MCP_OUTPUT_TOKENS` controls output size warnings (default 10,000 tokens)
- Environment variable expansion supported in `.mcp.json`: `${VAR}` and `${VAR:-default}`

**Sources**: `docs/claude-code-mcp-configuration.md`, `docs/mcp-local-server-connection.md`

## 2. Existing Security-Focused MCP Servers

### Socket.dev MCP Server

**Status**: Production-ready, actively maintained (102+ GitHub stars, MIT license)

**What it does**: Provides dependency security scoring across multiple ecosystems. Exposes a `depscore` tool that returns five scoring dimensions: supply chain risk, code quality, maintenance status, vulnerability evaluation, and license compatibility.

**Tool interface**:
```
depscore({
  packages: [
    { ecosystem: "npm", depname: "express", version: "4.18.2" }
  ]
})
```

**Supported ecosystems**: npm, PyPI, Cargo, and others.

**Deployment options**:
1. **Public hosted** (zero setup): `https://mcp.socket.dev/` — no API key needed
2. **Stdio** (local): `npx @socketsecurity/mcp@latest` with `SOCKET_API_KEY`
3. **HTTP** (local server): `MCP_HTTP_MODE=true npx @socketsecurity/mcp@latest --http`

**Claude Code setup**:
```bash
# Public hosted (recommended for quick start)
claude mcp add --transport http socket-mcp https://mcp.socket.dev/

# Local with API key
claude mcp add socket-mcp -e SOCKET_API_KEY="your-key" -- npx -y @socketsecurity/mcp@latest
```

**Strengths**: Zero-auth public server, batch processing, multi-ecosystem. **Limitations**: Read-only scoring (doesn't block installs), no CVE-level detail, scores may not distinguish between "known CVE" and "low maintenance quality."

### Snyk MCP Server

**Status**: Production, bundled with Snyk CLI (v1.1298.0+), experimental mode

**What it does**: Full security scanning suite — code vulnerabilities, dependency scanning (SCA), IaC scanning, container scanning, SBOM analysis.

**Key tools exposed**:
- `snyk_sca_scan` — Open source dependency vulnerability detection
- `snyk_code_scan` — First-party code security analysis
- `snyk_iac_scan` — Infrastructure as Code scanning
- `snyk_container_scan` — Container image vulnerabilities
- `snyk_sbom_scan` — SBOM analysis
- `snyk_aibom` — AI Bill of Materials

**Configuration**:
```json
{
  "mcpServers": {
    "SnykMCP": {
      "command": "snyk",
      "args": ["mcp", "-t", "stdio"],
      "env": { "SNYK_TOKEN": "${SNYK_TOKEN}" }
    }
  }
}
```

Or with npx: `npx -y snyk@latest mcp -t stdio`

**Strengths**: Comprehensive scanning (not just dependencies), automated remediation suggestions, CLI bundled. **Limitations**: Requires Snyk account/token, experimental mode, scans the project after the fact rather than intercepting individual installs.

### Snyk Agent Scan

**Status**: Separate tool for auditing MCP server configurations themselves

**What it does**: Discovers and scans agent configurations (Claude Code, Cursor, etc.) for security vulnerabilities in MCP servers and skills. Detects prompt injection, tool poisoning, tool shadowing, toxic flows, malware payloads, hardcoded secrets.

**Not an install-time checker** — this is a meta-security tool for auditing your MCP setup.

**Sources**: `docs/socket-dev-mcp-server.md`, `docs/snyk-mcp-server.md`, `docs/snyk-mcp-cheat-sheet.md`, `docs/snyk-agent-scan.md`

## 3. The MCP Server Protocol: Tools, Discovery, and Usage

### What Tools a Server Can Expose

An MCP server can expose three types of capabilities:
1. **Tools** — Functions the AI can call (the primary mechanism for package security)
2. **Resources** — Data the AI can read (files, database records, etc.)
3. **Prompts** — Reusable prompt templates

Tools are the relevant capability for package security. Each tool has:
- A **name** (used in tool calls and permission rules)
- A **description** (used by the AI to understand when to use it)
- An **input schema** (validated by the server, defined with Zod or Standard Schema)
- A **handler function** (executes the logic and returns structured content)

### How Claude Code Discovers and Uses Tools

1. Claude Code starts the MCP server process (for stdio) or connects (for HTTP)
2. Protocol handshake: capability negotiation
3. Claude Code calls `tools/list` to discover available tools
4. Tools are registered in the AI's context (with name, description, schema)
5. When the AI decides to use a tool, Claude Code calls `tools/call` on the server
6. Server validates input, executes logic, returns structured content
7. Claude Code presents the result to the AI

### Building a Custom MCP Server (TypeScript)

The official `@modelcontextprotocol/sdk` makes this straightforward:

```typescript
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

const server = new McpServer({
  name: "package-security",
  version: "1.0.0"
});

server.tool(
  "check_package",
  "Check a package for known vulnerabilities before installation",
  {
    name: z.string().describe("Package name"),
    version: z.string().optional().describe("Version to check"),
    ecosystem: z.enum(["npm", "pypi", "cargo", "crates.io"]).describe("Package ecosystem")
  },
  async ({ name, version, ecosystem }) => {
    // Query OSV.dev, Socket.dev, etc.
    const result = await checkVulnerabilities(name, version, ecosystem);
    return {
      content: [{ type: "text", text: JSON.stringify(result) }]
    };
  }
);

server.tool(
  "install_package",
  "Install a package after security validation. Always use this instead of raw npm/pip/cargo install.",
  {
    name: z.string(),
    version: z.string().optional(),
    ecosystem: z.enum(["npm", "pypi", "cargo"]),
    devDependency: z.boolean().optional()
  },
  async ({ name, version, ecosystem, devDependency }) => {
    // 1. Check vulnerabilities
    // 2. If clean, execute install command
    // 3. Log the attempt
    // 4. Return structured result
  }
);

const transport = new StdioServerTransport();
await server.connect(transport);
```

**Requirements**: Node.js 18+, TypeScript with `target: "ES2022"`, `module: "NodeNext"`.

**Python SDK** also available (`mcp` package on PyPI) with equivalent capabilities.

**Sources**: `docs/mcp-typescript-sdk.md`

## 4. Feasibility Assessment: Custom Package Security MCP Server

### Architecture: "install_package" Tool

**Fully feasible.** A custom MCP server can expose an `install_package` tool that:

1. **Receives** package name, version, ecosystem from the agent
2. **Queries vulnerability databases** (OSV.dev, Socket.dev, Snyk) in parallel
3. **Applies policy rules** (block critical CVEs, warn on medium, allow clean)
4. **Executes the install** if approved (via child process: `npm install`, `pip install`, etc.)
5. **Returns structured results** including allow/deny decision, reasons, CVE details
6. **Logs all attempts** to an audit file

### Vulnerability Database Integration

| Database | Auth Required | Rate Limits | Latency | Best For |
|----------|---------------|-------------|---------|----------|
| **OSV.dev** | No | None | ~100-500ms | CVE/vulnerability lookup, free, comprehensive |
| **Socket.dev** | No (public) or API key | Unknown | ~200-800ms | Supply chain risk scoring, quality metrics |
| **Snyk** | Yes (token) | Varies by plan | ~500ms-2s | Comprehensive scanning with remediation |
| **npm audit** | No | N/A (local) | ~1-3s | npm-specific, runs locally |
| **pip-audit** | No | N/A (local) | ~1-5s | PyPI-specific, runs locally |
| **cargo-audit** | No | N/A (local) | ~1-3s | Rust-specific, runs locally |

**Recommended approach**: Use OSV.dev as the primary database (free, no auth, no rate limits, comprehensive) with Socket.dev as supplementary (supply chain scoring). Fall back to local tools (npm audit, pip-audit) when network is unavailable.

### OSV.dev Query Example

```bash
curl -d '{
  "package": {"name": "lodash", "ecosystem": "npm"},
  "version": "4.17.20"
}' "https://api.osv.dev/v1/query"
```

Supports batch queries via `/v1/querybatch` for checking multiple packages at once.

### Structured Allow/Deny Decision

```typescript
interface InstallDecision {
  allowed: boolean;
  package: string;
  version: string;
  ecosystem: string;
  reasons: string[];
  vulnerabilities: {
    id: string;       // e.g., "GHSA-xxxx" or "CVE-2024-xxxx"
    severity: "critical" | "high" | "medium" | "low";
    summary: string;
    fixedIn?: string; // version with fix
  }[];
  supplyChainScore?: number; // from Socket.dev
  auditLogEntry: string;     // reference to audit log
}
```

### Audit Logging

The MCP server can append to a structured log file (JSON lines or similar):

```typescript
{
  timestamp: "2026-05-12T14:30:00Z",
  action: "install_attempt",
  package: "lodash@4.17.20",
  ecosystem: "npm",
  decision: "denied",
  reasons: ["CVE-2021-23337: Command Injection (high severity)"],
  session_id: "abc123",
  project: "/home/user/my-project"
}
```

## 5. Making Agents Prefer/Require the MCP Tool Over Raw Bash

This is the critical enforcement challenge. An MCP server alone is advisory — the agent can still run `npm install malicious-package` via the Bash tool. Three complementary mechanisms create a complete enforcement layer:

### Layer 1: Permission Deny Rules (Block Raw Installs)

Deny raw package install commands via settings.json:

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)",
      "Bash(npm i *)",
      "Bash(npm add *)",
      "Bash(pip install *)",
      "Bash(pip3 install *)",
      "Bash(cargo install *)",
      "Bash(cargo add *)",
      "Bash(pnpm add *)",
      "Bash(pnpm install *)",
      "Bash(yarn add *)",
      "Bash(bun add *)",
      "Bash(bun install *)"
    ],
    "allow": [
      "mcp__package_security__install_package",
      "mcp__package_security__check_package"
    ]
  }
}
```

**Strength**: Deterministic, enforced by Claude Code runtime. **Weakness**: Pattern matching is fragile — the agent could use `npx`, `node -e "require('child_process').exec('npm install ...')"`, pipe commands, or environment variable indirection to bypass.

### Layer 2: PreToolUse Hook (Intercept and Validate)

A PreToolUse hook on Bash that parses the command and blocks package install patterns:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/block-raw-installs.sh"
          }
        ]
      }
    ]
  }
}
```

The hook script can:
- Parse the command for install patterns (more sophisticated than glob matching)
- Check for evasion attempts (backticks, subshells, variable expansion)
- Return a structured deny decision with a message telling the agent to use the MCP tool instead
- Log blocked attempts

**Strength**: Runs deterministically, can use complex validation logic, provides feedback to the agent. **Weakness**: Still pattern-matching shell commands, sophisticated evasion possible.

### Layer 3: MCP Tool Hook (Delegate to MCP Server for Validation)

Claude Code supports `"type": "mcp_tool"` hooks, which can call an MCP server tool as part of a PreToolUse hook:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "mcp_tool",
            "server": "package_security",
            "tool": "validate_bash_command",
            "input": { "command": "${tool_input.command}" }
          }
        ]
      }
    ]
  }
}
```

**Critical limitation**: MCP tool hooks are **non-blocking on failure**. If the server is not connected or the tool returns an error, execution continues (the command is allowed through). For security-critical use, wrap in a command hook that fails-closed.

### Layer 4: CLAUDE.md Instructions (Soft Enforcement)

```markdown
## Package Installation Policy

NEVER use `npm install`, `pip install`, `cargo add`, or any raw package install command.
ALWAYS use the `install_package` MCP tool for ALL package installations.
The MCP tool performs security validation before installing.
```

**Strength**: Shapes agent behavior. **Weakness**: Non-deterministic, can be overridden by prompt injection or agent reasoning.

### Recommended Combined Architecture

```
Agent wants to install a package
         │
         ▼
    ┌─────────────┐
    │ CLAUDE.md    │ ← Soft: "Use install_package tool"
    │ instructions │
    └──────┬──────┘
           │ Agent chooses tool
           ▼
    ┌──────────────────┐     ┌──────────────────────┐
    │ Uses MCP tool?   │─Yes─│ MCP: install_package  │
    │                  │     │ - Query OSV.dev       │
    └──────┬───────────┘     │ - Check Socket.dev    │
           │ No (raw Bash)   │ - Apply policy        │
           ▼                 │ - Execute if clean    │
    ┌──────────────────┐     │ - Log attempt         │
    │ Permission deny  │     └──────────────────────┘
    │ rules            │
    │ (Bash(npm i *))  │
    └──────┬───────────┘
           │ If somehow bypassed
           ▼
    ┌──────────────────┐
    │ PreToolUse hook  │
    │ (parse command,  │
    │  block installs) │
    └──────────────────┘
```

## 6. MCP Server Architecture Patterns

### Pattern 1: Pure Validation Server (Recommended for Security)

The server only checks packages — it does not execute installs. The agent must then use Bash to install (after getting a "clean" result). This separates concerns but requires the hooks layer to ensure the agent actually checks before installing.

### Pattern 2: Install Wrapper Server (Most Secure)

The server checks AND installs. The agent calls `install_package` and gets back either an error (blocked) or a success (installed). Combined with deny rules blocking raw installs, this is the most complete solution.

### Pattern 3: Sidecar Validation (Hook-Centric)

No custom MCP server needed. A PreToolUse hook intercepts Bash install commands, calls OSV.dev directly (via curl in the hook script), and allows/denies based on the response. Simpler but less flexible.

## 7. Limitations and Failure Modes

### Latency

- OSV.dev queries: ~100-500ms per package
- Socket.dev scoring: ~200-800ms per package
- Total overhead per install: ~300ms-2s (acceptable for interactive development)
- Batch queries reduce latency for multi-package installs

### Reliability and Failure Modes

| Failure | Impact | Mitigation |
|---------|--------|------------|
| MCP server crashes | MCP tool hooks fail-open (non-blocking) | Use command hook wrapper that fails-closed |
| Vulnerability DB down | Cannot validate packages | Cache recent results; fall back to local audit tools |
| Stdio server exits | Not reconnected automatically | Wrap in process supervisor; user can restart via `/mcp` |
| HTTP server disconnects | Auto-reconnected (5 retries, exponential backoff) | Use HTTP transport for better resilience |
| Network unavailable | Cannot reach OSV.dev/Socket.dev | Fall back to local npm-audit/pip-audit; optionally fail-closed |

### Agent Bypass Potential

| Bypass Vector | Likelihood | Mitigation |
|---------------|-----------|------------|
| Direct `npm install` via Bash | High (without deny rules) | Permission deny rules + PreToolUse hook |
| `npx` to run arbitrary packages | Medium | Include `Bash(npx *)` in deny rules (may break legitimate use) |
| `node -e` or `python -c` with subprocess | Low | Hard to pattern-match; sandboxing helps |
| Editing package.json + running `npm install` (no args) | Medium | Hook on bare `npm install`; PostToolUse hook to diff lockfile |
| Subagent running installs | Low | Hooks fire for subagent tool calls too |
| Prompt injection via malicious README | Low-Medium | MCP tool hook provides structure, but agent reasoning can be influenced |

### MCP Server Security Concerns

The MCP ecosystem itself has security issues (per 2026 audits):
- 118 security findings across 68 MCP server packages
- 82% of implementations use file operations prone to path traversal
- 67% use APIs related to code injection
- A custom security server should be audited with `snyk-agent-scan` before deployment

## 8. Complete Implementation Sketch

### File Structure

```
.claude/
├── hooks/
│   └── block-raw-installs.sh
├── settings.json          # Permission rules + hook config
└── mcp-servers/
    └── package-security/  # Custom MCP server
        ├── package.json
        ├── tsconfig.json
        └── src/
            ├── index.ts       # Server entry point
            ├── tools/
            │   ├── check-package.ts
            │   └── install-package.ts
            ├── validators/
            │   ├── osv.ts      # OSV.dev client
            │   ├── socket.ts   # Socket.dev client
            │   └── policy.ts   # Allow/deny policy engine
            └── audit/
                └── logger.ts   # Structured audit logging
```

### settings.json

```json
{
  "permissions": {
    "deny": [
      "Bash(npm install *)",
      "Bash(npm i *)",
      "Bash(npm add *)",
      "Bash(pip install *)",
      "Bash(pip3 install *)",
      "Bash(uv pip install *)",
      "Bash(cargo add *)",
      "Bash(cargo install *)",
      "Bash(pnpm add *)",
      "Bash(yarn add *)",
      "Bash(bun add *)"
    ],
    "allow": [
      "mcp__package_security__install_package",
      "mcp__package_security__check_package",
      "mcp__package_security__audit_log"
    ]
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/block-raw-installs.sh"
          }
        ]
      }
    ]
  },
  "mcpServers": {
    "package_security": {
      "command": "node",
      "args": ["$CLAUDE_PROJECT_DIR/.claude/mcp-servers/package-security/dist/index.js"],
      "env": {
        "SOCKET_API_KEY": "${SOCKET_API_KEY:-}",
        "AUDIT_LOG_PATH": "${CLAUDE_PROJECT_DIR:-.}/.claude/package-audit.jsonl"
      },
      "alwaysLoad": true
    }
  }
}
```

### Hook Script (block-raw-installs.sh)

```bash
#!/usr/bin/env bash
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Patterns that indicate package installation
INSTALL_PATTERNS=(
  "npm install " "npm i " "npm add "
  "pip install " "pip3 install " "uv pip install "
  "cargo add " "cargo install "
  "pnpm add " "pnpm install "
  "yarn add "
  "bun add " "bun install "
)

for pattern in "${INSTALL_PATTERNS[@]}"; do
  if echo "$COMMAND" | grep -qi "$pattern"; then
    cat >&2 <<EOF
BLOCKED: Direct package installation detected.
Command: $COMMAND

Use the install_package MCP tool instead:
  mcp__package_security__install_package({
    name: "<package>",
    ecosystem: "npm|pypi|cargo",
    version: "<version>"
  })

This tool performs security validation (CVE checks, supply chain scoring)
before allowing the installation.
EOF
    exit 2
  fi
done

# Also catch bare "npm install" or "pip install" (no args = install from lockfile)
# These are generally safe but should still be logged
if echo "$COMMAND" | grep -qE "^(npm install|pip install|cargo build)$"; then
  # Allow but could log via PostToolUse hook
  exit 0
fi

exit 0
```

## 9. Comparison of Approaches

| Approach | Setup Effort | Security Strength | Flexibility | Maintenance |
|----------|-------------|-------------------|-------------|-------------|
| Socket.dev MCP only | Low (10 min) | Low (advisory only) | Low | None |
| Snyk MCP only | Low-Medium | Low-Medium (scans after install) | Medium | Snyk account |
| Permission deny rules only | Low | Medium (pattern bypass possible) | Low | Low |
| PreToolUse hook only | Medium | Medium-High | Medium | Medium |
| Custom MCP + deny rules + hook | High (days) | High | High | Medium-High |
| Custom MCP + hook + managed settings | High | Highest (enterprise) | High | High |

## 10. Conclusions

1. **A custom MCP server is feasible and the right tool for the validation logic layer.** The TypeScript SDK is mature, OSV.dev provides a free/unlimited vulnerability API, and the server architecture is straightforward.

2. **MCP alone is insufficient for enforcement.** The agent can bypass an MCP tool by using raw Bash. You must combine MCP with permission deny rules and/or PreToolUse hooks.

3. **The `"type": "mcp_tool"` hook is a powerful integration point** that lets PreToolUse hooks delegate validation to an MCP server — but it fails-open on error, which is dangerous for security. Wrap in a command hook for fail-closed behavior.

4. **Socket.dev's public MCP server is the fastest path to value.** Zero setup, no API key, provides supply chain scoring. Good for immediate use while building a custom solution.

5. **Snyk MCP is more comprehensive but solves a different problem** — it scans projects for existing vulnerabilities rather than intercepting individual package installs. Useful as a complementary post-install check.

6. **The biggest gap is install-via-lockfile.** An agent can edit `package.json` and run bare `npm install`, bypassing per-package checks. A PostToolUse hook that diffs the lockfile before/after could catch this, but adds complexity.

7. **Enterprise managed settings** (`allowManagedPermissionRulesOnly`, `allowManagedMcpServersOnly`) provide the strongest enforcement — deny rules that cannot be overridden by project or user settings.

## Open Questions

- Can the `dontAsk` permission mode combined with explicit MCP tool allow rules create a strict "only approved tools" environment?
- How does sandboxing interact with MCP server processes? (The server runs outside the sandbox but its child processes may not.)
- What is the real-world latency of OSV.dev batch queries for typical install operations (5-20 packages)?
- Can PostToolUse hooks reliably detect lockfile changes to catch the "edit package.json + npm install" bypass?
- Is there an existing MCP server that wraps install commands (not just scanning)?

## Sources

| Document | Path |
|----------|------|
| Claude Code MCP configuration | `docs/claude-code-mcp-configuration.md` |
| Socket.dev MCP server | `docs/socket-dev-mcp-server.md` |
| Snyk MCP server | `docs/snyk-mcp-server.md` |
| Snyk MCP cheat sheet | `docs/snyk-mcp-cheat-sheet.md` |
| Snyk Agent Scan | `docs/snyk-agent-scan.md` |
| Claude Code hooks guide | `docs/claude-code-hooks-guide.md` |
| MCP tool hooks | `docs/claude-code-mcp-tool-hooks.md` |
| Claude Code permissions | `docs/claude-code-permissions.md` |
| MCP TypeScript SDK | `docs/mcp-typescript-sdk.md` |
| OSV.dev API | `docs/osv-dev-api.md` |
| MCP local server connection | `docs/mcp-local-server-connection.md` |
