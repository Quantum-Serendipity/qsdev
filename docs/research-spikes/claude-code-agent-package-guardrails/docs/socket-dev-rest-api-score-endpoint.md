<!-- Source: https://docs.socket.dev/reference/getscorebynpmpackage -->
<!-- Retrieved: 2026-05-12 -->

# Socket.dev REST API: Get Score by Package

## Endpoint Details

**URL Format:** `https://api.socket.dev/v0/npm/{package}/{version}/score`

**HTTP Method:** GET

**Status:** Deprecated (use successor batch endpoint instead)

## Authentication

- Required: Yes (for REST API; public MCP server at mcp.socket.dev does NOT require auth)
- Type: Bearer token authentication
- Scopes: No scopes required, but authentication is required

## Request Parameters

**Path Parameters:**
- `package` (string, required): Package name
- `version` (string, required): Package version

## Response Format

Returns JSON containing comprehensive package evaluation metrics across five categories:

1. **depscore** (0-1): Overall average score
2. **supplyChainRisk** (0-1): Supply chain security factors
3. **quality** (0-1): Code quality metrics
4. **maintenance** (0-1): Maintenance activity indicators
5. **vulnerability** (0-1): Security vulnerability assessments
6. **license** (0-1): Licensing factors
7. **miscellaneous**: Package metadata

Each category includes specific metrics (e.g., issue counts by severity, dependency counts, lines of code).

## Rate Limiting

This endpoint consumes 1 unit of your quota.

Response codes indicate quota status:
- 429: Insufficient quota for API route
- 403: Insufficient max_quota for API method

## Available Ecosystems

- npm (referenced in URL structure; other ecosystems via different endpoints)

## Successor Information

Deprecated in favor of a batch package fetch endpoint available in the API reference documentation.
