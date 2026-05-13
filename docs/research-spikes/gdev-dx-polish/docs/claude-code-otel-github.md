<!-- Source: https://github.com/ColeMurray/claude-code-otel -->
<!-- Retrieved: 2026-05-12 -->

# Claude Code Observability Stack (claude-code-otel)

## Purpose
This project provides comprehensive monitoring for Claude Code usage, tracking spending, performance metrics, and development productivity through an integrated observability platform.

## Architecture

The system follows a standard observability pipeline:

**Data Flow**: Claude Code -> OpenTelemetry Collector -> Storage backends -> Grafana visualization

**Core Components**:
- **OpenTelemetry Collector**: Ingests telemetry via gRPC (port 4317) and HTTP (port 4318)
- **Prometheus**: Time-series metrics storage and querying (port 9090)
- **Loki**: Log aggregation and event storage (port 3100)
- **Grafana**: Dashboard and visualization interface (port 3000)

## Setup Requirements

Minimal prerequisites: Docker and Docker Compose. Quick start involves:
1. Running `make up` to launch services
2. Setting environment variables for telemetry export (endpoint: `http://localhost:4317`)
3. Accessing Grafana at `http://localhost:3000` (default credentials: admin/admin)

## Metrics & Data Collected

**Core Metrics**:
- Session counts, lines of code modified, commits, and pull requests
- Cost tracking by model with token usage breakdown (input/output/cache)
- API request counts and performance data

**Event Data**:
- User prompts, tool execution results, API requests/errors
- Tool permission decisions and execution timing

Configuration allows toggling prompt logging and cardinality control via environment variables.

## Dashboards Provided

Six dashboard sections organized around key concerns:
1. **Overview**: Session summary, cost, tokens, code changes
2. **Cost & Usage**: Model-based spending trends and token efficiency
3. **Tool Performance**: Usage frequency and success rates
4. **Performance & Errors**: API latency and error tracking
5. **User Activity**: Code productivity metrics
6. **Event Logs**: Real-time execution events for troubleshooting

## Known Limitations

No explicit limitations mentioned in documentation. Configuration is flexible for different deployment scales, though cardinality management recommendations suggest considerations for high-volume environments.
