# Signature Verification & Provenance Attestation Across Package Ecosystems

## Executive Summary

Cryptographic signatures and provenance attestation are the most structurally important defenses against package supply chain attacks, because they answer two fundamental questions: "Who built this?" and "Was it built from the claimed source code?" This report surveys the state of these defenses across all major package ecosystems as of May 2026.

The landscape divides into three tiers:

1. **Sigstore-native ecosystems** (npm, PyPI, RubyGems) — have SLSA L3-capable provenance infrastructure with Sigstore keyless signing, but consumer-side enforcement is weak or absent
2. **Integrity-only ecosystems** (Go, Cargo/crates.io) — provide strong tamper detection via checksums and transparency logs, but lack cryptographic identity binding to publishers
3. **Legacy-signing ecosystems** (Maven Central, NuGet, RubyGems legacy) — require or support traditional PKI/PGP signatures with varying enforcement, but face key management friction

The critical finding: **no major ecosystem currently allows consumers to require provenance at install time as a default**. The infrastructure is being built publisher-side, but consumer enforcement ranges from "opt-in manual check" (npm, PyPI) to "nonexistent" (Go, crates.io). The sole exception is pnpm's `trustPolicy: no-downgrade`, which is opt-in and detects credential compromise rather than requiring provenance.

---

## 1. SLSA (Supply-chain Levels for Software Artifacts)

### What It Is

SLSA (pronounced "salsa") is an OpenSSF-backed framework defining incrementally adoptable security levels for software build and distribution processes. It provides a common vocabulary for describing supply chain security maturity.

### The Levels

| Level | Name | Key Requirement | Guarantee |
|-------|------|-----------------|-----------|
| L0 | No guarantees | None | No provenance exists |
| L1 | Provenance exists | Build platform auto-generates provenance | You know *how* the artifact was built |
| L2 | Hosted build | Hosted platform generates and *signs* provenance | Provenance is authenticated; post-build tampering detectable |
| L3 | Hardened builds | Build isolation; secrets separated from user-defined steps | Insider threats and compromised credentials mitigated |

SLSA v1.2 organizes requirements into two tracks:
- **Build Track** (L1-L3): Covers how artifacts are built and how provenance is generated
- **Source Track**: Covers source code integrity (version control, code review)

### What Each Level Actually Proves

- **L1**: The build process is documented. A consumer can see *what* built the artifact but has no guarantee the provenance wasn't forged.
- **L2**: The provenance is digitally signed by the build platform (not the developer). A consumer can verify the provenance wasn't tampered with after the build. This is approximately what npm `--provenance` and PyPI Trusted Publisher attestations achieve.
- **L3**: The build ran in an isolated environment where other tenants and even the developer cannot influence the build process or inject secrets. GitHub Actions hosted runners achieve this for most practical purposes.

### Ecosystem Adoption

| Ecosystem | SLSA Level Achieved | Notes |
|-----------|-------------------|-------|
| npm | L3 (via GitHub Actions) | Provenance automatic with Trusted Publishing |
| PyPI | L3 (via GitHub Actions) | Attestations auto-generated with Trusted Publishing |
| GitHub Releases | L3 | Language-agnostic via `actions/attest-build-provenance` |
| Maven Central | L3-capable | Sigstore opt-in; PGP mandatory; <1% Sigstore adoption |
| RubyGems | L3-capable | Attestations supported since rubygems 3.6.0; early adoption |
| crates.io | L1 | SHA-256 checksums only; Sigstore RFC pending |
| Go modules | L1 | Checksum transparency only (no identity binding) |
| NuGet | L1 | Repository signatures provide integrity, not provenance |

**Sources**: `docs/slsa-spec-v1.2-about.md`, `docs/slsa-build-track-basics.md`, `docs/2026-state-of-registry-provenance.md`

---

## 2. Sigstore (Cosign, Rekor, Fulcio)

### Architecture

Sigstore is a suite of open-source tools that provides "keyless" signing — developers never manage long-lived signing keys. Instead, identity (via OIDC) is used as the root of trust.

**Components:**

