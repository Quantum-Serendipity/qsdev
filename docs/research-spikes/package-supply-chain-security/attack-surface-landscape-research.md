# Per-Ecosystem Supply Chain Attack Surface Landscape

## Executive Summary

This report maps the supply chain attack surface across seven major package ecosystems: npm, PyPI, Cargo/crates.io, Go modules, Maven Central, NuGet, and RubyGems. Each ecosystem has a distinct architectural model for publishing, distributing, and consuming packages, resulting in different vulnerability profiles. The npm ecosystem faces the most severe and active threat landscape due to lifecycle script execution, massive dependency trees, and the scale of its registry. Go modules stand out as architecturally the most resilient by design, with no install hooks, no registry accounts, and a global checksum database. All ecosystems are converging on a common set of defenses -- Trusted Publishing via OIDC, Sigstore-based attestations, and mandatory 2FA -- but adoption and enforcement vary widely.

---

## 1. npm (Node.js)

### Registry & Publishing Model

The npm registry (registry.npmjs.org) is the largest package registry in existence. Any user can create an account and publish packages using `npm publish`. As of late 2025, publishing options are being restricted to:

- **Local publishing with mandatory 2FA** (TOTP deprecated in favor of WebAuthn/passkeys)
- **Granular access tokens** with 7-day default / 90-day maximum expiration (classic tokens deprecated)
- **Trusted Publishing** via OIDC from GitHub Actions, GitLab CI/CD, or CircleCI

Trusted Publishing automatically generates provenance attestations achieving roughly SLSA Build Level 2.

### Known Attack Vectors

**Lifecycle script execution** is the dominant attack vector. npm's `preinstall`, `postinstall`, and `prepare` scripts execute arbitrary code with the installing user's full privileges during `npm install`. This is the mechanism exploited by the Shai-Hulud worm, Glassworm, and virtually every major npm compromise.

**Typosquatting** remains the most common entry point. Packages with names like `lodsash`, `expres`, `reacts` have been uploaded targeting misspellings of popular packages. This has evolved from opportunistic to industrialized, with automated campaigns publishing hundreds of variants.

**Slopsquatting** is a new vector where attackers register package names that AI coding assistants frequently hallucinate. Research shows LLMs recommend non-existent packages ~20% of the time, and 43% of hallucinated names recur across repeated queries, making them predictably exploitable.

**Account takeover** is the highest-impact vector. The September 2025 "Qix" attack compromised Josh Junon's account via a phishing email mimicking npm support, cascading into 18 packages including chalk, debug, ansi-styles, and strip-ansi (2.6 billion combined weekly downloads). The payload targeted cryptocurrency wallets.

**Dependency confusion** exploits the resolution order when both public and private registries are configured. Attackers publish higher-version packages on the public registry matching internal package names. Alex Birsan's 2021 research demonstrated this against Apple, Microsoft, Tesla, Uber, and dozens of others, earning over $130,000 in bounties.

**Abandoned package takeover**: The event-stream incident (2018) demonstrated this -- an inactive maintainer handed over the package to a malicious actor who added a cryptocurrency-stealing dependency (flatmap-stream). The malicious code specifically targeted Copay Bitcoin wallet users and went undetected for months.

**Protestware/maintainer sabotage**: In January 2022, Marak Squires sabotaged colors.js (20M+ weekly downloads) and faker.js (2.8M+ weekly downloads) by introducing infinite-loop gibberish output. In March 2022, the node-ipc maintainer added the peacenotwar dependency that overwrote files on Russian/Belarusian systems with heart emojis during the Ukraine invasion.

