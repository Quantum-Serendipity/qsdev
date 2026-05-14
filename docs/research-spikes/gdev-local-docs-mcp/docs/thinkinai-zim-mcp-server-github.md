# ZIM MCP Server (ThinkInAI-Hackathon/zim-mcp-server)
> Source: https://github.com/ThinkInAI-Hackathon/zim-mcp-server
> Retrieved: 2026-05-14

## Overview
MCP server enabling AI models to access and search ZIM format knowledge bases offline without internet connectivity.

## Available Tools
1. **list_zim_files**: Lists all ZIM files in allowed directories
2. **search_zim_file**: Searches within ZIM file content (query, limit, offset)
3. **get_zim_entry**: Retrieves detailed content from specific entries

## Technical Details
- **Language**: Python (100%)
- **Package Manager**: uv
- **Stars**: 17
- **Forks**: 6
- **License**: MIT

## Installation
Clone -> install uv -> uv sync -> download ZIM files from Kiwix Library -> configure Claude Desktop.

## Distinction from cameronrye/openzim-mcp
This is a simpler, hackathon-origin project with only 3 tools vs openzim-mcp's 21. Both are Python-based and use python-libzim.
