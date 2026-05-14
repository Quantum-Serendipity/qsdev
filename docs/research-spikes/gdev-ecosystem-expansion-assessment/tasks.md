# Tasks: gdev Ecosystem Expansion Assessment

## Phase 1: Scoping & Initial Research

### Pending

### Active

### Completed
- [x] **Coverage matrix & gap inventory** — Build comprehensive matrix of all tools/services across plan+spikes, categorized by type. Identify every gap category.
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 13 gap categories identified across cloud CLIs, K8s, services, API tools, DB migration, observability, git CLIs, documentation, IDE, MCP servers, consulting ops, runtime managers, code quality. Report: coverage-matrix-research.md

- [x] **Consulting engineer daily-driver audit** — Research standard dev tooling across active consulting orgs. What do engineers install day one?
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 12 categories, 100+ tools cataloged. 3 priority tiers. Report: consulting-daily-driver-research.md

- [x] **Cloud provider CLI & credential ecosystem** — Assess AWS/GCP/Azure/k8s CLI tools, SSO patterns, credential management for devenv integration
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 8 cloud CLIs, 4 credential helpers, 26 K8s tools. All in Nixpkgs. Report: cloud-k8s-tooling-research.md

- [x] **Development services expansion** — Compare devenv.sh's full native service catalog against consulting needs beyond the 6 planned
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 42 devenv.sh services cataloged. Kafka Tier 1, MinIO/Mailpit/Keycloak/NATS Tier 2. Report: dev-services-observability-research.md

- [x] **Kubernetes & container orchestration tools** — Evaluate kubectl, kustomize, Skaffold, Tilt, Lens, k9s for gdev integration
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 26 K8s tools across 6 categories. Report: cloud-k8s-tooling-research.md

- [x] **API dev & testing tool landscape** — Survey REST/GraphQL/gRPC development tools, clients, and documentation generators
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 15 tools surveyed, 10 recommended. Report: api-db-mcp-research.md §1

- [x] **Database migration & schema management** — Evaluate per-ecosystem migration frameworks and whether gdev should configure them
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 20+ tools. gdev detects and installs CLI binaries; doesn't choose tools. Report: api-db-mcp-research.md §2

- [x] **MCP server ecosystem expansion** — Catalog non-documentation MCP servers that enhance consulting workflows
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 17 candidates assessed. MySQL/SQLite auto-detect, Terraform/Sentry detect-and-offer, Atlassian highest-value optional. Report: api-db-mcp-research.md §3

- [x] **Observability local dev stack** — Assess whether gdev should provide local observability as devenv service templates
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: OTEL rejection was correct for infra ops, wrong for app dev. Recommend Docker sidecar via grafana/otel-lgtm. Report: dev-services-observability-research.md §4

- [x] **Rejected feature reconsideration** — Re-evaluate all 13 rejected features against "one stop shop" goal
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 8 confirmed, 1 full reconsideration (Copier), 3 partial (.editorconfig, .vscode, devcontainer, OTEL). Client profiles identified as top new feature. Report: rejected-features-consulting-ops-research.md

- [x] **Git platform CLIs, documentation tools & IDE configuration** — Assess git CLIs, doc/diagram tools, and IDE config patterns
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 30+ tools. gh highest-impact single addition. .editorconfig always-generate. .vscode/extensions.json opt-in. Report: git-docs-ide-research.md

- [x] **Addon architecture fit assessment** — For all identified expansions, determine if they fit devenv/claudecode/devinit or warrant new addons
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: All expansions fit existing 3-addon architecture. No fourth addon needed. Client profiles go in devinit. 10 phase amendments recommended. Report: addon-architecture-fit-research.md

- [x] **Phase 2 task decomposition** — Create focused sub-agent research tasks for each expansion category
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: 10 Phase 2 design tasks created covering implementation plan amendments.

## Phase 2: Design & Implementation Plan Amendments

### Pending

### Active

### Completed
- [x] **Cloud & K8s ecosystem module design** — 9 units (2.9-2.17): AWS/GCP/Azure/platform CLIs + K8s core/dev/security modules
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: cloud-k8s-module-design.md

- [x] **Client profile system design** — 8 units (6.7-6.10, 13.8-13.11): sops+age encryption, SecretSpec integration, wizard flow, compliance enforcement
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: client-profile-design.md

- [x] **Service template expansion + observability sidecar design** — 9 units (2.6-2.11, 12.12-12.14): Kafka/MinIO/Mailpit/Keycloak/NATS + grafana/otel-lgtm
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: service-observability-design.md

- [x] **MCP server expansion design** — 9 units (3.5.1-3.5.4, 12.8.1-12.8.5): registry, auto-detect DB, detect-and-offer, optional catalog, lifecycle, security
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: mcp-expansion-design.md

- [x] **Shell/workstation + tool detection + IDE + DB migration design** — 10 units (10.6-10.9, 8.8-8.9, 7b.1-7b.4): shell fragments, .editorconfig, .vscode, git/doc/API/DB modules
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: shell-tools-ide-design.md

- [x] **Copier template integration design** — 6 units (5.7-5.12): template registry, Copier runner, init/update flows, non-interactive mode, authoring spec
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: copier-integration-design.md

- [x] **Implementation plan amendment proposal** — Synthesized all 51 units into unified proposal with phase-by-phase details
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Report: implementation-plan-amendment-proposal.md

- [x] **Per-session context overlays & project clarity templates** — Research Claude Code env var support, Yaw Mode mechanism, devenv enterShell integration, and design CLAUDE.md project clarity section
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: Key finding: no CLAUDE_SKILLS_PATH exists. Yaw Mode uses undocumented CLAUDE_CONFIG_DIR. Recommended: static .claude/ generation (Phase 4) + enterShell env vars + future --add-dir integration. Designed Project Context section for CLAUDE.md with Copier integration. Report: context-overlays-clarity-research.md

- [x] **Learning Opportunities + Orient plugin evaluation** — Deep evaluation of DrCatHicks skill plugins for consulting skill-building
  - Outcome: success
  - Completed: 2026-05-14
  - Notes: SKILL.md format compatible, zero deps. Deploy via gdev enable. Orient strongest for Join mode onboarding. Exclude auto hook. Report: learning-opportunities-research.md

- [x] **mcph MCP orchestrator evaluation** — Evaluate Yaw Labs' mcph as potential replacement for custom MCP registry
  - Outcome: success (do not adopt)
  - Completed: 2026-05-14
  - Notes: Runtime proxy vs build-time generator — fundamentally different architecture. Cherry-pick mcp-compliance (88 tests, MIT) for Phase 17. Report: mcph-orchestrator-research.md

## Phase 3: Synthesis & Spike Completion

### Completed
- [x] **Depth checklist review** — All 6 items pass. See research.md Depth Checklist section.
  - Outcome: success
  - Completed: 2026-05-14

- [x] **Final conclusions** — Written in research.md. Spike complete.
  - Outcome: success
  - Completed: 2026-05-14
