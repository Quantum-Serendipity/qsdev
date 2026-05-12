<!-- Source: https://github.com/cachix/devenv -->
<!-- Retrieved: 2026-05-12 -->

# Devenv GitHub Repository Overview

## Project Description

Devenv is described as "Fast, Declarative, Reproducible, and Composable Developer Environments using Nix." It's an Apache 2.0 licensed open-source project maintained by Cachix with 6.8k GitHub stars and 488 forks.

## Core Features

### Developer Experience
- **Terminal UI** with live build progress, task hierarchy, and error details
- **Native shell reloading** that rebuilds in the background while maintaining shell interactivity
- **Instant environments** with incremental Nix evaluation caching achieving sub-100ms activation when no changes occur
- **LSP support** for devenv.nix with autocomplete, hover documentation, and go-to-definition via bundled nixd
- **Ad hoc environments** available from CLI without configuration files
- **Out-of-tree devenv support** to use configurations from other repositories

### Languages, Packages & Services
- Support for "50+ languages" with built-in tooling including compilers, LSP servers, formatters, and linters
- Access to "100,000+ packages" from Nixpkgs across Linux, macOS, x64, and ARM64 architectures
- "40+ services" available including PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, and Caddy

### Processes & Task Management
- Native process manager written in Rust with dependency ordering and restart policies
- Readiness probes supporting exec, HTTP, and systemd notify mechanisms
- Automatic port allocation preventing collisions in parallel environments
- Task execution with DAG-based ordering, caching, and parallel runs
- Custom scripts with access to environment packages

### Packaging & Deployment
- OCI container building without Docker
- Language-specific output packaging using tools like crate2nix and uv2nix
- Polyrepo support for cross-repository configuration references

### Configuration & Composition
- Profile support for environment variants
- Composable via imports for sharing across projects
- Input pinning and dependency overriding capabilities

### Security & Integration
- **SecretSpec** for declarative secrets management across multiple providers
- Git hooks integration via git-hooks.nix
- Built-in testing capabilities with automatic process management
- Direnv integration for automatic activation
- MCP server for AI assistant integration
- AI-powered environment generation

## Quick Start

The project provides `devenv init` to generate a `devenv.nix` configuration file. Users then run `devenv shell` to activate the environment. The configuration file supports environment variables, package specifications, language configurations, process definitions, and service setup.

## Available Commands

The CLI provides 25+ commands including:
- `init` and `generate` for setup
- `shell` for environment activation
- `update`, `search`, and `info` for management
- `up`, `processes`, and `tasks` for execution
- `container` for OCI building
- `test`, `repl`, `build`, and `eval` for development
- `lsp` and `mcp` for tooling integration

## Technical Implementation

The project is written primarily in Rust (53.9%) and Nix (44.3%), with minor components in Shell, Nushell, Dockerfile, and Python. The codebase is organized into multiple specialized modules including shell integration, process management, task execution, TUI components, and Nix evaluation caching.

The repository contains 6,544 commits and 49 releases, with the latest version being v2.1 released May 5, 2026.
