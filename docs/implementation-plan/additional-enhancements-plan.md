# Additional Enhancements Plan: gdev Post-MVP Expansion

## Overview

This plan extends the gdev-secure-devenv-bootstrap MVP (phases 1–22) with 15 additional phases (23–37) that deepen ecosystem coverage, add consulting-grade infrastructure, build a secure local documentation pipeline, and refine the developer experience through agentic quality tooling and session analytics.

Where the MVP establishes the core three-addon architecture (devenv, claudecode, devinit) with 27 language ecosystem modules and 6-layer security hardening, these enhancements expand into cloud/K8s operations, per-client encrypted profiles, MCP server lifecycle management with local-first documentation, managed hook policies for consulting enforcement, and observability instrumentation. The enhancements assume all 22 MVP phases are complete and a working `gdev init` → `devenv shell` pipeline exists.

All enhancements fit the existing 3-addon architecture — no fourth addon is required. The implementation adds approximately 97 units across 12 development phases and 3 validation phases. Development phases are organized into four groups: ecosystem expansion (23–27), MCP pipeline (28–29), consulting infrastructure (30–33), and developer experience refinement (34). Validation phases (35–37) mirror this grouping.

## Research Foundation

| Spike / Report | Contribution |
|---|---|
| `research-spikes/gdev-ecosystem-expansion-assessment/research.md` | 51 implementation units across 9 phase amendments: cloud CLIs, K8s ecosystem, service templates, MCP registry, Copier integration, client profiles, tool detection modules, IDE/shell config, observability sidecar. Tarpit test design principle. 13 confirmed non-expansions. |
| `research-spikes/gdev-local-docs-mcp/research.md` | 3-tier local documentation architecture (local/enterprise/fallback), 5 MCP server integrations, skill-level routing, `gdev docs` command design, wizard integration. Quantified web fetch injection risk (66–84% ASR). |
| `research-spikes/mcp-content-signing-verification/research.md` | Minisign signing pipeline, CI verification workflow, MCP startup verification, content diffing. Industry-wide gap: no documentation aggregator implements signing. 1–3 person-day effort. |
| `research-spikes/mcp-documentation-prompt-injection-hardening/research.md` | 5-layer defense architecture reducing ASR from ~25% to ~1–2%. Datamarking as highest single-defense impact. Content sanitization, structural framing, provenance metadata. |
| `research-spikes/sotoki-filtered-stackoverflow-subsets/research.md` | Per-ecosystem SO ZIM builds via sotoki `--include-tags` upstream PR. Size estimates per ecosystem (250 MB–13 GB). Shared ZIM store architecture. Fork-and-upstream recommended approach. |
| `research-spikes/claude-code-hooks-in-practice/research.md` | 6 production-ready consulting hook configurations. 3-tier deployment strategy (managed/user/project). Performance constraints (<200ms). Claude Code version pinning requirement. SessionStart bug documentation. |
| `research-spikes/claude-tools-consulting-adoption/research.md` | 4-tier observability deployment (ccusage → claude-history → session replay → team analytics). Metadata-only privacy constraint for consulting. Rudel rejection (uploads full transcripts). |
| `research-spikes/claude-code-analysis-tools/research.md` | 5-question session analysis taxonomy. JSONL session format documentation. Team analytics gap analysis. Tool landscape (ccusage 12K stars, Claude DevTools 2.7K, claude-replay 573, claude-history 110). |
| `research-spikes/agentic-workflow-state-of-art/research.md` | Scaffold architecture 27× more impactful than model changes. External verification as #1 quality multiplier (TDFlow 88.8% vs 49%). Constrained tools outperform unconstrained. Aggressive directives degrade Claude 4.6. Agent plateau at ~2 hours. |
| `research-spikes/nix-adoption-failure-reversion/research.md` | Zero documented abandonments of devShells+direnv scope. Champion dependency as #1 team failure mode. AI assistance neutralizes top 3 pain points. Shopify two-act lesson (raw Nix failed, devenv succeeded). |
| `research-spikes/consulting-tooling-adoption-roi/research.md` | $524K–$1.18M annual friction cost for 20-person team. 264–889% first-year ROI. Environment setup 2–5 days → under 1 hour. Utilization-based value framing. CI efficiency claims unsubstantiated — do not use. |
| `research-spikes/devcontainers-vs-nix-competitive-analysis/research.md` | Nix+direnv wins for multi-project consulting (sub-second switching). DevPod dead (mid-2025). Codespaces has sovereignty blockers. Dev Containers complementary, not competitive. Devbox as stepping-stone option. |

## Design Principles (Supplementary)

These complement the 15 principles in `plan.md`. Numbering continues from there.

16. **Tarpit test.** If a feature sells itself as a replacement for thinking clearly, it's a tarpit. Every proposed addition passes a 4-question rubric: Does this require understanding the domain? Does this hide failure modes? Does a purpose-built tool already exist? Does it compound with existing features or compete? This retroactively validates the 13 features rejected in the MVP plan and the 18 tools rejected in Phase 12 research, and provides an evaluation framework for future proposals.

