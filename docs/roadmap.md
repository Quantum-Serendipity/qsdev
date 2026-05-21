# Roadmap

Planned enhancements for the v1 release cycle. Items are grouped by theme and ordered roughly by priority within each group.

The full implementation plan spans 71 phases across 8 depth levels. This roadmap covers the publicly visible features; see the internal implementation plan for phase dependencies, unit estimates, and parallelization strategy.

## Foundation Enhancements

Features that improve the already-shipped MVP core.

- **Fragment-based orchestration** — Modules produce typed Fragment objects instead of directly writing to shared files; an orchestrator merges fragments with priority ordering and security floor enforcement
- **Golden-file test infrastructure** — Snapshot-based testing for all generated configuration files
- **CI template generation** — Security-hardened GitHub Actions workflows generated from ecosystem detection, with SHA-pinned action refs and `permissions: {}` defaults
- **DX polish** — `qsdev repair` with backup-before-modify, `qsdev info` for subsecond project context, `qsdev outdated` for cross-ecosystem dependency freshness, improved error messages and shell completion

## Security & Supply Chain

Deepening the 10-layer defense stack.

- **Native security policy engine** — YAML-based policy engine with 10 condition types, 4 action types, 28-probe package risk scoring (A–F grades), and 3-tier MCP trust scoring with confused deputy mitigation; <1ms evaluation
- **Agent self-protection rules** — 18+ absolute deny rules compiled into a Go binary PreToolUse hook with fail-closed semantics (exit code 2), path canonicalization, symlink resolution, and gate-dodging detection
- **Interactive bypass system** — Configuration-guard rules with `qsdev hook bypass-next` for authorized overrides, shadow-mode calibration for safe rule rollout, circuit breaker pattern
- **SBOM and attestation** — Dual-format SBOMs (SPDX 2.3 + CycloneDX 1.5) via Syft, cosign keyless signing, govulncheck OpenVEX output, SLSA Build Level 2 compliance; `qsdev sbom` CLI command
- **Consolidated security binary** — Single `qsdev-hook` binary replacing Python hook scripts, with <50ms PreToolUse execution and advanced evasion detection (base64, shell expansion, indirect writes)
- **Hook execution sandboxing** — Bubblewrap-based 7-layer sandbox (user/mount/PID/network/IPC namespaces) with Landlock LSM and seccomp-BPF syscall filtering
- **OpenGrep custom rule library** — ~160–220 custom security rules (core taint-focused + pattern rules) with OpenGrep Nix packaging
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

- **Multi-framework shared primitives** — 7 Go interfaces (DetectionAdapter, ConfigRenderer, HookDeployer, etc.) abstracting all AI framework addons
- **Universal MCP server** — Single `qsdev mcp serve` command serving 8+ AI frameworks with CWD-based project detection and framework auto-detection
- **AI config portability** — `qsdev init --portable` generating AGENTS.md; `.qsdev/ai-config.yaml` canonical format compiling to 9 targets (Claude, AGENTS.md, Cursor, Windsurf, Copilot, Gemini CLI, Amazon Q/Kiro, Aider, Continue.dev)
- **LSP default configuration library** — Pre-configured LSP settings for 22 language servers, PreToolUse hook redirecting grep-on-symbols to LSP tools
- **LSP sandboxing** — `mkSandboxedLsp` Nix function wrapping LSP servers in bubblewrap+Landlock+seccomp with 25 per-server policies
- **Multi-agent coordination** — Git worktree isolation, append-only JSONL event chronicle, structured task lifecycle with first-claim-wins semantics
- **Agentic quality patterns** — `orient` skill (PageRank-ranked repo map), `learning-opportunities` skill, pre-edit linting on AI-written files
- **Agent secret proxy** — Lightweight Go proxy on Unix domain socket with in-memory vault and HTTP header injection (agent never sees credentials)
- **MCP gateway security** — Docker MCP Gateway interceptors for client-agnostic enforcement; single `policy.yaml` compiles to both hook and Gateway formats

## Cloud & Infrastructure

- **Cloud credential isolation** — Per-project AWS, GCP, and Azure CLI credential scoping with 3-layer fail-safe (environment separation, credential file masking, agent deny rules)
- **Kubernetes ecosystem modules** — kubectl, enhanced Helm module, kubescape security scanning with per-project KUBECONFIG enforcement
- **Expanded service detection** — Kafka, MinIO, Mailpit, Keycloak, NATS (11 total services, up from 6)
- **Container runtime modernization** — Podman rootless as default backend, `ContainerRuntime` Go abstraction, `qsdev container migrate` tooling

## Developer Experience

- **Non-language tool detection** — Git platform detection (GitHub/GitLab/Bitbucket), documentation generators, API frameworks, database migration tools
- **IDE and shell configuration** — EditorConfig generation, VS Code settings/extensions.json, Starship prompt configuration, shell fragment generation
- **Copier template integration** — `qsdev init --from` for templated project scaffolding with registry support
- **Config integrity analysis** — 7-format semantic diff engine (YAML, JSON, TOML, INI, Dockerfile, Shell, Nix) with ~50 threat detection rules and SARIF output
- **Config vault** — Git-based config versioning at `.qsdev/config-store/` with `qsdev vault snapshot/restore/diff/log`, environment branches, sops+age encryption

## MCP & Documentation

- **MCP server registry** — `McpServerRegistry` with auto-detection, 5-level compliance grading, and 40-tool ceiling; `qsdev mcp list/health/grade` commands
- **Local documentation pipeline** — Local-first docs via DevDocs, openzim-mcp, man-mcp-server with SKILL.md routing layer for documentation priority ordering

## Consulting & Team Management

- **Encrypted client profiles** — `qsdev profile create/switch/list/delete` with sops+age encryption for client secrets and profile-scoped compliance enforcement
- **Consulting enforcement hooks** — 6 hook configs: credential scanning, destructive-op prevention, SOC 2 audit logging, dependency oversight, file boundary enforcement, tool approval gates
- **Observability and analytics** — JSONL event hub, OpenTelemetry sidecar, ccusage integration for per-model cost tracking; `qsdev observe` CLI
- **Consulting lifecycle management** — 6-phase teardown protocol covering 10 artifact layers, tamper-evident audit logs, cost governance with Stop hook budget enforcement
- **Bot identity and team knowledge** — 5-layer bot identity verification, git-monorepo team vault distributing 10 functional knowledge packs
- **Team report enhancements** — CI artifact aggregation for multi-project dashboards, historical trend tracking

## Ecosystem Expansion

- **Ecosystem module hardening** — Production-quality 5-level confidence scoring, lockfile generation/enforcement, and expanded security configs for all 27 modules across 4 tiers
- **Niche tool integrations** — SpotBugs (Java static analysis), Dockle (container CIS), buf (Protobuf), Spectral (OpenAPI linting), Atlas (database migrations), Granted (AWS multi-account)
- **External addon ecosystem** — `qsdev addons` CLI (16 commands) for git-repo-as-index addon registry with exec-based plugin protocol
- **Code quality stack** — ast-grep structural search MCP server, codebase-memory-mcp call graph navigation, post-tool quality pipeline

## Distribution & Adoption

- **Cross-platform validation** — E2E testing of all 27 ecosystem modules across Linux, macOS, and Windows
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
