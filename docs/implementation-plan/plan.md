# Implementation Plan: gdev Secure Development Environment Bootstrap

## Overview

Build three gdev addons (`devenv`, `claudecode`, `devinit`) that enable any developer at the company to run a single command — `qsdev init` — and get a fully configured, security-hardened development environment. The generated configuration provides defense-in-depth against supply chain attacks through six layers: age-gating on all package managers, install script blocking, lock file enforcement, vulnerability scanning, PreToolUse hook enforcement in Claude Code, and hardened Nix evaluation settings.

The system covers **27 language/platform ecosystems** organized into 4 priority tiers, from must-ship (JS/TS, Python, Go, Rust, Java/Kotlin, .NET, Docker, Terraform) through commonly-encountered (PHP, Ruby, Scala, Helm, Ansible, Bash/Shell) to specialized and rare. Each ecosystem is a self-contained module with detection heuristics, devenv.nix templates, security hardening configs, and pre-commit hooks. A **profile system** encodes organization-wide infrastructure choices (registry proxy, Nix cache, build cache, scanning tools) so teams adopt the full stack in one command.

Rather than building security enforcement from scratch, the Claude Code addon integrates the **existing ecosystem**: attach-guard for package guardrails, Trail of Bits skills for security auditing, the official Claude Code Security Review GitHub Action for CI, and managed settings for enterprise enforcement. The addon curates, configures, and deploys these tools — it doesn't replace them.

This plan synthesizes findings from four completed research spikes plus three ecosystem research investigations into a phased implementation sequence. The addons are pure Go, built against gdev's existing addon framework with no framework modifications required.

## Research Foundation

