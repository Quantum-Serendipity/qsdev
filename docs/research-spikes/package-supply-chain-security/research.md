# Research Summary: Package Supply Chain Security

## Overview

Deep-dive into strategies for mitigating supply chain and compromised package attacks across all major programming language package managers. Focus on defenses that can be configured once and then work invisibly in the background to keep software development, CI/CD pipelines, and deployed infrastructure secure. Areas of interest include:

- Hosting package mirrors or caches that only serve validated/verified packages
- Requiring minimum publication age before packages are consumable
- Signature verification and provenance attestation
- Lock file integrity and reproducible builds
- Registry-level protections (typosquatting detection, namespace policies)
- Runtime sandboxing of install scripts
- Any other defense-in-depth strategies being deployed in practice

The emphasis is on practical, operational solutions — things you set up once for an organization or development environment that then silently protect all downstream consumers.

**Scope exclusion:** NixOS-specific considerations — this research targets non-NixOS systems.

## Topics

### Publication Age Quarantine Gates & Package Hold Strategies
- **Status**: Complete
- **Report**: [`quarantine-gates-research.md`](quarantine-gates-research.md)
- **Summary**: Age-gating (delaying consumption of newly published package versions by a configurable period) has emerged as the highest-impact, lowest-cost defense against supply chain attacks. PyPI data shows 92% of malware is caught within 24 hours, making even a 3-day hold highly effective. As of mid-2026, native support exists in npm, pnpm, Yarn, Bun, Deno, pip, uv, and Cargo, plus configurable delays in Renovate, Dependabot, and Snyk. Enterprise tools (JFrog Curation, Sonatype Nexus Firewall) add policy-based quarantine. All major tools exempt security updates from age-gating, addressing the zero-day patch tension. The main gap: 10 different naming conventions across tools for the same concept, and no major public registry implements mandatory hold periods.

### Organizational Tooling & Policy Enforcement
- **Status**: Complete
- **Report**: [`org-tooling-research.md`](org-tooling-research.md)
- **Summary**: Surveyed 10 tools across five functional layers of supply chain defense: behavioral threat detection (Socket.dev, Phylum/Veracode), vulnerability scanning (Snyk, OSV Scanner, Grype+Syft), automated dependency updates (Dependabot, Renovate), project health assessment (OpenSSF Scorecard, deps.dev), and CI/CD runtime protection (StepSecurity Harden-Runner). No single tool covers the full threat surface -- a layered stack is essential. Socket.dev and Snyk are complementary (behavioral zero-day detection vs known CVE scanning). Renovate dominates Dependabot for multi-platform organizations needing policy-as-code dependency management. Harden-Runner is unique as the only tool protecting the CI pipeline itself from compromise. A recommended free stack (Dependabot + OSV Scanner + Socket free + Harden-Runner + Scorecard) provides strong baseline coverage; production stacks add Renovate, Socket paid tiers, Snyk, and Grype+Syft for containers.

### Private Registries & Validated Package Mirrors/Caches
- **Status**: Complete
- **Report**: [`private-registries-research.md`](private-registries-research.md)
- **Summary**: Evaluated 12 tools/platforms that act as intermediary registries, proxying, caching, and validating packages from upstream public registries before serving them to developers and CI/CD. The landscape divides sharply between tools that merely proxy/cache (cloud-native services like AWS CodeArtifact, Azure Artifacts, Google Artifact Registry) and tools that actively scan, quarantine, and policy-gate downloads (JFrog Artifactory with Curation/Xray, Sonatype Nexus with Repository Firewall, Cloudsmith). JFrog and Sonatype are the clear enterprise leaders for security; Verdaccio is the standout open-source option for npm-only teams with built-in age-gating and blocklists via its package-filter plugin. GitHub Packages does not proxy upstream registries at all and cannot serve as a "configure once" defense. No free, open-source, multi-ecosystem registry with security scanning exists -- this is the gap that commercial tools fill. For multi-ecosystem organizations, the recommendation is JFrog or Sonatype for comprehensive coverage, or Nexus Community Edition as a free registry layer paired with CI-level scanners (Socket.dev, Snyk, OSV Scanner) for budget-constrained teams.

