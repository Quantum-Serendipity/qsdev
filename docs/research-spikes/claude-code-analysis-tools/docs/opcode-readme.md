<!-- Source: https://github.com/winfunc/opcode -->
<!-- Retrieved: 2026-03-26 -->

# Opcode: Claude Code GUI & Toolkit

## Overview

Opcode is a desktop application built with Tauri 2 that serves as a visual command center for Claude Code. It bridges the gap between CLI tools and a graphical interface for managing AI-assisted development workflows. Formerly known as Claudia.

## Core Features

**Project & Session Management**
- Visual browser for Claude Code projects stored in `~/.claude/projects/`
- Session history with resumable past coding interactions
- Integrated search functionality across projects and sessions
- Metadata display including timestamps and first messages

**CC Agents**
- Custom AI agents with configurable system prompts
- Agent library for task-specific specialization
- Background execution in separate processes
- Detailed execution history with performance metrics

**Usage Analytics Dashboard**
- Real-time Claude API cost tracking
- Token consumption breakdown by model, project, and time period
- Visual trend charts and usage patterns
- Data export capabilities for accounting purposes

**MCP Server Management**
- Centralized Model Context Protocol server registry
- UI-based server configuration and testing
- Import functionality from Claude Desktop configurations
- Connection verification before deployment

**Timeline & Checkpoints**
- Session versioning with checkpoint creation
- Visual branching timeline navigation
- One-click restoration to previous states
- Session forking from existing checkpoints
- Diff viewer showing changes between checkpoints

**CLAUDE.md Management**
- Built-in markdown editor for CLAUDE.md files
- Real-time markdown preview rendering
- Project-wide file discovery
- Syntax highlighting support

## Technology Stack

- **Frontend:** React 18 + TypeScript + Vite 6
- **Backend:** Rust with Tauri 2
- **UI:** Tailwind CSS v4 + shadcn/ui
- **Database:** SQLite via rusqlite
- **Package Manager:** Bun

## Security & Privacy

- Process isolation for agent execution
- Per-agent permission controls for file and network access
- All data remains local; no telemetry or tracking

## Licensing

AGPL. Independent project not affiliated with Anthropic.
