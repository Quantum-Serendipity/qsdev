# Research Log: gdev Ecosystem Expansion Assessment

## 2026-05-14 16:00 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized to perform a thorough evaluation of the gdev implementation plan, all associated gdev research spikes, and the broader developer tooling ecosystem. Goal: identify tools, modules, methodologies, and capabilities that naturally fit into and enhance gdev's value proposition as a one-stop developer platform for a software engineering consulting org. Assess whether additions fit within the 3 existing addons (devenv, claudecode, devinit) or warrant new addons.
- **Next**: Read all gdev research spikes to understand current coverage, then identify gaps and expansion opportunities. Create Phase 1 tasks.

## 2026-05-14 17:30 — Development Services & Observability Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: [devenv.sh services catalog](https://devenv.sh/services/) → `docs/devenv-sh-services-catalog.md`, [OTEL Collector](https://devenv.sh/services/opentelemetry-collector/) → `docs/devenv-otel-collector-config.md`, [Prometheus](https://devenv.sh/services/prometheus/) → `docs/devenv-prometheus-config.md`, [Kafka](https://devenv.sh/services/kafka/) → `docs/devenv-kafka-config.md`, [Vault](https://devenv.sh/services/vault/) → `docs/devenv-vault-config.md`, [MinIO](https://devenv.sh/services/minio/) → `docs/devenv-minio-config.md`, [Keycloak](https://devenv.sh/services/keycloak/) → `docs/devenv-keycloak-config.md`, [NATS](https://devenv.sh/services/nats/) → `docs/devenv-nats-config.md`, [Mailpit](https://devenv.sh/services/mailpit/) → `docs/devenv-mailpit-config.md`, [Search services](https://devenv.sh/services/meilisearch/) → `docs/devenv-search-services-config.md`, [WireMock](https://devenv.sh/services/wiremock/) → `docs/devenv-wiremock-config.md`, [grafana/otel-lgtm](https://github.com/grafana/docker-otel-lgtm) → `docs/grafana-docker-otel-lgtm.md`
- **Summary**: Cataloged all 42 native devenv.sh services. Assessed 7 categories of expansion candidates. Key finding: the original OTEL rejection conflated two distinct use cases — Claude Code session monitoring (correctly rejected as infra ops) vs application development observability (should be reconsidered as Docker sidecar via grafana/otel-lgtm). Recommendation: add Kafka to Tier 1 essential services, add MinIO/Mailpit/Keycloak/NATS to Tier 2, offer `qsdev enable observability` as Docker-based sidecar. Pulsar/Consul/LocalStack/Memcached/Typesense all skip.
- **Next**: Complete remaining spike tasks.

## 2026-05-14 — Consulting Engineer Daily-Driver Audit Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: 20+ web searches, 4 full-page fetches. See `docs/web-search-sources-index.md` for complete index.
  - ThoughtWorks Technology Radar Vol 34 → `docs/thoughtworks-technology-radar-vol34-tools.md`
  - devenv.sh services/languages reference → `docs/devenv-sh-services-languages-reference.md`
  - 13 CLI tools for developers 2025 → `docs/13-cli-tools-developer-2025.md`
  - AWS tools for consultant work → `docs/aws-tools-consultant-work.md`
- **Summary**: Comprehensive audit of 12 categories of developer tooling needed by consulting engineers on day one. Cataloged 100+ tools with commonality ratings, Nix availability, configuration requirements, and devenv.sh native support. Key findings: (1) Cloud provider CLIs + credential management is the biggest gap — used on virtually every engagement. (2) "Modern coreutils" (ripgrep, fd, bat, fzf, jq, yq, delta) are expected by senior engineers and trivial to add. (3) kubectl + k9s are standard for cloud-native work. (4) gh (GitHub CLI) is essential. (5) sops+age for secrets-in-git is standard practice. (6) Time tracking CLIs don't exist meaningfully — web/app workflow. (7) IDE configuration should remain rejected scope. (8) Almost everything is Nix-packaged; gdev's value is curation+detection+configuration, not installation.
- **Next**: Feed findings into addon architecture fit assessment. Determine what goes in devenv addon vs new addon(s).