17. **Credential isolation, not credential management.** gdev configures per-project environment variables (`AWS_PROFILE`, `KUBECONFIG`, `CLOUDSDK_ACTIVE_CONFIG_NAME`) in devenv.nix. It never stores, retrieves, rotates, or manages actual credentials. The security value is preventing cross-client credential leakage — running `terraform apply` against the wrong AWS account. Credential lifecycle remains the domain of aws-vault, gcloud auth, az login, and SecretSpec.

18. **Local-first documentation.** Web-fetched content has a 66–84% prompt injection attack success rate in auto-execution mode, with 32% quarterly growth. Local documentation eliminates the dominant attack vector while introducing smaller, controllable residual risks. Context7 and web search are clearly labeled fallbacks, not defaults.

19. **Metadata-only analytics.** Team analytics operate on token counts, timing data, hook outcomes, and session metadata — never on source code content or prompt text. This is the architectural line that separates consulting-safe from consulting-unsafe. Tools that upload full transcripts (Rudel, Mantra) are explicitly rejected regardless of feature set.

20. **Champion-independent adoption.** Design for the scenario where the Nix champion departs. Generated configs use devenv as the abstraction layer (not raw Nix expressions). `devenv.nix` includes inline comments explaining each section. Adding common tools never requires understanding overlays, derivations, or the Nix language. The Shopify lesson: raw Nix stalled; devenv wrapper succeeded.

21. **Calm positive directives.** Generated CLAUDE.md templates and `.claude/rules/` files use calm, positive language. Research shows aggressive emphasis markers ("CRITICAL!", "MUST", "NEVER") measurably degrade Claude 4.6 performance. Reframe prohibitions as positive directives: "Use frozen lockfiles for all installs" rather than "NEVER install without a lockfile."

22. **Amplify, don't replace.** Derived from the tarpit test. Tools like learning-opportunities and orient amplify developer understanding — they don't replace thinking. The observability sidecar amplifies debugging — it doesn't automate diagnosis. Every enhancement should make the developer more capable, not more dependent.

## Prerequisites

These enhancements assume:

1. **MVP phases 1–22 complete.** The core three-addon architecture, 27 ecosystem modules, 6-layer security hardening, tool lifecycle system, and test infrastructure are all operational.
2. **devenv >= 2.0.** Required for native task definitions, DAG-based task scheduling, `devcontainer.enable`, and namespace conventions used by several enhancement phases.
3. **Phase 17 test infrastructure.** All validation phases (35–37) build on the testscript E2E framework, CI pipeline, and golden file infrastructure established in Phase 17.

## Confirmed Non-Expansions

The following were evaluated and explicitly rejected during research. They are documented here to prevent re-investigation:

