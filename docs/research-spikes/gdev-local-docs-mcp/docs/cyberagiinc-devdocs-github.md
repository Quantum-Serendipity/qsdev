<!-- Source: https://github.com/cyberagiinc/DevDocs -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs by CyberAGI - Project Analysis

## Core Function
DevDocs is a free, self-hosted web crawler and documentation management system designed to extract technical documentation from websites and make it queryable through AI-integrated interfaces. The project positions itself as an alternative to commercial services, emphasizing privacy and unlimited usage.

## Architecture Overview

**Frontend**: Next.js-based UI running on port 3001
**Backend**: Node.js API service on port 24125
**Crawling Engine**: Crawl4AI service (port 11235) for web scraping
**MCP Integration**: Model Context Protocol server for LLM connectivity

The system uses Docker containerization with separate Dockerfiles for frontend, backend, and MCP components.

## Crawling Capabilities

DevDocs implements intelligent web crawling with:
- Depth control (1-5 levels)
- Parallel processing for multiple pages
- Smart caching to avoid duplicate work
- Lazy loading support for dynamic content
- Selective URL extraction and mapping
- Output in Markdown, JSON, or LLM-ready formats

## Project Metrics
- **Stars**: 2.1k
- **Forks**: 188
- **Language Composition**: TypeScript (48.1%), Python (35.7%), Shell (11.9%)
- **License**: Apache 2.0

## Technology Stack
Built with Crawl4AI, Playwright for browser automation, and integrated with Anthropic's Claude through MCP protocol. The project explicitly partners with Anthropic and OpenAI.

## Key Differentiation from FireCrawl

The pricing comparison table indicates DevDocs offers unlimited free pages versus FireCrawl's paid tiers ($16-333/month), faster crawling (1000/min vs 20/min), and native MCP server integration—a feature described as absent from FireCrawl.

## Status Note
The README includes a warning that DevDocs "is not publicly maintained," with an "enhanced internal version at CyberAGI" and a promise of eventual public release.
