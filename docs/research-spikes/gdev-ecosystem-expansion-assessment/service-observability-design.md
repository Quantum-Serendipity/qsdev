# Service Template Expansion & Observability Sidecar — Implementation Unit Design

## Purpose

Design implementation units for two plan amendments:
- **Part A** — Expand Phase 3 service sub-templates with Kafka (Tier 1) and four Tier 2 services (MinIO, Mailpit, Keycloak, NATS)
- **Part B** — Add an observability sidecar to Phase 12 via `gdev enable observability`

All units follow the existing unit format from their respective phases.

## Research Foundation

| Source | What it provides |
|--------|-----------------|
| `dev-services-observability-research.md § 2-5` | Per-service assessment, devenv.sh quality ratings, detection heuristics, tier recommendations |
| `dev-services-observability-research.md § 4` | Observability architecture decision: Docker sidecar via `grafana/otel-lgtm`, not native devenv service |
| `docs/devenv-kafka-config.md` | KRaft mode default, Connect built-in, broker settings, JVM options |
| `docs/devenv-minio-config.md` | Bucket auto-creation, client (mc) included, afterStart hooks, web UI |
| `docs/devenv-mailpit-config.md` | SMTP 1025, web UI 8025, minimal config surface |
| `docs/devenv-keycloak-config.md` | Realm import/export, plugin support, dev-file/dev-mem DB modes, settings passthrough |
| `docs/devenv-nats-config.md` | JetStream persistence, monitoring endpoint, clustering, authorization |
| `docs/grafana-docker-otel-lgtm.md` | Ports 3000/4317/4318, all-in-one container, OTLP defaults, data persistence volume |
| `phases/03-devenv-addon-core-generation.md § Unit 2.3` | Existing service sub-template pattern (postgres/redis/mysql/mongodb/elasticsearch/rabbitmq) |
| `phases/12-extended-integrations-lifecycle.md § Unit 12.1` | Tool lifecycle system (`gdev enable/disable`), file ownership, shared-file surgery |

---

## Part A: Service Template Expansion (Phase 3 Amendment)

These units extend Phase 3 by adding six new service sub-templates to the existing set in Unit 2.3. Each service follows the identical pattern: a `.nix.tmpl` file, a `ServiceChoice` struct entry, wizard form group integration, and detection heuristics.

Unit 2.3's existing scope (PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, RabbitMQ) is unchanged. The new units are additive — they create additional sub-templates that compose into devenv.nix via the same mechanism.

---

### Unit 2.6: Kafka Service Sub-Template (Tier 1)

**Description:** Implement the Kafka devenv.nix sub-template with KRaft mode default, exposing `KAFKA_BOOTSTRAP_SERVERS` and optional Kafka Connect configuration.

**Context:** Kafka is the second most common message broker after RabbitMQ in consulting engagements. devenv.sh has excellent native Kafka support with KRaft mode as default (no Zookeeper dependency), Kafka Connect built-in, and comprehensive broker settings. KRaft mode eliminates the operational complexity of Zookeeper, making Kafka viable as a local dev service with devenv. The devenv module handles log directory formatting, listener configuration, and JVM options automatically. This is a Tier 1 service — it should be offered alongside the existing six in the wizard's service selection form group.

**Desired Outcome:** `gdev devenv add-service kafka` generates a valid Kafka service block in devenv.nix with KRaft mode, sensible defaults, and `KAFKA_BOOTSTRAP_SERVERS` environment variable.

**Steps:**
1. Create `templates/services/kafka.nix.tmpl` with the following template:
   ```nix
   # --- kafka ---
   services.kafka = {
     enable = true;
     defaultMode = "kraft";
     settings = {
       listeners = [ "PLAINTEXT://localhost:{{.Kafka.Port}}" ];
       "log.dirs" = [config.env.DEVENV_STATE + "/kafka-logs"];
     };
     {{- if .Kafka.ConnectEnabled }}
     connect = {
       enable = true;
       settings = {
         "bootstrap.servers" = ["localhost:{{.Kafka.Port}}"];
       };
     };
     {{- end }}
   };
   # --- end kafka ---
   ```
2. Define `KafkaServiceConfig` struct with fields: `Port` (default 9092), `ConnectEnabled` (default false), `JvmOpts` (default `["-Xmx256m", "-Xms256m"]`).
3. Add environment variable generation:
   ```nix
   env.KAFKA_BOOTSTRAP_SERVERS = "localhost:{{.Kafka.Port}}";
   ```
4. Implement detection heuristics in the service detector:
   - `docker-compose.yml` or `docker-compose.yaml`: image matching `confluentinc/cp-kafka`, `bitnami/kafka`, `apache/kafka`
   - `package.json` dependencies: `kafkajs`, `kafka-node`
   - `go.mod` imports: `github.com/IBM/sarama`, `github.com/segmentio/kafka-go`, `github.com/confluentinc/confluent-kafka-go`
   - `requirements.txt` / `pyproject.toml`: `confluent-kafka`, `kafka-python`, `aiokafka`
   - `pom.xml` / `build.gradle`: `spring-kafka`, `org.apache.kafka`
   - `application.properties` / `application.yml`: keys matching `spring.kafka.*`
   - Terraform files: resources matching `aws_msk_cluster`, `confluent_kafka_*`
