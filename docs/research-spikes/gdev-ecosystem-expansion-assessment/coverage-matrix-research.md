# Coverage Matrix & Gap Inventory

## Current gdev Coverage by Category

### Language Ecosystems (27 total — Phases 2, 7)
| Tier | Ecosystems | Status |
|------|-----------|--------|
| 1 (must-ship) | JS/TS (npm/pnpm/yarn/bun), Python (pip/uv/poetry), Go, Rust, Java/Kotlin (Maven/Gradle), C#/.NET, Docker, Terraform/OpenTofu | Designed |
| 2 (should-ship) | PHP/Composer, Ruby/Bundler, Scala/sbt, C/C++ (Conan/vcpkg/CMake/Meson), Helm, Ansible/Galaxy, Bash/Shell | Designed |
| 3 (nice-to-have) | Elixir/Mix, Dart/Flutter, Swift/SPM, Haskell/Cabal+Stack, Clojure/deps.edn, Bazel/bzlmod, Nix/flakes | Designed |
| 4 (reference only) | Perl/Carton, R/renv, Lua/LuaRocks, Zig, PowerShell/PSGallery | Designed |

### Security Tools (Phases 4, 5, 12)
| Tool | Category | Phase |
|------|----------|-------|
| Semgrep CE | SAST | 12 |
| Gitleaks | Secret scanning | 12 |
| Grype | Container vulnerability | 12 |
| Syft | SBOM generation | 12 |
| Cosign | Container signing | 12 |
| ScanCode Toolkit | License compliance | 12 |
| OSV Scanner | Vulnerability scanning | 5 |
| Harden-Runner | CI egress monitoring | 5 |
| ripsecrets | Secrets detection | 5 |
| Socket.dev MCP | Supply chain analysis | 4/5 |
| attach-guard | Package guardrails (PreToolUse) | 4 |
| package-guard.py | Custom PreToolUse hook | 4 |

### AI Agent Tools (Phases 4, 11, 14)
| Tool | Type | Phase |
|------|------|-------|
| agent-postmortem-skill | Verification skill | 11 |
| Version-Sentinel | Dependency guardrails | 11 |
| semble | Semantic code search MCP | 11 |
| Context7 | Library docs MCP | 12 |
| Trail of Bits skills (3) | Security audit skills | 4 |
| 7 consulting agents | Claude Code agents | 14 |
| 10 gdev operation skills | Claude Code skills | 14 |
| 8 workflow skills | Claude Code skills | 14 |

### Pre-commit Hooks (Phases 5, 12)
30+ hooks: prek, ripsecrets, gofmt, govet, staticcheck, govulncheck, prettier, eslint, ruff, mypy, bandit, rustfmt, clippy, shellcheck, shfmt, statix, hadolint, phpcs, phpstan, rubocop, scalafmt, clang-format, cppcheck, ansible-lint, commitlint, Semgrep, Gitleaks

### Infrastructure Profiles (Phase 1)
| Layer | Tools |
|-------|-------|
| Registry proxy | Nexus, Artifactory, GitHub Packages, GitLab, AWS, GCP, Azure, Verdaccio, artifact-keeper |
| Nix cache | Cachix, Attic, nix-serve |
| Build cache | sccache, ccache, Turborepo, Nx, Bazel Remote |
| Dependency updates | Renovate, Dependabot |

### Development Services (Phase 3 — devenv.nix)
PostgreSQL, Redis, MySQL/MariaDB, MongoDB, Elasticsearch, RabbitMQ

### DX Commands (Phases 12, 16)
qsdev doctor, setup, init, repair, info, outdated, update, teardown, enable, disable, status, list, check, team-report, evidence, changelog

### Git Workflow (Phase 16)
Branch naming hooks, PR templates, commit ticket extraction, automated PR labels, commitlint, git-cliff changelog

### MCP Servers (Phases 4, 11, 12)
Context7, GitHub, Socket.dev, semble, PostgreSQL MCP (5 configured)

### Local Documentation MCP (gdev-local-docs-mcp spike)
DevDocs (offline), Kiwix/ZIM (offline SO+docs), openzim-mcp, skill-level routing, enterprise hosting

### Cross-Platform (Phases 9, 10)
12 OS families, 12 package managers, 5 shells, GoReleaser, nFPM, Homebrew tap, Scoop bucket

---

## Explicitly Rejected Features
1. Standalone task runner (devenv 2.0 native)
2. Container management (Docker/Podman exist)
3. CI execution
4. Deployment
5. Code scaffolding
6. IDE config beyond Claude Code
7. OTEL infrastructure (just env vars)
8. Package manager installation (gdev setup handles)
9. Git server API
10. Vulnerability database
11. Merge queue automation
12. Release automation
13. Nix flake management

