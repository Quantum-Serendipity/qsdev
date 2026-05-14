# grafana/docker-otel-lgtm: OpenTelemetry Backend in a Docker Image

- **Source URL**: https://github.com/grafana/docker-otel-lgtm
- **Retrieval Date**: 2026-05-14

## Project Purpose

An open-source Docker image bundling complete observability infrastructure. "An OpenTelemetry backend in a Docker image. It bundles the OpenTelemetry Collector, Prometheus (metrics), Tempo (traces), Loki (logs), Pyroscope (profiles), and Grafana into a single container."

**Intended for development, demo, and testing environments** (not production).

## Included Components

- OpenTelemetry Collector — data ingestion and routing
- Prometheus — metrics storage and querying
- Tempo — distributed trace backend
- Loki — log aggregation
- Pyroscope — continuous profiling
- Grafana — visualization and dashboarding
- OBI (optional) — eBPF auto-instrumentation

## Getting Started

```bash
docker pull grafana/otel-lgtm:latest
docker run -p 3000:3000 -p 4317:4317 -p 4318:4318 --rm -ti grafana/otel-lgtm
```

## Exposed Ports

- 3000 — Grafana UI
- 3200 — Loki
- 4040 — Pyroscope
- 4317 — OpenTelemetry Collector (gRPC)
- 4318 — OpenTelemetry Collector (HTTP)
- 9090 — Prometheus

## Environment Variables

### Logging
ENABLE_LOGS_GRAFANA, ENABLE_LOGS_LOKI, ENABLE_LOGS_PROMETHEUS,
ENABLE_LOGS_TEMPO, ENABLE_LOGS_PYROSCOPE, ENABLE_LOGS_OTELCOL, ENABLE_LOGS_ALL

### OBI (eBPF)
ENABLE_OBI=true, OBI_TARGET=java|python|node, OTEL_EBPF_OPEN_PORT=8080

### Export
OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_HEADERS

### Backend Tuning
PROMETHEUS_EXTRA_ARGS, LOKI_EXTRA_ARGS

### Grafana
Default credentials: admin/admin. Configure via GF_* env vars.

## Data Persistence

Mount /data volume: `docker run -v my-data:/data grafana/otel-lgtm`

## Custom Config Files

- /otel-lgtm/prometheus.yaml
- /otel-lgtm/loki-config.yaml
- /otel-lgtm/tempo-config.yaml
- /otel-lgtm/pyroscope-config.yaml
- /otel-lgtm/otelcol-config.yaml

## OpenTelemetry Integration

Applications send data using standard OTLP defaults:
```bash
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318
```

## Example Applications

Java, Go, Python, .NET, Node.js examples on ports 8080-8084.