5. Add Kafka to the wizard service selection form group as a Tier 1 service (same group as PostgreSQL, Redis, etc.), with display name "Apache Kafka (Event Streaming)" and description "Local Kafka broker in KRaft mode (no Zookeeper)".
6. Wire `ServiceChoice{Name: "kafka"}` through the template data pipeline to compose the Kafka sub-template into devenv.nix.
7. Write unit tests: render Kafka-only, render Kafka + PostgreSQL combo, validate rendered Nix syntax, verify KRaft mode default, verify env var presence.

**Acceptance Criteria:**
- [ ] Kafka template renders valid Nix with `services.kafka.enable = true` and `defaultMode = "kraft"`
- [ ] `KAFKA_BOOTSTRAP_SERVERS` environment variable set in devenv.nix
- [ ] Kafka Connect block conditionally included when enabled
- [ ] Detection heuristics trigger for docker-compose Kafka images, kafkajs/sarama deps, and Spring Kafka config
- [ ] Kafka appears in the wizard service selection alongside existing Tier 1 services
- [ ] Rendered Nix passes `nix-instantiate --parse`
- [ ] Kafka + other service combinations render without conflicts

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 2 Kafka` — assessment, detection heuristics, tier justification
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devenv-kafka-config.md` — full devenv.sh Kafka service options
- `phases/03-devenv-addon-core-generation.md § Unit 2.3` — existing service sub-template pattern

**Status:** Not Started

---

### Unit 2.7: MinIO Service Sub-Template (Tier 2)

**Description:** Implement the MinIO devenv.nix sub-template with S3-compatible object storage, bucket auto-creation, and standard AWS S3 environment variables.

**Context:** MinIO is the most commonly needed infrastructure service beyond databases and message brokers. Any project using S3 for file storage benefits from local S3 emulation, and MinIO is more accessible than LocalStack for pure S3 needs. devenv.sh has excellent MinIO support including auto-bucket creation, the MinIO client (mc), afterStart hooks for permission setup, and a web console UI. This is a Tier 2 (detect-and-offer) service — the wizard offers it when S3 usage is detected in the project, or when the user explicitly selects it in the customize path.

**Desired Outcome:** `gdev devenv add-service minio` generates a valid MinIO service block in devenv.nix with default bucket, credentials, and standard S3-compatible environment variables.

**Steps:**
1. Create `templates/services/minio.nix.tmpl`:
   ```nix
   # --- minio ---
   services.minio = {
     enable = true;
     accessKey = "minioadmin";
     secretKey = "minioadmin";
     listenAddress = "127.0.0.1:{{.MinIO.Port}}";
     consoleAddress = "127.0.0.1:{{.MinIO.ConsolePort}}";
     region = "{{.MinIO.Region}}";
     browser = true;
     {{- if .MinIO.Buckets }}
     buckets = [
       {{- range .MinIO.Buckets }}
       "{{.}}"
       {{- end }}
     ];
     {{- end }}
   };
   # --- end minio ---
   ```
2. Define `MinIOServiceConfig` struct: `Port` (default 9000), `ConsolePort` (default 9001), `Region` (default "us-east-1"), `Buckets` (default `["dev-bucket"]`), `AccessKey` (default "minioadmin"), `SecretKey` (default "minioadmin").
3. Add environment variable generation:
   ```nix
   env.MINIO_ENDPOINT = "http://localhost:{{.MinIO.Port}}";
   env.AWS_ENDPOINT_URL = "http://localhost:{{.MinIO.Port}}";
   env.AWS_ACCESS_KEY_ID = "minioadmin";
   env.AWS_SECRET_ACCESS_KEY = "minioadmin";
   env.AWS_DEFAULT_REGION = "{{.MinIO.Region}}";
   ```
4. Implement detection heuristics:
   - `docker-compose.yml`: image matching `minio/minio`, `localstack/localstack`
   - `package.json` dependencies: `@aws-sdk/client-s3`, `minio`
   - `go.mod` imports: `github.com/minio/minio-go`, `github.com/aws/aws-sdk-go-v2/service/s3`
   - `requirements.txt` / `pyproject.toml`: `boto3` with S3 usage, `minio`
   - Environment patterns in `.env*` files: `AWS_S3_ENDPOINT`, `S3_ENDPOINT_URL`, `MINIO_*`
   - Terraform: `aws_s3_bucket` resources (when localstack/minio provider endpoint present)
5. Add MinIO to the wizard as a Tier 2 service in a "Infrastructure Services" form sub-group, with display name "MinIO (S3-Compatible Storage)" and description "Local S3 object storage with web console".
6. Wire through template pipeline. When detected, pre-check in wizard with "(detected S3 usage)" annotation.
7. Write unit tests: render MinIO-only, validate bucket list rendering, verify all five env vars, test detection from `@aws-sdk/client-s3` in package.json.

**Acceptance Criteria:**
- [ ] MinIO template renders valid Nix with `services.minio.enable = true`
- [ ] Default bucket created via `buckets` list
- [ ] S3-compatible environment variables set (`AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION`)
- [ ] `MINIO_ENDPOINT` set for MinIO-specific client usage
- [ ] Detection heuristics trigger for S3 SDK imports, minio docker-compose, and S3 env var patterns
- [ ] Wizard offers MinIO when S3 usage detected, with Tier 2 (detect-and-offer) behavior
- [ ] Rendered Nix passes `nix-instantiate --parse`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 3 MinIO` — assessment, detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devenv-minio-config.md` — full devenv.sh MinIO service options
- `phases/03-devenv-addon-core-generation.md § Unit 2.3` — existing service sub-template pattern

