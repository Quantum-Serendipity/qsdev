# Organizational Tooling & Policy Enforcement for Package Supply Chain Security

## Executive Summary

This report surveys 10 tools and platforms that can be configured once (in CI, as GitHub Apps, or as organizational policies) and then continuously monitor and enforce supply chain security across package ecosystems. These tools fall into five functional categories: **behavioral threat detection** (Socket.dev, Phylum/Veracode), **vulnerability scanning** (Snyk, OSV Scanner, Grype), **automated dependency updates** (Dependabot, Renovate), **project health assessment** (OpenSSF Scorecard, deps.dev), and **CI/CD runtime protection** (StepSecurity Harden-Runner). No single tool covers the full threat surface; the recommended approach is a layered stack combining complementary tools from each category.

---

## Tool-by-Tool Analysis

### 1. Socket.dev

**What it detects/prevents:** Socket takes a fundamentally different approach from traditional SCA tools. Instead of matching dependencies against CVE databases, it performs **behavioral analysis** using three techniques: static analysis of package source code, package metadata analysis, and maintainer behavior analysis. It identifies 70+ signals including: typosquatting (via Levenshtein distance + download count heuristics), malicious install scripts, network access, filesystem access, environment variable access, shell execution, obfuscated code, dynamic require/eval, dependency confusion, protestware, and compromised maintainer accounts. This catches threats *before* any CVE is filed.

**Set-and-forget configuration:** Install the GitHub App. It automatically scans every PR that modifies dependency manifests and posts inline comments flagging risky packages. The `socket ci` CLI command returns non-zero on unhealthy alerts for CI gate integration. Organizational policies can block categories of risk (e.g., block all packages with install scripts or network access).

**Ecosystems:** 10+ package managers. Full behavioral analysis for npm and PyPI. Vulnerability + supply chain analysis for Go, Maven, RubyGems, Cargo, NuGet, .NET, Scala, Kotlin. PHP, Swift, and Objective-C in development.

**Open source vs commercial:** Free for open-source repos and package search/health scores. Paid tiers: Team ($25/dev/month), Business ($50/dev/month), Enterprise (custom). GitHub is the primary integration platform; GitLab/Bitbucket/Azure require Enterprise tier.

**CI integration:** GitHub Actions via Python client (`pip install socketsecurity`), GitLab Pipeline, Bitbucket Pipeline, Jenkins, Azure DevOps. Also offers REST API, JS/Python SDKs, VS Code extension, Chrome extension.

**Limitations:**
- Not a vulnerability scanner -- does not replace CVE-based tools like Snyk or OSV Scanner
- Deepest behavioral analysis limited to npm and PyPI; other ecosystems get lighter coverage
- Multi-platform CI support (GitLab, Bitbucket, Azure) requires Enterprise tier
- SBOM export (CycloneDX) requires Business tier or above
- No self-hosted option

---

### 2. Snyk

**What it detects/prevents:** Snyk is a comprehensive developer security platform with six products. For supply chain specifically, **Snyk Open Source** (SCA) scans package manifests and lock files against Snyk's proprietary vulnerability database (24k+ new vulnerabilities added in 2024). Key differentiator: **reachability analysis** flags only vulnerabilities whose vulnerable functions are actually invoked in your code, dramatically reducing false positives. Also provides **Snyk Container** for Docker/OCI image scanning with base image upgrade recommendations.

**Set-and-forget configuration:** Enable the GitHub/GitLab/Bitbucket integration. Snyk automatically scans repos on push and PR, creates automated fix PRs for known vulnerabilities, and monitors continuously for newly disclosed CVEs affecting existing dependencies. Organization-level policies set severity thresholds for build-breaking.

**Ecosystems:** Broad language coverage: Apex, C/C++, Dart/Flutter, Elixir, Go, Groovy, Java, Kotlin, JavaScript, TypeScript, .NET, PHP, Python, Ruby, Rust, Scala, Swift/Objective-C. Package managers: npm, Maven, Gradle, pip, Go modules, NuGet, RubyGems, Composer, Cocoapods, Cargo, Hex.

**Open source vs commercial:** Free tier (200 tests/month for private repos). Team plan $25/dev/month (unlimited tests, max 10 licenses). Enterprise custom pricing ($52-$98/dev/month typical). No self-hosted option. Credit-based licensing introduced recently.

