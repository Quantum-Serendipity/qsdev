<!-- Source: https://github.com/mrjoshuak/godoc-mcp -->
<!-- Retrieved: 2026-05-14 -->

# godoc-mcp: Go Documentation MCP Server

## What It Does
`godoc-mcp` is "a Model Context Protocol (MCP) server that provides efficient access to Go documentation." It enables LLMs to understand Go projects with reduced token consumption by serving structured documentation instead of requiring full source file reading.

## Architecture & Documentation Access
The server operates as an MCP endpoint that:
- Queries Go packages using the `go doc` command
- Returns official package documentation in structured format
- Handles both local file paths and import paths (standard library and third-party)
- Automatically manages temporary Go module contexts for external packages
- Implements response caching for performance optimization

## Tools Provided

**`get_doc`**: Retrieves documentation for packages, types, functions, or methods with optional parameters for symbols, flags (`-all`, `-src`, `-u`, `-short`, `-c`), working directories, and pagination.

**`list_packages`**: Enumerates sub-packages under a given package path for discovery purposes.

## Technical Details
- **Language**: Go (99.0%), Dockerfile (1.0%)
- **License**: MIT
- **Stars**: 115
- **Transport Options**: stdio (default), SSE, HTTP
- **Docker Support**: Includes containerized deployment via `ghcr.io/mrjoshuak/godoc-mcp`
- **Integration**: Works with Claude Desktop, Claude Code, Docker MCP Gateway

## Key Features
- Token-efficient documentation retrieval
- Smart package discovery in multi-package projects
- Automatic module context setup
- No internet connection required
- Built-in caching and performance optimization
