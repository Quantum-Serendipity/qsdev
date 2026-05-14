<!-- Source: https://github.com/TechNickAI/claude_telemetry -->
<!-- Retrieved: 2026-03-26 -->

# claude_telemetry - OpenTelemetry Wrapper for Claude Code CLI

Thin observability layer that wraps Claude Code CLI with full telemetry capabilities. Captures tool calls, token usage, costs, and execution traces without modifying agent behavior.

## The Problem
Running Claude Code in headless environments (CI/CD, cron jobs, production servers) eliminates visibility into agent execution.

## The Solution
Drop-in replacement that swaps 'claude' command for 'claudia'. All commands pass through unchanged while sending structured traces to observability platforms.

## Core Features

- **Pass-through architecture:** All Claude Code flags work unchanged
- **Multi-backend support:** Logfire, Sentry, Honeycomb, Grafana, or any OTEL backend
- **Zero behavior changes:** Identical output to standard Claude Code
- **Structured trace hierarchy:** Parent spans for agent runs with child spans for tool calls
- **Cost tracking:** Automatic token usage and USD cost calculation

## Installation

```bash
pip install claude_telemetry
pip install "claude_telemetry[logfire]"  # With Logfire support
```

## Usage

```bash
claudia "Analyze my project and suggest improvements"
claudia --model opus "Refactor this module"
```

## What Gets Captured

**Per execution:** Prompt, model, token counts, execution time, USD cost, tool call count, errors
**Per tool call:** Tool name, input/output, execution time, success/failure

## Backend-Specific Features

- **Logfire:** Enhanced LLM-specific UI with token visualization
- **Sentry:** AI Performance Dashboard with gen_ai.* attributes

## Architecture
Pass-through design using SDK's built-in hook system for observability. OpenTelemetry chosen for vendor neutrality.
