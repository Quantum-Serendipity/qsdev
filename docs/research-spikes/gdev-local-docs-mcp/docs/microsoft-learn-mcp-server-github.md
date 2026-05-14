<!-- Source: https://github.com/MicrosoftDocs/mcp -->
<!-- Retrieved: 2026-05-14 -->

# Microsoft Learn MCP Server - Complete Details

## Purpose & Function
The Microsoft Learn MCP Server provides AI assistants and LLMs with "direct access to the latest official Microsoft documentation" without requiring API keys. It eliminates hallucinations by connecting models to trusted first-party sources rather than unreliable web searches.

## Core Architecture
The server operates as a remote HTTP endpoint accessible via the Model Context Protocol (MCP):
- **Endpoint:** `https://learn.microsoft.com/api/mcp`
- **Connection Type:** Streamable HTTP for compatible MCP clients
- **No Authentication Required:** Functions completely free with no sign-ups

## Available Tools
Three primary tools enable different documentation interactions:

1. **microsoft_docs_search** - "Performs semantic search against Microsoft official technical documentation"
2. **microsoft_docs_fetch** - Converts Microsoft documentation pages into markdown format
3. **microsoft_code_sample_search** - Locates official Microsoft/Azure code snippets with optional language filtering

## Documentation Coverage
The server indexes Microsoft Learn content including Azure services, .NET frameworks, C#/F#, ASP.NET Core, Entity Framework, NuGet, cloud architecture, and compliance guides. Experimental features include an OpenAI-compatible endpoint and token budget controls.

## Repository Metrics
- **Star Count:** 1.6k stars
- **Forks:** 190
- **Language Composition:** TypeScript (76.2%), PowerShell (19.3%), JavaScript (4.5%)
- **License:** Dual-licensed under CC-BY-4.0 and MIT (code)
- **Recent Activity:** 68 total commits on main branch

## Additional Components
- **CLI Tool:** `@microsoft/learn-cli` npm package for terminal access
- **Agent Skills:** Three portable instruction packages for enhanced agent reasoning
- **Platform Support:** VS Code, GitHub Copilot, Claude Desktop, Cursor, Visual Studio, Cline, and 10+ additional environments