| Rejected Feature | Reason |
|---|---|
| Runtime version managers (mise, asdf, nvm) | Fully redundant with devenv.sh Nix pinning |
| Standalone task runners (just, Taskfile, mise) | devenv 2.0 native task system is sufficient |
| Container management | Docker/Podman CLI already exists; gdev adds no value |
| CI execution | Out of scope — gdev generates CI configs, doesn't run CI |
| Deployment automation | Out of scope — gdev is a dev environment tool |
| Code scaffolding beyond Copier | Copier covers the template use case; more is scope creep |
| Time tracking CLIs | No viable quality CLI tools exist |
| Full IDE configuration | Only EditorConfig + extensions.json; deeper is per-IDE maintenance burden |
| Modern coreutils in devenv.nix | Personal tools belong at `~/.nix-profile`, not per-project |
| DevPod integration | Unmaintained since mid-2025 (Loft Labs pivoted to vCluster) |
| Codespaces positioning | IP sovereignty blockers for consulting |
| Rudel / Mantra analytics | Upload full transcripts including source code — incompatible with NDAs |
| mcph MCP proxy | Runtime cloud dependency, 5 weeks old, no license on core package |
| `learning-opportunities-auto` | PostToolUse hook conflicts with gdev's hook architecture |
| `CLAUDE_CONFIG_DIR` reliance | Undocumented, known bugs (#3833, #4739, #30538) |

## Phase Index

| # | Phase | Status | Dependencies | Summary |
|---|-------|--------|--------------|---------|
| 23 | Cloud CLI & Credential Isolation Modules | Not Started | Phases 1, 2 | AWS, GCP, Azure, misc cloud platform CLI modules with per-project credential env vars, Terraform provider detection, `gdev doctor` cloud checks |
| 24 | Kubernetes Ecosystem Modules | Not Started | Phases 1, 2, 23 | Core K8s tools (kubectl, kubectx, k9s, stern, kustomize), dev tools (Skaffold, Tilt, DevSpace), security tools (kubescape, kube-linter), Helm ecosystem, cloud-auth coordination, KUBECONFIG isolation |
| 25 | Service Template Expansion | Not Started | Phase 3 | Kafka (KRaft), MinIO, Mailpit, Keycloak, NATS service modules with detection engine tiering (Tier 1/Tier 2) and wizard sub-groups |
| 26 | Non-Language Tool Detection Modules | Not Started | Phase 1 | Git platform CLIs (gh, glab, git-lfs), documentation tools (mkdocs, mdbook, d2, adr-tools), API tools (grpcurl, buf, openapi-generator, bruno), database migration (Flyway, Prisma, Atlas, Alembic) |
| 27 | IDE, Shell & Workstation Configuration | Not Started | Phases 9, 10 | EditorConfig generation, VS Code extensions.json, shell fragment system (`gdev setup --shell`), personal tools via nix profile, Starship gdev module |
| 28 | MCP Server Registry & Lifecycle Management | Not Started | Phases 4, 12 | `McpServerRegistry` with metadata, auto-detection, `gdev mcp list/enable/disable`, optional catalog (Atlassian, Linear, Slack, Datadog, Grafana, DB MCPs), MCP compliance testing, security documentation |
| 29 | Local Documentation MCP Pipeline & Content Security | Not Started | Phase 28 | openzim-mcp, DevDocs MCP, man-mcp-server, MCP-NixOS integration, skill-level routing, `gdev docs` commands, Minisign content signing, 5-layer prompt injection hardening, sotoki integration planning |
| 30 | Client Profile System | Not Started | Phases 6, 13 | sops+age encrypted per-client profiles, profile CRUD commands, init-time wizard integration, SecretSpec generation, non-secret value baking, compliance enforcement in `gdev check` |
| 31 | Copier Template Integration | Not Started | Phase 6 | Template registry, `gdev template add/list/remove`, `gdev init --from <template>`, `gdev update --template`, non-interactive support, template authoring specification |
| 32 | Managed Hook Policy & Consulting Enforcement | Not Started | Phase 4 | 6 consulting hook configurations (credential scanning, destructive prevention, cost alerting, SOC 2 logging, test enforcement, client isolation), 3-tier deployment, Claude Code version pinning |
| 33 | Observability & Session Analytics | Not Started | Phases 12, 16 | OTel sidecar (grafana/otel-lgtm), ccusage cost tracking, metadata-only team analytics via hooks, `gdev observability` commands, OTEL env var generation |
| 34 | Agentic Quality, Learning & Project Clarity | Not Started | Phases 13, 14 | learning-opportunities skill, orient codebase exploration, project clarity CLAUDE.md template, tree-sitter repo map skill, pre-edit validation hook, time-to-first-environment benchmarking |
| 35 | Ecosystem & Tool Expansion Validation | Not Started | Phase 17, Phases 23–27 | Cloud/K8s module E2E, service template validation, tool detection accuracy, IDE/shell config generation, credential isolation verification |
| 36 | MCP & Documentation Pipeline Validation | Not Started | Phase 17, Phases 28–29 | MCP registry lifecycle E2E, documentation serving verification, content signing round-trip, prompt injection defense testing, skill routing validation |
| 37 | Consulting Infrastructure & Analytics Validation | Not Started | Phase 17, Phases 30–34 | Client profile encryption round-trip, Copier template E2E, hook policy enforcement, observability pipeline, agentic skill validation, analytics metadata-only verification |

## Phase Grouping

### Group A: Ecosystem Expansion (Phases 23–27)
Extends the MVP's ecosystem coverage from language runtimes into cloud infrastructure, Kubernetes operations, additional services, non-language tooling, and developer workstation configuration. All modules implement the existing `EcosystemModule` interface from Phase 1.

### Group B: MCP Pipeline (Phases 28–29)
Builds a complete MCP server management layer: registry-driven lifecycle, local-first documentation serving, cryptographic content signing, and prompt injection defense. Transforms MCP from hardcoded 5-server generation to a managed, extensible, and security-hardened pipeline.

### Group C: Consulting Infrastructure (Phases 30–33)
Adds the consulting-specific capabilities that differentiate gdev from generic dev environment tools: encrypted per-client profiles, project template management, managed security hook policies, and privacy-safe session analytics. This is the strongest consulting differentiator — no existing tool bundles these capabilities.

### Group D: Developer Experience Refinement (Phase 34)
Integrates research-backed agentic quality patterns and learning tools that make developers more effective when working with AI coding assistants. Grounded in quantitative research: scaffold architecture produces 27× more impact than model changes; external verification doubles task success rates.

### Group E: Enhancement Validation (Phases 35–37)
Three validation phases covering the four development groups. Builds on the testscript E2E framework, CI pipeline, and golden file infrastructure from MVP Phase 17. Follows the same validation methodology as MVP phases 18–22: positive controls (features work), negative controls (no regressions), and cross-feature interaction testing.

## Current Status

No enhancement work has started. Phase 23 is the entry point for Group A. Groups A–D can proceed partially in parallel once their MVP dependencies are met:
- **Group A** (Phases 23–27): Requires Phases 1, 2, 3, 9, 10 complete
- **Group B** (Phases 28–29): Requires Phases 4, 12 complete
- **Group C** (Phases 30–33): Requires Phases 4, 6, 12, 13, 16 complete
- **Group D** (Phase 34): Requires Phases 13, 14 complete
- **Group E** (Phases 35–37): Requires Phase 17 complete plus their respective development phases
