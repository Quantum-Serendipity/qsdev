<!-- Source: https://devenv.sh/mcp/ -->
<!-- Retrieved: 2026-05-14 -->

# devenv MCP Server Integration Documentation

## Overview
The devenv MCP server exposes devenv functionality to AI assistants using the Model Context Protocol standard. It can be launched with the command `devenv mcp`.

## Usage Modes

**Stdio Mode (Default)**
The server communicates via stdin/stdout, designed for when AI tools spawn devenv as a subprocess.

**HTTP Mode**
```
devenv mcp --http [port]
```
Launches the MCP server as an HTTP service with a default port of 8080.

## Available Tools

The MCP server provides two primary tools:

1. **search_packages** — locates packages within the nixpkgs input
2. **search_options** — searches for devenv configuration options

## Claude Code Integration

The documentation references a dedicated section on Claude Code integration with instructions for "configuring Claude Code to use the devenv MCP server automatically."