**Wormable propagation**: The September 2025 Shai-Hulud worm was the first self-propagating malware in npm history. It harvested npm tokens and GitHub PATs from infected machines, then automatically infected and republished packages those tokens had access to. It compromised 500+ packages before containment, with subsequent waves (Glassworm, Bitwarden compromise, SAP CAP packages) demonstrating the pattern is repeatable.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| event-stream | Nov 2018 | Crypto theft via hijacked maintainer account; 1.5M weekly downloads |
| ua-parser-js | Oct 2021 | Crypto miner + password stealer; 7M weekly downloads |
| colors.js / faker.js | Jan 2022 | Maintainer sabotage; broke thousands of apps |
| node-ipc / peacenotwar | Mar 2022 | Protestware; file destruction on Russian/Belarusian systems |
| Qix phishing cascade | Sep 2025 | 18 packages, 2.6B weekly downloads; crypto wallet theft |
| Shai-Hulud worm | Sep 2025 | First npm worm; 500+ packages auto-infected |
| Nx build system | Aug 2025 | Build system compromise |
| Axios | Mar 2026 | Account takeover; 100M+ weekly downloads |
| @bitwarden/cli | Apr 2026 | TeamPCP backdoor; security tool supply chain |

### Registry Protections

- **Mandatory 2FA** for high-impact packages (moving to FIDO-based, TOTP deprecated)
- **Granular access tokens** with short expiration (7-day default, 90-day max)
- **Trusted Publishing** via OIDC (GitHub Actions, GitLab CI/CD, CircleCI)
- **Provenance attestations** (automatic with Trusted Publishing)
- **Package signing** (repository signatures by npm, author signatures via provenance)
- **Scoped packages** (`@org/package`) for namespace isolation
- **GitHub secret scanning** for leaked npm tokens in repositories

**Critical gap**: The npm CLI itself provides zero consumer-side protections. Scripts run by default, no release cooldown, no per-package allowlisting, no trust policy enforcement. Alternative package managers (pnpm v11, Yarn Berry v4, Bun) have filled this gap with script blocking (default on), release cooldown periods (1-3 days default), and trust policy verification.

---

## 2. PyPI (Python)

### Registry & Publishing Model

PyPI is the primary Python package repository. Publishing uses `twine upload` or the `pypa/gh-action-pypi-publish` GitHub Action. Authentication methods:

- **API tokens** (scoped to specific projects or account-wide)
- **Trusted Publishing** via OIDC (GitHub Actions, GitLab CI/CD including self-managed, custom OIDC issuers)

Over 50,000 projects use Trusted Publishing, covering 20%+ of all file uploads. 52% of active users have enabled non-phishable 2FA.

### Known Attack Vectors

**setup.py execution**: Python's traditional packaging model executes `setup.py` at install time. While modern packaging has moved toward declarative `pyproject.toml` and build backends, many packages still use `setup.py` which runs arbitrary Python code during `pip install`. This is the primary mechanism for PyPI malware.

**Typosquatting**: PyPI has been flooded with typosquats -- `requets` for `requests`, `colorama-py` for `colorama`, `selemium` for `selenium`. Hundreds are removed monthly. The attack pattern is consistent: setup.py exfiltrates environment variables and SSH keys, often drops persistence.

**Dependency confusion**: PyPI has no namespace/scope system, making it particularly vulnerable. Internal package names used by organizations can be claimed on the public registry. Over 5,000 dependency confusion copycats have been found on PyPI.

**Cryptocurrency-targeted campaigns**: 9 of 23 crypto-related malicious campaigns in 2024 targeted PyPI. 128 phantom packages accumulated 121,539 downloads between July 2025 and January 2026.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| Colorama typosquat (Colorizr) | 2024 | Name confusion targeting popular terminal color library |
| aliyun-ai-labs-snippets-sdk | May 2024 | Fake SDK; removed within 24 hours |
| termncolor/sisaws/secmeasure | Jul-Aug 2025 | Typosquatting campaign series |
| Telnyx package (TeamPCP) | 2025 | Malicious versions 4.87.1/4.87.2 exfiltrating data |
| dYdX compromise | Feb 2026 | npm+PyPI dual-ecosystem wallet stealer campaign |
| Ultralytics compromise | 2025 | Compromised project using Trusted Publishing; attestations helped audit |

### Registry Protections

