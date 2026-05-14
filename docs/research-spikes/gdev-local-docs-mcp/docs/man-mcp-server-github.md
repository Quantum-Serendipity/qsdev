<!-- Source: https://github.com/guyru/man-mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# Man-MCP-Server Analysis

## Overview
This is a Model Context Protocol (MCP) server providing AI assistants access to Linux system documentation. The project enables searching and retrieving man pages from the local machine.

## Core Functionality
The server offers three main tools:
- **search_man_pages**: Find documentation by keyword using `apropos` command
- **get_man_page**: Retrieve complete man page content by name and optional section
- **list_man_sections**: Browse available man page sections (1-9) with descriptions

It also exposes man pages as resources via `man://` URIs for integration with compatible clients.

## Data Access
**Local only** — The server accesses man pages from the local Linux system exclusively. It relies on system commands (`man`, `apropos`) that are pre-installed on standard Linux distributions. No web fetching occurs.

## Technical Details
- **Language**: Python (70.5%) and Makefile (29.5%)
- **Requirements**: Python 3.10+, Linux OS with man page system
- **Features**: Asynchronous operations with timeout protection, ANSI code removal for AI consumption, error handling with fallback methods

## Project Metrics
- **Stars**: 13
- **Forks**: 5
- **Open Issues**: 0
- **License**: MIT
- **Installation**: Available as MCPB bundle for Claude Desktop or configurable for VS Code via `.vscode/mcp.json`

## Security Properties
The server includes "timeout protection" for subprocess calls and validates that retrieved content is not empty, minimizing risks from malformed man page data or system hangs.