**Status:** Not Started

---

### Unit 2.8: Mailpit Service Sub-Template (Tier 2)

**Description:** Implement the Mailpit devenv.nix sub-template for local SMTP email testing with a web UI.

**Context:** Nearly every web application sends email (password resets, notifications, onboarding flows). Mailpit is the modern replacement for MailHog — it catches all outbound SMTP and displays it in a web UI. devenv.sh support is simple: enable, configure SMTP and UI listen addresses, done. Configuration complexity is minimal, making this a low-effort, high-value template. This is Tier 2 (detect-and-offer) — the wizard offers it when SMTP configuration is detected in the project.

**Desired Outcome:** `gdev devenv add-service mailpit` generates a valid Mailpit service block in devenv.nix with standard SMTP environment variables.

**Steps:**
1. Create `templates/services/mailpit.nix.tmpl`:
   ```nix
   # --- mailpit ---
   services.mailpit = {
     enable = true;
     smtpListenAddress = "127.0.0.1:{{.Mailpit.SMTPPort}}";
     uiListenAddress = "127.0.0.1:{{.Mailpit.UIPort}}";
   };
   # --- end mailpit ---
   ```
2. Define `MailpitServiceConfig` struct: `SMTPPort` (default 1025), `UIPort` (default 8025).
3. Add environment variable generation:
   ```nix
   env.SMTP_HOST = "127.0.0.1";
   env.SMTP_PORT = "{{.Mailpit.SMTPPort}}";
   env.MAIL_HOST = "127.0.0.1";
   env.MAIL_PORT = "{{.Mailpit.SMTPPort}}";
   env.MAILPIT_URL = "http://localhost:{{.Mailpit.UIPort}}";
   ```
4. Implement detection heuristics:
   - `docker-compose.yml`: image matching `axllent/mailpit`, `mailhog/mailhog`
   - Environment patterns in config files: `SMTP_HOST`, `MAIL_HOST`, `MAILER_DSN`, `MAIL_MAILER`, `EMAIL_HOST`
   - `application.properties` / `application.yml`: `spring.mail.*` keys
   - Django `settings.py`: `EMAIL_BACKEND`, `EMAIL_HOST` variables
   - Laravel `.env`: `MAIL_MAILER`, `MAIL_HOST`
   - Rails `config/environments/`: `config.action_mailer.smtp_settings`
   - Presence of email-related test directories or files (e.g., `**/email/**`, `**/mailer/**`)
5. Add Mailpit to the wizard as a Tier 2 service in a "Development Tools" form sub-group, with display name "Mailpit (Email Testing)" and description "Local SMTP server with web UI — catches all outbound email".
6. Wire through template pipeline.
7. Write unit tests: render Mailpit-only, verify SMTP env vars, test detection from Spring `application.properties`, test detection from Django `settings.py`.

**Acceptance Criteria:**
- [ ] Mailpit template renders valid Nix with `services.mailpit.enable = true`
- [ ] SMTP environment variables set (`SMTP_HOST`, `SMTP_PORT`, `MAIL_HOST`, `MAIL_PORT`)
- [ ] `MAILPIT_URL` set for web UI access
- [ ] Detection heuristics trigger for SMTP config across Spring, Django, Laravel, Rails
- [ ] Detection heuristics trigger for MailHog docker-compose (migration to Mailpit)
- [ ] Rendered Nix passes `nix-instantiate --parse`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 5 Mailpit` — assessment, detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devenv-mailpit-config.md` — full devenv.sh Mailpit service options
- `phases/03-devenv-addon-core-generation.md § Unit 2.3` — existing service sub-template pattern

**Status:** Not Started

---

### Unit 2.9: Keycloak Service Sub-Template (Tier 2)

**Description:** Implement the Keycloak devenv.nix sub-template for local identity/auth with realm import support and dev database mode.

**Context:** Keycloak is the premier self-hosted identity platform. It has the most sophisticated devenv.sh service module — realm import/export, plugin support, dev database modes (in-memory or file-backed, no external DB), and full settings passthrough. Healthcare, government, and enterprise consulting engagements often require it. Many teams need a local Keycloak to develop and test OIDC/OAuth flows. This is Tier 2 (detect-and-offer) — the wizard offers it when Keycloak or OIDC configuration is detected.

**Desired Outcome:** `gdev devenv add-service keycloak` generates a valid Keycloak service block in devenv.nix with dev-file database mode, realm import from an existing JSON file if present, and standard OIDC environment variables.

**Steps:**
1. Create `templates/services/keycloak.nix.tmpl`:
   ```nix
   # --- keycloak ---
   services.keycloak = {
     enable = true;
     database.type = "{{.Keycloak.DatabaseType}}";
     initialAdminPassword = "{{.Keycloak.AdminPassword}}";
     settings = {
       hostname = "localhost";
       http-port = {{.Keycloak.Port}};
       http-relative-path = "/";
     };
     {{- if .Keycloak.RealmFile }}
     realms.dev = {
       path = "{{.Keycloak.RealmFile}}";
       import = true;
     };
     {{- end }}
   };
   # --- end keycloak ---
   ```