- **Trusted Publishing** via OIDC (most mature implementation across ecosystems)
- **Sigstore attestations** (17% of uploads include attestations; keyless signing via Fulcio+Rekor)
- **2FA enforcement** (email verification for TOTP; 52% non-phishable 2FA adoption)
- **Malware response**: 66% of reports handled within 4 hours, 92% within 24 hours
- **Proactive protections**: Typosquatting flagging, phishing detection, domain resurrection prevention
- **Organizations** feature for team-based package management (7,742 orgs as of 2025)

**Key gap**: No namespace/scope system exists. Any user can claim any unclaimed package name. No install-time script sandboxing. No release cooldown mechanism at the registry level.

---

## 3. Cargo / crates.io (Rust)

### Registry & Publishing Model

crates.io is the Rust package registry. Publishing uses `cargo publish` with an API token. crates.io notably does **not** support multi-factor authentication for publishing, making API tokens the sole authentication factor. Namespace policy: crate names are globally unique, first-come-first-served, with no scope/namespace system.

**Trusted Publishing** was introduced in July 2025 for GitHub Actions and later expanded to GitLab CI/CD. Over 770 packages have configured it, including high-profile crates like pyo3 and cc. Tokens are scoped to specific crates with 30-minute lifespans.

**Limitation**: First-version crate publishing via Trusted Publishing is not yet supported; a temporary token is needed for initial publication.

### Known Attack Vectors

**Build script execution (build.rs)**: Cargo executes `build.rs` build scripts at compile time with full system access. This is architecturally similar to npm's lifecycle scripts. Additionally, procedural macros run arbitrary code at compile time. rust-analyzer executes `cargo check` on project open, creating a potential 0-click RCE vector.

**Configuration-based attacks**: `.cargo/config.toml` in a repository can redefine the Rust compiler path, set rustc-wrapper, configure linker paths, and manipulate environment variables -- all potential code execution paths.

**Typosquatting**: The faster_log and async_println crates (May 2025) amassed 8,424 downloads while exfiltrating private wallet keys. The attack used a functional logger with a familiar name and copied design/README to pass casual review.

**Proxy caching gap**: Unlike Go, crates.io does not have an immutable proxy cache -- but the Cargo.lock file contains content hashes that detect modifications to previously-downloaded versions.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| faster_log / async_println | May 2025 | Wallet key exfiltration; 8,424 downloads |
| Crypto wallet stealers | Sep 2025 | Solana/Ethereum key theft |

### Registry Protections

- **Trusted Publishing** via OIDC (GitHub Actions, GitLab CI/CD; since July 2025)
- **Cargo.lock content hashes** detect tampering with cached versions
- **Security tab on crate pages** showing RustSec advisories
- **cargo-audit**: Checks for known vulnerabilities via RustSec database
- **cargo-deny**: Policy engine for licenses, bans, advisories, sources
- **cargo-vet**: Code review tracking with third-party audit imports (Google, Mozilla, Bytecode Alliance, ISRG)
- **No namespace reservation** system
- **No MFA support** for direct publishing (Trusted Publishing is the workaround)

**Key gap**: Build scripts and proc macros execute arbitrary code at build time with no sandboxing. No MFA for token-based publishing. No namespace/scope system. Crate names are first-come-first-served.

---

## 4. Go Modules

### Registry & Publishing Model

Go modules are architecturally unique: **there is no package registry account**. Module paths are URLs (typically GitHub repository paths) and the `go` tool fetches source directly from version control. There is no upload step, no maintainer account, and no separate credentials to compromise.

The **Go Module Proxy** (proxy.golang.org) is a caching proxy, not a registry. It runs `go mod download` and caches results. Authors don't register or upload -- the proxy caches on first access.

The **Go Checksum Database** (sum.golang.org) is a global, append-only, cryptographically-verifiable transparency log of module content hashes. Every `go get` operation verifies downloaded content against this database, ensuring:
- Version contents are globally immutable (no targeted backdoors)
- No key management required from module authors
- Requires no opt-in; works by default

### Known Attack Vectors

