# Research Log: Dev Containers vs Nix Competitive Analysis

## 2026-03-20 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Awaiting scope confirmation and task decomposition.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-03-20 — GitHub Codespaces Deep Dive Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: GitHub Docs (architecture, billing, prebuilds, security, org management), Tempered Works one-year review, vcluster comparison blog, GitHub Changelog (data residency), community discussions (performance, stability)
- **Summary**: Comprehensive research on GitHub Codespaces covering architecture (dedicated Azure VMs + Dev Containers), pricing ($0.18-$2.88/hr compute, $0.07/GiB/mo storage, 120 core-hours free tier for personal accounts), prebuilds (reduce startup from minutes to <1 min), org management (6 policy controls), three-level secrets system, hard offline limitation, performance trade-offs, multi-repo support, and consulting-specific analysis. Key consulting finding: Codespaces works well when clients use GitHub and accept Azure-hosted compute, but fails for clients with data sovereignty requirements, non-GitHub repos, or offline needs. GPU support deprecated Aug 2025. Data residency only in preview as of Jan 2026.
- **Next**: Dev Containers deep dive, Coder deep dive, DevPod deep dive.

## 2026-03-20 — DevPod Deep Dive Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: DevPod official docs (what-is-devpod, how-it-works, providers, credentials, devcontainer-json, provider-development, agent), GitHub README, GitHub Issues #1915 and #1946, vcluster comparison blog, web search results on limitations/real-world usage
- **Summary**: Comprehensive research on DevPod covering client-agent architecture (client-only, injected agent, vendor-specific tunnels), provider ecosystem (7 official providers for Docker/k8s/SSH/AWS/GCP/Azure/DO, 10+ community providers, custom provider development via provider.yaml), full devcontainer.json implementation (with 3 unsupported properties), credential forwarding (git/docker/GPG — forwarded not stored, but no per-workspace scoping), IDE support (VS Code, JetBrains, any SSH client), CLI automation capabilities, prebuilds, and auto-inactivity shutdown. Critical finding: DevPod is effectively unmaintained since mid-2025 — Loft Labs rebranded to vCluster Labs and shifted all resources to vCluster. Last release v0.6.15 (March 2025). Community fork exists (Issue #1946) but sustainability uncertain. Provider model is architecturally excellent for consulting (route each client to their own infra) but maintenance risk makes it unsuitable for production adoption.
- **Next**: Dev Containers deep dive, Coder deep dive, Nix-adjacent alternatives survey.

## 2026-03-20 — Nix-Adjacent Alternatives Survey Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: Devbox GitHub README, Devbox configuration docs, Devbox internals (DeepWiki), copier-org discussion (Devbox vs plain Nix), Flox GitHub README, Flox enterprise blog, Flox manifest/services docs, Flox pricing, Pixi GitHub README, pixi.toml reference, pixi vs conda comparison, team adoption case studies (Alan for Devbox, PostHog for Flox)
- **Summary**: Comprehensive survey of three Nix-adjacent tools. **Devbox** (11.4k stars, Apache 2.0) wraps Nix behind JSON config, generates flakes internally, resolves via NixHub API, offers plugin system and process-compose services. Best for standard web stacks without Nix expertise. **Flox** (3.8k stars, GPLv2, $25M Series B) wraps Nix behind TOML config, adds FloxHub for centralized environment sharing, environment layering/composition, enterprise features (SBOM, private catalogs, $40/seat/month team tier). Best for compliance-heavy or team-sharing scenarios. **Pixi** (6.6k stars, BSD-3) is not Nix-based at all — builds on conda ecosystem in Rust, 10x+ faster than conda, native Windows support, built-in task runner and lockfiles. Best for data science/ML/Python-heavy projects. Key insight: all three trade Nix's full power for accessibility; the leaky abstraction critique applies when you need custom overlays or patches. For standard consulting toolchains, Devbox has the strongest fit.
- **Next**: Dev Containers deep dive, Coder deep dive, consulting scenario matrix.

## 2026-03-20 — Coder Deep Dive Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: Coder GitHub README, raw GitHub docs (templates, devcontainers, workspace lifecycle, workspace scheduling, secrets, template permissions, organizations, organizations best practices, user management), Multi-Tenancy RFC Discussion #7638, web searches (architecture, pricing, enterprise features, IDE support, air-gapped deployments, multi-org, adoption, limitations, Codespaces comparison, Envbuilder)
- **Summary**: Comprehensive research on Coder covering architecture (coderd control plane + provisionerd Terraform executor + agent in workspace, WireGuard encrypted tunnels with DERP relay fallback, PostgreSQL 13+ datastore), Terraform template system (full IaC — anything Terraform can provision becomes a workspace: EC2, K8s pods, Docker containers; parameters, versioning, GitOps pipelines), workspace lifecycle (Running/Stopped/Failed/Unhealthy/Deleted/Dormant states, ephemeral vs persistent resources, auto-start/stop with activity detection, dormancy auto-deletion), IDE support (VS Code desktop/browser, JetBrains Gateway/Toolbox, Cursor, Windsurf, SSH, web terminal — genuinely IDE-agnostic), enterprise features (5 built-in RBAC roles + custom org-scoped roles, audit logging, HA multi-replica, workspace proxies for geo-distribution, SOC2 Type II), pricing (Community free with unlimited workspaces/users/templates under AGPL v3.0; Premium annual per-seat license adds multi-org, audit, quotas, HA, SCIM, custom roles — price not public), devcontainer.json support (sub-agent model: workspace runs Docker, agent discovers and builds dev containers via @devcontainers/cli; Envbuilder alternative for non-Docker environments), multi-tenant Organizations (Premium — each org gets separate provisioners with isolated cloud credentials, separate templates, separate admins, users can span orgs; provisioners run in isolated infra that control plane cannot access), secrets (dynamic injection via Terraform providers, Vault integration, cloud IAM per-workspace, SSH keys in-memory only), air-gapped deployment (full feature support, custom images, Terraform mirror, self-hosted docs), and real-world adoption (Fortune 500 across finance/defense/government, Palantir/Dropbox mentioned, 12.6k stars, ~$70M funding). Key consulting findings: (1) Coder directly solves the #1 Codespaces problem — client code stays on your infrastructure; (2) Organizations feature provides genuine multi-client isolation with separate provisioners and credentials; (3) operational overhead is high — requires Terraform/K8s expertise and ongoing platform engineering; (4) Premium license required for consulting-critical features; (5) best fit for firms with 20+ devs, existing K8s infrastructure, and long engagements in regulated industries.
- **Next**: Consulting scenario matrix, crisp answer.

## 2026-03-20 — Dev Containers Deep Dive Completed
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**: devcontainer.json reference (GitHub raw), Dev Container Features spec (GitHub raw), Features distribution spec (GitHub raw), devcontainer CLI README (GitHub raw), secrets support spec (GitHub raw), devcontainers/ci GitHub Action docs (GitHub raw), web searches on performance/overhead, IDE support (JetBrains/Neovim), secrets/credentials patterns, Docker Compose multi-container setups, prebuilds, consulting usage patterns, specification governance
- **Summary**: Comprehensive research on the Dev Containers specification and ecosystem. Covered: devcontainer.json format (3 container modes, lifecycle hooks, variable substitution, host requirements), Features system (composable install scripts distributed via OCI registries with dependency resolution), 6 lifecycle hooks with failure-stops-chain semantics, Docker Compose multi-container orchestration (two networking patterns), IDE support (VS Code excellent, JetBrains partial/cumbersome, Neovim community-only), open spec governance (CC BY 4.0 but Microsoft-copyrighted and controlled), devcontainer CLI for headless CI/CD, and detailed limitations (Docker dependency, macOS file I/O penalties, rebuild latency, credential management immaturity). Consulting-specific analysis: containers provide stronger isolation than Nix devShells but with meaningful overhead — slower project switching (5-30s vs sub-second), higher resource usage, Docker licensing considerations. Credential management patterns exist but are less elegant than direnv's .envrc.local approach. Enterprise clients more likely to accept Docker-based workflows than Nix.
- **Next**: Coder deep dive, consulting scenario matrix, crisp answer.

## 2026-03-20 — Consulting Scenario Matrix & Crisp Answer Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all six deep-dive reports into an eight-dimension consulting scenario matrix (isolation, onboarding, credentials, offline, client constraints, reproducibility, cost, adoption friction) and five scenario-based recommendations. Wrote the crisp "why not devcontainers?" answer: devcontainers trade speed for isolation (5-30s switching vs sub-second), and the real question isn't "devcontainers or Nix" but "which problem are you solving?" Decision framework maps when to lead with Nix vs containers vs cloud CDEs. Six key findings documented including DevPod being dead, Codespaces having hard consulting blockers, and Devbox as the pragmatic stepping stone.
- **Next**: Spike ready for completion review.

## 2026-03-27 — Spike Completed
- **Type**: decision
- **Status**: success
- **Depth**: deep
- **Summary**: All seven tasks completed across one phase. Produced six deep-dive reports (Dev Containers, Codespaces, Coder, DevPod, Nix-adjacent alternatives, consulting scenario matrix) plus the crisp "why not devcontainers?" answer. Core conclusion: devcontainers trade speed for isolation — Nix wins for daily multi-project consulting workflows (sub-second switching, zero infrastructure cost, offline capability), while containers win when clients mandate Docker, require process-level isolation, or need production parity. The tools aren't on a single spectrum; the right question is "which problem are you solving?" Six key findings: DevPod is dead, Codespaces has hard consulting blockers, Coder is the enterprise answer at enterprise cost, Dev Containers and Nix are complementary not competitive, Devbox is the pragmatic stepping stone, and no single tool covers all consulting scenarios. Depth checklist fully satisfied. 47+ source documents saved across all reports.
