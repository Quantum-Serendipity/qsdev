# @yawlabs/mcp-compliance README
- **Source**: https://github.com/YawLabs/mcp-compliance (via GitHub API)
- **Retrieved**: 2026-05-14

## Overview

**Test any MCP server for spec compliance.** 88-test suite covering transport, lifecycle, tools, resources, prompts, error handling, schema validation, and security against the MCP specification (2025-11-25). Works against HTTP endpoints and stdio servers alike. CLI, MCP server, and programmatic API.

- **Language**: TypeScript
- **License**: MIT
- **Stars**: 2
- **Created**: 2026-04-06
- **Last Updated**: 2026-05-14

## Why This Tool?

MCP servers are multiplying fast but most ship without compliance testing. Broken transport handling, missing error codes, malformed schemas, and silent capability violations are common.

## Features

- **88 tests across 8 categories** — transport, lifecycle, tools, resources, prompts, error handling, schema validation, and security
- **Capability-driven** — tests adapt to what the server declares. No false failures for undeclared features.
- **Graded scoring** — A-F letter grade with weighted score (required tests 70%, optional 30%)
- **CI-ready** — `--strict` mode exits with code 1 on required test failures
- **Spec-referenced** — every test links to exact MCP spec section
- **Three interfaces** — CLI for humans, MCP server for AI assistants, programmatic API for integration
- **Published methodology** — testing methodology and rule catalog are open (CC BY 4.0)

## Test Categories (8)

1. Transport
2. Lifecycle
3. Tools
4. Resources
5. Prompts
6. Error handling
7. Schema validation
8. Security

HTTP runs all 85 transport-applicable tests; stdio runs ~75 (HTTP-specific tests like CORS, TLS, session headers, rate limiting are gated out).

## Quick Start

```bash
# Remote HTTP server
npx @yawlabs/mcp-compliance test https://my-server.com/mcp

# Local stdio server
npx @yawlabs/mcp-compliance test npx @modelcontextprotocol/server-filesystem /tmp

# With env vars
npx @yawlabs/mcp-compliance test -E GITHUB_TOKEN=$GITHUB_TOKEN -- npx @modelcontextprotocol/server-github
```

## Output Formats

- terminal (default, colored)
- json
- sarif (for GitHub Code Scanning)
- github (::error/::warning annotations)
- markdown
- html (self-contained)

## CI Integration

GitHub Action available:
```yaml
- uses: YawLabs/mcp-compliance@v0
  with:
    target: 'node ./dist/server.js'
    format: github
    strict: 'true'
    min-grade: 'A'
```

Also supports: Docker, config files, watch mode, latency benchmarking, diff between runs.

## Additional CLI Commands

- `mcp-compliance init` — scaffold config interactively
- `mcp-compliance diff` — compare two runs, exit 1 if regressions
- `mcp-compliance benchmark` — latency testing with configurable requests and concurrency
- `mcp-compliance test --list` — preview which tests would run
