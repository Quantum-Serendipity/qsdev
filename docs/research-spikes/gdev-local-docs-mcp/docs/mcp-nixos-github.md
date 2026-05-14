<!-- Source: https://github.com/utensils/mcp-nixos -->
<!-- Retrieved: 2026-05-14 -->

# MCP-NixOS: Architecture & Details

## Core Information
- **Stars:** 638
- **License:** MIT
- **Primary Language:** Python (89.4%)
- **Latest Release:** v2.4.3 (April 25, 2026)
- **Repository:** github.com/utensils/mcp-nixos

## Architecture & Tools

The project provides a unified Model Context Protocol server with two consolidated tools:

1. **`nix` tool** - Handles searches, detailed lookups, statistics, and cache checks across multiple sources
2. **`nix_versions` tool** - Retrieves historical package versions with commit hashes and metadata

Total token usage: approximately 1,030 tokens -- described as minimalist compared to competing MCP servers.

## Data Sources & Access Methods

The system queries remote APIs rather than requiring local Nix installation:

- **search.nixos.org** - Official NixOS packages and options
- **FlakeHub.com** - Community flake registry
- **noogle.dev** - Nix function signatures
- **wiki.nixos.org** - Community documentation
- **nix.dev** - Official Nix tutorials
- **NixHub.io** - Package metadata and version history
- **cache.nixos.org** - Binary cache status checks

Local capability: Can explore flake inputs from the Nix store (requires Nix installation).

## Coverage Areas

- 130,000+ NixOS packages
- 23,000+ NixOS configuration options
- 5,000+ Home Manager options
- 1,000+ nix-darwin macOS settings
- 5,000+ Nixvim Neovim configuration options
- 600+ FlakeHub registry entries
- 2,000+ Nix function signatures

## Installation Methods

- **uvx** (recommended, no dependencies)
- Nix flake
- Docker container
- HTTP/FastMCP remote server
- Pi coding agent integration
- PyPI package installation

## Platform Support

"No Nix/NixOS Required! Works on any system - Windows, macOS, Linux."

## Security Properties

- No mention of authentication requirements for API access
- Docker isolation available
- HTTP transport option with stateless mode support