| Spike / Report | Contribution |
|---|---|
| `research-spikes/gdev-extension-design/research.md` | Addon architecture (3-addon split), wizard UX (huh forms), template engine (per-format strategies), migration strategy (hash tracking + per-file merge) |
| `research-spikes/gdev-extension-design/addon-architecture-design.md` | Three-addon composition model, config key namespacing, command hierarchy, inter-addon communication, profile system |
| `research-spikes/gdev-extension-design/devenv-addon-design.md` | 5 bootstrap steps, 4 commands, YAML/Nix generation, 3 project-type templates |
| `research-spikes/gdev-extension-design/claude-code-addon-design.md` | 7 bootstrap steps, 5 commands, 5-layer defense architecture, skill library, deny rules |
| `research-spikes/gdev-extension-design/wizard-flow-integration-design.md` | huh form construction, progressive disclosure, detection engine, merge mode |
| `research-spikes/gdev-extension-design/config-template-engine-design.md` | Per-format generation (text/template for Nix, struct marshaling for YAML/JSON), atomic writes, validation |
| `research-spikes/gdev-extension-design/migration-strategy-design.md` | SHA256 hash tracking, per-file merge strategies, section markers, three-way merge |
| `research-spikes/package-supply-chain-security/research.md` | Per-ecosystem attack surface, age-gating (92% PyPI malware removed within 24h), 6-step hardening roadmap, tool recommendations |
| `research-spikes/package-supply-chain-security/quarantine-gates-research.md` | .npmrc, pip.conf, pnpm-workspace.yaml, .yarnrc.yml, bunfig.toml age-gating configs |
| `research-spikes/package-supply-chain-security/lockfile-integrity-research.md` | Lock file enforcement configs per ecosystem, CI frozen-install commands |
| `research-spikes/package-supply-chain-security/install-sandboxing-research.md` | Install script blocking configs, @lavamoat/allow-scripts, pnpm strictDepBuilds |
| `research-spikes/devenv-security/research.md` | 6-layer defense-in-depth for devenv.sh, hardened boilerplate (4 files), trust model |
| `research-spikes/devenv-security/boilerplate-research.md` | Hardened devenv.yaml, devenv.nix, .envrc, nix.conf concrete configurations |
| `research-spikes/devenv-security/nix-conf-hardening-research.md` | 10 nix.conf security settings, 3 deployment formats |
| `research-spikes/devenv-security/precommit-hooks-research.md` | 17 pre-commit hooks in 3 tiers (baseline/enhanced/specialized) |
| `research-spikes/devenv-security/trust-model-research.md` | Trust boundaries, cache verification, input pinning |
| `research-spikes/claude-code-agent-package-guardrails/research.md` | 5-layer defense (hooks + deny rules + OS config + CLAUDE.md + skills), bypass vectors |
| `research-spikes/claude-code-agent-package-guardrails/unified-architecture.md` | Complete guardrail specification (1,727 lines), 3 deployment profiles |
| `research-spikes/claude-code-agent-package-guardrails/reference-hook-script.py` | PreToolUse hook with OSV.dev + age checking + safety flags |
| `research-spikes/claude-code-agent-package-guardrails/reference-deny-rules.md` | 48 deny rules covering 15+ package managers |
| `artifacts/language-ecosystem-coverage.md` | 27 ecosystems, 59 devenv.sh modules, per-ecosystem security configs, 4-tier priority matrix |
| `artifacts/claude-code-ecosystem-research.md` | attach-guard, Security Phoenix, Trail of Bits skills/config, marketplace plugins, managed settings |
| `artifacts/artifact-stores-caches-research.md` | Registry proxies, Nix caches, build caches, SBOM tools, recommended $0/mo consulting stack |
| `artifacts/cross-platform-distribution-research.md` | GoReleaser config, install scripts (bash + PowerShell), self-bootstrapping patterns, OS detection in Go, package manager abstraction, self-update mechanism |
| `artifacts/os-prerequisite-detection-research.md` | OS/distro detection matrix (12 families), tool prerequisite mapping (13 tools × 12 package managers), shell integration, privilege escalation, Windows/WSL2 considerations |
| `artifacts/agent-postmortem-skill-SKILL.md` | Prompt-based verification protocol: 4-step pipeline (Intent → Evidence → Verification → Postmortem), anti-fake-done guardrails |
| `artifacts/version-sentinel-*.md/json` | Claude Code plugin: PreToolUse hooks blocking unverified dependency changes, sidecar state, 5-ecosystem coverage (npm, pip, pyproject, cargo, csproj) |
| `artifacts/semble-*.md` | Semantic code search MCP server: tree-sitter AST chunking, hybrid BM25/semantic search, 98% token savings, Python >=3.10 |
| `artifacts/cross-platform-testing-infrastructure-research.md` | VM/container automation comparison, GitHub Actions CI matrix (3-tier), Docker image tags for 11 distros, WSL2 testing via setup-wsl, macOS/Windows runner strategies, cost estimates |
| `artifacts/e2e-test-automation-framework-research.md` | testscript framework, golden files, non-interactive patterns (--answers-file), BATS/Pester for install scripts, state management testing, coverage collection, prior art (Terraform, GoReleaser, mise, rustup) |
| `artifacts/language-ecosystem-test-targets-research.md` | New project creation steps + real GitHub repos for all 27 ecosystems, 5 polyglot combo repos, detection signal summary, PM conflict resolution rules |
| `artifacts/tool-lifecycle-conflict-matrix-research.md` | 17-tool × 22-file ownership matrix, per-tool enable/disable operations, 80+ test scenarios, shared file empty-state analysis, 8 open design decisions, 30+ edge cases |
| `artifacts/security-defense-validation-research.md` | Safe test fixtures for 10 defense layers, 30+ known-vulnerable package/CVE combinations, EICAR equivalents per layer, local registry setup (Verdaccio/devpi), hook test harnesses, Nix hardening derivations, OWASP alignment |
| `artifacts/phases-13-16-enhancement-research.md` | 5-spike synthesis: project configuration patterns (three-layer config, onboarding modes), Claude Code skill/agent format, compliance posture scoring, DX polish recommendations, 10+ rejected features with rationale |
| `research-spikes/gdev-claude-code-integration/` | Claude Code skill format, dynamic context injection, 10 gdev operation skills mapping, 5-layer safety architecture, CLI wrapper patterns |
| `research-spikes/gdev-agentic-workflows/` | 7 consulting agents + 15 skills catalog, agent file format, context budget management, deny rule conflict validation, consulting-specific workflow differentiation |
| `research-spikes/gdev-health-reporting/` | Scorecard-inspired posture scoring, 6-category drift detection, machine-readable output (JSON/SARIF), team CI aggregation, badge generation, prior art (OpenSSF Scorecard, npm audit, cargo audit) |
| `research-spikes/gdev-team-config-onboarding/` | Three-layer config pattern, four onboarding modes, Terraform-style version constraints, `qsdev check` CI enforcement, client profiles with compliance levels, consulting lifecycle (teardown/archive/evidence) |
| `research-spikes/gdev-dx-polish/` | qsdev repair/info/outdated/update/teardown design, git workflow automation, shell integration, task runner rejection (devenv 2.0 native), 10 rejected features with rationale |

## Design Principles

