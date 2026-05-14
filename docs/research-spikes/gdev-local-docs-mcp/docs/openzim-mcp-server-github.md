# OpenZIM MCP Server (cameronrye/openzim-mcp)
> Source: https://github.com/cameronrye/openzim-mcp
> Retrieved: 2026-05-14

## Project Description

OpenZIM MCP is described as "a modern, secure, and high-performance MCP (Model Context Protocol) server that enables AI models to access and search ZIM format knowledge bases offline." The ZIM format is an open file format developed by the openZIM project for offline storage and access to website content, particularly Wikipedia and other reference materials.

## Core Features

- **Dual Mode Support**: Simple mode (1 natural language tool) or Advanced mode (21 specialized tools)
- **Smart Navigation**: Browse by namespace (articles, metadata, media)
- **Context-Aware Discovery**: Article structure, relationships, and metadata extraction
- **Intelligent Search**: Advanced filtering, auto-complete, and relevance-ranked results
- **Performance Optimization**: Cached operations and pagination
- **HTTP Transport**: Long-running service capability with bearer-token auth, CORS support, and health endpoints
- **Batch Entry Retrieval**: Fetch up to 50 entries per call
- **Per-Entry MCP Resources**: Stream individual entries with native MIME types
- **Resource Subscriptions**: Notifications when archives change

## Simple Mode Tool

**`zim_query`** - Accepts natural language queries with compact output mode for small LLMs, featuring:
- Auto-fetching on strong title matches
- Non-Latin script support (Chinese, Cyrillic, Arabic, Devanagari, Hebrew)
- Conversational filler handling
- Case-insensitive title lookup with fuzzy fallback

## Advanced Mode Tools (21 Total)

**File & Metadata Operations:**
- `list_zim_files` - List available ZIM files with optional name filtering
- `get_zim_metadata` - Extract ZIM file metadata from M namespace
- `get_main_page` - Retrieve W namespace main page
- `get_server_health` - Server status and cache metrics
- `get_server_configuration` - Detailed configuration (sanitized)

**Search & Discovery:**
- `search_zim_file` - Full-text search within a ZIM file
- `search_with_filters` - Search with namespace and content-type filters
- `search_all` - Query every allowed ZIM file simultaneously
- `find_entry_by_title` - Resolve titles to entry paths (case-insensitive)
- `get_search_suggestions` - Auto-complete suggestions

**Content Retrieval:**
- `get_zim_entry` - Retrieve entry content with smart path resolution fallback
- `get_zim_entries` - Batch retrieve up to 50 entries
- `get_binary_entry` - Extract PDFs, images, videos with size caps
- `get_entry_summary` - Concise article summaries
- `get_table_of_contents` - Hierarchical heading tree
- `get_article_structure` - Extract headings and sections
- `extract_article_links` - Internal/external/media links with pagination
- `get_related_articles` - Outbound link-graph neighbors

**Navigation:**
- `list_namespaces` - Available namespaces and entry counts
- `browse_namespace` - Sample entries with pagination
- `walk_namespace` - Deterministic cursor-paginated iteration

## Architecture & Implementation

```
openzim_mcp/
├── Main entry points (main.py, __main__.py)
├── HTTP/Transport layer (http_app.py, server.py)
├── Configuration (config.py, defaults.py)
├── Security (security.py, exceptions.py)
├── Core processing (content_processor.py, cache.py, rate_limiter.py)
├── Simple mode (simple_tools.py, intent_parser.py)
├── ZIM operations package (zim/ with archive, content, namespace, search, structure mixins)
└── Tool registrations (tools/ with specialized modules)
```

## ZIM File Access

The server reads ZIM files using **libzim** (via python-libzim), the official library. Key characteristics:
- Supports both old-scheme and new-scheme (modern) ZIM archives
- Detects namespace scheme via `archive.has_new_namespace_scheme`
- New-scheme archives use C namespace exclusively; metadata through `archive.metadata_keys`
- Provides entry iteration, title-indexed suggestions, and full-text search indexes
- Handles path encoding differences (spaces vs underscores, URL encoding)

## Project Statistics

- **Stars**: 57
- **Forks**: 12
- **Issues**: 0 open
- **License**: MIT
- **Language**: Python (96.9%)
- **Latest Release**: v2.0.0a12 (May 14, 2026)
- **Test Coverage**: 80%+

## Installation

Available via:
- `uv tool install openzim-mcp` (isolated CLI, recommended)
- `pip install openzim-mcp`
- Development: `git clone` + `uv sync`

## Configuration

Managed through `OPENZIM_MCP_`-prefixed environment variables:
- `TOOL_MODE`: simple (default) or advanced
- `TRANSPORT`: stdio (default), http, or sse
- `HOST`/`PORT`: HTTP bind address (default 127.0.0.1:8000)
- `AUTH_TOKEN`: Required for non-loopback HTTP binding
- `CACHE__ENABLED`, `MAX_SIZE`, `TTL_SECONDS`
- `CONTENT__MAX_CONTENT_LENGTH`: Default 100,000 characters
