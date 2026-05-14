<!-- Source: https://github.com/upstash/context7 -->
<!-- Retrieved: 2026-05-14 -->

# Context7 Platform - Up-to-Date Code Documentation for LLMs

## Core Purpose
Context7 is a documentation retrieval system designed to inject up-to-date, version-specific library documentation directly into LLM prompts. It solves the problem of outdated training data by providing "up-to-date code examples straight from the source."

## Architecture & Operation

### Dual Integration Modes
1. **CLI + Skills**: Installs a skill guiding agents to fetch docs via `ctx7` commands without requiring MCP
2. **MCP Server**: Registers as a native Model Context Protocol server at `https://mcp.context7.com/mcp`

### Technology Stack
- Written in TypeScript (92.1%) with JavaScript components (7.7%)
- Operates as a pnpm monorepo with multiple packages
- Includes Dockerfile for containerized deployment

### Information Architecture
The system maintains a private backend consisting of:
- API backend
- Parsing engine
- Crawling engine

*(Repository hosts only the MCP server source code)*

## Documentation Sourcing & Storage

**Retrieval Method**: Context7 crawls and indexes library documentation, enabling retrieval based on:
- Library identifiers (e.g., `/supabase/supabase`, `/vercel/next.js`)
- Version-specific queries
- Natural language searches

## MCP Tools Exposed

1. **resolve-library-id**: Converts general library names into Context7-compatible IDs
2. **query-docs**: Retrieves documentation using library IDs, returns contextually relevant documentation

## Security Model

- OAuth authentication via `context7.com/dashboard`
- API key generation for rate-limit management
- Custom header authentication: `CONTEXT7_API_KEY`
- User reports flag suspicious/harmful content through project pages

## Key Differentiation from DevDocs

- **Active indexing** of current library versions
- **Automated version detection** from prompts
- **Query-aware ranking** of documentation relevance
- **Agent-native integration** via MCP
- **Cloud-hosted backend** (NOT local-first — queries go to context7.com)

## Limitations & Disclaimers

- "Community-contributed" projects may have accuracy variations
- "Cannot guarantee the accuracy, completeness, or security of all library documentation"
- Backend is proprietary/closed-source — only the MCP client is open
- Requires internet connectivity (cloud-first, not local-first)

## Community Metrics
- 55.3k GitHub stars
- 815 commits on master branch
- 74 releases (latest: ctx7@0.4.2)
