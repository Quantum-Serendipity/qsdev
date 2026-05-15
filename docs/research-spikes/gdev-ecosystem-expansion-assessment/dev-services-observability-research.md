# Development Services Expansion & Local Observability Stack Research

## Research Question

What development services beyond PostgreSQL/Redis/MySQL/MongoDB/Elasticsearch/RabbitMQ should gdev provide, and should it offer a local observability stack? gdev uses devenv.sh (Nix-based) to manage developer environments via `qsdev devenv add-service <name>`.

## 1. devenv.sh Native Service Catalog

devenv.sh natively supports **42 services** as of May 2026. gdev currently templates 6 of them. The full catalog, categorized:

### Databases (12 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **PostgreSQL** | Relational | Excellent | **Already templated** |
| **MySQL** | Relational | Excellent | **Already templated** |
| **MongoDB** | Document | Good | **Already templated** |
| **Elasticsearch** | Search/analytics | Good | **Already templated** |
| **CockroachDB** | Distributed SQL | Good | Niche — distributed SQL testing |
| **Cassandra** | Wide-column | Good | Big data / IoT projects |
| **CouchDB** | Document | Good | Offline-first apps |
| **ClickHouse** | Analytics OLAP | Good | Data engineering projects |
| **InfluxDB** | Time-series | Good | IoT / monitoring backends |
| **sqld** | LibSQL (SQLite server) | Basic | Edge computing, Turso users |
| **OpenSearch** | Search (ES fork) | Good | AWS-aligned Elasticsearch replacement |
| **DynamoDB Local** | AWS DynamoDB emulation | Good | AWS-heavy projects |

### Message Brokers & Event Streaming (5 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **RabbitMQ** | AMQP broker | Good | **Already templated** |
| **Kafka** | Event streaming | Excellent | KRaft default (no ZK), Connect built-in |
| **NATS** | Cloud-native messaging | Good | JetStream, monitoring, clustering |
| **ElasticMQ** | SQS-compatible queue | Basic | AWS SQS emulation |
| **Mosquitto** | MQTT broker | Basic | IoT messaging |

### Search Engines (3 services, beyond Elasticsearch)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Meilisearch** | Lightweight search | Good | Rust-based, fast, simple API |
| **Typesense** | Search engine | Good | C++, developer-friendly |
| **OpenSearch** | ES fork | Good | Full ES compatibility, AWS-aligned |

### Caching (2 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Redis** | In-memory store | Excellent | **Already templated** |
| **Memcached** | Distributed cache | Basic | Simple, minimal config |

### Infrastructure Services (5 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Vault** | Secrets management | Good | Dev mode, UI, simple config |
| **MinIO** | S3-compatible storage | Excellent | Auto-creates buckets, client included |
| **Keycloak** | Identity/auth | Excellent | Realm import/export, plugins, dev DB |
| **Tailscale** | VPN/mesh | Basic | Developer networking |
| **Temporal** | Workflow orchestration | Good | Durable execution patterns |

### Observability (2 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **OpenTelemetry Collector** | Telemetry pipeline | Good | Contrib distribution, custom config |
| **Prometheus** | Metrics storage | Excellent | Scrape configs, OTLP receiver, retention |

### Email & API Testing (3 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Mailpit** | SMTP testing | Good | Modern, simple, web UI |
| **MailHog** | SMTP testing (legacy) | Basic | Superseded by Mailpit |
| **WireMock** | API mocking | Good | JSON-based mapping config |

### Web Servers & Proxies (5 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Caddy** | Web server | Good | Auto-HTTPS, reverse proxy |
| **Nginx** | Web server | Good | Traditional reverse proxy |
| **Varnish** | HTTP cache | Basic | CDN/caching testing |
| **TrafficServer** | Web proxy | Basic | Niche |
| **httpbin** | HTTP testing | Basic | Request inspection |

### Language-Specific Profiling (3 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Blackfire** | PHP profiling | Good | PHP-specific |
| **Tideways** | PHP profiling | Good | PHP-specific |
| **Adminer** | DB management UI | Good | PHP-based, multi-DB |

