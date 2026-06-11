# Roadmap

Planned enhancements for the v1 release cycle. Items are grouped by theme and ordered roughly by priority within each group.

The full implementation plan spans 71 phases across 8 depth levels. This roadmap covers the publicly visible features; see the internal implementation plan for phase dependencies, unit estimates, and parallelization strategy.

## Recently Shipped

Features delivered since MVP:

- **Fragment-based orchestration** (P15) — Modules produce typed Fragment objects with 5 compose modes (replace, append, section, merge-JSON, merge-YAML); an orchestrator merges fragments with priority ordering and security floor enforcement
- **Managed hook policies** (P16) — Declarative hook registry with 3-tier deployment hierarchy (project, team, org), credential scanning, destructive operation prevention, SOC 2 audit logging, and dependency oversight hooks
- **Container runtime modernization** (P17) — Docker ecosystem module renamed to "container" with runtime-agnostic support for Docker and Podman rootless; `qsdev container detect` and `qsdev container migrate` for Docker-to-Podman migration analysis
- **Hook execution sandboxing** (P18) — Bubblewrap-based sandbox with Landlock filesystem restriction and seccomp-BPF syscall filtering; 5 degradation tiers from full isolation to unsandboxed; `qsdev sandbox exec` and `qsdev sandbox status` commands
- **Multi-framework shared primitives** (P19) — 7 Go interfaces (DetectionAdapter, ConfigRenderer, HookDeployer, etc.) abstracting all AI framework addons for future multi-framework support
- **OpenGrep custom rule library** (P20) — 96 taint-focused security rules covering 7 frameworks (Next.js, FastAPI, Gin, NestJS, SvelteKit, Prisma, Drizzle) across TypeScript, Python, and Go
- **Wizard & orchestration refinements** (P21) — `--theme` flag (charm, dracula, catppuccin, base16, default), `--quiet` flag, terminal detection guard, AI tools form consolidation
- **Golden-file test infrastructure** — Snapshot-based testing for generated configuration files with `GOLDEN_UPDATE=1` support
- **CI template generation** — Security-hardened GitHub Actions workflows with SHA-pinned action refs and `permissions: {}` defaults
- **DX polish** — `qsdev repair` with backup-before-modify, `qsdev info` for subsecond project context, `qsdev outdated` for cross-ecosystem dependency freshness, `qsdev check --auto-fix` for deleted files and deny rules, background self-update checks
- **Distribution & self-bootstrapping** (P22) — Redesigned `qsdev update` with 3-stage coordinated update (binary + configs + devenv inputs) and 7 flags (--check, --changelog, --dry-run, --force, --self-only, --configs-only, --deps-only); install script hardening with Sigstore cosign verification, Rosetta 2 detection, and musl detection; AUR PKGBUILD; dual-format SBOMs (SPDX + CycloneDX)
- **AI agent tooling** (P23) — Agent-postmortem and version-sentinel embedded MCP servers; `qsdev mcp status` and `qsdev mcp list` diagnostic commands; pluggable MCPServerProvider architecture with catalog-driven auto-injection
- **Cloud CLI detection** (P24) — AWS, GCP, and Azure ecosystem modules with auto-detection from project files (CDK, SAM, Terraform providers, cloud CLI configs); 3-layer credential isolation (environment separation, credential file masking, agent deny rules)
- **Service template expansion** (P25) — Kafka (KRaft/ZooKeeper), MinIO, Mailpit, Keycloak, and NATS service templates with configurable ports, auto-exported environment variables, and localhost-only binding; 12 total services
- **MCP registry & documentation pipeline** (P26) — MCP server registry with 5-level compliance grading; `qsdev mcp grade/install/update/remove/health` lifecycle commands; `qsdev docs` pipeline with DevDocs and ZIM offline documentation; 4 documentation MCP servers; lookup-docs skill with 5-source priority routing
- **Security pattern library & policy engine** (P27) — YAML-based policy engine with 10 condition types, 4 action types, and 3-tier bypass model; 28-probe package risk scoring with A-F grades; 9-probe MCP trust scoring; `qsdev policy check/list/show` and `qsdev session allow/clear/list` commands; SARIF 2.1.0 output
- **Agent self-protection** (P28) — 18 Tier 1 enforce-always rules across config protection, MCP integrity, binary integrity, and bypass prevention; evasion detection for base64-to-shell, hex encoding, hardlinks, and /proc tricks; path canonicalization; runs as first PreToolUse hook with fail-closed semantics

## Security & Supply Chain

Deepening the 14-layer defense stack.

- **Interactive bypass system** — Configuration-guard rules with `qsdev hook bypass-next` for authorized overrides, shadow-mode calibration for safe rule rollout, circuit breaker pattern
- **Consolidated security binary** — Single `qsdev-hook` binary replacing Python hook scripts, with <50ms PreToolUse execution and advanced evasion detection (base64, shell expansion, indirect writes)
- **Cache poisoning defense** — CREEP detection rules for CI caches, branch-scoped isolation (restore-only for PRs), OIDC sub claim enforcement
- **Content signing for MCP** — Minisign Ed25519 signing for documentation, Unicode normalization, datamarking for prompt injection defense

## Vulnerability Analysis

New capabilities for understanding vulnerability impact.

