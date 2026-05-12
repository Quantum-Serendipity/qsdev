# Cross-Reference: Package Supply Chain Security Findings Applied to Devenv.sh

**Source spike**: `research-spikes/package-supply-chain-security/`
**Target spike**: `research-spikes/devenv-security/`
**Date**: 2026-05-12

## Status of Source Spike

The package-supply-chain-security spike is in early Phase 1 with five active research tasks and no completed reports yet. Tasks in progress cover: ecosystem attack surface landscape, private registries & validated mirrors, publication age / quarantine gates, signature verification & provenance, and organizational tooling & policy enforcement. Two tasks (lock file integrity, install script sandboxing) remain pending.

This cross-reference is therefore primarily a **mapping of conceptual applicability** rather than a summary of completed findings. It identifies which of the general spike's research areas will be directly useful to devenv-security, what gaps exist, and where Nix's security model creates tensions with general supply chain best practices. This document should be revisited as the source spike produces completed research reports.

## Directly Applicable Findings (When Completed)

### 1. Signature Verification & Provenance → Binary Cache Trust

**Source task**: "Signature verification & provenance" (SLSA, Sigstore, npm provenance, etc.)
**Devenv application**: Nix binary caches use Ed25519 key-based signing. Devenv.sh integrates with Cachix for binary caching and configures `extra-trusted-public-keys` and `extra-substituters` in the flake's `nixConfig`. The general spike's findings on provenance attestation (SLSA levels, Sigstore) map to Nix's binary cache verification, but Nix uses its own signing mechanism rather than Sigstore/SLSA.