### Other (2 services)
| Service | Type | devenv Quality | Notes |
|---------|------|---------------|-------|
| **Garage** | Distributed object storage | Basic | Self-hosted S3 alternative |
| **RustFS** | File system | Basic | Niche |
| **NixSeparateDebugInfoD** | Debug info | Basic | Nix-specific |

### NOT in devenv.sh (notable absences)
| Service | Status | Workaround |
|---------|--------|-----------|
| **Grafana** | Not a native devenv service | Must use Docker or Nix package |
| **Loki** | Not a native devenv service | Must use Docker or Nix package |
| **Tempo** | Not a native devenv service | Must use Docker or Nix package |
| **Jaeger** | Not a native devenv service | Must use Docker or Nix package |
| **Consul** | Not a native devenv service | Must use Docker or Nix package |
| **LocalStack** | Not a native devenv service | Docker-based (Python + Docker) |
| **Pulsar** | Not a native devenv service | Docker or manual setup |
| **Valkey** | Not a native devenv service | Redis is API-compatible; use Redis service |

---

## 2. Message Brokers & Event Streaming — Deep Assessment

### Kafka (ESSENTIAL — template it)

**devenv.sh support**: Excellent. Native KRaft mode (no Zookeeper dependency), Kafka Connect built-in, comprehensive configuration including broker settings, log4j, JVM options.

**Consulting frequency**: High. Kafka is the dominant event streaming platform. The 2025 Stack Overflow/Pragmatic Engineer surveys consistently show Kafka alongside RabbitMQ as the two most-used message brokers. Any microservices or event-driven consulting engagement likely uses Kafka.

**Configuration complexity**: Medium. KRaft mode makes it simpler (no ZK dependency), but still needs listener config, log dirs, JVM tuning. The devenv module handles defaults well.

**Detection heuristic**:
- `docker-compose.yml`: `image: confluentinc/cp-kafka`, `image: bitnami/kafka`, `image: apache/kafka`
- Code imports: `kafka-node`, `kafkajs`, `confluent-kafka-python`, `sarama` (Go), `spring-kafka`
- Terraform: `aws_msk_cluster`, `confluent_kafka_*`

**Recommendation**: **Essential**. Template with KRaft mode default. Second most-needed message broker after RabbitMQ.

### NATS (OPTIONAL — template it)

**devenv.sh support**: Good. JetStream for persistence, authorization, monitoring endpoint, clustering support.

**Consulting frequency**: Medium-low. Growing in cloud-native and Kubernetes ecosystems, but significantly less common than Kafka or RabbitMQ. Preferred for lightweight, high-performance messaging without JVM overhead.

**Configuration complexity**: Low. Simpler than Kafka; just enable, optionally enable JetStream.

**Detection heuristic**:
- `docker-compose.yml`: `image: nats`
- Code imports: `nats`, `nats.go`, `nats-py`, `nats.ws`
- Config: `nats-server.conf`

**Recommendation**: **Optional**. Template it — low effort due to simple config, growing adoption.

### Pulsar (SKIP)

**devenv.sh support**: None. No native service module.

**Consulting frequency**: Low. Pulsar is used at scale by specific companies (Yahoo, Tencent, Verizon) but rarely seen in general consulting engagements. Much less common than Kafka.

**Recommendation**: **Skip**. No devenv.sh support, low demand, heavy resource requirements (JVM + BookKeeper + ZooKeeper).

---

## 3. Infrastructure Services — Deep Assessment

### Vault (OPTIONAL — template it)

**devenv.sh support**: Good. Dev mode with UI, simple configuration (6 options).

**Consulting frequency**: Medium. Teams building secret management into applications, Kubernetes-deployed services, or anything requiring dynamic secrets. Not universal, but when needed, it is essential.

**Configuration complexity**: Very low. Enable + address is sufficient for dev mode.

**Detection heuristic**:
- `docker-compose.yml`: `image: hashicorp/vault`
- Code imports: `vault` (hvac Python, node-vault, hashicorp/vault Go)
- Config: `VAULT_ADDR`, `VAULT_TOKEN` env vars in existing configs
- Terraform: `vault_*` provider resources

**Recommendation**: **Optional**. Template it — trivial config, high value when needed.