1. **Security by default, not by opt-in.** Every generated config starts hardened. Developers loosen restrictions explicitly, not the reverse. Age-gating, install script blocking, and lock file enforcement are on by default for every ecosystem that supports them.

2. **Unobtrusive defense-in-depth.** Six defense layers work independently — if one is bypassed, five remain. The developer's normal workflow is unchanged; security enforcements happen in the background.

3. **One command, working environment.** `qsdev init` detects project type, generates all config files, and produces a working `devenv shell` in under 60 seconds. The wizard asks 1 question on the quick path, 5 form groups for customizers.

4. **Curate, don't reinvent.** Integrate existing best-in-class tools (attach-guard for package guardrails, Trail of Bits for security audit skills, OSV Scanner for vulnerability detection, Renovate for dependency updates) rather than building custom implementations. The addon's value is curation, configuration, and deployment — not reimplementation.

5. **Format-matched generation.** Struct marshaling for YAML/JSON/XML (prevents syntax bugs), text/template for Nix/Markdown (conditional content), embed.FS copy for skills/rules (no templating needed).

6. **Re-runnable without destruction.** SHA256 hash tracking distinguishes machine-owned files from human-edited files. devenv.nix is never auto-overwritten. CLAUDE.md uses section markers. settings.json uses three-way merge.

7. **Ecosystem module architecture.** Each language/platform is a self-contained module with: detection heuristics, devenv.nix template fragment, security hardening configs, pre-commit hooks, and CI commands. New ecosystems are added by implementing the module interface — no changes to core code.

8. **Profile-driven infrastructure.** Organization-wide choices (registry proxy, Nix cache, build cache, scanning tools) are encoded in profiles. Teams adopt the full stack via `qsdev init --profile consulting-default --yes`. Individual developers never need to know the infrastructure details.

9. **Single binary, zero prerequisites.** gdev ships as a static Go binary (`CGO_ENABLED=0`) with all templates, skills, and completions embedded via `embed.FS`. Installation requires only curl (Unix) or PowerShell (Windows). The binary self-bootstraps everything else via `qsdev devenv setup`.

10. **Platform-agnostic with platform-aware execution.** Core logic is cross-platform Go. Platform-specific code (OS detection, package managers, privilege escalation) lives behind interfaces with build-tagged implementations. The tool works on macOS, Windows (native + WSL2), and all major Linux distributions without recompilation.

11. **Detect, don't assume.** Before installing or configuring anything, detect what's already present. `qsdev devenv doctor` reports system state; `qsdev devenv setup` acts only on gaps. Never overwrite user-installed tools or assume a specific environment.

12. **AI agent tools are opt-in enhancements.** Agent-postmortem-skill, Version-Sentinel, and semble are deployed by the Claude Code addon as optional enhancements with smart defaults. Each has clear value (verification rigor, version guardrails, search efficiency) and zero-to-low overhead. They complement rather than conflict with the security hardening stack.

13. **Every tool is individually toggleable.** `qsdev enable <tool>` adds it; `qsdev disable <tool>` cleanly removes it — including shared-file sections in devenv.nix, settings.json, CLAUDE.md, and .mcp.json. Tool adoption is a reversible, low-risk decision that encourages experimentation. File ownership tracking ensures no orphaned artifacts.

14. **Every defense is provably working.** Each security layer has safe test fixtures that trigger detection (positive control) and legitimate operations that must not be blocked (negative control). Defenses are validated by intentional triggering — not by assumption. Modeled on the EICAR test file principle.

15. **Self-operating through AI agents.** gdev deploys Claude Code skills and agents that let Claude Code invoke gdev commands, follow consulting best practices, and automate common workflows. Developers say "set up this repo" and the AI does it. Safety layers ensure side-effect operations require human confirmation while read-only diagnostics are Claude-invocable.

## Ecosystem Coverage

### Tier 1 — Must Ship (per-language config generators + templates)
JavaScript/TypeScript (npm, pnpm, yarn, bun), Python (pip, uv, poetry), Go, Rust (Cargo), Java/Kotlin (Maven, Gradle), C#/.NET (NuGet, dotnet), Docker/Containerfiles (Hadolint), Terraform/OpenTofu

### Tier 2 — Should Ship (commonly encountered on client engagements)
PHP (Composer), Ruby (Bundler), Scala (sbt), Helm, Ansible (Galaxy), Bash/Shell (shellcheck, shfmt), C/C++ (Conan, vcpkg, CMake)

