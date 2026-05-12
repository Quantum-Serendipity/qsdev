# Research Log: Package Supply Chain Security

## 2026-05-12 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Investigating strategies to mitigate supply chain and compromised package attacks across programming language package managers, with a focus on configure-once-and-forget defenses: package mirrors/caches serving only validated packages, publication age requirements, and other defense-in-depth measures that protect development, CI/CD, and deployment environments invisibly.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-05-12 — Phase 1 Tasks Defined
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Seven Phase 1 tasks created covering: ecosystem attack surface landscape, private registries/mirrors, publication age quarantine, signature verification/provenance, lock file integrity, install script sandboxing, and organizational tooling. NixOS-specific considerations excluded per user — research targets non-NixOS systems.
- **Next**: Begin research. Delegate tasks to sub-agents in parallel where possible.

## 2026-05-12 14:00 — Publication Age Quarantine Gates Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Package Managers Need to Cool Down](https://nesbitt.io/2026/03/04/package-managers-need-to-cool-down.html) → `docs/nesbitt-package-managers-cool-down.md`
  - [Renovate minimumReleaseAge docs](https://docs.renovatebot.com/key-concepts/minimum-release-age/) → `docs/renovate-minimum-release-age-docs.md`
  - [Socket Firewall Overview](https://docs.socket.dev/docs/socket-firewall-overview) → `docs/socket-firewall-overview.md`
  - [JFrog Curation Time-Delay Policies](https://academy.jfrog.com/how-to-use-curation-time-delay-policies-to-block-package-hijacks) → `docs/jfrog-curation-time-delay-policies.md`
  - [Sonatype Nexus Firewall Quarantine](https://help.sonatype.com/en/firewall-quarantine.html) → `docs/sonatype-nexus-firewall-quarantine.md`
  - [StepSecurity NPM Cooldown Check](https://www.stepsecurity.io/blog/introducing-the-npm-package-cooldown-check) → `docs/stepsecurity-npm-cooldown-check.md`
  - [Dependabot Cooldown Config](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference) → `docs/dependabot-cooldown-configuration.md`
  - [pnpm Supply Chain Security](https://pnpm.io/supply-chain-security) → `docs/pnpm-supply-chain-security.md`
  - [set-minimum-package-release-age](https://github.com/dehrenschwender/set-minimum-package-release-age) → `docs/set-minimum-package-release-age-tool.md`
  - [gem.coop Dependency Cooldowns](https://socket.dev/blog/gem-coop-tests-dependency-cooldowns) → `docs/gem-coop-dependency-cooldowns.md`
  - [PyPI 2025 Year in Review](https://blog.pypi.org/posts/2025-12-31-pypi-2025-in-review/) → `docs/pypi-2025-year-in-review-malware.md`
  - [npm minimumReleaseAge](https://socket.dev/blog/npm-introduces-minimumreleaseage-and-bulk-oidc-configuration) → `docs/npm-minimumreleaseage-socket-blog.md`
  - [Dependency Cooldowns Defense](https://christian-schneider.net/blog/dependency-cooldowns-supply-chain-defense/) → `docs/schneider-dependency-cooldowns-defense.md`
  - [Spring 2026 OSS Incidents](https://dev.to/trknhr/lessons-from-the-spring-2026-oss-incidents-hardening-npm-pnpm-and-github-actions-against-1jnp) → `docs/spring-2026-oss-incidents-hardening.md`
- **Summary**: Completed deep research on publication age quarantine gates. Covered: theory and detection time statistics (PyPI handles 92% of malware within 24h), native package manager support across JS/Python/Ruby/Rust ecosystems, Renovate/Dependabot/Snyk update tool configurations, JFrog Curation time-delay and Sonatype Nexus Firewall quarantine, registry-level approaches (gem.coop is the only registry-level implementation), custom tooling (set-minimum-package-release-age), tradeoffs and exception handling, and comparison with alternative defenses. Key finding: age-gating has moved from fringe idea to ecosystem default within ~8 months, with 10 different naming conventions across tools for the same concept.
- **Next**: Mark task as completed. Continue with remaining Phase 1 tasks.

## 2026-05-12 15:30 — Organizational Tooling & Policy Enforcement Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Socket.dev FAQ](https://docs.socket.dev/docs/faq) → `docs/socket-dev-faq.md`
  - [Socket.dev Review 2026](https://appsecsanta.com/socket) → `docs/socket-dev-review-2026.md`
  - [Snyk Review 2026](https://appsecsanta.com/snyk) → `docs/snyk-review-2026.md`
  - [Dependabot Version Updates](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates/about-dependabot-version-updates) → `docs/dependabot-version-updates.md`
  - [Dependabot vs Renovate](https://appsecsanta.com/sca-tools/dependabot-vs-renovate) → `docs/dependabot-vs-renovate-comparison.md`
  - [Renovate Configuration Options](https://docs.renovatebot.com/configuration-options/) → `docs/renovate-configuration-options.md`
  - [Renovate Merge Confidence](https://docs.renovatebot.com/merge-confidence/) → `docs/renovate-merge-confidence.md`
  - [OSV-Scanner Overview](https://google.github.io/osv-scanner/) → `docs/osv-scanner-overview.md`
  - [OpenSSF Scorecard](https://github.com/ossf/scorecard) → `docs/openssf-scorecard-github.md`
  - [deps.dev API v3](https://docs.deps.dev/api/v3/) → `docs/deps-dev-api-v3.md`
  - [Phylum/Veracode Acquisition](https://www.businesswire.com/news/home/20250106967344/en/) → `docs/phylum-veracode-acquisition.md`
  - [Grype + Syft (Anchore)](https://anchore.com/opensource/) → `docs/grype-syft-anchore-opensource.md`
  - [StepSecurity Harden-Runner](https://github.com/step-security/harden-runner) → `docs/stepsecurity-harden-runner-github.md`
  - [Renovate Bot Comparison](https://docs.renovatebot.com/bot-comparison/) → via `docs/dependabot-vs-renovate-comparison.md`
- **Summary**: Completed deep research on 10 organizational tooling platforms for supply chain security. Categorized into five functional layers: behavioral threat detection (Socket.dev, Phylum/Veracode), vulnerability scanning (Snyk, OSV Scanner, Grype), automated dependency updates (Dependabot, Renovate), project health assessment (OpenSSF Scorecard, deps.dev), and CI/CD runtime protection (StepSecurity Harden-Runner). For each tool: documented detection capabilities, set-and-forget configuration, ecosystem support, pricing, CI integration, and limitations. Produced comparison matrix and recommended stacks (free and production). Key findings: (1) no single tool covers the full threat surface — layered stacks are essential; (2) Socket.dev and Snyk are complementary not competing (behavioral vs CVE-based); (3) Renovate dominates Dependabot for multi-platform orgs; (4) Harden-Runner is the only tool protecting the CI pipeline itself; (5) Phylum is no longer standalone (acquired by Veracode Jan 2025).
- **Next**: Update tasks.md. Consider depth checklist review.

## 2026-05-12 16:30 — Private Registries & Validated Mirrors Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [JFrog Curation Q&A](https://www.securityscientist.net/blog/12-questions-and-answers-about-jfrog-curation-jfrog/) → `docs/jfrog-curation-overview.md`
  - [JFrog Xray Review 2026](https://appsecsanta.com/jfrog-xray) → `docs/jfrog-xray-review.md`
  - [JFrog Pricing 2026](https://jfrog.com/pricing/) → referenced in report
  - [Sonatype Repository Firewall](https://www.sonatype.com/products/sonatype-repository-firewall) → `docs/sonatype-repository-firewall.md`
  - [Sonatype Nexus Repository](https://www.sonatype.com/products/sonatype-nexus-repository) → `docs/sonatype-nexus-repository.md`
  - [Verdaccio Architecture](https://deepwiki.com/verdaccio/verdaccio/1.1-key-features-and-use-cases) → `docs/verdaccio-features-architecture.md`
  - [Verdaccio Package Filter Plugin](https://github.com/verdaccio/verdaccio/blob/8.x/packages/plugins/package-filter/README.md) → `docs/verdaccio-package-filter-plugin.md`
  - [Devpi Server Overview](https://deepwiki.com/devpi/devpi/2-devpi-server) → `docs/devpi-server-overview.md`
  - [Cloudsmith Upstream Proxying](https://docs.cloudsmith.com/repositories/upstreams) → `docs/cloudsmith-upstream-proxying.md`
  - [GitLab Virtual Registry](https://docs.gitlab.com/user/packages/virtual_registry/) → `docs/gitlab-virtual-registry.md`
  - [GitHub Packages Intro](https://docs.github.com/en/packages/learn-github-packages/introduction-to-github-packages) → `docs/github-packages-overview.md`
  - [Google AR Remote Repos](https://docs.cloud.google.com/artifact-registry/docs/repositories/remote-overview) → `docs/google-artifact-registry-remote-repos.md`
  - [Azure Artifacts Upstream Behavior](https://learn.microsoft.com/en-us/azure/devops/artifacts/concepts/upstream-behavior) → `docs/azure-artifacts-upstream-behavior.md`
  - [AWS CodeArtifact Upstream Repos](https://docs.aws.amazon.com/codeartifact/latest/ug/repos-upstream.html) → `docs/aws-codeartifact-upstream-repos.md`
  - [PHP Packagist Supply Chain Security](https://blog.packagist.com/strengthening-php-supply-chain-security-with-a-transparency-log-for-packagist-org/) → `docs/php-packagist-supply-chain-security.md`
  - [Bytesafe npm Firewall](https://bytesafe.dev/supply-chain-security/) → `docs/bytesafe-npm-firewall.md`
- **Summary**: Completed deep research on private registries and validated package mirrors. Evaluated 12 tools/platforms across three categories: enterprise universal registries (JFrog Artifactory, Sonatype Nexus), ecosystem-specific registries (Verdaccio, Devpi, Bytesafe, Private Packagist), and cloud-native services (AWS CodeArtifact, Azure Artifacts, Google Artifact Registry, GitHub Packages, GitLab Package Registry, Cloudsmith). Key findings: (1) landscape divides sharply between tools that merely proxy/cache vs tools that actively scan and gate; (2) JFrog and Sonatype are the clear security leaders; (3) cloud-native services are caching layers, not security layers; (4) GitHub Packages doesn't proxy upstream at all; (5) Verdaccio is the standout OSS option for npm with built-in age-gating and blocklists; (6) no free multi-ecosystem registry with security features exists.
- **Next**: Continue with remaining Phase 1 tasks.

## 2026-05-12 17:30 — Per-Ecosystem Attack Surface Landscape Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Go Supply Chain Mitigations Blog](https://go.dev/blog/supply-chain) → `docs/go-supply-chain-mitigations-blog.md`
  - [npm Trusted Publishing Docs](https://docs.npmjs.com/trusted-publishers/) → `docs/npm-trusted-publishing-docs.md`
  - [PyPI 2025 Year in Review](https://blog.pypi.org/posts/2025-12-31-pypi-2025-in-review/) → `docs/pypi-2025-year-in-review.md`
  - [Maven Central Publishing Requirements](https://central.sonatype.org/publish/requirements/) → `docs/maven-central-publishing-requirements.md`
  - [NuGet Supply Chain Security](https://devblogs.microsoft.com/dotnet/building-a-safer-future-how-nuget-is-tackling-software-supply-chain-threats/) → `docs/nuget-supply-chain-security-measures.md`
  - [Ruby Central Security Strengthening](https://rubycentral.org/news/securing-rubys-future-how-ruby-central-is-strengthening-security/) → `docs/ruby-central-security-strengthening.md`
  - [npm Threat Landscape (Unit 42)](https://unit42.paloaltonetworks.com/monitoring-npm-supply-chain-attacks/) → `docs/npm-threat-landscape-unit42.md`
  - [npm Supply Chain Security 2026 (Mondoo)](https://mondoo.com/blog/npm-supply-chain-security-package-manager-defenses-2026) → `docs/npm-supply-chain-security-2026-mondoo.md`
  - [Rust Supply Chain Security Practices](https://blog.ortham.net/posts/2025-10-02-rust-supply-chain-security/) → `docs/rust-supply-chain-security-practices.md`
  - [BoltDB Typosquatting Attack](https://socket.dev/blog/malicious-package-exploits-go-module-proxy-caching-for-persistence) → `docs/go-boltdb-typosquatting-attack.md`
  - [PyPI Attestation Security Model](https://docs.pypi.org/attestations/security-model/) → `docs/pypi-attestation-security-model.md`
  - [NuGet Security Best Practices](https://learn.microsoft.com/en-us/nuget/concepts/security-best-practices) → `docs/nuget-security-best-practices-microsoft.md`
- **Summary**: Completed comprehensive landscape survey of all seven target ecosystems (npm, PyPI, Cargo, Go modules, Maven Central, NuGet, RubyGems). For each ecosystem documented: registry model, publishing authentication, known attack vectors with severity assessments, notable real-world incidents with dates and impact, and current registry protections. Produced cross-ecosystem comparison matrices for attack vectors and protections. Key findings: (1) npm faces the most severe threat landscape due to lifecycle script execution, massive dependency trees, and proven wormable propagation; (2) Go modules are architecturally the most secure (no registry accounts, no install hooks, global checksum database); (3) Maven Central's domain-verified namespaces are the strongest anti-typosquatting measure; (4) all ecosystems are converging on Trusted Publishing via OIDC but adoption varies widely; (5) critical gap exists between publisher-side and consumer-side protections -- most ecosystems have invested in publisher security while leaving consumers largely unprotected.
- **Next**: Mark task as completed. Continue with remaining Phase 1 tasks (signature verification, lock file integrity, install script sandboxing).

## 2026-05-12 18:30 — Lock File Integrity & Reproducible Installs Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Supply-Chain Guardrails for npm, pnpm, and Yarn](https://www.coinspect.com/blog/supply-chain-guardrails/) → `docs/coinspect-supply-chain-guardrails-npm-pnpm-yarn.md`
  - [npm Lockfiles as Security Blindspot (Snyk)](https://snyk.io/blog/why-npm-lockfiles-can-be-a-security-blindspot-for-injecting-malicious-modules/) → `docs/snyk-lockfile-security-blindspot.md`
  - [Lockfile Poisoning Attack Vector (SafeDep)](https://safedep.substack.com/p/lockfile-poisoning-an-attack-vector) → `docs/safedep-lockfile-poisoning-attack.md`
  - [Python Supply Chain Security: Defense in Depth](https://bernat.tech/posts/securing-python-supply-chain/) → `docs/bernat-python-supply-chain-defense-in-depth.md`
  - [Design Space of Lockfiles Across Package Managers (arXiv)](https://arxiv.org/html/2505.04834v2) → `docs/arxiv-lockfile-design-space-across-package-managers.md`
  - [go.sum Is Not a Lockfile (Filippo Valsorda)](https://words.filippo.io/gosum/) → `docs/filippo-go-sum-not-lockfile.md`
  - [Gradle Dependency Locking Docs](https://docs.gradle.org/current/userguide/dependency_locking.html) → `docs/gradle-dependency-locking-docs.md`
  - [Maven Lockfile Plugin](https://github.com/chains-project/maven-lockfile) → `docs/maven-lockfile-plugin.md`
  - [NuGet Lock File Wiki](https://github.com/NuGet/Home/wiki/Enable-repeatable-package-restore-using-lock-file) → `docs/nuget-lock-file-repeatable-restore.md`
  - [Bazel Lockfile Docs](https://bazel.build/external/lockfile) → `docs/bazel-lockfile-docs.md`
  - [npm CI/CD Locked Dependencies](https://charlesjones.dev/blog/npm-supply-chain-attacks-ci-cd-locked-dependencies) → `docs/charlesjones-npm-ci-cd-locked-dependencies.md`
  - [Yarn Security Features / Hardened Mode](https://yarnpkg.com/features/security) → `docs/yarn-security-features-hardened-mode.md`
  - [PEP 751 pylock.toml](https://peps.python.org/pep-0751/) → `docs/pep-751-pylock-toml-format.md`
  - [Reproducible Docker Images with Locked Dependencies](https://oneuptime.com/blog/post/2026-02-08-how-to-build-reproducible-docker-images-with-locked-dependencies/view) → `docs/reproducible-docker-images-locked-dependencies.md`
  - [Bundler v2.6: Lockfile Checksums](https://bundler.io/blog/2024/12/19/bundler-v2-6.html)
  - [lockfile-lint (Liran Tal)](https://github.com/lirantal/lockfile-lint)
  - [Cargo FAQ](https://doc.rust-lang.org/cargo/faq.html)
  - [Go Modules Reference](https://go.dev/ref/mod)
  - [Go Checksum Database Proposal](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md)
- **Summary**: Completed deep research on lock file integrity and reproducible installs across nine ecosystems (npm, Yarn, pnpm, pip/Poetry/uv, Cargo, Go, Maven, Gradle, NuGet, Bundler) plus Bazel and Docker. Covered: lock file mechanics and enforcement flags per ecosystem, hash verification comparison (algorithm, included-by-default, what it verifies against), CI enforcement quick reference with configure-once dotfile patterns, reproducible build strategies (Bazel, container-based), lockfile poisoning attacks (mechanism, why they succeed, which ecosystems are vulnerable, mitigation tools like lockfile-lint and Yarn Hardened Mode), and tradeoffs of strict enforcement. Produced ecosystem maturity ranking. Key findings: (1) Go is architecturally strongest (ecosystem-wide checksum transparency log); (2) pnpm is structurally immune to lockfile poisoning URL attacks because it doesn't store tarball URLs; (3) Gradle is weakest (no checksums, 0.9% adoption); (4) PEP 751 pylock.toml makes Python the first ecosystem with mandatory hashes in its standard lock file format; (5) lockfile poisoning is a real and distinct attack class requiring dedicated tooling beyond lock file enforcement itself.
- **Next**: Run depth checklist review. Update tasks.md to mark complete.

## 2026-05-12 19:00 — Signature Verification & Provenance Attestation Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [SLSA Spec v1.2 About](https://slsa.dev/spec/v1.2/about) → `docs/slsa-spec-v1.2-about.md`
  - [SLSA Build Track Basics](https://slsa.dev/spec/v1.2/build-track-basics) → `docs/slsa-build-track-basics.md`
  - [Sigstore Cosign Signing Overview](https://docs.sigstore.dev/cosign/signing/overview/) → `docs/sigstore-cosign-signing-overview.md`
  - [npm Provenance Statements](https://docs.npmjs.com/generating-provenance-statements/) → `docs/npm-provenance-statements.md`
  - [npm Trusted Publishing](https://docs.npmjs.com/trusted-publishers/) → `docs/npm-trusted-publishing.md`
  - [npm Supply Chain Security 2026](https://mondoo.com/blog/npm-supply-chain-security-package-manager-defenses-2026) → `docs/npm-supply-chain-security-2026-mondoo.md`
  - [PyPI Trusted Publishers](https://docs.pypi.org/trusted-publishers/) → `docs/pypi-trusted-publishers.md`
  - [PyPI Attestations Security Model](https://docs.pypi.org/attestations/security-model/) → `docs/pypi-attestations-security-model.md`
  - [PyPI Attestations - Trail of Bits](https://blog.trailofbits.com/2024/11/14/attestations-a-new-generation-of-signatures-on-pypi/) → `docs/pypi-attestations-trail-of-bits.md`
  - [Go Checksum Database Design](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md) → `docs/go-checksum-database-design.md`
  - [Go Module Mirror Launch](https://go.dev/blog/module-mirror-launch) → `docs/go-module-mirror-launch.md`
  - [cargo-vet How It Works](https://mozilla.github.io/cargo-vet/how-it-works.html) → `docs/cargo-vet-how-it-works.md`
  - [Rust Foundation Artifact Signing](https://rustfoundation.org/media/improving-supply-chain-security-for-rust-through-artifact-signing/) → `docs/rust-foundation-artifact-signing.md`
  - [Maven Central Sigstore Validation](https://socket.dev/blog/maven-central-adds-sigstore-signature-validation) → `docs/maven-central-sigstore-validation.md`
  - [NuGet Signed Packages Reference](https://learn.microsoft.com/en-us/nuget/reference/signed-packages-reference) → `docs/nuget-signed-packages-reference.md`
  - [NuGet Manage Trust Boundaries](https://learn.microsoft.com/en-us/nuget/consume-packages/installing-signed-packages) → `docs/nuget-manage-trust-boundaries.md`
  - [RubyGems Security Guide](https://guides.rubygems.org/security/) → `docs/rubygems-security-guide.md`
  - [RubyGems Trusted Publishing](https://blog.rubygems.org/2023/12/14/trusted-publishing.html) → `docs/rubygems-trusted-publishing.md`
  - [2026 State of Registry Provenance](https://zenn.dev/sqer/articles/e4df3d397f5651) → `docs/2026-state-of-registry-provenance.md`
- **Summary**: Completed deep research on signature verification and provenance attestation across nine package ecosystems. Covered SLSA framework (levels L0-L3, build/source tracks), Sigstore architecture (Fulcio, Rekor, cosign keyless signing), npm provenance (Trusted Publishing, --provenance flag, audit signatures), PyPI Trusted Publishers and PEP 740 attestations, Go module checksum database (transparency log, fail-closed design), cargo-vet (human review attestation model), Maven Central (mandatory PGP, optional Sigstore), NuGet (X.509 author/repository signatures, signatureValidationMode=require), and RubyGems (legacy gem cert, emerging Sigstore attestations). Key finding: publisher-side provenance infrastructure is largely solved via Sigstore convergence, but consumer-side enforcement is almost entirely absent across all ecosystems. Go's checksum DB is the only default-enforced integrity mechanism; NuGet's signatureValidationMode=require is the only configurable signature enforcement. No ecosystem supports requiring provenance attestation at install time.
- **Next**: Continue with remaining Phase 1 tasks (install script sandboxing).

## 2026-05-12 20:00 — Install Script Sandboxing & Runtime Protections Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [npm Ignore-Scripts Best Practices](https://www.nodejs-security.com/blog/npm-ignore-scripts-best-practices-as-security-mitigation-for-malicious-packages) → `docs/npm-ignore-scripts-best-practices.md`
  - [npm Supply Chain Defenses 2026 (Mondoo)](https://mondoo.com/blog/npm-supply-chain-security-package-manager-defenses-2026) → `docs/npm-supply-chain-defenses-2026.md`
  - [@lavamoat/allow-scripts Guide](https://lavamoat.github.io/guides/allow-scripts/) → `docs/lavamoat-allow-scripts.md`
  - [pnpm Supply Chain Security](https://pnpm.io/supply-chain-security) → `docs/pnpm-supply-chain-security.md`
  - [pnpm Build Script Security](https://deepwiki.com/pnpm/pnpm/3.5-build-script-security) → `docs/pnpm-build-script-security.md`
  - [Deno Protects Against npm Exploits](https://deno.com/blog/deno-protects-npm-exploits) → `docs/deno-protects-npm-exploits.md`
  - [Deno Security & Permissions](https://docs.deno.com/runtime/fundamentals/security/) → `docs/deno-security-permissions.md`
  - [Python Package Installation Attacks (Veracode)](https://www.veracode.com/blog/python-package-installation-attacks/) → `docs/python-package-installation-attacks.md`
  - [Python Supply Chain Defense Guide](https://bernat.tech/posts/securing-python-supply-chain/) → `docs/python-supply-chain-defense-guide.md`
  - [PyPI Security Best Practices](https://github.com/lirantal/pypi-security-best-practices) → `docs/pypi-security-best-practices.md`
  - [Rust Sandboxed Build Scripts Goals](https://rust-lang.github.io/rust-project-goals/2024h2/sandboxed-build-script.html) → `docs/rust-sandboxed-build-scripts.md`
  - [Rust Build Security Supply Chain](https://rust-secure-code.github.io/rust-supply-chain-security/build.html) → `docs/rust-build-security-supply-chain.md`
  - [Cargo Build Script Allowlist Issue #13681](https://github.com/rust-lang/cargo/issues/13681) → `docs/cargo-build-script-allowlist-issue.md`
  - [Cackle: Rust Supply Chain ACLs](https://davidlattimore.github.io/posts/2023/10/09/making-supply-chain-attacks-harder.html) → `docs/cackle-rust-supply-chain.md`
  - [How Go Mitigates Supply Chain Attacks](https://go.dev/blog/supply-chain) → `docs/go-supply-chain-mitigations.md`
  - [Socket Firewall Overview](https://socket.dev/blog/introducing-socket-firewall) → `docs/socket-firewall-overview.md`
  - [Codex Sandboxing Implementation](https://deepwiki.com/openai/codex/5.6-sandboxing-implementation) → `docs/codex-sandboxing-implementation.md`
  - [Agent Sandbox Deep Dive](https://pierce.dev/notes/a-deep-dive-on-agent-sandboxes) → `docs/agent-sandbox-deep-dive.md`
- **Summary**: Completed deep research on install script sandboxing and runtime protections. Covered all seven target ecosystems' install-time code execution mechanisms, with detailed analysis of: npm ignore-scripts configuration and @lavamoat/allow-scripts allowlisting; pnpm v10+ allowBuilds (most mature JS solution); Deno's secure-by-default permission model; Python's wheel-vs-sdist attack surface and --only-binary defense; Rust's build.rs/proc-macro risks and the Cackle sandbox tool; Go's design-level elimination of install hooks (gold standard comparison); Ruby extconf.rb and JVM build plugin vectors; NuGet's deprecated-but-persistent init.ps1. Also covered network isolation patterns (two-phase download-then-build), OS-level sandboxing tools (bubblewrap, Landlock, seccomp, firejail), container/microVM isolation (Docker, Firecracker, gVisor, Kata), and emerging tools (Socket Firewall, Birdcage). Produced cross-ecosystem maturity comparison. Key finding: no ecosystem ships OS-level sandboxing of install scripts as a default; the best available defenses are ecosystem-level script blocking (pnpm allowBuilds, npm ignore-scripts) layered with environment-level isolation.
- **Next**: All Phase 1 tasks now complete. Ready for Phase 2 synthesis or spike completion.

## 2026-05-12 21:00 — Phase 1 Complete: All Tasks Done, Synthesis Written
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: All 7 Phase 1 research tasks completed successfully via parallel sub-agents. 7 detailed research reports produced, 102 source documents saved to docs/. Wrote synthesis conclusions in research.md identifying 8 cross-cutting findings and a prioritized implementation roadmap. Key themes: publisher-consumer protection gap, five configure-once defense layers, age-gating as highest-impact first step, lock files as both defense and attack surface, Sigstore convergence with enforcement lagging, and slopsquatting as emerging AI-driven threat.
- **Next**: Spike ready for completion or deeper Phase 2 investigation on specific topics if desired.

## 2026-05-12 22:00 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. 7 research reports (3,522 lines total), 102 source documents, 8 cross-cutting conclusions, and a prioritized 6-step implementation roadmap. Core finding: the publisher-consumer protection gap is the defining structural problem — consumer orgs must layer five configure-once defenses (age-gating, install script blocking, lock file enforcement, scanning/monitoring, private registry) because registries don't enforce consumer-side protections. Age-gating is the highest-impact first step (92% malware caught within 24h). 3 follow-on candidates flushed to proposed-spikes.md: slopsquatting defense, CI runtime protection for non-GitHub platforms, and consumer-side provenance enforcement tracking.
