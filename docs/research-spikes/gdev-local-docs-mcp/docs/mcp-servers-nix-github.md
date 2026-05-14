<!-- Source: https://github.com/natsukium/mcp-servers-nix -->
<!-- Retrieved: 2026-05-14 -->

# mcp-servers-nix: Nix-Based MCP Server Configuration Framework

## Overview

mcp-servers-nix is a Nix repository that provides "a Nix-based configuration framework for Model Control Protocol (MCP) servers with ready-to-use packages." It enables developers to declaratively define and deploy MCP servers across multiple platforms using the Nix package manager.

## Core Features

- Declarative, composable server configurations
- Reproducible builds via Nix's deterministic approach
- Pre-built modules for popular MCP server types
- Credential handling via `envFile` and `passwordCommand` options
- Multi-framework support (Flakes, flake-parts, devenv, Home Manager)

## Available MCP Servers

28+ pre-configured server modules, including:
- **Data/Integration**: GitHub, Notion, Slack, Home Assistant, Grafana
- **Developer Tools**: Filesystem, Git, Playwright, Terraform
- **Utilities**: Fetch, Memory, Time, Sequential Thinking
- **Language/Text**: TextLint, DeepL, Tavily search

## Integration Examples

### devenv Integration
Uses `claude.code.mcpServers` for development environment configuration.

### Home Manager Integration
Uses `programs.mcp.servers` configuration option.

## Configuration Flavors

Adapts output formats for different clients:
- Claude Desktop (`mcpServers`)
- VS Code (`mcp.servers`)
- Zed editor (`context_servers`)
- Codex CLI (`.mcp.toml`)

## Quick Usage Pattern

Uses `mkConfig` function to output properly formatted server definitions with command paths and arguments for target client.

- Licensed under Apache 2.0
- 252 GitHub stars, actively maintained