### Tier 3 — Nice to Have (specialized ecosystems)
Elixir (Mix/Hex), Dart/Flutter (pub), Swift (SPM), Haskell (Cabal, Stack), Clojure (deps.edn), Bazel (bzlmod), Nix (flake inputs)

### Tier 4 — Reference Docs Only (rare but documented)
Perl (Carton), R (renv), Lua (LuaRocks), Zig, PowerShell (PSGallery), Groovy, F#, Objective-C, WASM, Pulumi

### Claude Code Security Ecosystem Integration

| Tool | Type | Integration |
|------|------|-------------|
| **attach-guard** | PreToolUse hook plugin | Reference architecture; fork/extend or configure directly |
| **Security Phoenix** | Skills + hooks suite | Reference for full lifecycle (SessionStart → SessionEnd) |
| **Trail of Bits skills** | 40+ professional skills | Embed `supply-chain-risk-auditor`, `differential-review`, `insecure-defaults` |
| **Trail of Bits claude-code-config** | Production defaults | Reference for generated settings.json |
| **Claude Code Security Review** | GitHub Action | Include in generated CI workflows |
| **SonarQube plugin** | Marketplace | Configure when available |
| **Aikido plugin** | Marketplace | Configure when available |
| **Socket.dev MCP** | MCP server | Configure in .mcp.json |
| **Managed settings** | Enterprise enforcement | Template for `/etc/claude-code/managed-settings.json` |

### AI Agent Enhancement Tools

| Tool | Type | Integration |
|------|------|-------------|
| **agent-postmortem-skill** | Prompt-based skill (SKILL.md) | Embed in `.claude/skills/agent-postmortem/`, template with per-ecosystem verification commands |
| **Version-Sentinel** | Claude Code plugin (PreToolUse hooks) | Plugin marketplace install, configurable freshness window, per-project ignore file |
| **semble** | MCP server / sub-agent | MCP config in `.mcp.json` or sub-agent in `.claude/agents/semble-search.md` |

### Infrastructure Stack (Consulting Firm Profile — $0/mo)

| Layer | Tool | Why |
|-------|------|-----|
| Registry proxy | Nexus Community + Socket Firewall Free | Free multi-ecosystem proxy + malicious blocking |
| Nix binary cache | Cachix (free tier) or Attic (self-hosted) | Shared derivations across team |
| Compilation cache | sccache (S3 backend) | Multi-language, cloud-native |
| Monorepo cache | Turborepo (Vercel) or Nx Cloud (free) | Free managed caching |
| Vulnerability scanning | OSV Scanner + Socket.dev (free tier) | Zero-day detection + CVE scanning |
| Dependency updates | Renovate | 90+ ecosystems, policy-as-code, age-gating |
| CI protection | Harden-Runner (community) | CI runtime monitoring |
| SBOM | Syft + sbomnix | Container + Nix coverage |

## Phase Index

