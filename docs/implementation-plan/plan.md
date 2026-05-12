# Implementation Plan: gdev Secure Development Environment Bootstrap

## Overview

Build three gdev addons (`devenv`, `claudecode`, `devinit`) that enable any developer at the company to run a single command — `gdev init` — and get a fully configured, security-hardened development environment. The generated configuration provides defense-in-depth against supply chain attacks through six layers: age-gating on all package managers, install script blocking, lock file enforcement, vulnerability scanning, PreToolUse hook enforcement in Claude Code, and hardened Nix evaluation settings.

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

## Design Principles

1. **Security by default, not by opt-in.** Every generated config starts hardened. Developers loosen restrictions explicitly, not the reverse. Age-gating, install script blocking, and lock file enforcement are on by default for every ecosystem that supports them.

2. **Unobtrusive defense-in-depth.** Six defense layers work independently — if one is bypassed, five remain. The developer's normal workflow is unchanged; security enforcements happen in the background.

3. **One command, working environment.** `gdev init` detects project type, generates all config files, and produces a working `devenv shell` in under 60 seconds. The wizard asks 1 question on the quick path, 5 form groups for customizers.

4. **Curate, don't reinvent.** Integrate existing best-in-class tools (attach-guard for package guardrails, Trail of Bits for security audit skills, OSV Scanner for vulnerability detection, Renovate for dependency updates) rather than building custom implementations. The addon's value is curation, configuration, and deployment — not reimplementation.

5. **Format-matched generation.** Struct marshaling for YAML/JSON/XML (prevents syntax bugs), text/template for Nix/Markdown (conditional content), embed.FS copy for skills/rules (no templating needed).

6. **Re-runnable without destruction.** SHA256 hash tracking distinguishes machine-owned files from human-edited files. devenv.nix is never auto-overwritten. CLAUDE.md uses section markers. settings.json uses three-way merge.

7. **Ecosystem module architecture.** Each language/platform is a self-contained module with: detection heuristics, devenv.nix template fragment, security hardening configs, pre-commit hooks, and CI commands. New ecosystems are added by implementing the module interface — no changes to core code.

8. **Profile-driven infrastructure.** Organization-wide choices (registry proxy, Nix cache, build cache, scanning tools) are encoded in profiles. Teams adopt the full stack via `gdev init --profile consulting-default --yes`. Individual developers never need to know the infrastructure details.

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
| 8 | Migration, Update & Polish | Not Started | Phases 3-6 | `gdev init --update`, three-way merge, section markers, team standards versioning, integration tests, documentation |

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

## Current Status
No work has started. Phase 1 is the entry point.