**Typosquatting via vanity imports**: Since module paths map to repository URLs, attackers create repositories with similar names (e.g., `github.com/boltdb-go/bolt` vs legitimate `github.com/boltdb/bolt`).

**Proxy caching persistence**: The BoltDB attack exploited a critical architectural tension: once a malicious version is cached in the proxy, the attacker can rewrite the GitHub repository to show clean code. Developers auditing the GitHub repo see clean code, but `go get` delivers the cached malicious version. This persisted for over three years. The backdoor embedded a persistent TCP connection for remote command execution.

**No install hooks** -- Go explicitly rejected post-install scripts. Code only executes during `go test` or binary execution. This eliminates the largest attack vector present in npm, PyPI, and Cargo.

**Minimal Version Selection (MVS)**: Unlike other ecosystems that resolve "latest compatible," Go uses MVS, meaning transitive dependencies use the versions specified in their own go.mod -- not the latest. A compromised new version doesn't automatically flow to consumers.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| BoltDB typosquat (boltdb-go/bolt) | 2021-2025 | 3+ year persistence via proxy cache; RCE backdoor; 8,367 dependents |
| MongoDB module impersonation | 2025 | GitLab caught module impersonation attempt |
| BufferZoneCorp sleeper attack | May 2026 | CI/CD pipeline draining via Go modules |

### Registry Protections

- **No registry accounts** to compromise -- VCS is the source of truth
- **Global Checksum Database** (sum.golang.org) ensures immutable, globally-consistent module contents
- **No install hooks** -- fetching/building code never executes it
- **go.sum file** with content hashes checked into version control
- **Minimal Version Selection** prevents automatic uptake of compromised new versions
- **VCS sandbox** in the proxy (only git/Mercurial enabled by default)
- **Module proxy availability** prevents "left-pad" scenarios
- **Cultural mitigation**: Strong "zero dependencies" culture, rich standard library

**Key gap**: Once a malicious version is cached in the proxy, removal is difficult. No automated malware scanning. Typosquatting remains possible through similar repository URLs. The checksum database guarantees consistency but not safety -- it ensures everyone gets the same (potentially malicious) code.

---

## 5. Maven Central / Gradle (Java/Kotlin)

### Registry & Publishing Model

Maven Central is the primary Java/Kotlin repository, operated by Sonatype. Publishing requires:

- **Namespace verification**: Publishers must prove domain ownership for their groupId (e.g., owning `mycompany.com` for `com.mycompany`). GitHub-based publishers can use `io.github.<username>`.
- **GPG signatures**: Every artifact must be cryptographically signed with GPG/PGP
- **Mandatory metadata**: POM file with coordinates, project info, license, developer info, SCM details
- **Javadoc and sources**: Required alongside binary artifacts
- **File checksums**: MD5 and SHA1 mandatory; SHA256/SHA512 optional

This is the most rigorous publishing process of any ecosystem. The domain verification requirement is the single strongest anti-typosquatting measure available.

### Known Attack Vectors

**MavenGate (January 2024)**: Exploits abandoned libraries by purchasing expired domains used as groupIds. Of 33,938 domains analyzed, 6,170 (18.18%) were vulnerable. Sonatype responded by disabling accounts with expired domains.

**Maven-Hijack**: Exploits Maven's packaging order and Java classloading. Attackers inject malicious classes with the same fully qualified name as legitimate ones, relying on classpath ordering for execution priority.

**Dependency confusion**: Rare in Maven due to groupId namespace verification, but possible when organizations don't register their internal groupIds. Maven resolves from whichever repository provides a version first.

**Cross-ecosystem worm spread**: Shai-Hulud v2 (November 2025) spread from npm to Maven Central by rebundling compromised npm components as Java dependencies.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| Brandjacking malware | Dec 2020-Jan 2021 | Typosquatting packages on Maven Central |
| MavenGate | Jan 2024 | 18% of domains vulnerable to expired-domain takeover |
| Shai-Hulud v2 crossover | Nov 2025 | npm worm components rebundled for Maven |

