# devenv.sh Services Catalog

- **Source URL**: https://devenv.sh/services/
- **Retrieval Date**: 2026-05-14

## Complete List of Native Services (42 total)

1. **Adminer** — Database management web UI
2. **Blackfire** — PHP performance profiling
3. **Caddy** — Modern web server with automatic HTTPS
4. **Cassandra** — Distributed wide-column database
5. **ClickHouse** — Column-oriented analytics database
6. **CockroachDB** — Distributed SQL database
7. **CouchDB** — Document-oriented database
8. **DynamoDB Local** — AWS DynamoDB local emulation
9. **ElasticMQ** — SQS-compatible message queue
10. **Elasticsearch** — Search and analytics engine
11. **Garage** — S3-compatible distributed object storage
12. **httpbin** — HTTP request/response testing service
13. **InfluxDB** — Time-series database
14. **Kafka** — Distributed event streaming platform
15. **Keycloak** — Identity and access management
16. **MailHog** — Email testing tool (older, superseded by Mailpit)
17. **Mailpit** — Email & SMTP testing tool
18. **Meilisearch** — Lightweight search engine
19. **Memcached** — Distributed in-memory caching
20. **MinIO** — S3-compatible object storage
21. **MongoDB** — Document database
22. **Mosquitto** — MQTT message broker (IoT)
23. **MySQL** — Relational database
24. **NATS** — Cloud-native message broker
25. **Nginx** — Web server / reverse proxy
26. **NixSeparateDebugInfoD** — Nix debug info service
27. **OpenSearch** — Elasticsearch fork (search engine)
28. **OpenTelemetry Collector** — Telemetry data collection
29. **PostgreSQL** — Relational database
30. **Prometheus** — Metrics monitoring system
31. **RabbitMQ** — Message broker (AMQP)
32. **Redis** — In-memory data store
33. **RustFS** — File system service
34. **sqld** — LibSQL server (SQLite-compatible)
35. **Tailscale** — VPN/mesh networking
36. **Temporal** — Workflow orchestration engine
37. **Tideways** — PHP application profiling
38. **TrafficServer** — HTTP caching proxy
39. **Typesense** — Search engine
40. **Varnish** — HTTP accelerator/cache
41. **Vault** — Secrets management
42. **WireMock** — API mocking/stubbing

## General Notes

- Services are "pre-configured interfaces for existing software"
- Start with `devenv up` (foreground) or `devenv up -d` (background)
- State persists in `$DEVENV_STATE` directory
- Each service has an `enable` option and service-specific configuration
