# Artifact Stores, Package Caches, and Registry Infrastructure Research

> Comprehensive reference for the gdev secure development environment bootstrap addon.
> Covers private registries, binary caches, security scanners, SBOM/provenance tools,
> and how each would be configured by a gdev addon.
>
> Last updated: 2026-05-12
> Research sources: `research-spikes/package-supply-chain-security/` (completed spike),
> plus targeted web research for binary cache and SBOM tools.

---

## Table of Contents

1. [Private Registries / Registry Proxies](#1-private-registries--registry-proxies)
2. [Artifact Keeper (Open-Source Universal Registry)](#2-artifact-keeper)
3. [Binary Cache / Build Artifact Systems](#3-binary-cache--build-artifact-systems)
4. [Package Security Scanning Services](#4-package-security-scanning-services)
5. [SBOM and Provenance Tools](#5-sbom-and-provenance-tools)
6. [gdev Addon Integration Patterns](#6-gdev-addon-integration-patterns)
7. [Recommendations by Organization Profile](#7-recommendations-by-organization-profile)

---

## 1. Private Registries / Registry Proxies

Private registries sit between developers and public registries, proxying, caching, scanning, and policy-gating every package download. Once a developer's package manager is pointed at the private registry, all subsequent installs are silently filtered. This is the highest-leverage "configure once" defense.

### 1.1 JFrog Artifactory + Curation + Xray

**What it does:** Universal artifact management platform with 40+ package format support. Acts as a transparent proxy to upstream registries ("remote repositories"), aggregates multiple sources behind a single URL ("virtual repositories"), and hosts private packages ("local repositories").

**Ecosystems:** npm, PyPI, Maven, Gradle, NuGet, Go, Cargo, RubyGems, Composer (PHP), Docker/OCI, Helm, Conan (C/C++), Conda, CocoaPods, Debian/APK/RPM, Hugging Face, Ansible, Chocolatey, Hex, Swift, Terraform, and 20+ more.

**Security features:**
- **JFrog Curation** (preventive gate): Intercepts package requests *before* caching. Evaluates metadata against configurable policies: CVE severity thresholds, malicious package databases (1,500+ known malicious), license restrictions, unmaintained package detection, community trust signals. Observe-only mode for calibration. Currently supports npm, PyPI, Maven, Go.
- **JFrog Xray** (analytical): Binary-level SCA on stored artifacts. NVD + GitHub Advisories + JFrog proprietary DB (2.8M+ catalogued malicious artifacts). Contextual Analysis for exploitability. EPSS scoring. Policy engine can block downloads, fail builds, trigger alerts.
- **Block Downloads from Cached Remote Repositories** (April 2026): Curation policies now enforce on already-cached packages, closing bypass gap.

**Self-hosted vs cloud:**
- SaaS: Pro from $150/month (25 GB), Enterprise X from $950/month (125 GB), Enterprise+ custom
- Self-Managed: Pro X from $27,000/year, Enterprise X from $51,000/year
- Security add-ons (Curation, Advanced Security) may require higher tiers

**gdev addon configuration:**
```ini
# .npmrc
registry=https://myorg.jfrog.io/artifactory/api/npm/npm-virtual/
//myorg.jfrog.io/artifactory/api/npm/npm-virtual/:_authToken=${JFROG_NPM_TOKEN}

# pip.conf
[global]
index-url = https://myorg.jfrog.io/artifactory/api/pypi/pypi-virtual/simple
trusted-host = myorg.jfrog.io

# settings.xml (Maven)
<mirror>
  <id>artifactory</id>
  <url>https://myorg.jfrog.io/artifactory/maven-virtual/</url>
  <mirrorOf>*</mirrorOf>
</mirror>

# .cargo/config.toml
[registries.artifactory]
index = "sparse+https://myorg.jfrog.io/artifactory/api/cargo/cargo-virtual/index/"

# Environment variables
GOPROXY=https://myorg.jfrog.io/artifactory/api/go/go-virtual,direct
GONOSUMDB=*.internal.company.com
```

---

### 1.2 Sonatype Nexus Repository + Repository Firewall

**What it does:** Artifact repository manager supporting 20+ formats with proxy, group, and hosted repository types. Companion products add security scanning and malicious package blocking.

**Ecosystems:** Maven, npm, Docker, PyPI, RubyGems, NuGet, Helm, Cargo, CocoaPods, Conan, Composer, Conda, Go Modules, Gradle, APT, P2, OBR, OCI, R Lang, Yum, Hugging Face.

**Security features:**
- **Repository Firewall** (separate product): AI-powered malicious package blocking in real-time. Quarantines suspicious components. Dependency confusion protection. Registry-agnostic -- works with Artifactory, Cloudsmith, Azure Artifacts too. Two tiers: Pro (blocking) and Enterprise (full policy engine, governance).
- **Sonatype Lifecycle** (separate product): SCA for vulnerability scanning, license compliance, SBOM generation.
- **Zscaler integration**: Firewall operates at network edge, blocking malicious OSS before reaching any repository manager.

**Self-hosted vs cloud:**
- Community Edition: Free (capped at ~200K requests or ~100K components)
- Nexus Pro: Typically $5,000-$20,000/year for 10-50 developers
- Nexus Cloud: Consumption-based, $3,000-$15,000+/month
- Repository Firewall: Separate commercial product, sales-driven pricing
- Full Platform bundle: Typically $50K+/year

**gdev addon configuration:** Same pattern as JFrog -- point package managers at Nexus group repository URLs. Authentication via tokens or basic auth.

```ini
# .npmrc
registry=https://nexus.myorg.com/repository/npm-group/
//nexus.myorg.com/repository/npm-group/:_authToken=${NEXUS_NPM_TOKEN}

# pip.conf
[global]
index-url = https://nexus.myorg.com/repository/pypi-group/simple
```

---

### 1.3 GitHub Packages

**What it does:** Package hosting integrated with GitHub repositories. Supports publishing and consuming packages.

**Ecosystems:** npm, RubyGems, Maven, Gradle, NuGet, Docker/OCI.

**Security features:** None at registry level. Dependabot (separate feature) scans repos. No policy enforcement, no age-gating, no blocklists.

**Critical limitation: Does NOT proxy upstream registries.** Cannot serve as a "configure once" proxy defense. Developers must maintain access to both GitHub Packages and public registries.

**Self-hosted vs cloud:** Cloud-only. Free for public packages. Private packages: free quota based on GitHub plan.

**gdev addon configuration:** Not recommended as primary registry. Useful only for publishing private packages.

```ini
# .npmrc (only for scoped packages)
@myorg:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

---

### 1.4 GitLab Package Registry + Virtual Registry

**What it does:** Package hosting integrated with GitLab. Virtual Registry (GA since GitLab 18.10) proxies and caches from up to 20 upstream registries behind a single URL.

**Ecosystems:** npm, Maven, PyPI, NuGet, Composer, Conan, Go, Helm, Terraform, Generic. **Virtual Registry supports Maven and container images only.**

**Security features:** Container scanning via GitLab security features. No language package scanning at registry level. No age-gating, no blocklists, no malware detection.

**Self-hosted vs cloud:** Virtual Registry requires Premium ($29/user/month) or Ultimate ($99/user/month).

**gdev addon configuration:** Limited to Maven virtual registry. Not recommended for multi-ecosystem proxying.

---

### 1.5 AWS CodeArtifact

**What it does:** Managed artifact repository service with upstream proxying via "external connections" to one public registry per repository.

**Ecosystems:** npm, PyPI, Maven, NuGet, Swift, Ruby, Cargo, generic packages.

**Security features:**
- **Package Origin Controls**: Per-package ALLOW/BLOCK for upstream access. Defends against dependency confusion. Not retroactive for existing packages.
- No vulnerability scanning, no malware detection, no age-gating.
- IAM-based access control with short-lived tokens (12h default).

**Self-hosted vs cloud:** Cloud-only (AWS). Pay-as-you-go: $0.05/GB-month storage, $0.05/10K requests. Free tier: 2 GB + 100K requests/month.

**gdev addon configuration:**
```bash
# Login command (generates short-lived token)
aws codeartifact login --tool npm --repository my-repo --domain my-domain --region us-east-1

# Environment variables for CI
export CODEARTIFACT_AUTH_TOKEN=$(aws codeartifact get-authorization-token --domain my-domain --query authorizationToken --output text)

# pip
pip config set global.index-url https://aws:${CODEARTIFACT_AUTH_TOKEN}@my-domain-123456789.d.codeartifact.us-east-1.amazonaws.com/pypi/my-repo/simple/
```

**Limitation:** Token refresh required every 12 hours. gdev addon would need to generate a wrapper script or credential helper.

---

### 1.6 Google Artifact Registry

**What it does:** Managed artifact service with remote repository proxying and virtual repository aggregation.

**Ecosystems:** Docker/OCI, Maven, npm, Python, Go, Apt (preview), Yum (preview).

**Security features:**
- Container vulnerability scanning (Container Scanning API)
- Binary Authorization for container deployment policies
- VPC Service Controls for network perimeter
- No language package vulnerability scanning, no age-gating, no malware detection

**Self-hosted vs cloud:** Cloud-only (GCP). Pay-as-you-go: storage + network egress.

**gdev addon configuration:**
```bash
# npm
npm config set registry https://us-npm.pkg.dev/my-project/my-repo/

# pip
pip config set global.index-url https://us-python.pkg.dev/my-project/my-repo/simple/

# Maven (settings.xml)
<repository>
  <id>artifact-registry</id>
  <url>artifactregistry://us-maven.pkg.dev/my-project/my-repo</url>
</repository>
```

---

### 1.7 Azure Artifacts

**What it does:** Package management service with upstream source proxying.

**Ecosystems:** NuGet, npm, Maven, Python (PyPI), Cargo, Universal Packages.

**Security features:**
- **Allow External Versions**: Per-package toggle controlling whether public registry versions can be saved. Dependency confusion protection.
- No vulnerability scanning, no malware detection, no age-gating.

**Self-hosted vs cloud:** Cloud-only (Azure). 2 GiB free per organization. Additional: $2/GiB decreasing at scale.

**gdev addon configuration:**
```ini
# .npmrc
registry=https://pkgs.dev.azure.com/myorg/myproject/_packaging/my-feed/npm/registry/
//pkgs.dev.azure.com/myorg/myproject/_packaging/my-feed/npm/registry/:_authToken=${AZURE_ARTIFACTS_TOKEN}

# pip.conf
[global]
index-url = https://pkgs.dev.azure.com/myorg/myproject/_packaging/my-feed/pypi/simple/

# nuget.config
<packageSources>
  <add key="AzureArtifacts" value="https://pkgs.dev.azure.com/myorg/myproject/_packaging/my-feed/nuget/v3/index.json" />
</packageSources>
```

---

### 1.8 Verdaccio (npm proxy, self-hosted)

**What it does:** Lightweight npm registry that acts as a transparent caching proxy to npmjs.org. The standout open-source option for npm security.

**Ecosystems:** npm only.

**Security features:**
- **Age-gating (`minAgeDays`)**: Hides package versions published within last N days. Setting `minAgeDays: 7` blocks "publish-and-exploit" attacks.
- **Date freezing (`dateThreshold`)**: Serves only versions published before a specific date.
- **Blocklists**: Block scopes (`@evilscope`), packages, or version ranges.
- **Allowlists**: Whitelist internal packages (`@my-org/*`) to bypass age-gating.
- **Replace strategy**: Substitutes blocked versions with nearest older safe version.
- Proxies `npm audit` to upstream.

**Self-hosted vs cloud:** Fully open source (MIT). Free. Self-hosted only. Docker/Kubernetes deployable.

**gdev addon configuration:**
```ini
# .npmrc
registry=http://verdaccio.internal:4873/

# Or via environment variable
NPM_CONFIG_REGISTRY=http://verdaccio.internal:4873/
```

```yaml
# verdaccio config.yaml (server-side)
uplinks:
  npmjs:
    url: https://registry.npmjs.org/
packages:
  '@my-org/*':
    access: $authenticated
    publish: $authenticated
  '**':
    access: $authenticated
    proxy: npmjs
middlewares:
  '@verdaccio/package-filter':
    minAgeDays: 7
    allowlist:
      '@my-org/*': true
```

---

### 1.9 Cloudsmith

**What it does:** Cloud-native multi-format artifact management with security scanning and OPA-based policy engine.

**Ecosystems:** 27+ formats including Alpine, Cargo, Conda, Composer, CRAN, Dart, Debian, Docker, Go, Helm, Hex, Maven, npm, NuGet, Python, RPM, Ruby, Terraform, and more.

**Security features:**
- Vulnerability scanning via OSV.dev, EPSS, OpenSSF malicious package data
- Malware detection for known malicious packages
- License compliance and policy enforcement
- **Policy-as-Code (OPA)**: Rego-based rules for cool-down periods, exploitability prioritization, SBOM inspection
- GPG/PGP signing verification

**Self-hosted vs cloud:** SaaS-only. Pro: $149/month (5 GB storage, 25 GB delivery). **Warning:** Overage charges $1.50/GB beyond included delivery -- real-world costs can be 3-4x base price.

**gdev addon configuration:** Point package managers at Cloudsmith repository URLs with API key authentication.

---

### 1.10 Bytesafe (npm-focused)

**What it does:** Secure proxy for public package registries with firewall registry for centralized policy enforcement.

**Ecosystems:** npm (primary), Maven, NuGet, PyPI.

**Security features:**
- Vulnerability scanning enabled by default for firewall registries
- License compliance enabled by default
- Block Install Scripts: quarantines npm packages with pre/post-install scripts
- Dependency confusion protection

**Self-hosted vs cloud:** SaaS. Community Edition available on GitHub with basic features.

---

## 2. Artifact Keeper

### 2.1 What It Is

Artifact Keeper is an open-source (MIT licensed) universal artifact registry written in Rust, positioning itself as a drop-in replacement for JFrog Artifactory and Sonatype Nexus. It launched in early 2026 and supports 45+ package formats with zero feature gates -- everything ships in the open-source release.

**This is NOT a Claude Code tool or a development utility.** It is a standalone self-hosted artifact registry server comparable to Artifactory/Nexus.

### 2.2 Architecture

| Component | Technology |
|-----------|-----------|
| Backend | Rust + Axum web framework |
| Database | PostgreSQL 16 (JSONB for metadata) |
| Search | OpenSearch (full-text indexing) |
| Storage | Filesystem or S3-compatible backends (content-addressed by SHA-256) |
| Plugin Runtime | Wasmtime (WebAssembly) with WIT-based contracts |
| Frontend | React |
| Container base | DISA STIG-approved Red Hat UBI 9 |

### 2.3 Package Format Support (45+)

**Languages & runtimes:** Maven, npm, PyPI, NuGet, Cargo, Go, RubyGems, Hex, Composer, Pub, CocoaPods, Swift, CRAN, SBT
**Containers & infra:** Docker/OCI, Helm, Terraform, Vagrant
**System packages:** RPM, Debian, Alpine (APK), Conda, OPKG
**Config management:** Chef, Puppet, Ansible
**ML/AI:** HuggingFace, generic ML artifacts
**Other:** Conan, Git LFS, Bazel, P2, VS Code extensions, JetBrains plugins, Protobuf/BSR

Custom formats via WASM plugins (any WASM-compatible language).

### 2.4 Security Features

- **Dual vulnerability scanners**: Trivy (filesystem/container analysis) + Grype (dependency trees)
- **Deduplication**: SHA-256 hashing prevents re-scanning identical artifacts
- **Vulnerability grading**: A-F scoring based on finding severity
- **Policy engine**: Configurable rules to block or quarantine artifacts
- **Artifact signing**: GPG/RSA integrated into Debian, RPM, Alpine, Conda handlers
- **OpenSCAP compliance scanner** for additional hardening
- **Multi-auth**: JWT, OIDC, LDAP, SAML 2.0, API keys

### 2.5 Remote & Virtual Repositories

Supports remote repositories (proxying upstream registries) and virtual repositories (aggregating multiple sources behind a single URL), similar to Artifactory/Nexus.

### 2.6 Replication

"Borg Replication" provides recursive peer-to-peer replication with chunked transfers, network-aware scheduling, and P2P mesh topology for multi-site deployments.

### 2.7 Deployment Options

- Docker Compose (5-minute quickstart)
- Standalone binaries (Linux, macOS; amd64, arm64)
- AWS AMI via Packer
- Kubernetes-compatible containers
- Windows Service (beta)

### 2.8 Pricing

**Completely free.** MIT licensed. No per-user fees. No open-core model. Every feature ships in the open-source release.

### 2.9 Comparison to Alternatives

| Feature | Artifact Keeper | JFrog Artifactory | Sonatype Nexus |
|---------|:-:|:-:|:-:|
| Package formats | 45+ | 40+ | 20+ |
| License | MIT (free) | Commercial ($27K+/yr) | Community (free, capped) / Pro ($5K+/yr) |
| Security scanning | Built-in (Trivy+Grype) | Xray (add-on) | Firewall (add-on) |
| WASM plugins | Yes | No | No |
| Maturity | New (2026) | 15+ years | 15+ years |
| Enterprise adoption | Early | Dominant | Strong |
| Migration API | From Artifactory | N/A | N/A |

**Risk assessment:** Very new project. No track record at enterprise scale. The MIT license and zero-cost model are appealing for budget-constrained teams, but the lack of production battle-testing is a significant concern. Worth monitoring but not recommended as primary choice for production deployments in 2026.

### 2.10 gdev Addon Configuration

Same pattern as Artifactory/Nexus -- point package managers at Artifact Keeper repository URLs:

```ini
# .npmrc
registry=https://ak.internal:8080/api/npm/npm-virtual/

# pip.conf
[global]
index-url = https://ak.internal:8080/api/pypi/pypi-virtual/simple

# Environment variables
GOPROXY=https://ak.internal:8080/api/go/go-virtual,direct
```

---

## 3. Binary Cache / Build Artifact Systems

These systems cache compilation outputs, build artifacts, and derivation results to avoid redundant work across developers and CI.

### 3.1 Cachix (Nix Binary Cache)

**What it does:** Hosted Nix binary cache service. Stores pre-built Nix derivations so other machines can download them instead of building from source. The dominant hosted option for Nix binary caches.

**What it caches:** Nix store paths (compiled packages, derivations, closures). Entries are compressed (up to 90% storage savings). Served via CloudFlare CDN.

**Security model:**
- Nix's native cryptographic signing: all store paths are signed with the cache's private key
- Consumers configure `trusted-public-keys` to verify signatures
- Private caches restrict read/write access per user/team
- Public caches are readable by anyone

**Self-hosted vs cloud:**
- Cloud (primary): Free tier (5 GiB for public/open-source), Starter (50 GiB), Standard (250 GiB), Pro (1.5 TiB). All paid plans have unlimited bandwidth.
- Self-hosted: Available by contacting support
- Annual plans include one complimentary month

| Tier | Storage | Bandwidth | Deploy Agents | Price |
|------|---------|-----------|---------------|-------|
| Free | 5 GiB | Unlimited | 20 | $0 |
| Starter | 50 GiB | Unlimited | 100 | Contact |
| Standard | 250 GiB | Unlimited | 200 | Contact |
| Pro | 1.5 TiB | Unlimited | 1,000 | Contact |

**Additional features:**
- **Cachix Deploy**: Continuous deployment to NixOS/nix-darwin/home-manager profiles using binary cache. GA since January 2026.
- **Upstream caches**: Configurable list of fallback caches.

**gdev addon configuration:**
```nix
# devenv.nix or flake.nix
{
  nixConfig = {
    extra-substituters = [ "https://myorg.cachix.org" ];
    extra-trusted-public-keys = [ "myorg.cachix.org-1:AAAA...=" ];
  };
}
```
```bash
# Environment variables
CACHIX_AUTH_TOKEN=xxx  # For private caches
CACHIX_SIGNING_KEY=xxx # For pushing to cache

# CLI push
cachix push myorg ./result
```
```ini
# nix.conf (system-wide)
substituters = https://cache.nixos.org https://myorg.cachix.org
trusted-public-keys = cache.nixos.org-1:6NCH... myorg.cachix.org-1:AAAA...
```

---

### 3.2 Attic (Self-Hosted Nix Binary Cache)

**What it does:** Self-hosted, multi-tenant Nix binary cache server backed by S3-compatible storage. Written in Rust. Designed for organizations that want full control over their binary cache infrastructure.

**What it caches:** Nix store paths, same as Cachix.

**Key features:**
- **Multi-tenancy**: Isolated caches where tenants are mutually untrusting and cannot pollute other caches' views
- **Global deduplication**: Individual caches are restricted views of a shared content-addressed store
- **Managed signing**: Server handles signing during path fetches -- users pushing paths cannot access signing keys directly
- **Garbage collection**: LRU-based cleanup of unused store paths
- **Scalable**: Supports single-machine to serverless (fly.io) deployments

**Security model:**
- Server-side signing (signing keys never leave the server)
- JWT-based authentication for push/pull operations
- Multi-tenant isolation prevents cross-cache contamination
- S3-compatible storage backend (can leverage S3 encryption at rest)

**Self-hosted vs cloud:** Self-hosted only. Apache 2.0 license. Free.

**Status:** Self-described as "early prototype" -- still under active development. NixOS module available.

**gdev addon configuration:**
```bash
# Login
attic login myserver https://attic.internal token-xxx

# Push
attic push myorg:main ./result

# Configure as substituter
attic use myorg:main  # Adds to nix.conf automatically
```
```ini
# nix.conf (equivalent manual config)
substituters = https://cache.nixos.org https://attic.internal/myorg/main
trusted-public-keys = cache.nixos.org-1:6NCH... myorg:BBBB...=
```

---

### 3.3 nix-serve

**What it does:** Minimal standalone Nix binary cache server. Serves a local Nix store over HTTP. The simplest option for sharing a single machine's store.

**What it caches:** Serves the host machine's `/nix/store` directly over HTTP.

**Security model:**
- Nix store path signing with a secret key
- No authentication (relies on network-level access control)
- No IPv6 or SSL/HTTPS support natively (use nginx reverse proxy)

**Self-hosted vs cloud:** Self-hosted only. Open source. Part of nixpkgs.

**Limitations:**
- No garbage collection or deduplication (serves the host store as-is)
- No multi-tenancy
- No access control beyond network ACLs
- Uses Perl/Starman web server -- minimal and dated
- Suitable only for small teams or LAN-only deployments

**gdev addon configuration:**
```bash
# Generate signing key
nix-store --generate-binary-cache-key cache.myorg.com-1 ./secret ./public

# Start server
NIX_SECRET_KEY_FILE=./secret nix run nixpkgs#nix-serve -- --port 5000
```
```ini
# nix.conf on clients
substituters = https://cache.nixos.org http://build-server:5000
trusted-public-keys = cache.nixos.org-1:6NCH... cache.myorg.com-1:CCCC...=
```

**NixOS module:**
```nix
services.nix-serve = {
  enable = true;
  secretKeyFile = "/var/lib/nix-serve/secret";
};
services.nginx.virtualHosts."cache.myorg.com" = {
  locations."/".proxyPass = "http://localhost:5000";
};
```

---

### 3.4 Bazel Remote Cache

**What it does:** Stores Bazel build action results (compiled outputs, test results) so identical actions on any machine can skip re-execution. Uses the Remote Execution API (REAPI) protocol.

**What it caches:** Two data types:
- **Action Cache (AC)**: Maps action hashes to result metadata
- **Content-Addressable Store (CAS)**: Stores output files by content hash

**Supported backends:**
- **bazel-remote** (open-source, purpose-built): Disk-based with max size enforcement, Docker-deployable, GC built-in
- **Google Cloud Storage**: Managed, HTTP-compatible
- **AWS S3**: Via bazel-remote or custom HTTP server
- **nginx + WebDAV**: Any HTTP/1.1 server supporting PUT/GET
- **BuildBuddy**: Managed service with UI and analytics
- **Hazelcast, Apache httpd**: Community-reported success

**Security model:**
- **Critical concern**: Write access to the Action Cache allows cache poisoning -- an attacker can point any action to compromised results
- **Best practice**: Restrict AC writes to CI-only (`--remote_upload_local_results=false` for developers). Developers get read-only cache access.
- HTTP Basic Auth supported (requires HTTPS)
- `--experimental_guard_against_concurrent_changes` detects input file modifications

**Self-hosted vs cloud:**
- bazel-remote: Open source, self-hosted
- BuildBuddy: Free tier available, paid plans for teams
- GCS/S3: Pay-as-you-go cloud storage costs

**gdev addon configuration:**
```bash
# .bazelrc
build --remote_cache=https://bazel-cache.internal:9090
build --remote_upload_local_results=false  # Read-only for developers; CI pushes
build --remote_cache_header=Authorization=Bearer ${BAZEL_CACHE_TOKEN}

# For GCS backend
build --remote_cache=https://storage.googleapis.com/my-bazel-cache
build --google_default_credentials
```

---

### 3.5 sccache (Shared Compilation Cache)

**What it does:** Compiler caching tool (similar to ccache) that stores compilation results locally or in cloud storage. Built by Mozilla in Rust. Supports distributed compilation with security features.

**What it caches:** C/C++ (gcc, clang, MSVC, diab), Rust (rustc), CUDA (nvcc), ROCm HIP (hipcc), assembler. Caches compilation outputs keyed by input file hashes, compiler flags, and environment.

**Storage backends:**
- Local disk (default)
- Cloud: S3, R2 (Cloudflare), GCS, Azure Blob, Alibaba OSS, Tencent COS
- Cache systems: Redis, Memcached
- CI integration: GitHub Actions cache
- WebDAV (compatible with ccache/Bazel/Gradle)
- **Multi-level hierarchical caching** with automatic backfill

**Security model:**
- Distributed compilation includes authentication, transport encryption, and sandboxed compiler execution (superior to icecream)
- No signing of cached artifacts -- trusts the storage backend
- `SCCACHE_BASEDIRS` for path normalization enables safe cache sharing

**Self-hosted vs cloud:** Open source (Apache 2.0). Self-hosted or use cloud storage backends.

**Limitations:**
- Rust: Cannot cache crates invoking the system linker (bin, dylib, cdylib, proc-macro). Incremental compilation bypasses caching.
- C++20 modules: Partial support (Clang only)

**gdev addon configuration:**
```bash
# Environment variables
export RUSTC_WRAPPER=sccache
export SCCACHE_BUCKET=my-sccache-bucket         # S3 backend
export SCCACHE_REGION=us-east-1
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
export SCCACHE_BASEDIRS=/home/dev/repos          # Enable cross-directory cache sharing

# Or for Redis backend
export SCCACHE_REDIS=redis://cache.internal:6379

# CMake integration
cmake -DCMAKE_C_COMPILER_LAUNCHER=sccache -DCMAKE_CXX_COMPILER_LAUNCHER=sccache

# Cargo config (~/.cargo/config.toml)
[build]
rustc-wrapper = "sccache"
```

---

### 3.6 ccache (Compiler Cache)

**What it does:** The original compiler cache tool. Wraps C/C++ compilers to cache compilation results and skip redundant builds.

**What it caches:** C/C++ compilation outputs. Uses BLAKE3 hashing for input identification and Zstandard compression.

**Storage backends:**
- Local disk (default, `cache_dir`)
- File backend (NFS-shared directory)
- Redis/Redis-compatible (memory or disk-based)
- HTTP backend (GET/PUT/DELETE -- compatible with many HTTP servers)
- `remote_only` mode disables local storage for shared-cache-only setups

**Security model:**
- No signing of cached artifacts
- Trusts the storage backend
- No authentication built-in (relies on storage backend auth)

**Self-hosted vs cloud:** Open source (GPLv3). Self-hosted only.

**Recent improvements (2025-2026):**
- Distributed ThinLTO caching for Clang
- `remote_only` configuration for ephemeral environments
- Improved manifest merging between local and remote storage
- `remote_storage` replaces deprecated `secondary_storage`

**Comparison to sccache:**
- ccache: C/C++ only. Simpler. No Rust support. No cloud-native backends (S3/GCS/Azure). Better for pure C/C++ local development.
- sccache: Multi-language (C/C++/Rust/CUDA). Cloud storage backends. Distributed compilation. Better for multi-language teams and CI.

**gdev addon configuration:**
```bash
# Environment variables
export CC="ccache gcc"
export CXX="ccache g++"
export CCACHE_DIR=/shared/ccache          # Shared cache directory
export CCACHE_MAXSIZE=20G
export CCACHE_COMPRESS=true

# Remote storage (Redis)
export CCACHE_REMOTE_STORAGE="redis://cache.internal:6379"

# Remote storage (HTTP)
export CCACHE_REMOTE_STORAGE="http://cache.internal:8080/ccache"

# CMake integration
cmake -DCMAKE_C_COMPILER_LAUNCHER=ccache -DCMAKE_CXX_COMPILER_LAUNCHER=ccache

# ccache.conf (per-project or global)
max_size = 20G
remote_storage = redis://cache.internal:6379
remote_only = false
compression = true
```

---

### 3.7 Turborepo Remote Cache

**What it does:** Caches task outputs in JavaScript/TypeScript monorepos so identical tasks (build, test, lint) skip re-execution across machines. Part of the Turborepo build system.

**What it caches:** Build outputs, test results, lint results -- any task output defined in `turbo.json`. Treats logs as artifacts too.

**Security features:**
- **HMAC-SHA256 artifact signing**: Verifies integrity and authenticity of cached artifacts. Failed verification treated as cache miss.
- Bearer token authentication

**Self-hosted vs cloud:**
- **Vercel (managed)**: Free across all plans. Zero-config. Recommended by Turborepo.
- **Self-hosted**: Multiple community implementations:
  - `ducktors/turborepo-remote-cache`: Most popular OSS implementation. Supports local disk, S3, GCS, Azure. **Warning:** Any valid AUTH_TOKEN has access to any team -- teams not reliable for access control.
  - `brunojppb/turbo-cache-server`: Alternative implementation
  - Custom: Any HTTP server implementing the Turborepo Remote Cache API

**gdev addon configuration:**
```json
// turbo.json
{
  "remoteCache": {
    "signature": true
  }
}
```
```bash
# Environment variables
TURBO_TOKEN=xxx
TURBO_TEAM=my-team
TURBO_API=https://turbo-cache.internal  # For self-hosted
TURBO_REMOTE_CACHE_SIGNATURE_KEY=my-secret-key

# CLI setup (Vercel)
turbo login
turbo link

# CLI setup (self-hosted)
turbo login --manual  # Prompts for API URL, team, token
```

---

### 3.8 Nx Cloud (Remote Cache)

**What it does:** Caches task outputs in Nx monorepos. Similar to Turborepo Remote Cache but for the Nx build system. Also provides distributed task execution and CI analytics.

**What it caches:** Build outputs, test results, lint results -- any Nx task output.

**Self-hosted options:**
- **@nx/s3-cache**: Amazon S3
- **@nx/gcs-cache**: Google Cloud Storage
- **@nx/azure-cache**: Azure Blob Storage
- **@nx/shared-fs-cache**: Shared filesystem (NFS)
- **Custom server**: OpenAPI spec available for building proprietary cache servers (PUT/GET at `/v1/cache/{hash}`, bearer token auth)

**Security model:**
- **CREEP vulnerability (CVE-2025-36852)**: Critical vulnerability in bucket-based self-hosted caches allowing anyone with PR access to poison production builds. "Many organizations are unaware of this security risk."
- Activation key required (free, automated self-service)
- Bearer token authentication for custom servers

**Self-hosted vs cloud:**
- **Nx Cloud**: Free Hobby tier (50K monthly credits), Team plan (usage-based from $0), Enterprise (custom)
- **Self-hosted plugins**: All free but require activation key. Since Nx v20.8 (April 2025), free self-hosted caching was reintroduced.

**gdev addon configuration:**
```bash
# Install self-hosted cache plugin
nx add @nx/s3-cache

# Environment variables
NX_KEY=activation-key                              # Stored in .nx/key/key.ini
NX_SELF_HOSTED_REMOTE_CACHE_SERVER=https://cache.internal  # Custom server
NX_SELF_HOSTED_REMOTE_CACHE_ACCESS_TOKEN=xxx

# nx.json configuration is auto-generated by nx add
```

**Limitation:** Self-hosted solutions documented as suitable for "proof of concepts and small teams" with security caveats for bucket-based implementations.

---

## 4. Package Security Scanning Services

These tools scan dependencies for vulnerabilities, malicious code, and license issues. Some act as registry-level gates; others integrate into CI.

### 4.1 Socket.dev

**What it does:** Behavioral analysis of packages -- detects malicious code, typosquatting, install scripts, network access, and supply chain attacks *before* any CVE is filed. Fundamentally different from CVE-based scanners.

**Detection method:** Static analysis of package source code + metadata analysis + maintainer behavior analysis. 70+ signals including typosquatting, obfuscated code, dynamic require/eval, dependency confusion, protestware.

**Ecosystems:** 10+ package managers. Full behavioral analysis for npm and PyPI. Vulnerability + supply chain analysis for Go, Maven, RubyGems, Cargo, NuGet, .NET, Scala, Kotlin.

**Registry proxy/gate:**
- **Socket Firewall Free**: Lightweight CLI tool (`sfw`) that wraps package manager commands, intercepts registry traffic, and blocks malicious packages before installation. No API key needed. Supports npm, yarn, pnpm, pip, uv, cargo.
- **Socket Firewall Enterprise**: Full registry proxy with outbound proxy support, custom CAs, Splunk logging. Supports npm, PyPI, Maven, Cargo, RubyGems, NuGet, Go, OpenVSX.

**Free tier:** Free for open-source repos and package search. Socket Firewall Free for developer machines. Paid: Team $25/dev/month, Business $50/dev/month, Enterprise custom.

**CI integration:** GitHub Actions, GitLab Pipeline, Bitbucket Pipeline, Jenkins, Azure DevOps. REST API, JS/Python SDKs, VS Code extension.

**gdev addon configuration:**
```bash
# Install Socket Firewall (free, no API key)
npm i -g @anthropic/socket-firewall  # or via nix
# Wrap package manager commands:
sfw npm install
sfw pip install flask
sfw cargo build

# CI integration (GitHub Actions)
# .github/workflows/socket.yml -- Socket GitHub App handles PR scanning automatically

# Environment variable for paid tiers
SOCKET_SECURITY_API_KEY=xxx
```

---

### 4.2 Snyk

**What it does:** Comprehensive developer security platform. For supply chain: scans package manifests/lockfiles against Snyk's proprietary vulnerability database (24K+ new vulns in 2024). Key differentiator: **reachability analysis** flags only vulnerabilities whose vulnerable functions are actually invoked.

**Ecosystems:** 20+ languages including npm, Maven, Gradle, pip, Go modules, NuGet, RubyGems, Composer, Cargo, Hex. Also: Snyk Container for Docker/OCI images.

**Registry proxy/gate:** No. CI/PR scanning tool only. Does not act as a registry proxy.

**Free tier:** 200 tests/month for private repos. Team $25/dev/month (max 10 licenses). Enterprise $52-$98/dev/month.

**CI integration:** GitHub Actions, GitLab CI, Jenkins, CircleCI, Azure Pipelines, Bitbucket, Travis, AWS CodePipeline. Single step: `snyk test` / `snyk monitor`.

**gdev addon configuration:**
```bash
# Environment variable
SNYK_TOKEN=xxx

# CI step
snyk test --severity-threshold=high
snyk monitor  # Continuous monitoring

# .snyk policy file (per-project)
ignore:
  'SNYK-JS-LODASH-567746':
    - '*':
        reason: 'Not reachable in our code'
        expires: 2026-12-31
```

---

### 4.3 Mend (formerly WhiteSource)

**What it does:** Enterprise SCA platform scanning applications to discover open-source components, check against vulnerability databases and license registries, and auto-remediate. Owns Renovate (dependency update tool).

**Ecosystems:** 200+ programming languages.

**Registry proxy/gate:** No direct registry proxy. Integrates with SCM platforms for PR scanning and automated fix PRs via Renovate.

**Free tier:** None. Commercial only. Pricing not published publicly -- sales-driven.
- Mend AppSec: ~$1,000/contributing developer/year
- Mend AI Premium: ~$300/contributing developer/year
- Mend Renovate Enterprise: ~$250/contributing developer/year

**CI integration:** GitHub, GitLab, Bitbucket, Azure DevOps, Jenkins. Renovate handles automated dependency updates.

**gdev addon configuration:** Primarily SCM-level integration (GitHub App). No per-project file configuration needed beyond Renovate's `renovate.json`.

---

### 4.4 Checkmarx SCA

**What it does:** Enterprise SCA with behavioral analysis for malicious package detection. Has identified 420,000+ malicious packages. Exploitable path analysis traces call paths to determine if vulnerable functions are actually reachable.

**Ecosystems:** Broad language coverage. Scans transitive dependencies to unlimited depth. Supports private JFrog Artifactory registries.

**Registry proxy/gate:** No direct registry proxy. CI/SCM integration.

**Free tier:** None. Enterprise-only. Median annual contract ~$54,000 (range $25K-$111K). Subscription-based by modules, deployment model, and usage metrics.

**CI integration:** Integrates with major CI platforms. SBOM generation in CycloneDX format.

---

### 4.5 Veracode SCA (includes Phylum)

**What it does:** Enterprise SCA combining traditional vulnerability scanning with Phylum's ML-powered behavioral analysis (acquired January 2025). Claims 60% more accurate malicious package detection. Reachability analysis.

**Registry proxy/gate:** **Yes.** Package registry firewall for npm and PyPI blocks malicious packages before installation. Acts as a proxy with real-time threat intelligence.

**Ecosystems:** npm, PyPI (for registry firewall). Broader coverage for SCA scanning.

**Free tier:** None. Enterprise-only. SCA starts at ~$12,000/year. Full Enterprise Suite often exceeds $100,000/year.

**CI integration:** Veracode platform. Integrates with SAST and DAST for correlated findings.

**Note:** Phylum's standalone product and open-source tools (phylum-dev on GitHub) are archived/unmaintained. Socket.dev is the recommended alternative for behavioral analysis without enterprise commitment.

---

### 4.6 Dependabot

**What it does:** GitHub's built-in dependency management. Security updates (auto-PRs for CVEs from GitHub Advisory Database) and version updates (keeps all deps current).

**Ecosystems:** 30+ package managers including npm, pip, Maven, Gradle, Bundler, Cargo, Docker, Terraform, GitHub Actions, Go modules, NuGet, Composer, Hex, pub (Dart), uv (Python).

**Registry proxy/gate:** No. PR creation tool only.

**Free tier:** Completely free. Built into GitHub.

**CI integration:** Native to GitHub. No CI configuration needed for security alerts. `.github/dependabot.yml` for version updates.

**Limitation:** GitHub only. No GitLab, Bitbucket, or Azure DevOps support.

**gdev addon configuration:**
```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: npm
    directory: "/"
    schedule:
      interval: weekly
    groups:
      minor-and-patch:
        update-types: ["minor", "patch"]
  - package-ecosystem: pip
    directory: "/"
    schedule:
      interval: weekly
  - package-ecosystem: github-actions
    directory: "/"
    schedule:
      interval: weekly
```

---

### 4.7 Renovate

**What it does:** Automated dependency update tool with deep configurability. Creates PRs with changelogs, release notes, and merge confidence data. Not a vulnerability scanner, but `minimumReleaseAge` provides quarantine-like protection.

**Ecosystems:** 90+ package managers -- broadest coverage of any tool.

**Registry proxy/gate:** No. PR creation tool.

**Free tier:** Open source (AGPL-3.0). Free Mend-hosted GitHub App. Free self-hosted.

**Key security features:**
- `minimumReleaseAge`: Delay PRs by N days (e.g., 3 days)
- `matchConfidence`: Gate automerge on "High"/"Very High" merge confidence
- Shared presets via `extends` for org-wide policies
- Platform-agnostic: GitHub, GitLab, Bitbucket, Azure DevOps, Gitea

**gdev addon configuration:**
```json
// renovate.json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["config:recommended"],
  "minimumReleaseAge": "3 days",
  "packageRules": [
    {
      "matchUpdateTypes": ["patch"],
      "automerge": true,
      "automergeType": "pr",
      "matchConfidence": ["high", "very high"]
    }
  ]
}
```

---

### 4.8 OSV Scanner (Google)

**What it does:** Open-source vulnerability scanner matching dependencies against the OSV database (aggregates GitHub Advisories, PyPI, RustSec, Go Vulnerability Database, etc.). Fewer false positives than raw CVE matching due to ecosystem-specific version ranges.

**Ecosystems:** C/C++, Dart, Elixir, Go, Java, JavaScript, PHP, Python, R, Ruby, Rust. V2 adds container image scanning.

**Registry proxy/gate:** No. CLI/CI scanner only.

**Free tier:** Fully open source (Apache 2.0). No commercial tier.

**CI integration:** GitHub Actions (reusable workflows), any CI via CLI. JSON output. Offline mode for air-gapped environments.

**gdev addon configuration:**
```yaml
# .github/workflows/osv-scan.yml
- uses: google/osv-scanner-action/osv-scanner-action@v2
  with:
    scan-args: |-
      --recursive
      ./
```

---

### 4.9 Grype + Syft (Anchore)

**What it does:** Complementary pair: **Syft** generates SBOMs from container images/filesystems/archives. **Grype** scans SBOMs against vulnerability databases (NVD, GitHub, distro-specific feeds).

**Ecosystems:** Container images (strongest), plus Go, Python, Java, JavaScript, Ruby, Rust, PHP, .NET, Alpine, Debian, RPM.

**Registry proxy/gate:** No. CLI/CI scanner.

**Free tier:** Both fully open source (Apache 2.0). Anchore Enterprise adds compliance/management for commercial use.

**CI integration:** GitHub Actions, GitLab CI, Azure DevOps, Jenkins, CircleCI, Bitbucket.

**gdev addon configuration:**
```bash
# Generate SBOM
syft packages dir:. -o cyclonedx-json > sbom.cdx.json

# Scan for vulnerabilities
grype sbom:./sbom.cdx.json --fail-on high

# Scan container directly
grype myimage:latest --fail-on critical

# CI pipeline step
syft packages . -o spdx-json > sbom.spdx.json && grype sbom:./sbom.spdx.json --fail-on high
```

---

## 5. SBOM and Provenance Tools

### 5.1 Syft (SBOM Generation)

**What it does:** Generates Software Bills of Materials from container images, filesystems, and archives. Discovers direct and transitive dependencies.

**Output formats:** JSON, SPDX, CycloneDX.

**Ecosystems:** Alpine (apk), Debian (dpkg), RPM, Go, Python, Java, JavaScript, Ruby, Rust, PHP, .NET, and many more.

**Self-hosted vs cloud:** Open source (Apache 2.0). CLI tool. Anchore Enterprise for commercial management.

**gdev addon configuration:**
```bash
# Generate SBOM from project directory
syft packages dir:. -o cyclonedx-json > sbom.cdx.json
syft packages dir:. -o spdx-json > sbom.spdx.json

# Generate SBOM from container image
syft packages myimage:latest -o cyclonedx-json > sbom.cdx.json

# CI integration (generate on every build)
syft packages . -o cyclonedx-json=sbom.cdx.json -o spdx-json=sbom.spdx.json
```

---

### 5.2 sbomnix (Nix-Specific SBOM)

**What it does:** Suite of command-line tools for Nix-specific software supply chain analysis. Generates SBOMs from Nix flake references or store paths, with metadata enrichment from nixpkgs.

**Tools in the suite:**
1. **sbomnix**: SBOM generation from Nix flakes/store paths
2. **nixgraph**: Dependency graph visualization
3. **nixmeta**: Nixpkgs meta-attribute summary
4. **vulnxscan**: Vulnerability scanner using SBOMs
5. **repology_cli/repology_cve**: Repology.org integration
6. **nix_outdated**: Outdated dependency identification
7. **provenance**: SLSA v1.0 compliant provenance attestation generation

**Output formats:** CycloneDX (sbom.cdx.json), SPDX (sbom.spdx.json), CSV.

**Dependency analysis:**
- Runtime dependencies (default): subset of buildtime deps, requires built target
- Buildtime dependencies (`--buildtime`): full closure, no build required

**Metadata enrichment:** For flake references, resolves nixpkgs version and enriches with descriptions, licenses, maintainers, homepage links.

**Self-hosted vs cloud:** Open source (Apache 2.0). CLI tool. Nix flake.

**gdev addon configuration:**
```bash
# Generate SBOM from flake
nix run github:tiiuae/sbomnix#sbomnix -- .#mypackage

# Generate SBOM with buildtime deps
nix run github:tiiuae/sbomnix#sbomnix -- .#mypackage --buildtime

# Vulnerability scan
nix run github:tiiuae/sbomnix#vulnxscan -- .#mypackage

# SLSA provenance
nix run github:tiiuae/sbomnix#provenance -- .#mypackage
```

---

### 5.3 SLSA Framework

**What it is:** OpenSSF-backed framework defining incrementally adoptable security levels for software build and distribution. Provides common vocabulary for supply chain maturity.

**The levels:**

| Level | Key Requirement | Guarantee |
|-------|-----------------|-----------|
| L0 | None | No provenance |
| L1 | Build platform auto-generates provenance | Know *how* artifact was built |
| L2 | Hosted platform signs provenance | Provenance authenticated; tampering detectable |
| L3 | Build isolation; secrets separated | Insider threats mitigated |

**Ecosystem adoption:**

| Ecosystem | SLSA Level | Mechanism |
|-----------|-----------|-----------|
| npm | L3 (GitHub Actions) | Provenance auto with Trusted Publishing |
| PyPI | L3 (GitHub Actions) | PEP 740 attestations auto with Trusted Publishing |
| Maven Central | L3-capable | Sigstore opt-in; PGP mandatory; <1% Sigstore adoption |
| RubyGems | L3-capable | `--attestation` flag in rubygems >= 3.6.0 |
| crates.io | L1 | SHA-256 checksums only; Sigstore RFC pending |
| Go modules | L1 | Checksum transparency only |
| NuGet | L1 | Repository signatures only |

**gdev addon integration:** SLSA is a framework, not a tool. The addon would:
1. Generate CI workflows that produce SLSA L3 provenance (GitHub Actions `actions/attest-build-provenance`)
2. Configure Trusted Publishing for npm/PyPI where applicable
3. Include provenance verification commands in generated CI pipelines

---

### 5.4 Sigstore / cosign

**What it is:** Suite of open-source tools providing "keyless" cryptographic signing. Developers never manage long-lived signing keys -- OIDC identity is the root of trust.

**Components:**
- **Fulcio**: Certificate Authority issuing short-lived X.509 certs (valid ~10 minutes) after OIDC verification
- **Rekor**: Transparency log (append-only Merkle tree) recording every signing event
- **Cosign**: Client tool orchestrating signing flow (generate ephemeral key, get OIDC token, sign, record in Rekor, destroy key)

**Supported OIDC providers:** GitHub (auto-detected in Actions), Google (GCP), Microsoft, GitLab, custom.

**Ecosystem integration:**
- npm: Production (provenance attestations via Sigstore)
- PyPI: Production (PEP 740 attestations)
- Maven Central: Production opt-in (`.sigstore.json` bundles)
- RubyGems: Early production (`--attestation` flag)
- Homebrew: Production (bottle provenance)
- Kubernetes: Production (policy controller)
- crates.io: RFC stage
- Go, NuGet: Not integrated

**Consumer enforcement gap:** No major ecosystem currently allows consumers to *require* provenance at install time. pnpm's `trustPolicy: no-downgrade` is the sole partial exception.

**gdev addon configuration:**
```bash
# Sign a container image
cosign sign --yes myregistry/myimage:latest

# Verify a container image
cosign verify --certificate-identity=workflow@github.com \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  myregistry/myimage:latest

# Sign a blob/artifact
cosign sign-blob --yes --output-signature sig.sig --output-certificate cert.pem artifact.tar.gz

# Verify npm package provenance
npm audit signatures

# CI: Generate provenance attestation (GitHub Actions)
- uses: actions/attest-build-provenance@v2
  with:
    subject-path: ./dist/**
```

---

### 5.5 in-toto (Supply Chain Attestation)

**What it is:** CNCF-graduated framework for protecting supply chain integrity by verifying that each step in the chain was carried out as planned, by authorized personnel, and the product was not tampered with in transit.

**How it works:**
1. Define a **layout** specifying expected steps and trusted actors
2. Each step generates a signed **attestation** (link metadata) recording what was done
3. Verification confirms all steps were completed by authorized actors with correct inputs/outputs

**Key concepts:**
- **Layout**: Policy file defining expected supply chain steps, who can perform them, and what artifacts flow between them
- **Link metadata**: Signed record of a single step's execution (materials in, products out)
- **Attestation framework** (v1.2.0, March 2026): Standardized format for authenticated metadata about software artifacts. Used by SLSA provenance.

**Language implementations:** Go (most mature), Python, Rust, Java.

**Integration with SLSA:** SLSA provenance attestations use the in-toto attestation format. They are complementary -- SLSA defines *what* to attest (build provenance), in-toto defines *how* to attest (format, verification).

**Integration with Sigstore:** in-toto attestations can be signed with Sigstore (keyless signing). This is how npm and PyPI provenance works: in-toto format + Sigstore signature.

**Related tools:**
- **Witness**: Dynamic CLI that integrates into pipelines using in-toto spec, with embedded OPA Rego policy engine
- **GitHub Actions**: `actions/attest-build-provenance` generates in-toto attestations

**Self-hosted vs cloud:** Open source (Apache 2.0). Framework/specification, not a hosted service.

**gdev addon configuration:** in-toto is primarily configured through CI pipeline integration:
```yaml
# GitHub Actions example with Witness
- name: Run build with attestation
  run: |
    witness run --step build --attestor git --attestor github -- \
      npm run build

# Verify attestations
witness verify --policy policy.json --attestations attestations/
```

---

## 6. gdev Addon Integration Patterns

### 6.1 Environment Variables (Universal Pattern)

Every tool can be configured via environment variables, which is the primary mechanism a gdev addon would use:

```bash
# Registry proxy (pick one based on org's choice)
NPM_CONFIG_REGISTRY=https://registry.internal/npm/
PIP_INDEX_URL=https://registry.internal/pypi/simple/
GOPROXY=https://registry.internal/go,direct
CARGO_REGISTRIES_INTERNAL_INDEX=sparse+https://registry.internal/cargo/

# Cache systems
RUSTC_WRAPPER=sccache
SCCACHE_BUCKET=my-cache-bucket
CACHIX_AUTH_TOKEN=xxx
TURBO_TOKEN=xxx
TURBO_TEAM=my-team
NX_KEY=xxx

# Security scanning
SNYK_TOKEN=xxx
SOCKET_SECURITY_API_KEY=xxx
```

### 6.2 Config File Generation (Per-Ecosystem)

The gdev addon would generate these files based on wizard answers:

| File | Tool | Purpose |
|------|------|---------|
| `.npmrc` | npm/pnpm | Registry URL, auth token, `ignore-scripts=true`, `min-release-age` |
| `pip.conf` | pip/uv | Index URL, `--only-binary :all:`, `--require-hashes` |
| `.cargo/config.toml` | Cargo | Registry, `rustc-wrapper = sccache` |
| `settings.xml` | Maven | Mirror URL, server credentials |
| `nuget.config` | NuGet | Package source, `signatureValidationMode` |
| `.bazelrc` | Bazel | Remote cache URL, auth, read-only flag |
| `turbo.json` | Turborepo | Remote cache config, artifact signing |
| `nx.json` | Nx | Remote cache plugin config |
| `nix.conf` | Nix | Substituters, trusted-public-keys |
| `.github/dependabot.yml` | Dependabot | Update schedule per ecosystem |
| `renovate.json` | Renovate | minimumReleaseAge, automerge policies |

### 6.3 Profile-Based Configuration

A gdev profile system would encode org-wide choices:

```yaml
# gdev profile: consulting-default
registry:
  type: nexus-community  # or: jfrog, artifact-keeper, cloudsmith, none
  url: https://nexus.internal
  ecosystems: [npm, pypi, maven, go, cargo]

cache:
  nix: cachix           # or: attic, nix-serve, none
  compilation: sccache   # or: ccache, none
  monorepo: turborepo    # or: nx, none

scanning:
  behavioral: socket-free  # or: socket-paid, none
  vulnerability: osv       # or: snyk, grype, none
  dependency-updates: renovate  # or: dependabot
  ci-protection: harden-runner  # or: none

sbom:
  generator: syft        # or: sbomnix (for Nix projects), none
  signing: cosign        # or: none
```

---

## 7. Recommendations by Organization Profile

### 7.1 Software Consulting Firm (Primary Target)

A consulting firm working across multiple client projects with varying tech stacks needs maximum flexibility with minimal per-project cost.

**Recommended stack:**

| Layer | Tool | Cost | Why |
|-------|------|------|-----|
| Registry proxy | Nexus Community + Socket Firewall Free | $0 | Free multi-ecosystem proxy + free malicious package blocking |
| Nix binary cache | Cachix (free tier) or Attic (self-hosted) | $0 | Shared derivations across team |
| Compilation cache | sccache (S3 backend) | S3 costs only | Multi-language, cloud-native |
| Monorepo cache | Turborepo (Vercel) or Nx Cloud (free) | $0 | Free managed caching |
| Behavioral scanning | Socket.dev (free tier) | $0 | Zero-day malware detection |
| CVE scanning | OSV Scanner | $0 | Free, low false positives |
| Dependency updates | Renovate | $0 | 90+ ecosystems, policy-as-code |
| CI protection | Harden-Runner (community) | $0 | CI runtime monitoring |
| SBOM | Syft + sbomnix | $0 | Container + Nix coverage |

**Total cost: $0-$50/month** (S3 storage for sccache). All tools are free or have sufficient free tiers.

**Upgrade path:** When budget allows, add Socket.dev Team ($25/dev/month) for private repo scanning, and consider JFrog or Sonatype for enterprise-grade registry with security policies.

### 7.2 Startup (Budget-Conscious, GitHub-Native)

| Layer | Tool | Cost |
|-------|------|------|
| Registry proxy | None (use Socket Firewall Free at developer machines) | $0 |
| Dependency updates | Dependabot | $0 |
| CVE scanning | OSV Scanner (GitHub Action) | $0 |
| Behavioral scanning | Socket.dev (free) | $0 |
| CI protection | Harden-Runner (community) | $0 |

### 7.3 Enterprise (Compliance-Driven)

| Layer | Tool | Cost |
|-------|------|------|
| Registry proxy | JFrog Artifactory + Curation + Xray | $50K-100K+/year |
| Nix binary cache | Cachix Pro or Attic (self-hosted) | $varies |
| Scanning | Snyk + Socket.dev Enterprise | ~$100/dev/month |
| SBOM compliance | Syft + Grype + cosign | $0 (OSS) |
| Dependency updates | Renovate Enterprise | $250/dev/year |

### 7.4 Tool Selection Decision Tree

```
Need multi-ecosystem registry proxy?
├── Yes, with security scanning built-in
│   ├── Budget > $25K/year → JFrog Artifactory or Sonatype Nexus
│   ├── Budget > $0, want OSS → Artifact Keeper (new, evaluate carefully)
│   └── Budget = $0 → Nexus Community (proxy only) + CI-level scanners
├── Yes, proxy only (no scanning)
│   ├── AWS-native → CodeArtifact
│   ├── GCP-native → Google Artifact Registry
│   ├── Azure-native → Azure Artifacts
│   └── Self-hosted → Nexus Community
├── npm only → Verdaccio (with age-gating)
└── No proxy needed → Socket Firewall Free at developer machines

Need Nix binary cache?
├── Hosted, minimal ops → Cachix
├── Self-hosted, multi-tenant → Attic
└── Self-hosted, minimal → nix-serve

Need compilation cache?
├── Multi-language (C/C++/Rust/CUDA) → sccache
├── C/C++ only → ccache
└── Not needed

Need monorepo task cache?
├── Turborepo project → Vercel Remote Cache (free) or self-hosted
└── Nx project → Nx Cloud (free) or self-hosted with S3

Need security scanning?
├── Behavioral (zero-day) → Socket.dev
├── CVE database → OSV Scanner (free) or Snyk (paid, reachability)
├── Container → Grype + Syft
└── All of the above (layered)
```

---

## Appendix: Security Capabilities Comparison Matrix

### Registry Proxies

| Tool | Vuln Scan | Malware | License | Age-Gate | Blocklist | Dep Confusion | Policy Engine | Ecosystems |
|------|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| JFrog (full) | Yes | Yes | Yes | Via policy | Yes | Yes | Rich | 40+ |
| Sonatype (full) | Yes* | Yes | Yes* | No | Yes* | Yes | Yes* | 20+ |
| Artifact Keeper | Yes | Partial | No | No | Via policy | No | Basic | 45+ |
| Cloudsmith | Yes | Yes | Yes | Via OPA | Via OPA | Via priority | OPA | 27+ |
| Verdaccio | No** | No | No | **Yes** | **Yes** | No | Basic | 1 (npm) |
| CodeArtifact | No | No | No | No | No | Yes*** | No | 8 |
| Azure Artifacts | No | No | No | No | No | Yes | No | 6 |
| Google AR | Containers | No | No | No | No | Via priority | Containers | 6 |
| GitHub Packages | No | No | No | No | No | N/A**** | No | 6 |
| Bytesafe | Yes | No | Yes | No | Yes | Yes | Yes | 4 |

\* Requires separate Firewall/Lifecycle products
\** Proxies `npm audit` to upstream
\*** Not retroactive for existing packages
\**** Does not proxy upstream registries

### Security Scanners

| Tool | Type | Method | Registry Gate? | Free Tier | Self-Hosted |
|------|------|--------|:-:|:-:|:-:|
| Socket.dev | Behavioral | Static+behavioral analysis | **Yes (Firewall)** | Yes | No |
| Snyk | CVE | DB matching + reachability | No | 200 tests/mo | No |
| Mend | CVE + remediation | DB matching + Renovate | No | No | No |
| Checkmarx SCA | CVE + behavioral | ML + path analysis | No | No | Yes |
| Veracode SCA | CVE + behavioral (Phylum) | ML + sandbox | **Yes (npm/PyPI)** | No | No |
| Dependabot | CVE | GitHub Advisory DB | No | Free | No |
| Renovate | Age-gating | Version tracking | No | Free (OSS) | Yes |
| OSV Scanner | CVE | OSV DB matching | No | Free (OSS) | Yes |
| Grype+Syft | CVE + SBOM | DB matching + SBOM | No | Free (OSS) | Yes |

### Binary Caches

| Tool | What it caches | Remote storage | Security | Free? |
|------|---------------|----------------|----------|:-----:|
| Cachix | Nix store paths | CloudFlare CDN | Nix signing | Free tier |
| Attic | Nix store paths | S3-compatible | Server-side signing, multi-tenant | OSS |
| nix-serve | Nix store paths | Local only | Nix signing | OSS |
| Bazel Remote Cache | Build actions | S3/GCS/HTTP | AC write restriction critical | OSS |
| sccache | Compilation | S3/GCS/Azure/Redis | Distributed auth+encryption | OSS |
| ccache | Compilation | Redis/HTTP/NFS | None built-in | OSS |
| Turborepo RC | Task outputs | Vercel/S3/custom | HMAC-SHA256 signing | Free (Vercel) |
| Nx Cloud | Task outputs | S3/GCS/Azure/custom | CREEP vuln in bucket-based | Free tier |
