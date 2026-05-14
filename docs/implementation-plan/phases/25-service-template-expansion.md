# Phase 25: Service Template Expansion

## Goal

Expand gdev's service detection and templating from 6 services (PostgreSQL, MySQL, Redis, Elasticsearch, MongoDB, RabbitMQ — the MVP set) to 11 by adding Kafka, MinIO, Mailpit, Keycloak, and NATS. Introduce Tier 1/Tier 2 classification into the service detection engine so the wizard quick path surfaces the most-needed services and the customize path exposes the full catalog. All services use devenv's native `services.*` module system.

## Dependencies

Phase 3 complete (devenv addon core generation, service detection engine, devenv.nix template generation, CLI commands). Phase 6 complete (wizard orchestration, progressive disclosure, customize path).

## Phase Outputs

- 5 new service modules: Kafka (Tier 1), MinIO (Tier 2), Mailpit (Tier 2), Keycloak (Tier 2), NATS (Tier 2)
- Tiered detection engine with Tier 1 (quick path) and Tier 2 (customize path) classification
- Wizard form sub-groups: Databases, Message Brokers, Infrastructure, Development Tools
- Detection annotation surface: wizard shows which signals triggered each service suggestion
- Env var generation for all new services
- Optional config blocks for connector-heavy Kafka and realm-import Keycloak

---

### Unit 25.1: Kafka Service Module (Tier 1)

**Description:** Implement the Kafka service module using KRaft mode exclusively — no Zookeeper dependency. This is Tier 1 because Kafka is the dominant event streaming platform and the second most-used message broker after RabbitMQ.

**Context:** devenv.sh has excellent native Kafka support with KRaft as the default mode. KRaft was stabilized in Kafka 3.3 and became the only supported mode in Kafka 4.0; there is no reason to support Zookeeper mode in new tooling. The consulting frequency assessment in the ecosystem expansion research rates Kafka as essential — any microservices or event-driven consulting engagement likely uses it. The detection signal set is broad: JS/TS (kafkajs, kafka-node), Go (IBM/sarama, segmentio/kafka-go), Python (confluent-kafka, kafka-python, aiokafka), JVM (spring-kafka, org.apache.kafka), Terraform (aws_msk_cluster, confluent_kafka_*), and docker-compose images.

JVM memory pressure is real in local development: an unconfigured Kafka process will default to large heap sizes. The `-Xmx256m -Xms256m` JVM options cap memory usage at a developer-laptop-friendly level while remaining functional for development workloads.

**Code-Grounded Note:** The existing service module interface from Phase 3 defines `Detect() DetectionResult`, `Generate() ServiceConfig`, and `SecurityConfig() SecurityHardening`. The Kafka module implements this interface. The devenv.sh `services.kafka` module accepts `enable`, `settings` (a pass-through to Kafka broker properties), and `jvmOptions`. The Kafka Connect optional block is a separate `services.kafka-connect` module in devenv.sh — the Kafka module should emit it only when connector-heavy patterns are detected.

**Desired Outcome:** Projects with Kafka dependencies get `services.kafka.enable = true;` with KRaft defaults and memory-capped JVM options automatically added to devenv.nix. The `KAFKA_BOOTSTRAP_SERVERS` env var is exported in enterShell. Kafka appears in the wizard quick path when detected.

**Steps:**