| # | Phase | Status | Dependencies | Summary |
|---|-------|--------|--------------|---------|
| 1 | Foundation & Shared Infrastructure | Not Started | None | Go module setup, shared types, ecosystem module interface, detection engine, template engine, atomic write pipeline, hash tracking |
| 2 | Ecosystem Modules — Tier 1 | Not Started | Phase 1 | 8 ecosystem modules (JS/TS, Python, Go, Rust, Java/Kotlin, .NET, Docker, Terraform) with devenv.nix templates, security configs, pre-commit hooks |
| 3 | devenv Addon — Core Generation | Not Started | Phases 1, 2 | devenv.yaml/devenv.nix/.envrc generation composing ecosystem modules, hardened defaults from devenv-security spike, CLI commands |
| 4 | Claude Code Addon — Core Generation | Not Started | Phases 1, 2 | settings.json with deny rules, CLAUDE.md with section markers, hook deployment (attach-guard + custom), skill/rule library (Trail of Bits + custom), .mcp.json, CLI commands |
| 5 | Security & Infrastructure Integration | Not Started | Phases 3, 4 | Per-ecosystem package manager hardening configs, registry proxy config, build cache config, Nix cache config, pre-commit hook suite, CI scanning workflows, nix.conf hardening, SBOM generation |
| 6 | Wizard & Orchestration (devinit) | Not Started | Phases 3, 4 | huh wizard forms, quick path + customize, detection pre-population, profile system (including infrastructure profiles), plan preview, merge mode, non-interactive/CI flags |
| 7 | Ecosystem Modules — Tiers 2-4 | Not Started | Phase 2 | 19 additional ecosystem modules (PHP, Ruby, Scala, C/C++, Helm, Ansible, Bash, Elixir, Dart, Swift, Haskell, Clojure, Bazel, Nix, Perl, R, Lua, Zig, PowerShell) |
| 8 | Migration, Update & Polish | Not Started | Phases 3-6 | `qsdev init --update`, three-way merge, section markers, team standards versioning, integration tests, documentation |
| 9 | Cross-Platform System Detection | Not Started | Phase 1 | OS detection engine (12 distro families), package manager abstraction (12 managers), tool prerequisite detection, shell integration, privilege escalation, `qsdev devenv doctor`, `qsdev devenv setup` |
| 10 | Distribution & Self-Bootstrapping | Not Started | Phases 1, 9 | GoReleaser multi-platform builds, install scripts (bash + PowerShell), Homebrew/Scoop/APT/RPM packaging, self-update, shell completions, `qsdev version` |
| 11 | AI Agent Tooling Integration | Not Started | Phases 2, 4, 6 | agent-postmortem-skill (templated verification), Version-Sentinel (plugin + config), semble (MCP/sub-agent), wizard integration, per-ecosystem coverage registries |
| 12 | Extended Integrations & Lifecycle | Not Started | Phases 1-8, 9, 11 | Tool lifecycle management (`qsdev enable/disable/status/list`), Semgrep (SAST), Gitleaks (secrets), Grype+Syft+Cosign (containers), ScanCode (licenses), SecretSpec (dev secrets), CI workflow generation, Context7 MCP, git-cliff (changelog), retroactive lifecycle for Phase 4/11 tools |
| 13 | Project Configuration & Team Standards | Not Started | Phases 1, 6, 8, 12 | `.qsdev.yaml` project config file, three-layer resolution (binary → project → local), four onboarding modes (Create/Join/Update/Repair), `qsdev check` CI enforcement, config versioning, client-specific profiles with compliance levels |
| 14 | Claude Code Integration & Agentic Skills | Not Started | Phases 4, 11, 12, 13 | 10 gdev operation skills (6 user-only, 4 Claude-invocable), 7 consulting agents, 8+ workflow skills, context budget management, deny rule conflict validation, devenv task definitions, CLAUDE.md enhancement |
| 15 | Health, Status & Compliance Reporting | Not Started | Phases 1, 12, 13, 14 | `qsdev status` with scoring (0-100, A-F, conformance labels), 6-category drift detection, `qsdev evidence` compliance reports, machine-readable output (JSON/SARIF), badge generation, team aggregation pipeline |
| 16 | Developer Experience Polish | Not Started | Phases 3, 9, 10, 12, 13, 15 | `qsdev repair` self-healing, `qsdev info`, `qsdev outdated`, `qsdev update` coordinated updates, `qsdev teardown` clean exit, git workflow automation, shell & environment integration |
| 16.1 | Structured Logging, Error Reporting & Bug Feedback | Not Started | None (can be built independently) | Two-tier structured logging (`log/slog` JSONL to `.qsdev/logs/` + `~/.qsdev/logs/`), `RedactingHandler` for privacy-safe logs, `--debug` flag, `QSDEV_LOG` env var, `qsdev logs` management commands, `qsdev report bug` wizard with GH Issue submission, external tool log ingestion (npm, nix, devenv) with provider interface, `CaptureWriter` for ephemeral tool output |
| 17 | Test Infrastructure & Framework | Not Started | Phase 1, 10, 6 | CI pipeline (3-tier: PR/nightly/release), testscript E2E framework, BATS + Pester install script tests, `--non-interactive` + `--answers-file` flags, golden files, coverage collection, build-once-test-many artifacts |
| 18 | Cross-Platform Installation Validation | Not Started | Phases 17, 9, 10 | Install script E2E on 24 OS targets, `qsdev devenv doctor` validation across all distro families, `qsdev devenv setup --dry-run` verification, package manager distribution testing (Homebrew/Scoop/APT/RPM), self-update + shell completions |
| 19 | Ecosystem & Configuration Onboarding Validation | Not Started | Phases 17, 1-8, 6, 13 | Greenfield `qsdev init` for all 27 ecosystems, brownfield onboarding against 14+ real open-source repos, polyglot project composition, detection accuracy, `.qsdev.yaml` Join mode testing |
| 20 | Tool Lifecycle & Integration Validation | Not Started | Phases 17, 12, 4, 11, 13-16 | 80+ test scenarios: individual lifecycle round-trips (16 tools), shared file integrity (6 formats), idempotency, cross-tool interactions, wizard integration, migration/upgrade, new command validation (check/status/repair/info/outdated/update/teardown) |
| 21 | Security Defense Validation | Not Started | Phases 17, 4-5, 12 | Safe test fixtures for 10 defense layers: age-gating (Verdaccio), script blocking (@lavamoat canary), lock file enforcement, vuln scanning, PreToolUse hooks, Nix hardening, Semgrep SAST, Gitleaks secrets, container security, license compliance |
| 22 | Agentic Skills, Compliance & DX Validation | Not Started | Phases 17, 13-16 | Configuration onboarding modes, skill/agent file validation, deny rule conflict matrix, health/compliance reporting accuracy, DX command validation (repair/info/outdated/update/teardown), team workflow testing |

