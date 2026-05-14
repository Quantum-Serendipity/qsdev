<!-- Source: https://github.com/jiegec/devdocs-mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# jiegec/devdocs-mcp-server - DevDocs MCP Server

## Architecture & Purpose
This project implements a Model Context Protocol (MCP) server that bridges AI assistants with DevDocs documentation. It operates in two modes: stdio (default for MCP clients) and HTTP (for direct web access).

## Core Functionality

**Three Primary Tools:**
- `search_devdocs` - Locates documentation entries using fuzzy matching
- `read_devdocs` - Retrieves specific documentation files
- `list_doc_sets` - Enumerates available documentation collections

## How It Works

The system extracts documentation from the official DevDocs Docker image, storing it locally. When queried, it performs intelligent pattern matching across documentation files and converts HTML output to Markdown format automatically.

## Installation & Dependencies

Built with Python using Poetry for dependency management. Setup requires:
```
poetry install
python -m devdocs_mcp_server.extract_docs
```

The extraction step downloads and processes the latest DevDocs data into a `docs` directory.

## Command-Line Interface

Users can interact directly via:
- `devdocs search "query"` - Find documentation
- `devdocs read python/list.html` - Access specific files
- `devdocs list-sets` - View available documentation

Optional `--doc-set` parameter restricts searches to specific documentation collections.

## Technical Stack

Primarily Python (99.7% of codebase) with minimal Shell scripting. Uses Ruff for code quality assurance and pytest for testing.

## Key Differentiator

Extracts docs from the DevDocs Docker image directly rather than running a live DevDocs instance. This means it works with static files, no Ruby/Sinatra server needed at runtime.

## Transport

Supports both stdio and HTTP transport modes.

**License**: Not specified
