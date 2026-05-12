# Private Registries & Validated Package Mirrors/Caches

## Executive Summary

Private registries and validated mirrors are the highest-leverage "configure once, invisible in operation" defense against package supply chain attacks. The core idea: instead of letting every developer and CI/CD pipeline pull directly from public registries (npmjs.org, PyPI, Maven Central, etc.), all package requests route through an organizational intermediary that can proxy, cache, scan, and enforce policies on every download. Once a developer's package manager is pointed at the private registry, every subsequent `npm install`, `pip install`, or `mvn dependency:resolve` is silently filtered.

This report evaluates 12 tools and platforms across three categories: enterprise universal registries (JFrog Artifactory, Sonatype Nexus), ecosystem-specific registries (Verdaccio, Devpi, Bytesafe, Private Packagist), and cloud-native artifact services (AWS CodeArtifact, Azure Artifacts, Google Artifact Registry, GitHub Packages, GitLab Package Registry, Cloudsmith).

**Key finding:** The landscape divides sharply between tools that merely proxy/cache (providing availability and performance benefits) and tools that actively scan, quarantine, and gate packages (providing actual security). Most cloud-native services and lightweight registries fall in the first category; JFrog and Sonatype are the clear leaders in the second category. Verdaccio is the notable exception among lightweight tools, offering age-gating and blocklist/allowlist filtering that provides meaningful security for npm-only organizations.

---

## Tool-by-Tool Analysis

### 1. JFrog Artifactory + Curation + Xray

**Ecosystems supported (40+ package types):** npm, PyPI, Maven, Gradle, NuGet, Go, Cargo, RubyGems, Composer (PHP), Docker/OCI, Helm, Conan (C/C++), Conda, CocoaPods, Debian/APK/RPM, Hugging Face (ML models), Ansible, Chocolatey, Hex, Swift, Terraform, and more.

**How proxying works:** Artifactory uses "remote repositories" that act as transparent proxies to upstream public registries. When a developer requests a package, Artifactory checks its local cache first; on a miss, it fetches from the upstream, caches it, and serves it. Multiple remote repositories can be aggregated behind a single "virtual repository" endpoint, giving developers one URL to configure. The proxying is fully transparent to the client package manager.

**Security scanning and policy enforcement:**

Artifactory's security operates in two complementary layers:

- **JFrog Curation** (preventive): Sits at the remote repository proxy layer and intercepts package requests *before* they are cached. Evaluates metadata against configurable policies: CVE severity thresholds, malicious package databases (1,500+ known malicious packages), license restrictions, unmaintained package detection, and packages lacking community trust signals. When a package is blocked, Curation provides suggested compliant alternatives. Supports observe-only mode for initial calibration. Currently supports npm, PyPI, Maven, and Go modules.

- **JFrog Xray** (analytical): Scans artifacts already stored in Artifactory. Performs binary-level SCA (examines actual compiled artifacts, not just manifests). Draws from NVD, GitHub Advisories, and JFrog's proprietary database (2.8M+ catalogued malicious artifacts). Supports Contextual Analysis — determining whether vulnerabilities are actually exploitable in context. Policy engine can block downloads, fail builds, trigger alerts, or fire webhooks based on CVE severity, CVSS scores, specific CVE IDs, license types, or component age. EPSS scoring support for prioritization.

- **Recent enhancement (April 2026):** Block Downloads from Cached Remote Repositories — enables Curation policies to enforce restrictions even on packages already in cache, closing a gap where previously-cached packages could bypass newly-added policies.

**"Configure once" setup:** Point npm/pip/maven at the virtual repository URL. Example: `npm set registry https://myorg.jfrog.io/artifactory/api/npm/npm-virtual/`. CI/CD pipelines use the same URL. All policy enforcement happens server-side.

**Cost model:**
- SaaS: Pro from $150/month (25 GB), Enterprise X from $950/month (125 GB), Enterprise+ custom
- Self-Managed: Pro X from $27,000/year, Enterprise X from $51,000/year
- Security add-ons (Curation, Advanced Security) may require higher tiers or bundles

**Limitations:**
- Curation currently limited to npm, PyPI, Maven, Go (not all 40+ formats yet)
- Cost is significant — enterprise deployments easily run $50K-$100K+/year
- Requires Artifactory as the base platform (no standalone Xray/Curation)
- Complexity: full platform has a steep learning curve

---