## Validation-Informed Adjustments

Four parallel validation agents verified all research findings on 2026-05-12. Full results in `artifacts/validation-findings.md`. Key adjustments incorporated:

1. **gdev bootstrap already uses charmbracelet/huh** — validates our wizard library choice. New env detection helpers (`SkipInContainer`, `SkipIfNoGUI`) are available for bootstrap steps.
2. **devenv 2.0 requires explicit git-hooks input** — must generate `inputs.git-hooks.url` in devenv.yaml whenever hooks are enabled.
3. **prek replaces pre-commit** as default hook runner in devenv 1.11+. Same config, different binary.
4. **Use Python PreToolUse hook script**, not bash version — bash CVSS parsing is broken.
5. **Fix npm age check** to use `time[dist-tags.latest]` instead of `time.modified`.
6. **Fix OSV queries** to include version when available.
7. **Replace Phylum with Socket.dev** in all tool recommendations.
8. **Use individual hook entries** in settings.json, not `||`-compound `if` field.
9. **NixOS 25.11 approaching EOL** (2026-06-30) — default to `nixos-26.05` when available.
10. **Trivy supply chain compromise (March 2026)** — reference in generated security docs.
11. **Composer 2.9 blocks vulnerable packages by default** — strongest built-in defense of any package manager; document and leverage.
12. **JVM needs dual config generators** — Maven `settings.xml` + Gradle `verification-metadata.xml` are separate code paths.
13. **Curate existing Claude Code security tools** — attach-guard, Trail of Bits, Security Phoenix are reference implementations; integrate rather than rebuild.
14. **artifact-keeper** is promising but too new (early 2026) for production recommendation — recommend Nexus Community as default, artifact-keeper as watch item.

### Phase 9-11 Research Findings (2026-05-12)

Five parallel research agents investigated cross-platform distribution, OS prerequisite detection, and three AI agent tools. Key findings:

15. **Ship static binaries, not source.** `CGO_ENABLED=0` produces binaries with zero runtime dependencies. GoReleaser handles the full matrix (5 targets). The "first tool" bootstrap problem is solved: gdev doesn't need Go to run — it IS a Go binary.
16. **Rustup-style two-stage bootstrap is the pattern.** Thin install script downloads pre-built binary; binary does all real work. `curl | sh` (Unix) and `irm | iex` (Windows) are the primary distribution channels. Package managers (Homebrew, Scoop, APT/RPM) are secondary.
17. **OS detection is 40 lines, not a library.** Custom `/etc/os-release` parsing + `runtime.GOOS` + WSL2 detection covers all cases. Third-party OS detection libraries are a supply chain risk for trivial code.
18. **WSL2 is the Windows answer for Nix.** Nix does not run natively on Windows. When Nix-dependent features are needed, gdev should detect WSL2 and offer `wsl --install` if missing.
19. **Batch elevated operations.** A single `sudo apt-get install -y git go nodejs direnv` is better than 4 separate sudo prompts. The privilege escalation layer collects all system packages before prompting.
20. **agent-postmortem-skill is prompt-only (3.6KB SKILL.md).** No code, no dependencies, MIT licensed. Value-add: gdev templates it with project-specific verification commands per detected ecosystem.
21. **Version-Sentinel covers 5 of 8 Tier 1 ecosystems.** npm, pip, pyproject.toml, Cargo.toml, .csproj are covered. go.mod, pom.xml, build.gradle are NOT covered. This gap must be documented clearly in generated CLAUDE.md sections.
22. **semble requires Python >=3.10 and uvx.** Integration is lightweight (MCP config or sub-agent file) but adds a Python runtime prerequisite. Should be gated on Python availability in the wizard.
23. **All three tools are MIT-licensed and very new** (2-25 days old, 2-798 stars). Integration should be modular — easy to swap out if better alternatives emerge or if these projects are abandoned.