2. Define `KeycloakServiceConfig` struct: `Port` (default 8080), `DatabaseType` (default "dev-file"), `AdminPassword` (default "admin"), `RealmFile` (default "" — empty means no realm import).
3. Add environment variable generation:
   ```nix
   env.KEYCLOAK_URL = "http://localhost:{{.Keycloak.Port}}";
   env.KEYCLOAK_ADMIN = "admin";
   env.KEYCLOAK_ADMIN_PASSWORD = "{{.Keycloak.AdminPassword}}";
   env.OIDC_ISSUER_URL = "http://localhost:{{.Keycloak.Port}}/realms/dev";
   ```
4. Implement detection at init time for existing realm files: glob for `**/realm*.json`, `**/keycloak*.json` in project root — if found, pre-populate `RealmFile` in wizard.
5. Implement detection heuristics:
   - `docker-compose.yml`: image matching `quay.io/keycloak/keycloak`, `jboss/keycloak`
   - `package.json` dependencies: `keycloak-js`, `keycloak-admin-client`, `openid-client`
   - `requirements.txt` / `pyproject.toml`: `python-keycloak`
   - Environment patterns: `KEYCLOAK_*`, `OIDC_ISSUER*`, `OAUTH2_*` in config files
   - Config files: `keycloak.json`, `realm-export.json`
   - Terraform: `keycloak_realm`, `keycloak_*` provider resources
6. Add Keycloak to the wizard as a Tier 2 service in an "Infrastructure Services" form sub-group, with display name "Keycloak (Identity & Auth)" and description "Local OIDC/OAuth2 identity provider with admin console".
7. Handle port conflict: Keycloak defaults to 8080 which conflicts with many app servers. If another service or the project's detected app server uses 8080, auto-adjust to 8180 and note in wizard.
8. Write unit tests: render Keycloak-only, render with realm import, verify OIDC env vars, test port conflict resolution, test detection from keycloak-js dependency.

**Acceptance Criteria:**
- [ ] Keycloak template renders valid Nix with `services.keycloak.enable = true` and `database.type = "dev-file"`
- [ ] Realm import conditionally included when realm JSON file detected
- [ ] OIDC environment variables set (`KEYCLOAK_URL`, `OIDC_ISSUER_URL`)
- [ ] Port conflict detection avoids collision with common app server port 8080
- [ ] Detection heuristics trigger for Keycloak docker-compose, keycloak-js deps, and OIDC env patterns
- [ ] Rendered Nix passes `nix-instantiate --parse`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 3 Keycloak` — assessment, detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devenv-keycloak-config.md` — full devenv.sh Keycloak service options (realm import/export, plugins, SSL/TLS)
- `phases/03-devenv-addon-core-generation.md § Unit 2.3` — existing service sub-template pattern

**Status:** Not Started

---

### Unit 2.10: NATS Service Sub-Template (Tier 2)

**Description:** Implement the NATS devenv.nix sub-template for lightweight cloud-native messaging with optional JetStream persistence.

**Context:** NATS is a lightweight, high-performance messaging system growing in cloud-native and Kubernetes ecosystems. It offers a simpler alternative to Kafka without JVM overhead. devenv.sh has good NATS support with JetStream for persistence, a built-in monitoring endpoint, authorization, and clustering. Configuration complexity is low — just enable, optionally enable JetStream. This is Tier 2 (detect-and-offer) — the wizard offers it when NATS client libraries are detected in the project.

**Desired Outcome:** `gdev devenv add-service nats` generates a valid NATS service block in devenv.nix with optional JetStream and monitoring, and exposes `NATS_URL`.

**Steps:**
1. Create `templates/services/nats.nix.tmpl`:
   ```nix
   # --- nats ---
   services.nats = {
     enable = true;
     host = "127.0.0.1";
     port = {{.NATS.Port}};
     monitoring = {
       enable = true;
       port = {{.NATS.MonitoringPort}};
     };
     {{- if .NATS.JetStreamEnabled }}
     jetstream = {
       enable = true;
       maxMemory = "{{.NATS.JetStreamMaxMemory}}";
       maxFileStore = "{{.NATS.JetStreamMaxFileStore}}";
     };
     {{- end }}
   };
   # --- end nats ---
   ```
2. Define `NATSServiceConfig` struct: `Port` (default 4222), `MonitoringPort` (default 8222), `JetStreamEnabled` (default false), `JetStreamMaxMemory` (default "1G"), `JetStreamMaxFileStore` (default "10G").
3. Add environment variable generation:
   ```nix
   env.NATS_URL = "nats://localhost:{{.NATS.Port}}";
   {{- if .NATS.JetStreamEnabled }}
   env.NATS_JETSTREAM = "true";
   {{- end }}
   ```
4. Implement detection heuristics:
   - `docker-compose.yml`: image matching `nats`, `nats:*`, `synadia/nats-server`
   - `package.json` dependencies: `nats`, `nats.ws`
   - `go.mod` imports: `github.com/nats-io/nats.go`
   - `requirements.txt` / `pyproject.toml`: `nats-py`, `asyncio-nats-client`
   - Config files: `nats-server.conf`, `nats.conf`
   - Rust `Cargo.toml`: `async-nats`, `nats`
5. Add NATS to the wizard as a Tier 2 service in a "Message Brokers" form sub-group alongside Kafka and RabbitMQ, with display name "NATS (Lightweight Messaging)" and description "Cloud-native messaging with optional JetStream persistence".
6. When NATS is selected, show a follow-up toggle: "Enable JetStream persistence?" (default: no). JetStream provides Kafka-like stream persistence and is needed when the project uses NATS for event sourcing or durable queues rather than fire-and-forget messaging.
7. Write unit tests: render NATS-only (no JetStream), render NATS + JetStream, verify env vars, verify monitoring port, test detection from `nats.go` import in go.mod.

