# Agentic Session Observability Research

## Research Question

Should gdev integrate or configure OpenTelemetry-based observability for Claude Code sessions? Is this practically useful for a consulting firm or just noise?

## Current Landscape

### Native Claude Code OTEL Support (April 2026)

Claude Code has **first-party OpenTelemetry support** as of April 2, 2026. This is not a third-party hack -- it is built into the CLI and Agent SDK. The three OTEL pillars are all supported:

| Signal | Contents | Enable With |
|--------|----------|-------------|
| Metrics | Token counters, cost, sessions, LoC, tool decisions | `OTEL_METRICS_EXPORTER` |
| Log events | Prompts, API requests/errors, tool results | `OTEL_LOGS_EXPORTER` |
| Traces (beta) | Spans for interactions, LLM requests, tool calls, hooks | `OTEL_TRACES_EXPORTER` + `CLAUDE_CODE_ENHANCED_TELEMETRY_BETA=1` |

Configuration is entirely via standard OTEL environment variables (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`, etc.). No code changes required -- just set env vars and telemetry flows to any OTLP-compatible collector.

### Trace Structure

The trace hierarchy is well-designed:
- `claude_code.interaction` -- one agent turn
- `claude_code.llm_request` -- individual API call (model, latency, tokens)
- `claude_code.tool` -- tool invocations with child spans for permission wait and execution
- `claude_code.hook` -- hook execution (beta)

Subagent calls nest correctly under parent spans, giving a complete delegation tree. W3C trace context propagates automatically, including into Bash commands run by the agent.

### Privacy Controls

Telemetry is structural by default (durations, tool names, model names). Content logging is opt-in via:
- `OTEL_LOG_USER_PROMPTS=1` -- prompt text
- `OTEL_LOG_TOOL_DETAILS=1` -- file paths, shell commands
- `OTEL_LOG_TOOL_CONTENT=1` -- full tool I/O (truncated at 60KB)
- `OTEL_LOG_RAW_API_BODIES` -- full API request/response JSON

### Third-Party Tools

**claude-code-otel (ColeMurray)**: Docker Compose stack with OTEL Collector -> Prometheus + Loki -> Grafana. Provides 6 pre-built dashboards (overview, cost/usage, tool performance, errors, user activity, event logs). Good for self-hosted monitoring.

**claude_telemetry (TechNickAI)**: Python wrapper ("claudia" command) that instruments Claude Code sessions via hooks. Supports Logfire, Sentry, Honeycomb, Datadog, or any OTLP endpoint. Captures per-execution and per-tool-call metrics. Requires Python 3.10+.

**claude-usage (phuryn)**: Local-only dashboard that reads Claude Code's JSONL transcripts from `~/.claude/projects/`. Pure Python stdlib, SQLite backend. Shows token breakdown, model identification, daily/weekly summaries. No external dependencies.

### Built-in Cost Tracking

Claude Code has `/usage` (aliases: `/cost`, `/stats`) that shows session cost, plan usage limits, and activity stats. The dollar figure is an estimate computed locally from token counts. The API provides `/v1/organizations/cost_report` and `/v1/organizations/usage_report/messages` for organization-level tracking.

## Analysis for gdev

### Where OTEL Adds Real Value

1. **Cost attribution per client project**: For a consulting firm billing AI usage to clients, OTEL resource attributes like `project.name=acme-corp` and `client.id=acme` enable cost breakdown by engagement. The built-in `/cost` command only shows session-level totals.

2. **Audit trails**: The tool_decision, tool_result, and permission_mode_changed events create a per-user audit trail. For consulting firms with compliance requirements (SOC 2, client security policies), this is genuinely useful.

3. **Team-level usage visibility**: Without OTEL, each developer's usage is siloed in their local `~/.claude/` directory. OTEL centralizes this to a shared backend, enabling team dashboards.

4. **Long-running agent monitoring**: For `cowork` sessions or CI-integrated agents, OTEL provides real-time visibility into what the agent is doing, how many tokens it has consumed, and whether it is stuck.

### Where OTEL Is Noise

1. **Individual developer sessions**: A single developer working interactively does not need Grafana dashboards. `/cost` and local JSONL transcripts are sufficient.

2. **Small teams without billing attribution**: If AI costs are not client-billable, the overhead of running a collector + Prometheus + Grafana exceeds the value of the data.

3. **Privacy-sensitive environments**: Some clients may prohibit telemetry export, even structural. The opt-in privacy model is good but adds compliance burden.

## Recommendation for gdev

**Include as an optional, profile-driven configuration -- not a default.**

Concrete implementation:
- gdev's `claudecode` addon should generate the OTEL environment variables in `.envrc` or `devenv.nix` when a profile includes telemetry configuration
- The profile specifies the collector endpoint, service name, and which signals to enable
- `gdev init --profile consulting-with-otel` sets it up; plain `gdev init` does not
- gdev should NOT ship or manage the collector infrastructure (Prometheus/Grafana/etc.) -- that is orthogonal infrastructure
- Generate a `starship.toml` or env var that indicates OTEL is active (visual confirmation)

**Estimated value: medium for consulting firms with client billing, low for individual developers.**

The right abstraction is: gdev configures the Claude Code side (env vars), the firm operates the collector side (infrastructure). gdev bridges the gap by making the env var configuration automatic and profile-driven rather than manual.

## Depth Checklist

- [x] Underlying mechanism explained -- OTEL env vars, three signal types, trace hierarchy
- [x] Key tradeoffs -- value for billing/audit vs noise for individuals, privacy controls
- [x] Compared to alternatives -- native /cost, local JSONL, third-party wrappers
- [x] Failure modes -- flush timeout on short-lived calls, console exporter conflict with SDK pipe, cowork sessions not in local logs
- [x] Concrete examples -- claude-code-otel Docker stack, claude_telemetry wrapper, claude-usage local dashboard
- [x] Standalone-readable -- yes
