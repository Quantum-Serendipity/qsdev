<!-- Source: https://github.com/ColeMurray/claude-code-otel -->
<!-- Retrieved: 2026-03-26 -->

# claude-code-otel - OpenTelemetry Observability for Claude Code

Comprehensive monitoring solution for Claude Code using OpenTelemetry, Prometheus, Loki, and Grafana.

## Key Features

- Cost tracking by model with usage breakdown
- User analytics (DAU/WAU/MAU metrics)
- Tool usage frequency and success rates
- API performance metrics and latency analysis
- Productivity measurement through code changes, commits, and pull requests
- Real-time dashboards with 30-second refresh intervals

## Architecture

Claude Code -> OpenTelemetry Collector -> Prometheus (metrics) + Loki (logs/events) -> Grafana (visualization)

**Core Components:**
- OpenTelemetry Collector (Ports 4317/4318): Metrics and logs ingestion
- Prometheus (Port 9090): Metrics storage and querying
- Loki (Port 3100): Log aggregation
- Grafana (Port 3000): Dashboard visualization

## Quick Start

1. `make up` to start all services
2. Configure Claude Code with env vars: `CLAUDE_CODE_ENABLE_TELEMETRY=1`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`
3. Access Grafana at http://localhost:3000

## Available Metrics

Session counts, active users, lines of code modifications, PRs and commits, token usage by type, cost by model version, tool usage decisions and permissions.

## Use Cases

- **Engineering Teams:** Track productivity gains, optimize costs
- **Platform Teams:** Plan capacity, monitor SLAs
- **Management:** Measure ROI, understand adoption patterns

## License
MIT
