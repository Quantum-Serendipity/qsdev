# ZIM MCP Server (zicojiao/zim-mcp-server)
> Source: https://github.com/zicojiao/zim-mcp-server
> Retrieved: 2026-05-14

## Description
A Model Context Protocol server enabling reading and searching of ZIM files -- an offline reference format developed by Kiwix for storing Wikipedia and comparable content.

## Key Information
- **Language:** TypeScript (100%)
- **License:** MIT
- **Stars:** 12
- **Forks:** 2
- **Issues:** 1
- **Status:** Experimental (WSL2 tested only)

## Available Tools

1. **list-zim-files** -- Displays all ZIM files within permitted directories
2. **search-zim-file** -- Queries content within ZIM archives
   - Mandatory parameters: zimFilePath, query
   - Optional: limit (default 10), offset (default 0)
3. **get-zim-entry** -- Retrieves full content for specific entries
   - Mandatory parameters: zimFilePath, entryPath
   - Optional: maxContentLength (default 10000)

## Architecture

Built as a Node.js application leveraging the Model Context Protocol framework, designed specifically for Windows Subsystem for Linux 2 environments. The system processes offline reference archives and exposes search/retrieval functionality through standardized MCP tool interfaces.

Requires pnpm for dependency management and compilation via TypeScript.