### Registry Protections

- **Domain-verified namespaces** (strongest anti-typosquatting of any ecosystem)
- **Mandatory GPG signatures** on all artifacts
- **SLSA provenance** (introduced 2025 for new artifacts)
- **Immutable releases** (versions cannot be overwritten or deleted once published)
- **Sonatype malware scanning** via automated analysis
- **Expired domain monitoring** (post-MavenGate)

**Key gap**: GPG signing verifies that the signer has a key, but keys aren't bound to verified identities by default. Build plugins (Maven/Gradle) can execute arbitrary code. No install-time script blocking mechanism. The publishing process is heavy, which deters casual attackers but also creates friction for legitimate maintainers.

---

## 6. NuGet (.NET)

### Registry & Publishing Model

NuGet.org is the primary .NET package registry. Publishing uses `dotnet nuget push` or `nuget push`. Authentication:

- **API keys** (scoped to packages, with expiration)
- **Mandatory 2FA** for all nuget.org accounts (100% adoption achieved)
- **Package ID prefix reservation** restricts publishing under reserved namespaces to authorized users

NuGet supports both **author signing** (optional, using developer certificates) and **repository signing** (all packages on nuget.org are repository-signed automatically).

### Known Attack Vectors

**Typosquatting**: Packages mimicking legitimate Microsoft or utility libraries. The NCryptYo package (August 2024) masqueraded as NCrypto targeting ASP.NET developers, using JIT hooking for credential theft.

**Time-delayed logic bombs**: The shanhai666 campaign (published 2023-2024) embedded malware in ICS-targeting packages with trigger dates set for August 2027 and November 2028 -- designed to remain dormant for years before activating.

**MSBuild execution**: NuGet packages can include MSBuild props/targets files that execute at build time, similar to npm lifecycle scripts but requiring the package to be restored and the project to be built.

**Dependency confusion**: Possible when using multiple package sources. NuGet's Package Source Mapping (introduced to counter this) allows declaring which source each package should come from.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| NCryptYo (ASP.NET targeting) | Aug 2024 | JIT hooking, credential theft |
| shanhai666 ICS targeting | 2023-2024 | Logic bombs with 2027/2028 trigger dates |
| PowerShell loader packages | 2023-2024 | Obfuscated scripts in fake utility libraries |

### Registry Protections

- **Mandatory 2FA** for all accounts (strongest enforcement of any ecosystem)
- **Package ID prefix reservation** for namespace protection
- **Repository signing** on all packages (automatic)
- **Optional author signing** with certificates
- **NuGetAudit** (since .NET 8) warns about vulnerable packages during restore
- **Package Source Mapping** to control package source resolution
- **Central Package Management** for version standardization
- **Lock files** with content hashes
- **HTTPS everywhere** for all NuGet interactions
- **Vulnerability notifications** for known CVEs

**Planned**: OpenID Connect authentication, build provenance tracking, verified publisher badges, SBOMs, automated vulnerability remediation.

**Key gap**: No Trusted Publishing via OIDC yet (planned). MSBuild targets can execute arbitrary code. Package signing is optional for authors and not widely adopted. No release cooldown mechanism.

---

## 7. RubyGems (Ruby)

### Registry & Publishing Model

RubyGems.org is the Ruby package registry. Publishing uses `gem push`. Authentication:

- **API keys** (gem-specific or account-wide)
- **MFA enforcement**: Required for owners of top ~370 gems (>180M total downloads). Universal MFA under consideration.
- **Trusted Publishing** via OIDC (GitHub Actions; announced late 2023)

### Known Attack Vectors

**Account takeover**: The rest-client gem (August 2019) was compromised when a maintainer's account credentials were stolen. Malicious versions 1.6.10-1.6.13 siphoned URLs and environment variables. Downloaded ~1,000 times before removal.

**strong_password hijack**: A malicious version 0.0.7 (2020) allowed remote code execution via pastebin.com payloads, triggered only in production environments. 537 downloads of the malicious version.