**CI integration:** GitHub Actions, GitLab CI, Jenkins, CircleCI, Azure Pipelines, Bitbucket Pipelines, Travis CI, AWS CodePipeline. Single pipeline step: `snyk test` (scan) / `snyk monitor` (continuous monitoring).

**Limitations:**
- CVE-focused: does not detect behavioral threats like Socket does (no install script analysis, no typosquatting detection)
- Free tier is restrictive (200 tests/month); Team tier capped at 10 licenses
- Enterprise pricing can be expensive at scale
- No self-hosted deployment option
- Reachability analysis not available for all languages

**How it differs from Socket:** Snyk and Socket are complementary, not competing. Snyk detects *known* vulnerabilities (CVEs) and generates fix PRs. Socket detects *behavioral anomalies* and *zero-day malicious packages* before any CVE exists. A team using both gets CVE coverage (Snyk) plus proactive threat detection (Socket).

---

### 3. Dependabot

**What it detects/prevents:** GitHub's built-in dependency management with two functions: (1) **Security updates** -- automatically creates PRs to update dependencies with known vulnerabilities from the GitHub Advisory Database, and (2) **Version updates** -- keeps all dependencies current regardless of vulnerability status. Also scans GitHub Actions workflow files for vulnerable action versions.

**Set-and-forget configuration:** Security alerts are enabled by default on public GitHub repos. Version updates require a `.github/dependabot.yml` configuration file specifying package ecosystems, directory, and schedule. Auto-triage rules allow bulk dismiss/snooze of low-severity alerts. Update grouping reduces PR noise by combining related updates into single PRs.

**Ecosystems:** 30+ package managers including npm, pip, Maven, Gradle, Bundler, Cargo, Docker, Terraform, GitHub Actions, Go modules, NuGet, Composer, Hex, pub (Dart), and uv (Python, added December 2025).

**Open source vs commercial:** Completely free. Built into GitHub (no separate product). Available on GitHub.com and GitHub Enterprise Server.

**CI integration:** Native to GitHub -- no CI configuration needed for alerts. Version update PRs trigger normal CI workflows. Can be combined with GitHub Actions for automerge (via `dependabot/fetch-metadata` action).

**Limitations:**
- **GitHub only** -- no support for GitLab, Bitbucket, or Azure DevOps
- No shared configuration across repositories (each repo needs its own `dependabot.yml`)
- No built-in automerge (requires separate GitHub Actions workflow)
- No dependency dashboard for overview
- Monorepo updates cannot be combined into a single PR
- Auto-pauses when PRs are ignored by maintainers
- No behavioral analysis -- purely CVE-based via GitHub Advisory Database
- No merge confidence data beyond basic compatibility scores

---

### 4. Renovate

**What it detects/prevents:** Automated dependency update tool, similar to Dependabot but significantly more configurable. Creates PRs for outdated dependencies with detailed changelogs, release notes, and merge confidence data. Not a vulnerability scanner per se, but its `minimumReleaseAge` feature provides quarantine-like protection against supply chain attacks by delaying adoption of new releases.

**Set-and-forget configuration:** Install the Mend Renovate GitHub App (or self-host). Renovate sends an onboarding PR showing all detected dependencies. Key security-relevant configuration:
- `minimumReleaseAge`: Delay PRs for new releases by N days (e.g., 3 days for npm where packages can be unpublished within 72 hours)
- `packageRules` with `matchUpdateTypes`: Automerge patch/minor, require review for major
- `matchConfidence`: Only automerge updates with "High" or "Very High" merge confidence
- `automergeSchedule`: Restrict automerge to business hours
- Shared presets via `extends` for organization-wide policies

**Merge Confidence** is a key differentiator: four badges (Age, Adoption, Passing, Confidence) displayed on each PR, derived from aggregated test results across all Renovate users. npm packages require minimum 3-day age before "High" confidence. Paying customers can gate automerge on confidence level.

**Ecosystems:** 90+ package managers -- the broadest coverage of any tool in this category. Includes everything Dependabot covers plus Poetry, Pipenv, Kubernetes manifests, Helm charts, CircleCI configs, and regex managers for arbitrary version strings in any file.

