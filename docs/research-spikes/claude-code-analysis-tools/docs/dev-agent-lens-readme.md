<!-- Source: https://github.com/Teraflop-Inc/dev-agent-lens -->
<!-- Retrieved: 2026-03-26 -->

# Dev-Agent-Lens: Claude Code Observability via LiteLLM + Arize

## Project Summary

Dev-Agent-Lens is a transparent proxy layer enabling observability for Claude Code through LiteLLM and Arize integration. The system intercepts API calls and provides full visibility into Claude Code usage while maintaining compatibility with standard workflows.

## Core Architecture

Routes Claude CLI/SDK requests through a LiteLLM proxy (port 4000), which connects to OpenTelemetry exporters and optional PostgreSQL storage. Traces flow to either Arize AX (cloud) or Phoenix (local observability platform).

## Quick Start (2 minutes)

1. Copy environment template: `cp .env.example .env`
2. Select backend:
   - Arize AX: `docker compose --profile arize up -d`
   - Phoenix: `docker compose --profile phoenix up -d`
3. Run Claude Code through wrapper: `./claude-lens`

## Key Features

- **OAuth Passthrough**: Seamless authentication for Pro/Max plans without storing credentials
- **Model Routing**: Wildcard pattern support (`claude-*`, `anthropic/*`) allows Claude Code to select models automatically
- **Project Isolation**: Route traces to different projects via `CLAUDE_LENS_PROJECT` environment variable
- **Zero Configuration**: Works transparently without workflow changes
- **Dual Backend Support**: Cloud (Arize AX) or local (Phoenix) observability

## Configuration Files

| File | Purpose |
|------|---------|
| `claude-lens` | Wrapper script launching Claude Code with proxy settings |
| `litellm_config.yaml` | Model routing and callback configuration |
| `docker-compose.yml` | Service orchestration with health checks |
| `.env` | Local credentials (not version-controlled) |

## SDK Integration

Comprehensive examples provided in TypeScript and Python for basic usage, code review agents, custom tools, documentation generation, streaming responses, security analysis, incident response agents, and session management.
