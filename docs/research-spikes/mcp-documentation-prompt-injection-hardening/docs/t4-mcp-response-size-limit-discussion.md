# Response Size Limit for MCP Responses - GitHub Discussion #2211

- **Source**: https://github.com/modelcontextprotocol/modelcontextprotocol/discussions/2211
- **Retrieved**: 2026-05-14

## Problem Statement
MCP servers can return oversized responses without pagination, causing multiple issues when AI agents forward them to models. This leads to "Context limit exceeded errors, which may reset or break the active agent/session" and "unnecessary token usage and increased credit/cost consumption."

## Proposed Solutions

**Client-Side Enforcement:**
Maximum response size validation at the MCP client level with configurable limits of "256 KB-512 KB (configurable)" before responses reach model context.

**Server-Side Awareness:**
Servers could handle large content "out of band if it is larger than the max size" through mechanisms like download links or file locations for local access.

**Capability Negotiation:**
Clients should declare `max_response_bytes` during initialization, enabling servers to "handle the payload size limit with several strategies (pagination, summarization, error..)."

## Current Implementations

- Some hosts already store large outputs in temporary files accessible through other tools
- One developer created Sift, a gateway that "sits between the client and existing MCP servers, stores large tool results as artifacts, and returns a much smaller reference"

## Key Tensions

Collaborators debated whether clients should truncate responses versus servers providing complete data. The challenge: servers lack knowledge of what clients actually need, making server-side filtering potentially lossy.

## Consensus Status
No final agreement on mandatory maximum sizes; discussion remains focused on protocol-level support for size negotiation.