**Open source vs commercial:** Open source (AGPL-3.0). Free Mend-hosted GitHub App. Free self-hosted deployment. Commercial Mend Renovate Enterprise tier adds organization management, enhanced merge confidence workflows, and priority support.

**CI integration:** Platform-agnostic: GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, Forgejo, SCM-Manager. Self-hosted via npm package, Docker images, GitHub Action, or GitLab Runner.

**Limitations:**
- Not a vulnerability scanner -- does not scan for CVEs (pair with OSV Scanner/Snyk/Dependabot alerts)
- Merge Confidence API requires Mend access token for self-hosted instances
- Configuration complexity -- the 90+ manager support and regex capabilities create a steep learning curve
- Self-hosted instances require maintenance (cron job, credentials management)
- Merge Confidence data only available for popular packages; niche packages show "Neutral"

**Renovate vs Dependabot:** Renovate wins on configurability (shared presets, regex managers, built-in automerge, multi-platform, 90+ managers, merge confidence, dependency dashboard). Dependabot wins on simplicity (zero-config security alerts, built into GitHub, no maintenance). For organizations on GitHub-only with simple needs, Dependabot suffices. For multi-platform organizations wanting policy-as-code dependency management, Renovate is the clear choice.

---

### 5. OSV Scanner (Google)

**What it detects/prevents:** Open-source vulnerability scanner that matches project dependencies against the OSV (Open Source Vulnerabilities) database. The OSV database aggregates advisories from multiple authoritative sources (GitHub Advisory Database, PyPI, RustSec, Go Vulnerability Database, etc.) using a machine-readable format that precisely maps to developer package versions -- resulting in fewer false positives than raw CVE matching.

**Set-and-forget configuration:** Add the OSV-Scanner GitHub Action to your CI workflow. Reusable workflows scan: (1) new dependencies in PRs for introduced vulnerabilities, and (2) entire project on schedule for newly disclosed vulnerabilities. V2 adds **guided remediation** -- analyzes the dependency graph and recommends the minimum set of upgrades ranked by severity, dependency depth, and ROI.

**Ecosystems:** C/C++, Dart, Elixir, Go, Java, JavaScript, PHP, Python, R, Ruby, Rust. Package managers: npm, pip, yarn, Maven, Go modules, Cargo, gem, Composer, NuGet, and others. V2 adds layer-aware container image scanning for Debian, Ubuntu, and Alpine.

**Open source vs commercial:** Fully open source (Apache 2.0). No commercial tier. Backed by Google.

**CI integration:** GitHub Actions (reusable workflows provided), or run as CLI in any CI system. Supports JSON output for pipeline integration. Offline mode available for air-gapped environments.

**Limitations:**
- Vulnerability scanning only -- no behavioral analysis, no malicious package detection
- OSV database coverage depends on upstream advisory sources; some ecosystems have sparser coverage
- Guided remediation is relatively new (V2, March 2025)
- No PR creation -- only identifies vulnerabilities (pair with Renovate/Dependabot for automated fixes)
- No reachability analysis (unlike Snyk)

**How it compares to CVE databases:** OSV uses ecosystem-specific version ranges rather than CPE matching, which is why it produces fewer false positives. A CVE like "affects package X versions < 2.0" in NVD might match incorrectly due to CPE ambiguity. OSV's format maps directly to package manager version constraints.

---

### 6. OpenSSF Scorecard

**What it detects/prevents:** Automated assessment of open source project security posture. Evaluates repositories against 19 security checks and assigns scores of 0-10 per check with a risk-weighted aggregate. Not a vulnerability scanner -- it measures *practices* (does the project use branch protection? code review? SAST? signed releases?) to help you evaluate the trustworthiness of dependencies before adopting them.

**Set-and-forget configuration:** Two usage modes:
1. **For your own repos:** Install the Scorecard GitHub Action. It runs on code changes, publishes results to the Security tab, and displays a badge. This surfaces which security practices your project is missing.
2. **For evaluating dependencies:** Use the REST API (api.scorecard.dev) or CLI to check scores before adding a new dependency. Can be integrated into PR automation to flag low-scoring new dependencies. The deps.dev API also surfaces Scorecard data.

