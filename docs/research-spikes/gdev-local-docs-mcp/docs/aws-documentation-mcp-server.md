<!-- Source: https://awslabs.github.io/mcp/servers/aws-documentation-mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# AWS Documentation MCP Server Analysis

## Core Purpose
The server functions as a Model Context Protocol bridge that enables AI assistants to access and interact with AWS documentation. It "provides tools to access AWS documentation, search for content, and get recommendations."

## Key Capabilities
The system offers five primary tools:

1. **read_documentation**: Converts AWS docs pages into markdown format
2. **search_documentation**: Uses "the official AWS Documentation Search API" (global only)
3. **read_sections**: Fetches specific documentation sections as markdown
4. **recommend**: Delivers content suggestions for documentation pages
5. **get_available_services**: Lists AWS services in China regions (China-specific)

## Access Architecture
The server uses HTTP requests to fetch documentation. It supports custom User-Agent configuration "for corporate environments with proxy servers or firewalls that block certain User-Agent strings," indicating direct web-based access rather than local caching.

## Deployment & Runtime
- Built on Python 3.10+
- Runs via `uvx` command or Docker containerization
- Operates as a stdio-based MCP process
- Environment variables control partition selection (`aws` vs `aws-cn`) and logging

## Security & Connectivity
The architecture is **online-dependent** -- it retrieves documentation in real-time without described caching mechanisms. Security features include User-Agent customization for network environments and partition isolation for China region documentation.

## Limitations
No offline capability, caching layer, or local documentation storage is mentioned in the documentation.
