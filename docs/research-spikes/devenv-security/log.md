# Research Log: Devenv.sh Security Boilerplate

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Deep investigation into devenv.sh: how it works, how to configure a boilerplate default setup that includes package security, supply chain attack prevention, and protection against compromised packages being installed or run. The setup should work invisibly in the background, giving developers extra security without friction. Related spike: `package-supply-chain-security` covers general supply chain defenses across package managers — this spike focuses on devenv.sh as the delivery platform.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-12 — Cross-reference with package-supply-chain-security spike
- **Type**: analysis
- **Status**: success
- **Depth**: moderate
- **Summary**: Cross-referenced the package-supply-chain-security spike's task definitions and scope against devenv.sh's security needs. The source spike is still in early Phase 1 (no completed reports), so this is primarily a conceptual mapping. Identified 6 directly applicable research areas (signature verification → binary cache trust, private registries → Nix cache infra, quarantine → channel lag, org tooling → Nix-native scanners, lock files → flake.lock, install sandboxing → Nix build sandbox). Identified 7 gaps unique to devenv.sh (shell hook security, flake input trust, binary cache substitution, plugin security, FOD risks, pre-commit integrity, nix.conf trust boundaries). Documented key tensions between general supply chain best practices and Nix's model (different provenance standards, coarser lock granularity, FOD sandbox exceptions). Noted relevant CVEs: CVE-2024-27297, CVE-2024-38531, CVE-2026-39860.
- **Next**: Revisit this cross-reference as the source spike completes research tasks. Prioritize devenv-specific gaps (shell hooks, binary cache trust policy, flake input verification) that the general spike won't cover.