- **Nixpkgs PURL mapping database** — Maps nixpkgs derivations to upstream Package URLs, making Nix packages visible to standard vulnerability scanners (OSV.dev, GHSA) for the first time; pure-Go SQLite, <10ms queries
- **Vulnerability reachability engine** — Four-tier pipeline: T0 heuristic (40–60% reduction), T1 import graph (additional 20–30%), T2 export surface (additional 10–20%), T3 function-level call graph (research-gated)
- **Nix patch-aware suppression** — 5-stage multi-signal patch detection pipeline generating automatic OpenVEX suppression documents for backported security fixes
- **Nix overlay identity tracking** — Overlay detection via drvPath comparison, 7-category modification taxonomy
- **Real-time vulnerability monitor** — `qsdev-watchd` systemd user service with SQLite project registry, OSV batch queries weighted by reachability, battery-aware scheduling
- **TypeScript call graph engine** — ts-morph engine with nominal+RTA hybrid resolution for function-level npm vulnerability reporting

## AI Agent Integration

Expanding AI framework support and agent capabilities.

- **Universal MCP server** — Single `qsdev mcp serve` command serving 8+ AI frameworks with CWD-based project detection and framework auto-detection
- **AI config portability** — `qsdev init --portable` generating AGENTS.md; `.qsdev/ai-config.yaml` canonical format compiling to 9 targets (Claude, AGENTS.md, Cursor, Windsurf, Copilot, Gemini CLI, Amazon Q/Kiro, Aider, Continue.dev)
- **LSP default configuration library** — Pre-configured LSP settings for 22 language servers, PreToolUse hook redirecting grep-on-symbols to LSP tools
- **LSP sandboxing** — `mkSandboxedLsp` Nix function wrapping LSP servers in bubblewrap+Landlock+seccomp with 25 per-server policies
- **Multi-agent coordination** — Git worktree isolation, append-only JSONL event chronicle, structured task lifecycle with first-claim-wins semantics
- **Agentic quality patterns** — `orient` skill (PageRank-ranked repo map), `learning-opportunities` skill, pre-edit linting on AI-written files
- **Agent secret proxy** — Lightweight Go proxy on Unix domain socket with in-memory vault and HTTP header injection (agent never sees credentials)
- **MCP gateway security** — Docker MCP Gateway interceptors for client-agnostic enforcement; single `policy.yaml` compiles to both hook and Gateway formats

## Cloud & Infrastructure

- **Kubernetes ecosystem modules** — kubectl, enhanced Helm module, kubescape security scanning with per-project KUBECONFIG enforcement

## Developer Experience

- **Non-language tool detection** — Git platform detection (GitHub/GitLab/Bitbucket), documentation generators, API frameworks, database migration tools
- **IDE and shell configuration** — EditorConfig generation, VS Code settings/extensions.json, Starship prompt configuration, shell fragment generation
- **Copier template integration** — `qsdev init --from` for templated project scaffolding with registry support
- **Config integrity analysis** — 7-format semantic diff engine (YAML, JSON, TOML, INI, Dockerfile, Shell, Nix) with ~50 threat detection rules and SARIF output
- **Config vault** — Git-based config versioning at `.qsdev/config-store/` with `qsdev vault snapshot/restore/diff/log`, environment branches, sops+age encryption

## Consulting & Team Management

- **Encrypted client profiles** — `qsdev profile create/switch/list/delete` with sops+age encryption for client secrets and profile-scoped compliance enforcement
- **Observability and analytics** — JSONL event hub, OpenTelemetry sidecar, ccusage integration for per-model cost tracking; `qsdev observe` CLI
- **Consulting lifecycle management** — 6-phase teardown protocol covering 10 artifact layers, tamper-evident audit logs, cost governance with Stop hook budget enforcement
- **Bot identity and team knowledge** — 5-layer bot identity verification, git-monorepo team vault distributing 10 functional knowledge packs
- **Team report enhancements** — CI artifact aggregation for multi-project dashboards, historical trend tracking

## Ecosystem Expansion

- **Ecosystem module hardening** — Production-quality 5-level confidence scoring, lockfile generation/enforcement, and expanded security configs for all 30 modules across 4 tiers
- **Niche tool integrations** — SpotBugs (Java static analysis), Dockle (container CIS), buf (Protobuf), Spectral (OpenAPI linting), Atlas (database migrations), Granted (AWS multi-account)
- **External addon ecosystem** — `qsdev addons` CLI (16 commands) for git-repo-as-index addon registry with exec-based plugin protocol
- **Code quality stack** — ast-grep structural search MCP server, codebase-memory-mcp call graph navigation, post-tool quality pipeline

## Distribution & Adoption

- **Cross-platform validation** — E2E testing of all 30 ecosystem modules across Linux, macOS, and Windows
- **Adoption materials** — Elevator pitch playbook (4 audience variants) with ROI framework, timed demo script, README generation template
- **Internationalization** — Project Fluent i18n with `--locale` flag, AI-batch seeded translations (ja, zh-Hans, es, fr, de, pt-BR, ko)

## Confirmed Exclusions

These features have been explicitly evaluated and excluded from the v1 scope:

- Runtime version managers (mise/asdf/nvm) — redundant with devenv.sh Nix pinning
- Standalone task runners (just/Taskfile) — devenv native task system is sufficient
- Container management / deployment automation — out of scope
- DevPod integration — unmaintained since mid-2025
- Full IDE configuration — only EditorConfig + extensions.json
- Per-tool AI config generation (.cursorrules, .windsurfrules) — AGENTS.md is the universal fallback
- Multi-platform CI runtime protection — GitHub Actions only for v1
