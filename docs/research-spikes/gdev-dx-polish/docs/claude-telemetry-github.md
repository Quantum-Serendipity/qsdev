<!-- Source: https://github.com/TechNickAI/claude_telemetry -->
<!-- Retrieved: 2026-05-12 -->

# claude_telemetry: OpenTelemetry Wrapper for Claude Code CLI

## What It Does

`claude_telemetry` wraps the Claude Code CLI with observability capabilities. It captures execution data from Claude agents running in headless environments (CI/CD, cron jobs, production) where console visibility is limited. The tool operates as a "drop-in replacement that swaps 'claude' command for 'claudia'" while forwarding all flags unchanged to the underlying SDK.

## Architecture & Design

**Pass-Through Model**: The library doesn't interpret or validate flags -- it converts the `extra_args` dictionary directly into CLI arguments and passes them to the Claude Code SDK unchanged. This means future Claude Code features work immediately without updating the wrapper.

**Observability Hooks**: Rather than modifying SDK code, the system uses the SDK's hook mechanism at four key points:
- `UserPromptSubmit`: Opens parent span, logs prompt
- `PreToolUse`: Opens child span, captures tool input
- `PostToolUse`: Captures tool output, closes span
- Session completion: Adds metrics, closes parent span

**Span Hierarchy**: Execution creates a parent span containing child spans for each tool call, with events marking prompt submission and completion.

## Supported Backends

**Enhanced Integration**:
- Logfire (LLM-specific UI with token visualization)
- Sentry (AI Performance dashboard with error tracking)

**Standard OTEL Support**:
- Honeycomb
- Datadog
- Grafana Cloud
- Self-hosted OTEL collectors
- Any OTLP-compatible endpoint

## Data Collection

**Per Execution**:
- Prompt and system instructions
- Model identifier
- Input/output/total token counts
- Cost in USD
- Tool call count
- Execution duration
- Error/failure status

**Per Tool Call**:
- Tool name (Read, Write, Bash, etc.)
- Input parameters
- Output results
- Individual execution timing
- Success/failure indicator

## Installation

```bash
pip install claude_telemetry
pip install "claude_telemetry[logfire]"  # With Logfire support
```

## Configuration

**Environment Variables**:
- `LOGFIRE_TOKEN`: For Logfire backend
- `SENTRY_DSN`: For Sentry monitoring
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Custom OTEL endpoint
- `OTEL_EXPORTER_OTLP_HEADERS`: Authentication headers
- `OTEL_SERVICE_NAME`: Service identifier (defaults to "claude-agents")

## Key Limitations

- Requires Python 3.10+
- Telemetry is sent asynchronously post-execution (traces appear after agent completion)
- The system captures data but doesn't modify agent behavior or decision-making
- MCP server configuration uses standard Claude settings (no special wrapper configuration needed)