**Typosquatting**: Gems named `httparty` (vs `httparty`), `json-web-token`, and others mimicking popular gems. 60+ malicious gems posing as social media automation tools delivered functionality while exfiltrating credentials.

**Gemspec install hooks**: Ruby gems can define `extensions` that compile native code and execute arbitrary commands during installation, similar to npm lifecycle scripts.

### Notable Real-World Incidents

| Incident | Date | Impact |
|----------|------|--------|
| rest-client backdoor | Aug 2019 | Account takeover; credential exfiltration |
| strong_password RCE | 2020 | Remote code execution via hijacked gem |
| bootstrap-sass compromise | 2019 | Malicious code injection |
| Social media tool typosquats | 2024-2025 | 60+ gems with credential theft |
| BufferZoneCorp sleeper | May 2026 | CI/CD pipeline targeting |

### Registry Protections

- **MFA enforcement** for top gems (>180M downloads)
- **Trusted Publishing** via OIDC (GitHub Actions)
- **Sigstore integration** (in progress; funded by AWS)
- **Bundler lockfile checksums** for tamper detection
- **Mend.io automated scanning** for vulnerabilities
- **Manual malware reviews** by RubyGems team
- **Trail of Bits security audit** (late 2024; 33 findings, 1 high-severity)
- **CVE-2024-21654 MFA bypass** identified and fixed

**Key gap**: MFA not universal (only top gems). Sigstore integration still in progress. No namespace/scope system. gem extensions can execute arbitrary code. Limited automated malware detection. Smallest security team among major registries.

---

## Cross-Ecosystem Comparison

### Attack Vector Matrix

| Attack Vector | npm | PyPI | Cargo | Go | Maven | NuGet | RubyGems |
|--------------|-----|------|-------|-----|-------|-------|----------|
| Install-time code execution | **HIGH** (lifecycle scripts) | **HIGH** (setup.py) | **HIGH** (build.rs, proc macros) | **NONE** | **MEDIUM** (build plugins) | **MEDIUM** (MSBuild targets) | **MEDIUM** (extensions) |
| Typosquatting | **HIGH** | **HIGH** | **MEDIUM** | **LOW** (URL-based) | **LOW** (domain-verified) | **MEDIUM** | **HIGH** |
| Account takeover | **HIGH** (most targeted) | **MEDIUM** | **MEDIUM** | **N/A** (no accounts) | **LOW** | **LOW** | **MEDIUM** |
| Dependency confusion | **HIGH** | **HIGH** | **MEDIUM** | **LOW** | **LOW** (domain-verified) | **MEDIUM** | **MEDIUM** |
| Abandoned package takeover | **HIGH** | **MEDIUM** | **LOW** | **LOW** | **MEDIUM** (MavenGate) | **LOW** | **MEDIUM** |
| Protestware/sabotage | **HIGH** (proven) | **LOW** | **LOW** | **LOW** | **LOW** | **LOW** | **LOW** |
| Wormable propagation | **PROVEN** | **LOW** | **LOW** | **NONE** | **LOW** | **LOW** | **LOW** |

### Protection Matrix

| Protection | npm | PyPI | Cargo | Go | Maven | NuGet | RubyGems |
|-----------|-----|------|-------|-----|-------|-------|----------|
| Mandatory 2FA | Partial (high-impact) | Partial (52% non-phishable) | **NO** | N/A | Implicit (domain) | **YES** (100%) | Partial (top gems) |
| Trusted Publishing | YES | YES (most mature) | YES (since Jul 2025) | N/A | NO | NO (planned) | YES |
| Provenance/Attestations | YES (SLSA L2) | YES (Sigstore) | NO | N/A (sumdb serves similar) | YES (SLSA, 2025) | NO (planned) | In progress (Sigstore) |
| Namespace verification | Scopes (optional) | **NO** | **NO** | Implicit (URLs) | **YES** (domain) | YES (prefix reservation) | **NO** |
| Content hash verification | package-lock.json | pip hash checking | Cargo.lock | go.sum + sumdb | Checksums | Lock files | Bundler checksums |
| Malware scanning | Limited | YES (reactive) | NO (community tools) | NO | YES (Sonatype) | Limited | YES (Mend.io + manual) |
| Install script controls | **NO** (npm CLI) / YES (pnpm/yarn/bun) | **NO** | **NO** | N/A (no scripts) | **NO** | **NO** | **NO** |
| Release cooldown | **NO** (npm CLI) / YES (pnpm 1d, yarn 3d) | **NO** | **NO** | N/A | **NO** | **NO** | **NO** |

