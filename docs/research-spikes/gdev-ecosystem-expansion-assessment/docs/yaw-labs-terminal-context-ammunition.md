# Yaw Labs: Terminal Startup for People Who Treat Context Like Ammunition
- **Source**: https://www.siliconsnark.com/yaw-labs-built-a-terminal-startup-for-people-who-treat-context-like-ammunition/
- **Retrieved**: 2026-05-14

## Company Overview
Yaw Labs built three interconnected developer tools targeting people managing complex AI agent workflows with terminals, MCP servers, and context management needs.

## Product 1: Yaw Terminal

**Core Concept**: Cross-platform desktop terminal (Windows, macOS, Linux) functioning as a unified control room for development work.

**Key Features**:
- Split panes with built-in file editor
- SSH and database connections (PostgreSQL, MySQL, SQL Server, MongoDB, Redis)
- Remote session management for tmux and GNU Screen
- Encrypted local credential storage
- AI assistant capable of viewing terminal output
- Auto-detection of tools: Claude Code, Codex, Gemini CLI, Vibe CLI
- Paired split-pane workflow enabling side-by-side agent and command execution
- Free to use, no account required, no usage tracking

**Philosophy**: "The terminal is no longer a lonely rectangle" but rather "a shared workspace between you, your remote sessions, your databases, your coding agent, and whatever strange half-automated ritual your team has adopted this month."

**Yaw Mode Feature**: Per-session overlay of rules, skills, and agents layered onto Claude Code without altering `~/.claude/` config—described as enabling developers to avoid alt-tabbing between five adjacent tools.

## Product 2: mcp.hosting

**Core Concept**: Centralized orchestration layer addressing MCP server sprawl and configuration complexity.

**Key Features**:
- "One config for every MCP server" synced across clients
- Free tier supporting up to three servers
- Cloud-based central management of server lists

**Supporting Ecosystem**: Open-source implementations including:
- `mcph` orchestrator
- Compliance test suite
- MCP servers for AWS, Tailscale, SSH, npm, Caddy, LemonSqueezy, Electron

**Problem Addressed**: The article notes "every AI client now wants tools. Every tool now wants an MCP server. Every server wants config, auth, updates, transport decisions, and one more way to fail on a Wednesday."

## Product 3: typed.cloud

**Positioning**: "Drop-in fallback for Claude Code" with environment variable swaps (`ANTHROPIC_BASE_URL` plus typed API key and model selection).

**Pricing Structure**:
- Starter: $10/month
- Pro: $20/month (positioned as equivalent to Claude Pro)
- Max: $100/month
- Annual discounts available

**Claimed Advantages Over Claude**:
- Monthly billing instead of rolling usage windows
- Top-ups replacing current usage roulette
- Overage pricing at $1.67 per million input tokens, $8.33 per million output tokens (described as "roughly 44 to 67 percent cheaper than Claude's Sonnet and Opus overage")
- Features: prompt caching, image input, "at least equivalent capacity"

**Transparency**: Explicitly states it uses a different underlying model than Claude, positioning itself toward users prioritizing "coding throughput, monthly predictability, and long context tiers" over strict model identity.

## Strategic Thesis

The reviewer concludes Yaw Labs identified a genuine market gap: "people whose coding life now involves terminals, agents, MCP servers, and at least one recurring argument about pricing." The portfolio addresses operational pain points in agentic development workflows where "the leverage moves toward execution, orchestration, and workflow glue" once model layers commoditize.

The company's philosophy emphasizes respecting developer privacy ("bring your own keys, keep the traffic between you and the model provider") and acknowledging that modern development involves simultaneous context across multiple tools and sessions.