1. Create `internal/services/kafka/kafka.go` implementing the `ServiceModule` interface:
   ```go
   type KafkaModule struct{}

   func (m *KafkaModule) Name() string         { return "kafka" }
   func (m *KafkaModule) Tier() ServiceTier    { return TierOne }
   func (m *KafkaModule) DisplayName() string  { return "Apache Kafka" }
   func (m *KafkaModule) Description() string  {
       return "Event streaming platform (KRaft mode, no Zookeeper)"
   }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - Scan `package.json` dependencies for: `kafkajs`, `kafka-node`, `kafka-js`, `node-rdkafka`
   - Scan `go.mod` for: `github.com/IBM/sarama`, `github.com/Shopify/sarama`, `github.com/segmentio/kafka-go`, `github.com/confluentinc/confluent-kafka-go`
   - Scan Python dependency files (`requirements.txt`, `pyproject.toml`, `Pipfile`) for: `confluent-kafka`, `kafka-python`, `aiokafka`, `faust`
   - Scan JVM files (`pom.xml`, `build.gradle`, `build.gradle.kts`) for: `spring-kafka`, `org.apache.kafka`, `kafka-clients`, `kafka-streams`
   - Scan Terraform files (`*.tf`) for: `aws_msk_cluster`, `aws_msk_configuration`, `confluent_kafka_cluster`, `confluent_kafka_topic`
   - Scan `docker-compose.yml` / `docker-compose.yaml` for image names containing: `kafka`, `confluent`, `bitnami/kafka`, `apache/kafka`
   - Scan environment config files (`.env`, `.env.example`, `config/*.yaml`) for: `KAFKA_BOOTSTRAP_SERVERS`, `KAFKA_BROKERS`, `KAFKA_URL`
   - Return `DetectionResult` with signals list (for wizard annotation) and confidence score

3. Implement `Generate(opts ServiceOptions) ServiceBlock`:
   - Core devenv.nix block:
     ```nix
     services.kafka = {
       enable = true;
       settings = {
         # KRaft mode is the default in devenv — no Zookeeper needed
         "log.retention.hours" = 24;
         "auto.create.topics.enable" = true;
       };
     };
     ```
   - JVM options via `devenv.nix` enterShell or `services.kafka.jvmOptions`:
     ```nix
     KAFKA_JVM_PERFORMANCE_OPTS = "-Xmx256m -Xms256m -XX:+UseG1GC";
     ```
   - enterShell env var export:
     ```nix
     KAFKA_BOOTSTRAP_SERVERS = "localhost:9092";
     ```
   - Optional Kafka Connect block: emit only when `connectorsDetected` is true (detected by `kafka-connect` in docker-compose, `KafkaConnector` CRD in Kubernetes YAML, or `connector.class` in properties files):
     ```nix
     services.kafka-connect = {
       enable = true;
       settings = {
         "group.id" = "connect-cluster";
       };
     };
     ```

4. Implement connector detection as a sub-signal:
   - `docker-compose.yml` images: `confluentinc/cp-kafka-connect`, `debezium/connect`
   - `*.properties` files with `connector.class =`
   - Source code referencing `KafkaConnector`, `SinkConnector`, `SourceConnector`
   - If detected, set `connectorsDetected: true` in the detection result

5. Implement `SecurityConfig() SecurityHardening`:
   - No authentication hardening for local dev (Kafka auth is complex and adds friction without value in dev)
   - Document in CLAUDE.md: "Dev Kafka runs unauthenticated — production requires SASL/TLS configuration"

6. Register the module in `internal/services/registry.go`:
   - Add `kafka.KafkaModule{}` to `AllServiceModules`
   - Tier 1 placement ensures it appears in wizard quick path alongside PostgreSQL, MySQL, Redis, Elasticsearch

7. Write unit tests in `internal/services/kafka/kafka_test.go`:
   - `kafkajs` in `package.json` → detected with `npm-import` signal
   - `sarama` in `go.mod` → detected with `go-import` signal
   - `aws_msk_cluster` in Terraform → detected with `terraform-resource` signal
   - `image: bitnami/kafka` in docker-compose → detected with `docker-compose-image` signal
   - `KAFKA_BOOTSTRAP_SERVERS` in `.env.example` → detected with `env-var` signal
   - Connector detection: `confluentinc/cp-kafka-connect` image → `connectorsDetected: true`
   - No signals present → not detected
   - Generated devenv.nix contains `services.kafka.enable = true` and `KAFKA_BOOTSTRAP_SERVERS`
   - Connector signals present → generated block includes `services.kafka-connect`

**Acceptance Criteria:**
- [ ] Kafka module implements `ServiceModule` interface with `Tier() = TierOne`
- [ ] Detection covers kafkajs/kafka-node (Node), sarama/kafka-go (Go), confluent-kafka/kafka-python/aiokafka (Python), spring-kafka/kafka-clients (JVM), aws_msk_cluster/confluent_kafka_* (Terraform), kafka image names (docker-compose), KAFKA_BOOTSTRAP_SERVERS env var references
- [ ] Generated devenv.nix block uses `services.kafka.enable = true` with KRaft default (no Zookeeper config)
- [ ] JVM memory options set to `-Xmx256m -Xms256m` by default
- [ ] `KAFKA_BOOTSTRAP_SERVERS=localhost:9092` exported in enterShell
- [ ] Optional Kafka Connect block emitted only when connector signals are detected
- [ ] Module registered as Tier 1 — appears in wizard quick path when detected
- [ ] Detection result includes signal list for wizard annotation display
- [ ] Unit tests cover all detection signal categories and both generated block variants

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — Kafka assessment, KRaft rationale, detection heuristics, JVM memory guidance, Kafka Connect signals
- `research-spikes/gdev-ecosystem-expansion-assessment/coverage-matrix-research.md` — service tiering framework

**Status:** Not Started

---

### Unit 25.2: MinIO Service Module (Tier 2)

**Description:** Implement the MinIO service module providing S3-compatible local object storage. Detection targets projects using S3 APIs with local/dev endpoint overrides or explicit MinIO references — not all S3 SDK users.

**Context:** Any web application project using file uploads, object storage, or CDN-backed assets benefits from local S3 emulation. MinIO is the dominant self-hosted S3-compatible store, and devenv.sh has excellent native MinIO support including auto-bucket creation, the `mc` (MinIO Client) CLI, and afterStart hooks. The detection heuristic must be more specific than "uses boto3" because boto3 is used for real AWS production workloads too — the signal must be `boto3` (or `@aws-sdk/client-s3`) combined with a local endpoint override indicating the project is already configured to point at a local server. MinIO in docker-compose is a strong unconditional signal. The full AWS SDK env var set is emitted so projects using the standard AWS env var configuration work without code changes.

**Code-Grounded Note:** devenv.sh's `services.minio` module supports `enable`, `buckets` (list of bucket names to create), `package` (to pin a version), and `accessKey`/`secretKey`. For dev environments, the canonical credentials are `minioadmin`/`minioadmin` — this is a widely-known default and is appropriate for local-only dev. Never emit real AWS credentials in generated config; the generated values must be clearly marked as dev-only placeholders.

**Desired Outcome:** Projects with MinIO or local-S3 patterns get `services.minio.enable = true;` in devenv.nix and the full AWS SDK env var set pointing to localhost:9000. The Tier 2 classification ensures MinIO appears in the wizard customize path rather than the quick path.

**Steps:**

1. Create `internal/services/minio/minio.go` implementing `ServiceModule`:
   ```go
   func (m *MinIOModule) Name() string         { return "minio" }
   func (m *MinIOModule) Tier() ServiceTier    { return TierTwo }
   func (m *MinIOModule) DisplayName() string  { return "MinIO (S3-compatible storage)" }
   func (m *MinIOModule) WizardGroup() string  { return "Infrastructure" }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **Strong signals** (each independently sufficient):
     - `docker-compose.yml` image: `minio/minio`, `quay.io/minio/minio`, `bitnami/minio`
     - `.env` / `.env.example` containing `MINIO_` prefixed variables
     - Direct `minio` package in dependencies (`minio-go` in go.mod, `minio` in Python deps, `minio` in package.json)
   - **Combined signals** (require at least two to trigger):
     - `@aws-sdk/client-s3` or `boto3` in dependencies AND one of: `S3_ENDPOINT_URL`, `AWS_ENDPOINT_URL`, `AWS_S3_ENDPOINT`, `LOCALSTACK_HOSTNAME` in env files
     - `aws_s3_bucket` in Terraform AND `endpoint` override in provider config (indicating non-AWS target)
   - Return with `strength: "strong"` vs `strength: "combined"` for wizard annotation

3. Implement `Generate(opts ServiceOptions) ServiceBlock`:
   - Core devenv.nix block:
     ```nix
     services.minio = {
       enable = true;
       # Buckets created automatically on first start
       buckets = [ "dev-uploads" "dev-assets" ];
     };
     ```
   - enterShell env vars (AWS SDK standard configuration):
     ```nix
     # MinIO local S3-compatible storage
     # NOTE: These are local dev credentials only — never use in production
     AWS_ENDPOINT_URL      = "http://localhost:9000";
     AWS_ACCESS_KEY_ID     = "minioadmin";
     AWS_SECRET_ACCESS_KEY = "minioadmin";
     AWS_DEFAULT_REGION    = "us-east-1";
     # For tools that use S3_ENDPOINT_URL instead of AWS_ENDPOINT_URL
     S3_ENDPOINT_URL       = "http://localhost:9000";
     MINIO_CONSOLE_URL     = "http://localhost:9001";
     ```
   - If bucket names are detectable from code (constants referencing `BUCKET_NAME`, `S3_BUCKET` env vars in source), extract and include them in the `buckets` list

4. Implement `SecurityConfig() SecurityHardening`:
   - Add note to CLAUDE.md section: "MinIO credentials (minioadmin/minioadmin) are dev-only placeholders. Never commit real AWS credentials. Use SecretSpec or environment injection for production values."
   - The `AWS_ACCESS_KEY_ID=minioadmin` value must never be confused with a real AWS key — it does not match the `AKIA*` pattern that gitleaks/ripsecrets look for

5. Write unit tests:
   - `minio/minio` in docker-compose → strong detection, `docker-compose-image` signal
   - `minio-go` in go.mod → strong detection, `go-import` signal
   - `boto3` alone (no endpoint override) → NOT detected
   - `boto3` + `S3_ENDPOINT_URL` in `.env.example` → combined detection
   - `aws_s3_bucket` in Terraform with endpoint override → combined detection
   - Generated block includes `AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
   - Generated credentials are dev placeholders, not matching real AWS key patterns

**Acceptance Criteria:**
- [ ] MinIO module implements `ServiceModule` interface with `Tier() = TierTwo` and `WizardGroup() = "Infrastructure"`
- [ ] Strong signals (minio docker-compose image, minio package import, MINIO_ env vars) independently trigger detection
- [ ] Combined signal (S3 SDK + local endpoint reference) triggers detection without false-positives on pure AWS projects
- [ ] Generated block uses `services.minio.enable = true` with default bucket list
- [ ] Full AWS SDK env var set emitted: `AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION`, `S3_ENDPOINT_URL`
- [ ] Emitted credentials are clearly commented as local dev placeholders
- [ ] `MINIO_CONSOLE_URL=http://localhost:9001` exported for web UI access
- [ ] Security note added to CLAUDE.md section warning against committing real credentials
- [ ] Unit tests cover strong signals, combined signals, false-positive avoidance (boto3 alone)

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — MinIO assessment, detection heuristics, devenv.sh support quality, recommended env var set

**Status:** Not Started

---

### Unit 25.3: Mailpit Service Module (Tier 2)

**Description:** Implement the Mailpit service module for SMTP email catch-all and web UI testing. Mailpit intercepts all outbound SMTP traffic in development, making it safe to run email-sending code locally without accidentally delivering to real addresses.

**Context:** Any application that sends email (registration, password reset, notifications, order confirmation) needs a way to test email delivery in development without sending real emails. Mailpit is the modern replacement for MailHog — written in Go, actively maintained, with a cleaner web UI and better performance. The devenv.sh `services.mailpit` module is rated as "Good" in the research. Detection targets explicit SMTP configuration in the project (config files or env vars) plus common email library imports.

**Code-Grounded Note:** devenv.sh's `services.mailpit` module exposes `enable`, `smtpListenAddress` (default `localhost:1025`), and `uiListenAddress` (default `localhost:8025`). Mailpit binds to localhost by default — no firewall exposure. The SMTP port 1025 is used instead of 25 to avoid requiring root privileges.

**Desired Outcome:** Projects with email-sending code or SMTP configuration get `services.mailpit.enable = true;` in devenv.nix and SMTP env vars pointing to localhost:1025. The web UI URL is exported so developers can check captured emails at localhost:8025.

**Steps:**

1. Create `internal/services/mailpit/mailpit.go` implementing `ServiceModule`:
   ```go
   func (m *MailpitModule) Name() string         { return "mailpit" }
   func (m *MailpitModule) Tier() ServiceTier    { return TierTwo }
   func (m *MailpitModule) DisplayName() string  { return "Mailpit (SMTP email testing)" }
   func (m *MailpitModule) WizardGroup() string  { return "Development Tools" }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **Env var signals** in `.env`, `.env.example`, config files:
     - `EMAIL_HOST`, `EMAIL_BACKEND`, `SMTP_HOST`, `SMTP_URL`, `SMTP_SERVER`, `MAIL_HOST`, `MAIL_MAILER`
     - Any `smtp://` URL reference in config
   - **Dependency signals**:
     - Node: `nodemailer`, `@sendgrid/mail`, `mailgun.js`, `resend`, `postmark` in `package.json`
     - Python: `smtplib` import in source files, `django.core.mail` import, `flask-mail` in requirements
     - Go: `gopkg.in/gomail.v2`, `github.com/jordan-wright/email`, `github.com/wneessen/go-mail` in go.mod
     - PHP: `symfony/mailer`, `swiftmailer/swiftmailer` in composer.json
     - Ruby: `mail` gem in Gemfile, ActionMailer in Rails projects
   - **Config file signals**:
     - `config/mail.php` (Laravel)
     - `app/config/email.php`
     - `config/mailer.yaml` (Symfony)
   - Return detection result with signal category annotations

3. Implement `Generate(opts ServiceOptions) ServiceBlock`:
   - Core devenv.nix block:
     ```nix
     services.mailpit = {
       enable = true;
       # All outbound SMTP is caught here — nothing actually sent to recipients
     };
     ```
   - enterShell env vars matching common framework config variable names:
     ```nix
     # Mailpit SMTP catch-all (all emails captured, none delivered)
     SMTP_HOST          = "localhost";
     SMTP_PORT          = "1025";
     MAIL_HOST          = "localhost";
     MAIL_PORT          = "1025";
     EMAIL_HOST         = "localhost";
     EMAIL_PORT         = "1025";
     EMAIL_BACKEND      = "django.core.mail.backends.smtp.EmailBackend";  # Django
     MAILPIT_UI         = "http://localhost:8025";
     ```
   - Add CLAUDE.md note: "Email testing: Mailpit catches all SMTP at localhost:1025. View captured emails at http://localhost:8025"

4. Write unit tests:
   - `SMTP_HOST` in `.env.example` → detected with `env-var` signal
   - `nodemailer` in `package.json` → detected with `npm-import` signal
   - `smtplib` import in Python source → detected with `python-import` signal
   - `gopkg.in/gomail.v2` in go.mod → detected with `go-import` signal
   - `symfony/mailer` in composer.json → detected with `composer-import` signal
   - No email signals → not detected
   - Generated block uses `services.mailpit.enable = true`
   - Generated env vars include `SMTP_HOST`, `SMTP_PORT`, `MAILPIT_UI`

**Acceptance Criteria:**
- [ ] Mailpit module implements `ServiceModule` interface with `Tier() = TierTwo` and `WizardGroup() = "Development Tools"`
- [ ] Detection covers env var patterns (SMTP_HOST, EMAIL_HOST, MAIL_HOST), Node/Python/Go/PHP/Ruby email library imports, and framework-specific config files
- [ ] Generated block uses `services.mailpit.enable = true`
- [ ] `SMTP_HOST=localhost`, `SMTP_PORT=1025` exported in enterShell
- [ ] `MAILPIT_UI=http://localhost:8025` exported for web UI access
- [ ] Common framework aliases exported: `MAIL_HOST`, `MAIL_PORT`, `EMAIL_HOST`, `EMAIL_PORT`, `EMAIL_BACKEND` (Django)
- [ ] CLAUDE.md section documents the catch-all behavior and web UI URL
- [ ] Unit tests cover all ecosystem-specific import signals and env var signals

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — Mailpit vs MailHog assessment, devenv.sh support quality, detection rationale

**Status:** Not Started

---

### Unit 25.4: Keycloak Service Module (Tier 2)

**Description:** Implement the Keycloak service module for local identity and OIDC/OAuth2 testing. Uses Keycloak's built-in dev-file persistence mode so no external database is required. Optionally imports a realm from `realm-export.json` when present.

**Context:** Keycloak is the go-to self-hosted identity provider for consulting engagements that cannot use SaaS identity (Auth0, Okta, Cognito). Healthcare, government, and enterprise clients frequently require self-hosted identity. The devenv.sh `services.keycloak` module is rated "Excellent" — it supports realm import/export, plugin loading, dev database modes, settings passthrough, and SSL/TLS. For local development, Keycloak's `--features=dev-profile` flag runs with an embedded H2 database persisted to disk (dev-file mode), eliminating the need to depend on PostgreSQL or MySQL for the Keycloak service itself.

**Code-Grounded Note:** The devenv.sh keycloak module accepts `enable`, `initialAdminPassword`, `settings` (passed to `kc.sh start-dev`), and `themes`/`providers` for customization. Dev-file persistence is Keycloak's default when `kc.sh start-dev` is used. The realm import feature reads from `<projectRoot>/realm-export.json` via `--import-realm` flag. Keycloak 22+ uses Quarkus-based startup which is faster than the legacy WildFly version.

**Desired Outcome:** Projects with Keycloak dependencies or OIDC patterns get a configured Keycloak dev server in devenv.nix without requiring a separate database. If `realm-export.json` exists in the project root, it is automatically imported on first start.

**Steps:**

1. Create `internal/services/keycloak/keycloak.go` implementing `ServiceModule`:
   ```go
   func (m *KeycloakModule) Name() string         { return "keycloak" }
   func (m *KeycloakModule) Tier() ServiceTier    { return TierTwo }
   func (m *KeycloakModule) DisplayName() string  { return "Keycloak (Identity & OIDC)" }
   func (m *KeycloakModule) WizardGroup() string  { return "Infrastructure" }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **Node dependency signals**: `keycloak-js`, `@keycloak/keycloak-admin-client`, `keycloak-connect`, `openid-client`
   - **Python dependency signals**: `python-keycloak`, `oic`, `authlib`
   - **Config signals**:
     - `KEYCLOAK_URL`, `KEYCLOAK_AUTH_SERVER_URL`, `KEYCLOAK_REALM`, `OIDC_ISSUER`, `OIDC_AUTHORITY` in env files
     - `keycloak.json` file (Keycloak adapter config format)
     - `realm-export.json` in project root (explicit Keycloak export)
   - **Terraform signals**: `keycloak_realm`, `keycloak_client`, `keycloak_*` resources in `*.tf` files
   - **docker-compose signals**: image names containing `keycloak` (`quay.io/keycloak/keycloak`, `jboss/keycloak`)

3. Implement `Generate(opts ServiceOptions) ServiceBlock`:
   - Check if `realm-export.json` exists in project root; set `hasRealmImport` accordingly
   - Core devenv.nix block (without realm import):
     ```nix
     services.keycloak = {
       enable = true;
       # Dev-file mode: embedded H2 DB, no PostgreSQL dependency needed
       settings = {
         http-port = 8080;
         hostname = "localhost";
         hostname-strict = false;
         hostname-strict-backchannel = false;
       };
       initialAdminPassword = "admin";
     };
     ```
   - With realm import (when `realm-export.json` detected):
     ```nix
     services.keycloak = {
       enable = true;
       settings = {
         http-port = 8080;
         hostname = "localhost";
         hostname-strict = false;
         hostname-strict-backchannel = false;
         # Import realm from project root on first start
       };
       initialAdminPassword = "admin";
     };
     ```
   - enterShell env vars:
     ```nix
     KEYCLOAK_URL       = "http://localhost:8080";
     KEYCLOAK_REALM     = "dev";
     KEYCLOAK_CLIENT_ID = "dev-client";
     OIDC_ISSUER        = "http://localhost:8080/realms/dev";
     ```
   - CLAUDE.md note: "Keycloak admin UI at http://localhost:8080 — login: admin/admin. Import/export realms via Admin Console or `kc.sh export --dir .`"

4. Implement `SecurityConfig() SecurityHardening`:
   - Add note: "Keycloak admin credentials (admin/admin) are dev-only. Production deployments require secrets management."
   - No pre-commit hook additions (Keycloak config files do not contain secrets in standard patterns)

5. Write unit tests:
   - `keycloak-js` in `package.json` → detected with `npm-import` signal
   - `KEYCLOAK_URL` in `.env.example` → detected with `env-var` signal
   - `keycloak_realm` in Terraform → detected with `terraform-resource` signal
   - `realm-export.json` present → detected + `hasRealmImport: true`
   - `docker-compose` with `quay.io/keycloak/keycloak` → detected with `docker-compose-image` signal
   - Generated block has `services.keycloak.enable = true` with dev settings
   - `realm-export.json` present → generated block includes import configuration
   - Env vars include `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `OIDC_ISSUER`

**Acceptance Criteria:**
- [ ] Keycloak module implements `ServiceModule` interface with `Tier() = TierTwo` and `WizardGroup() = "Infrastructure"`
- [ ] Detection covers keycloak-js/@keycloak/* (Node), python-keycloak/authlib (Python), KEYCLOAK_*/OIDC_* env vars, keycloak.json, realm-export.json, Terraform keycloak_* resources, keycloak docker-compose images
- [ ] Generated block uses `services.keycloak.enable = true` with dev-mode settings (no external DB)
- [ ] Realm import enabled when `realm-export.json` exists in project root
- [ ] `initialAdminPassword` set to `"admin"` (dev-only placeholder, not real credential)
- [ ] `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, `OIDC_ISSUER` exported in enterShell
- [ ] CLAUDE.md section documents admin UI URL, login credentials, and realm management commands
- [ ] Unit tests cover detection signals, realm import detection, generated block variants

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — Keycloak assessment, devenv.sh excellence rating, dev-file mode, realm import support

**Status:** Not Started

---

### Unit 25.5: NATS Service Module (Tier 2)

**Description:** Implement the NATS service module for lightweight cloud-native messaging with optional JetStream persistence. JetStream is enabled by default when JetStream patterns are detected in the project.

**Context:** NATS is a lightweight alternative to Kafka for projects that need high-performance messaging without JVM overhead. It is particularly popular in cloud-native and Kubernetes ecosystems. The research rates NATS as medium-frequency in consulting engagements — growing but significantly less common than Kafka or RabbitMQ. The devenv.sh `services.nats` module supports JetStream, authorization, monitoring, and clustering. The JetStream feature transforms NATS from a pure pub/sub broker into a persistent stream processor — it should be enabled by default only when code patterns indicate the project uses it.

**Code-Grounded Note:** devenv.sh's `services.nats` module accepts `enable`, `jetstream` (bool), `port` (default 4222), `httpPort` (monitoring, default 8222), and `serverConfig` (raw NATS config passthrough). JetStream stores data in a local directory; the devenv module handles the storage path automatically.

**Desired Outcome:** Projects with NATS dependencies get `services.nats.enable = true;` in devenv.nix with JetStream enabled when detected. The `NATS_URL` env var is exported. NATS appears in the wizard customize path under Message Brokers.

**Steps:**

1. Create `internal/services/nats/nats.go` implementing `ServiceModule`:
   ```go
   func (m *NATSModule) Name() string         { return "nats" }
   func (m *NATSModule) Tier() ServiceTier    { return TierTwo }
   func (m *NATSModule) DisplayName() string  { return "NATS" }
   func (m *NATSModule) WizardGroup() string  { return "Message Brokers" }
   ```

2. Implement `Detect(projectRoot string) DetectionResult`:
   - **Dependency signals**:
     - Go: `github.com/nats-io/nats.go`, `github.com/nats-io/nats.go/jetstream` in go.mod
     - Python: `nats-py`, `asyncio-nats-client` in requirements/pyproject.toml
     - Node: `nats`, `@nats-io/nats-base-client` in package.json
     - Rust: `async-nats` in Cargo.toml
   - **Config signals**:
     - `NATS_URL`, `NATS_SERVER`, `NATS_SERVERS` in env files
     - `nats-server.conf` in project root or config directory
   - **docker-compose signals**: image names `nats`, `nats:latest`, `nats:alpine`, `synadia/*`
   - **JetStream sub-signal** (determines `jetstreamDetected` flag):
     - `jetstream` keyword in source code (string literals, function names)
     - `nats.go/jetstream` import path
     - `JetStream()` method calls in source
     - `NATS_STREAM_*` env vars

3. Implement `Generate(opts ServiceOptions) ServiceBlock`:
   - Without JetStream:
     ```nix
     services.nats = {
       enable = true;
     };
     ```
   - With JetStream (when `jetstreamDetected: true`):
     ```nix
     services.nats = {
       enable = true;
       jetstream = true;
     };
     ```
   - enterShell env vars:
     ```nix
     NATS_URL     = "nats://localhost:4222";
     NATS_MONITOR = "http://localhost:8222";
     ```

4. Write unit tests:
   - `github.com/nats-io/nats.go` in go.mod → detected with `go-import` signal
   - `nats-py` in requirements.txt → detected with `python-import` signal
   - `nats` in package.json → detected with `npm-import` signal
   - `async-nats` in Cargo.toml → detected with `cargo-import` signal
   - `NATS_URL` in `.env.example` → detected with `env-var` signal
   - `image: nats` in docker-compose → detected with `docker-compose-image` signal
   - `github.com/nats-io/nats.go/jetstream` import → `jetstreamDetected: true`
   - `NATS_STREAM_NAME` env var → `jetstreamDetected: true`
   - JetStream not detected → generated block has `jetstream = true` omitted
   - JetStream detected → generated block includes `jetstream = true`
   - `NATS_URL=nats://localhost:4222` in generated enterShell

**Acceptance Criteria:**
- [ ] NATS module implements `ServiceModule` interface with `Tier() = TierTwo` and `WizardGroup() = "Message Brokers"`
- [ ] Detection covers nats.go (Go), nats-py (Python), nats (Node), async-nats (Rust), NATS_URL env vars, nats-server.conf, nats docker-compose images
- [ ] JetStream sub-signal detected from: nats.go/jetstream import, `JetStream()` method calls, NATS_STREAM_* env vars
- [ ] Generated block uses `services.nats.enable = true`
- [ ] `jetstream = true` added to generated block only when JetStream is detected
- [ ] `NATS_URL=nats://localhost:4222` exported in enterShell
- [ ] `NATS_MONITOR=http://localhost:8222` exported for monitoring endpoint
- [ ] Unit tests cover all ecosystem signals, JetStream detection, both generated block variants

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — NATS assessment, JetStream feature, devenv.sh support, detection heuristics, consulting frequency

**Status:** Not Started

---

### Unit 25.6: Service Detection Engine Tiering

**Description:** Extend the existing Phase 3 service detection engine with Tier 1/Tier 2 classification, wizard form sub-groups, and detection annotation surfacing. Services are separated into quick-path and customize-path buckets based on their tier.

**Context:** The Phase 3 detection engine treats all services identically. With 11 services (6 MVP + 5 new), presenting all of them on the wizard quick path would overwhelm users. The tiering system mirrors the language ecosystem tier approach: Tier 1 services (PostgreSQL, MySQL, Redis, Kafka, Elasticsearch) are surfaced in the quick path when detected; Tier 2 services (MongoDB, RabbitMQ, MinIO, Mailpit, Keycloak, NATS) are surfaced in the customize path or only when strongly detected. The customize path organizes services into sub-groups (Databases, Message Brokers, Infrastructure, Development Tools) matching the `WizardGroup()` field each module declares.

**Code-Grounded Note:** The Phase 3 wizard integration in `addons/devinit/service_wizard.go` currently renders a flat list of detected services. This unit adds a `ServiceTier` field to `DetectionResult`, a `WizardGroup()` method to `ServiceModule`, and updates the wizard rendering logic to route Tier 1 to quick path and Tier 2 to customize path. The `--answers-file` non-interactive mode should accept services by name regardless of tier.

**Desired Outcome:** The wizard quick path asks "We detected PostgreSQL, Redis, Kafka — include them?" with a single Yes/No. The customize path presents services in four named sub-groups. Detection signals are shown next to each suggested service so engineers understand why it was suggested.

**Steps:**

1. Define the `ServiceTier` type and update the `ServiceModule` interface in `internal/services/types.go`:
   ```go
   type ServiceTier int
   const (
       TierOne ServiceTier = iota + 1  // Quick path: high-frequency services
       TierTwo                          // Customize path: situational services
   )

   // Add to ServiceModule interface:
   Tier() ServiceTier
   WizardGroup() string  // "Databases" | "Message Brokers" | "Infrastructure" | "Development Tools"
   ```

2. Update existing Phase 3 service modules to declare their tier:
   - PostgreSQL: `TierOne`, `"Databases"`
   - MySQL: `TierOne`, `"Databases"`
   - Redis: `TierOne`, `"Databases"` (also used as cache/message broker but primarily tier 1)
   - Elasticsearch: `TierOne`, `"Infrastructure"`
   - MongoDB: `TierTwo`, `"Databases"`
   - RabbitMQ: `TierTwo`, `"Message Brokers"`

3. Update `DetectionResult` to include signal annotations:
   ```go
   type DetectionResult struct {
       Detected   bool
       Confidence float64        // 0.0 to 1.0
       Signals    []DetectedSignal
   }

   type DetectedSignal struct {
       Type        string  // "npm-import", "go-import", "docker-compose-image", "env-var", etc.
       Value       string  // the specific value found, e.g., "kafkajs"
       FilePath    string  // relative path to file where signal was found
       Strength    string  // "strong" | "combined"
   }
   ```

4. Update the service detection engine to classify results by tier:
   ```go
   type TieredDetectionResults struct {
       TierOne []ServiceDetection   // Quick path candidates
       TierTwo []ServiceDetection   // Customize path candidates
   }

   type ServiceDetection struct {
       Module  ServiceModule
       Result  DetectionResult
   }

   func DetectServicesWithTiers(projectRoot string) TieredDetectionResults
   ```

5. Update wizard quick path rendering in `addons/devinit/service_wizard.go`:
   - Quick path: show detected Tier 1 services as a multi-select with signal annotations
     ```
     Detected services:
       [x] PostgreSQL (signals: docker-compose image "postgres:16", DATABASE_URL in .env.example)
       [x] Kafka     (signals: go.mod import "github.com/IBM/sarama", KAFKA_BOOTSTRAP_SERVERS in .env)
     ```
   - Tier 2 services with strong detection: show after Tier 1 quick path with "Additional detected services" label
   - Tier 2 services without detection: available in customize path only

6. Update wizard customize path rendering:
   - Organize services into four sub-group sections within the customize form:
     ```
     ── Databases ──────────────────────────────────────────
     [x] PostgreSQL    [ ] MySQL    [ ] MongoDB    [ ] SQLite
     
     ── Message Brokers ────────────────────────────────────
     [x] Kafka    [ ] RabbitMQ    [ ] NATS
     
     ── Infrastructure ─────────────────────────────────────
     [ ] Redis    [ ] Elasticsearch    [ ] MinIO    [ ] Keycloak
     
     ── Development Tools ──────────────────────────────────
     [ ] Mailpit
     ```

7. Update `--answers-file` format to accept services by name:
   ```yaml
   services:
     - name: kafka
     - name: minio
     - name: mailpit
   ```
   - All services accepted by name regardless of tier (non-interactive mode bypasses tier routing)

8. Add Keycloak dependency awareness:
   - If Keycloak is selected and no database service is selected, display info message: "Keycloak uses its embedded dev-file DB in this configuration — no PostgreSQL needed."
   - This is informational only, not a blocking prompt

9. Write integration tests:
   - Project with only Tier 1 signals → wizard quick path shows Tier 1, no Tier 2 presented
   - Project with Tier 2 strong signal (e.g., minio/minio docker-compose) → shown after quick path
   - Tier 2 service without detection signal → not shown on quick path or after-quick, only in customize
   - Detection annotations shown correctly with signal type, value, and file path
   - `--answers-file` with `services: [{name: kafka}, {name: nats}]` → both services generated regardless of tier
   - Keycloak selected without database → info message rendered, no blocking error

**Acceptance Criteria:**
- [ ] `ServiceTier` type added with `TierOne` and `TierTwo` values; `ServiceModule` interface extended with `Tier()` and `WizardGroup()`
- [ ] All 6 existing Phase 3 services updated with tier and wizard group declarations
- [ ] `DetectionResult` extended with `Signals []DetectedSignal` including type, value, file path, and strength
- [ ] `DetectServicesWithTiers()` returns results split into `TierOne` and `TierTwo` slices
- [ ] Wizard quick path shows only Tier 1 detected services with signal annotations
- [ ] Wizard customize path organizes all services into four sub-groups: Databases, Message Brokers, Infrastructure, Development Tools
- [ ] Tier 2 services with strong detection signals shown as secondary suggestions after quick path
- [ ] `--answers-file` accepts services by name regardless of tier
- [ ] Keycloak selection without database triggers informational message (not blocking error)
- [ ] Integration tests verify tier routing, signal annotation display, customize sub-groups, and answers-file bypass

**Research Citations:**
- `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` — service tiering rationale, consulting frequency ratings, full catalog assessment
- `research-spikes/gdev-ecosystem-expansion-assessment/coverage-matrix-research.md` — tiering framework, wizard structure
- `phases/06-wizard-orchestration.md` — existing wizard structure, quick path vs customize path, answers-file format

**Status:** Not Started

---

## Code-Grounded Implementation Notes

### Existing Types to Extend

| Type | Location | Change Needed |
|------|----------|---------------|
| `ServiceModule` interface | `internal/services/types.go` | Add `Tier() ServiceTier` and `WizardGroup() string` methods |
| `DetectionResult` | `internal/services/types.go` | Add `Signals []DetectedSignal` field |
| Service registry | `internal/services/registry.go` | Register all 5 new modules |
| Wizard service form | `addons/devinit/service_wizard.go` | Add tier routing and sub-group rendering |

### New Service Modules

| Module | Package | Tier | Group |
|--------|---------|------|-------|
| `kafka.KafkaModule` | `internal/services/kafka` | 1 | Message Brokers |
| `minio.MinIOModule` | `internal/services/minio` | 2 | Infrastructure |
| `mailpit.MailpitModule` | `internal/services/mailpit` | 2 | Development Tools |
| `keycloak.KeycloakModule` | `internal/services/keycloak` | 2 | Infrastructure |
| `nats.NATSModule` | `internal/services/nats` | 2 | Message Brokers |

### devenv.sh Module Verification

All five services have been verified as native devenv.sh modules. KRaft mode is confirmed as default for Kafka (no Zookeeper config needed). Keycloak dev-file mode confirmed as default for `kc.sh start-dev`. MinIO auto-bucket creation via afterStart hooks confirmed.

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] All 11 services (6 MVP + 5 new) generate valid devenv.nix blocks that `nix flake check` accepts
- [ ] Tier 1 services appear in wizard quick path when detected; Tier 2 in customize path
- [ ] Detection annotation surface shows signal type, value, and source file for each detected service
- [ ] Wizard customize path presents four labeled sub-groups: Databases, Message Brokers, Infrastructure, Development Tools
- [ ] Keycloak generates working dev config without requiring PostgreSQL or MySQL service
- [ ] Kafka generates KRaft-mode config with memory-capped JVM options
- [ ] MinIO emits dev-only credentials with clear comments distinguishing from real AWS credentials
- [ ] All new env vars follow documented naming conventions (AWS SDK standard for MinIO, SMTP standard for Mailpit, etc.)
- [ ] `--answers-file` accepts all services by name regardless of tier classification