**Complete check list:** Binary-Artifacts (High), Branch-Protection (High), CI-Tests (Low), CII-Best-Practices (Low), Code-Review (High), Contributors (Low), Dangerous-Workflow (Critical), Dependency-Update-Tool (High), Fuzzing (Medium), License (Low), Maintained (High), Pinned-Dependencies (Medium), Packaging (Medium), SAST (Medium), Security-Policy (Medium), Signed-Releases (High), Token-Permissions (High), Vulnerabilities (High), Webhooks (Critical).

**Ecosystems:** Platform-level, not ecosystem-specific. Evaluates any GitHub, GitLab, or Gitea repository regardless of language.

**Open source vs commercial:** Fully open source (Apache 2.0). Google scans 1 million+ repos weekly and publishes results in a public BigQuery dataset. REST API free to use.

**CI integration:** GitHub Action (primary), CLI (Docker, Homebrew, Nix, standalone binaries). Results can be consumed as JSON for automated policy decisions.

**Limitations:**
- Heuristic-based: false positives and negatives are common
- Not all checks apply to all project types (aggregate scores can be misleading)
- Weekly REST API scans omit 3 checks (CI-Tests, Contributors, Dependency-Update-Tool) due to API costs
- GitHub-focused: project scan list is GitHub-only (GitLab/Gitea support exists for CLI but not bulk scanning)
- Evaluates project *practices*, not code *content* -- a well-maintained project can still have vulnerabilities

---

### 7. deps.dev (Google Open Source Insights)

**What it detects/prevents:** Dependency intelligence service providing security metadata, dependency graphs, license information, and project health signals for 50+ million package versions. Not a scanning tool -- it's a **data platform** that other tools and workflows can query to make informed decisions about dependencies.

**Set-and-forget configuration:** Not a CI tool per se, but can be integrated into dependency evaluation workflows:
- Query the API when adding new dependencies to check vulnerability status, license compliance, and Scorecard data
- Use the BigQuery dataset for bulk analysis of organizational dependency health
- Hash queries enable SBOM enrichment and incident response (look up file hash to identify package version)

**API endpoints:** GetPackage, GetVersion, GetRequirements, GetDependencies, GetProject (includes Scorecard data and OSS-Fuzz coverage), GetProjectPackageVersions, GetAdvisory, Query (including hash-based lookup).

**Ecosystems:** Cargo, Go, Maven, npm, NuGet, PyPI, RubyGems. Project hosts: GitHub, GitLab, Bitbucket.

**Open source vs commercial:** Free API (JSON over HTTP and gRPC). Free BigQuery dataset. Data licensed under CC-BY 4.0. Backed by Google.

**Key data provided:**
- Full resolved dependency graphs (npm, Cargo, Maven, PyPI)
- Security advisories from OSV.dev with CVSS scores
- OpenSSF Scorecard results per project
- OSS-Fuzz coverage metrics
- SLSA provenance and Sigstore attestation verification
- License identification (SPDX 2.1)
- Package publication timestamps and deprecation status

**Limitations:**
- Data service only -- does not scan your project or create PRs
- Resolved dependency graphs only available for npm, Cargo, Maven, PyPI
- Go modules limited to those fetched via proxy.golang.org
- No CI integration out of the box (requires custom scripting)
- No rate limits documented, but subject to Google API Terms of Service

---

### 8. Phylum (now Veracode SCA)

**What it detects/prevents:** Phylum pioneered ML-powered detection of malicious packages using both static and dynamic analysis. Packages were executed in a sandbox ("Bird Cage") to observe actual runtime behavior: network connections, file operations, process spawning. This was combined with static analysis to detect typosquatting, dependency confusion, compromised maintainer accounts, and malicious code injection. Risk scoring across five domains: software vulnerabilities, license issues, author risk, engineering risk, and malicious code.

**Current status (2026):** Phylum's technology was acquired by Veracode in January 2025. The standalone Phylum product is **no longer available**. The technology has been integrated into Veracode SCA, which now includes a "package registry firewall" blocking malicious packages for npm and PyPI before installation. Veracode reports 60% more accurate malicious package detection post-integration. Open source components on GitHub (phylum-dev) are archived/unmaintained.

**Pre-acquisition ecosystems:** npm, PyPI, RubyGems, Maven, NuGet, Go, Cargo, Packagist.

