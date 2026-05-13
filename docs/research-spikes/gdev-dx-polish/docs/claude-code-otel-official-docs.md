<!-- Source: https://code.claude.com/docs/en/agent-sdk/observability -->
<!-- Retrieved: 2026-05-12 -->

# Observability with OpenTelemetry (Official Claude Code Docs)

Export traces, metrics, and events from the Agent SDK to your observability backend using OpenTelemetry.

## How telemetry flows from the SDK

The Agent SDK runs the Claude Code CLI as a child process and communicates with it over a local pipe. The CLI has OpenTelemetry instrumentation built in: it records spans around each model request and tool execution, emits metrics for token and cost counters, and emits structured log events for prompts and tool results. The SDK does not produce telemetry of its own. Instead, it passes configuration through to the CLI process, and the CLI exports directly to your collector.

## Three Independent Signals

| Signal     | What it contains                                                            | Enable with                                                         |
| ---------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Metrics    | Counters for tokens, cost, sessions, lines of code, and tool decisions      | `OTEL_METRICS_EXPORTER`                                             |
| Log events | Structured records for each prompt, API request, API error, and tool result | `OTEL_LOGS_EXPORTER`                                                |
| Traces     | Spans for each interaction, model request, tool call, and hook (beta)       | `OTEL_TRACES_EXPORTER` plus `CLAUDE_CODE_ENHANCED_TELEMETRY_BETA=1` |

## Enable telemetry export

Telemetry is off until you set `CLAUDE_CODE_ENABLE_TELEMETRY=1` and choose at least one exporter.

Key environment variables:
- `CLAUDE_CODE_ENABLE_TELEMETRY`: Master switch
- `CLAUDE_CODE_ENHANCED_TELEMETRY_BETA`: Required for traces
- `OTEL_TRACES_EXPORTER`: Set to "otlp"
- `OTEL_METRICS_EXPORTER`: Set to "otlp"
- `OTEL_LOGS_EXPORTER`: Set to "otlp"
- `OTEL_EXPORTER_OTLP_PROTOCOL`: "http/protobuf"
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Collector URL
- `OTEL_EXPORTER_OTLP_HEADERS`: Auth headers

## Trace Spans

- **`claude_code.interaction`**: wraps a single turn of the agent loop
- **`claude_code.llm_request`**: wraps each call to Claude API, with model name, latency, token counts
- **`claude_code.tool`**: wraps each tool invocation, with child spans for permission wait and execution
- **`claude_code.hook`**: wraps each hook execution (requires detailed beta tracing)

Subagent spans nest under parent agent's `claude_code.tool` span -- full delegation chain appears as one trace.

## Trace Context Propagation

The SDK automatically propagates W3C trace context into the CLI subprocess. When called while an OpenTelemetry span is active, the SDK injects TRACEPARENT and TRACESTATE. The CLI also forwards TRACEPARENT to every Bash and PowerShell command it runs.

## Tagging & Filtering

Override `OTEL_SERVICE_NAME` (default: "claude-code") and add `OTEL_RESOURCE_ATTRIBUTES` for deployment metadata. Can attach end-user identity as resource attributes for multi-tenant applications.

## Sensitive Data Controls

| Variable                  | Adds                                                    |
| ------------------------- | ------------------------------------------------------- |
| `OTEL_LOG_USER_PROMPTS=1` | Prompt text on events and interaction span               |
| `OTEL_LOG_TOOL_DETAILS=1` | Tool input arguments (file paths, shell commands)        |
| `OTEL_LOG_TOOL_CONTENT=1` | Full tool input/output bodies, truncated at 60 KB        |
| `OTEL_LOG_RAW_API_BODIES` | Full API request/response JSON as log events             |

## Flush Behavior

Metrics export every 60s, traces/logs every 5s by default. Can be shortened via `OTEL_METRIC_EXPORT_INTERVAL`, `OTEL_LOGS_EXPORT_INTERVAL`, `OTEL_TRACES_EXPORT_INTERVAL`.

## OTLP Metrics (launched April 2, 2026 in public preview)

Completes all three pillars of observability via OTLP. Native support -- no third-party wrappers needed for basic monitoring.