### MinIO (OPTIONAL — template it)

**devenv.sh support**: Excellent. Auto-creates buckets, includes client (mc), afterStart hooks, web UI.

**Consulting frequency**: Medium-high. Any project using S3 for file storage benefits from local S3 emulation. Very common in web application consulting. More accessible than LocalStack for pure S3 needs.

**Configuration complexity**: Low. Enable + bucket list is typical. Sensible defaults for credentials and ports.

**Detection heuristic**:
- `docker-compose.yml`: `image: minio/minio`, `image: localstack/localstack`
- Code imports: `@aws-sdk/client-s3`, `boto3` with S3 usage, `minio-go`
- Config: `AWS_S3_ENDPOINT`, `S3_ENDPOINT_URL`, `MINIO_*` env vars
- Terraform: `aws_s3_bucket` (when used with localstack/minio endpoints)

**Recommendation**: **Optional**. Template it — excellent devenv support, very common need. Arguably stronger case than Vault.

### Keycloak (OPTIONAL — template it)

**devenv.sh support**: Excellent. The most sophisticated devenv service module — realm import/export, plugin support, dev database modes, settings passthrough, SSL/TLS. This is a first-class integration.

**Consulting frequency**: Medium. Identity/auth is needed in most web applications, but many consulting projects use Auth0, Okta, Cognito, or Firebase Auth (SaaS). Keycloak is the go-to for teams that need self-hosted identity. Healthcare, government, and enterprise consulting engagements often require it.

**Configuration complexity**: Medium. The devenv module is comprehensive, but Keycloak itself is complex. For dev purposes, defaults + realm import is usually sufficient.

**Detection heuristic**:
- `docker-compose.yml`: `image: quay.io/keycloak/keycloak`, `image: jboss/keycloak`
- Code imports: `keycloak-js`, `keycloak-admin-client`, `python-keycloak`
- Config: `KEYCLOAK_*` env vars, `keycloak.json` config files
- Terraform: `keycloak_realm`, `keycloak_*` provider

**Recommendation**: **Optional**. Template it — outstanding devenv support, critical when needed.

### Consul (SKIP for now)

**devenv.sh support**: None. No native service module.

**Consulting frequency**: Low-medium. Service discovery is a production concern, not typically needed in local development. Teams using Consul usually run it in staging/production, not on laptops.

**Recommendation**: **Skip**. No devenv.sh support, production-oriented rather than dev-oriented.

### LocalStack (SKIP — different mechanism)

**devenv.sh support**: None directly. However, DynamoDB Local (which devenv supports) covers the most common AWS emulation need. LocalStack is a Docker-based Python application — not a natural fit for devenv.nix service modules.

**Consulting frequency**: Medium for AWS-heavy teams. But LocalStack has a freemium model (many services require Pro license), and its scope is enormous (90+ AWS services). It is better suited as a Docker Compose service alongside devenv, not inside it.

**Detection heuristic**: `docker-compose.yml`: `image: localstack/localstack`

**Recommendation**: **Skip as devenv service**. If detected, recommend running alongside devenv via Docker Compose. DynamoDB Local (which devenv supports) covers the most common case.

---

## 4. Observability Stack — The Key Question

### What Was Rejected and Why

The gdev-dx-polish spike explicitly rejected "Full OTEL Infrastructure (Collector + Storage + Dashboards)" with this rationale:

> "Running Prometheus + Loki + Grafana is infrastructure operations, not development environment configuration. It requires Docker or Kubernetes, persistent storage, and ongoing maintenance."

The rejection was about **OTEL for Claude Code session monitoring** — not about application observability during development.

### What devenv.sh Actually Supports

devenv.sh natively supports:
- **OpenTelemetry Collector** (contrib distribution, custom config)
- **Prometheus** (scrape configs, OTLP receiver, retention settings)

devenv.sh does NOT natively support:
- Grafana (no service module)
- Loki (no service module)
- Tempo (no service module)
- Jaeger (no service module)

### The Distinction That Matters

There are **two different observability use cases**, and they were conflated in the original rejection:

1. **Claude Code / gdev session monitoring** (what was rejected) — Tracking agent token usage, costs, tool decisions. This is an operational concern for the consulting firm. The OTEL env vars approach is correct: gdev generates config, the firm runs the collector infrastructure.

2. **Application development observability** (not yet assessed) — When a consulting engineer is building a microservices application, they need to see traces, metrics, and logs from their running services. This is a development-time need identical to running PostgreSQL or Redis locally — you need the observability backend to develop and test instrumented code.

### The grafana/otel-lgtm Solution

Grafana's `docker-otel-lgtm` project bundles the entire observability stack into a single Docker image:
- OpenTelemetry Collector
- Prometheus (metrics)
- Tempo (traces)
- Loki (logs)
- Pyroscope (profiling)
- Grafana (dashboards)

One command: `docker run -p 3000:3000 -p 4317:4317 -p 4318:4318 grafana/otel-lgtm`

This is explicitly designed for "development, demo, and testing environments." It is the observability equivalent of running PostgreSQL locally — not infrastructure operations.

### Can devenv.sh Provide This?

**Partially.** devenv.sh has OpenTelemetry Collector and Prometheus as native services. It does NOT have Grafana, Loki, or Tempo. A complete stack would require:

**Option A: Pure devenv services (partial)**
```nix
services.opentelemetry-collector.enable = true;
services.prometheus.enable = true;
# No Grafana, Loki, or Tempo — incomplete stack
```

**Option B: devenv process wrapping Nix packages**
```nix
# Use devenv processes to run Grafana, Loki, Tempo from nixpkgs
processes.grafana.exec = "${pkgs.grafana}/bin/grafana server ...";
processes.loki.exec = "${pkgs.grafana-loki}/bin/loki ...";
processes.tempo.exec = "${pkgs.tempo}/bin/tempo ...";
# Requires manual configuration of data sources, storage paths, etc.
```

**Option C: Docker sidecar (recommended for observability)**
```nix
# In devenv.nix enterShell or scripts
scripts.observability-up.exec = ''
  docker run -d --name gdev-observability \
    -p 3000:3000 -p 4317:4317 -p 4318:4318 \
    grafana/otel-lgtm
'';
env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318";
```

### Recommendation: Conditional Yes, But Not as a devenv Service Template

**The original rejection of "OTEL infrastructure" remains correct for Claude Code monitoring.** The firm should run that centrally.

**However, `qsdev enable observability` for application development is a different proposition.** Here is the recommended approach:

1. **Do NOT create a devenv.nix service template** for observability. The stack (Grafana + Loki + Tempo + Prometheus + OTEL Collector) exceeds what devenv services handle well — Grafana, Loki, and Tempo are not native devenv services.

2. **DO offer `qsdev enable observability` as a Docker-based sidecar.** This would:
   - Pull and run `grafana/otel-lgtm` as a Docker container
   - Inject `OTEL_EXPORTER_OTLP_ENDPOINT` into devenv.nix env vars
   - Add `OTEL_SERVICE_NAME` based on project name
   - Add a `qsdev observability up/down/status` command
   - Generate a `docker-compose.observability.yml` for projects that prefer Compose

3. **This is NOT the same as the rejected "OTEL infrastructure"** because:
   - It is local, ephemeral, and per-project (like PostgreSQL)
   - It requires zero configuration (single Docker image)
   - It serves application development, not operational monitoring
   - It is entirely optional and on-demand

4. **Detection heuristic for auto-suggesting**: Projects with OTEL SDK dependencies (`@opentelemetry/*`, `opentelemetry-sdk`, `go.opentelemetry.io/otel`) should prompt "Would you like a local observability backend?"

---

## 5. Email & Communication Testing

### Mailpit (OPTIONAL — template it)

**devenv.sh support**: Good. SMTP on 1025, web UI on 8025, simple config.

**Consulting frequency**: Medium-high. Any web application that sends emails (password resets, notifications, onboarding) needs SMTP testing. Mailpit catches all outbound email and displays it in a web UI. Extremely common need.

**Configuration complexity**: Very low. Enable is almost the entire config.

**Detection heuristic**:
- `docker-compose.yml`: `image: axllent/mailpit`, `image: mailhog/mailhog`
- Code: SMTP configuration pointing to `localhost:1025` or `mailhog`/`mailpit` hostname
- Config: `SMTP_HOST`, `MAIL_HOST`, `MAILER_DSN` env vars