---

## Gap Categories Identified

### Category A: Cloud Provider CLI & Credential Management
- AWS CLI (aws-cli v2, aws-vault, aws-sso-util, saml2aws)
- GCP CLI (gcloud, gsutil, bq)
- Azure CLI (az, azd)
- Multi-cloud credential management, SSO integration
- Cloud-specific environment variables and profiles
- **Current coverage**: Zero. Not mentioned in any phase or spike.

### Category B: Kubernetes & Container Orchestration
- kubectl, kustomize, Skaffold, Tilt, DevSpace
- k9s, Lens (terminal/GUI cluster management)
- Helm (covered as ecosystem module, but not as K8s tool)
- kube-context switching, kubeconfig management
- **Current coverage**: Helm ecosystem module only. No K8s tooling.

### Category C: Development Services Expansion
- Kafka, Zookeeper, NATS, Pulsar (message brokers)
- Vault, Consul (HashiCorp stack)
- MinIO (S3-compatible storage)
- Keycloak (identity/auth)
- Jaeger, Zipkin (tracing)
- Prometheus, Grafana (monitoring)
- Mailpit/MailHog (email testing)
- LocalStack (AWS emulation)
- **Current coverage**: 6 services only.

### Category D: API Development & Testing
- httpie, curl, bruno, Insomnia (REST clients)
- grpcurl, grpc-web (gRPC tools)
- Swagger/OpenAPI generators (swagger-codegen, openapi-generator)
- GraphQL tools (graphql-playground, altair)
- Postman/Newman (API testing)
- **Current coverage**: Zero.

### Category E: Database Migration & Schema Management
- Flyway, Liquibase (JVM)
- golang-migrate (Go)
- Prisma, Drizzle, Knex (JS/TS)
- Alembic, Django migrations (Python)
- Entity Framework migrations (.NET)
- Atlas (schema-as-code)
- **Current coverage**: Zero. Per-project concern, but gdev could detect and configure.

### Category F: Observability & Local Dev Stack
- OTEL Collector
- Grafana + Loki + Tempo + Prometheus (local stack)
- Jaeger (distributed tracing)
- **Current coverage**: OTEL env vars only. Explicitly no infrastructure.

### Category G: Git Platform CLIs & Integration
- gh (GitHub CLI)
- glab (GitLab CLI)
- Bitbucket CLI
- **Current coverage**: GitHub MCP server only. No CLI tool installation.

### Category H: Documentation & Diagramming
- Mermaid, D2, PlantUML (diagramming)
- mdbook, docusaurus, mkdocs (doc sites)
- ADR tools (adr-tools, log4brains)
- **Current coverage**: write-adr skill in Phase 14. No tool installation.

### Category I: IDE/Editor Configuration
- VS Code extensions and settings.json
- JetBrains settings (shared via .idea/)
- Neovim/Helix LSP configuration
- EditorConfig (already standard)
- **Current coverage**: Explicitly rejected. Claude Code only.

### Category J: MCP Server Ecosystem Expansion
- Database MCP servers (MySQL, MongoDB, Redis beyond PostgreSQL)
- Ticketing MCP (Linear, Jira)
- Communication MCP (Slack)
- Observability MCP (Grafana, Datadog)
- Cloud provider MCP (AWS, GCP)
- File/search MCP (Brave Search, filesystem)
- **Current coverage**: 5 servers configured. local-docs-mcp spike covers documentation servers.

### Category K: Consulting-Specific Operational Tools
- Time tracking CLIs (Harvest, Toggl, Clockify)
- Client VPN configuration patterns
- Multi-tenant environment switching
- Engagement lifecycle templates
- SOW/contract reference patterns
- **Current coverage**: Consulting agents/skills in Phase 14. teardown/evidence in Phase 16.

### Category L: Runtime Version Management
- mise (polyglot, replaces asdf/rtx)
- fnm, nvm (Node.js version managers)
- pyenv, rbenv, jenv
- **Current coverage**: devenv.sh handles this natively. May be redundant.

### Category M: Code Quality & Coverage
- Codecov, Coveralls (coverage reporting)
- CodeClimate (quality metrics)
- Biome (JS/TS linter+formatter replacing ESLint+Prettier)
- oxlint (faster ESLint alternative)
- **Current coverage**: Semgrep for SAST. Coverage collection in Phase 17 (CI).
