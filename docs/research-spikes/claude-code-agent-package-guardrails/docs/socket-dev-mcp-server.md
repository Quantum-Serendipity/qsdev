<!-- Source: https://docs.socket.dev/docs/guide-to-socket-mcp + https://github.com/SocketDev/socket-mcp -->
<!-- Retrieved: 2026-05-12 -->

# Socket MCP Server

## What It Is

Socket MCP Server is a Model Context Protocol integration that enables AI assistants to check dependency vulnerability scores and security information. It's designed for seamless integration with Claude, VS Code Copilot, Cursor, and other MCP clients.

## Core Features

- **Dependency Security Scanning**: Comprehensive security metrics for npm, PyPI, Cargo, and other ecosystems
- **Public Hosted Service**: Available at `https://mcp.socket.dev/` with zero setup
- **Multiple Deployment Options**: Stdio, HTTP, or public service
- **Batch Processing**: Analyze multiple dependencies simultaneously
- **No Authentication for Public Server**: The hosted version requires no API keys

## Tool Exposed: `depscore`

The primary tool allows queries for dependency security information with these parameters:

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `packages` | Array | Yes | Package objects to analyze |
| `ecosystem` | String | No | Default: "npm" (supports pypi, cargo, etc.) |
| `depname` | String | Yes | Package name |
| `version` | String | No | Default: "unknown" |

### Security Metrics Returned

Results include five scoring dimensions:
- Supply chain risk assessment
- Code quality metrics
- Maintenance status
- Vulnerability evaluation
- License compatibility

### Example Query

"Check the security score for express version 4.18.2"
Response: `pkg:npm/express@4.18.2: supply_chain: 1.0, quality: 0.9, maintenance: 1.0, vulnerability: 1.0, license: 1.0`

## Deployment Options

### Option 1: Public Server (Recommended)
Configure any MCP client to connect to `https://mcp.socket.dev/` without authentication.

### Option 2a: Stdio Mode (Local)
Run via: `npx @socketsecurity/mcp@latest` with `SOCKET_API_KEY` environment variable set.

Configuration example:
```json
{
  "mcpServers": {
    "socket-mcp": {
      "command": "npx",
      "args": ["@socketsecurity/mcp@latest"],
      "env": { "SOCKET_API_KEY": "your-key" }
    }
  }
}
```

### Option 2b: HTTP Mode (Local Server)
Launch with: `MCP_HTTP_MODE=true SOCKET_API_KEY=key npx @socketsecurity/mcp@latest --http`

Key environment variables for HTTP mode:
- `SOCKET_API_KEY`: Required unless OAuth enabled
- `SOCKET_OAUTH_ISSUER`: For OAuth token validation
- `MCP_PORT`: Default 3000
- `TRUST_PROXY`: Enable behind reverse proxy

## Claude Code Setup

Install using:
```bash
claude mcp add socket-mcp -e SOCKET_API_KEY="your-key" -- npx -y @socketsecurity/mcp@latest
```

Or use the public hosted server (no API key needed):
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

## Health Check Endpoint

HTTP mode provides `GET /health` returning:
```json
{
  "status": "healthy",
  "service": "socket-mcp",
  "version": "0.0.3"
}
```

## Practical Usage Examples

- "Check the security score for express version 4.18.2"
- "Analyze the security of my package.json dependencies"
- "What are vulnerability scores for react, lodash, and axios?"

## Supported Ecosystems

npm, PyPI, Cargo, and other package management systems.

## Development Requirements

- Node.js v16+
- Direct TypeScript execution via Node's type-stripping feature
- MIT License, 102+ stars on GitHub
