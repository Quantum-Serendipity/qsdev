<!-- Source: https://github.com/patoles/agent-flow -->
<!-- Retrieved: 2026-03-26 -->

# Agent Flow: Real-Time Claude Code Agent Visualization

## Project Description

Agent Flow is a VS Code extension that provides real-time visualization of Claude Code agent orchestration. Enables users to "watch your agents think, branch, and coordinate as they work."

## Core Purpose

Developed while creating CraftMyGame, an AI-driven game platform. Debugging agent behavior was challenging, so this visualization solution was built to make agent execution transparent and understandable.

## Key Capabilities

- **Understanding Agent Behavior**: Reveals how Claude decomposes problems, selects tools, and coordinates between subagents
- **Debugging**: Trace complete sequence of decisions and tool invocations when issues occur
- **Performance Analysis**: Identifies slow operations, unnecessary branching, and redundant processing patterns
- **Learning Tool**: Build better prompt-writing intuition by watching Claude's interpretation and execution

## Primary Features

- Interactive node graph visualization with real-time tool calls and branching flows
- Automatic detection of active Claude Code sessions in workspaces
- HTTP hook server for zero-latency event streaming
- Multi-session tracking with tabbed interface
- Pan, zoom, and interactive canvas for inspecting details
- Timeline and transcript panels with file attention heatmap
- JSONL log file replay capabilities

## Installation & Setup

Install from VS Code extensions, access via Command Palette: "Agent Flow: Open Agent Flow." Auto-configures Claude Code hooks on first use.

## Technical Requirements

- VS Code 1.85+
- Claude Code CLI with active sessions (for auto-detection)

## License

Apache 2.0
