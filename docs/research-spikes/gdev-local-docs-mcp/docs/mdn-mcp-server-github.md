<!-- Source: https://github.com/mdn/mcp -->
<!-- Retrieved: 2026-05-14 -->

# MDN MCP Server Analysis

## Project Overview
This is an experimental Model Context Protocol (MCP) server by Mozilla's MDN team that integrates MDN documentation, search functionality, and Browser Compatibility Data for use with LLM chatbots and coding agents.

## Key Metrics
- **Stars**: 33
- **Forks**: 7
- **License**: MPL-2.0
- **Languages**: JavaScript (94.6%), TypeScript (5.4%)
- **Commits**: 141 on main branch
- **Status**: Experimental (may be withdrawn anytime)

## Architecture & Tools

The server comprises:
- **Main files**: `index.js`, `server.js`, `transport.js`
- **Configuration**: `mcp.json` for MCP setup
- **Development tools**: ESLint, npm scripts, TypeScript support
- **Testing/Inspection**: MCP inspector included
- **Additional modules**: Logging, Sentry integration, Glean analytics, scripts folder

## Access Methods

The server operates in two modes:

1. **Remote**: `https://mcp.mdn.mozilla.net/` (HTTP transport)
2. **Local**: Runs on `http://localhost:3002/` after `npm start`

The implementation uses standard HTTP transport rather than direct scraping or local file access, suggesting it queries backend MDN services.

## Data & Privacy

Key considerations:
- Query data is stored during the experimental phase
- "This data will **not** be associated with any information designed to identify users"
- Users can opt out via `X-Moz-1st-Party-Data-Opt-Out: 1` header
- Complies with Mozilla's Acceptable Usage Policy

## Dependencies
Uses npm for package management with Node.js (v18+ based on `.nvmrc`).
