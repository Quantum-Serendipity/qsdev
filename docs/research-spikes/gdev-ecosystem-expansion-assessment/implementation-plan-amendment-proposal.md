# Implementation Plan Amendment Proposal: gdev Ecosystem Expansion

## Summary

This proposal adds **51 implementation units** across **9 phase amendments** (including 1 new phase) to the gdev-secure-devenv-bootstrap implementation plan. The amendments expand gdev from a security-focused dev environment bootstrapper into a comprehensive one-stop developer platform for a software engineering consulting organization.

All expansions fit within the existing 3-addon architecture (devenv, claudecode, devinit). No new addons are needed.

**Critical dependency**: devenv >= 2.0 is now an explicit minimum version requirement.

---

## Amendment Index

| Phase | Amendment | Units Added | Design Document |
|-------|-----------|-------------|-----------------|
| **Phase 2** | Cloud & K8s ecosystem modules | 9 units (2.9-2.17) | `cloud-k8s-module-design.md` |
| **Phase 3** | Service template expansion | 6 units (2.6-2.11) | `service-observability-design.md` |
| **Phase 4** | MCP server registry & auto-detect | 4 units (3.5.1-3.5.4) | `mcp-expansion-design.md` |
| **Phase 6** | Copier templates + client profile wizard | 10 units (5.7-5.12, 6.7-6.10) | `copier-integration-design.md`, `client-profile-design.md` |
| **Phase 7b** *(new)* | Non-language tool detection modules | 4 units (7b.1-7b.4) | `shell-tools-ide-design.md` |
| **Phase 8** | IDE config generation | 2 units (8.8-8.9) | `shell-tools-ide-design.md` |
| **Phase 10** | Shell/workstation configuration | 4 units (10.6-10.9) | `shell-tools-ide-design.md` |
| **Phase 12** | MCP lifecycle + observability sidecar | 8 units (12.8.1-12.8.5, 12.12-12.14) | `mcp-expansion-design.md`, `service-observability-design.md` |
| **Phase 13** | Client profile system | 4 units (13.8-13.11) | `client-profile-design.md` |

**Total: 51 new units across 9 phase amendments**

---

## Phase-by-Phase Details

### Phase 2 Amendment: Cloud & Kubernetes Ecosystem Modules

**Current scope**: 8 language ecosystem modules (JS/TS, Python, Go, Rust, Java/Kotlin, .NET, Docker, Terraform)

**Added scope**: 2 new module categories (cloud, kubernetes) with 9 units

| Unit | Title | Category |
|------|-------|----------|
| 2.9 | AWS Cloud Module | Cloud Tier 1 |
| 2.10 | GCP Cloud Module | Cloud Tier 1 |
| 2.11 | Azure Cloud Module | Cloud Tier 1 |
| 2.12 | Cloud Platform CLIs (Cloudflare/DO/Fly.io/Vercel/Netlify) | Cloud Tier 3 |
| 2.13 | Cloud Module Shared Infrastructure | Cloud shared |
| 2.14 | Kubernetes Core Module | K8s Tier 1 |
| 2.15 | Kubernetes Development Module | K8s Tier 2 |
| 2.16 | Kubernetes Security Module | K8s Tier 3 |
| 2.17 | Kubernetes Module Shared Infrastructure | K8s shared |

**Key design decisions**:
- Per-project env var isolation (`AWS_PROFILE`, `KUBECONFIG`, `CLOUDSDK_ACTIVE_CONFIG_NAME`) prevents cross-project credential leakage
- gdev installs tools and provides scaffolding but NEVER manages credentials
- Doctor checks have 5s timeouts and graceful degradation
- Cloud modules reuse the existing Terraform module's provider parser

**Dependencies**: Phase 1 (shared infrastructure, ecosystem module interface)

---

### Phase 3 Amendment: Service Template Expansion

**Current scope**: 6 services (PostgreSQL, Redis, MySQL/MariaDB, MongoDB, Elasticsearch, RabbitMQ)

**Added scope**: 5 new services + detection engine expansion with 6 units

| Unit | Title | Tier |
|------|-------|------|
| 2.6 | Kafka Service Sub-Template | Tier 1 (essential) |
| 2.7 | MinIO Service Sub-Template | Tier 2 (detect-and-offer) |
| 2.8 | Mailpit Service Sub-Template | Tier 2 |
| 2.9 | Keycloak Service Sub-Template | Tier 2 |
| 2.10 | NATS Service Sub-Template | Tier 2 |
| 2.11 | Service Detection Engine Expansion & Wizard Integration | Cross-cutting |

**Key design decisions**:
- Kafka uses KRaft mode (no Zookeeper dependency)
- MinIO generates full AWS S3 env vars for SDK compatibility
- Keycloak supports realm import from JSON
- Each service has multi-ecosystem detection heuristics
- Tier 2 services use detect-and-offer (wizard prompt, not auto-enabled)