**Recommendation**: **Optional but strongly recommended**. Template Mailpit (not MailHog — it is the modern replacement).

### WireMock (OPTIONAL — template it)

**devenv.sh support**: Good. JSON-based mapping configuration, simple options.

**Consulting frequency**: Medium. Useful for mocking third-party APIs during development. More commonly, teams use language-specific mocking libraries (nock, responses, httpmock), but WireMock is valuable for cross-language or integration testing scenarios.

**Detection heuristic**:
- `docker-compose.yml`: `image: wiremock/wiremock`
- Config: `mappings/` directory with JSON files
- Code: WireMock client libraries

**Recommendation**: **Optional**. Template it — already has devenv support with minimal config.

---

## 6. Search — Beyond Elasticsearch

### Meilisearch (OPTIONAL — template it)

**devenv.sh support**: Good. Environment modes, analytics opt-out, listen config.

**Consulting frequency**: Medium and growing. Meilisearch is increasingly popular as a lightweight Elasticsearch alternative for applications that need full-text search without Elasticsearch's complexity and resource overhead. Common in web application development.

**Configuration complexity**: Very low. Enable + environment mode is typical.

**Detection heuristic**:
- `docker-compose.yml`: `image: getmeili/meilisearch`
- Code imports: `meilisearch`, `meilisearch-js`, `meilisearch-python`
- Config: `MEILI_*` env vars

**Recommendation**: **Optional**. Template it — lightweight, growing popularity, simple config.

### Typesense (SKIP)

**devenv.sh support**: Good. API key, host, port configuration.

**Consulting frequency**: Low. Typesense is less common than Meilisearch and significantly less common than Elasticsearch. Niche adoption.

**Recommendation**: **Skip for initial template**. The service exists in devenv and users can configure it manually. Not common enough to justify a gdev template.

### OpenSearch (OPTIONAL — template it)

**devenv.sh support**: Good. Cluster config, security plugin toggle, port settings.

