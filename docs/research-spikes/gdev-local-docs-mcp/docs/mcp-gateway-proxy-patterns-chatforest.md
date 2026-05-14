<!-- Source: https://chatforest.com/guides/mcp-gateway-proxy-patterns/ -->
<!-- Retrieved: 2026-05-14 -->

# MCP Gateway & Proxy Patterns: Aggregating, Securing, and Scaling MCP Servers

## Core Gateway Architecture Patterns

### Transport Bridging
Converts between incompatible MCP transports:
- **stdio → HTTP**: Local proxy receives stdio connections from Claude Desktop, forwards requests over HTTP to remote servers
- **HTTP → stdio**: HTTP server spawns/connects to local stdio-based servers, enabling remote client access

Key tools:
- **supergateway** (2,500+ stars, TypeScript) - bidirectional stdio/SSE bridging
- **mcp-proxy** (2,400+ stars, Python) - Streamable HTTP and stdio with OAuth2 support

### Server Aggregation Pattern
Single gateway endpoint connects multiple backend MCP servers, presenting unified tool catalog:

```
┌──────────┐     ┌───────────────┐     ┌──────────────┐
│  Claude  │     │   Gateway     │────▶│  GitHub MCP   │
│  Client  │────▶│  (single MCP  │────▶│  Slack MCP    │
│          │     │   endpoint)   │────▶│  Database MCP │
└──────────┘     └───────────────┘     └──────────────┘
```

Gateway routes tool calls to appropriate backend servers; clients see unified endpoint.

**Implementations:**
- **IBM ContextForge** (3,500+ stars) - federates MCP, A2A, REST/gRPC with plugins and guardrails
- **MetaMCP** (2,200+ stars) - Docker-based aggregator with middleware for dynamic filtering
- **Docker MCP Gateway** (1,300+ stars) - isolated containers with resource limits, 300+ verified servers
- **combine-mcp** (30+ stars) - minimal Go tool merging stdio servers

### Security Gateway Pattern
Intercepts every MCP message, applies policies before forwarding:
- Tool call approval requirements
- PII detection in requests/responses
- Server-side credential injection (prevents agent visibility)
- Guardrails filtering harmful content
- Comprehensive audit logging

**Tools:**
- **Lasso MCP Gateway** (360+ stars) - plugin-based with token masking and PII detection via Presidio
- **MCP Guardian** (190+ stars) - Rust proxy enabling real-time approval/denial
- **Peta** (40+ stars) - vault-backed credential injection, policy approvals, web admin console

### Cloud-Native Gateway
**Envoy AI Gateway** (1,500+ stars) uses token-encoding architecture: "encoding session state into the client session ID rather than maintaining a centralized session store. This eliminates the need for Redis or a database for session management, enabling horizontal scaling without external dependencies."

Provides connection management, load balancing, circuit breaking, rate limiting, observability.

## Reverse Proxy Configuration

### nginx for MCP/SSE
Critical setting: disable buffering to prevent SSE stream buffering:

```
location /mcp {
    proxy_pass http://localhost:3000;
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 300s;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

**Key detail**: SSE requires empty Connection header, not WebSocket upgrade syntax.

### Streamable HTTP Transport
Newer transport operates over standard HTTP/HTTPS, eliminating SSE buffering workarounds.

### Reverse Proxy Selection
- **Caddy** - simplest with automatic TLS
- **nginx** - battle-tested, widest docs, requires manual SSE config
- **Traefik** - optimal for Kubernetes/Docker with auto-discovery

## Security Considerations

**Tool poisoning prevention**: "gateways can intercept tool descriptions and compare cryptographic signatures against a trusted registration catalog. They can also sanitize descriptions by removing harmful directives, enforcing length limits, and filtering excessive privilege requests."

**Server isolation**: Docker MCP Gateway containers with restricted privileges, network access, resource limits contain compromise blast radius.

**Credential management**: "Instead of distributing API keys across client configurations (where AI agents can see them), gateways inject credentials server-side at execution time."

**Data loss prevention**: Guardrails scan MCP traffic for sensitive data.

**Audit/compliance**: Centralized logging via gateway provides single audit trail queryable by user, operation, timerange, policy decision; OpenTelemetry integration emerging standard.

## Managed Cloud Services

- **AWS**: Bedrock AgentCore Gateway converts APIs/Lambda to MCP tools; API Gateway added MCP proxy (Dec 2025)
- **Azure**: API Management for centralized auth; Azure Functions MCP extension (preview Apr 2025); mcp.azure.com curated center
- **Kong**: Gateway 3.12 (Oct 2025) - AI MCP Proxy plugin bridging MCP/HTTP, OAuth2 support
- **Other vendors**: Gravitee APIM 4.8, WSO2 MCP Gateway, Tyk AI Studio

## Edge-Based Patterns

**Cloudflare Workers MCP** (630+ stars): Multi-domain architecture with central Gateway Worker exposing `/mcp` endpoint, Domain Workers executing tools independently, central tool registry with scope checks, edge-based request routing and authorization.

## Selection Guide

- **Transport bridging only**: mcp-proxy or supergateway
- **Small team**: combine-mcp or MetaMCP
- **Enterprise security**: Lasso or Peta plus ContextForge/Docker MCP Gateway
- **Cloud-native scale**: Envoy AI Gateway or managed cloud services
- **Comprehensive solution**: IBM ContextForge
