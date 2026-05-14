# Research Summary: gdev Ecosystem Expansion Assessment

## Overview

Comprehensive evaluation of the gdev implementation plan (22 phases, 3 addons: devenv, claudecode, devinit) and all associated gdev research spikes, with the goal of identifying tools, modules, methodologies, and developer utilities that naturally enhance gdev's capabilities as a one-stop developer platform for a software engineering consulting organization. The assessment covers:

1. **Gap analysis** — What's missing from the current plan that active consulting engineers need daily?
2. **Ecosystem fit** — Which existing tools, services, and integrations should gdev provide out of the box?
3. **Addon architecture** — Do additions fit within devenv/claudecode/devinit, or do they warrant new addons?
4. **Developer tooling** — IDEs, toolchains, cloud CLI tools, services, and applications commonly used in consulting engagements

## Topics

### Consulting Engineer Daily-Driver Tooling Audit
- **Status**: Complete
- **Report**: [consulting-daily-driver-research.md](consulting-daily-driver-research.md)
- **Summary**: Comprehensive audit of 12 categories of developer tooling across consulting firms (Thoughtworks, Slalom, Accenture, EPAM, Cognizant). Cataloged 100+ tools with commonality ratings, Nix packaging status, configuration requirements, and devenv.sh native support. The largest gaps in gdev's current coverage are cloud provider CLIs with credential management (aws-cli/gcloud/az plus aws-vault/saml2aws), developer productivity CLI tools (the "modern coreutils" bundle: ripgrep, fd, bat, fzf, jq, yq, delta), Kubernetes operational tools (kubectl, k9s, kubectx), and git platform CLIs (gh/glab). Nearly all identified tools are Nix-packaged; gdev's primary value-add is curation, context-aware detection, and correct configuration rather than raw installation. Three priority tiers were defined: Tier 1 (install by default), Tier 2 (install when detected/opted-in), and Tier 3 (available but not default).

### Development Services Expansion & Local Observability Stack
- **Status**: Complete
- **Report**: [dev-services-observability-research.md](dev-services-observability-research.md)
- **Summary**: Cataloged all 42 native devenv.sh services against the 6 currently planned for gdev. devenv.sh covers databases (12), message brokers (5), search (3), caching (2), infrastructure (5), observability (2), email/API testing (3), web servers (5), and more. Kafka should be elevated to Tier 1 (essential) alongside the existing 6 services due to its dominance in event-driven architectures and excellent KRaft-mode devenv support. MinIO (S3 emulation), Mailpit (SMTP testing), Keycloak (identity), and NATS (lightweight messaging) form a strongly-recommended Tier 2 expansion. The original rejection of "OTEL infrastructure" conflated two distinct use cases: Claude Code session monitoring (correctly rejected as infrastructure ops) and application development observability (a legitimate dev-time need analogous to running PostgreSQL locally). Recommend offering `gdev enable observability` as a Docker-based sidecar using grafana/otel-lgtm rather than as a devenv.nix service template, since Grafana/Loki/Tempo are not native devenv services.

### Cloud Provider CLIs, Credential Management & Kubernetes Tooling
- **Status**: Complete
- **Report**: [cloud-k8s-tooling-research.md](cloud-k8s-tooling-research.md)
- **Summary**: gdev's largest coverage gap is cloud provider CLIs and Kubernetes tooling -- zero current coverage despite these being daily-use tools for consulting engineers. Research covered 8 cloud CLIs (AWS CLI v2, Google Cloud SDK, Azure CLI, Wrangler, doctl, flyctl, Vercel, Netlify), 4 AWS credential helpers (aws-vault, saml2aws, aws-sso-cli, Leapp), and 26 Kubernetes tools across 6 subcategories (core, context switching, development workflows, observability, security scanning, service mesh). Every major tool was verified present in Nixpkgs with exact package names and current versions. The key finding is that installation is trivially solved by devenv.sh/Nix -- the hard problem is credential management and multi-client isolation for consulting engineers who switch between client cloud accounts and K8s clusters daily. Recommended architecture: two new gdev devenv module categories (`cloud` and `kubernetes`), a 4-tier installation framework driven by project file detection heuristics, per-project environment variable isolation (`AWS_PROFILE`, `KUBECONFIG`, `CLOUDSDK_ACTIVE_CONFIG_NAME`), and `gdev doctor` checks for auth status. gdev should install tools and provide scaffolding but should NOT manage credentials directly (security risk, per-client variation).

