# devenv.sh NATS Service Configuration

- **Source URL**: https://devenv.sh/services/nats/
- **Retrieval Date**: 2026-05-14

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| services.nats.enable | boolean | false | Enable NATS messaging server |
| services.nats.package | package | pkgs.nats-server | NATS server package |
| services.nats.host | string | "127.0.0.1" | Listen address |
| services.nats.port | uint16 | 4222 | Client connection port |
| services.nats.authorization.enable | boolean | false | Client auth |
| services.nats.authorization.user | string | "" | Auth username |
| services.nats.authorization.password | string | "" | Auth password |
| services.nats.authorization.token | string | "" | Auth token |
| services.nats.clientAdvertise | string | "" | URL for cluster advertising |
| services.nats.serverName | string | "" | Server name for clusters |
| services.nats.debug | boolean | false | Debug logging |
| services.nats.trace | boolean | false | Protocol tracing |
| services.nats.logFile | string | "" | Log file path |
| services.nats.monitoring.enable | boolean | true | HTTP monitoring endpoint |
| services.nats.monitoring.port | uint16 | 8222 | Monitoring port |
| services.nats.jetstream.enable | boolean | false | JetStream persistence |
| services.nats.jetstream.maxMemory | string | "1G" | Max memory for streams |
| services.nats.jetstream.maxFileStore | string | "10G" | Max disk for file streams |
| services.nats.settings | attrs | {} | Advanced config (TLS, clustering, MQTT, WebSocket) |

## Notes

- JetStream provides Kafka-like persistence/streaming
- Built-in monitoring endpoint enabled by default
- Supports clustering, TLS, MQTT gateway, WebSocket via settings
