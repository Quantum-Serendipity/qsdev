<!-- Source: https://docs.mantra.gonewx.com/features/mcp-hub -->
<!-- Retrieved: 2026-03-26 -->

# Mantra MCP Hub — Documentation

## What is MCP Hub?

MCP Hub is "a complete implementation of the MCP (Model Context Protocol) open standard initiated by Anthropic." It functions as a dual-purpose system: aggregating multiple MCP services into a single Streamable HTTP server endpoint, acting as both "traffic relay" and "management panel."

## Core Functions

1. **Unified Entry Point**: Aggregates multiple MCP services (supporting stdio, SSE and all standard protocols).
2. **Transparent Takeover**: Can take over MCP configurations from Claude Code, Cursor and other tools, allowing centralized permission management.
3. **Security Management**: Graphical interface for managing service association states and tool enable/disable status.

## Feature Set

- **Service Management**: Adding, editing and switching MCP services, supporting all standard MCP protocols.
- **Environment Variables**: Configuring global or project-specific environment variables for MCP services, with sensitive information hiding.
- **OAuth Credentials**: Securely storing authentication tokens required by remote MCP services (such as Google Drive).
- **Status Monitoring**: Real-time viewing of Hub runtime status, connected clients, and active service counts.

## Configuration Takeover Mechanism

Imports MCP configurations from external AI tools without manual reconfiguration. Supports Claude Code, Cursor, Gemini CLI, and Codex — both user-level and project-level configurations.

Intelligent merge engine uses a three-tier classification strategy: new services are directly imported, existing services with configuration changes trigger update prompts, and conflicting services receive differential comparison with manual decision support.

## Cross-Tool MCP Sharing

Enables unified MCP access across multiple tools through configuration takeover. Project-level management includes sidebar panels for managing MCP service associations from project right-click menus and scope configuration imports.

## Technical Implementation

### MCP Roots Protocol
AI tools communicate their current working directory via `roots/list` requests. MCP Hub uses longest prefix matching algorithms to automatically route to corresponding MCP services based on project context.

### Tool Policy (Permission Management)
Project-level tool granularity permission control. Different tool access permissions per project — e.g., enabling file modification in Project A while only reading files in Project B. Individual tools like `read_file` or `shell_execute` can be enabled, disabled, or intercepted.

### Built-in Inspector
Real-time JSON-RPC communication logging between AI models and MCP services, manual tool invocation for verification, error diagnosis for timeout/permission/format issues.

## Data Protection & Reliability
Atomic operations for takeover and recovery, with automatic rollback on failure. Recent 5 backup versions retained with automatic integrity validation.

## System Integration
Background service via system tray. Real-time status notifications through tray icon changes.

## Protocol Compliance
Adheres to MCP Streamable HTTP specifications (2025-03-26 version), supporting unified `/mcp` endpoints, session management via `MCP-Session-Id` headers, origin verification, and backward compatibility with `/sse` and `/message` endpoints.
