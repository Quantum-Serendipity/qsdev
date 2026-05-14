<!-- Source: https://github.com/katvito/devdocs-mcp -->
<!-- Retrieved: 2026-05-14 -->

# katvito/devdocs-mcp - DevDocs MCP Server

## Purpose & Functionality

This MCP (Model Context Protocol) server enables AI editors like Claude and Cursor to access offline DevDocs documentation. It bridges AI editors with a local DevDocs instance, allowing developers to query technical documentation programmatically.

## Architecture Overview

The project follows **clean architecture principles** with distinct layers:

- **Application Layer**: Document management and error handling
- **Domain Layer**: Repository interfaces, value objects, and type definitions
- **Infrastructure Layer**: DevDocs repository implementation
- **MCP Layer**: Protocol handling, response conversion, and validation
- **Utilities**: Configuration and logging

## How It Works

The server operates through a Docker Compose setup that launches:
1. A **DevDocs container** (port 9292) serving documentation
2. An **MCP server container** that queries the DevDocs instance

The MCP server translates AI editor requests into documentation searches and returns formatted results.

## Exposed Tools & Resources

Two primary MCP tools are available:

1. **`view_available_docs`**: Lists supported documentation languages
2. **`search_specific_docs`**: Searches within specific documentation using a slug parameter

The system supports slash commands (e.g., `/devdocs/python-3.12`) for easier documentation access in compatible editors.

## Data Access Method

The server accesses DevDocs data through HTTP requests to the local DevDocs instance at `http://devdocs:9292`. It downloads and indexes documentation locally for offline searching.

## Installation & Setup

```
git clone https://github.com/katvito/devdocs-mcp.git
cd devdocs-mcp
cp .env.template .env
docker-compose up -d
```

Configuration varies by editor (Claude or Cursor), requiring the MCP server path and environment variables.

## Dependencies

- **Docker & Docker Compose**
- **Node.js 18+**
- **npm or yarn**

## Technical Stack

Languages: JavaScript (57.5%), TypeScript (40.3%), Dockerfile (1.3%), Shell (0.9%)

## Configuration

Environment variables control behavior:
- `LOG_LEVEL`: debug, info, warn, error
- `LOG_FORMAT`: json, text, plain
- `DEVDOCS_BASE_URL`: Default `http://devdocs:9292`
- `DOCUMENTS_PATH`: Storage location for downloaded documentation

## Transport Method

Uses **stdio transport** (shell script execution via `mcp-run.sh`), the standard MCP communication protocol for local servers.

## Security Considerations

- Local-only operation by default (no external API exposure)
- Docker containerization isolates services
- Environment variable configuration prevents hardcoded credentials
- No mention of authentication mechanisms for the MCP server itself

## Limitations

- Large initial DevDocs download (10+ minutes)
- Requires Docker infrastructure
- Limited to DevDocs-available documentation
- No built-in rate limiting or caching optimization mentioned

## Notable Features

- **Offline-first design**: Functions without internet after initial setup
- **AI editor integration**: Seamless Claude Code and Cursor support
- **Multi-language support**: Supports slash commands in English and Japanese
- **Development-ready**: Includes testing infrastructure (Jest) and linting

**License**: MIT