**Dependencies**: Phases 1, 2 (same as existing Phase 3)

---

### Phase 4 Amendment: MCP Server Registry & Auto-Detect

**Current scope**: 5 hardcoded MCP servers in .mcp.json (Context7, GitHub, Socket.dev, semble, PostgreSQL)

**Added scope**: Registry-driven composition + 2 auto-detect + 2 detect-and-offer servers with 4 units

| Unit | Title |
|------|-------|
| 3.5.1 | MCP Server Registry and Tool Budget Tracking |
| 3.5.2 | Auto-Detect Database MCP Servers (MySQL, SQLite) |
| 3.5.3 | Detect-and-Offer MCP Servers (Terraform, Sentry) |
| 3.5.4 | .mcp.json Registry-Driven Composition |

**Key design decisions**:
- 40-tool ceiling enforced by `CanEnable()` budget check
- Three security tiers: auto-detect (low risk), detect-and-offer (medium), optional catalog (high)
- Registry replaces hardcoded server list — extensible for Phase 12 additions
- `--yes` flag does NOT auto-enable detect-and-offer servers

**Dependencies**: Phases 1, 2 (same as existing Phase 4)

---

### Phase 6 Amendment: Copier Templates + Client Profile Wizard

**Current scope**: huh wizard forms, quick/customize paths, detection pre-population, profile system

**Added scope**: Copier template integration (6 units) + client profile wizard flow (4 units)

| Unit | Title | Feature |
|------|-------|---------|
| 5.7 | Template Registry & Resolution | Copier |
| 5.8 | Copier Availability Gate & Invocation | Copier |
| 5.9 | `gdev init --from` Orchestration Flow | Copier |
| 5.10 | `gdev update --template` Flow | Copier |
| 5.11 | Non-Interactive & CI Mode for Template Workflows | Copier |
| 5.12 | Firm-Wide Template Standards Documentation | Copier |
| 6.7 | Age Key Management in gdev setup | Client profiles |
| 6.8 | Profile CRUD Commands | Client profiles |
| 6.9 | Init-Time Profile Selection in Wizard | Client profiles |
| 6.10 | Non-Interactive Profile Mode | Client profiles |

**Key design decisions**:
- `--from <template>` runs Copier first, then normal gdev init on generated project
- `--client-profile <name>` is distinct from existing `--profile` (project type)
- Profile YAML files are always sops+age encrypted at rest
- Profile CRUD wraps `sops edit` for atomic decrypt-edit-encrypt
- Wizard shows profile list between quick-path and language selection

**Dependencies**: Phases 3, 4 (existing), plus sops/age prerequisites

---

### Phase 7b (NEW): Non-Language Tool Detection Modules

**Rationale**: Phase 7 currently has 19 language modules. Adding tool detection modules would bloat it to 23+. Splitting into 7a (languages, existing) and 7b (tools, new) keeps phases focused. 7b can run in parallel with 7a.

| Unit | Title |
|------|-------|
| 7b.1 | Git Platform Detection Module |
| 7b.2 | Documentation Tools Detection Module |
| 7b.3 | API Tools Detection Module |
| 7b.4 | Database Migration Detection Module |

**Key design decisions**:
- Same `EcosystemModule` interface as language modules
- Detection heuristics: `.github/` → gh, `.proto` → grpcurl+buf, `mkdocs.yml` → mkdocs, `prisma/schema.prisma` → prisma-engines
- DB migration tools split: install CLI for native-dep tools (Flyway, Prisma, diesel-cli), document in CLAUDE.md for project-managed tools (Alembic, Drizzle)
- Each module can generate CLAUDE.md workflow sections

**Dependencies**: Phase 2 (ecosystem module interface)

---

### Phase 8 Amendment: IDE Config Generation

**Current scope**: `gdev init --update`, three-way merge, section markers, team standards versioning

**Added scope**: 2 new generated file types

| Unit | Title |
|------|-------|
| 8.8 | EditorConfig Generation |
| 8.9 | VS Code Extensions Recommendation Generation |

**Key design decisions**:
- `.editorconfig` is always generated by `gdev init` (universally safe, editor-agnostic)
- `.vscode/extensions.json` is opt-in via `gdev enable vscode`
- Both use existing atomic write pipeline and hash tracking
- Ecosystem-aware rules (Go: tabs, Python: 4-space, JS/TS: 2-space)
- Extension recommendations are ecosystem-detected (14 extension mappings)

**Dependencies**: Phases 3-6 (same as existing Phase 8)

---

### Phase 10 Amendment: Shell/Workstation Configuration

**Current scope**: GoReleaser builds, install scripts, packaging, self-update, shell completions

**Added scope**: Personal workstation configuration mode with 4 units