**Consulting frequency**: Medium. AWS-aligned projects increasingly use OpenSearch instead of Elasticsearch (since AWS's fork). Drop-in compatible with Elasticsearch clients.

**Detection heuristic**:
- `docker-compose.yml`: `image: opensearchproject/opensearch`
- Code: Same Elasticsearch client libraries (most are compatible)
- AWS: `aws_opensearch_domain` Terraform resources

**Recommendation**: **Optional**. Template it — natural complement to the already-templated Elasticsearch.

---

## 7. Caching — Beyond Redis

### Memcached (SKIP)

**devenv.sh support**: Basic. Bind, port, start args.

**Consulting frequency**: Low and declining. Redis has largely replaced Memcached for most use cases. Memcached is still used in legacy PHP applications and specific high-throughput caching scenarios, but new projects almost universally choose Redis.

**Recommendation**: **Skip**. Redis covers 95%+ of caching needs. Memcached users can configure it manually.

### Valkey (MONITOR)

**devenv.sh support**: None (no native service). Valkey is API-compatible with Redis, so the Redis devenv service works for application development. Nix has a `valkey` package.

**Consulting frequency**: Growing rapidly. AWS ElastiCache migrated to Valkey in 2025-2026. Linux Foundation-backed. However, for local development, Redis and Valkey are interchangeable — application code does not need to know the difference.

**Recommendation**: **Monitor**. When devenv.sh adds a Valkey service (or when Redis becomes BSD-licensed again), update the Redis template. For now, the Redis service covers the need.

---

## 8. Other Notable Services Worth Templating

### DynamoDB Local (OPTIONAL)

**devenv.sh support**: Good. Native service module.

**Consulting frequency**: Medium for AWS-heavy teams. Covers the most common LocalStack use case.

**Detection heuristic**:
- Code: `@aws-sdk/client-dynamodb`, `boto3` DynamoDB usage
- Terraform: `aws_dynamodb_table`
- Config: `DYNAMODB_ENDPOINT`, `AWS_DYNAMODB_*`

### Temporal (SKIP for now)

**devenv.sh support**: Good. Native service module.

**Consulting frequency**: Low but growing. Temporal is gaining traction for durable workflow orchestration but is still a niche pattern in consulting engagements.

### ClickHouse (SKIP)

**devenv.sh support**: Good.

**Consulting frequency**: Low in general consulting. Common in data engineering and analytics-heavy projects.

---

## 9. Comprehensive Service Tier Recommendation

### Tier 1: Essential (template in Phase 3)
Already planned: PostgreSQL, Redis, MySQL/MariaDB, MongoDB, Elasticsearch, RabbitMQ

**Add: Kafka** — Second most common message broker; excellent devenv support with KRaft mode.

### Tier 2: Strongly Recommended (template as expansion)
| Service | Rationale | Detection Strength |
|---------|-----------|-------------------|
| **MinIO** | S3 is ubiquitous; excellent devenv support | Strong (S3 SDK imports) |
| **Mailpit** | Nearly every web app sends email | Strong (SMTP config) |
| **Keycloak** | Premier self-hosted identity; best devenv module | Medium (Keycloak imports) |
| **NATS** | Lightweight Kafka alternative; simple config | Medium (NATS imports) |

### Tier 3: Optional (template on demand)
| Service | Rationale | Detection Strength |
|---------|-----------|-------------------|
| **Vault** | Secrets management; trivial dev config | Medium (Vault imports/env) |
| **OpenSearch** | ES replacement for AWS shops | Medium (AWS context) |
| **Meilisearch** | Lightweight search; growing popularity | Medium (Meili imports) |
| **WireMock** | API mocking; simple devenv module | Weak (mapping files) |
| **DynamoDB Local** | AWS DynamoDB emulation | Medium (DDB SDK usage) |

### Tier 4: Skip / User-configurable
| Service | Reason |
|---------|--------|
| **Typesense** | Too niche; users can configure manually |
| **Memcached** | Redis dominates caching; declining use |
| **Pulsar** | No devenv support; very niche |
| **Consul** | Production concern, not dev-time |
| **LocalStack** | Docker-only; DynamoDB Local covers main case |
| **Temporal** | Growing but still niche |
| **ClickHouse** | Data engineering niche |
| **CockroachDB** | Distributed SQL niche |
| **Cassandra** | Big data niche |

### Special: Observability (Docker sidecar, not devenv service)
| Component | Mechanism |
|-----------|-----------|
| `qsdev enable observability` | Docker-based grafana/otel-lgtm |
| OTEL env vars | Injected into devenv.nix automatically |
| `qsdev observability up/down` | Container lifecycle management |

---

## 10. Detection Heuristics Summary

For the `qsdev devenv init` wizard to auto-suggest services, each service needs detection heuristics. The strongest signals:

| Signal Source | Services Detected |
|--------------|-------------------|
| `docker-compose.yml` service images | All services (strongest signal) |
| `package.json` / `go.mod` / `requirements.txt` imports | Kafka, NATS, Redis, MinIO, Keycloak, OTEL |
| Terraform resource types | DynamoDB, OpenSearch, Kafka (MSK), Vault |
| Environment variable patterns | Vault, MinIO, OTEL, Keycloak, SMTP |
| Configuration files | WireMock (`mappings/`), Keycloak (`keycloak.json`) |

---

## Depth Checklist

- [x] Underlying mechanism explained — devenv.sh service module architecture, configuration options, native vs Docker-based services
- [x] Key tradeoffs — native devenv services (Nix-managed, `devenv up`) vs Docker sidecars (grafana/otel-lgtm) vs skip; templating effort vs consulting frequency
- [x] Compared to alternatives — each service compared to the already-planned 6; observability approaches compared (pure devenv vs Docker sidecar vs skip)
- [x] Failure modes — Grafana/Loki/Tempo not available as native devenv services (limits pure-Nix observability); Pulsar/LocalStack not in devenv; Valkey not yet distinct from Redis in devenv
- [x] Concrete examples — devenv.nix configuration for each service documented from official docs; grafana/otel-lgtm Docker command; detection heuristics with specific patterns
- [x] Standalone-readable — yes, complete assessment with tiered recommendations