### 2. Sonatype Nexus Repository + Repository Firewall

**Ecosystems supported (20+ formats):** Maven, npm, Docker, PyPI, RubyGems, NuGet, Helm, Cargo, CocoaPods, Conan, Composer, Conda, Go Modules, Gradle, APT, P2, OBR, OCI, R Lang, Yum, Hugging Face.

**How proxying works:** Nexus uses "proxy repositories" that cache packages from upstream registries on first request. Multiple proxy and hosted repositories can be combined into "group repositories" — a single URL serving packages from all members. Smart caching claims to reduce build latency by up to 95%.

**Security scanning and policy enforcement:**

Nexus Repository alone provides basic malware risk alerts (notifications when known malware is detected). The real security capabilities come from the companion products:

- **Repository Firewall** (separate product): Uses proprietary AI and Sonatype's security research to identify and block malicious packages in real-time. Quarantines suspicious components and automatically releases them if confirmed safe. Supports dependency confusion protection. Does NOT require Nexus Repository — works with Artifactory, Cloudsmith, Azure Artifacts, and others. Two tiers:
  - **Firewall Pro:** Malicious package blocking with straightforward onboarding
  - **Firewall Enterprise:** Full policy engine, governance workflows, waivers, broader SDLC coverage

- **Sonatype Lifecycle** (separate product): SCA tool for vulnerability scanning, license compliance, SBOM generation. Integrates with Nexus Repository for CI/CD pipeline gates.

- Notably, Sonatype Firewall can operate at the **network edge** via Zscaler integration, blocking malicious OSS before it reaches any repository manager.

**"Configure once" setup:** Similar to Artifactory — point package managers at the Nexus group repository URL. Firewall sits transparently in front.

**Cost model:**
- Nexus Repository Community Edition: Free (capped at ~200K requests or ~100K components before needing Pro)
- Nexus Repository Pro: Typically $5,000-$20,000/year for 10-50 developers
- Nexus Repository Cloud: Consumption-based, $3,000-$15,000+/month
- Repository Firewall: Separate commercial product (pricing not publicly disclosed, requires sales conversation)
- Full Sonatype Platform (Nexus + Firewall + Lifecycle): Typically sold as a bundle

**Limitations:**
- Security scanning requires additional products beyond base Nexus Repository
- Full platform can rival or exceed JFrog in total cost
- Community Edition has meaningful usage caps
- Sales-driven pricing makes cost comparison difficult

---

### 3. Verdaccio (npm-specific)

**Ecosystems supported:** npm only.

**How proxying works:** Verdaccio acts as a transparent caching proxy to npmjs.org (and other npm-compatible registries). Configure "uplinks" in `config.yaml` pointing to upstream registries. Supports multiple uplinks with fallback logic and cache-first strategy. Caches packages locally; on cache miss, fetches from upstream and stores for future requests.

**Security scanning and policy enforcement:**

Verdaccio stands out among lightweight registries with its built-in `@verdaccio/package-filter` plugin (Verdaccio 6.x+):

- **Age-gating (`minAgeDays`):** Hide package versions published within the last N days. Setting `minAgeDays: 7` means no package younger than 7 days is served. This is a powerful defense against "publish-and-exploit" attacks where malicious packages are published and exploited within hours.

- **Date freezing (`dateThreshold`):** Serve only versions published before a specific date. Useful for emergency response (freeze registry state to a known-good point in time).

- **Blocklists:** Block entire scopes (`@evilscope`), specific packages, or version ranges using semver syntax.

- **Allowlists:** Whitelist specific scopes, packages, or versions that override all blocking rules (including age thresholds). Essential for internal packages (`@my-org/*`) that shouldn't be age-gated.

- **Replace strategy:** Instead of erroring on blocked versions, substitute with the nearest older safe version — preserving transitive dependency resolution.

- Proxies `npm audit` requests to upstream for vulnerability scanning integration.

**"Configure once" setup:** `npm set registry http://my-verdaccio:4873/`. Docker deployment available. Kubernetes-friendly.

**Cost model:** Fully open source (MIT license). Free. Self-hosted only.

**Limitations:**
- npm only — no help for Python, Java, Go, etc.
- No built-in vulnerability scanning (relies on upstream `npm audit`)
- No malware detection intelligence
- Age filtering works on manifest metadata only — already-cached tarballs are not retroactively removed
- Single-ecosystem limits organizational applicability
- No HA built-in (though can be achieved with external storage backends)