**Pre-acquisition integrations:** CLI, GitHub App, GitHub Actions, GitLab CI, Jenkins, package firewall/allowlist.

**Open source vs commercial:** No longer available as standalone. Requires Veracode enterprise licensing.

**Limitations:**
- No longer independently available
- Veracode integration narrows focus to npm and PyPI
- Enterprise pricing only
- For teams wanting similar behavioral analysis without enterprise commitment, Socket.dev is the closest alternative

---

### 9. Grype + Syft (Anchore)

**What they detect/prevent:** A complementary pair: **Syft** generates Software Bills of Materials (SBOMs) from container images, filesystems, and archives. **Grype** scans SBOMs (or images/directories directly) against aggregated vulnerability databases (NVD, GitHub Advisories, distribution-specific feeds from Red Hat, Debian, Ubuntu, etc.). Together they provide SBOM generation + vulnerability scanning in a single pipeline.

**Set-and-forget configuration:** Add Syft and Grype steps to CI pipeline. Syft generates an SBOM on every build; Grype scans it and fails the build if vulnerabilities exceed a severity threshold. SBOMs can be stored as build artifacts for compliance and incident response.

**Syft capabilities:**
- Generates SBOMs from container images, filesystems, archives
- Discovers direct and transitive dependencies
- Output formats: JSON, SPDX, CycloneDX
- Ecosystems: Alpine (apk), Debian (dpkg), RPM, Go, Python, Java, JavaScript, Ruby, Rust, PHP, .NET, and many more
- V1.2+ adds enhanced binary-only environment scanning

**Grype capabilities:**
- Scans container images, filesystems, and SBOMs
- Cross-references against NVD, GitHub, Red Hat, Debian, Ubuntu, and other feeds
- Severity-based filtering and fail thresholds
- JSON output for pipeline integration

**Open source vs commercial:** Both fully open source (Apache 2.0). **Anchore Enterprise** adds continuous compliance, multi-team pipeline management, policy controls, and enterprise-grade reporting.

**CI integration:** GitHub Actions, GitLab CI, Azure DevOps, Jenkins, CircleCI, Bitbucket.

