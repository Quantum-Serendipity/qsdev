<!-- Source: https://github.com/modelcontextprotocol/typescript-sdk -->
<!-- Retrieved: 2026-05-12 -->

# MCP TypeScript SDK

The official TypeScript SDK for Model Context Protocol servers and clients.

## Core Packages

- **`@modelcontextprotocol/server`**: For building MCP servers
- **`@modelcontextprotocol/client`**: For building MCP clients

## Server Implementation

### Basic Server Setup

```typescript
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";

const server = new McpServer({ name: 'greeting-server', version: '1.0.0' });
```

### Tool Registration

Tools are registered using a method that accepts:
- Tool name and description
- Input schema (using Zod v4 or compatible Standard Schema libraries)
- Handler function for execution

```typescript
import { z } from "zod";

server.tool("greet", "Greet someone by name", {
  name: z.string()
}, async ({ name }) => {
  return {
    content: [{ type: "text", text: `Hello, ${name}!` }]
  };
});
```

### Transport Options

- **Stdio transport** (`StdioServerTransport`) — for local process communication
- **Streamable HTTP** — for web-based communication (recommended over deprecated SSE)
- Optional middleware packages for specific frameworks (Express, Hono, Node.js HTTP)

### Server Lifecycle

```typescript
const transport = new StdioServerTransport();
await server.connect(transport);
```

## Schema Validation

Tools use Standard Schema for input validation, supporting:
- Zod v4
- Valibot
- ArkType
- Other compatible validation libraries

## TypeScript Configuration

Required tsconfig settings:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext"
  }
}
```

Requires Node.js 18+.

## Key Architecture Points

- Servers expose tools, resources, and prompts
- Clients discover available capabilities via protocol handshake
- Tools have typed input schemas and return structured content
- Transport layer is abstracted (stdio for local, HTTP for remote)
- Servers can send notifications (e.g., `list_changed` for dynamic tool updates)