---

### 4. Devpi (PyPI-specific)

**Ecosystems supported:** Python (PyPI) only.

**How proxying works:** devpi-server serves as a caching proxy to PyPI. Exploits PyPI package immutability — once cached, a package version never needs re-validation. Supports hierarchical index inheritance: create indexes that inherit from `root/pypi` and overlay with private packages. Each user/team can have multiple indexes.

**Security scanning and policy enforcement:**

Devpi has **no built-in security scanning or policy enforcement**. It is purely a proxy/cache and private package host. No age-gating, no blocklists, no vulnerability scanning. A third-party plugin `devpi-private-mirrors` exists for allowlisting which packages can be mirrored from PyPI, but this is a manual curation effort.

**"Configure once" setup:** `pip install --index-url http://my-devpi:3141/root/pypi/+simple/`. Can also configure via `pip.conf` or `PIP_INDEX_URL` environment variable.

**Cost model:** Fully open source (MIT license). Free. Self-hosted only.

**Limitations:**
- Python only
- No security features whatsoever — purely a proxy/cache
- No age-gating, no blocklists, no scanning
- Useful for performance and availability but not for supply chain security
- Package allowlisting requires a separate plugin and manual maintenance

---

### 5. Cloudsmith

**Ecosystems supported (27+ formats):** Alpine, Cargo, Conda, Composer, CRAN, Dart, Debian, Docker, Generic, Go, Gradle, Helm, Hex, Hugging Face, Maven, npm, NuGet, Python, RPM, Ruby, sbt, Swift, CocoaPods, Conan, LuaRocks, OCI, Terraform, Unity, Vagrant.

**How proxying works:** "Upstream proxying" allows transparent access to packages from upstream repositories. Three indexing strategies: Ahead-of-Time (deterministic), Just-in-Time (learns on first cache), and Real-Time (queries on each request). Upstreams are prioritized (1..n order). Quick Configure Wizard provides pre-configured connections to canonical registries.

**Security scanning and policy enforcement:**

- **Vulnerability scanning:** Continuous package enrichment pulling from OSV.dev, EPSS, and OpenSSF malicious package data
- **Malware detection:** Automatic scanning for known malicious packages
- **License compliance:** Automatic license identification and policy enforcement
- **Policy-as-Code (OPA):** Open Policy Agent-based rules written in Rego. Supports cool-down periods, exploitability prioritization, deep SBOM inspection, malicious package detection
- **Signing/verification:** GPG, PGP, and other signing standard verification
- **SBOM generation:** Available for inspecting dependency trees

**"Configure once" setup:** Point package managers at Cloudsmith repository URLs. Supports standard package manager configuration (npm, pip, maven, etc.).

**Cost model:**
- Pro: $149/month (5 GB storage, 25 GB delivery)
- Team: ~$99/month
- Velocity: ~$299/month
- Ultra/Enterprise: Custom pricing
- **Warning:** Overage charges are significant — $1.50/GB beyond included delivery. Real-world costs can be 3-4x base price with moderate CI/CD usage

**Limitations:**
- SaaS-only (no self-hosted option)
- Overage pricing model can lead to unexpected costs
- Younger platform than Artifactory/Nexus — less battle-tested
- OPA-based policies are powerful but require Rego expertise
- $72M Series C funding (April 2026) suggests rapid development but also startup risk

---

### 6. AWS CodeArtifact

**Ecosystems supported:** npm, PyPI, Maven, NuGet, Swift, Ruby, Cargo, generic packages. Repositories are polyglot.

**How proxying works:** Uses "external connections" to link to one public registry per repository. Upstream CodeArtifact repositories can be chained (up to 10). Packages fetched from upstream are retained in intermediate repositories. Single-endpoint access to both internal and external packages.

**Security scanning and policy enforcement:**

- **Package Origin Controls:** Configures per-package whether publishing and/or upstream access is allowed (ALLOW/BLOCK). Defends against dependency confusion by blocking upstream for internally-published package names. However, **pre-existing packages are NOT protected by default** — requires manual/bulk configuration.
- **No built-in vulnerability scanning**
- **No malware detection**
- **No age-gating or quarantine**
- IAM-based access control with short-lived tokens
- CloudTrail auditing for access logs