**What to pull in**: General principles of multi-party trust and provenance verification. The Nix-specific implementation is [Trustix](https://github.com/nix-community/trustix) — a Merkle-tree append-only log that enables M-of-N consensus across independent builders, decentralizing binary cache trust beyond a single signing key.

**Key tension**: The general spike assumes registry-level provenance (e.g., npm provenance linking to a GitHub Actions build). Nix's model is fundamentally different — provenance is tied to the derivation (build recipe) and its content hash, not to a registry attestation. This is both a strength (derivation pinning is more deterministic) and a weakness (trust still rests on whoever controls the binary cache signing keys).

### 2. Private Registries & Validated Mirrors → Nix Binary Cache Infrastructure

**Source task**: "Private registries & validated mirrors" (Artifactory, Verdaccio, Devpi, etc.)
**Devenv application**: The Nix equivalent is hosting your own binary cache (via `nix-serve`, Attic, Cachix, or S3-compatible storage). A security-hardened devenv boilerplate should configure which substituters are trusted and enforce signature verification.

**What to pull in**: General patterns for organizational proxy/cache architecture — the principle of a validated intermediary between upstream and developers applies regardless of ecosystem. Specific product recommendations (Artifactory, etc.) are less relevant since Nix has its own cache ecosystem.

**Devenv-specific configuration**:
- `trusted-substituters` in `nix.conf` (system-level): only listed caches can be used
- `trusted-public-keys`: enforce that all cached artifacts are signed by known keys
- `cachix.enable` in devenv.nix: controls automatic Cachix integration
- Devenv auto-configures cache settings, which is convenient but means the project's `devenv.nix` can modify which binary caches a developer trusts — a potential vector if the project config is compromised

### 3. Publication Age / Quarantine Gates → Nixpkgs Channel Lag

**Source task**: "Publication age / quarantine gates" (Socket.dev hold-back, registry quarantine)
**Devenv application**: Nix already has a natural quarantine effect. Nixpkgs packages go through a review process, Hydra builds, and channel promotion before reaching `nixpkgs-unstable` or stable channels. This delay (typically days to weeks) acts as a passive quarantine.

**What to pull in**: The concept of intentional hold-back periods. For devenv.sh, pinning `nixpkgs` to a specific commit (which flake lock files do by default) and only updating deliberately achieves a similar effect to registry quarantine. The general spike's findings on optimal quarantine periods and detection windows will help calibrate how aggressively to pin vs. update.

**Key tension**: Quarantine is inherent in Nix channel promotion, but developers using `nixpkgs-unstable` or following HEAD get less quarantine benefit. Devenv.sh defaults to nixpkgs-unstable — the boilerplate should consider whether to default to a more conservative channel.

### 4. Organizational Tooling & Policy Enforcement → Nix-Native Equivalents

**Source task**: "Organizational tooling & policy enforcement" (Socket.dev, Snyk, OSV Scanner, etc.)
**Devenv application**: Most general-purpose scanners (Snyk, Dependabot) don't understand Nix derivations. Nix-specific alternatives include:
- [sbomnix](https://github.com/tiiuae/sbomnix): generates SBOMs from Nix targets for vulnerability scanning
- `vulnix`: scans Nix closures against NVD/CVE databases
- Devenv's built-in `pre-commit` hooks can integrate security scanning tools

**What to pull in**: The organizational patterns (CI gates, pre-commit checks, automated scanning) translate directly even though the specific tools differ. The general spike's framework for evaluating scanning tools can be applied to Nix-specific equivalents.

### 5. Lock File Integrity → Flake Lock Files

**Source task**: "Lock file integrity & reproducible installs" (pending)
**Devenv application**: Nix flake.lock files pin every input to a specific content hash and git revision. This is significantly stronger than most language-ecosystem lock files because Nix's content-addressed store means the locked hash covers the entire dependency closure, not just direct dependencies.

**What to pull in**: Best practices for lock file hygiene (reviewing diffs, CI enforcement, detecting unexpected changes). The general spike's findings on `--frozen-lockfile` enforcement map to Nix's `--no-update-lock-file` flag.

**Key difference**: Nix lock files pin flake inputs (nixpkgs commit, devenv source, etc.), not individual packages. This is coarser-grained but more holistic — you can't change one package without changing the entire nixpkgs commit. Whether this is better or worse depends on threat model.

### 6. Install Script Sandboxing → Nix Build Sandboxing

**Source task**: "Install script sandboxing & runtime protections" (pending)
**Devenv application**: Nix builds run in a sandbox by default (network-isolated, filesystem-isolated, no access to `/home`, `/tmp` cleaned). This is far stronger than anything available in npm/pip/cargo ecosystems. However, Fixed-Output Derivations (FODs) — used for fetching sources — have network access by design and are a known weak point.

**What to pull in**: The general principle that install-time code execution is a primary attack vector. Nix's sandbox is stronger by default, but the FOD exception and shell hooks in devenv.sh (which run outside the sandbox, in the developer's shell) need specific hardening attention.

## Gaps: What Devenv.sh Needs That the General Spike Won't Cover

### G1: Shell Hook Security
Devenv.sh executes shell hooks (`enterShell`, `scripts`, custom commands) in the developer's user context with full system access. These are not sandboxed. A compromised devenv.nix could exfiltrate credentials, modify files, or install backdoors via shell hooks. The general supply chain spike doesn't cover this because it's unique to Nix/devenv.

### G2: Flake Input Trust Chain
Devenv.sh pulls its own source, nixpkgs, and any additional flake inputs from Git repositories. A compromised flake input is analogous to a compromised package registry, but the trust model is different — you're trusting specific Git commits rather than registry-published versions. Flake input poisoning (via compromised upstream repos or GitHub account takeover) is a Nix-specific attack vector not covered by the general spike.

### G3: Binary Cache Substitution Attacks
If a developer's machine trusts a malicious binary cache, pre-built derivations could be substituted with backdoored versions. The general spike covers registry compromise, but Nix binary cache trust is architecturally different — the `trusted-substituters` and `trusted-public-keys` settings in `nix.conf` are the defense surface, and devenv.sh projects can request additional caches be trusted.

### G4: Devenv Plugin/Module Security
Devenv supports community modules and plugins. These are Nix expressions that can define arbitrary packages, shell hooks, and services. The general spike covers third-party dependency risk but doesn't address Nix module-level trust (evaluating arbitrary Nix code at environment build time).

### G5: Content-Addressed Derivation Gaps
Nix's content-addressed store provides strong integrity guarantees post-build, but Fixed-Output Derivations (FODs) are a known security gap. FODs have network access during build and are trusted based solely on output hash — they could perform malicious actions during the build that aren't captured in the output hash. CVE-2024-27297 demonstrated that FOD content could be modified after registration, and CVE-2026-39860 showed a regression in FOD sandbox handling. The general spike's install-script-sandboxing research may touch on analogous concepts but won't cover Nix-specific FOD risks.

### G6: Pre-commit Hook Integrity
Devenv.sh auto-generates `.pre-commit-config.yaml` as a symlink into the Nix store. This is good (config is derived from the Nix expression, harder to tamper with), but the hooks themselves run in the developer's shell context. Ensuring pre-commit hooks for security scanning are present and enforced (not bypassable) is a devenv-specific configuration concern.

### G7: `nix.conf` System vs. Project Trust Boundaries
A critical devenv-specific concern: project-level `devenv.nix` files can request additional trusted substituters and public keys. If the developer is a `trusted-user` in system `nix.conf`, these project-level requests are automatically honored. This means cloning and entering a malicious project's devenv could silently add attacker-controlled binary caches. The boilerplate needs to address whether developers should be trusted-users and how to handle per-project cache trust.

## Nix-Specific Findings to Pull into Devenv-Security Spike

1. **Nix's inherent supply chain advantages**: Content-addressed store, sandboxed builds, deterministic derivations, channel-based quarantine delay, flake lock pinning. These provide a stronger baseline than any language-specific package manager.

2. **Nix's specific weaknesses**: FOD network access, shell hook execution outside sandbox, binary cache trust model (single signing key by default), `trusted-user` privilege escalation from project configs.

3. **Trustix for multi-party binary cache verification**: Decentralizes trust beyond a single cache signer. Relevant for organizational deployments where a private cache is used.

4. **CVE history**: CVE-2024-27297 (FOD content contamination), CVE-2024-38531 (sandbox escape via build directory permissions), CVE-2026-39860 (FOD sandbox regression), GHSA-g3g9-5vj6-r3gj (symlink-based file write in FOD registration). These demonstrate that Nix's sandbox, while strong, has had real bypasses — defense in depth is warranted.

5. **sbomnix for SBOM generation**: Enables feeding Nix dependency data into standard vulnerability scanning pipelines that the general spike's organizational tooling research will recommend.

## Contradictions and Tensions

| General Best Practice | Nix/Devenv Reality | Tension |
|---|---|---|
| Use registry-level provenance (SLSA, Sigstore) | Nix uses its own Ed25519 signing; no SLSA/Sigstore integration | Nix's provenance model is orthogonal to the emerging industry standard |
| Scan dependencies with Snyk/Dependabot/etc. | Most scanners don't parse Nix derivations or flake.lock | Need Nix-specific tooling (vulnix, sbomnix) instead |
| Enforce lock file updates through CI | Flake lock pins entire nixpkgs commit, not individual packages | Lock granularity is different; "update one dependency" isn't possible |
| Sandbox install scripts | Nix sandboxes builds by default, but FODs and shell hooks escape it | Nix is stronger for builds, weaker for FODs and devenv shell hooks |
| Use quarantine / hold-back periods | Nixpkgs channel promotion provides natural delay | Works by default but only if using stable/released channels, not HEAD |
| Pin to specific package versions | Nix pins to nixpkgs commits; individual package pinning requires overlays | Coarser-grained pinning model; different risk profile |
| Restrict install-time network access | Nix sandbox blocks network except for FODs | FODs are the exception that proves the rule; they're the supply chain entry point |

## Recommendations for Devenv-Security Boilerplate

Based on this cross-reference, the devenv-security spike should prioritize:

1. **Shell hook sandboxing or auditing** — This is the biggest gap between Nix's strong build sandbox and actual developer security in devenv.sh.
2. **Binary cache trust policy** — Define a boilerplate `nix.conf` policy for trusted-substituters and whether developers should be trusted-users.
3. **Flake input verification** — Practices for reviewing and restricting which flake inputs a project can declare.
4. **FOD-aware security posture** — Acknowledge and mitigate the FOD security gap, particularly for projects that fetch many external sources.
5. **Nix-native scanning integration** — Integrate vulnix or sbomnix into devenv's pre-commit hooks for automated vulnerability checking.

## Source Files

- `research-spikes/package-supply-chain-security/research.md` — Spike overview and scope
- `research-spikes/package-supply-chain-security/tasks.md` — Task definitions and status
- `research-spikes/package-supply-chain-security/log.md` — Research log entries