**Acceptance Criteria:**
- [ ] NATS template renders valid Nix with `services.nats.enable = true`
- [ ] JetStream block conditionally included when enabled
- [ ] Monitoring endpoint enabled by default
- [ ] `NATS_URL` environment variable set
- [ ] Detection heuristics trigger for NATS client libraries across Node, Go, Python, Rust
- [ ] Wizard offers JetStream toggle as follow-up when NATS selected
- [ ] Rendered Nix passes `nix-instantiate --parse`

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 2 NATS` — assessment, detection heuristics
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/devenv-nats-config.md` — full devenv.sh NATS service options (JetStream, monitoring, auth, clustering)
- `phases/03-devenv-addon-core-generation.md § Unit 2.3` — existing service sub-template pattern

**Status:** Not Started

---

### Unit 2.11: Service Detection Engine Expansion & Wizard Integration

**Description:** Extend the service detection engine and wizard form groups to support Tier 1/Tier 2 service tiering, detect-and-offer behavior, and the new service form group layout.

**Context:** The existing service detection (Unit 2.3) handles six Tier 1 services as equal checkboxes in the wizard. With the addition of Kafka as Tier 1 and four Tier 2 services (MinIO, Mailpit, Keycloak, NATS), the wizard needs tiered presentation: Tier 1 services are always shown in the service form group, while Tier 2 services appear only when detected (pre-checked with a "(detected)" annotation) or when the user is on the customize wizard path. Detection heuristics from Units 2.6-2.10 feed into this engine, but the engine itself — tiering logic, form group layout, and the "detect-and-offer" interaction pattern — is cross-cutting and warrants its own unit.

**Desired Outcome:** The wizard service form group shows all Tier 1 services (7 total) always, and shows Tier 2 services when detected or when in customize mode, with clear tier labeling and detection annotations.

**Steps:**
1. Extend `ServiceDefinition` with a `Tier` field (`Tier1Essential`, `Tier2DetectAndOffer`).
2. Update wizard form builder to read tier:
   - Quick path: show Tier 1 services + any Tier 2 services that were detected (pre-checked).
   - Customize path: show all services, Tier 2 grouped separately with labels.
3. Implement composite detection runner that executes all service detection heuristics from Units 2.3 and 2.6-2.10 against the project directory. Each heuristic returns `(detected bool, signals []string)` where `signals` are the specific files/patterns that triggered detection.
4. Annotate detected services in the wizard: "PostgreSQL (detected: docker-compose.yml)" or "MinIO (detected: @aws-sdk/client-s3 in package.json)".
5. Organize wizard form into sub-groups:
   - **Databases**: PostgreSQL, MySQL/MariaDB, MongoDB, Elasticsearch
   - **Message Brokers**: RabbitMQ, Kafka, NATS
   - **Infrastructure**: MinIO, Keycloak
   - **Development Tools**: Mailpit
6. Add `--service` flag expansion to support all new services: `gdev devenv init --service kafka --service minio`.
7. Write integration tests: project with `docker-compose.yml` containing postgres + kafka + minio images triggers correct detection; project with `package.json` containing `kafkajs` + `@aws-sdk/client-s3` triggers Kafka + MinIO detection.

**Acceptance Criteria:**
- [ ] Tier 1 services always visible in wizard (7 services: existing 6 + Kafka)
- [ ] Tier 2 services appear when detected, with detection signal annotation
- [ ] Customize path shows all services regardless of detection
- [ ] Detection signals are specific and displayed to user (not just boolean)
- [ ] Form sub-groups organize services logically
- [ ] `--service` flag accepts all new service names
- [ ] Multiple concurrent detections compose correctly (Kafka + MinIO + PostgreSQL)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 9` — tier recommendation matrix
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 10` — detection heuristics summary
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md` — huh form construction, progressive disclosure
- `phases/03-devenv-addon-core-generation.md § Unit 2.5` — existing CLI command registration

**Status:** Not Started

---

## Part B: Observability Sidecar (Phase 12 Amendment)

These units add observability as a lifecycle-managed tool in Phase 12, using the `gdev enable/disable` system from Unit 12.1. The observability stack runs as a Docker container (`grafana/otel-lgtm`), not as native devenv services — because Grafana, Loki, and Tempo are not available as devenv.sh service modules.

This is architecturally distinct from the service templates in Part A: those are devenv.nix native services managed by `devenv up`. The observability sidecar is a Docker container with its own lifecycle, managed through the Phase 12 tool lifecycle system.

---

### Unit 12.12: Observability Tool Registration & OTEL Environment Variable Generation

**Description:** Register `observability` as a lifecycle-managed tool that generates OTEL environment variables in devenv.nix and a Docker container management script, using `grafana/otel-lgtm` as the all-in-one observability backend.

**Context:** The gdev-dx-polish spike rejected "OTEL infrastructure" for Claude Code session monitoring — a correct rejection. However, application development observability is a different use case: engineers building instrumented microservices need local trace/metric/log backends to develop against. `grafana/otel-lgtm` bundles the OpenTelemetry Collector, Prometheus, Tempo, Loki, Pyroscope, and Grafana into a single Docker image purpose-built for dev/demo/test environments. This is the observability equivalent of running PostgreSQL locally.

The tool integrates via the Phase 12 lifecycle system (`gdev enable observability` / `gdev disable observability`). It contributes: OTEL environment variables to devenv.nix (shared file), a Docker management script to devenv scripts (shared file), and an exclusive Docker Compose override file for teams preferring Compose.

**Desired Outcome:** `gdev enable observability` adds OTEL environment variables to devenv.nix, creates Docker container management scripts, and enables a local Grafana dashboard at `http://localhost:3000`.