- **Fulcio** — Certificate Authority that issues short-lived X.509 certificates (valid ~10 minutes). It verifies an OIDC token from an identity provider, then issues a certificate binding the caller's identity to an ephemeral public key.
- **Rekor** — Transparency log (append-only Merkle tree) that records every signing event. Entries are immutable and publicly auditable. Provides timestamping.
- **Cosign** — Client tool that orchestrates the signing flow: generates ephemeral keypair, obtains OIDC token, gets Fulcio certificate, signs artifact, records in Rekor, destroys private key.

### How Keyless Signing Works

1. Developer/CI authenticates with an OIDC provider (GitHub, Google, Microsoft)
2. Fulcio verifies the OIDC token and issues a short-lived certificate binding the OIDC identity to an ephemeral public key
3. The artifact is signed with the ephemeral private key
4. The signature + certificate are recorded in Rekor with a timestamp
5. The private key is destroyed — it existed only in memory for seconds
6. Verification: compare signature against the Rekor-timestamped certificate; confirm the signing happened during the certificate's validity window

### Root of Trust

Sigstore distributes its root of trust (Fulcio's root CA certificate and Rekor's public key) through The Update Framework (TUF), protecting against key compromise and rollback attacks.

### Supported OIDC Providers

- GitHub (including automatic detection in GitHub Actions)
- Google (including automatic detection on GCP)
- Microsoft
- GitLab
- Custom OIDC providers (for self-hosted Sigstore instances)

### Package Ecosystem Integration

| Ecosystem | Integration Status | Mechanism |
|-----------|-------------------|-----------|
| npm | Production | Provenance attestations signed via Sigstore |
| PyPI | Production | PEP 740 attestations signed via Sigstore |
| Maven Central | Production (opt-in) | `.sigstore.json` bundles alongside PGP |
| RubyGems | Early production | `--attestation` flag in rubygems >= 3.6.0 |
| Homebrew | Production | Bottle provenance via Sigstore |
| Kubernetes | Production | Policy Controller for image verification |
| crates.io | RFC stage | RFC #3403 pending |
| Go | Not integrated | Checksums only |
| NuGet | Not integrated | X.509 PKI only |

### What Sigstore Does NOT Do

- Does not verify *what* the code does — only *who* built it and *from where*
- Does not replace code review or vulnerability scanning
- Does not guarantee the OIDC identity itself is trustworthy
- Does not work for self-hosted CI runners (in most ecosystem integrations)

**Sources**: `docs/sigstore-cosign-signing-overview.md`, `docs/2026-state-of-registry-provenance.md`

---

## 3. npm Provenance

### How It Works

npm provenance creates a cryptographic link between a published package, its source commit, and its CI/CD build environment. When `npm publish --provenance` runs in a supported CI environment:

1. The CI environment issues an OIDC token identifying the workflow, repository, and commit
2. npm CLI sends the token to Sigstore's Fulcio, which issues a short-lived certificate
3. The package tarball is signed with the ephemeral key
4. The signature is recorded in Sigstore's Rekor transparency log
5. The provenance attestation is uploaded to the npm registry alongside the package

### Supported CI/CD Systems

- **GitHub Actions** — cloud-hosted runners only; requires `permissions: id-token: write`
- **GitLab CI/CD** — GitLab.com shared runners; requires `id_tokens` configuration
- **CircleCI** — cloud only (provenance not available, only trusted publishing)

Self-hosted runners are NOT supported.

### Trusted Publishing (OIDC)

npm's Trusted Publishing (GA) eliminates access tokens entirely:
- Configure trust relationship in npmjs.com UI (repository, workflow file, etc.)
- CI/CD workflow authenticates via OIDC — no npm token stored anywhere
- Provenance attestations generated automatically (no `--provenance` flag needed)
- Short-lived tokens valid only for the duration of the publish operation

### Consumer Verification

```bash
npm audit signatures
```

This reports:
- Registry signatures verified (all packages)
- Provenance attestations verified (packages that have them)

Example output: "1175 packages have verified registry signatures, 142 packages have verified attestations"

### What Provenance Proves and Doesn't Prove

**Proves:**
- The package was built from a specific source commit
- The build ran on a specific CI/CD platform
- The package hasn't been tampered with since signing

**Does NOT prove:**
- The source code is safe or free of malicious intent
- The maintainer account hasn't been compromised (provenance only binds to CI identity)
- The package's dependencies are safe

### Consumer Enforcement: The Gap

