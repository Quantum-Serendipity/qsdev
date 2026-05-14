# Claude Code OTEL - ColeMurray/claude-code-otel
- **Source**: https://github.com/ColeMurray/claude-code-otel
- **Retrieved**: 2026-03-27

## Architecture
Claude Code → OpenTelemetry Collector → Prometheus (metrics) + Loki (events/logs) → Grafana (visualization)

## OpenTelemetry Integration
Claude Code exports telemetry through OpenTelemetry's standard protocols. Enable via `CLAUDE_CODE_ENABLE_TELEMETRY=1`. Exporters send data via gRPC (port 4317) or HTTP (port 4318) to centralized collector.

## Captured Metrics

### Session & Productivity
- CLI session initiation counts
- Code modifications (lines added/removed)
- Git commits and pull requests created
- Cost tracking per model

### Token & Usage
- Input/output token consumption
- Cache token utilization
- Token creation metrics
- Cost-per-token analysis by model

### Tool Execution
- Code edit tool permission decisions
- Tool execution results with timing data
- API request tracking with duration measurements
- API error capture with status codes

### User Activity
- User prompt submissions (with optional content logging)
- Tool decision logs for audit trails
- Session-level tracking for productivity insights

## Configuration
Key environment variables:
- `OTEL_METRICS_EXPORTER=otlp`
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`
- `OTEL_METRIC_EXPORT_INTERVAL=60000` (production: 1 minute)
- `OTEL_LOG_USER_PROMPTS=1` (optional prompt content logging)
- Cardinality control via `OTEL_METRICS_INCLUDE_SESSION_ID`, `OTEL_METRICS_INCLUDE_ACCOUNT_UUID`

## Dashboard
- Cost & Usage Analysis: Spending breakdown across Claude model versions
- Tool Performance: Frequency rankings, success rates, execution time
- User Activity: Session counts, productivity indicators, DAU/WAU/MAU
- Real-time Monitoring: 30-second refresh intervals

## Note
This uses Claude Code's built-in OTEL support rather than custom hooks. The telemetry is native to Claude Code — `CLAUDE_CODE_ENABLE_TELEMETRY=1` turns it on. The value is in the Grafana dashboard and collector configuration, not custom hook scripts.