## 2026-05-14 18:15 — Cloud Provider CLI & Kubernetes Tooling Deep Research
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Ship Your Toolchain (maxdaten.io)](https://www.maxdaten.io/2026-01-31-ship-your-toolchain-not-just-infrastructure) → `docs/ship-your-toolchain-devenv.md`
  - [aws-vault USAGE.md](https://github.com/99designs/aws-vault/blob/master/USAGE.md) → `docs/aws-vault-usage.md`
  - [devenv-k8s module](https://github.com/LCOGT/devenv-k8s) → `docs/devenv-k8s-reusable-module.md`
  - [Skaffold vs Tilt vs DevSpace](https://www.vcluster.com/blog/skaffold-vs-tilt-vs-devspace) → `docs/skaffold-vs-tilt-vs-devspace.md`
  - [K8s Security Tools 2026 (ARMO)](https://www.armosec.io/blog/best-open-source-kubernetes-security-tools/) → `docs/k8s-security-tools-2026.md`
  - [devenv.sh Helm module](https://devenv.sh/languages/helm/) → `docs/devenv-helm-module.md`
  - [Stern/Kubetail](https://github.com/stern/stern) → `docs/stern-log-tailing.md`
  - [AWS Credential Helpers](https://github.com/99designs/aws-vault) → `docs/aws-credential-helpers-comparison.md`
  - [GCP ADC & Workload Identity](https://docs.cloud.google.com/docs/authentication/application-default-credentials) → `docs/gcp-adc-workload-identity.md`
- **Summary**: Completed comprehensive research on cloud provider CLIs (AWS, GCP, Azure + 5 platform CLIs), credential management tools (aws-vault, saml2aws, aws-sso-cli, Leapp), Kubernetes tools (kubectl + 25 related tools), and devenv.sh integration patterns. Verified all 40+ tool packages in Nixpkgs with exact names and versions via `nix eval`. Developed 4-tier recommendation framework, detection heuristics for auto-installation, and consulting-specific multi-client credential isolation patterns. Key finding: every major tool is in Nixpkgs; the hard problem is credential management and multi-client isolation, not installation. Report: `cloud-k8s-tooling-research.md`.
- **Next**: Complete remaining expansion category tasks (API dev, database migration, MCP servers, etc.). Feed into addon architecture fit assessment.

## 2026-05-14 19:30 — Rejected Features Reconsideration & Consulting Ops Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [devenv tasks documentation](https://devenv.sh/tasks/) → `docs/devenv-tasks-documentation.md`
  - [devenv 2.0 release notes](https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/) → `docs/devenv-2.0-release-notes.md`
  - [devenv OTEL Collector service](https://devenv.sh/services/opentelemetry-collector/) → `docs/devenv-otel-collector-service.md`
  - [mise feature overview](https://mise.jdx.dev/) → `docs/mise-feature-overview.md`
  - [devenv devcontainer integration](https://devenv.sh/integrations/codespaces-devcontainer/) → `docs/devenv-devcontainer-integration.md`
  - [Copier comparison to alternatives](https://copier.readthedocs.io/en/stable/comparisons/) → `docs/copier-comparison-to-alternatives.md`
  - Time tracking CLI survey → `docs/time-tracking-cli-tools.md`
  - Client isolation patterns → `docs/client-isolation-credential-switching.md`
  - Release automation comparison → `docs/release-automation-comparison.md`
  - 10+ web searches on competing tools, consulting patterns, runtime version managers
- **Summary**: Comprehensive re-evaluation of all 13 rejected features against "one stop shop" goal. Result: 8 confirmed rejections, 1 full reconsideration (code scaffolding via Copier for firm templates), 3 partial reconsiderations (devcontainer generation, .editorconfig/.vscode/extensions.json, OTEL service template). Task runner rejection strengthened by devenv 2.0's expanded capabilities. Identified "client profiles" (`qsdev switch <client>`) as the single most impactful consulting-specific feature — bundles AWS profile + git identity + SSH key + env vars + time tracking into switchable profiles. Runtime version managers (mise, asdf, nvm) are fully redundant with devenv and should not be integrated. Report: `rejected-features-consulting-ops-research.md`.
- **Next**: Update tasks.md. Feed findings into addon architecture fit assessment.

## 2026-05-14 20:00 — API Dev, DB Migration & MCP Server Ecosystem Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [MCP Official Servers (GitHub)](https://github.com/modelcontextprotocol/servers) -> `docs/mcp-official-servers-github.md`
  - [Best MCP Servers 2026 (MCPBundles)](https://www.mcpbundles.com/blog/best-mcp-servers) -> `docs/mcp-bundles-best-servers-2026.md`
  - [18 DevOps MCP Servers (Medium/k8slens)](https://medium.com/k8slens/18-best-devops-mcp-servers-for-2026-the-definitive-guide-bfde04654a35) -> `docs/devops-mcp-servers-2026-medium.md`
  - [Terraform MCP Server (HashiCorp)](https://developer.hashicorp.com/terraform/mcp-server) -> `docs/terraform-mcp-server-hashicorp.md`
  - [15 MCP Servers for Claude Code (Codersera)](https://codersera.com/blog/best-mcp-servers-claude-code-cursor-2026/) -> `docs/mcp-servers-claude-code-cursor-2026.md`
  - [DB Migration Tools Comparison (Codelit)](https://codelit.io/blog/database-migration-tools-comparison) -> `docs/db-migration-tools-comparison-codelit.md`
  - [Postman Alternatives (Better Stack)](https://betterstack.com/community/comparisons/postman-alternative/) -> `docs/postman-alternatives-betterstack-2026.md`
  - [Slack MCP (Official)](https://slack.com/help/articles/48855576908307) -> `docs/slack-mcp-server-official.md`
  - [Linear MCP (Official)](https://linear.app/docs/mcp) -> `docs/linear-mcp-server-official.md`
  - [Datadog MCP (Official)](https://www.datadoghq.com/blog/datadog-mcp-server-use-cases/) -> `docs/datadog-mcp-server-use-cases.md`
  - [Atlassian MCP Server GA](https://www.mindstudio.ai/blog/atlassian-mcp-server-ga-claude-reads-writes-jira-confluence-compass-oauth)
  - Nixpkgs availability verified via nix eval/search for 30+ packages
- **Summary**: Completed deep research across three expansion categories. API tools: identified 15 tools across HTTP clients, GraphQL, gRPC/protobuf, OpenAPI, and testing frameworks; recommended 10 for gdev detect-and-offer based on project signals (.proto, openapi.yaml, .bru files). Key picks: httpie (ergonomic curl), bruno (Git-native Postman replacement), grpcurl+buf (protobuf toolchain), openapi-generator+redocly (OpenAPI), k6 (load testing). DB migrations: surveyed 20+ tools across 8 language ecosystems; key finding is gdev should detect and install CLI binaries (for tools with native deps like Rust/JVM) and document in CLAUDE.md (for tools installed via project package managers like npm/pip). MCP servers: assessed 17 candidate servers beyond current 5; recommended MySQL MCP and SQLite MCP as auto-detected additions, Terraform and Sentry as detect-and-offer, and Atlassian (highest-value addon for consulting)/Linear/Slack/Datadog/Grafana/GitLab/AWS/Azure as optional addons. Critical constraint: 40-tool ceiling means never more than ~6 servers simultaneously. Three-tier security model: low-risk (DB read-only) auto-configure, medium-risk (ticketing/observability) opt-in, high-risk (cloud/comms) explicit only. Report: `api-db-mcp-research.md`.
- **Next**: Feed findings into addon architecture fit assessment. Update research.md with topic summaries.

## 2026-05-14 20:30 — Git Platform CLIs, Documentation Tools, and IDE Configuration Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Mise IDE integration](https://mise.jdx.dev/ide-integration.html) → `docs/mise-ide-integration.md`
  - [Text-to-diagram tools comparison](https://text-to-diagram.com/) → `docs/text-to-diagram-tools-comparison.md`
  - [VS Code workspace settings (Atomic Object)](https://spin.atomicobject.com/vscode-workspace-settings/) → `docs/vscode-workspace-settings-team-sharing.md`
  - [Essential git tools (dev.to)](https://dev.to/vaib/top-17-essential-git-tools-for-enhanced-developer-productivity-7f3) → `docs/essential-git-tools-devto.md`
  - [Devcontainers in 2025 (Ivan Lee)](https://ivanlee.me/devcontainers-in-2025-a-personal-take/) → `docs/devcontainers-2025-assessment.md`
  - [devenv.sh VS Code support](https://devenv.sh/editor-support/vscode/) → `docs/devenv-sh-vscode-integration.md`
  - [EditorConfig global standard](https://medium.com/@siva_bankapalli) → `docs/editorconfig-global-standard.md`
  - [VS Code recommended extensions guide](https://leonardofaria.net/2023/02/10/using-recommended-extensions-and-settings-in-vs-code) → `docs/vscode-recommended-extensions-guide.md`
  - [git-crypt nixpkgs](https://mynixos.com/nixpkgs/package/git-crypt), [d2 nixpkgs](https://raw.githubusercontent.com/NixOS/nixpkgs/master/pkgs/by-name/d2/d2/package.nix), [delta nixpkgs](https://mynixos.com/nixpkgs/package/delta), [markdownlint-cli nixpkgs](https://mynixos.com/nixpkgs/package/markdownlint-cli)
  - DX polish spike `what-not-to-include-research.md` — original IDE rejection rationale
  - 12+ web searches on git CLIs, TUI tools, diagramming, ADR tools, markdown linters, IDE config patterns, devcontainer patterns, extension packs, EditorConfig
- **Summary**: Assessed 30+ tools across three gap categories (G: Git Platform CLIs, H: Documentation/Diagramming, I: IDE Configuration). Key findings: (1) gh is the highest-impact single addition — essential for GitHub-hosted projects, complementary to GitHub MCP server. (2) git-lfs should auto-install when .gitattributes filter=lfs detected. (3) Documentation tools fit naturally into ecosystem detection (mkdocs.yml → mkdocs package). (4) The IDE config rejection should be narrowed, not reversed: .editorconfig is universally safe (always generate), .vscode/extensions.json is low-risk (opt-in via `qsdev enable vscode`). No major dev environment tool auto-generates VS Code config, but devcontainers do. (5) LSP servers should be included in devenv.nix alongside language runtimes. (6) delta is the strongest git productivity tool candidate — transforms every diff with 3 lines of gitconfig. Report: `git-docs-ide-research.md`.
- **Next**: Complete remaining spike tasks. Feed all findings into addon architecture fit assessment.

## 2026-05-14 21:00 — Phase 1 Complete: Addon Architecture Fit Assessment & Phase 2 Decomposition
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all 6 sub-agent research reports (consulting daily-driver, cloud/K8s, dev services, API/DB/MCP, git/docs/IDE, rejected features) into addon architecture fit assessment and Phase 2 task plan. Key finding: **all expansions fit within the existing 3-addon architecture** — no fourth addon needed. Client profiles (`qsdev switch <client>`) go in devinit as orchestration. Mapped every expansion to specific implementation plan phase amendments (10 phase amendments across Phases 1-14). Created 10 Phase 2 design tasks for translating research into implementation plan updates.
- **Next**: Phase 2 design work — produce implementation units for each expansion category, then synthesize into unified implementation plan amendment proposal.

## 2026-05-14 21:30 — Design Corrections from User Review
- **Type**: decision
- **Status**: success
- **Depth**: moderate
- **Summary**: Three corrections applied based on user feedback:
  1. **Client profiles**: Redesigned from runtime switching (`qsdev switch <client>`) to init-time profile selection. Profiles in `~/.qsdev/clients/` are selected during `qsdev init` and baked into project config. No runtime context switching — devenv already isolates per-project.
  2. **Modern coreutils**: Removed from devenv.nix always-install bundle. These are personal shell tools, not per-project concerns. Redesigned as `qsdev setup` personal shell configuration mode managing `~/.qsdev/shell/` dotfile fragments.
  3. **devenv 2.0 minimum**: Documented as explicit minimum version requirement. Task runner rejection depends on 2.0's DAG tasks, caching, JSON data passing, process integration. `qsdev doctor` should check devenv version.
- **Next**: Phase 2 design work.

## 2026-05-14 23:00 — Phase 2 Complete: All Design Tasks + Amendment Proposal
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Completed all 7 Phase 2 design tasks via 6 parallel sub-agents. Produced 51 implementation units across 6 design documents: cloud-k8s-module-design.md (9 units), client-profile-design.md (8 units), service-observability-design.md (9 units), mcp-expansion-design.md (9 units), shell-tools-ide-design.md (10 units), copier-integration-design.md (6 units). Synthesized into unified implementation-plan-amendment-proposal.md mapping all units to 9 phase amendments (including new Phase 7b). Net impact: +51 units, +1 phase, +9 cloud/K8s modules, +4 tool detection modules, +5 services, +9 MCP servers, +8 gdev commands.
- **Next**: Phase 3 synthesis and review — apply depth checklist to all deliverables, fill gaps, write final conclusions.

## 2026-05-14 23:30 — External Ideas Analysis (Yaw Labs, Learning Opportunities, Fight Slop with Clarity)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Yaw Labs terminal/context article](https://www.siliconsnark.com/yaw-labs-built-a-terminal-startup-for-people-who-treat-context-like-ammunition/) → `docs/yaw-labs-terminal-context-ammunition.md`
  - [DrCatHicks/learning-opportunities](https://github.com/DrCatHicks/learning-opportunities) → `docs/learning-opportunities-drcathicks.md`
  - [Fight Slop with Clarity](https://blog.vtemian.com/post/fight-slop-with-clarity/) → `docs/fight-slop-with-clarity-vtemian.md`
- **Summary**: Analyzed three external sources for gdev-relevant ideas. Strongest candidates: (1) **Learning Opportunities skill plugin** (DrCatHicks) — research-backed Claude Code skill for developer skill-building, directly deployable in gdev's skill library, addresses consulting firm's interest in genuine engineer understanding. (2) **Orient codebase orientation plugin** — generates orientation lessons for unfamiliar codebases, maps to Join mode onboarding. (3) **Per-session context overlays** (Yaw Labs) — enterShell-driven Claude Code context that auto-shifts per project. (4) **Project clarity questions** (vtemian) — CLAUDE.md template addition forcing articulation of project purpose. Also identified `mcph` MCP orchestrator as investigate-before-building candidate for Phase 4 MCP registry, and five learning science risks as design constraints for Phase 14 agent prompts. The "tarpit test" validates gdev's existing rejection philosophy.
- **Next**: Add research topics to tasks.md for deeper investigation of mcph and learning-opportunities.

## 2026-05-14 22:00 — Service Template Expansion & Observability Sidecar Design Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Designed 9 implementation units across two plan amendments. Phase 3 amendment: 6 units — Kafka (Tier 1), MinIO, Mailpit, Keycloak, NATS (all Tier 2), plus a cross-cutting service detection engine/wizard integration unit. Each service unit includes concrete devenv.nix template code, default config values, environment variables, detection heuristics, and wizard integration. Phase 12 amendment: 3 units — observability tool registration with OTEL env vars + Docker scripts (12.12), container lifecycle integration with enterShell auto-start (12.13), and CLI commands + wizard integration (12.14). All units grounded in Phase 1 research findings and devenv.sh service documentation.
- **Next**: Complete remaining Phase 2 design tasks (cloud/K8s modules, client profiles, MCP expansion, shell/workstation tools, Copier integration). Then synthesize all designs into unified amendment proposal.

## 2026-05-14 — Learning Opportunities / Orient / Auto Deep Evaluation
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [DrCatHicks/learning-opportunities repo](https://github.com/DrCatHicks/learning-opportunities) — full file tree + 10 files fetched via gh api
  - learning-opportunities/skills/learning-opportunities/SKILL.md → `docs/learning-opportunities-skill-md.md`
  - orient/skills/orient/SKILL.md → `docs/orient-skill-md.md`
  - learning-opportunities-auto/hooks/* → `docs/learning-opportunities-auto-hook.md`
  - learning-opportunities/docs/MEASURE-THIS.md → `docs/measure-this-playbook.md`
  - CLAUDE.md, README.md, marketplace.json, plugin.json files (all 3 plugins)
  - CHANGELOG.md, orient-bibliography.md, PRINCIPLES.md
  - [DrCatHicks/learning-goal repo](https://github.com/DrCatHicks/learning-goal) — metadata (153 stars, CC-BY-4.0)
  - Dr. Cat Hicks credentials via web search (PhD UC San Diego, IEEE Software, Routledge book 2026)
- **Summary**: Deep evaluation of all three plugins for gdev integration. SKILL.md files use standard Claude Code YAML frontmatter format — directly compatible with gdev's embedded skill library and `deploySkills()`. No runtime dependencies (pure markdown). learning-opportunities (1,530 stars, 50 forks, CC-BY-4.0) is a well-constructed instructional skill with 6 exercise types and a 2-exercise-per-session cap. orient generates structured codebase orientation files using program comprehension research. learning-opportunities-auto is a PostToolUse bash hook that fires on git commit — should be excluded from gdev due to hook architecture conflicts and redundancy with the skill's autonomous behavior. MEASURE-THIS.md provides validated survey instruments (n=3,267) under CC-BY-SA 4.0 — reference material, not a deployable tool. Research foundation is highly credible (peer-reviewed, large samples, established learning science). Recommend: include learning-opportunities and orient as opt-in skills via `qsdev enable`, exclude auto hook, reference measurement framework in consulting documentation.
- **Next**: Report written to learning-opportunities-research.md. Update research.md and tasks.md.

## 2026-05-14 — mcph MCP Orchestrator Evaluation (Yaw Labs / mcp.hosting)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [YawLabs GitHub org](https://github.com/YawLabs) -> `docs/yawlabs-github-org-repos.md`
  - [mcph README](https://raw.githubusercontent.com/YawLabs/mcph/main/README.md) -> `docs/mcph-readme-full.md`
  - [mcph ROADMAP.md](https://raw.githubusercontent.com/YawLabs/mcph/main/ROADMAP.md) -> `docs/mcph-roadmap.md`
  - [mcph package.json](https://raw.githubusercontent.com/YawLabs/mcph/main/package.json) -> `docs/mcph-package-json.md`
  - [mcph types.ts](https://raw.githubusercontent.com/YawLabs/mcph/main/src/types.ts) -> `docs/mcph-types-ts.md`
  - [mcph meta-tools.ts](https://raw.githubusercontent.com/YawLabs/mcph/main/src/meta-tools.ts) -> `docs/mcph-meta-tools-summary.md`
  - [mcph config schema](https://raw.githubusercontent.com/YawLabs/mcph/main/schemas/mcph.config.v1.json) -> `docs/mcph-config-schema-v1.md`
  - [mcph upstream.ts](https://raw.githubusercontent.com/YawLabs/mcph/main/src/upstream.ts) -> `docs/mcph-upstream-architecture.md`
  - [mcph credentials.ts](https://raw.githubusercontent.com/YawLabs/mcph/main/src/credentials.ts) -> `docs/mcph-credentials-handling.md`
  - [mcp-compliance README](https://github.com/YawLabs/mcp-compliance) -> `docs/yawlabs-mcp-compliance-readme.md`
  - [aws-mcp README](https://raw.githubusercontent.com/YawLabs/aws-mcp/main/README.md) -> `docs/yawlabs-aws-mcp-readme.md`
  - [tailscale-mcp README](https://raw.githubusercontent.com/YawLabs/tailscale-mcp/main/README.md) -> `docs/yawlabs-tailscale-mcp-readme.md`
  - [mcp-hosting-deploy README](https://github.com/YawLabs/mcp-hosting-deploy) -> `docs/mcp-hosting-deploy-readme.md`
- **Summary**: Comprehensive evaluation of mcph (v0.47.5, TypeScript, 1 star, 1 contributor, 5 weeks old, no license). mcph is a runtime MCP proxy that sits between AI clients and upstream servers, handling dynamic routing, health monitoring, and cloud-synced configuration via mcp.hosting. It solves a fundamentally different problem than gdev's Unit 3.5.1 McpServerDef registry: mcph is a runtime orchestrator; gdev is a build-time code generator producing static .mcp.json. Adoption would require replacing gdev's entire MCP config pipeline with a cloud-dependent runtime proxy from a pre-seed startup. **Verdict: Do not adopt mcph.** Cherry-pick the mcp-compliance test suite (88 tests, 8 categories, MIT licensed) for Phase 17 test infrastructure, and evaluate aws-mcp (24 tools, SSO re-auth) as an optional catalog entry. Full report: `mcph-orchestrator-research.md`.
- **Next**: None -- evaluation complete. Findings feed into Phase 4 and Phase 17 implementation decisions.

## 2026-05-14 24:00 — Spike Complete: Final Synthesis & Depth Checklist
- **Type**: decision
- **Status**: success
- **Depth**: deep
- **Summary**: Spike complete. Three phases delivered: Phase 1 (14 scoping/research tasks, 7 research reports, 13 gap categories identified), Phase 2 (7 design tasks, 6 design documents, 51 implementation units, unified amendment proposal), Phase 3 (3 external research tasks, depth checklist review, final conclusions). Total deliverables: 22 research/design files, 81 source documents in docs/. The spike produced a complete implementation plan amendment proposal adding 51 units across 9 phase amendments. Key outcomes: all expansions fit existing 3-addon architecture, cloud CLIs and client profiles are highest-priority expansions, devenv >= 2.0 is an explicit minimum requirement, learning-opportunities and orient are ready-to-deploy consulting skill plugins, mcph is rejected but its compliance suite is cherry-pickable, and the tarpit test is formalized as a design principle.
- **Next**: Apply amendments to the implementation plan files. Start implementation.

## 2026-05-14 — Per-Session Context Overlays & Project Clarity Templates Research
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Claude Code Environment Variables (official)](https://code.claude.com/docs/en/env-vars) → `docs/claude-code-environment-variables-official.md`
  - [Claude Code Settings (official)](https://code.claude.com/docs/en/settings) → `docs/claude-code-settings-configuration-official.md`
  - [Yaw Mode Technical Implementation](https://yaw.sh/blog/claude-code-yaw-mode) → `docs/yaw-mode-technical-implementation.md`
  - [direnv Integration with Claude Code](https://github.com/anthropics/claude-code/issues/42229) → `docs/claude-code-direnv-integration-hooks.md`
  - [CLAUDE_CONFIG_DIR Feature Request](https://github.com/anthropics/claude-code/issues/25762) → `docs/claude-config-dir-feature-request.md`
  - [--add-dir CLAUDE.md Loading](https://github.com/anthropics/claude-code/issues/21138) → `docs/claude-code-add-dir-claude-md-loading.md`
  - [Additional Directories Settings Request](https://github.com/anthropics/claude-code/issues/3146) → `docs/claude-code-additional-directories-settings-request.md`
  - [CLAUDE.md 2026 Architecture](https://www.obviousworks.ch) → `docs/claude-md-2026-architecture-obviousworks.md`
- **Summary**: Researched two ideas from external analysis: (A) per-session Claude Code context overlays inspired by Yaw Mode, and (B) project clarity questions for CLAUDE.md templates. Key findings: (1) Claude Code has NO env vars for skills/rules paths — the original `CLAUDE_SKILLS_PATH` hypothesis is invalid. (2) Yaw Mode uses undocumented `CLAUDE_CONFIG_DIR` with temp directory overlay, hardlinks, and symlinks — technically clever but fragile and not recommended for gdev v1. (3) devenv enterShell can set `CLAUDE_CODE_EFFORT_LEVEL` and `GDEV_*` metadata vars that Claude Code inherits from the parent shell. (4) `--add-dir` with `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` is a promising future mechanism for shared organization context. (5) Designed a Project Context section for CLAUDE.md with clarity questions, consulting-specific fields, Copier integration, and the tarpit test as a design principle. Three implementation strategies designed, with Strategy 1 (existing Phase 4 static generation) recommended as primary.
- **Next**: Report written to context-overlays-clarity-research.md. Summary added to research.md.

## 2026-05-14 21:45 — Client Profile Security Design
- **Type**: decision
- **Status**: success
- **Depth**: moderate
- **Summary**: User raised that client profiles contain sensitive data and should be encrypted. Designed two-layer security approach using tools already in the plan: (1) sops+age for encryption at rest — profile YAML files in `~/.qsdev/clients/` are sops-encrypted, decrypted only during `qsdev init` via age key in `~/.qsdev/keys/`. (2) SecretSpec for runtime secret resolution — sensitive values in profiles reference SecretSpec providers (keyring, 1Password, dotenv, env) rather than containing plaintext. During `qsdev init --profile <client>`, non-secret values are baked into project config; secret values generate `secretspec.toml` entries for devenv runtime resolution. No plaintext secrets ever written to project files. Both sops+age and SecretSpec were already identified in prior research — sops+age as standard consulting practice, SecretSpec as devenv 2.0 native.
- **Next**: Phase 2 design work.
