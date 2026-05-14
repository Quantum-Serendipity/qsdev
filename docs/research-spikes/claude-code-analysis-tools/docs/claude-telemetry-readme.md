<!-- Source: https://github.com/TechNickAI/claude_telemetry -->
<!-- Retrieved: 2026-03-26 -->

# Claude Telemetry: OpenTelemetry Wrapper for Claude Code CLI

## Overview

claude_telemetry is a thin observability layer wrapping Anthropic's Claude Code CLI. It captures tool calls, token usage, costs, and execution traces without modifying behavior, enabling visibility into headless agent deployments.

## Core Problem & Solution

**The Problem:** Running Claude Code in CI/CD, cron jobs, or production environments lacks console visibility. You cannot see which tools were called, track token usage, monitor costs, or debug failures without local reproduction.

**The Solution:** Adds OpenTelemetry instrumentation via SDK hooks, forwarding structured traces to any OTEL-compatible backend while maintaining identical Claude Code behavior.

## Key Features

- **Drop-in replacement**: Swap `claude` for `claudia` command with zero behavior changes
- **Pass-through architecture**: All Claude Code flags work unchanged
- **Multi-backend support**: Logfire, Sentry, Honeycomb, Datadog, or any OTEL endpoint
- **Complete visibility**: Captures prompts, model selection, token counts, tool execution, costs, timing
- **Hierarchical tracing**: Parent span for execution with child spans for each tool call
- **MCP server support**: Compatible with existing Model Context Protocol configurations

## Installation

```bash
pip install claude_telemetry                  # Basic
pip install "claude_telemetry[logfire]"       # With Logfire support
pip install claude_telemetry sentry-sdk       # With Sentry support
```

## Quick Start

```bash
# Before
claude code "Analyze my project"

# After - identical output, now observable
claudia "Analyze my project"
```

## What Gets Captured

**Per Execution:** Prompt and system instructions, model used, input/output/total token counts, estimated cost in USD, tool call count, execution duration, errors and failures.

**Per Tool Call:** Tool name, input parameters, output results, individual execution time, success/failure status.

## Span Hierarchy

```
claude.agent.run (parent span)
  user.prompt (event)
  tool.read (child span)
  tool.write (child span)
  agent.completed (event)
```

## Use Cases

- **Headless/Production:** Monitor agent execution in CI/CD and cron jobs
- **Cost Tracking:** Identify expensive workflows via per-execution token and cost metrics
- **Debugging:** Access full execution context without reproduction
- **Optimization:** Analyze tool usage patterns and model performance

## Project Status

- **License:** MIT
- **Python:** 3.10+
- **Dependencies:** OpenTelemetry SDK, Claude Code SDK
- **Performance:** Async telemetry, <10ms overhead per operation