## 2026-05-12 14:30 — Prior Art & Community Practices Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Nix Attacker vs Defender](https://blog.devsecopsguides.com/p/nix-package-management-the-attacker) → `docs/nix-attacker-vs-defender-battlefield.md`
  - [devenv Sandbox PR #2427](https://github.com/cachix/devenv/pull/2427) → `docs/devenv-sandbox-pr-2427.md`
  - [NixOS Wiki Security](https://wiki.nixos.org/wiki/Security) → `docs/nixos-wiki-security.md`
  - [sbomnix README](https://github.com/tiiuae/sbomnix) → `docs/sbomnix-readme.md`
  - [Discourse: Supply Chain Attacks](https://discourse.nixos.org/t/is-nix-vulnerable-to-supply-chain-attacks/72411) → `docs/discourse-nix-supply-chain-attacks.md`
  - [Garnix: Stop Trusting Caches](https://garnix.io/blog/stop-trusting-nix-caches/) → `docs/garnix-stop-trusting-nix-caches.md`
  - [vulnix README](https://github.com/nix-community/vulnix) → `docs/vulnix-readme.md`
  - [nix-mineral README](https://github.com/cynicsketch/nix-mineral) → `docs/nix-mineral-readme.md`
  - [Discourse: Supply Chain Security Project](https://discourse.nixos.org/t/nixpkgs-supply-chain-security-project/34345) → `docs/discourse-nixpkgs-supply-chain-security-project.md`
  - [Discourse: State of SBOM](https://discourse.nixos.org/t/nix-state-of-the-sbom/73629) → `docs/discourse-nix-state-of-sbom.md`
  - [nix-security-tracker README](https://github.com/NixOS/nix-security-tracker/blob/main/README.md) → `docs/nix-security-tracker-readme.md`
  - [Pwning Nix Ecosystem](https://ptrpa.ws/nixpkgs-actions-abuse) → `docs/pwning-nix-ecosystem.md`
  - [devenv Auto-Activation](https://devenv.sh/auto-activation/) → `docs/devenv-auto-activation.md`
  - [devenv SecretSpec](https://devenv.sh/integrations/secretspec/) → `docs/devenv-secretspec.md`
  - [devenv Git Hooks](https://devenv.sh/git-hooks/) → `docs/devenv-git-hooks.md`
  - [devenv Options Reference](https://devenv.sh/reference/options/) → `docs/devenv-options-reference-security.md`
  - [devenv 2025 Blog Archive](https://devenv.sh/blog/archive/2025/) → `docs/devenv-2025-blog-archive.md`
  - [RFC 0062: Content-Addressed Paths](https://github.com/NixOS/rfcs/blob/master/rfcs/0062-content-addressed-paths.md) → `docs/rfc-0062-content-addressed-paths.md`
  - [Hardening NixOS Guide](https://saylesss88.github.io/nix/hardening_NixOS.html) → `docs/hardening-nixos-guide.md`
  - [Cloud Dev Env Security Comparison](https://northflank.com/blog/github-codespaces-alternatives) → `docs/cloud-dev-env-security-comparison.md`
  - [nixConfig Flake Security Risks](https://notashelf.dev/posts/reject-flake-content) → `docs/nixconfig-flake-security-risks.md`
  - [Nix Build Sandboxing](https://discourse.nixos.org/t/what-is-sandboxing-and-what-does-it-entail/15533) → `docs/nix-build-sandboxing-discourse.md`
- **Summary**: Comprehensive prior art survey across 8 research areas. Key finding: no hardened devenv.sh boilerplate exists anywhere in the community. Individual security capabilities are fragmented across tools (vulnix, sbomnix, SecretSpec, build sandboxing) but nobody has assembled them into a cohesive secure-by-default template. devenv.sh is actively developing security features (SecretSpec production since v1.8, sandbox PRs in draft). The broader Nix ecosystem has maturing supply chain infrastructure (security tracker operational since 2025, SBOM tooling functional). Three real-world incidents documented (GitHub Actions abuse, cache poisoning risks, nixConfig injection). Identified 6 immediately actionable hardening measures, 4 near-term measures, and 3 systemic improvements requiring ecosystem maturity.
- **Next**: Use findings to inform P1-T1 through P1-T4 research. Begin designing the actual boilerplate configuration in Phase 2.

## 2026-05-12 16:00 — Configuration Options Inventory Complete (P1-T4)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [devenv.nix options reference](https://devenv.sh/reference/options/) -> `docs/devenv-nix-options-reference.md`
  - [devenv.yaml options reference](https://devenv.sh/reference/yaml-options/) -> `docs/devenv-yaml-options-reference.md`
  - [devenv pre-commit hooks](https://devenv.sh/pre-commit-hooks/) -> `docs/devenv-pre-commit-hooks.md`
  - [devenv git hooks](https://devenv.sh/git-hooks/) -> `docs/devenv-git-hooks-configuration.md`
  - [devenv inputs](https://devenv.sh/inputs/) -> `docs/devenv-inputs-configuration.md`
  - [devenv processes](https://devenv.sh/processes/) -> `docs/devenv-processes-configuration.md`
  - [devenv containers](https://devenv.sh/containers/) -> `docs/devenv-containers-configuration.md`
  - [devenv scripts](https://devenv.sh/scripts/) -> `docs/devenv-scripts-configuration.md`
  - [devenv files and variables](https://devenv.sh/files-and-variables/) -> `docs/devenv-files-and-variables.md`
  - [devenv testing](https://devenv.sh/tests/) -> `docs/devenv-testing.md`
  - [devenv packages](https://devenv.sh/packages/) -> `docs/devenv-packages.md`
  - [devenv imports/composition](https://devenv.sh/composing-using-imports/) -> `docs/devenv-imports-composition.md`
  - [devenv flakes integration](https://devenv.sh/guides/using-with-flakes/) -> `docs/devenv-flakes-integration.md`
  - [devenv top-level module source](https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix) -> `docs/devenv-top-level-module.md`
  - [SecretSpec integration](https://devenv.sh/integrations/secretspec/) -> `docs/secretspec-integration.md`
  - [SecretSpec announcement](https://devenv.sh/blog/2025/07/21/announcing-secretspec-declarative-secrets-management/) -> `docs/secretspec-announcement.md`
  - [nix.conf security settings](https://nix.dev/manual/nix/2.19/command-ref/conf-file) -> `docs/nix-conf-security-settings.md`
  - [git-hooks.nix complete hook list](https://github.com/cachix/git-hooks.nix/blob/master/modules/hooks.nix) -> `docs/git-hooks-nix-complete-list.md`
- **Summary**: Complete survey of all devenv.sh configuration knobs relevant to security. Covered 9 sections: devenv.nix options (13 option groups: packages, env, enterShell, enterTest, scripts, dotenv, git-hooks, processes, containers, files, services, overlays, unsetEnvVars), devenv.yaml options (8 groups: inputs, nixpkgs controls, clean, impure, imports, secretspec, require_version, reload), Nix daemon settings (12 security settings: sandbox, restrict-eval, allowed-uris, trusted-substituters, trusted-public-keys, require-sigs, trusted-users, allowed-users, sandbox-paths, sandbox-fallback, filter-syscalls, secret-key-files), pre-commit hooks (120+ total, 6 security-relevant built-in, custom hook support for gitleaks/semgrep/tfsec), lock file analysis (pins inputs not packages, no CVE scanning), container/process isolation analysis (no runtime isolation for either), environment variable management (7-layer model, clean option critical), and execution model analysis (no sandboxing for any execution context). Key finding: devenv's security features are almost entirely opt-in. A hardened boilerplate must explicitly enable clean environments, ripsecrets hook, license blocklisting, secretspec, and stable nixpkgs channels. Provided complete hardened boilerplate example.
- **Next**: Complete remaining Phase 1 tasks (T1-T3). Use this inventory to inform the Phase 2 boilerplate design.

## 2026-05-12 17:00 — Nix Security Mechanisms Deep Dive Complete (P1-T3)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [nix.conf reference (2.28)](https://nix.dev/manual/nix/2.28/command-ref/conf-file) → `docs/nix-conf-security-options.md`
  - [CA derivations wiki](https://wiki.nixos.org/wiki/Ca-derivations) → `docs/ca-derivations-wiki.md`
  - [Nix multi-user mode](https://nix.dev/manual/nix/stable/installation/multi-user) → `docs/nix-multi-user-mode.md`
  - [Nix sandboxing discussion](https://discourse.nixos.org/t/what-is-sandboxing-and-what-does-it-entail/15533) → `docs/nix-sandboxing-discourse.md`
  - [Nix store paths (Nix Pills)](https://nixos.org/guides/nix-pills/18-nix-store-paths) → `docs/nix-store-path-computation.md`
  - [Nix security advisories](https://github.com/NixOS/nix/security/advisories) → `docs/nix-security-advisories.md`
  - [Nix experimental features](https://nix.dev/manual/nix/2.28/development/experimental-features) → `docs/nix-experimental-features.md`
  - [Flake lock file format](https://nix.dev/manual/nix/2.24/command-ref/new-cli/nix3-flake) → `docs/nix-flake-lock-format.md`
  - [NixOS security wiki](https://wiki.nixos.org/wiki/Security) → `docs/nixos-security-wiki.md`
  - [Determinate Systems security](https://manual.determinate.systems/installation/nix-security.html) → `docs/determinate-nix-security.md`
  - [Tweag untrusted CI caching](https://www.tweag.io/blog/2019-11-21-untrusted-ci/) → `docs/tweag-untrusted-ci-binary-cache.md`
  - [devenv flake.nix nixConfig](https://github.com/cachix/devenv/blob/main/flake.nix) → `docs/devenv-flake-nix-config.md`
  - [devenv binary caching](https://devenv.sh/binary-caching/) → `docs/devenv-binary-caching.md`
  - [Secure supply chains with Nix](https://nixcademy.com/posts/secure-supply-chain-with-nix/) → `docs/nixcademy-secure-supply-chain.md`
  - [Pwning the Nix ecosystem](https://ptrpa.ws/nixpkgs-actions-abuse) → `docs/pwning-nix-ecosystem.md`
- **Summary**: Investigated 9 Nix security mechanisms: (1) build sandbox -- Linux namespace isolation, on by default, FODs bypass network isolation, 3 historical sandbox escape CVEs; (2) CA derivations -- experimental (~65% stabilized), eliminates need for trusted sigs but not yet usable; (3) binary cache signatures -- Ed25519 verification, devenv adds 2 Cachix caches; (4) restricted/pure eval -- pure eval on by default with flakes, devenv supports impure override; (5) flake locks -- narHash SHA-256 integrity, nix flake update is the real attack surface; (6) store path integrity -- immutable paths with NAR hash, no runtime monitoring; (7) daemon trust -- trusted-users = root equivalent, devenv docs recommend it dangerously; (8) NixOS features -- AppArmor/SELinux not integrated, systemd hardening available; (9) recent improvements -- verified-fetches experimental, supply chain project funded. Critical finding: devenv's recommendation to add users to trusted-users is the single largest security regression in a typical setup. Hardening requires system-level nix.conf changes.
- **Next**: Complete P1-T1 (architecture) and P1-T2 (security surface). Then move to Phase 2 boilerplate design.

## 2026-05-12 18:00 — Security Attack Surface Deep Research Complete (P1-T2)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Nix Attacker vs Defender (DevSecOps Guides)](https://blog.devsecopsguides.com/p/nix-package-management-the-attacker) → `docs/nix-attacker-vs-defender-devsecopsguides.md`
  - [Nix Supply Chain Discourse](https://discourse.nixos.org/t/is-nix-vulnerable-to-supply-chain-attacks/72411) → `docs/nix-supply-chain-discourse.md`
  - [Binary Cache Signatures Discourse](https://discourse.nixos.org/t/what-guarantees-do-signatures-by-binary-caches-give/34802) → `docs/nix-binary-cache-signatures-discourse.md`
  - [Direnv .envrc TOCTOU Issue #445](https://github.com/direnv/direnv/issues/445) → `docs/direnv-envrc-toctou-issue-445.md`
  - [Devenv enterShell Fork Bomb Issue #2497](https://github.com/cachix/devenv/issues/2497) → `docs/devenv-entershell-fork-bomb-issue-2497.md`
  - [Nixpkgs Repository Security Discourse](https://discourse.nixos.org/t/security-of-nixpkgs-repository/15463) → `docs/nixpkgs-repository-security-discourse.md`
  - [Nixpkgs Trust Model Discourse](https://discourse.nixos.org/t/trust-model-for-nixpkgs/9450) → `docs/nixpkgs-trust-model-discourse.md`
  - [Env Vars World-Readable Advisory](https://discourse.nixos.org/t/security-advisory-environment-variables-accessible-during-a-build-might-be-world-readable/52601) → `docs/nix-env-vars-world-readable-advisory.md`
  - [Devenv Process Management Docs](https://devenv.sh/processes/) → `docs/devenv-process-management-docs.md`
  - [Devenv Bypass Cachix Issue #1658](https://github.com/cachix/devenv/issues/1658) → `docs/devenv-bypass-cachix-issue-1658.md`
  - [Devenv Installation Guide](https://devenv.sh/getting-started/) → `docs/devenv-installation-guide.md`
  - [Secure Supply Chain Nix Demo](https://github.com/applicative-systems/secure-supply-chain) → `docs/secure-supply-chain-nix-demo.md`
  - [Trustix Distributed Trust (Tweag)](https://www.tweag.io/blog/2020-12-16-trustix-announcement/) → `docs/trustix-distributed-trust-tweag.md`
  - [Nix Advanced Attributes](https://nix.dev/manual/nix/2.18/language/advanced-attributes) → `docs/nix-advanced-attributes-security.md`
  - [Nix Sandboxing Discourse](https://discourse.nixos.org/t/what-is-sandboxing-and-what-does-it-entail/15533) → `docs/nix-sandboxing-discourse.md`
  - [Devenv Scripts Docs](https://devenv.sh/scripts/) → `docs/devenv-scripts-docs.md`
  - [Nix Flake Checker (Determinate)](https://determinate.systems/blog/flake-checker/) → `docs/nix-flake-checker-determinate.md`
  - [Devenv Git Hooks Docs](https://devenv.sh/git-hooks/) → `docs/devenv-git-hooks-docs.md`
  - [Devenv Direnv Integration Docs](https://devenv.sh/integrations/direnv/) → `docs/devenv-direnv-integration-docs.md`
  - [Devenv Binary Caching Docs](https://devenv.sh/binary-caching/) → `docs/devenv-binary-caching-docs.md`
- **Summary**: Complete threat model covering 10 major attack vectors with 25+ sub-vectors across the full devenv.sh stack. Vectors mapped: (1) package source attacks including malicious overlays, nixpkgs compromise, and typosquatting; (2) binary cache poisoning including key compromise, misconfiguration, provenance gaps, and third-party trust; (3) shell hook injection via enterShell, scripts, git hooks, and re-evaluation loops; (4) module/plugin system trust boundaries and unsandboxed evaluation; (5) direnv integration risks including devenv.nix change bypass and TOCTOU; (6) build-time code execution including FOD network access and post-build hooks; (7) flake input manipulation including lock tampering, follows substitution, registry attacks, and --override-input; (8) environment variable leakage including store world-readability and env-vars advisory; (9) process/service management with no isolation; (10) devenv's own supply chain. Three cross-cutting architectural observations: (a) Nix evaluation is unsandboxed -- the single most important security fact; (b) trust model is "trust on first allow" with no per-change review for devenv.nix modifications; (c) all defenses converge on code review as the primary control point. Report at `security-surface-research.md`.
- **Next**: All Phase 1 tasks complete. Begin Phase 2: design hardened boilerplate configuration.

## 2026-05-12 19:00 — Architecture & Internals Research Complete (P1-T1)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [devenv 2.0 blog post](https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/) -> `docs/devenv-2-0-blog-post.md`
  - [devenv 1.3 caching](https://devenv.sh/blog/2024/10/03/devenv-13-instant-developer-environments-with-nix-caching/) -> `docs/devenv-1-3-caching-architecture.md`
  - [devenv 1.0 Rust rewrite](https://devenv.sh/blog/2024/03/20/devenv-10-rewrite-in-rust/) -> `docs/devenv-1-0-rust-rewrite.md`
  - [devenv 1.1 module system](https://devenv.sh/blog/2024/09/11/devenv-11-nested-nix-outputs-using-the-module-system/) -> `docs/devenv-1-1-module-system.md`
  - [devenv GitHub README](https://github.com/cachix/devenv) -> `docs/devenv-github-readme.md`
  - [devenv basics](https://devenv.sh/basics/) -> `docs/devenv-basics.md`
  - [devenv files and variables](https://devenv.sh/files-and-variables/) -> `docs/devenv-files-and-variables.md`
  - [devenv inputs](https://devenv.sh/inputs/) -> `docs/devenv-inputs.md`
  - [devenv binary caching](https://devenv.sh/binary-caching/) -> `docs/devenv-binary-caching.md`
  - [devenv direnv integration](https://devenv.sh/integrations/direnv/) -> `docs/devenv-direnv-integration.md`
  - [devenv using with flakes](https://devenv.sh/guides/using-with-flakes/) -> `docs/devenv-using-with-flakes.md`
  - [devenv composing imports](https://devenv.sh/composing-using-imports/) -> `docs/devenv-composing-imports.md`
  - [devenv-nixpkgs repo](https://github.com/cachix/devenv-nixpkgs) -> `docs/devenv-nixpkgs-repo.md`
  - [devenv.lock example](https://github.com/NixOS/20th-nix/blob/main/devenv.lock) -> `docs/devenv-lock-example.md`
  - [Discourse: devenv vs services-flake](https://discourse.nixos.org/t/devenv-vs-services-flake-vs/59074) -> `docs/discourse-devenv-vs-services-flake.md`
- **Summary**: Comprehensive architecture analysis covering all 10 research questions in a 13-section report at `architecture-research.md`. Key findings: devenv is a Rust CLI + NixOS module system wrapper; v2.0 uses C FFI (nix-bindings-rust) for per-attribute incremental caching with sub-100ms activation; devenv.yaml controls inputs/imports/nixpkgs-config while devenv.nix defines the environment via module options; default package source is devenv-nixpkgs/rolling (Cachix-maintained fork with less community review); devenv.lock uses flake lock format v7 pinning inputs but not individual packages; no separate plugin API (everything is NixOS modules); binary cache trust is signature-based with devenv.cachix.org auto-trusted; runtime is completely unsandboxed; devenv requires impure evaluation; clean shell and impure mode are primary isolation controls. Includes detailed comparison table with nix develop, lorri, nix-direnv, and services-flake.
- **Next**: All Phase 1 tasks now complete. Begin Phase 2 boilerplate design.

## 2026-05-12 — Phase 1 Complete — Synthesis of Key Findings
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: All 6 Phase 1 tasks completed successfully. 88 source documents saved to docs/, 6 research reports written. Key findings that drive Phase 2 boilerplate design:
  1. **No hardened devenv boilerplate exists anywhere** — confirmed gap, this is novel work
  2. **Nix evaluation is unsandboxed** — most critical architectural fact; malicious .nix code runs with full user privileges before any sandbox applies
  3. **`trusted-users` is root-equivalent** — devenv recommends this; must be replaced with `trusted-substituters` + `trusted-public-keys`
  4. **`devenv-nixpkgs/rolling` is a Cachix-maintained fork** with less review than upstream nixpkgs
  5. **Nearly all security features are opt-in** with permissive defaults
  6. **No runtime sandboxing** for dev shell sessions (sandbox only applies during nix-build)
  7. **`devenv.local.nix` can override all security controls** without team visibility
  8. **Shell hooks (enterShell) run with full user privileges** outside any sandbox
  9. **`devenv.lock` pins nixpkgs commits, not individual packages** — no per-package CVE scanning
  10. **Binary cache trust is single-key, all-or-nothing** for third-party caches
- **Next**: Define Phase 2 tasks targeting boilerplate design, system-level hardening guide, pre-commit security hooks, and trust model documentation.

## 2026-05-12 — Trust Model Documentation Complete (P2-T4)
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Wrote the developer-facing trust model document at `trust-model-research.md`. The document covers 5 sections: (1) enumeration of 8 trust dependencies in plain language (Nix daemon, devenv binary, nixpkgs, binary caches, flake inputs, devenv.nix, devenv modules, direnv); (2) verification matrix showing what is verified, what is missing, and what an attacker needs for each dependency; (3) code review checklist for devenv.nix, devenv.yaml, devenv.lock, and .envrc changes with concrete examples of suspicious patterns; (4) red flag table with 3 tiers (block PR, elevated scrutiny, information-only) covering binary cache additions, impure mode, fetchurl in .nix, overlays from external sources, shell hooks downloading content, and more; (5) mapping from hardened boilerplate choices back to specific threats (clean env vs credential leakage, impure:false vs eval-time exfiltration, secretspec vs store-readable secrets, upstream nixpkgs vs fork trust, git-hooks vs committed secrets). Synthesized from all 6 Phase 1 research reports without requiring additional web fetches -- all source material was already in docs/.
- **Next**: Continue with remaining Phase 2 tasks (P2-T1 boilerplate, P2-T2 nix.conf hardening, P2-T3 pre-commit hooks, P2-T5 vulnerability scanning, P2-T6 runtime isolation).

## 2026-05-12 — Hardened Boilerplate Design Complete (P2-T1)
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [NixOS 25.11 stable release](https://status.nixos.org/) -> channel verification
  - [devenv 2.1 release](https://devenv.sh/blog/2026/05/07/devenv-21-nix-with-zsh-fish-and-nushell-via-libghostty/) -> version verification
  - [devenv top-level.nix unsetEnvVars](https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix) -> `docs/devenv-top-level-nix-unsetenvvars.md`
  - [devenv.yaml options reference](https://devenv.sh/reference/yaml-options/) -> `docs/devenv-yaml-options-complete-2026.md`
- **Summary**: Designed the centerpiece deliverable: a production-ready security-hardened devenv.sh boilerplate consisting of 4 files (devenv.yaml, devenv.nix, devenv.local.nix.example, .envrc) plus supplementary secretspec.toml example and companion nix.conf requirements. The boilerplate is fully self-documenting with inline comments mapping each setting to specific threat model vectors from `security-surface-research.md`. Key design decisions: (1) Pin to upstream `nixos-25.11` instead of `devenv-nixpkgs/rolling` — broader community review over cache convenience; (2) `clean.enabled: true` with explicit keep-list — strips ambient credentials by default; (3) 35+ credential variables in `unsetEnvVars` as second-layer defense; (4) `dotenv.enable = false` with SecretSpec as the replacement; (5) 7 git hooks including 2 custom (lock-file-audit, nix-secrets-check); (6) `enterTest` assertions for CI validation; (7) 3 security utility scripts using Nix store path references to prevent PATH hijacking. Every setting classified as MUST-HAVE (cannot weaken without team review), RECOMMENDED (should have, low friction), or OPTIONAL (extra hardening). Explicit "what this does NOT protect against" section covers 6 limitations. Full report at `boilerplate-research.md`.
- **Next**: Run revision cycle on the boilerplate report. Continue with P2-T2 (nix.conf), P2-T3 (pre-commit suite), P2-T5 (vulnerability scanning), P2-T6 (runtime isolation).

## 2026-05-12 — System-Level nix.conf Hardening Guide Complete (P2-T2)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [nix.conf reference (2.28)](https://nix.dev/manual/nix/2.28/command-ref/conf-file) -> `docs/nix-conf-reference-2-28-hardening-settings.md`
  - [nix.conf per-user limitations](https://nix.dev/manual/nix/2.28/command-ref/conf-file) -> `docs/nix-conf-per-user-limitations.md`
  - Phase 1 sources reused: `docs/nix-conf-security-options.md`, `docs/nix-conf-security-settings.md`, `docs/garnix-stop-trusting-nix-caches.md`, `docs/nix-multi-user-mode.md`, `docs/nix-sandboxing-discourse.md`, `docs/nixconfig-flake-security-risks.md`, `docs/determinate-nix-security.md`, `docs/tweag-untrusted-ci-binary-cache.md`, `docs/devenv-binary-caching.md`, `docs/devenv-flake-nix-config.md`
- **Summary**: Wrote comprehensive system-level nix.conf hardening guide at `nix-conf-hardening-research.md`. Covers 10 settings in depth: (1) sandbox + sandbox-fallback -- why relaxed mode and silent fallback are dangerous, what breaks on strict enforcement; (2) require-sigs -- signature enforcement, what happens on unsigned packages; (3) trusted-users -- why devenv's recommendation is root-equivalent, detailed privilege escalation chain; (4) trusted-substituters -- explicit allowlist of 3 caches (cache.nixos.org, devenv.cachix.org, cachix.cachix.org) with signing keys and justification; (5) trusted-public-keys -- key management and compromise response; (6) allowed-users -- daemon access control; (7) restrict-eval + allowed-uris -- evaluation-time access control, recommended for CI but not workstations due to devenv's impure eval requirement; (8) filter-syscalls -- seccomp filtering for setuid/xattr prevention; (9) extra-sandbox-paths -- why to keep empty, legitimate use cases table; (10) connect-timeout + download-attempts -- cache exposure limits. Provides three complete configuration formats: NixOS module (nix.settings), standalone nix.conf (non-NixOS Linux, macOS), and per-user nix.conf (~/.config/nix/nix.conf) with explicit tables of what can vs cannot be set per-user. Includes deployment checklist and summary reference table.
- **Next**: Continue with P2-T3 (pre-commit hooks), P2-T5 (vulnerability scanning), P2-T6 (runtime isolation).

## 2026-05-12 — Pre-Commit Security Hook Suite Research Complete (P2-T3)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [ripsecrets](https://github.com/sirwart/ripsecrets) → `docs/ripsecrets-readme.md`, `docs/ripsecrets-benchmarks.md`
  - [gitleaks](https://github.com/gitleaks/gitleaks) → `docs/gitleaks-readme.md`
  - [trufflehog](https://github.com/trufflesecurity/trufflehog) → `docs/trufflehog-readme.md`
  - [detect-secrets](https://github.com/Yelp/detect-secrets) → `docs/detect-secrets-readme.md`
  - [semgrep](https://github.com/semgrep/semgrep) → `docs/semgrep-readme.md`
  - [flake-checker](https://github.com/DeterminateSystems/flake-checker) → `docs/flake-checker-github-readme.md`
  - [git-hooks.nix custom hook schema](https://flake.parts/options/git-hooks-nix.html) → `docs/git-hooks-nix-custom-hook-schema.md`
  - Phase 1 sources reused: `docs/git-hooks-nix-complete-list.md`, `docs/vulnix-readme.md`, `docs/nix-flake-checker-determinate.md`, `docs/devenv-git-hooks-docs.md`, `docs/devenv-pre-commit-hooks.md`
- **Summary**: Comprehensive pre-commit security hook suite documented at `precommit-hooks-research.md`. Covers 10 research questions across 11 sections. Key findings: (1) Only ripsecrets is a built-in security hook in git-hooks.nix; all others (gitleaks, trufflehog, semgrep, bandit, gosec, grype, vulnix, flake-checker) require custom hook definitions. (2) Secret scanner comparison: ripsecrets is 95x faster than trufflehog and 226x faster than detect-secrets -- recommended as commit-time default with gitleaks at pre-push and trufflehog at CI-only for verification. (3) No built-in lock file audit hook exists; designed a custom hook that detects owner changes and new inputs in devenv.lock/flake.lock. (4) Dependency vuln scanning (vulnix 15-60s, grype 30-45s) is too slow for commit-time -- CI-only. (5) SAST tools: bandit (1-5s) and gosec (2-10s) are commit-time viable; semgrep (5-30s) is pre-push. (6) License compliance: reuse hook is built-in; Nix license blocklisting in devenv.yaml is stronger than hooks (cannot be bypassed with --no-verify). (7) flake-checker validates branch support, recency (30-day), and upstream ownership with CEL policy customization. (8) Complete custom hook attribute reference with 20+ attributes and 14 valid stages documented. (9) Performance classification for 17 hooks across 3 tiers (commit-time <5s, pre-push <30s, CI-only unlimited). (10) --no-verify cannot be blocked client-side; defense-in-depth requires 5 layers from local hooks through server-side pre-receive; devenv auto-reinstalls hooks on shell entry. Complete hardened hook suite devenv.nix provided combining all recommendations.
- **Next**: Continue with P2-T5 (vulnerability scanning), P2-T6 (runtime isolation).

## 2026-05-12 — Runtime Isolation Options Research Complete (P2-T6)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [devenv PR #2427 detailed](https://github.com/cachix/devenv/pull/2427) → `docs/devenv-sandbox-pr-2427-detailed.md`
  - [devenv PR #1783 (Landlock)](https://github.com/cachix/devenv/pull/1783) → `docs/devenv-landlock-pr-1783.md`
  - [nix-sandbox (bwrap tool)](https://github.com/fabian-thomas/nix-sandbox) → `docs/nix-sandbox-bubblewrap-tool.md`
  - [bubblewrap shell tutorial](https://sloonz.github.io/posts/sandboxing-1/) → `docs/bubblewrap-sandboxing-shell-tutorial.md`
  - [bubblewrap-claude reference](https://github.com/matgawin/bubblewrap-claude) → `docs/bubblewrap-claude-code-sandbox.md`
  - [devenv containers docs](https://devenv.sh/containers/) → `docs/devenv-containers-docs-detailed.md`
  - [devenv devcontainer integration](https://devenv.sh/integrations/codespaces-devcontainer/) → `docs/devenv-codespaces-devcontainer-integration.md`
  - [devcontainer module source](https://github.com/cachix/devenv/blob/main/src/modules/integrations/devcontainer.nix) → `docs/devenv-devcontainer-module-source.md`
  - [Landlock overview](https://landlock.io/) → `docs/landlock-unprivileged-sandboxing.md`
  - [landrun CLI](https://github.com/Zouuup/landrun) → `docs/landrun-landlock-sandbox-tool.md`
  - [systemd sandboxing (Red Hat)](https://www.redhat.com/en/blog/mastering-systemd) → `docs/systemd-sandboxing-redhat.md`
  - [systemd zero-code sandboxing (Cloudflare)](https://blog.cloudflare.com/sandboxing-in-linux-with-zero-lines-of-code/) → `docs/systemd-zero-code-sandboxing-cloudflare.md`
  - [Firejail on NixOS](https://wiki.nixos.org/wiki/Firejail) → `docs/firejail-nixos-wiki.md`
- **Summary**: Evaluated 7 runtime isolation approaches for devenv.sh: (1) PR #2427 (bubblewrap, whole-shell) -- draft since Jan 2026, unresolved design debate with Landlock approach, breaks shell customization, no reviews; (2) PR #1783 (Landlock, per-executable) -- draft since Mar 2025, author abandoned, better UX but direnv compatibility blocker due to Landlock's monotonic restriction property; (3) Manual bubblewrap wrapping -- functional today with working example provided, breaks direnv/SSH/shell configs; (4) Devcontainer generation -- production-ready, `devcontainer.enable = true` generates .devcontainer.json but with zero security hardening, isolation depends on container runtime; (5) systemd --user -- CRITICAL FINDING: PrivateTmp/ProtectSystem/ProtectHome DO NOT WORK with user services, only NoNewPrivileges/SystemCallFilter/MemoryDenyWriteExecute are available; (6) unshare/namespaces -- same primitives as bwrap, less ergonomic, no advantage; (7) Firejail -- requires NixOS system config, SUID, not portable. Additionally discovered landrun (Landlock CLI wrapper) as a strong candidate for immediate use. Wrote comparison matrix and 4-tier recommendation: include today (hardened devcontainer, bwrap script, landrun script, NoNewPrivileges), track for future (native PRs, Landlock network), exclude (Firejail, raw unshare, systemd FS isolation). Full report at `runtime-isolation-research.md`.
- **Next**: Continue with P2-T5 (vulnerability scanning). Runtime isolation section feeds into boilerplate design.

## 2026-05-12 — Vulnerability Scanning Integration Research Complete (P2-T5)
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [vulnix README (full)](https://github.com/nix-community/vulnix) -> `docs/vulnix-readme-full.md`
  - [vulnix manpage](https://github.com/nix-community/vulnix/blob/master/doc/vulnix.1.md) -> `docs/vulnix-manpage.md`
  - [vulnix whitelist format](https://github.com/nix-community/vulnix/blob/master/doc/vulnix-whitelist.5.md) -> `docs/vulnix-whitelist-format.md`
  - [vulnix releases](https://github.com/nix-community/vulnix/releases) -> `docs/vulnix-releases.md`
  - [vulnix introduction (Flying Circus)](https://flyingcircus.io/en/about-us/blog-news/details-view/introducing-vulnix-a-vulnerability-scanner-for-nixos) -> `docs/vulnix-flyingcircus-introduction.md`
  - [sbomnix README (full)](https://github.com/tiiuae/sbomnix) -> `docs/sbomnix-readme-full.md`
  - [sbomnix vulnxscan docs](https://github.com/tiiuae/sbomnix/blob/main/doc/vulnxscan.md) -> `docs/sbomnix-vulnxscan-docs.md`
  - [ghafscan daily scanning](https://github.com/tiiuae/ghafscan) -> `docs/ghafscan-daily-vuln-scanning.md`
  - [Trivy NixOS support issue](https://github.com/aquasecurity/trivy/issues/1673) -> `docs/trivy-nixos-support-issue-1673.md`
  - [Trivy supply chain compromise](https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23) -> `docs/trivy-supply-chain-compromise-2026.md`
  - [Grype README](https://github.com/anchore/grype) -> `docs/grype-readme.md`
  - [flake-checker README (full)](https://github.com/DeterminateSystems/flake-checker) -> `docs/flake-checker-readme-full.md`
  - [nix-security-tracker interface](https://tracker.security.nixos.org/) -> `docs/nix-security-tracker-interface.md`
  - [nix-security-tracker contributing](https://github.com/NixOS/nix-security-tracker/blob/main/CONTRIBUTING.md) -> `docs/nix-security-tracker-contributing.md`
  - [Discourse: vuln scanning with Nix SBOMs](https://discourse.nixos.org/t/how-to-do-vulnerability-scanning-with-nix-sboms/66161) -> `docs/discourse-vuln-scanning-nix-sboms.md`
  - [devenv GitHub Actions integration](https://devenv.sh/integrations/github-actions/) -> `docs/devenv-github-actions-integration.md`
- **Summary**: Deep investigation of 6 vulnerability scanning tools for devenv.sh integration plus CI pipeline design and scanning strategy. Key findings: (1) vulnxscan (from sbomnix) is the recommended primary CI scanner -- combines vulnix + grype + OSV with multi-scanner consensus scoring; (2) flake-checker is the only tool fast enough for commit-time hooks (<5s); (3) Trivy does not support Nix store scanning (issue #1673 closed/stale) and was itself compromised in a supply chain attack March 2026 -- use only for container scanning if at all; (4) No tool can detect language-level package vulns (Python, Node) in Nix closures -- fundamental PURL specification gap; (5) nix-security-tracker has no public API -- not directly integrable; (6) Grype consumes sbomnix CycloneDX SBOMs but misses Nix-specific packages due to PURL gap; (7) ghafscan provides the reference architecture for production daily scanning. Designed 4-tier scanning strategy (commit <5s, pre-push <30s, CI 5-15min, nightly 15-40min) with concrete devenv.nix, GitHub Actions, and GitLab CI pipeline examples. Full report at `vuln-scanning-research.md`.
- **Next**: Integrate vulnerability scanning recommendations into the hardened boilerplate. All Phase 2 research tasks now complete.

## 2026-05-12 — Depth Checklist Review Complete (P3-T1)
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Reviewed all 12 research reports against the 6-item depth checklist. Results: 7 full passes, 5 with 1-2 PARTIAL items (minor alternatives comparisons). Zero failures, zero substantive contradictions between reports. Cross-report consistency verified across 4 axes: boilerplate↔config-options, hooks↔boilerplate, nix.conf↔mechanisms, trust-model↔threat-model. One cosmetic inconsistency: nixpkgs version varies between examples (24.11 vs 25.11). No additional research needed. Report at `depth-review-research.md`.
- **Next**: Write final conclusions in research.md.

## 2026-05-12 — Spike Complete — Final Conclusions Written (P3-T2)
- **Type**: decision
- **Status**: success
- **Depth**: deep
- **Summary**: Wrote final conclusions in research.md answering the original research question. The hardened devenv.sh boilerplate is a 6-layer defense-in-depth strategy: input hardening (upstream nixpkgs), environment isolation (clean shell + unsetEnvVars), secrets management (SecretSpec), automated scanning (pre-commit hooks + CI pipelines), system hardening (nix.conf), and runtime isolation (devcontainers/landrun — future native sandbox). Key architectural limitation: Nix evaluation is unsandboxed, so code review remains the primary security boundary. This is confirmed novel work — no hardened devenv.sh boilerplate existed prior. Spike produced 13 research reports, 129 source documents, all passing depth review.
- **Next**: Spike ready for closure via `/complete-spike`.

## 2026-05-12 — Spike Closed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized and closed. All 14 tasks across 3 phases completed successfully. 13 research reports produced, 129 source documents saved, 12/12 topic reports pass depth checklist. Conclusions written answering the original research question with a 6-layer defense-in-depth boilerplate strategy. 4 open questions resolved during research, 4 remaining (native sandbox timeline, Landlock/direnv compat, Nix PURL gap, devenv-nixpkgs fork evolution). 3 follow-on candidates flushed to `proposed-spikes.md`: boilerplate empirical validation, native sandbox PR tracking, and Nix PURL language-level vulnerability scanning gap. Spike moved from `active-spikes.md` to `completed-spikes.md`.