**Limitations:**
- Vulnerability scanning only -- no behavioral/malicious package detection
- Strongest for container image scanning; language-specific lockfile scanning is secondary
- No automated fix PRs (identifies vulnerabilities but doesn't propose fixes)
- Database updates require periodic syncs
- No reachability analysis

**Complementary role:** Grype+Syft are the go-to for **container-focused** supply chain security and regulatory SBOM compliance (Executive Order 14028). They complement language-level tools like Snyk or OSV Scanner.

---

### 10. StepSecurity Harden-Runner

**What it detects/prevents:** EDR-like security agent for GitHub Actions runners. Monitors three dimensions during CI/CD execution: **network egress** (all outbound connections), **file integrity** (source code modifications during builds), and **process activity** (execution and arguments). Detects and can block exfiltration of CI/CD secrets and source code. Maintains a Global Block List of IOC domains/IPs from active supply chain attacks, enforced automatically.

**Set-and-forget configuration:** Add a single step to the beginning of each GitHub Actions job:
```yaml
- name: Harden Runner
  uses: step-security/harden-runner@v2.17.0
  with:
    egress-policy: audit  # start with audit, move to block
```
Start in **audit mode** to build a behavioral baseline from historical workflow data. The system automatically identifies expected outbound domains. Then switch to **block mode** with an allowlist of those domains. Any deviation (new outbound connection, source code modification) is flagged or blocked.

**Runner support:** GitHub-hosted runners (Linux: full support; Windows/macOS: audit mode only), self-hosted runners (Enterprise tier), Kubernetes/ARC runners (DaemonSet deployment, Enterprise tier).

**Open source vs commercial:**
- Community (Free): audit mode, domain allowlist blocking, source code modification detection, anomaly detection. Public repos only.
- Enterprise (Paid): private repos, self-hosted runners, GitHub Checks integration, file write + process visibility, GITHUB_TOKEN permission recommendations.

**CI integration:** GitHub Actions only. Not portable to GitLab CI, Jenkins, or other CI systems.

**Real-world detections:**
- tj-actions/changed-files compromise (CVE-2025-30066)
- Malicious trivy-action exfiltrating secrets (March 2026)
- Compromised axios npm package
- NX build system compromise
- Scale: 25 million+ workflow runs/week across 11,000+ projects (Microsoft, Google, CISA, Kubernetes, AWS)

**Limitations:**
- GitHub Actions only -- not applicable to other CI systems
- Block mode only on Linux runners
- Requires per-workflow configuration (can be automated via StepSecurity's online tool)
- Free tier limited to public repos
- Does not scan dependencies themselves -- only monitors runtime behavior of CI workflows

---

## Comparison Matrix

| Tool | Threat Type | Detection Method | Ecosystems | OSS? | Self-Hosted? | Platforms |
|------|------------|-----------------|------------|------|-------------|-----------|
| **Socket.dev** | Malicious packages, behavioral threats | Static + behavioral analysis | 10+ (deep: npm, PyPI) | Freemium | No | GitHub (primary); GitLab/BB/Azure (Enterprise) |
| **Snyk** | Known CVEs, container vulns | Database matching + reachability | 20+ languages, 12+ pkg mgrs | Freemium | No | GitHub, GitLab, Bitbucket, Azure, Jenkins |
| **Dependabot** | Known CVEs, outdated deps | GitHub Advisory Database | 30+ pkg managers | Free | No | GitHub only |
| **Renovate** | Outdated deps, risky updates | Version tracking + merge confidence | 90+ pkg managers | OSS (AGPL-3.0) | Yes | GitHub, GitLab, BB, Azure, Gitea |
| **OSV Scanner** | Known vulnerabilities | OSV database matching | 12+ languages | OSS (Apache 2.0) | Yes (CLI) | Any CI (GitHub Actions provided) |
| **Scorecard** | Poor security practices | Heuristic checks (19 checks) | Any repository | OSS (Apache 2.0) | Yes (CLI) | GitHub, GitLab (CLI) |
| **deps.dev** | Vulnerability + health data | Aggregated metadata | 7 ecosystems | Free API | No | API (any integration) |
| **Phylum/Veracode** | Malicious packages | Static + dynamic (sandbox) | npm, PyPI (current) | No | No | Veracode platform |
| **Grype+Syft** | Known CVEs (containers) | SBOM + database matching | OS + 12+ lang ecosystems | OSS (Apache 2.0) | Yes | Any CI |
| **Harden-Runner** | CI exfiltration, tampering | Runtime monitoring (EDR) | N/A (CI-level) | Freemium | Enterprise | GitHub Actions only |

## Complementary vs Overlapping

### Clearly Complementary (Different Threat Surfaces)

1. **Socket.dev + Snyk/OSV Scanner**: Socket catches *behavioral threats and zero-days*; Snyk/OSV catches *known CVEs*. Zero overlap.
2. **Harden-Runner + Everything Else**: Harden-Runner protects the *CI pipeline itself* from compromise. All other tools protect the *dependencies flowing through* that pipeline.
3. **Grype+Syft + Language-Level Scanners**: Grype+Syft excel at container and OS-package scanning; language-level scanners (Snyk, OSV) excel at application dependency scanning.
4. **Scorecard/deps.dev + Scanners**: Scorecard evaluates *project trustworthiness*; scanners evaluate *specific vulnerability exposure*. Use Scorecard to decide whether to adopt a dependency; use scanners to monitor it after adoption.

### Partially Overlapping (Choose Based on Needs)

1. **Dependabot vs Renovate**: Both create dependency update PRs. Choose one, not both. Renovate for multi-platform and advanced policy needs; Dependabot for GitHub-only simplicity.
2. **Snyk vs OSV Scanner vs Grype**: All scan for known vulnerabilities. Snyk adds reachability analysis and fix PRs but costs money. OSV Scanner is free and covers application dependencies well. Grype excels at container scanning. A pragmatic choice: OSV Scanner for application deps + Grype for containers (both free), or Snyk if budget allows (replaces both with better UX).
3. **Socket.dev vs Phylum/Veracode**: Both do behavioral/malicious package detection. Phylum is no longer standalone; Socket is the practical choice here.

---

## Recommended Stack

### Minimal Free Stack (Excellent Coverage)

For a GitHub-only team wanting maximum protection with no cost:

1. **Dependabot** (security alerts + version updates) -- built-in, zero effort
2. **OSV Scanner** (GitHub Action) -- vulnerability scanning with guided remediation
3. **Socket.dev** (free tier) -- behavioral threat detection on open-source repos
4. **Harden-Runner** (community tier) -- CI runtime protection on public repos
5. **Scorecard** (GitHub Action) -- assess your own project's security posture

**Cost:** $0. **Coverage:** Known CVEs, behavioral threats, CI exfiltration, project health. **Gap:** No container scanning, no private repo support for Socket/Harden-Runner free tiers.

### Recommended Production Stack

For an organization with budget and multi-ecosystem needs:

1. **Renovate** (self-hosted or Mend app) -- dependency updates with `minimumReleaseAge`, merge confidence gating, automerge for high-confidence patches
2. **Socket.dev** (Team or Business) -- behavioral analysis on all PRs, private repo support
3. **Snyk** or **OSV Scanner** -- CVE scanning with reachability (Snyk) or free/self-hosted scanning (OSV). Use Snyk if budget allows; OSV Scanner + Renovate fix PRs if not
4. **Grype + Syft** -- container SBOM generation and vulnerability scanning
5. **Harden-Runner** (Enterprise) -- CI runtime protection for private repos and self-hosted runners
6. **Scorecard** (GitHub Action + API) -- evaluate new dependencies before adoption; maintain your own project's score
7. **deps.dev API** -- integrate into dependency evaluation workflows for license, provenance, and health data

**Cost:** ~$50-100/dev/month (Socket Team/Business + optional Snyk). **Coverage:** Full threat surface -- known CVEs, behavioral threats, container vulnerabilities, CI exfiltration, dependency health, SBOM compliance.

### Defense-in-Depth Layer Map

| Layer | Threat | Tool(s) |
|-------|--------|---------|
| **Before adoption** | Risky/low-quality dependency | Scorecard, deps.dev |
| **At PR time** | Malicious package, typosquatting | Socket.dev |
| **At PR time** | Known CVE introduced | Snyk / OSV Scanner / Dependabot alerts |
| **Ongoing** | New CVE disclosed for existing dep | Snyk monitor / Dependabot alerts / OSV Scanner scheduled scan |
| **Ongoing** | Outdated dependencies | Renovate / Dependabot version updates |
| **At build time** | Container vulnerabilities | Grype + Syft |
| **At build time** | CI pipeline compromise / exfiltration | Harden-Runner |
| **Compliance** | SBOM generation | Syft (CycloneDX/SPDX output) |

---

## Open Questions

1. **Socket.dev depth beyond npm/PyPI**: How quickly is Socket expanding full behavioral analysis to Go, Java, and Rust? Current coverage for those ecosystems is lighter.
2. **Renovate + OSV Scanner integration**: Can Renovate be configured to use OSV vulnerability data to prioritize security-relevant updates? Currently these are independent tools.
3. **Harden-Runner portability**: Are there equivalent tools for GitLab CI or Jenkins, or is CI runtime protection a GitHub Actions-only capability?
4. **SLSA + Sigstore adoption curve**: How many packages in each ecosystem now publish verifiable provenance, and is it worth enforcing as a policy gate?

## Sources

All source documents saved to `docs/`:
- `socket-dev-faq.md` -- Socket.dev FAQ and technical details
- `socket-dev-review-2026.md` -- Socket.dev AppSec Santa review
- `snyk-review-2026.md` -- Snyk AppSec Santa review
- `dependabot-version-updates.md` -- GitHub Dependabot documentation
- `dependabot-vs-renovate-comparison.md` -- Dependabot vs Renovate comparison
- `renovate-configuration-options.md` -- Renovate configuration documentation
- `renovate-merge-confidence.md` -- Renovate Merge Confidence documentation
- `osv-scanner-overview.md` -- OSV-Scanner official documentation
- `openssf-scorecard-github.md` -- OpenSSF Scorecard GitHub repository
- `deps-dev-api-v3.md` -- deps.dev API v3 documentation
- `phylum-veracode-acquisition.md` -- Phylum/Veracode acquisition details
- `grype-syft-anchore-opensource.md` -- Anchore open source tools
- `stepsecurity-harden-runner-github.md` -- StepSecurity Harden-Runner
