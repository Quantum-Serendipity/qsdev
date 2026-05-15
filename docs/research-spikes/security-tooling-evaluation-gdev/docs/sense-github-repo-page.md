# Sense GitHub Repository Page
- **Source**: https://github.com/luuuc/sense
- **Retrieved**: 2026-05-15

---

# Sense: MCP Server for AI Coding Agents

## Repository Overview

**Repository:** luuuc/sense
**Description:** MCP server providing Claude Code, Cursor, and Codex CLI with structural codebase understanding through symbol graphs, blast radius analysis, semantic search, and convention detection.

## Key Metrics

- **Stars:** 4
- **Forks:** 1
- **Language Composition:** Go (88.1%), Python (8.8%), Shell (3.1%)
- **License:** O'Saasy (MIT-style with SaaS-competition rights)
- **Latest Release:** v0.84.3 (May 15, 2026)
- **Total Commits:** 688 on main branch

## Core Functionality

Sense operates as a local MCP server requiring zero external dependencies. The tool provides four primary capabilities:

1. **sense_graph** - Symbol relationships, callers, and inheritance tracking
2. **sense_search** - Hybrid semantic and keyword search with text fallback
3. **sense_blast** - Impact analysis with risk scoring
4. **sense_conventions** - Project pattern detection from source code

## Performance Improvements

According to benchmark data across seven real-world codebases, Sense achieves:

- "Tool calls per task: 19→10 (-47%)"
- "Tokens per task: 228K→156K (-32%)"
- "Cost per task: $0.42→$0.31 (-26%)"
- "Score per 100K tokens: 0.19→0.30 (+64%)"

## Language Support

**Full Tier (13 languages):** Ruby (Rails), TypeScript/JavaScript (React), Python (Django/FastAPI), Go, Rust, ERB, Java, Kotlin, C#, C++, C, PHP, Scala

## Technical Specifications

- **Binary Size:** ~60 MB
- **Index Size:** 100-200 MB per project
- **Query Performance:** 0.2ms/3ms (p50/p95)
- **Scan Speed:** 4.9s full, 2.3s incremental (on 382-file codebase)

## Installation & Setup

Installation: `curl -fsSL https://luuuc.github.io/sense/install.sh | sh`

Two-command setup:
1. `sense scan` - Indexes codebase with tree-sitter parsing
2. `sense setup` - Auto-configures Claude Code, Cursor, or Codex CLI

## Directory Structure

Core directories include: `.github/`, `cmd/sense/`, `internal/`, `mcpb/`, `docs/`, `bench/`, `testdata/`, and `smithery/`.

## Platform Support

Supported: Linux (amd64/arm64), macOS (Apple Silicon/Intel)
Windows: WSL2 with Linux binary (native builds pending)