### Rejected Features Reconsideration, Consulting Operations & Runtime Version Management
- **Status**: Complete
- **Report**: [rejected-features-consulting-ops-research.md](rejected-features-consulting-ops-research.md)
- **Summary**: Comprehensive re-evaluation of all 13 rejected features against the "one stop shop" consulting platform goal, plus assessment of consulting-specific operational tools and runtime version manager integration. Of 13 rejected features, 8 rejections were confirmed (task runner rejection was actually strengthened by devenv 2.0's expanded DAG-based task system with parallel execution, caching, JSON data passing, and process integration). One feature was fully reconsidered: code scaffolding should be integrated via Copier, which uniquely supports template updates for evolving firm-wide project standards. Three features received partial reconsideration with minimal-footprint additions: devcontainer generation (one toggle in devenv.nix), .editorconfig and .vscode/extensions.json generation (universal, low-maintenance files that compound with ecosystem detection), and an optional OTEL Collector devenv service template. For consulting operations, "client profiles" (`gdev switch <client>`) emerged as the single most impactful new feature -- bundling AWS profile, git identity, SSH key, env vars, and time tracking context into switchable profiles addresses a pain point no existing tool solves holistically. Time tracking CLIs exist but are community-maintained and low-quality; gdev should include a vendor CLI in devenv packages rather than build tracking features. Runtime version managers (mise, asdf, nvm, pyenv) are fully redundant with devenv.sh's Nix-based version pinning and should not be integrated.

### API Development & Testing Tools
- **Status**: Complete
- **Report**: [api-db-mcp-research.md](api-db-mcp-research.md) Section 1
- **Summary**: Surveyed 15 API development tools across HTTP clients, GraphQL, gRPC/protobuf, OpenAPI, and testing frameworks, verifying Nixpkgs availability for each. gdev currently has zero coverage of API tooling. The recommended expansion adds 10 tools in a detect-and-offer model triggered by project file signals. grpcurl and buf should auto-install when `.proto` files are detected (the protobuf toolchain is fragmented without them). httpie is the ergonomic curl replacement consulting engineers expect. Bruno is the clear Git-native Postman replacement (stores collections as `.bru` files). openapi-generator-cli and redocly cover OpenAPI spec-driven workflows. k6 handles load testing. Tools skipped: xh (httpie sufficient), swagger-codegen (superseded), spectral (not in nixpkgs, redocly covers it), pact/dredd (too niche). The three-tier model (always-install / detect-and-configure / optional addon) extends naturally from the existing Phase 2 language ecosystem detection.

### Database Migration & Schema Management
- **Status**: Complete
- **Report**: [api-db-mcp-research.md](api-db-mcp-research.md) Section 2
- **Summary**: Surveyed 20+ migration tools across 8 language ecosystems (JVM, Go, JS/TS, Python, .NET, Rust, Ruby, PHP) plus cross-language schema-as-code tools. The key finding: migration tools are deeply per-project concerns -- gdev should NOT choose migration tools, but should detect what the project uses and remove friction. For tools with native system dependencies (Flyway, Liquibase, Prisma, diesel-cli, sqlx-cli, sea-orm-cli, goose, go-migrate, Atlas, dbmate), gdev should detect config files and add the CLI to devenv.nix. For tools installed by project package managers (Drizzle, Knex, Alembic, Django migrations, ActiveRecord, Laravel, EF Core), gdev should detect and document in CLAUDE.md so AI agents understand the migration workflow. Detection heuristics are straightforward: `prisma/schema.prisma`, `flyway.conf`, `diesel.toml`, `atlas.hcl`, `alembic.ini`, `drizzle.config.ts`, etc. This is a natural extension of Phase 2 ecosystem detection.

### Git Platform CLIs, Documentation Tools & IDE Configuration
- **Status**: Complete
- **Report**: [git-docs-ide-research.md](git-docs-ide-research.md)
- **Summary**: Assessed 30+ tools across three gap categories. gh (GitHub CLI) is the highest-impact single addition -- practically essential for GitHub-hosted projects and complementary to the existing GitHub MCP server. git-lfs should auto-install when `.gitattributes` contains LFS filters. glab should install when GitLab remotes are detected. Documentation tools (mkdocs, mdbook, mermaid-cli, d2, plantuml) fit naturally into ecosystem detection using existing config-file heuristics (mkdocs.yml, book.toml, *.d2). For IDE configuration, the DX polish spike's rejection should be narrowed rather than reversed: .editorconfig is universally safe and should always be generated (editor-agnostic, near-zero risk, prevents noisy mixed-formatting diffs); .vscode/extensions.json is low-risk and should be opt-in via `gdev enable vscode` (creates recommendations, never auto-installs). No major dev environment tool (devenv.sh, mise) auto-generates VS Code config, but devcontainers do -- and devenv can generate devcontainer.json. LSP servers should be included in devenv.nix packages alongside language runtimes to benefit all editor users. delta (syntax-highlighting diff pager) is the strongest git productivity tool candidate. All tools fit within the existing devenv addon architecture.

### MCP Server Ecosystem Expansion
- **Status**: Complete
- **Report**: [api-db-mcp-research.md](api-db-mcp-research.md) Section 3
- **Summary**: Assessed 17 candidate MCP servers across 7 categories (database, cloud provider, ticketing, communication, observability, CI/CD, search) beyond gdev's current 5 servers. The MCP ecosystem has exploded to 10,000+ servers, but the critical constraint remains: a practical ceiling of ~40 active tools across all servers (each server exposes 5-15 tools, each adding 4-6K input tokens). Never more than ~6 servers should be simultaneously active. Recommended additions: MySQL MCP and SQLite MCP as auto-detected (paralleling existing PostgreSQL MCP), Terraform MCP and Sentry MCP as detect-and-offer, and Atlassian (Jira/Confluence) MCP as the highest-value optional addon for consulting (GA since Feb 2026, OAuth 2.1, respects existing permissions). Additional optional addons: Linear, Slack, Datadog, Grafana, GitLab, AWS, Azure MCPs. Skipped: MongoDB/Redis MCPs (data models less amenable to MCP exploration), multi-DB servers (security risk), Brave Search (Claude Code has built-in web search), Memory/Filesystem (Claude Code has native equivalents). A three-tier security model governs auto-configuration: low-risk (DB read-only) auto-configure, medium-risk (ticketing/observability) opt-in, high-risk (cloud/comms) explicit only.

### Addon Architecture Fit Assessment
- **Status**: Complete
- **Report**: [addon-architecture-fit-research.md](addon-architecture-fit-research.md)
- **Summary**: All identified ecosystem expansions fit within the existing 3-addon architecture (devenv, claudecode, devinit). No fourth addon is warranted. The devenv addon absorbs the most content (cloud/K8s modules, service templates, tool packages, LSP servers). The claudecode addon gets expanded MCP server configuration and CLAUDE.md sections for new tools. The devinit addon gets client profiles, Copier template integration, .editorconfig/.vscode generation, and expanded detection heuristics. The strongest candidate for a fourth addon was client profiles (`gdev switch <client>`), but these are fundamentally orchestration — coordinating which configuration is active — which is devinit's responsibility. Ten implementation plan phase amendments were identified covering Phases 1-14. No new phases are needed, though Phase 7 may benefit from splitting into language (7a) and non-language (7b) tool modules given the volume increase from ~19 to ~34 modules.

### Coverage Matrix
- **Status**: Complete
- **Report**: [coverage-matrix-research.md](coverage-matrix-research.md)
- **Summary**: Comprehensive inventory of all tools, services, and integrations across the 22-phase implementation plan and 7+ gdev research spikes. 13 gap categories identified: cloud CLIs (A), K8s tools (B), development services (C), API tools (D), DB migration (E), observability (F), git CLIs (G), documentation tools (H), IDE config (I), MCP servers (J), consulting ops (K), runtime managers (L), code quality (M). Categories A (cloud CLIs) and B (K8s tools) are the largest gaps. Category L (runtime managers) is fully redundant with devenv.sh.

### mcph MCP Orchestrator Evaluation (Yaw Labs / mcp.hosting)
- **Status**: Complete
- **Report**: [mcph-orchestrator-research.md](mcph-orchestrator-research.md)
- **Summary**: Evaluated mcph (`@yawlabs/mcph` v0.47.5, TypeScript, 1 contributor) as a potential replacement for gdev's Unit 3.5.1 McpServerDef registry. mcph is a runtime MCP proxy that interposes between AI clients and upstream servers, providing cloud-synced configuration, BM25-based intelligent routing, per-server health monitoring, and credential injection from encrypted cloud storage via mcp.hosting. However, it solves a fundamentally different problem: mcph is a runtime orchestrator that must be running during AI sessions; gdev is a build-time code generator that produces static `.mcp.json` files and exits. Adopting mcph would require replacing gdev's entire MCP config pipeline with a cloud-dependent proxy from a 5-week-old startup with 1 star, 1 contributor, and no license on the core package. Verdict: do not adopt mcph as a dependency. Two components are worth cherry-picking: (1) `@yawlabs/mcp-compliance` (88-test spec compliance suite, MIT licensed) for Phase 17 test infrastructure, and (2) `@yawlabs/aws-mcp` (24-tool AWS server with SSO re-auth) as a potential optional catalog entry. The Yaw Labs ecosystem shows high engineering quality for a solo project and is worth monitoring for maturity over the next 6-12 months.

### Learning Opportunities / Orient / Auto — Deep Plugin Evaluation
- **Status**: Complete
- **Report**: [learning-opportunities-research.md](learning-opportunities-research.md)
- **Summary**: Deep evaluation of DrCatHicks/learning-opportunities (1,530 stars, CC-BY-4.0) for gdev integration. The repo contains three Claude Code plugins in a monorepo marketplace: learning-opportunities (core skill with 6 exercise types grounded in learning science), orient (codebase orientation generator using program comprehension research), and learning-opportunities-auto (PostToolUse bash hook). All skill files use standard YAML frontmatter SKILL.md format, directly compatible with gdev's `deploySkills()` pattern — zero runtime dependencies, pure markdown deployed via file copy. Recommend including learning-opportunities and orient as opt-in skills via `gdev enable learning-opportunities` and `gdev enable orient`. Exclude the auto hook (conflicts with gdev's hook architecture, redundant with the skill's autonomous behavior). Orient maps directly to Phase 13 Join mode onboarding — generates structured orientation exercises for unfamiliar codebases. MEASURE-THIS.md provides validated survey instruments (n=3,267 developers, IEEE Software) but is a methodology reference, not a deployable tool. Research foundation is highly credible: Dr. Cat Hicks holds a PhD in Quantitative Experimental Psychology (UC San Diego), has published in IEEE Software, and has upcoming Routledge book "The Psychology of Software Teams." The five learning science risks (generation effect, fluency illusion, spacing effect, metacognition gap, testing deficit) are amplified in consulting contexts and should inform Phase 14 agent prompt design.

### Per-Session Claude Code Context Overlays & Project Clarity Templates
- **Status**: Complete
- **Report**: [context-overlays-clarity-research.md](context-overlays-clarity-research.md)
- **Summary**: Deep investigation into two external ideas for gdev integration: (A) per-session Claude Code context overlays via devenv enterShell, inspired by Yaw Labs' "Yaw Mode," and (B) project clarity documentation as CLAUDE.md template content, from vtemian's "Fight Slop with Clarity." For Part A, the key finding is that Claude Code does NOT support environment-based skills/rules path configuration (no `CLAUDE_SKILLS_PATH` or similar). Yaw Mode uses an undocumented `CLAUDE_CONFIG_DIR` variable to redirect `~/.claude/` to a temporary overlay directory with hardlinked state files and symlinked skills, but this variable is buggy and unstable. The recommended approach is gdev's existing Phase 4 plan (static `.claude/` directory generation) plus devenv environment variables for the few supported behaviors (`CLAUDE_CODE_EFFORT_LEVEL`, `GDEV_*` metadata variables). The `--add-dir` flag with `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` is a promising future mechanism for loading shared organization context. For Part B, a "Project Context" section was designed for CLAUDE.md with clarity questions (purpose, stakeholders, success criteria, exclusions, consulting-specific fields). This integrates with Copier templates so answers flow into both CLAUDE.md and README.md during `gdev init --from <template>`. The tarpit test was formalized as a design principle for plan.md and as a working principle in generated CLAUDE.md.

### External Ideas Analysis (Yaw Labs, Learning Opportunities, Fight Slop with Clarity)
- **Status**: Complete
- **Report**: [external-ideas-research.md](external-ideas-research.md)
- **Summary**: Analyzed three external sources for gdev-relevant ideas. Four actionable candidates identified: (1) **Learning Opportunities** (DrCatHicks) — a research-backed Claude Code skill plugin that prompts evidence-based learning exercises after significant development work, addressing five learning science risks amplified by AI-assisted coding. Directly deployable in gdev's skill library as opt-in. CC-BY-4.0 licensed. (2) **Orient** — companion plugin generating codebase orientation lessons for unfamiliar repos, mapping to gdev's Join mode onboarding. (3) **Per-session context overlays** — Yaw Labs' concept of layering Claude Code rules/skills per-session via environment, achievable through devenv enterShell hooks setting Claude Code env vars. (4) **Project clarity questions** — from "Fight Slop with Clarity," codifiable as a CLAUDE.md template section ("What problem? Whose problem? What does success look like? What to exclude?"). Additionally, `mcph` (open-source MCP orchestrator) warrants investigation as a potential replacement for our custom MCP registry (Unit 3.5.1), and the five learning science risks (generation effect, fluency illusion, spacing effect, metacognition gap, testing deficit) should inform Phase 14 consulting agent prompt design. The "tarpit test" ("if a tool sells itself as a replacement for thinking clearly, it's a tarpit") validates gdev's existing feature rejection philosophy.

## Open Questions

- ~~What developer tools are consulting engineers at Highspring expected to use daily that aren't covered by the plan?~~ Answered: See consulting-daily-driver-research.md
- ~~Which cloud providers/CLI tools should gdev configure out of the box vs leave to profiles?~~ Answered: See cloud-k8s-tooling-research.md (4-tier framework)
- ~~Should IDE configuration (beyond Claude Code) be in scope, or is that explicitly rejected?~~ Answered: Partial reconsideration — .editorconfig and .vscode/extensions.json only
- ~~Is there a fourth addon warranted, or do all expansions fit within the existing three?~~ Answered: No. All expansions fit existing 3-addon architecture. See addon-architecture-fit-research.md
- ~~What's the boundary between gdev's responsibility and devenv.sh's native capabilities?~~ Answered: gdev generates devenv.nix config; devenv provides runtime. gdev wraps tools, doesn't reimplement them.
- ~~Should client profiles live in `.gdev.local.yaml` or a separate config?~~ Answered: `~/.qsdev/clients/<name>.yaml`, sops+age encrypted. See client-profile-design.md
- What is the minimal viable Copier template for a Highspring engagement?
- ~~Should Phase 7 be split into 7a and 7b?~~ Answered: Yes. 7a = language Tiers 2-4, 7b = non-language tool modules. See shell-tools-ide-design.md

## Conclusions

### The 3-addon architecture holds

Every identified expansion maps cleanly to one of the three existing addons. The devenv addon absorbs most new content (tool packages, service templates, cloud/K8s modules). The claudecode addon expands MCP server configuration and CLAUDE.md content. The devinit addon gains client profiles, Copier integration, and new detection heuristics. No fourth addon is justified.

### Top-priority expansion categories

1. **Cloud CLIs + credential management** (gap A) — Largest gap. AWS CLI, GCP SDK, Azure CLI, plus credential helpers (aws-vault, saml2aws). Installation is trivial (all in Nixpkgs); credential isolation is the hard problem.
2. **Client profiles as init-time selection** — sops+age encrypted profiles in `~/.qsdev/clients/` selected during `gdev init`. Non-secret values (aws_profile name, git email, registry URLs, compliance level) baked into project config. Secret values generate SecretSpec entries resolved at devenv runtime via providers (keyring/1Password/env). Two-layer security: encryption at rest + runtime resolution. No plaintext secrets in project files. Strongest consulting differentiator.
3. **Kubernetes tools** (gap B) — kubectl, kubectx, k9s, stern are daily-use for cloud-native consulting. New `kubernetes` module category.
4. **Kafka service template** — Tier 1 essential, excellent devenv.sh KRaft support.
5. **Shell/workstation configuration** — Rather than bundling modern coreutils (ripgrep, fd, bat, fzf, jq, yq, delta, eza, zoxide) into devenv, explore `gdev setup` providing a personal shell configuration mode that manages `~/.qsdev/shell/` dotfile fragments. These tools belong in the engineer's personal shell, not per-project devenv.

### Moderate-priority expansions

6. **gh (GitHub CLI)** — Auto-install when `.github/` detected. Highest-impact single git tool addition.
7. **MCP server expansion** — MySQL/SQLite (auto-detect), Terraform/Sentry (detect-and-offer), Atlassian (optional, highest consulting value).
8. **Service template expansion** — MinIO, Mailpit, Keycloak, NATS as Tier 2 services.
9. **API tools** — grpcurl+buf auto-detect on `.proto`; httpie, bruno, openapi-generator detect-and-offer.
10. **Copier template integration** — `gdev init --from <template>` for firm-wide project standards.

### Lower-priority but worthwhile

11. **.editorconfig generation** — Always-generate, universally safe.
12. **Observability sidecar** — `gdev enable observability` via Docker grafana/otel-lgtm.
13. **DB migration tool detection** — Detect and install CLI binaries, document in CLAUDE.md.
14. **Documentation tools** — Detect mkdocs.yml/book.toml and install tools.
15. **.vscode/extensions.json** — Opt-in via `gdev enable vscode`.
16. **Devcontainer toggle** — `devcontainer.enable = true` in devenv.nix.

### Confirmed non-expansions

- Runtime version managers (mise/asdf/nvm) — fully redundant with devenv.sh
- Standalone task runners — devenv 2.0 tasks are sufficient
- Container management, CI execution, deployment, scaffolding beyond Copier — correctly rejected
- Time tracking CLIs — don't meaningfully exist; web/app workflow
- Full IDE configuration — only .editorconfig and .vscode/extensions.json
- Modern coreutils in devenv — these are personal shell tools, not per-project. Manage from shell config, not devenv.nix.

### Critical dependency: devenv 2.0 minimum

The task runner rejection (and devenv task definition generation) depends on devenv 2.0's expanded DAG-based task system with parallel execution, caching, JSON data passing, and process integration. **devenv >= 2.0 must be documented as a minimum version requirement** for the implementation plan. Key 2.0 features gdev depends on:
- `before`/`after` DAG task ordering
- `status` and `execIfModified` caching
- `$DEVENV_TASK_INPUT` / `$DEVENV_TASKS_OUTPUTS` JSON data passing
- `devenv:processes:*` auto-exposed tasks
- `namespace:task` convention
- `devcontainer.enable` native toggle

### Implementation plan impact

9 phase amendments producing 51 new implementation units (see [implementation-plan-amendment-proposal.md](implementation-plan-amendment-proposal.md)):

| Phase | Amendment | Units |
|-------|-----------|-------|
| 2 | Cloud & K8s modules | +9 |
| 3 | Service templates (Kafka, MinIO, Mailpit, Keycloak, NATS) | +6 |
| 4 | MCP server registry & auto-detect | +4 |
| 6 | Copier templates + client profile wizard | +10 |
| 7b (new) | Non-language tool detection modules | +4 |
| 8 | .editorconfig + .vscode generation | +2 |
| 10 | Shell/workstation configuration | +4 |
| 12 | MCP lifecycle + observability sidecar | +8 |
| 13 | Client profile system | +4 |

Phase 7 splits into 7a (language Tiers 2-4) and 7b (non-language tool modules). All 6 detailed design documents are in this spike directory.

### Phase 2 Design Documents

- [cloud-k8s-module-design.md](cloud-k8s-module-design.md) — 9 units for AWS/GCP/Azure/platform CLIs + K8s core/dev/security
- [client-profile-design.md](client-profile-design.md) — 8 units for sops+age profiles, SecretSpec, wizard flow, compliance
- [service-observability-design.md](service-observability-design.md) — 9 units for services + grafana/otel-lgtm sidecar
- [mcp-expansion-design.md](mcp-expansion-design.md) — 9 units for MCP registry, auto-detect, lifecycle, optional catalog
- [shell-tools-ide-design.md](shell-tools-ide-design.md) — 10 units for shell config, .editorconfig, .vscode, git/doc/API/DB modules
- [copier-integration-design.md](copier-integration-design.md) — 6 units for template registry, init/update flows, authoring spec

### External ecosystem findings

- **Learning Opportunities + Orient** (DrCatHicks, CC-BY-4.0, 1,530 stars) — Deploy as opt-in skills in claudecode addon. SKILL.md format is directly compatible with gdev's `deploySkills()`. Orient is the strongest consulting integration for Join mode codebase onboarding. Exclude the auto hook (conflicts with gdev's PostToolUse hook architecture). See [learning-opportunities-research.md](learning-opportunities-research.md).
- **mcph MCP orchestrator** — Do not adopt (runtime proxy, TypeScript, 1 star, no license). Cherry-pick `mcp-compliance` test suite (88 tests, MIT) for Phase 17. Evaluate `aws-mcp` (24 tools, SSO re-auth) as optional catalog entry. See [mcph-orchestrator-research.md](mcph-orchestrator-research.md).
- **Per-session context overlays** — No `CLAUDE_SKILLS_PATH` env var exists. `CLAUDE_CONFIG_DIR` (Yaw Mode mechanism) is undocumented and buggy. Recommended: static `.claude/` generation (Phase 4) + devenv enterShell env vars + future `--add-dir` mechanism. See [context-overlays-clarity-research.md](context-overlays-clarity-research.md).
- **Project clarity template** — "Project Context" section designed for CLAUDE.md with purpose, stakeholders, success criteria, exclusions, and consulting-specific fields. Integrates with Copier templates. Tarpit test formalized as design principle. See [context-overlays-clarity-research.md](context-overlays-clarity-research.md).