### Per-Ecosystem Supply Chain Attack Surface Landscape
- **Status**: Complete
- **Report**: [`attack-surface-landscape-research.md`](attack-surface-landscape-research.md)
- **Summary**: Mapped the supply chain attack surface across all seven target ecosystems (npm, PyPI, Cargo/crates.io, Go modules, Maven Central, NuGet, RubyGems). Each ecosystem has a distinct architectural model resulting in different vulnerability profiles. npm faces the most severe active threat landscape due to lifecycle script execution, massive dependency trees, and proven wormable propagation (Shai-Hulud, September 2025). Go modules are architecturally the most resilient by design -- no registry accounts, no install hooks, global checksum database, Minimal Version Selection. Maven Central's domain-verified namespaces provide the strongest anti-typosquatting protection. All ecosystems are converging on Trusted Publishing via OIDC and Sigstore-based attestations, but adoption varies from mature (PyPI, npm) to planned (NuGet). A critical cross-cutting finding: most ecosystems have invested heavily in publisher-side security (who can publish) while leaving consumer-side protections (what happens at install time) largely unaddressed -- only Go eliminates install-time code execution by design, and only alternative JS package managers (pnpm, Yarn, Bun) provide consumer-side defenses in the npm ecosystem.

### Lock File Integrity & Reproducible Installs
- **Status**: Complete
- **Report**: [`lockfile-integrity-research.md`](lockfile-integrity-research.md)
- **Summary**: Comprehensive analysis of lock file mechanics, hash verification, and CI enforcement across ten ecosystems (npm, Yarn, pnpm, pip/Poetry/uv, Cargo, Go, Maven, Gradle, NuGet, Bundler) plus Bazel and Docker container builds. Lock files with hash pinning are the highest-leverage "configure-once" defense -- they eliminate version resolution drift and detect registry compromise or MITM via cryptographic verification. Go is architecturally strongest (ecosystem-wide checksum transparency log via sum.golang.org); pnpm is structurally immune to lockfile poisoning URL attacks (doesn't store tarball URLs); Gradle is weakest (no checksums, 0.9% adoption). Lock files also introduce their own attack surface: lockfile poisoning via PR manipulation is a real and growing threat, mitigated by tools like lockfile-lint, Yarn Hardened Mode, and CODEOWNERS policies. PEP 751 (pylock.toml, accepted March 2025) makes Python the first ecosystem with mandatory hashes in its standard lock file format. The report includes configure-once dotfile patterns, Dockerfile snippets, and an ecosystem maturity ranking.

### Signature Verification & Provenance Attestation
- **Status**: Complete
- **Report**: [`signature-provenance-research.md`](signature-provenance-research.md)
- **Summary**: Surveyed cryptographic signature and provenance attestation mechanisms across nine package ecosystems (npm, PyPI, Go, Cargo/crates.io, Maven, NuGet, RubyGems, plus the cross-cutting SLSA framework and Sigstore infrastructure). Sigstore keyless signing is converging as the universal provenance layer -- ecosystems adopting it immediately achieve SLSA L3-capable provenance. npm (~7% adoption) and PyPI (~17% adoption) lead with automatic Sigstore attestations via Trusted Publishing; Maven Central requires PGP and optionally accepts Sigstore; Go provides the strongest consumer enforcement via its checksum database (integrity, not provenance); NuGet offers `signatureValidationMode=require` for X.509 signatures; cargo-vet provides a unique human-review attestation model. The critical gap: **no major ecosystem allows consumers to require provenance at install time**. Publisher-side infrastructure is largely solved; consumer enforcement is 1-2 years behind. The sole partial exception is pnpm's `trustPolicy: no-downgrade` (opt-in, detects provenance downgrade rather than requiring it).

### Install Script Sandboxing & Runtime Protections
- **Status**: Complete
- **Report**: [`install-sandboxing-research.md`](install-sandboxing-research.md)
- **Summary**: Install-time code execution is the single most exploited attack vector in package supply chain compromises. Most major ecosystems allow arbitrary code execution during installation -- npm via lifecycle scripts (postinstall), Python via setup.py in source distributions, Rust via build.rs and proc macros, Ruby via extconf.rb, and JVM via build plugins. Go is the gold standard, having made the explicit design decision that "neither fetching nor building code will let that code execute." The best configure-once defenses available today are: pnpm's `allowBuilds` with default script blocking (most mature JS solution); npm's `ignore-scripts=true` paired with `@lavamoat/allow-scripts` for per-package version-pinned allowlisting; Python's `--only-binary :all:` to refuse source distributions; and Deno's permission model which blocks all scripts by default with per-package `--allow-scripts=npm:pkg` granularity. For deeper isolation, the two-phase install pattern (download online, build offline via bubblewrap/firejail/containers) neutralizes data exfiltration. Rust's Cackle tool provides the most sophisticated per-ecosystem sandboxing (Bubblewrap-based build script isolation with API-level ACLs). No ecosystem has yet shipped true OS-level sandboxing of install scripts as a default -- this remains the critical gap.

## Open Questions

- How quickly is Socket.dev expanding full behavioral analysis beyond npm/PyPI to Go, Java, Rust?
- Can Renovate be configured to use OSV vulnerability data to prioritize security-relevant updates?
- Are there Harden-Runner equivalents for GitLab CI or Jenkins, or is CI runtime protection GitHub Actions-only?
- When will pip implement install-time attestation verification? (Trail of Bits plugin architecture in progress)
- Will npm CLI ever add provenance enforcement, or will this remain a pnpm-only feature?
- How will the Rust ecosystem resolve RFC #3403 for Sigstore integration with crates.io?
- Will Maven Central eventually require Sigstore signatures alongside PGP?

## Conclusions

### Cross-Cutting Findings

**1. The publisher-consumer gap is the defining structural problem.** Every ecosystem has invested heavily in who-can-publish (2FA, Trusted Publishing, OIDC, attestations) but almost none have addressed what-happens-at-install-time. Go is the only ecosystem where consumer-side protections are default-on (no install hooks + checksum transparency log). pnpm is the JS leader in consumer defense but still opt-in. This gap means configure-once defenses must be layered by the consumer org, not relied upon from registries.

**2. Five defense layers, each configure-once.** An effective supply chain security posture requires all five:
   1. **Age-gating** — delay consumption of new versions by 3-7 days (catches 92%+ of malware). Configure via package manager native settings or Renovate/Dependabot stabilityDays.
   2. **Install script blocking** — disable arbitrary code execution at install time (npm `ignore-scripts`, pnpm `allowBuilds`, Python `--only-binary`). Allowlist exceptions per-package.
   3. **Lock file enforcement** — require frozen installs in CI with hash verification. Protect lock files via CODEOWNERS and lockfile-lint.
   4. **Scanning & monitoring** — layer behavioral detection (Socket.dev) with CVE scanning (Snyk/OSV Scanner) and CI runtime protection (Harden-Runner).
   5. **Private registry** — proxy upstream through a validated mirror (JFrog/Sonatype for enterprise, Verdaccio for npm OSS) that applies policies before packages reach developers.

**3. Age-gating is the highest-impact, lowest-cost defense.** With 92% of PyPI malware caught within 24 hours and most attack opportunity windows under one week, a 3-day hold period eliminates the vast majority of threats with minimal developer friction. Native support now exists in npm, pnpm, Yarn, Bun, Deno, pip, uv, and Cargo. This should be the first thing any org configures.

**4. No single tool covers the full threat surface.** Socket.dev (behavioral zero-day detection) and Snyk (known CVE scanning) are complementary, not competing. Harden-Runner protects the CI pipeline itself — a vector no other tool addresses. A minimum viable free stack: Dependabot + OSV Scanner + Socket free + Harden-Runner + Scorecard.

**5. Lock files are both defense and attack surface.** While frozen installs with hash pinning are essential, lockfile poisoning via PR manipulation is a real and growing attack class. Defenses must protect the protector: CODEOWNERS on lock files, lockfile-lint, Yarn Hardened Mode.

**6. Ecosystem maturity varies dramatically.** Go is architecturally the most secure (no install hooks, checksum transparency, MVS). npm is the most actively threatened (wormable attacks, massive dependency trees, lifecycle script exploitation). Maven Central has the strongest anti-typosquatting via domain-verified namespaces. Gradle is the weakest on lock file enforcement (0.9% adoption).

**7. Sigstore is converging as the universal provenance layer** but consumer enforcement is 1-2 years behind publisher adoption. No major ecosystem allows requiring provenance at install time. pnpm's `trustPolicy: no-downgrade` is the sole partial exception.

**8. Slopsquatting is an emerging threat.** AI coding assistants hallucinate package names ~20% of the time, and 43% of hallucinated names recur predictably — attackers can register these. This is a new vector that existing defenses (age-gating, scanning) partially address but that will grow with AI adoption.

### Recommended Implementation Priority

For an organization starting from zero:
1. Enable age-gating in all package managers (immediate, free, highest impact)
2. Disable install scripts by default with per-package allowlists (immediate, free)
3. Enforce frozen lock files in CI with hash verification (immediate, free)
4. Deploy Socket.dev + OSV Scanner in CI (same week, free tier)
5. Add Harden-Runner to GitHub Actions workflows (same week, free)
6. Evaluate private registry (JFrog/Sonatype) for centralized policy enforcement (month 1-2, budget-dependent)