**"Configure once" setup:** `aws codeartifact login --tool npm --repository my-repo --domain my-domain`. Configures `~/.npmrc` or pip/maven settings. Tokens are short-lived (12 hours by default), requiring token refresh automation.

**Cost model:** Pay-as-you-go: $0.05/GB-month storage, $0.05 per 10,000 requests. Always-free tier: 2 GB storage + 100,000 requests/month. Same-region AWS data transfer free.

**Limitations:**
- Only one external connection per repository
- No scanning, no malware detection, no age-gating
- Short-lived tokens require refresh automation
- Tightly coupled to AWS ecosystem
- Package Origin Controls are opt-in and not retroactive

---

### 7. Azure Artifacts

**Ecosystems supported:** NuGet, npm, Maven, Python (PyPI), Cargo, Universal Packages.

**How proxying works:** "Upstream sources" allow a feed to proxy public registries. Structured search order: packages published to feed first, then saved upstream packages, then live upstream sources. Packages pulled through upstreams are cached in the feed.

**Security scanning and policy enforcement:**

- **Allow External Versions:** Per-package toggle (default OFF) controlling whether public registry versions can be saved. When a private package exists, new versions with the same name from public registries are blocked — providing dependency confusion protection.
- **No built-in vulnerability scanning**
- **No malware detection**
- **No age-gating or quarantine**
- REST API available for programmatic per-package configuration

**"Configure once" setup:** Configure package managers to use the Azure Artifacts feed URL. Authentication via Azure DevOps PATs or service connections. Deep integration with Azure Pipelines for CI/CD.

**Cost model:** 2 GiB free storage per organization. Additional storage: $2/GiB decreasing to $0.25/GiB at scale. Unlimited users at no extra charge.

**Limitations:**
- No scanning or malware detection
- Per-package external version control is tedious at scale
- Changes can take up to 3 hours to propagate
- Limited to Azure DevOps ecosystem for best experience
- Universal Packages format is Azure-proprietary

---

### 8. Google Artifact Registry

**Ecosystems supported:** Docker/OCI, Maven, npm, Python, Go, Apt (preview), Yum (preview).

**How proxying works:** "Remote repositories" proxy upstream sources. First request downloads and caches; subsequent requests served from cache. "Virtual repositories" aggregate multiple remote + standard repositories behind a single URL with priority ordering. Custom upstream URLs supported (not just canonical registries).

**Security scanning and policy enforcement:**

- **Container vulnerability scanning:** Container Scanning API enables automatic scanning of Docker images in standard and remote repositories
- **Binary Authorization:** Integration with Binary Authorization for container deployment policies
- **VPC Service Controls:** Deny access to upstream sources outside perimeter by default
- **Dependency confusion mitigation:** Virtual repositories can prioritize private over remote repositories
- **No language package vulnerability scanning** (scanning focused on containers)
- **No age-gating or quarantine**
- **No malware detection for language packages**

**"Configure once" setup:** Configure package managers to use Artifact Registry URLs. gcloud CLI for authentication. Native integration with Cloud Build and GKE.

**Cost model:** Pay-as-you-go: storage + network egress. Free tier available. Network egress charges can accumulate significantly in high-traffic scenarios.

**Limitations:**
- Language package scanning limited (container-focused)
- Apt/Yum support in preview
- Upstream sources must be internet-accessible
- Tightly coupled to Google Cloud ecosystem
- No age-gating or policy engine for language packages

---

### 9. GitHub Packages

**Ecosystems supported:** npm, RubyGems, Maven, Gradle, NuGet, Docker/OCI.

**How proxying works:** **GitHub Packages does NOT support upstream proxying.** It is a publish/consume registry only. There is no transparent proxy or caching of public registry packages. You can only install packages that have been explicitly published to GitHub Packages.

**Security scanning and policy enforcement:**

- Dependabot (separate feature) scans repositories for vulnerable dependencies
- No package-level scanning within the registry itself
- No policy enforcement, no age-gating, no blocklists
- Repository-linked or granular permissions depending on registry type