### Design principle addition: The Tarpit Test

"If a tool sells itself as a replacement for thinking clearly, it's a tarpit." Every gdev feature must amplify existing clarity, not substitute for it. This validates the 13 rejected features and should be added to plan.md as a numbered design principle.

## Depth Checklist

- [x] **Underlying mechanism explained** — Each expansion category has detailed mechanism analysis: devenv.nix generation patterns, MCP server composition, sops+age encryption flows, SecretSpec provider resolution, Docker sidecar lifecycle, Copier 3-way merge, enterShell env propagation
- [x] **Key tradeoffs and limitations identified** — Build-time vs runtime (mcph rejected), per-project vs personal shell (coreutils), install vs manage (credentials), auto-detect vs opt-in (MCP servers), 40-tool ceiling constraint, devenv 2.0 minimum dependency
- [x] **Compared to alternatives** — mise vs devenv (redundant), mcph vs custom registry (different architecture), Copier vs cookiecutter (template updates), act vs CI execution (correctly rejected), SecretSpec vs sops-only (complementary layers)
- [x] **Failure modes and edge cases** — MCP tool budget overflow, sops/age not installed, Copier not available (Python gate), CLAUDE_CONFIG_DIR bugs, Docker not available for observability sidecar, profile compliance tampering
- [x] **Concrete examples and reference implementations** — 51 implementation units with concrete Nix code, YAML schemas, detection heuristics, CLI commands, .mcp.json blocks, CLAUDE.md sections, Go struct definitions
- [x] **Report is standalone-readable** — Yes. Each research report and design document is self-contained with sufficient context for implementation without consulting conversation history
