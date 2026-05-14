# Kiwix Wiki MCP Server (jeffreyrampineda/kiwix-wiki-mcp-server)
> Source: https://github.com/jeffreyrampineda/kiwix-wiki-mcp-server
> Retrieved: 2026-05-14

## Overview
A Model Context Protocol (MCP) server enabling offline Wikipedia and content access through Kiwix integration.

## Key Details
- **Language:** TypeScript (100%)
- **License:** ISC License
- **Stars:** 14
- **Forks:** 2
- **Contributors:** 1 (jeffreyrampineda)

## Architecture & Access Method
The server connects to a locally-running Kiwix server instance via HTTP at `http://localhost:8080` (configurable). It does NOT directly read ZIM files but instead communicates with the running Kiwix service.

## Available Tools

1. **search_wiki** - Queries offline wiki content with customizable result limits (default: 10, max: 50)
2. **get_article** - Retrieves complete article content via URL/path
3. **list_libraries** - Displays available offline content repositories

## Dependencies & Setup
- Requires Node.js and npm
- Kiwix server installation (Ubuntu/Debian via apt, macOS via Homebrew, or manual download)
- ZIM format content files from kiwix.org library
- Build via TypeScript compilation with `npm run build`

## Operational Requirements
Users must: install Kiwix tools, obtain ZIM files from library.kiwix.org, launch the server on designated port, then run the MCP server to enable client access.

## Key Distinction
This server is a thin wrapper around kiwix-serve, NOT a direct ZIM reader. It requires a running kiwix-serve instance.