**npm CLI provides NO way to require provenance at install time.** There is no `--require-provenance` flag, no config option to reject packages without attestations. `npm audit signatures` is a manual, after-the-fact check.

**pnpm** is the only npm-compatible package manager with enforcement:
- `trustPolicy: no-downgrade` — blocks installs where authentication strength decreases between versions (e.g., a package that previously had provenance suddenly doesn't)
- This detects credential compromise, not absence of provenance

### Adoption

~7% of npm packages have provenance attestations (2026). The low number reflects that provenance requires CI/CD publishing — packages published from developer laptops cannot have provenance.

**Sources**: `docs/npm-provenance-statements.md`, `docs/npm-trusted-publishing.md`, `docs/npm-supply-chain-security-2026-mondoo.md`

---

## 4. PyPI Trusted Publishers & Attestations

### Trusted Publishers (OIDC)

PyPI's Trusted Publishing eliminates long-lived API tokens for package uploads:

1. Project maintainer configures a Trusted Publisher on pypi.org (specifying CI provider, repository, workflow)
2. During CI/CD, the workflow requests an OIDC token from the CI provider
3. PyPI validates the token against the Trusted Publisher configuration
4. PyPI issues a short-lived API token (valid 15 minutes) for the upload

**Supported CI providers:**
- GitHub Actions
- GitLab CI/CD
- Google Cloud Build
- ActiveState

### Attestations (PEP 740)

Since October 2024, PyPI automatically generates Sigstore-signed attestations for all packages published via Trusted Publishing. This is zero-configuration — if you use Trusted Publishing, attestations happen by default.

Attestations follow the in-toto Attestation Framework and are stored as `.provenance` objects in PyPI's JSON Simple API.

### Security Model

**What attestations guarantee:**
- Origin verification — package came from an authorized Trusted Publisher
- Change detection — observers can notice when a project's Trusted Publisher changes (potential takeover indicator)
- Cryptographic binding — identity is bound to a signing key via OIDC

**What attestations do NOT guarantee:**
- Code trustworthiness ("An attestation will tell you *where* a PyPI package came from, but not *whether* you should trust it")
- Safety of the source code or build process
- That the OIDC identity is controlled by a trustworthy actor

### Consumer Verification

**Current state: incomplete.** PyPI provides an Integrity API for programmatic access to attestations, and the `pypi-attestations` package enables verification. However:

- **pip does not verify attestations during install** — there is no `--require-attestation` flag
- Trail of Bits is developing a pip plugin architecture for attestation verification
- PEP 751 (standardized lockfiles) may enable "trust on first use" identity tracking in the future

Manual verification is possible via:
```bash
pip install pypi-attestations
python -m pypi_attestations verify <package>
```

### Adoption

- ~17% of PyPI packages (132,360+) have attestations as of 2026
- Only 5% of the 360 most-downloaded packages have attestations
- ~20,000 packages produce attestations by default via Trusted Publishing
- Two-thirds of top packages haven't released new versions since attestations became default

**Sources**: `docs/pypi-trusted-publishers.md`, `docs/pypi-attestations-security-model.md`, `docs/pypi-attestations-trail-of-bits.md`

---

## 5. Go Module Checksum Database (sum.golang.org)

### Architecture

Go takes a fundamentally different approach from Sigstore-based ecosystems. Rather than signing individual packages, Go uses a **global transparency log of checksums** to ensure every consumer receives the same module content.

**Three services (all default since Go 1.13):**
- **proxy.golang.org** — Module proxy that caches module metadata and source code
- **sum.golang.org** — Checksum database backed by a Merkle tree (transparent log via Trillian)
- **index.golang.org** — Feed of new module versions

### How Verification Works

1. When `go get` downloads a new module version, it queries `sum.golang.org` for the expected checksums
2. The checksum database returns the hash along with a signed tree head and inclusion proof
3. The `go` command verifies the cryptographic proof and compares against the downloaded content
4. Verified checksums are recorded in `go.sum`
5. Subsequent builds verify against `go.sum` entries (no database contact needed)

The transparency log ensures the database cannot serve different checksums to different users without detection (fork attack prevention).

### Configuration

```bash
# Default: use Google's checksum database
GOSUMDB=sum.golang.org

# Exclude private modules from checksum verification
GONOSUMDB=*.internal.company.com,github.com/private-org/*

# Disable entirely (NOT recommended)
GONOSUMDB=*

# Module proxy configuration (independent of checksum DB)
GOPROXY=https://proxy.golang.org,direct
GONOPROXY=*.internal.company.com
```

**Critical behavior:** If the checksum database is unreachable and the module is not in `go.sum`, the build FAILS. This is a fail-closed design that prevents silent downgrade attacks.

### What Attacks This Prevents

- **Man-in-the-middle substitution** — Cannot serve different code to different users
- **Module repository compromise** — Even if GitHub/GitLab is compromised, altered modules are detected
- **Proxy compromise** — Proxies don't need trust; content is cryptographically verified against the checksum DB
- **Disappearing dependencies** — Module proxy caches content permanently

### What This Does NOT Provide

- **No identity binding** — You know the content is consistent globally, but not *who* built it
- **No provenance** — No link back to source commit or build environment
- **No publisher authentication** — Anyone who can push to the module's repository can publish

### Enforcement

Go's checksum database is **enforced by default for all public modules**. This is the strongest consumer-side enforcement of any ecosystem — it's not opt-in, it's opt-out. However, it provides integrity (same bytes for everyone) rather than provenance (who built it from what source).

**Sources**: `docs/go-checksum-database-design.md`, `docs/go-module-mirror-launch.md`

---

## 6. Cargo / crates.io

### Current State: No Cryptographic Signing

As of May 2026, crates.io has:
- **SHA-256 checksums** in the crate index (integrity verification)
- **Trusted Publishing** GA (GitHub Actions July 2025, GitLab CI/CD January 2026) — OIDC-based, eliminates API tokens
- **No cryptographic signatures** on published crates
- **No provenance attestations** linking crates to source repositories
- **No transparency log integration**

### Sigstore RFC Status

RFC #3403 proposes Sigstore integration for cargo/crates.io but remains in the proposal stage. The Rust Foundation announced PKI infrastructure plans in December 2023, aiming to:
1. Sign the crate index (leveraging existing SHA-256 checksums)
2. Create delegated certificates for index entry signing
3. Eventually sign individual crate files

This work appears to be progressing slowly through the RFC process.

### cargo-vet: A Different Model

cargo-vet (developed by Mozilla) takes a fundamentally different approach — **human review attestations rather than build provenance**.

**How it works:**
1. Developers audit third-party crate source code
2. Audits are recorded in `supply-chain/audits.toml` in the project repository
3. CI runs `cargo vet` as a linter — patches fail if they add unaudited dependencies
4. Organizations can import audits from other trusted organizations (e.g., Mozilla, Google)

**Key design choices:**
- **Criteria-based** — audits attest to specific properties (e.g., "safe-to-run", "safe-to-deploy")
- **Differential audits** — review only the diff between versions, not the entire crate
- **Exemptions** — existing dependencies start exempted; teams work down the backlog incrementally
- **Shared ecosystem** — organizations publish their audit files; others can import them

**Practical usage pattern:**
```toml
# supply-chain/config.toml
[imports.mozilla]
url = "https://raw.githubusercontent.com/nickel-org/nickel.rs/main/supply-chain/audits.toml"

[imports.google]
url = "https://chromium.googlesource.com/chromiumos/third_party/rust_crates/+/refs/heads/main/cargo-vet/audits.toml"
```

**What cargo-vet provides that provenance doesn't:**
- Human attestation that code was reviewed and deemed safe for a specific purpose
- Defense against malicious-but-properly-built packages
- Organizational trust delegation

**What cargo-vet lacks:**
- Automation — requires human effort per crate version
- Scalability — doesn't scale to thousands of transitive dependencies without imports
- Build provenance — says nothing about how the crate was built

### Consumer Enforcement

- `cargo vet` can be run in CI as a blocking check — this is the intended enforcement mechanism
- No way to require cryptographic signatures at `cargo install` or `cargo build` time
- Crate checksums are verified against the index automatically

**Sources**: `docs/cargo-vet-how-it-works.md`, `docs/rust-foundation-artifact-signing.md`, `docs/2026-state-of-registry-provenance.md`

---

## 7. Maven Central

### GPG Signing (Mandatory)

Maven Central is unique in **requiring** cryptographic signatures for all published artifacts. Every artifact must be accompanied by a `.asc` PGP/GPG signature file.

**Publisher requirements:**
- Generate a GPG keypair
- Publish the public key to a keyserver (e.g., keys.openpgp.org)
- Sign all artifacts (JAR, POM, sources, javadoc) during the build
- Upload signatures alongside artifacts

**Consumer verification:**
```bash
# Manual: download .asc file and verify
gpg --verify artifact.jar.asc artifact.jar

# Maven plugin: pgpverify-maven-plugin
mvn org.simplify4u.plugins:pgpverify-maven-plugin:check
```

### The PGP Problem

While mandatory, PGP signatures on Maven Central have significant practical limitations:
- **No chain of trust** — signing keys are self-generated; there's no verification that the key belongs to the claimed author
- **Key management burden** — developers must generate, protect, and rotate keys
- **Rare consumer verification** — very few consumers actually verify PGP signatures; most builds skip verification entirely
- **Key server reliability** — public key servers have availability issues

### Sigstore Integration (Optional, Since January 2025)

Maven Central now accepts `.sigstore.json` bundles alongside traditional PGP signatures:

- Publishers can include Sigstore signatures using the `sigstore-maven-plugin`
- The Central Publisher Portal validates Sigstore signatures at upload time
- Invalid Sigstore signatures produce warnings; missing ones don't
- Sigstore signing uses keyless OIDC-based identity (same as npm/PyPI)

**Future plans:** Maven Central "may eventually make both Sigstore and PGP signatures required" depending on adoption, but has "no intention of replacing PGP signatures."

### Consumer Enforcement

- **PGP verification**: The `pgpverify-maven-plugin` can be configured to fail builds on unsigned or unverifiable artifacts, but this is opt-in and rarely used
- **Sigstore verification**: No standard consumer-side verification tooling yet for Sigstore bundles during Maven builds
- **Gradle**: The Signing Plugin supports PGP signing for publishing; the `gradle-witness` plugin can verify dependency hashes

### Adoption

- PGP signatures: 100% (mandatory)
- Sigstore signatures: <1% (opt-in, new)

**Sources**: `docs/maven-central-sigstore-validation.md`

---

## 8. NuGet

### Signature Architecture

NuGet supports two types of X.509-based signatures:

**Author signatures:**
- Created by the package author using a code-signing certificate
- Guarantee the package hasn't been modified since the author signed it
- Portable — valid regardless of where the package is obtained
- Limitation: **only supported by nuget.exe on Windows**

**Repository signatures:**
- Created by the hosting repository (nuget.org)
- Guarantee package integrity within the repository
- **All packages on nuget.org are automatically repository-signed**

### Certificate Requirements

- Code signing certificate valid for `id-kp-codeSigning`
- RSA public key >= 2048 bits
- Must chain to a trusted root authority (trusted by default on Windows)
- RFC 3161 timestamp required for signature longevity

### Consumer Enforcement

NuGet provides the most mature consumer-side enforcement of any ecosystem via `nuget.config`:

```xml
<config>
  <!-- Require all packages be signed by trusted signers -->
  <add key="signatureValidationMode" value="require" />
</config>

<trustedSigners>
  <!-- Trust specific author certificates -->
  <author name="MyCompany">
    <certificate fingerprint="CE40881FF5F..." hashAlgorithm="SHA256" allowUntrustedRoot="false" />
  </author>
  
  <!-- Trust all packages from nuget.org -->
  <repository name="nuget.org" serviceIndex="https://api.nuget.org/v3/index.json">
    <certificate fingerprint="0E5F38F57DC..." hashAlgorithm="SHA256" allowUntrustedRoot="false" />
    <!-- Optionally restrict to specific owners -->
    <owners>microsoft;nuget</owners>
  </repository>
</trustedSigners>
```

**When `signatureValidationMode=require`:**
- All packages must be signed by a certificate fingerprint listed in `trustedSigners`
- Unsigned packages are rejected
- Packages signed with unlisted certificates are rejected
- This is a hard enforcement mechanism — builds fail

**Verification commands:**
```bash
dotnet nuget verify package.nupkg
nuget verify package.nupkg
```

### Practical Limitations

- Author signing is Windows-only (nuget.exe), limiting cross-platform adoption
- Most packages on nuget.org only have repository signatures, not author signatures
- Enforcing `require` mode with author trust would break most dependency graphs
- No provenance/SLSA integration — signatures prove integrity, not build provenance
- Certificates are tied to traditional PKI (not OIDC/Sigstore)

### NuGet and Provenance

NuGet does not have Sigstore integration or SLSA provenance. GitHub Actions can generate separate provenance attestations for NuGet packages using `actions/attest-build-provenance`, but these are GitHub-specific and not integrated into NuGet's signature verification.

**Sources**: `docs/nuget-signed-packages-reference.md`, `docs/nuget-manage-trust-boundaries.md`

---

## 9. RubyGems

### Legacy Signing (gem cert)

RubyGems has supported cryptographic gem signing since version 0.8.11 using self-signed X.509 certificates:

```bash
# Generate signing key
gem cert --build your@email.com

# Sign a gem (configured in gemspec)
spec.signing_key = File.expand_path("~/.ssh/gem-private_key.pem")
spec.cert_chain = ["certs/your_cert.pem"]

# Install with signature verification
gem install gemname -P HighSecurity
```

**Security policies:**
- `NoSecurity` — no verification (default)
- `LowSecurity` — verify signatures if present
- `MediumSecurity` — all signed gems must verify
- `HighSecurity` — all gems must be signed and verify, full chain verification

### Why Legacy Signing Failed

The system is "not widely used" because:
- Requires manual key generation and management per developer
- No chain of trust — self-signed certificates with no root CA
- No key distribution mechanism — consumers must manually trust each author's cert
- Enabling HighSecurity would break nearly all gem installations (most gems are unsigned)
- Bundler's `--trust-policy` flag exists but is rarely used

### Modern: Trusted Publishing & Attestations

RubyGems has adopted the same OIDC-based model as npm and PyPI:

**Trusted Publishing (GA):**
- OIDC-based publishing from GitHub Actions (only supported CI provider)
- Short-lived tokens replace long-lived API keys
- Setup: configure repository/workflow on rubygems.org

**Attestations (rubygems >= 3.6.0):**
- `gem push --attestation` flag adds Sigstore-signed attestations
- Rails and other major gems have begun releasing with attestations
- Adoption tracking: https://segiddins.github.io/are-we-attested-yet/

### Consumer Enforcement

- Legacy `gem cert` policies (HighSecurity) are theoretically enforceable but practically unusable
- No way to require Sigstore attestations during `gem install` or `bundle install`
- No `bundle audit attestations` or equivalent consumer-side check
- SHA-256 checksums available for manual verification but not enforced

**Sources**: `docs/rubygems-security-guide.md`, `docs/rubygems-trusted-publishing.md`

---

## 10. Consumer Enforcement: Cross-Ecosystem Comparison

This section answers the key question: **What can a consumer configure TODAY to require valid signatures or provenance?**

### Enforcement Matrix

| Ecosystem | Can Require Signatures? | Can Require Provenance? | What Breaks? |
|-----------|------------------------|------------------------|--------------|
| **npm** | No | No (pnpm: partial) | N/A — not available |
| **PyPI** | No | No | N/A — not available |
| **Go** | N/A (checksums enforced by default) | No | Nothing — already default |
| **Cargo** | No | No | N/A |
| **Maven** | Yes (pgpverify plugin) | No | Varies — all artifacts have PGP sigs, but key trust is weak |
| **NuGet** | Yes (`signatureValidationMode=require`) | No | Most packages — author signing is rare/Windows-only |
| **RubyGems** | Yes (legacy `-P HighSecurity`) | No | Nearly everything — most gems unsigned |

### Detailed Enforcement Options

**npm**: 
- `npm audit signatures` — manual post-install check only
- pnpm `trustPolicy: no-downgrade` — opt-in, detects provenance *downgrade* (not absence)
- No way to block installs of packages without provenance

**PyPI**:
- `pypi-attestations` package for manual verification
- pip plugin architecture for install-time verification is under development
- PEP 751 may enable "trust on first use" via lockfiles in the future

**Go**:
- Checksum verification is enforced by default — fail-closed
- `GONOSUMDB` to exclude private modules
- This is integrity enforcement, not provenance enforcement

**Cargo**:
- `cargo vet` in CI — blocks PRs adding unaudited dependencies
- No signature or provenance enforcement at build time

**Maven**:
- `pgpverify-maven-plugin` — can fail builds on unverified PGP signatures
- Sigstore verification not integrated into build tooling yet

**NuGet**:
- `signatureValidationMode=require` + `trustedSigners` in nuget.config
- Most practical: trust nuget.org's repository signature (all packages have it)
- Most restrictive: trust only specific author certificates (very few packages qualify)

**RubyGems**:
- `gem install -P HighSecurity` or Bundler `--trust-policy` — theoretically works, practically unusable

### The Fundamental Problem

The infrastructure for provenance is being built primarily on the **publisher side**. Registries (npm, PyPI, RubyGems) can now accept and store Sigstore-signed attestations. But the **consumer side** — the package managers that developers actually use — has not yet implemented enforcement. The gap between "provenance exists in the registry" and "my build fails if provenance is missing" remains almost entirely unbridged.

This creates a window where:
1. Attackers who compromise credentials can publish without provenance, and consumers won't notice
2. Packages that previously had provenance can drop it silently (except with pnpm's no-downgrade policy)
3. The ecosystem cannot transition to mandatory provenance until a critical mass of packages have it

### What Would Help

1. **pip `--require-attestation`** — reject packages without valid attestations (under development)
2. **npm CLI provenance enforcement** — even a `--require-provenance` flag would be progress
3. **Lock file integration** — record provenance identity in lockfiles so changes are detectable
4. **Registry-side enforcement** — registries could require Trusted Publishing for new packages (PyPI has discussed this)
5. **pnpm-style no-downgrade as default** — detect when provenance disappears

---

## Cross-Cutting Analysis

### The Sigstore Convergence

Sigstore is emerging as the universal provenance layer across ecosystems. Its keyless signing model solves the key management problem that killed earlier signing systems (PGP for Maven, gem cert for RubyGems, GPG for others). Ecosystems that adopt Sigstore immediately achieve SLSA L3-capable provenance. The remaining difference between ecosystems is adoption rate, not capability.

### The Provenance Paradox

Provenance proves *where* code was built and *from which source*, but not *what the code does*. A malicious maintainer can publish harmful code through a fully trusted CI/CD pipeline with valid provenance, and every check will pass. Provenance is necessary for supply chain security but not sufficient — it must be combined with code review (cargo-vet model), vulnerability scanning, and behavioral analysis.

### Configure-Once Defenses That Exist Today

For an organization wanting to set up invisible-in-operation defenses:

1. **Go**: Already done. Checksum database is enforced by default. Configure `GONOSUMDB` for private modules.
2. **NuGet**: Set `signatureValidationMode=require` with repository trust for nuget.org. This works today with all public packages.
3. **Maven**: Add `pgpverify-maven-plugin` to parent POM. All Maven Central artifacts have PGP signatures.
4. **Cargo**: Add `cargo vet` to CI pipeline. Import audits from Mozilla, Google, etc.
5. **npm**: Switch to pnpm; enable `strictDepBuilds`, `minimumReleaseAge`, and `trustPolicy: no-downgrade`.
6. **PyPI**: No consumer enforcement available yet. Use `pip-audit` for vulnerability scanning as a partial substitute.
7. **RubyGems**: No practical enforcement available.

### Adoption Trajectory

The trend is clear: all major ecosystems are converging on Sigstore-based provenance with OIDC-based Trusted Publishing. The publisher side is largely solved. The consumer enforcement gap will likely close within 1-2 years as pip, npm, and other tools add verification flags. The strategic move is to adopt Trusted Publishing now on the publisher side so packages have attestations when enforcement arrives.

---

## Open Questions

1. When will pip implement install-time attestation verification? (Trail of Bits plugin architecture is in progress)
2. Will npm CLI ever add provenance enforcement, or will this remain a pnpm-only feature?
3. How will the Rust ecosystem resolve RFC #3403 for Sigstore integration with crates.io?
4. Will Maven Central eventually require Sigstore signatures alongside PGP?
5. Can NuGet's signature model extend beyond Windows-only author signing?
6. What is the trajectory for RubyGems attestation adoption among top gems?
