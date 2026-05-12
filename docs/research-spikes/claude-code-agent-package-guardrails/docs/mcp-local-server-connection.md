<!-- Source: https://modelcontextprotocol.io/docs/develop/connect-local-servers -->
<!-- Retrieved: 2026-05-12 -->

# Connect to local MCP servers

Model Context Protocol (MCP) servers extend AI applications' capabilities by providing secure, controlled access to local resources and tools.

## How MCP Servers Work

MCP servers are programs that run on your computer and provide specific capabilities to AI tools through a standardized protocol. Each server exposes tools that the AI can use to perform actions, with user approval.

## Configuration Structure

The mcpServers configuration tells the client to start servers with specific parameters:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "/Users/username/Desktop",
        "/Users/username/Downloads"
      ]
    }
  }
}
```

Fields:
- `"command"`: The executable to run (e.g., `npx`, `python`, `node`)
- `"args"`: Arguments passed to the command
- `"env"`: Environment variables for the server process

## Protocol Flow

1. Client starts the server process (for stdio transport)
2. Client and server perform capability negotiation handshake
3. Server advertises available tools, resources, and prompts
4. Client discovers tools and makes them available to the AI model
5. AI model can request tool calls, which the client proxies to the server
6. User approval may be required before execution

## Security Considerations

- Only grant access to directories/resources you're comfortable with
- The server runs with your user account permissions
- All actions can require explicit user approval before execution

## Troubleshooting

- Check configuration file JSON syntax
- Ensure file paths are absolute, not relative
- Check server logs for connection errors
- Try manually running the server command to verify it works
- stdio servers: check that Node.js is installed (for npx-based servers)

## Configuration File Locations

- **Claude Desktop macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Desktop Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Claude Code**: `~/.claude.json`, `.mcp.json`, or via `claude mcp add`
