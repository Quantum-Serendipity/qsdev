<!-- Source: https://github.com/cameronrye/openzim-mcp -->
<!-- Retrieved: 2026-05-14 -->

# OpenZIM MCP: Comprehensive Technical Overview

## Project Fundamentals

**OpenZIM MCP** is a Model Context Protocol server enabling AI models to access offline ZIM (Zeno IMproved) knowledge bases. The repository shows **57 stars** and **12 forks**, written primarily in Python (96.9%). Licensed under MIT, it's actively maintained with 34 releases and the latest being v2.0.0a12 (May 2026).

## Architecture & Design

The system employs a modular architecture with dependency injection. Core components include:

- **HTTP/SSE Transport Layer**: Supports both streamable HTTP and legacy SSE with bearer-token authentication, CORS configuration, and health endpoints
- **ZIM Operations Package** (`zim/`): Four specialized mixins handling archive access, content retrieval, namespace browsing, and search operations
- **Content Processing Pipeline**: HTML-to-text conversion, heading resolution with fallback logic, and link extraction filtering non-navigable schemes
- **Intelligent Caching**: LRU cache with TTL for frequently accessed entries, plus path-mapping caches for smart retrieval

The server can run in **Simple Mode** (one natural-language tool) or **Advanced Mode** (21 specialized tools). Simple mode keeps prefill token costs minimal by exposing only `zim_query`.

## The 21 Advanced Mode Tools

**File Operations (1):**
- `list_zim_files` with case-insensitive filtering

**Content Retrieval (2):**
- `get_zim_entry` with smart fallback to search-based retrieval
- `get_zim_entries` for batch retrieval of up to 50 entries

**Search Tools (4):**
- `search_zim_file` with pagination cursors
- `search_with_filters` supporting namespace and MIME-type constraints
- `search_all` querying every archive simultaneously
- `find_entry_by_title` with typo-tolerance fallback

**Navigation (3):**
- `browse_namespace` for random sampling
- `walk_namespace` for deterministic cursor-paginated iteration
- `get_search_suggestions` with 2-character minimum

**Structure & Links (5):**
- `get_article_structure` extracting headings and metadata
- `extract_article_links` with per-category pagination
- `get_table_of_contents` providing hierarchical heading trees
- `get_entry_summary` capping output at configurable word limits
- `get_related_articles` via outbound link-graph neighbors

**Metadata (2):**
- `get_zim_metadata` from M-namespace entries
- `list_namespaces` with entry counts per namespace

**Binary Content (1):**
- `get_binary_entry` for PDFs, images, and embedded media with explicit size caps

**Server Management (2):**
- `get_server_health` with cache metrics and accessibility checks
- `get_server_configuration` with redacted diagnostics

**Plus: 3 MCP Prompts** (`/research`, `/summarize`, `/explore`) and **MCP Resources** exposing ZIM files and individual entries via `zim://` URIs.

## ZIM Archive Support & Offline Capabilities

The tool supports **ZIM files from Kiwix**, primarily Wikipedia content. Key offline features:

- Works entirely without network connectivity once archives are downloaded
- Supports both old-scheme and new-scheme ZIM formats (modern Wikipedia builds use new-scheme)
- Handles namespace schemes: C (content), M (metadata), W (main page), X (media), I (images)
- Full-text search via libzim's indexed search (when present)
- Embedded binary content retrieval (PDFs, images, videos)

## Search Implementation

Search operates through multiple pathways:

1. **Full-text search** when libzim index exists
2. **Suggestion-based fallback** using libzim's title index when direct search fails
3. **Typo-tolerance** with single-edit-distance matching for title lookups
4. **Cursor-based pagination** allowing resumption without query restatement
5. **Compact mode** (v1.2.0+) reducing response size 3-6x by truncating snippets to 250 characters and capping final output at 6,000 characters

## Security Architecture

Protection mechanisms include:

- **Path traversal prevention** with secure validation
- **Input sanitization** for all user-supplied data
- **Redacted diagnostics** preventing PID and filesystem path disclosure
- **Timing-safe token comparison** for bearer-token authentication
- **ReDoS protection** on regex operations (markdown link stripping uses threading-based timeouts)
- **Non-loopback binding restrictions** requiring bearer tokens

## Performance & Resource Management

- **Intelligent caching** with configurable TTL (default 3,600 seconds)
- **Batch retrieval** reducing HTTP round-trip costs
- **Resource pooling** for efficient archive handle management
- **Lazy loading** of components
- **Per-entry size caps**: text bodies at 256 KB UTF-8; binary bodies are refused if oversized

## Deployment Options

Supports three transport modes:
1. **stdio** (default): Local MCP client connections
2. **HTTP**: Long-running service with auth, CORS, health endpoints
3. **SSE**: Legacy server-sent events

Multi-architecture Docker image available (`linux/amd64`, `linux/arm64`) running as non-root.
