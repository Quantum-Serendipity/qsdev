<!-- Source: https://github.com/metatool-ai/metamcp -->
<!-- Retrieved: 2026-05-14 -->

# MetaMCP: MCP Aggregator, Orchestrator, Middleware, Gateway

MetaMCP is a MCP proxy that lets you dynamically aggregate MCP servers into a unified MCP server, and apply middlewares. MetaMCP itself is a MCP server so it can be easily plugged into ANY MCP clients.

## Architecture

**Stack Components:**
- Frontend: Next.js
- Backend: Express.js with tRPC
- Authentication: Better Auth
- Structure: Monorepo using Turborepo with Docker publishing

The system maintains idle sessions for each configured MCP server to minimize cold start delays, with a default of one idle session per configured server.

## Aggregation Mechanism

1. **Server Configuration**: Administrators define MCP servers (typically STDIO-based) with command and argument specifications
2. **Namespace Grouping**: One or more servers organize into namespaces with shared middleware and tool overrides
3. **Endpoint Creation**: Namespaces map to endpoints exposed via SSE, Streamable HTTP, or OpenAPI protocols
4. **Tool Aggregation**: When a client requests tools, MetaMCP queries each connected server, aggregates responses, applies middleware transformations, then returns a unified list

## Configuration Format

```json
"ServerName": {
  "type": "STDIO",
  "command": "uvx",
  "args": ["mcp-package-name"]
}
```

Environment variables support raw values or ${ENV_VAR_NAME} reference syntax.

## Middleware Capabilities

Applied at namespace level to intercept and transform requests/responses. Current: "Filter inactive tools" for LLM context optimization. Future: tool logging, error tracing, validation, security scanning.

## Tool Management

- Edit tool display names, titles, and descriptions per namespace
- Attach custom MCP annotations
- Metadata badges indicate overridden or annotated tools
- Annotation merging preserves upstream server metadata while adding namespace-specific hints

## Protocol Support

- Tools, Resources, and Prompts
- OAuth-enabled MCP servers
- MCP Spec 2025-06-18 standard OAuth

## Authentication

- API key via Authorization: Bearer header
- Session cookies
- MCP OAuth (standard spec)
- OpenID Connect (OIDC)

## Traffic Management

- Endpoint rate-limiting: Shared counter, returns 503
- User rate-limiting: Per-individual counters, returns 429
- Token bucket algorithm with configurable refill rates

## Deployment

Docker Compose recommended. Includes PostgreSQL. Minimum 2GB-4GB memory.

## Stats

2.3k stars, 341 forks, 79 open issues, 32 releases (latest v2.4.22). TypeScript 97.9%.