**Steps:**
1. Register `observability` in the tool registry:
   ```go
   Tool{
       Name:        "observability",
       DisplayName: "Local Observability Stack (OTEL + Grafana)",
       Category:    "infrastructure",
       Description: "Docker-based Grafana + Loki + Tempo + Prometheus via grafana/otel-lgtm",
       Default:     OnWhenDetected,
       DetectFunc:  detectOTELUsage,
       Prerequisites: []string{"docker"},
       OwnedFiles: []FileOwnership{
           {Path: "devenv.nix", Ownership: Shared, SectionID: "observability"},
           {Path: "docker-compose.observability.yml", Ownership: Exclusive},
       },
   }
   ```
2. Implement `detectOTELUsage` detection function:
   - `package.json` dependencies: `@opentelemetry/*` packages (any match)
   - `go.mod` imports: `go.opentelemetry.io/otel*`
   - `requirements.txt` / `pyproject.toml`: `opentelemetry-sdk`, `opentelemetry-api`, `opentelemetry-*`
   - `pom.xml` / `build.gradle`: `io.opentelemetry`
   - `Cargo.toml`: `opentelemetry`, `tracing-opentelemetry`
   - `.NET` `*.csproj`: `OpenTelemetry.*` package references
   - Existing `OTEL_*` environment variables in `.env` files or config
3. Generate OTEL environment variables in devenv.nix (shared file, `observability` section):
   ```nix
   # --- observability ---
   env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4318";
   env.OTEL_EXPORTER_OTLP_PROTOCOL = "http/protobuf";
   env.OTEL_SERVICE_NAME = "{{.ProjectName}}";
   env.OTEL_RESOURCE_ATTRIBUTES = "deployment.environment=development,service.version=dev";
   env.OTEL_TRACES_EXPORTER = "otlp";
   env.OTEL_METRICS_EXPORTER = "otlp";
   env.OTEL_LOGS_EXPORTER = "otlp";
   env.GRAFANA_URL = "http://localhost:3000";
   # --- end observability ---
   ```
4. Generate devenv.nix script entries (shared file, `observability` section) for container lifecycle:
   ```nix
   # --- observability ---
   scripts.observability-up.exec = ''
     if ! docker ps --format '{{.Names}}' | grep -q '^gdev-observability$'; then
       echo "Starting observability stack..."
       docker run -d --name gdev-observability \
         -p 3000:3000 -p 4317:4317 -p 4318:4318 \
         -v "''${DEVENV_STATE}/observability-data:/data" \
         grafana/otel-lgtm:latest
       echo "Grafana: http://localhost:3000 (admin/admin)"
       echo "OTLP gRPC: localhost:4317 | OTLP HTTP: localhost:4318"
     else
       echo "Observability stack already running."
     fi
   '';
   scripts.observability-down.exec = ''
     if docker ps --format '{{.Names}}' | grep -q '^gdev-observability$'; then
       docker stop gdev-observability && docker rm gdev-observability
       echo "Observability stack stopped."
     else
       echo "Observability stack is not running."
     fi
   '';
   scripts.observability-status.exec = ''
     if docker ps --format '{{.Names}}' | grep -q '^gdev-observability$'; then
       echo "Observability stack: RUNNING"
       echo "  Grafana:   http://localhost:3000"
       echo "  OTLP gRPC: localhost:4317"
       echo "  OTLP HTTP: localhost:4318"
       docker ps --filter name=gdev-observability --format 'table {{.Status}}\t{{.Ports}}'
     else
       echo "Observability stack: STOPPED"
       echo "  Run 'observability-up' to start"
     fi
   '';
   # --- end observability ---
   ```
5. Generate `docker-compose.observability.yml` (exclusive file) as an alternative for teams preferring Compose:
   ```yaml
   # Managed by gdev — do not edit
   # Usage: docker compose -f docker-compose.observability.yml up -d
   services:
     observability:
       image: grafana/otel-lgtm:latest
       container_name: gdev-observability
       ports:
         - "3000:3000"   # Grafana UI
         - "4317:4317"   # OTLP gRPC
         - "4318:4318"   # OTLP HTTP
       volumes:
         - observability-data:/data
       restart: unless-stopped
   volumes:
     observability-data:
   ```
6. Contribute CLAUDE.md section (shared, `observability` section) documenting:
   - What the observability stack provides
   - How to start/stop (`observability-up`, `observability-down`, `observability-status`)
   - Grafana URL and default credentials
   - How OTEL env vars are auto-configured
   - That this is for application development, not production monitoring
7. Write unit tests: verify OTEL env vars render correctly, verify Docker script template, verify Docker Compose YAML is valid, verify detection from OTEL SDK packages.

