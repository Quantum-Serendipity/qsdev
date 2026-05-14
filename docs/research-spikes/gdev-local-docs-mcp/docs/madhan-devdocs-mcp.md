<!-- Source: https://github.com/madhan-g-p/DevDocs-MCP -->
<!-- Retrieved: 2026-05-14 -->

# madhan-g-p/DevDocs-MCP - Documentation Authority for AI Agents

## Overview

DevDocs-MCP is a Model Context Protocol (MCP) server built in NestJS/TypeScript that delivers "Eliminate AI hallucinations with local, version-aware, and authoritative documentation." It functions as a local Documentation Intelligence Layer, preventing training data drift and network latency by caching versioned API documentation offline.

## Core Problem & Solution

**The Problem**: AI agents struggle with three issues:
- Hallucination (incorrect API signatures)
- Latency (fetching live docs per query)
- Version Drift (confusing React 18 with React 19)

**The Solution**: DevDocs-MCP acts as middleware between IDE agents and documentation sources, providing just-in-time documentation from local storage filtered by project dependency versions.

## Architecture & Key Features

**Data Storage Strategy**:
- SQLite metadata database (`sql.js` for zero native dependencies)
- Cached JSON documentation files on local disk
- Unified `/app/data` mount containing both `mcp.db` and `docs/` folders

**Version-Pinning Capabilities**:
- Automatically maps documentation to specific library versions from `package.json`
- Lazy-ingestion engine that caches DevDocs offline
- Project-aware context manager ensuring version consistency

**Technical Advantages**:
- Node-only architecture (no Python/C++ build dependencies)
- Offline-first (no internet required post-ingestion)
- Ranked fuzzy search for instant relevant entry discovery
- Structured, LLM-optimized outputs

## Exposed Tools & Integration

The server provides:
- **Ingest tool**: Download documentation for specific stack versions
- **Search functionality**: Ranked fuzzy search across documentation entries
- **Explain tool**: Content retrieval with version awareness

## Installation & Deployment

**Local Setup**:
```
pnpm install
cp .env.example .env
pnpm build && pnpm start:prod
```

**Docker Support**:
- Docker Compose with three storage modes (named volumes, host paths)
- Dockerfile available for containerized deployment
- Public image: `madhandock1/devdocs-mcp:latest`

Supports RooCode, Cline, Claude Desktop, and remote HTTP/SSE clients.

## Unique Distinguishing Features

1. **Zero Web Scraping**: Uses official DevDocs structured datasets, not web scraping
2. **Version-Pinning**: Reads project package.json to serve version-matched docs
3. **Portable Configuration**: Environment variables allow relative paths
4. **Unified Data Model**: Single `/app/data` folder simplifies backup/migration
5. **NestJS/TypeScript stack**: No Python/C++ dependencies

## Requirements

- Node.js 18+
- Uses `pnpm`
- No native build dependencies required