**"Configure once" setup:** Configure `.npmrc` or equivalent to authenticate against `npm.pkg.github.com` (but must also maintain access to npmjs.org for public packages, since GitHub Packages doesn't proxy).

**Cost model:** Free for public packages. Private packages: free quota based on plan, overage billing available.

**Limitations:**
- **No upstream proxying — disqualifies it as a "configure once" proxy defense**
- Cannot replace a public registry; only supplements it
- Certain registries support only repository-scoped permissions
- Limited management APIs

---

### 10. GitLab Package Registry + Virtual Registry

**Ecosystems supported:** npm, Maven, PyPI, NuGet, Composer, Conan, Go, Helm, Terraform, Generic. Virtual Registry currently supports **Maven and container images only**.

**How proxying works:** The Virtual Registry (GA since GitLab 18.10) proxies and caches packages from up to 20 upstream registries behind a single URL. Priority-ordered upstream traversal with configurable cache validity (default 24h, 0-365 days). Dependency proxy for container images has been available longer.

**Security scanning and policy enforcement:**

- Container scanning available through GitLab's security scanning features
- No specific policy enforcement in the Virtual Registry for language packages
- No age-gating, no blocklists, no malware detection at the registry level
- GitLab's broader security features (SAST, DAST, dependency scanning) operate at the CI/CD pipeline level, not at the registry level

**"Configure once" setup:** Point Maven/Docker clients at the virtual registry URL. Authentication via GitLab tokens.

**Cost model:** Virtual Registry requires Premium or Ultimate tier. Package Registry available on all tiers. Premium starts at $29/user/month, Ultimate at $99/user/month.

**Limitations:**
- Virtual Registry only supports Maven and containers — not npm, PyPI, etc.
- Premium/Ultimate tier required for virtual registry
- No security scanning at the registry level
- Users must be direct members of the top-level group
- Relatively new feature (GA in 18.10)

---

### 11. Bytesafe (npm-focused)

**Ecosystems supported:** npm (primary), Maven, NuGet, PyPI.

**How proxying works:** Acts as a secure proxy to public package registries. Firewall registry centralizes policies and automatically enforces them before packages reach downstream registries.

**Security scanning and policy enforcement:**

- **Vulnerability scanning:** Enabled by default for new firewall registries. Quarantines packages surpassing thresholds.
- **License compliance:** Enabled by default.
- **Block Install Scripts:** Quarantines npm packages with pre/post-install scripts.
- **Dependency confusion protection:** Configurable.

**"Configure once" setup:** Point npm at the Bytesafe registry URL.

**Cost model:** Commercial SaaS. Community Edition (open source) available on GitHub with basic features.

**Limitations:**
- Primarily npm-focused
- Smaller community than Artifactory/Nexus
- Less mature platform
- Limited ecosystem breadth

---

### 12. Private Packagist / Satis (PHP-specific)

**Ecosystems supported:** PHP (Composer) only.

**How proxying works:** Private Packagist mirrors public packages and hosts private packages. Satis is a static repository generator — it does NOT proxy; it generates a static index that Composer reads. Packeton is an open-source alternative.

**Security scanning and policy enforcement:**

- Packagist.org is developing a transparency log for security-relevant events
- No scanning or policy enforcement in Private Packagist beyond basic access control
- Dependency confusion defense via Composer's `canonical` repository option
- No age-gating, no blocklists, no vulnerability scanning at registry level

**"Configure once" setup:** Configure `composer.json` repositories to point at Private Packagist, using `canonical: true` to prevent dependency confusion.

**Cost model:** Private Packagist: Commercial SaaS. Satis: Open source (free). Packeton: Open source (free).

**Limitations:**
- PHP only
- Satis is static (no proxy capability)
- No security scanning in the registry layer
- Dependency confusion requires manual `canonical` configuration

---

## Comparative Analysis

### Security Capabilities Matrix

| Tool | Vuln Scanning | Malware Detection | License Compliance | Age-Gating | Blocklists | Dependency Confusion | Policy Engine |
|------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **JFrog (Curation+Xray)** | Yes | Yes | Yes | Via policy | Yes | Yes | Yes (rich) |
| **Sonatype (Nexus+Firewall)** | Yes* | Yes | Yes* | No | Yes* | Yes | Yes* |
| **Verdaccio** | No** | No | No | **Yes** | **Yes** | No | Basic |
| **Devpi** | No | No | No | No | Plugin*** | No | No |
| **Cloudsmith** | Yes | Yes | Yes | Via OPA | Via OPA | Via priority | Yes (OPA) |
| **AWS CodeArtifact** | No | No | No | No | No | Yes**** | No |
| **Azure Artifacts** | No | No | No | No | No | Yes | No |
| **Google Artifact Registry** | Containers | No | No | No | No | Via priority | Container-only |
| **GitHub Packages** | No | No | No | No | No | N/A***** | No |
| **GitLab Virtual Registry** | No | No | No | No | No | Via priority | No |
| **Bytesafe** | Yes | No | Yes | No | Yes | Yes | Yes |
| **Private Packagist** | No | No | No | No | No | Via canonical | No |

\* Requires separate Firewall/Lifecycle products
\** Proxies `npm audit` to upstream
\*** Via `devpi-private-mirrors` plugin (allowlisting only)
\**** Not retroactive for existing packages
\***** Does not proxy upstream registries

### Ecosystem Breadth

| Tool | Package Types | Best For |
|------|:---:|---|
| **JFrog Artifactory** | 40+ | Multi-ecosystem enterprise |
| **Sonatype Nexus** | 20+ | Multi-ecosystem enterprise |
| **Cloudsmith** | 27+ | Multi-ecosystem cloud-native |
| **GitLab Package Registry** | 10+ | GitLab-native teams |
| **AWS CodeArtifact** | 8 | AWS-native teams |
| **Azure Artifacts** | 6 | Azure DevOps teams |
| **Google Artifact Registry** | 6 | GCP-native teams |
| **GitHub Packages** | 6 | GitHub-native teams |
| **Bytesafe** | 4 | npm-focused teams |
| **Verdaccio** | 1 (npm) | Small npm-only teams |
| **Devpi** | 1 (Python) | Python-only teams |
| **Private Packagist** | 1 (PHP) | PHP teams |

### Cost Comparison

| Tool | Free Tier | Entry Cost | Enterprise Cost | Model |
|------|---|---|---|---|
| **Verdaccio** | Fully free | $0 | $0 (self-hosted) | Open source |
| **Devpi** | Fully free | $0 | $0 (self-hosted) | Open source |
| **Satis/Packeton** | Fully free | $0 | $0 (self-hosted) | Open source |
| **Nexus Community** | Yes (capped) | $0 | $5K-20K/yr (Pro) | OSS + commercial |
| **AWS CodeArtifact** | 2GB + 100K req | ~$5-50/month | Pay-as-you-go | Cloud consumption |
| **Azure Artifacts** | 2 GiB | ~$2/GiB extra | Pay-as-you-go | Cloud consumption |
| **Google Artifact Registry** | Small free tier | Pay-as-you-go | Pay-as-you-go | Cloud consumption |
| **GitHub Packages** | Plan-dependent | Plan-dependent | Plan-dependent | Included with GitHub |
| **Cloudsmith** | No | $149/month | Custom | SaaS + overages |
| **Bytesafe** | Community Ed. | Commercial | Commercial | SaaS |
| **JFrog Artifactory** | No* | $150/month | $27K-51K+/yr | Commercial |
| **Sonatype Full Platform** | No | ~$5K/yr | $50K+/yr | Commercial |

\* JFrog offers a limited free-tier cloud instance but with very limited features

---

## Recommendations by Organization Profile

### Multi-Ecosystem Enterprise (Best Overall)

**JFrog Artifactory + Curation + Xray** or **Sonatype Nexus + Repository Firewall** are the only options providing comprehensive security scanning across multiple ecosystems. JFrog has broader format support (40+ vs 20+) and its Curation feature provides true pre-download prevention. Sonatype's Firewall is notable for being registry-agnostic (works with Artifactory too) and can operate at the network edge.

**Recommendation:** JFrog if format breadth and integrated platform are priorities. Sonatype if you want flexibility to pair Firewall with any repository manager, or if you're already a Nexus shop.

### Cloud-Native Teams (AWS/Azure/GCP)

Cloud-native artifact services (CodeArtifact, Azure Artifacts, Google Artifact Registry) provide basic proxying and caching with minimal setup and pay-as-you-go pricing. However, they offer **minimal security capabilities** — dependency confusion protection at most, no scanning, no malware detection, no age-gating. They are suitable as a caching/availability layer but should be paired with a security tool (Sonatype Firewall, Snyk, Socket.dev) for actual supply chain defense.

**Recommendation:** Use your cloud's artifact service for caching/availability, but layer a security tool on top. Don't rely on CodeArtifact/Azure Artifacts alone for supply chain security.

### Small Teams / Budget-Constrained

**Verdaccio** (npm) provides the best security-to-cost ratio for JavaScript-focused teams. The `minAgeDays` age-gating alone blocks a large class of "publish-and-exploit" attacks, and scope/package blocklists enable emergency response. Pair with `npm audit` for vulnerability scanning.

For Python teams, **Devpi** provides caching but no security. Consider pairing with a CI-level scanner (pip-audit, Safety).

**Recommendation:** Verdaccio for npm, Devpi + pip-audit for Python. Accept the single-ecosystem limitation.

### Multi-Ecosystem but Budget-Constrained

**Cloudsmith** offers the widest format support (27+) with meaningful security features (vuln scanning, malware detection, OPA policies) at a lower price point than JFrog/Sonatype. However, overage pricing can be surprising, and it's SaaS-only.

**Nexus Repository Community Edition** is free and supports 20+ formats, but security features require paid add-ons.

**Recommendation:** Nexus Community for the registry layer (free, self-hosted), paired with a CI-level security scanner (Snyk, Socket.dev, OSV Scanner).

---

## Key Insights

1. **The registry is the ideal enforcement point.** Scanning in CI/CD catches problems after code is already on developer machines. A registry-level gate catches problems before `npm install` completes. This is fundamentally better for "configure once" defense.

2. **Age-gating is underappreciated.** Most supply chain attacks exploit packages within hours of publication. A simple 7-day hold-back (Verdaccio's `minAgeDays`, JFrog Curation's operational risk policies) blocks a large class of attacks with minimal developer friction. Only Verdaccio and JFrog offer this as a first-class feature.

3. **Dependency confusion protection is table-stakes but insufficient.** Most tools now offer it (CodeArtifact's origin controls, Azure's external versions toggle, priority-based resolution). But it addresses only one attack vector — not malware injection, typosquatting, or account takeover.

4. **Cloud-native services are caching layers, not security layers.** AWS CodeArtifact, Azure Artifacts, and Google Artifact Registry provide availability and performance benefits but should NOT be relied upon for supply chain security. Their security features are limited to dependency confusion prevention.

5. **GitHub Packages is not a proxy.** It cannot replace public registries and does not fit the "configure once" model. It's a publishing platform, not a supply chain defense.

6. **The OSS landscape is ecosystem-specific.** There is no free, open-source, multi-ecosystem registry with security features. Verdaccio (npm) is the closest to useful, but Python, Java, Go, etc. have no equivalent. This is a gap that commercial tools fill.

7. **Sonatype Firewall's registry-agnostic design is strategically interesting.** It can layer on top of any repository manager (including Artifactory), enabling organizations to add security to existing infrastructure without replacing it.

---

## Open Questions

- How does JFrog Curation's malware detection compare to Sonatype Firewall's in practice? Independent benchmarks are scarce.
- What is the false positive rate for age-gating? How often do legitimate urgent security patches get blocked by `minAgeDays` policies, and what's the operational cost of the allowlist maintenance?
- Can Cloudsmith's OPA-based policies express age-gating rules comparable to Verdaccio's `minAgeDays`?
- What's the total cost of ownership comparison between self-hosted Nexus + Firewall vs. JFrog Cloud at enterprise scale?

---

## Sources

All source documents saved to `docs/`:
- `docs/jfrog-curation-overview.md` — JFrog Curation features and integration
- `docs/jfrog-xray-review.md` — JFrog Xray capabilities and pricing
- `docs/sonatype-repository-firewall.md` — Sonatype Firewall malware detection
- `docs/sonatype-nexus-repository.md` — Nexus Repository features and pricing
- `docs/verdaccio-features-architecture.md` — Verdaccio architecture and features
- `docs/verdaccio-package-filter-plugin.md` — Verdaccio age-gating and filtering
- `docs/devpi-server-overview.md` — Devpi proxy/cache capabilities
- `docs/cloudsmith-upstream-proxying.md` — Cloudsmith proxy and format support
- `docs/gitlab-virtual-registry.md` — GitLab Virtual Registry features
- `docs/github-packages-overview.md` — GitHub Packages limitations
- `docs/google-artifact-registry-remote-repos.md` — Google AR remote repositories
- `docs/azure-artifacts-upstream-behavior.md` — Azure Artifacts supply chain protection
- `docs/aws-codeartifact-upstream-repos.md` — AWS CodeArtifact upstream and origin controls
- `docs/php-packagist-supply-chain-security.md` — PHP ecosystem approach
- `docs/bytesafe-npm-firewall.md` — Bytesafe firewall features