**Acceptance Criteria:**
- [ ] `gdev enable observability` adds OTEL env vars to devenv.nix, creates Docker scripts, generates docker-compose.observability.yml
- [ ] `gdev disable observability` removes all observability artifacts cleanly (OTEL env vars, scripts, compose file, CLAUDE.md section)
- [ ] OTEL env vars use standard OpenTelemetry SDK variable names (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, etc.)
- [ ] `OTEL_EXPORTER_OTLP_ENDPOINT` points to `http://localhost:4318` (HTTP/protobuf, the widest-compatibility default)
- [ ] Docker container uses persistent volume under `DEVENV_STATE` for data retention across restarts
- [ ] Detection heuristics trigger for OTEL SDK dependencies in Node, Go, Python, Java, Rust, .NET
- [ ] Docker prerequisite check: `gdev enable observability` fails cleanly with guidance if Docker is not available
- [ ] Grafana accessible at `http://localhost:3000` with pre-configured OTEL data sources (built into the image)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 4` — observability architecture decision (Docker sidecar, not native devenv)
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/grafana-docker-otel-lgtm.md` — container ports, env vars, data persistence, included components
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — tool lifecycle system, file ownership, shared-file surgery
- `research-spikes/gdev-dx-polish/research.md` — original OTEL rejection (correct for Claude Code monitoring, does not apply to app dev observability)

**Status:** Not Started

---

### Unit 12.13: Observability Container Lifecycle Integration

**Description:** Integrate the observability Docker container with devenv shell lifecycle so it auto-starts on `devenv shell` entry and optionally auto-stops on exit.

**Context:** Unit 12.12 provides manual `observability-up`/`observability-down` scripts. This unit adds automatic lifecycle management: when a developer enters `devenv shell` (or the shell activates via direnv), the observability container starts if not already running. On shell exit, the container can optionally be stopped. This mirrors how `devenv up` manages native services — the observability container should feel like a native service even though it runs in Docker.

The lifecycle hook uses devenv's `enterShell` for start and a trap-based approach for stop. Auto-stop is opt-out (default: leave running) because developers often have multiple shells open and stopping on any shell exit would kill the container for all of them.

**Desired Outcome:** Entering devenv shell auto-starts the observability container. The developer sees a one-line status message. Exiting the last shell optionally stops it.

**Steps:**
1. Add `enterShell` hook to the observability section of devenv.nix (shared file, `observability` section):
   ```nix
   # --- observability-lifecycle ---
   enterShell = ''
     # Auto-start observability stack if enabled and Docker available
     if command -v docker &>/dev/null; then
       if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^gdev-observability$'; then
         observability-up 2>/dev/null
       else
         echo "Observability: http://localhost:3000"
       fi
     fi
   '';
   # --- end observability-lifecycle ---
   ```
2. Add `ObservabilityConfig` fields to the tool's configuration: `AutoStart` (default true), `AutoStop` (default false), `GrafanaPort` (default 3000), `OTLPGrpcPort` (default 4317), `OTLPHttpPort` (default 4318).
3. When `AutoStop` is enabled, add a shell trap to `enterShell`:
   ```nix
   enterShell = ''
     # ... auto-start logic ...
     if [[ "''${GDEV_OBSERVABILITY_AUTOSTOP:-false}" == "true" ]]; then
       trap 'observability-down 2>/dev/null' EXIT
     fi
   '';
   ```
4. Support port customization via wizard: if ports 3000/4317/4318 conflict with the project, allow custom ports. Update all generated env vars and Docker port mappings accordingly.
5. Add port conflict detection: scan `docker-compose.yml` and devenv.nix for services already using ports 3000, 4317, or 4318. If found, auto-adjust observability ports (3001, 4319, 4320) and warn in wizard.
6. Write integration tests: mock `docker` command, verify enterShell triggers observability-up, verify port conflict detection shifts ports.

**Acceptance Criteria:**
- [ ] `devenv shell` auto-starts the observability container when enabled and Docker available
- [ ] Status message displayed on shell entry (either "starting" or URL)
- [ ] Auto-stop is off by default (container persists across shell sessions)
- [ ] Port conflict detection finds collisions and auto-adjusts
- [ ] Custom ports propagate to all env vars, Docker mappings, and CLAUDE.md documentation
- [ ] Graceful degradation: if Docker is unavailable, shell entry proceeds without error (just skips observability)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 4` — recommendation for Docker sidecar lifecycle
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/grafana-docker-otel-lgtm.md` — exposed ports, data persistence
- `research-spikes/gdev-extension-design/devenv-addon-design.md § enterShell` — devenv shell entry hooks
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — shared-file section markers for enterShell

**Status:** Not Started

---

### Unit 12.14: Observability CLI Commands & Wizard Integration

**Description:** Add `gdev observability up/down/status/logs` commands as thin wrappers around the Docker container lifecycle, and integrate observability into the wizard and detection flow.

**Context:** While Units 12.12-12.13 provide devenv-integrated scripts (`observability-up/down/status`), developers may want to manage the observability stack outside of a devenv shell — before entering it, or from a different terminal. `gdev observability *` commands provide this capability. These commands work regardless of whether the user is in a devenv shell, making them useful for troubleshooting and ad-hoc usage.

The wizard integration uses the tool lifecycle's detect-and-offer pattern: when OTEL SDK dependencies are found, the wizard suggests "Would you like a local observability backend?" with a brief explanation.

**Desired Outcome:** `gdev observability up` starts the stack from any terminal. `gdev observability status` shows health. The wizard offers observability when OTEL SDKs are detected.