### Architectural Security Ranking (Best to Worst)

1. **Go modules** -- No registry accounts, no install hooks, global checksum transparency, VCS as source of truth, Minimal Version Selection. Architecturally the most secure by design.

2. **Maven Central** -- Domain-verified namespaces, mandatory GPG signatures, immutable releases, rigorous publishing requirements. Highest barrier to entry deters casual attackers.

3. **NuGet** -- Universal 2FA, prefix reservation, repository signing, Package Source Mapping. Strong institutional backing from Microsoft.

4. **Cargo/crates.io** -- Content-hashed lockfiles, strong tooling ecosystem (cargo-vet, cargo-deny, cargo-audit), Trusted Publishing. Weakened by build.rs execution and no MFA.

5. **RubyGems** -- MFA for top gems, Trusted Publishing, active security investment. Limited by small team, partial MFA enforcement, and in-progress Sigstore integration.

6. **PyPI** -- Most mature Trusted Publishing and attestation infrastructure. Severely weakened by no namespace system, setup.py execution, and massive typosquatting surface.

7. **npm** -- Most active threat landscape. Strong publisher-side improvements (Trusted Publishing, granular tokens, provenance) but the npm CLI itself provides zero consumer-side protections. The alternative package managers (pnpm, Yarn, Bun) provide the protections npm CLI lacks.

### Key Insight: Publisher vs Consumer Security

The most important architectural distinction is between **publisher-side protections** (who can publish, how publishing is authenticated, provenance of builds) and **consumer-side protections** (what happens when a developer installs a package). Most ecosystems have invested heavily in publisher-side security while leaving the consumer side largely unprotected.

- **Publisher security** (Trusted Publishing, 2FA, attestations) prevents unauthorized publication but cannot stop a legitimate maintainer from publishing malicious code or a compromised CI pipeline from producing tainted builds.
- **Consumer security** (install script blocking, release cooldowns, trust policies) prevents already-published malicious code from executing on developer machines. Only pnpm v11, Yarn Berry v4, and Bun provide this for the JavaScript ecosystem. Go provides it architecturally. All other ecosystems largely lack consumer-side defenses.

The asymmetry means that even with perfect publisher authentication, a single compromised maintainer or CI pipeline can still deliver malicious code to every consumer.

---

## Sources

All raw source material is saved in `docs/`:
- `go-supply-chain-mitigations-blog.md` -- Go official blog on supply chain mitigations
- `npm-trusted-publishing-docs.md` -- npm Trusted Publishing documentation
- `pypi-2025-year-in-review.md` -- PyPI 2025 year in review with security stats
- `maven-central-publishing-requirements.md` -- Maven Central publishing requirements
- `nuget-supply-chain-security-measures.md` -- NuGet supply chain security blog
- `ruby-central-security-strengthening.md` -- Ruby Central security initiatives
- `npm-threat-landscape-unit42.md` -- Unit 42 npm threat landscape analysis
- `npm-supply-chain-security-2026-mondoo.md` -- npm 2026 consumer/publisher defense analysis
- `rust-supply-chain-security-practices.md` -- Rust supply chain security practices
- `go-boltdb-typosquatting-attack.md` -- BoltDB typosquatting attack analysis
- `pypi-attestation-security-model.md` -- PyPI attestation security model
- `nuget-security-best-practices-microsoft.md` -- NuGet security best practices (Microsoft)