### Phase 12 Research Findings (2026-05-12)

Eight parallel agents (4 gap analysis, 3 ecosystem research, 1 tool synthesis) produced 4,263 lines of analysis. Key findings:

24. **No tool lifecycle management exists.** `qsdev init` is a one-way door — there's no `qsdev disable <tool>` or `qsdev enable <tool>`. Developers can't experiment with tools without committing to manual cleanup. This is the highest-priority gap for Phase 12.
25. **Shared-file surgery is the hard problem.** Toggling a tool requires editing its sections in devenv.nix, settings.json, CLAUDE.md, .mcp.json, and CI workflows — files that multiple tools contribute to. Section markers (CLAUDE.md, devenv.nix) and structured parsing (JSON, YAML) are the two strategies. CI workflows are fully regenerated rather than surgically edited.
26. **Trivy compromised March 2026, KICS compromised April-May 2026.** Both by "TeamPCP" threat actor. Grype replaces Trivy as default container scanner. Checkov replaces KICS for IaC scanning. Pin all CI tools by SHA hash.
27. **Context7 (55K stars) is the highest-impact MCP server** for a consulting firm — provides version-specific library docs for 50K+ libraries, preventing stale API hallucination across unfamiliar client stacks.
28. **3-6 MCP servers is the sweet spot.** More than 10 slows agents without proportional benefit. Default to context7 + github + detected ecosystem servers.
29. **Semgrep CE is the only $0 SAST covering all 27 ecosystems.** LGPL-2.1, 11K stars, 3,000+ community rules. CodeQL requires $49/user/month for private repos.
30. **ScanCode Toolkit for license compliance.** Apache-2.0, audit-grade accuracy, used by Eclipse Foundation. Too slow for pre-commit but appropriate for weekly CI.
31. **SecretSpec ships with devenv 2.0.** Apache-2.0, declarative secrets with provider abstraction (keyring/dotenv/env/1Password). SOPS not yet supported as provider.
32. **18 tools explicitly rejected** with specific reasons: CodeQL (licensing), Snyk (pricing), Trivy (compromised), FOSSA (SaaS), CodeRabbit/Qodo (per-seat SaaS), SOPS (competes with SecretSpec), release-please/semantic-release (Node.js deps), SonarQube/Aikido (server infra), configuration drift detection (solved by Nix/devenv), local service orchestration (devenv 2.0 native).

### Phase 17-21 Research Findings (Test Infrastructure) (2026-05-12)

Five parallel research agents produced 6,300+ lines of analysis covering cross-platform testing infrastructure, language ecosystem test targets, tool lifecycle conflict matrices, security defense validation, and E2E test automation frameworks. Key findings:

33. **Docker containers cover 90% of Linux testing.** Containers on GitHub Actions (`container:` directive) test OS detection, package managers, and binary installation with 1-5s startup. Only login-shell verification, systemd, and reboot-persistence need full VMs. Vagrant is effectively dead for this use case (outdated boxes, no ARM64).
34. **24 OS configurations cover all target families.** 5 native runners (macOS ARM64/Intel, Windows, Ubuntu amd64/arm64) + 11 Docker containers (Debian, Ubuntu, Fedora, Rocky, Alma, Arch, openSUSE TW/Leap, Alpine, Void, Gentoo) + WSL2 + NixOS. Derivative distros (Pop!_OS, Manjaro, EndeavourOS, Garuda) safely skipped via `ID_LIKE` detection. Linux Mint included as most divergent Ubuntu derivative.
35. **WSL2 testable in CI.** `Vampire/setup-wsl@v4` on `windows-2025` runner supports Ubuntu, Debian, Fedora, Alpine inside WSL2. No self-hosted runners needed.
36. **testscript is the Go-native E2E framework.** Roger Peppe's txtar-based engine (same behind Go's 900+ script tests) provides platform conditions, custom commands, golden files, and coverage integration. Custom commands (`yaml_has`, `json_path`, `nix_valid`) extend it for gdev's file formats.
37. **`--answers-file` is the single most important testing enabler.** Bypasses TUI wizard while exercising the same config logic. All E2E tests use it. teatest reserved for wizard UI testing only.
38. **BATS for bash, Pester for PowerShell.** Native test frameworks for each install script language. Both produce JUnit XML for CI integration. @lavamoat/preinstall-always-fail is the EICAR equivalent for install script blocking.
39. **80+ tool lifecycle test scenarios identified.** Complete tool × file matrix (17 tools × 22 files), 8 open design decisions, 30+ edge cases. No explicit tool conflicts exist (by design), but pre-commit hook ordering (ripsecrets → gitleaks → semgrep) must be deterministic.
40. **Safe test fixtures exist for all 10 defense layers.** Verdaccio (npm age-gating), devpi (Python), corrupted lockfiles (8 ecosystems), known-CVE manifests (30+ package/version/CVE combinations), piped-JSON hook harness, Nix sandbox-escape derivations, Semgrep `# ruleid:` annotations, AWS example keys (AKIAIOSFODNN7EXAMPLE) as Gitleaks EICAR, local OCI registry for Cosign round-trips.
41. **Cost: ~$2.93/PR for full matrix on private repos.** 5 native + 11 container jobs × 5 min each. macOS concurrency (5 jobs max on free plan) is the binding constraint. At 20 PRs/week: ~$240/month.
42. **Three-tier CI trigger strategy.** Every PR: Tier 1 native + Tier 2 containers (~5 min). Nightly: full Tier 3 extended matrix (~15 min). Release: everything including package manager installs (~20 min). Balances developer velocity against coverage.