| Unit | Title |
|------|-------|
| 10.6 | Shell Fragment Directory & Init System |
| 10.7 | Modern Coreutils Installation via Nix Profile |
| 10.8 | Shell Aliases & Coreutils Configuration Fragments |
| 10.9 | Starship Prompt Configuration |

**Key design decisions**:
- `gdev setup --shell` manages `~/.qsdev/shell/` (personal, not per-project)
- Coreutils installed system-wide via `nix profile install` (not devenv.nix)
- Non-destructive: never modifies ~/.bashrc or ~/.zshrc — provides source instructions
- Fragment-based: per-concern files (aliases.sh, coreutils.sh, starship.toml)
- Supports bash, zsh, fish

**Dependencies**: Phases 1, 9 (same as existing Phase 10)

---

### Phase 12 Amendment: MCP Lifecycle + Observability Sidecar

**Current scope**: Tool lifecycle management, Semgrep, Gitleaks, Grype, Syft, Cosign, ScanCode, SecretSpec, Context7, git-cliff

**Added scope**: MCP lifecycle commands + optional catalog servers + observability sidecar with 8 units

| Unit | Title | Feature |
|------|-------|---------|
| 12.8.1 | MCP Lifecycle Commands | MCP |
| 12.8.2 | Ticketing MCP Servers (Atlassian, Linear) | MCP |
| 12.8.3 | Communication MCP Server (Slack) | MCP |
| 12.8.4 | Observability MCP Servers (Datadog, Grafana) | MCP |
| 12.8.5 | Security Documentation and Credential Hygiene | MCP |
| 12.12 | Observability Tool Registration & OTEL Env Var Generation | Observability |
| 12.13 | Observability Container Lifecycle Integration | Observability |
| 12.14 | Observability CLI Commands & Wizard Integration | Observability |

**Key design decisions**:
- `gdev mcp list` shows tool counts and budget remaining
- Slack MCP has mandatory security warning (cannot skip with `--yes`)
- Atlassian MCP is highest-value optional server for consulting
- Observability uses Docker `grafana/otel-lgtm` container, not native devenv services
- `gdev enable observability` / `gdev disable observability` lifecycle
- OTEL env vars auto-generated when observability enabled

**Dependencies**: Phases 1-8, 9, 11 (same as existing Phase 12)

---

### Phase 13 Amendment: Client Profile System

**Current scope**: `.gdev.yaml` project config, three-layer resolution, four onboarding modes, `gdev check` CI enforcement

**Added scope**: Client profile schema, SecretSpec integration, config propagation, compliance enforcement with 4 units

| Unit | Title |
|------|-------|
| 13.8 | Client Profile Schema & sops Encryption Layer |
| 13.9 | SecretSpec Integration & Generation |
| 13.10 | Baked Config Propagation to .gdev.yaml |
| 13.11 | Profile-Aware Compliance Enforcement in gdev check |

**Key design decisions**:
- `ClientProfile` Go struct versioned independently of `.gdev.yaml`
- Two-layer security: sops+age at rest, SecretSpec at runtime
- Non-secret values baked into `.gdev.yaml` `client` block
- Secret values generate `secretspec.toml` entries (gitignored)
- Profile compliance violations at critical severity are not suppressible
- Profile hash stored for change detection in Update mode

**Dependencies**: Phases 1, 6, 8, 12 (same as existing Phase 13)

---

## Plan-Level Changes

### Phase Index Update

The plan.md Phase Index table needs these updates:

| # | Phase | Change |
|---|-------|--------|
| 2 | Ecosystem Modules — Tier 1 | Add: "+ Cloud & Kubernetes modules" to Summary |
| 3 | devenv Addon — Core Generation | Add: "+ 5 additional service templates (Kafka, MinIO, Mailpit, Keycloak, NATS)" |
| 4 | Claude Code Addon — Core Generation | Add: "+ MCP server registry with tool budget tracking" |
| 6 | Wizard & Orchestration | Add: "+ Copier template integration, client profile wizard" |
| 7 | Ecosystem Modules — Tiers 2-4 | Rename to "7a". Add: "7b: Non-Language Tool Detection Modules" as new row |
| 8 | Migration, Update & Polish | Add: "+ .editorconfig and .vscode/extensions.json generation" |
| 10 | Distribution & Self-Bootstrapping | Add: "+ `gdev setup --shell` personal workstation configuration" |
| 12 | Extended Integrations & Lifecycle | Add: "+ MCP lifecycle commands, observability sidecar, optional MCP catalog" |
| 13 | Project Configuration & Team Standards | Add: "+ Client profile system (sops+age + SecretSpec)" |

### New Prerequisite: devenv >= 2.0

Add to plan.md Design Principles or a new Prerequisites section:

> **Minimum devenv version: 2.0** (released March 2026). The plan depends on devenv 2.0 features: DAG-based task system (parallel execution, caching, JSON data passing, process integration), `devcontainer.enable` native toggle, and namespace task convention. `gdev doctor` must check devenv version and warn if < 2.0.

### New Ecosystem Coverage Entries

Add to plan.md Ecosystem Coverage section:

**Cloud Provider CLIs** (new category):
- Tier 1: AWS CLI v2, Google Cloud SDK, Azure CLI
- Tier 3: Wrangler, doctl, flyctl, Vercel CLI, Netlify CLI

**Kubernetes Tools** (new category):
- Tier 1: kubectl, kubectx, k9s, stern, kustomize
- Tier 2: Skaffold, Tilt, DevSpace
- Tier 3: kubescape, kube-bench, kube-linter, polaris

**Development Services** (expand existing):
- Add Tier 1: Kafka
- Add Tier 2: MinIO, Mailpit, Keycloak, NATS

**MCP Servers** (expand existing table):

| Tool | Type | Integration |
|------|------|-------------|
| MySQL MCP | MCP server (auto-detect) | .mcp.json when MySQL service detected |
| SQLite MCP | MCP server (auto-detect) | .mcp.json when .sqlite/.db files found |
| Terraform MCP | MCP server (detect-and-offer) | .mcp.json when .tf files found |
| Sentry MCP | MCP server (detect-and-offer) | .mcp.json when Sentry SDK detected |
| Atlassian MCP | MCP server (optional) | .mcp.json via `gdev enable mcp-atlassian` |
| Linear MCP | MCP server (optional) | .mcp.json via `gdev enable mcp-linear` |
| Slack MCP | MCP server (optional) | .mcp.json via `gdev enable mcp-slack` |
| Datadog MCP | MCP server (optional) | .mcp.json via `gdev enable mcp-datadog` |
| Grafana MCP | MCP server (optional) | .mcp.json via `gdev enable mcp-grafana` |

### Research Foundation Update

Add to plan.md Research Foundation table:

| Spike / Report | Contribution |
|---|---|
| `research-spikes/gdev-ecosystem-expansion-assessment/research.md` | Ecosystem expansion assessment: 13 gap categories, 100+ tools cataloged, addon architecture fit, 51 implementation units |
| `research-spikes/gdev-ecosystem-expansion-assessment/cloud-k8s-tooling-research.md` | Cloud CLI/K8s tooling landscape, Nixpkgs verification, 4-tier recommendation framework, credential isolation patterns |
| `research-spikes/gdev-ecosystem-expansion-assessment/dev-services-observability-research.md` | 42 devenv.sh services cataloged, service tiering, observability sidecar design (grafana/otel-lgtm) |
| `research-spikes/gdev-ecosystem-expansion-assessment/consulting-daily-driver-research.md` | 100+ consulting engineer tools cataloged with commonality ratings and Nix availability |
| `research-spikes/gdev-ecosystem-expansion-assessment/api-db-mcp-research.md` | API tools, DB migration frameworks, MCP server ecosystem (17 candidates assessed) |
| `research-spikes/gdev-ecosystem-expansion-assessment/git-docs-ide-research.md` | Git CLIs, documentation tools, IDE config patterns, .editorconfig/.vscode design |
| `research-spikes/gdev-ecosystem-expansion-assessment/rejected-features-consulting-ops-research.md` | 13 rejected features re-evaluated, Copier reconsideration, client profiles, devenv 2.0 analysis |

---

## Validation Phase Impact

The existing validation phases (17-22) will need expanded test scenarios to cover:

- **Phase 18** (Cross-Platform): Cloud CLI installation across OS families, `gdev setup --shell` validation
- **Phase 19** (Ecosystem Onboarding): Cloud/K8s module detection, service template selection, Copier template flows
- **Phase 20** (Tool Lifecycle): MCP server enable/disable, observability sidecar lifecycle, expanded tool count (from ~17 to ~26 tools)
- **Phase 21** (Security Defense): sops+age profile encryption round-trips, SecretSpec credential resolution, MCP credential hygiene
- **Phase 22** (Agentic/Compliance/DX): Client profile wizard flows, profile compliance enforcement, `gdev setup --shell` verification

These are incremental additions to existing validation phases, not new phases.

---

## Net Impact Summary

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Implementation units | ~120 (est.) | ~171 | +51 |
| Phases | 22 | 23 (7 split to 7a/7b) | +1 |
| Language ecosystem modules | 27 | 27 | 0 |
| Cloud/K8s modules | 0 | 9 | +9 |
| Tool detection modules | 0 | 4 | +4 |
| Development services | 6 | 11 | +5 |
| MCP servers | 5 | 14 | +9 |
| Pre-commit hooks | 30+ | 30+ | 0 |
| gdev commands | ~16 | ~24 | +8 |
| Design documents | 0 | 6 | +6 |
