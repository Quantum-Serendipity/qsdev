<!-- Source: https://github.com/ColeMurray/claude-code-otel -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Observability Stack (claude-code-otel)

## Project Summary

A comprehensive monitoring solution for Claude Code usage, performance, and costs. Implements OpenTelemetry standards to collect metrics and events, routing through Prometheus and Loki for storage, visualizing in Grafana dashboards.

## Core Architecture

Claude Code -> OpenTelemetry Collector -> Prometheus (metrics) + Loki (logs) -> Grafana (visualization)

**Key Components:**
- **OpenTelemetry Collector**: Receives telemetry data via gRPC (4317) and HTTP (4318)
- **Prometheus**: Time-series metrics storage and querying (port 9090)
- **Loki**: Log aggregation and event storage (port 3100)
- **Grafana**: Dashboard visualization and analysis (port 3000)

## Quick Start

1. Launch: `make up`
2. Configure Claude Code:
```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```
3. Access Grafana: http://localhost:3000 (admin/admin)

## Monitored Metrics

- Session counts and activity tracking
- Lines of code modified (additions/removals)
- Pull requests and commits created
- Token usage breakdown (input, output, cache, creation)
- Cost tracking per model and time period
- Tool usage patterns and success rates
- API latency and error rates

## Dashboard Sections

1. **Overview**: High-level KPIs
2. **Cost & Usage**: Cost trends by model, token breakdown
3. **Tool Usage & Performance**: Tool frequency, success rates
4. **Performance & Errors**: API latency, error rate tracking
5. **User Activity & Productivity**: Code metrics, commits, PRs
6. **Event Logs**: Real-time tool execution events

## Privacy & Security

- Prompt content logging disabled by default
- All data remains within your infrastructure
- Configurable authentication

## Technical Requirements

- Docker and Docker Compose
- ~2GB RAM for full stack
- Ports 3000 (Grafana), 3100 (Loki), 4317-4318 (Collector), 9090 (Prometheus)