### Phase 13-16 Enhancement Research Findings (2026-05-12)

Five parallel research agents investigated project configuration patterns, Claude Code skill/agent integration, health and compliance reporting, agentic workflow patterns, and developer experience polish. Key findings:

43. **Three-layer config with security floor.** Binary defaults → `.qsdev.yaml` (project, checked into git) → `.qsdev.local.yaml` (developer, gitignored). Security level acts as a floor — local overrides cannot weaken project security policy. Pattern validated by EditorConfig, ESLint flat config, Renovate shared presets, and mise's `.mise.local.toml`.
44. **Four onboarding modes eliminate guesswork.** `qsdev init` auto-detects mode: Create (fresh project), Join (existing .qsdev.yaml, new developer), Update (newer gdev available), Repair (config drift detected). Join mode targets under 2 minutes from git clone to productive environment.
45. **Skills over commands.** Claude Code `.claude/skills/` with YAML frontmatter supersede the legacy `.claude/commands/` format. Skills support directories for supporting files, full frontmatter control (disable-model-invocation, allowed-tools, arguments), and auto-invocation. gdev deploys skills exclusively.
46. **Dynamic context injection is the key pattern.** The `!`qsdev devenv doctor --json`` preprocessor in skill files runs shell commands before Claude sees the content, injecting live system state. This makes gdev skills context-aware without stale documentation.
47. **Five-layer safety for agentic operations.** (1) Skill-level `disable-model-invocation` for side-effect operations, (2) `allowed-tools: Bash(gdev *)` scoping, (3) gdev `--dry-run` and `--non-interactive` flags, (4) Claude Code permission system (deny rules, auto mode), (5) enterprise hooks via managed settings.
48. **Scorecard-inspired posture scoring.** Three-layer model: defense coverage (40%), configuration health (30%), dependency health (30%). Produces both numeric score (0-100, A-F grades) and conformance labels (baseline/enhanced PASS/FAIL). Versioned JSON schema from day one (lesson from cargo-audit instability).
49. **Six-category drift detection in under 100ms.** All local checks: file modification (SHA256), version drift, tool availability, section marker integrity, lock file drift, pre-commit hook drift. No network calls for baseline assessment.
50. **devenv 2.0 task system eliminates task runner need.** devenv has parallel execution, dependency ordering, lifecycle hooks, and caching built in. Adding just/Taskfile/mise as task runners would duplicate existing capability. gdev generates devenv task definitions per ecosystem instead.
51. **Conservative repair, never destructive.** `qsdev repair` backs up before any change, never auto-modifies user-edited devenv.nix (generates .new + diff instead), and uses `--dry-run` preview by default. Recovery from 4 failure categories: Nix/devenv, config corruption, tool failures, environment drift.
52. **Ten features explicitly rejected.** Task runner (devenv native), container management (Docker/Podman exist), CI execution, deployment, code scaffolding, IDE config beyond Claude Code, OTEL infrastructure (just env vars), package manager installation (qsdev devenv setup), Git server API, vulnerability database. Three-test framework: purpose-built tool exists? file generation or runtime? compounds with existing features?

## Current Status
No implementation work has started. Phase 1 is the entry point. The plan encompasses 22 phases: 16 development phases (1-16) and 6 validation phases (17-22).