**Steps:**
1. Register `gdev observability` command group with four sub-commands:
   - `gdev observability up` — Start the `grafana/otel-lgtm` container (equivalent to `observability-up` script but works outside devenv shell).
   - `gdev observability down` — Stop and remove the container.
   - `gdev observability status` — Show container state, port mappings, uptime, and Grafana URL.
   - `gdev observability logs` — Tail Docker container logs (`docker logs -f gdev-observability`).
2. Commands read port configuration from `.devinit/.gdev-init-answers.yaml` (the saved answers file) so they use the correct ports even if customized.
3. `gdev observability up` should:
   - Check Docker availability, fail with install guidance if missing.
   - Check if container already running, print status and exit if so.
   - Pull image if not cached (`docker pull grafana/otel-lgtm:latest` with progress).
   - Start container with correct port mappings and volume mount.
   - Wait for Grafana to be healthy (HTTP check on port 3000, up to 30s timeout).
   - Print summary: Grafana URL, OTLP endpoints, default credentials.
4. `gdev observability status` should show:
   - Container state (running/stopped/not found)
   - Port mappings
   - Uptime
   - Data volume size
   - Quick test: `curl -s localhost:4318/v1/traces` returns 200 (collector accepting data)
5. Integrate into the wizard via the tool lifecycle's detect-and-offer pattern:
   - When `detectOTELUsage()` returns true, show an info panel: "OTEL SDK detected in your project. gdev can provide a local observability backend (Grafana + traces + metrics + logs) via Docker."
   - Follow with a toggle: "Enable local observability stack?" (default: yes when detected)
   - On the customize path, always show observability in the "Infrastructure" tools section.
6. Add `gdev enable observability` / `gdev disable observability` aliases that delegate to the tool lifecycle system (Unit 12.1). These are the primary enable/disable path; the `gdev observability` commands are for runtime management of an already-enabled tool.
7. Write tests: `gdev observability status` with no container returns "not found", `gdev observability up` with mock Docker validates correct flags, wizard integration shows observability when OTEL detected.

**Acceptance Criteria:**
- [ ] `gdev observability up` starts container and waits for health check
- [ ] `gdev observability down` stops and removes container cleanly
- [ ] `gdev observability status` shows container state and endpoints
- [ ] `gdev observability logs` tails container logs
- [ ] Commands work outside devenv shell (read config from saved answers)
- [ ] Wizard offers observability when OTEL SDK deps detected, with clear explanation
- [ ] `gdev enable/disable observability` properly delegates to lifecycle system
- [ ] Docker not-installed error includes platform-specific install guidance

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md § 4` — `gdev observability up/down/status` command design
- `research-spikes/gdev-ecosystem-expansion-assessment/docs/grafana-docker-otel-lgtm.md` — health check endpoints, default credentials
- `phases/12-extended-integrations-lifecycle.md § Unit 12.1` — tool lifecycle system (`gdev enable/disable`)
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md` — detect-and-offer wizard pattern

**Status:** Not Started

---

## Summary

### Phase 3 Amendment — New Units

| Unit | Title | Tier | Service |
|------|-------|------|---------|
| 2.6 | Kafka Service Sub-Template | Tier 1 (essential) | Apache Kafka (KRaft) |
| 2.7 | MinIO Service Sub-Template | Tier 2 (detect-and-offer) | MinIO (S3-compatible) |
| 2.8 | Mailpit Service Sub-Template | Tier 2 (detect-and-offer) | Mailpit (SMTP testing) |
| 2.9 | Keycloak Service Sub-Template | Tier 2 (detect-and-offer) | Keycloak (Identity/auth) |
| 2.10 | NATS Service Sub-Template | Tier 2 (detect-and-offer) | NATS (Messaging) |
| 2.11 | Service Detection Engine Expansion & Wizard Integration | Cross-cutting | Tiering, detection, form groups |

### Phase 12 Amendment — New Units

| Unit | Title | Mechanism |
|------|-------|-----------|
| 12.12 | Observability Tool Registration & OTEL Env Var Generation | Lifecycle tool, devenv.nix env vars, Docker scripts, Compose file |
| 12.13 | Observability Container Lifecycle Integration | enterShell auto-start, port conflict detection |
| 12.14 | Observability CLI Commands & Wizard Integration | `gdev observability up/down/status/logs`, wizard detect-and-offer |

### Phase 3 Updated Completion Criteria

Add to existing Phase 3 completion criteria:
- [ ] `gdev devenv add-service kafka` produces valid Kafka block with KRaft mode
- [ ] All five new service templates pass `nix-instantiate --parse`
- [ ] Tier 2 services detected and offered when project signals present
- [ ] Service form groups organized into Databases / Message Brokers / Infrastructure / Development Tools
- [ ] Port conflicts between Keycloak (8080) and other services resolved automatically

### Phase 12 Updated Completion Criteria

Add to existing Phase 12 completion criteria:
- [ ] `gdev enable observability` generates OTEL env vars, Docker scripts, Compose file
- [ ] `gdev disable observability` removes all observability artifacts cleanly
- [ ] `gdev observability up` starts grafana/otel-lgtm container with health check
- [ ] OTEL SDK detection triggers wizard offer across all supported ecosystems
- [ ] Port conflict detection prevents collisions with project services
- [ ] Observability container data persists across restarts via volume mount
