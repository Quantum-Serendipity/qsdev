<!-- Source: https://github.com/arabold/docs-mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# Grounded Docs MCP Server: Comprehensive Overview

## Core Purpose
This open-source project provides an AI documentation indexing system that prevents language model hallucinations by grounding them in real, current documentation. It functions as a personal documentation expert accessible to AI coding assistants.

## Key Statistics
- **Stars:** 1,300+
- **Forks:** 153
- **Latest Release:** v2.2.1 (March 30, 2026)
- **Language:** TypeScript (99.8%)
- **License:** MIT
- **Repository:** arabold/docs-mcp-server

## What It Does

### Primary Functionality
The system fetches documentation from multiple sources—official websites, GitHub repositories, npm/PyPI packages, and local files—then creates a searchable index tailored to specific library versions in your project.

### Supported Document Formats
**Documents:** PDF, Word, Excel, PowerPoint, OpenDocument, RTF, EPUB, Jupyter Notebooks

**Archives:** ZIP, TAR files (contents extracted automatically)

**Markup:** Markdown, HTML, reStructuredText, AsciiDoc, Org Mode

**Code:** 90+ programming languages including Python, JavaScript, TypeScript, Go, Rust, Java

**Data:** JSON, YAML, TOML, CSV, XML, SQL, GraphQL

## Architecture & How It Works

### Three Usage Modes
1. **CLI** – Command-line interface for scraping and searching
2. **Web UI** – Browser interface at localhost:6280 for documentation management
3. **MCP Server** – Long-running endpoint for AI clients (Claude, Cline, VS Code)

### Key Operations
- **Scraping:** Fetches documentation from URLs with special support for hash-routed SPAs (single-page applications)
- **Indexing:** Creates searchable database of all documentation
- **Semantic Search:** Optional embedding models (OpenAI, Ollama, Gemini) enable vector-based semantic search

### Configuration
Supports environment variables and configuration files for customization. Hash-preservation mode automatically upgrades from fetch to Playwright for client-side rendering evaluation.

## Comparison to Competitors

The documentation positions this as "the open-source alternative to Context7, Nia, and Ref.Tools," emphasizing:
- Privacy (runs locally; code never leaves your network)
- Broad compatibility with MCP-compatible clients
- Multiple documentation source support
- Version-specific targeting

## Security & Privacy
- Operates entirely on local machines
- No external data transmission during indexing
- Optional OAuth2/OIDC authentication support
- Privacy-first telemetry approach

## Notable Features

**Caching:** Documented support for storing indexed documentation locally

**Agent Skills:** Includes pre-built skills in the `/skills` directory teaching AI assistants CLI usage patterns

**Docker Support:** Containerized deployment available via ghcr.io registry

**Search Output:** Supports multiple structured formats (JSON, YAML, custom formatting)

## Getting Started
Installation via `npx @arabold/docs-mcp-server@latest`, Docker, or Node.js 22+. Web UI provides graphical interface for adding documentation sources without CLI knowledge required.
