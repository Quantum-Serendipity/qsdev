<!-- Source: https://docs.mantra.gonewx.com/features/context-causality -->
<!-- Retrieved: 2026-03-26 -->

# Mantra Context Causality Feature — Documentation

## Overview

Introduced in v0.11.0, Context Causality uses AI to automatically analyze and establish logical connections between reference files and code changes, revealing why AI generated specific code.

## Problem It Solves

In complex AI conversations, AI reads multiple files (documentation, base classes, utility functions) as references before generating code. Traditional tools only show which files were accessed but cannot identify which specific content influenced the final code generation.

## Core Capabilities

### 1. Automatic Reference File Extraction
The parser automatically extracts file paths from tool calls like `read_file` and displays a complete inventory of all files involved in each message at the message header.

### 2. Context Promotion (Reference Blocks)
Tool execution results — such as file contents — are elevated from plain text to semantic "Reference Blocks," transforming them into meaningful content units rather than static data.

### 3. AI Causality Mapping
The system analyzes causal relationships between Reference Blocks and CodeDiff sections through background AI processing:

- **High confidence (> 0.8)**: Treated as direct causes with strong UI associations
- **Low confidence (< 0.3)**: Classified as background knowledge, placed in the sidebar

## Interaction Features

- **Hover Preview**: In the narrative panel, hovering over code changes automatically highlights corresponding document segments through visual connections
- **Context Panel**: Clicking the "context" icon beside messages reveals complete data dependency relationships

## Related Features

Links to Replay Mode, Time Travel tracking, and Message Filtering.
