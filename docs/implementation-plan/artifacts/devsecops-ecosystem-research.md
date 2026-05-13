# DevSecOps & Software Supply Chain Security Tool Ecosystem Research

Comprehensive survey of tools that could be integrated into or configured by gdev, a Go CLI that generates security-hardened configurations for 27 language ecosystems across macOS, Linux, and Windows.

**Last updated:** 2026-05-12

## Deduplication Note

The following tools are already covered in the gdev implementation plan and are NOT re-researched here. They are referenced only where comparison is needed:

- OSV Scanner (vulnerability scanning)
- Socket.dev (behavioral analysis)
- Renovate (dependency updates)
- Harden-Runner (CI runtime monitoring)
- Syft + sbomnix (SBOM generation)
- Nexus Community (registry proxy)
- Cachix/Attic (Nix binary cache)
- sccache (compilation cache)
- attach-guard (Claude Code package guardrails)
- Version-Sentinel (dependency version verification)
- Cosign (SBOM/container signing)
- Hadolint (Dockerfile linting)

---

## 1. Software Composition Analysis (SCA)

Tools beyond OSV Scanner and Socket.dev for identifying vulnerable, malicious, or risky dependencies.

### 1.1 Grype

| Field | Value |
|-------|-------|
| **URL** | https://github.com/anchore/grype |
| **What it does** | Vulnerability scanner for container images, filesystems, and SBOMs. Matches packages against NVD, OSV, GitHub Advisory, and distro-specific databases. Accepts Syft SBOM output directly. |
| **Integration approach** | CLI tool, CI action, pairs with Syft for SBOM-then-scan pipeline |
| **Ecosystem coverage** | 20+ language ecosystems (Go, Python, JS, Java, Rust, Ruby, PHP, .NET, Dart, Haskell, Elixir) plus all major Linux distros (Alpine, Debian, Ubuntu, RHEL, Amazon Linux, Oracle, SUSE, Arch, Gentoo) |
| **Maturity** | 11.5k GitHub stars, Anchore-backed, first release 2020, active development |
| **License** | Apache-2.0 |
| **Configuration complexity** | Minimal. Single binary, YAML config for DB sources and match settings. gdev can auto-generate `.grype.yaml` with custom severity thresholds and ignore rules |
| **Cost** | Free/open-source. Anchore Enterprise adds policy bundles and UI |

**gdev integration notes:** Grype + Syft form a natural pair — Syft generates the SBOM, Grype scans it for vulnerabilities. gdev already plans Syft integration; adding Grype as the scan step completes the pipeline. Each finding includes CVSS severity, EPSS exploit probability, and KEV catalog status for triage.

### 1.2 OWASP Dependency-Check

| Field | Value |
|-------|-------|
| **URL** | https://github.com/dependency-check/DependencyCheck |
| **What it does** | Identifies publicly disclosed vulnerabilities in project dependencies using CPE matching against NVD. Strongest in JVM ecosystems with native Maven/Gradle plugins. |
| **Integration approach** | CLI tool, Maven plugin, Gradle plugin, Ant task, Jenkins plugin, GitHub Action |
| **Ecosystem coverage** | Java (Maven, Gradle), .NET (NuGet), JavaScript (npm via RetireJS), Python (pip), Ruby (Bundler), Go modules |
| **Maturity** | 7.4k GitHub stars, OWASP Flagship Project, 290+ contributors, active since 2012 |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate. XML/CLI config, suppression files for false positives. gdev could auto-generate Maven plugin config and suppression templates |
| **Cost** | Free/open-source |

**gdev integration notes:** Best value for Java/Kotlin shops already using Maven or Gradle where the native plugin integration is seamless. However, for multi-ecosystem coverage, Grype is more versatile. Consider generating OWASP Dependency-Check configs specifically for JVM ecosystem modules as an optional enhancement.

### 1.3 Endor Labs (Commercial — noted for completeness)

| Field | Value |
|-------|-------|
| **URL** | https://www.endorlabs.com |
| **What it does** | Function-level reachability analysis that builds call graphs from source code and traces data flow to vulnerable methods. Claims 97% noise reduction by filtering vulnerabilities in unreachable code. |
| **Integration approach** | CI/CD integration, GitHub App |
| **Ecosystem coverage** | 40+ languages for SCA; function-level reachability for Java, JavaScript, Python, Go, Kotlin, .NET, Rust |
| **Maturity** | Backed by Lightspeed, Dell, and others. Customers include OpenAI, Snowflake, Atlassian |
| **License** | Commercial only, no free tier |
| **Cost** | Commercial only. No free tier. Not recommended for gdev defaults but worth noting as the reachability analysis leader |

### Category Recommendation: **Grype**

**Why:** Apache-2.0 license, single Go binary (matches gdev's distribution model), natural pairing with already-planned Syft integration, broadest ecosystem coverage of any free SCA scanner, and composite scoring (CVSS + EPSS + KEV) enables intelligent triage. gdev can auto-generate `.grype.yaml` with per-profile severity thresholds and add `grype` to the pre-commit/CI pipeline alongside Syft SBOM generation.

---

## 2. Static Application Security Testing (SAST)

Language-specific and multi-language static analysis tools that could be configured per-ecosystem.

### 2.1 Semgrep (Community Edition)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/semgrep/semgrep |
| **What it does** | Lightweight static analysis that finds bugs and enforces coding standards using pattern-matching rules that look like source code. Supports custom YAML rules and a large community rule registry. |
| **Integration approach** | CLI tool, pre-commit hook, CI action, IDE extension (VS Code, IntelliJ) |
| **Ecosystem coverage** | 30+ languages: Python, JavaScript, TypeScript, Go, Java, C, C++, C#, Ruby, Rust, Kotlin, PHP, Swift, Scala, Terraform, Dockerfile, Elixir, Dart, Clojure, Lua, R, Bash, YAML, JSON, XML, HTML, and more |
| **Maturity** | 15.1k GitHub stars, founded 2020 (from Facebook's pfff/sgrep), corporate-backed (Semgrep Inc.), very active |
| **License** | LGPL-2.1 (CE/CLI). Community rules under permissive Semgrep Rules License. Pro rules require paid plan |
| **Configuration complexity** | Low. YAML-based rules and config. gdev can auto-generate `.semgrep.yaml` with curated rule sets per ecosystem (e.g., `p/owasp-top-ten`, `p/javascript`, `p/python`, `p/golang`) |
| **Cost** | Free (CE). Semgrep AppSec Platform: Team ($40/contributor/mo) adds cross-file taint tracking, pro rules, and secrets scanning |

**gdev integration notes:** Semgrep CE is the strongest free SAST tool for gdev. Its per-language rule registries map directly to gdev's ecosystem modules. Each ecosystem module can specify its Semgrep rule sets, and gdev generates a `.semgrep.yaml` that composes them. Pre-commit hook runs in milliseconds on changed files. The YAML rule format means gdev can also ship custom consulting-firm rules (e.g., "no hardcoded AWS regions", "no eval() in Python").

### 2.2 CodeQL

| Field | Value |
|-------|-------|
| **URL** | https://github.com/github/codeql |
| **What it does** | Semantic code analysis engine that treats code as data, building a queryable database. Finds complex vulnerability patterns through data flow and taint tracking that pattern-matching tools miss. |
| **Integration approach** | GitHub Actions (primary), VS Code extension, CLI |
| **Ecosystem coverage** | C, C++, C#, Go, Java, JavaScript/TypeScript, Python, Ruby, Rust, Swift, Kotlin (beta), GitHub Actions |
| **Maturity** | GitHub/Microsoft-backed, powers GitHub Advanced Security code scanning, MIT-licensed queries |
| **License** | MIT (query libraries). CodeQL engine is proprietary but free for open-source repos. Private repo scanning requires GitHub Advanced Security ($49/active committer/mo) |
| **Configuration complexity** | Moderate. Requires database build step (compilation for compiled languages). gdev could generate `.github/codeql/codeql-config.yml` and CI workflow |
| **Cost** | Free for public repos. $49/active committer/mo for private repos (via GHAS) |

**gdev integration notes:** CodeQL is deeper than Semgrep for semantic analysis but has two barriers: (1) private repos require paid GHAS, and (2) it requires a database build step. Best integrated as CI-only (not pre-commit) with auto-generated workflow files. For a consulting firm, the $49/committer cost may be justified for client-facing code.

### 2.3 SonarQube (Community Edition)

| Field | Value |
|-------|-------|
| **URL** | https://www.sonarsource.com/open-source-editions/sonarqube-community-edition/ |
| **What it does** | Code quality and security analysis platform combining bug detection, vulnerability scanning, code smell identification, and maintainability metrics. |
| **Integration approach** | Server-based (self-hosted), CI scanner plugin, IDE (SonarLint) |
| **Ecosystem coverage** | 10+ languages: Java, C#, JavaScript/TypeScript, Python, Go, PHP, Ruby, Kotlin, Scala, C/C++ (paid only) |
| **Maturity** | Founded 2007, SonarSource-backed, millions of users |
| **License** | LGPL-3.0 (Community). Developer/Enterprise editions are commercial |
| **Configuration complexity** | High. Requires running a SonarQube server. gdev could generate `sonar-project.properties` but cannot deploy the server |
| **Cost** | Community Edition free but limited. Developer Edition starts at ~$150/year |

**gdev integration notes:** SonarQube's server-based architecture doesn't fit gdev's "single binary, zero prerequisites" model. However, gdev could generate `sonar-project.properties` for teams that already have SonarQube deployed. Lower priority than Semgrep.

### Category Recommendation: **Semgrep CE**

**Why:** LGPL-2.1 license, 30+ language coverage that nearly perfectly overlaps gdev's 27 ecosystems, YAML-based configuration that gdev can trivially auto-generate, pre-commit hook support for immediate developer feedback, and a large community rule registry organized by language and vulnerability type. The per-ecosystem rule mapping is a natural fit for gdev's module architecture.

---

## 3. Secret Scanning

Tools for detecting accidentally committed secrets beyond GitHub's built-in secret scanning.

### 3.1 Gitleaks

| Field | Value |
|-------|-------|
| **URL** | https://github.com/gitleaks/gitleaks |
| **What it does** | Scans git repositories for hardcoded secrets using regex patterns and entropy analysis. Detects API keys, tokens, passwords, and other credentials before they leave the developer's machine. |
| **Integration approach** | Pre-commit hook (native support), CLI tool, CI action (gitleaks-action), SARIF output for GitHub Advanced Security |
| **Ecosystem coverage** | Language-agnostic (scans any text file). 150+ built-in rules covering AWS, GCP, Azure, GitHub, Slack, Stripe, and many more services |
| **Maturity** | 24.4k GitHub stars, active since 2017, widely adopted |
| **License** | MIT |
| **Configuration complexity** | Low. Single `.gitleaks.toml` config file for custom rules and allowlists. gdev can auto-generate this with per-ecosystem patterns (e.g., `.env` file patterns for Node, `settings.py` patterns for Django) |
| **Cost** | Free/open-source. Gitleaks-Action v2+ requires license for commercial use ($1.99/mo individual, enterprise pricing available) |

**gdev integration notes:** Gitleaks is the ideal pre-commit secret scanner for gdev. It's a single Go binary (like gdev itself), runs in milliseconds, and its `.gitleaks.toml` config can be auto-generated with project-specific allowlists. The pre-commit hook catches secrets before they enter git history. SARIF output feeds into GitHub Security tab for CI integration.

### 3.2 TruffleHog

| Field | Value |
|-------|-------|
| **URL** | https://github.com/trufflesecurity/trufflehog |
| **What it does** | Detects and classifies 800+ secret types with live credential verification — actually tests whether a detected secret is still active by attempting authentication against the service. |
| **Integration approach** | Pre-commit hook, CLI tool, CI action, filesystem/S3/GCS scanning |
| **Ecosystem coverage** | Language-agnostic. 800+ detector types with verification for AWS, GCP, Azure, GitHub, GitLab, Slack, Twilio, Mailchimp, and many more |
| **Maturity** | 24.5k GitHub stars, Truffle Security-backed, active since 2016 (v3 rewrite in Go) |
| **License** | AGPL-3.0 (important: copyleft — may affect distribution if embedded) |
| **Configuration complexity** | Low-medium. CLI flags for scan modes; pre-commit config uses `--since-commit HEAD --results=verified --fail`. Less configurable than Gitleaks for custom rules |
| **Cost** | Free/open-source (AGPL). TruffleHog Enterprise adds team features |

**gdev integration notes:** TruffleHog's credential verification eliminates a class of false positives that plague other scanners. However, verification requires network access (not suitable for air-gapped/offline pre-commit). The AGPL-3.0 license may be a concern if gdev distributes TruffleHog configs that tightly couple to it. Best used as a CI complement to Gitleaks.

### 3.3 detect-secrets (Yelp)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/Yelp/detect-secrets |
| **What it does** | Enterprise-friendly secret scanner using a baseline approach — accepts existing secrets in a `.secrets.baseline` file while blocking new ones from being committed. Scans git diffs, not full history. |
| **Integration approach** | Pre-commit hook (native), CLI tool |
| **Ecosystem coverage** | Language-agnostic. 27 built-in plugins for various secret types, plus custom regex plugins |
| **Maturity** | 4.3k GitHub stars, Yelp-backed, active since 2018 |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low. Baseline workflow: `detect-secrets scan > .secrets.baseline`, then pre-commit hook blocks new secrets. gdev can auto-generate the baseline and hook config |
| **Cost** | Free/open-source, no paid tier |

**gdev integration notes:** The baseline approach is particularly valuable for brownfield projects (common in consulting) where legacy codebases have existing secrets that can't be immediately rotated. However, it's Python-based (heavier than Go-based alternatives) and has fewer built-in detectors than Gitleaks or TruffleHog.

### Category Recommendation: **Gitleaks**

**Why:** MIT license (most permissive), single Go binary, fastest pre-commit performance (sub-second), excellent `.gitleaks.toml` configuration that gdev can auto-generate per ecosystem, SARIF output for GitHub Security integration, and 24.4k stars indicating strong community trust. For CI pipelines, pair with TruffleHog for verification-based scanning of the full history.

**Recommended gdev strategy:** Generate Gitleaks as the pre-commit hook (fast, local, catches secrets before commit). Generate TruffleHog as the CI scanner (deeper, verifies credential liveness, scans full diffs). This layered approach gives speed locally and depth in CI.

---

## 4. Container Security

Image scanning, runtime security, and Dockerfile hardening beyond Hadolint.

### 4.1 Grype (for container images)

Already covered in Section 1.1 — Grype scans container images as a primary use case. Accepts OCI images, Docker archives, and Singularity images. Combined with Syft for SBOM generation, it provides a complete container security pipeline.

### 4.2 Trivy (IaC + Container — USE WITH CAUTION)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/aquasecurity/trivy |
| **What it does** | All-in-one security scanner: container images, filesystems, IaC (Terraform, CloudFormation, Kubernetes, Helm), secrets, and licenses in a single binary. |
| **Integration approach** | CLI tool, CI action, Kubernetes operator, IDE extension |
| **Ecosystem coverage** | Containers (OS packages + language packages), Terraform, CloudFormation, ARM, Kubernetes, Helm, Dockerfile, 20+ language ecosystems |
| **Maturity** | 32k GitHub stars, Aqua Security-backed, most popular open-source security scanner |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low. Single `trivy.yaml` config. gdev can auto-generate with severity thresholds and ignore policies |
| **Cost** | Free/open-source |

**CRITICAL WARNING — Trivy Supply Chain Compromise (March 2026):** In March 2026, Trivy suffered a severe supply chain attack (CVE-2026-33634). Attackers compromised the `trivy-action` GitHub Action, `setup-trivy`, and official Docker Hub images (tags 0.69.4-0.69.6 and `latest`). The malware ("TeamPCP Cloud Stealer") exfiltrated CI/CD secrets, SSH keys, cloud credentials, and Kubernetes secrets from over 1,000 enterprise environments. The Vect ransomware group subsequently used this stolen data for double-extortion. While Aqua Security has remediated the immediate issue, vulnerability database updates remained suspended as of late March 2026, and trust in the distribution channel has been severely damaged.

**gdev integration notes:** Despite Trivy's breadth, the March 2026 compromise makes it a risky recommendation for gdev defaults. The compromise specifically targeted the CI integration paths (GitHub Actions, Docker images) that gdev would generate configs for. Recommend Grype + Syft (Anchore ecosystem) as the primary container/SCA scanner, with Trivy as an optional alternative for teams that have verified their supply chain. gdev-generated configs should pin Trivy by hash, never by tag.

### 4.3 Dockle

| Field | Value |
|-------|-------|
| **URL** | https://github.com/goodwithtech/dockle |
| **What it does** | Container image linter focused on best practices and CIS Docker Benchmark compliance. Checks for running as root, use of HEALTHCHECK, exposed secrets in layers, and more. Complements vulnerability scanners by checking image configuration rather than package vulnerabilities. |
| **Integration approach** | CLI tool, CI action |
| **Ecosystem coverage** | Docker/OCI images |
| **Maturity** | 3.2k GitHub stars, active since 2019 |
| **License** | Apache-2.0 |
| **Configuration complexity** | Minimal. CLI flags only; `.dockleignore` for suppression. gdev can generate ignore files |
| **Cost** | Free/open-source |

**gdev integration notes:** Dockle fills the gap between Hadolint (Dockerfile linting) and Grype (vulnerability scanning) by checking the built image for configuration issues. It's a natural addition to the Docker ecosystem module's CI pipeline: Hadolint (Dockerfile) -> docker build -> Dockle (image config) -> Grype (vulnerabilities).

### Category Recommendation: **Grype** (vulnerability scanning) + **Dockle** (config compliance)

**Why:** Grype provides vulnerability scanning (already recommended for SCA), and Dockle adds CIS Docker Benchmark compliance checking that Hadolint and vulnerability scanners miss. Both are Apache-2.0, Go binaries, and low-config. Trivy is powerful but the March 2026 supply chain compromise makes it unsuitable as a gdev default until trust is fully restored.

---

## 5. Infrastructure-as-Code Security Scanning

Terraform, Kubernetes, CloudFormation security scanners.

### 5.1 Checkov

| Field | Value |
|-------|-------|
| **URL** | https://github.com/bridgecrewio/checkov |
| **What it does** | Static analysis for IaC and SCA for images/packages. Unique graph-based analysis maps relationships between resources and checks cross-resource security properties (e.g., verifies an EC2 instance's security group in a specific subnet in a specific VPC is properly configured as a chain). |
| **Integration approach** | CLI tool, pre-commit hook, CI action (Jenkins, Azure Pipelines, BitBucket, CircleCI, Argo), VS Code extension |
| **Ecosystem coverage** | Terraform, Terraform Plan, CloudFormation, AWS SAM, Kubernetes, Helm, Kustomize, Dockerfile, Serverless, Bicep, OpenAPI, ARM Templates, OpenTofu, GitHub Actions workflows |
| **Maturity** | 80M+ downloads, Palo Alto Networks-backed (acquired Bridgecrew 2021), 1,000+ built-in policies, compliance mappings for CIS Benchmarks, SOC 2, HIPAA, PCI DSS |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate. `.checkov.yaml` config file plus inline skip comments. Custom policies in Python or YAML. gdev can auto-generate `.checkov.yaml` with per-ecosystem check selections and skip rules |
| **Cost** | Free/open-source. Prisma Cloud adds enterprise features |
| **Output formats** | CLI text, JSON, JUnit XML, SARIF, CSV, CycloneDX, GitHub Markdown |

**gdev integration notes:** Checkov is the strongest IaC scanner for gdev because its graph-based analysis catches cross-resource misconfigurations that regex-based scanners miss. Its broad platform coverage (Terraform, K8s, CloudFormation, Helm, Dockerfile, OpenTofu) means a single tool covers most of gdev's IaC-related ecosystem modules. The Python runtime is the main downside (heavier than Go tools), but it's pip-installable and works in devenv.sh environments.

### 5.2 KICS (Keeping Infrastructure as Code Secure) — USE WITH CAUTION

| Field | Value |
|-------|-------|
| **URL** | https://github.com/Checkmarx/kics |
| **What it does** | IaC security scanner with 2,400+ Rego-based queries covering 22+ platforms, including niche targets like Google Deployment Manager, Pulumi, Crossplane, and Knative that other scanners miss. |
| **Integration approach** | CLI tool, CI action, Docker image |
| **Ecosystem coverage** | 22+ platforms: Terraform, CloudFormation, Kubernetes, Ansible, Docker Compose, GitHub Workflows, Helm, Pulumi, Crossplane, Knative, Serverless, OpenAPI, AWS SAM, ARM, Bicep, Google Deployment Manager |
| **Maturity** | 2.6k GitHub stars, Checkmarx-backed, CIS Level 2 certified, release cadence slowed since March 2025 |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low-moderate. CLI flags and config file. Rego-based custom queries |
| **Cost** | Free/open-source |

**CRITICAL WARNING — Checkmarx Supply Chain Compromise (April-May 2026):** In April-May 2026, Checkmarx suffered a supply chain attack by the same TeamPCP threat actor that compromised Trivy. Malicious KICS Docker images and VS Code extensions were published, and the Checkmarx Jenkins AST Plugin was also compromised. While the open-source KICS Go binary from GitHub releases appears unaffected, the broader Checkmarx ecosystem compromise means distribution channels should be verified carefully.

**gdev integration notes:** KICS has the widest platform coverage of any IaC scanner, covering niche targets like Pulumi, Crossplane, and Knative. However, the Checkmarx supply chain compromise and slowed release cadence are concerns. Recommend as an optional secondary scanner for teams using platforms not covered by Checkov.

### 5.3 TFLint

| Field | Value |
|-------|-------|
| **URL** | https://github.com/terraform-linters/tflint |
| **What it does** | Terraform-specific linter that catches provider-specific issues (e.g., invalid AWS instance types, deprecated GCP APIs) that generic IaC scanners miss. Plugin architecture supports AWS, Azure, GCP, and custom rulesets. |
| **Integration approach** | CLI tool, pre-commit hook, CI action |
| **Ecosystem coverage** | Terraform only (with provider-specific plugins for AWS, Azure, GCP) |
| **Maturity** | 5.2k GitHub stars, community-maintained, active since 2016 |
| **License** | MPL-2.0 |
| **Configuration complexity** | Low. `.tflint.hcl` config file. gdev can auto-generate with detected cloud provider plugins |
| **Cost** | Free/open-source |

**gdev integration notes:** TFLint complements Checkov for the Terraform ecosystem module. Checkov catches security misconfigs; TFLint catches provider-specific correctness issues. Both run as pre-commit hooks. gdev should generate `.tflint.hcl` with auto-detected provider plugins (AWS/Azure/GCP based on provider blocks in `.tf` files).

### Category Recommendation: **Checkov** (primary) + **TFLint** (Terraform supplement)

**Why:** Checkov has the broadest IaC coverage with unique graph-based cross-resource analysis, 1,000+ built-in policies with compliance mappings, and Apache-2.0 licensing. TFLint adds Terraform-specific provider validation that Checkov doesn't do. Together they cover both security (Checkov) and correctness (TFLint) for IaC ecosystems.

---

## 6. License Compliance

Automated license detection, policy enforcement, and SBOM-driven compliance.

### 6.1 ScanCode Toolkit

| Field | Value |
|-------|-------|
| **URL** | https://github.com/aboutcode-org/scancode-toolkit |
| **What it does** | Reference-grade license and copyright detection engine that does full text comparison against a database of license texts, rather than relying on regex or probabilistic matching. Detects licenses, copyrights, and package dependencies. |
| **Integration approach** | CLI tool, Python library, CI integration via scripts |
| **Ecosystem coverage** | Language-agnostic — scans any source code, binaries, or archives. Outputs in JSON, YAML, HTML, CycloneDX, SPDX |
| **Maturity** | ~3.8k GitHub stars, aboutcode.org/nexB-backed, considered the reference tool in the license detection domain, used by many other compliance tools internally |
| **License** | Apache-2.0 (primary), CC-BY-4.0 (reference datasets) |
| **Configuration complexity** | Moderate. CLI-driven with many scan options. No simple config file format. gdev could generate wrapper scripts with appropriate flags |
| **Cost** | Free/open-source |

**gdev integration notes:** ScanCode is the most accurate license detection engine available. However, it's Python-based and relatively slow for large codebases. Best used as a CI-only tool, not pre-commit. gdev can generate CI workflow steps that run ScanCode and output SPDX/CycloneDX license reports.

### 6.2 FOSSology

| Field | Value |
|-------|-------|
| **URL** | https://github.com/fossology/fossology |
| **What it does** | Full license compliance workflow system with database and web UI. Runs license, copyright, and export control scans. One-click SPDX file generation. |
| **Integration approach** | Server-based (self-hosted), CLI toolkit, REST API |
| **Ecosystem coverage** | Language-agnostic source scanning |
| **Maturity** | Linux Foundation project, active since 2008, mature compliance platform |
| **License** | GPL-2.0 |
| **Configuration complexity** | High. Requires server deployment with PostgreSQL database. Not suitable for auto-generation |
| **Cost** | Free/open-source |

**gdev integration notes:** FOSSology is a full compliance platform, not a tool gdev can configure with a generated config file. Too heavyweight for gdev's model. Mention in docs for teams with dedicated compliance workflows.

### 6.3 Syft + Grype License Scanning

Syft (already in the plan) can extract license information as part of SBOM generation. Grype can filter/flag specific licenses. This lightweight approach may be sufficient for most consulting engagements without a dedicated license tool.

### Category Recommendation: **ScanCode Toolkit** (CI) + **Syft license extraction** (lightweight)

**Why:** ScanCode is the reference-grade license detector with Apache-2.0 licensing and the most accurate matching engine. For lightweight needs, Syft's SBOM output already includes license data that can be policy-checked. gdev should generate: (1) Syft-based license extraction in the SBOM pipeline for all projects, and (2) ScanCode CI steps for projects that require formal license compliance (e.g., client deliverables with OSS components). A simple policy file (allowed/denied license list) can gate CI on either tool's output.

---

## 7. Dependency Pinning and Reproducibility

Tools for ensuring reproducible builds beyond lockfiles.

### 7.1 Maven-Lockfile

| Field | Value |
|-------|-------|
| **URL** | https://github.com/chains-project/maven-lockfile |
| **What it does** | Fills Maven's missing native lockfile support. Generates and validates lockfiles containing exact versions and checksums of all direct and transitive dependencies. Can reproduce builds from historical commits. |
| **Integration approach** | Maven plugin |
| **Ecosystem coverage** | Java/Maven only |
| **Maturity** | Academic project (CHAINS research group, KTH Royal Institute of Technology), relatively new, niche |
| **License** | MIT |
| **Configuration complexity** | Low. Maven plugin addition to `pom.xml`. gdev can auto-add the plugin configuration |
| **Cost** | Free/open-source |

### 7.2 Gradle Dependency Verification

| Field | Value |
|-------|-------|
| **URL** | https://docs.gradle.org/current/userguide/dependency_verification.html |
| **What it does** | Built-in Gradle feature that verifies downloaded dependency checksums and signatures against `gradle/verification-metadata.xml`. Ensures byte-level integrity of all dependencies. |
| **Integration approach** | Native Gradle feature (no plugin needed) |
| **Ecosystem coverage** | Java/Kotlin/Scala/Groovy (Gradle ecosystem) |
| **Maturity** | Part of Gradle since 6.2 (2020), mature, widely supported |
| **License** | Apache-2.0 (Gradle itself) |
| **Configuration complexity** | Low-moderate. `./gradlew --write-verification-metadata sha256,sha512` generates the metadata file. gdev can generate the initial command and config |
| **Cost** | Free (built into Gradle) |

### 7.3 npm/pnpm/yarn/bun Overrides for Integrity

These are already covered in the existing lockfile integrity research spike. The key tools are:
- `npm ci` / `pnpm install --frozen-lockfile` / `yarn install --immutable` (CI enforcement)
- npm `package-lock.json` integrity hashes, pnpm `pnpm-lock.yaml` integrity hashes
- Bun lockfile binary format with integrity checks

### 7.4 Nix Flake Locks

Already covered in the plan's Nix ecosystem module. `flake.lock` provides cryptographic pinning of all inputs by default.

### Category Recommendation: **Gradle Dependency Verification** (JVM) + **Maven-Lockfile** (Maven)

**Why:** Most ecosystems already have lockfile enforcement (covered in existing plan). The remaining gap is JVM: Maven lacks native lockfiles, and Gradle's verification-metadata goes beyond version locking to byte-level integrity. gdev should generate: (1) Maven-Lockfile plugin config for Maven projects, (2) Gradle verification-metadata initialization for Gradle projects, (3) lock enforcement CI commands for all other ecosystems (already planned). These are ecosystem-module-level configs, not cross-cutting tools.

---

## 8. CI/CD Pipeline Security

Tools for securing the build pipeline itself.

### 8.1 StepSecurity secure-repo

| Field | Value |
|-------|-------|
| **URL** | https://github.com/step-security/secure-repo |
| **What it does** | Analyzes GitHub Actions workflows and automatically applies security hardening: pins actions by SHA (not tag), adds `permissions` blocks with least-privilege, identifies risky patterns. Companion to Harden-Runner (already in plan). |
| **Integration approach** | GitHub App, CLI tool, web UI (app.stepsecurity.io) |
| **Ecosystem coverage** | GitHub Actions workflows |
| **Maturity** | StepSecurity-backed (same company as Harden-Runner), trusted by 10,000+ repos including Microsoft, Google, Kubernetes |
| **License** | Apache-2.0 |
| **Configuration complexity** | Zero for the app (auto-generates PRs). gdev could generate pre-hardened workflow templates |
| **Cost** | Free for public repos. Enterprise tier for private repos |

**gdev integration notes:** Since gdev already generates CI workflow files, the better approach is to generate them correctly from the start: pin all actions by SHA, include minimal `permissions` blocks, and follow all OSSF Scorecard best practices. StepSecurity's secure-repo can be recommended as a one-time audit tool for existing repos, but gdev's generated workflows should already incorporate these patterns.

### 8.2 OpenSSF Allstar

| Field | Value |
|-------|-------|
| **URL** | https://github.com/ossf/allstar |
| **What it does** | GitHub App that continuously enforces repository security policies: branch protection, security policy file, admin organization membership, binary artifact detection. Files issues or auto-remediates when policies are violated. |
| **Integration approach** | GitHub App (org-level or repo-level) |
| **Ecosystem coverage** | GitHub repositories (org-wide) |
| **Maturity** | OpenSSF project, Google-backed |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low. YAML config files in `.allstar/` directory. gdev could generate org-level and repo-level Allstar configs |
| **Cost** | Free/open-source |

**gdev integration notes:** Allstar enforces repository-level security settings that are orthogonal to code-level security tools. gdev could generate `.allstar/` config files that enforce branch protection rules, require SECURITY.md, and block binary artifacts. This is a low-cost, high-value addition for the consulting firm's GitHub org.

### 8.3 OpenSSF Scorecard

| Field | Value |
|-------|-------|
| **URL** | https://github.com/ossf/scorecard |
| **What it does** | Assesses open-source project security health through 18 automated checks across source code, build, dependencies, testing, and maintenance. Produces a 0-10 score per check. |
| **Integration approach** | CLI tool, GitHub Action (scorecard-action), weekly BigQuery dataset |
| **Ecosystem coverage** | Any GitHub/GitLab repository |
| **Maturity** | 5.4k GitHub stars, OpenSSF/Google-backed, Apache-2.0, Go binary |
| **License** | Apache-2.0 |
| **Configuration complexity** | Minimal. GitHub Action config only. gdev can generate the workflow |
| **Cost** | Free/open-source |

**gdev integration notes:** Scorecard is valuable both for assessing the firm's own repos and for evaluating dependencies. gdev can: (1) generate a Scorecard GitHub Action that runs on schedule and reports the repo's security score, (2) reference Scorecard in dependency evaluation guidance in generated CLAUDE.md files.

### Category Recommendation: **OpenSSF Scorecard** (assessment) + **Allstar** (enforcement)

**Why:** Scorecard and Allstar are complementary OpenSSF projects — Scorecard measures security health, Allstar enforces it. Both are Apache-2.0, backed by Google/OpenSSF, and work at the repository/org level. gdev should generate: (1) Scorecard GitHub Action for scheduled security assessment, (2) Allstar config files for policy enforcement, (3) pre-hardened CI workflow templates that satisfy Scorecard checks by construction.

---

## 9. Runtime Security

Development-time sandboxing, process isolation for build scripts.

### 9.1 Bubblewrap (bwrap)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/containers/bubblewrap |
| **What it does** | Low-level unprivileged sandboxing tool using Linux user namespaces. Creates isolated environments with restricted filesystem, network, and process visibility. Used by Flatpak and Claude Code's sandbox runtime. |
| **Integration approach** | CLI tool (wraps arbitrary commands), library |
| **Ecosystem coverage** | Linux only (macOS uses Seatbelt instead) |
| **Maturity** | Part of the containers/ GitHub org (same as Podman), used by Flatpak, stable since 2016 |
| **License** | LGPL-2.0-or-later |
| **Configuration complexity** | High. CLI flags only (no config file), requires understanding of Linux namespaces. gdev could generate wrapper scripts with pre-configured sandbox profiles |
| **Cost** | Free/open-source |

**gdev integration notes:** Bubblewrap is the foundation for sandboxing npm install scripts, pip install, and other untrusted build steps. gdev could generate sandbox wrapper scripts per ecosystem (e.g., `bwrap --ro-bind / / --tmpfs /tmp --dev /dev --unshare-net npm install` to run npm install without network after lockfile resolution). This is the most impactful runtime security feature gdev could offer but requires careful per-ecosystem tuning.

### 9.2 Firejail

| Field | Value |
|-------|-------|
| **URL** | https://github.com/netblue30/firejail |
| **What it does** | SUID namespace + seccomp sandbox with 1000+ pre-built application profiles. Easier to use than bubblewrap with declarative profile files, but requires SUID binary installation. |
| **Integration approach** | CLI tool with profile files |
| **Ecosystem coverage** | Linux only |
| **Maturity** | 6.3k GitHub stars, active since 2014, experimental Landlock support added March 2025 |
| **License** | GPL-2.0 |
| **Configuration complexity** | Low-moderate. Declarative `.profile` files. gdev could generate Firejail profiles for build tools |
| **Cost** | Free/open-source |

**gdev integration notes:** Firejail's profile-based approach is more user-friendly than bubblewrap, but the SUID requirement and GPL-2.0 license are concerns. The experimental Landlock support is promising but not mature enough for production recommendations.

### 9.3 Landlock (Linux LSM)

| Field | Value |
|-------|-------|
| **URL** | https://landlock.io/ |
| **What it does** | Linux kernel security module (LSM) that allows unprivileged processes to restrict their own filesystem and network access. No special binaries needed — just syscalls from the sandboxed process itself. |
| **Integration approach** | Kernel API (requires Linux 5.13+ for V1, 6.2+ for V3 with network) |
| **Ecosystem coverage** | Linux only (kernel 5.13+) |
| **Maturity** | In mainline kernel since 5.13 (2021), V4 with expanded capabilities in 6.7+ |
| **License** | GPL-2.0 (kernel module) |
| **Configuration complexity** | High (direct syscall API), but wrapper libraries exist in Go, Rust, Python |
| **Cost** | Free (part of Linux kernel) |

**gdev integration notes:** Landlock is the most promising long-term sandbox technology because it requires no special binaries and works unprivileged. OpenAI Codex already uses Landlock + seccomp as its default sandbox. However, direct integration requires Go FFI or a wrapper binary. Best as a future enhancement — generate configs that enable Landlock when available, fall back to bubblewrap otherwise.

### Category Recommendation: **Bubblewrap** (Linux) with platform-specific alternatives

**Why:** Bubblewrap is the proven unprivileged sandbox already used by Flatpak and Claude Code. Its LGPL-2.0 license is commercially friendly. gdev should generate ecosystem-specific bwrap wrapper scripts for sandboxing untrusted build steps (npm install, pip install, cargo build). On macOS, generate equivalent Seatbelt profiles. This is a high-impact, high-effort feature — recommend for Phase 2+ after core generation is complete.

---

## 10. Supply Chain Attestation

SLSA compliance, build provenance, artifact signing.

### 10.1 GitHub Artifact Attestations (actions/attest-build-provenance)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/actions/attest-build-provenance |
| **What it does** | Generates signed SLSA Build Level 3 provenance attestations for any artifact built in GitHub Actions. Uses Sigstore for keyless signing with short-lived OIDC-bound certificates. Attestations stored in GitHub's attestation store. |
| **Integration approach** | GitHub Action (one-step addition to any workflow) |
| **Ecosystem coverage** | Any artifact built in GitHub Actions (binaries, containers, packages) |
| **Maturity** | GitHub-maintained (first-party), generally available since June 2024 |
| **License** | MIT |
| **Configuration complexity** | Minimal. Single action step with `subject-path` or `subject-digest`. gdev can auto-generate this in any CI workflow |
| **Cost** | Free for public repos. Requires GitHub Enterprise Cloud for private repos |

**gdev integration notes:** This is the lowest-friction path to SLSA Level 3 provenance. gdev should add `actions/attest-build-provenance` to every generated GitHub Actions workflow that produces artifacts. The step is a single YAML block. Combined with Cosign verification (already in plan), this creates a complete sign-and-verify pipeline.

### 10.2 slsa-github-generator

| Field | Value |
|-------|-------|
| **URL** | https://github.com/slsa-framework/slsa-github-generator |
| **What it does** | Language-specific SLSA Level 3 provenance generators for GitHub Actions. Provides reusable workflows for Go, Node.js, containers, and generic artifacts. Provenance generation happens in an isolated workflow (key SLSA L3 requirement). |
| **Integration approach** | GitHub Actions reusable workflows |
| **Ecosystem coverage** | Go, Node.js, container images, generic artifacts |
| **Maturity** | SLSA Framework/OpenSSF-maintained, used by major projects |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate. Requires calling a reusable workflow, which restricts what can run in the same job. More complex than `actions/attest` but stronger isolation |
| **Cost** | Free/open-source |

**gdev integration notes:** For projects requiring strict SLSA L3 compliance with isolated build provenance, slsa-github-generator provides stronger guarantees than `actions/attest-build-provenance` (which runs in the same job). gdev could offer this as a "strict" profile option for regulated environments.

### 10.3 Witness (in-toto)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/in-toto/witness |
| **What it does** | Pluggable framework for software supply chain attestation using the in-toto specification. Creates an audit trail for every step of the SDLC. Includes an embedded OPA Rego policy engine for attestation verification. |
| **Integration approach** | CLI tool (wraps build commands), CI integration, Archivista server for attestation storage |
| **Ecosystem coverage** | Any build system (wraps arbitrary commands). Integrations for GitHub, GitLab, AWS, GCP |
| **Maturity** | 525 GitHub stars, donated by TestifySec to CNCF in-toto project, Apache-2.0 |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate-high. Requires defining attestation steps and policies. More complex than GitHub-native options |
| **Cost** | Free/open-source. TestifySec offers managed hosting |

**gdev integration notes:** Witness is the most flexible attestation framework but also the most complex. Best for organizations that need cross-platform attestation (not GitHub-only) or want to integrate attestation with OPA policy enforcement. For most consulting engagements on GitHub, `actions/attest-build-provenance` is sufficient.

### 10.4 GUAC (Graph for Understanding Artifact Composition)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/guacsec/guac |
| **What it does** | Aggregates SBOMs, SLSA attestations, vulnerability data, and scorecard results into a queryable graph database. Enables cross-artifact supply chain queries (e.g., "which of my services depend on this vulnerable library?"). |
| **Integration approach** | Server-based (self-hosted), GraphQL API, CLI ingest tools |
| **Ecosystem coverage** | Consumes any SBOM (CycloneDX, SPDX), SLSA attestation, OSV data, Scorecard results |
| **Maturity** | OpenSSF Incubating project, Google/Kusari/Purdue-backed, active development |
| **License** | Apache-2.0 |
| **Configuration complexity** | High. Requires server deployment. gdev cannot auto-generate a meaningful config |
| **Cost** | Free/open-source |

**gdev integration notes:** GUAC is a platform, not a tool gdev can configure. However, gdev-generated SBOMs (Syft), attestations (Cosign, actions/attest), and vulnerability reports (Grype) are all inputs GUAC can consume. gdev docs should reference GUAC for organizations that want aggregate supply chain visibility.

### Category Recommendation: **GitHub Artifact Attestations** (actions/attest-build-provenance)

**Why:** Lowest friction path to SLSA Level 3 provenance. MIT license, GitHub-maintained, single workflow step, free for public repos. gdev should add this to every generated GitHub Actions workflow that produces artifacts. For strict compliance needs, offer slsa-github-generator as an optional upgrade. Cosign (already in plan) handles verification.

---

## 11. Pre-commit Hook Ecosystem

Security-focused hooks beyond what's in the existing plan.

The existing plan covers standard pre-commit hooks for formatting, linting, and basic security. This section covers additional security-focused hooks.

### 11.1 pre-commit Framework Security Hooks (Consolidated)

The following hooks integrate with the tools recommended in this document:

| Hook | Tool | Purpose | Speed |
|------|------|---------|-------|
| `gitleaks` | Gitleaks | Secret scanning | Sub-second |
| `semgrep` | Semgrep CE | SAST | 1-5 seconds |
| `checkov` | Checkov | IaC scanning | 2-10 seconds |
| `trufflehog` | TruffleHog | Secret verification | 2-5 seconds (pre-commit mode) |
| `detect-secrets` | detect-secrets | Baseline secret scanning | Sub-second |
| `tflint` | TFLint | Terraform linting | Sub-second |

### 11.2 pre-commit Hook Security Concerns

**Important:** Dependabot now supports automatic dependency updates for pre-commit hooks (announced March 2026). This means hook versions in `.pre-commit-config.yaml` can be kept current automatically. However, this also introduces a supply chain risk — auto-updating hooks means trusting upstream repositories. gdev should pin hooks by SHA (not tag) and recommend Dependabot for managed updates with review.

### 11.3 Recommended Pre-commit Stack for gdev

**Baseline tier (all projects):**
- Gitleaks (secret scanning)
- Semgrep with ecosystem-specific rules (SAST)
- Standard formatting/linting hooks per ecosystem (already in plan)

**Enhanced tier (projects with IaC):**
- Checkov (IaC security)
- TFLint (Terraform-specific)
- Hadolint (Dockerfile — already in plan)

**Strict tier (regulated/high-security projects):**
- All of the above
- TruffleHog (pre-commit mode for credential verification)
- detect-secrets with baseline (legacy codebase adoption)

### Category Recommendation: **Gitleaks + Semgrep as default pre-commit hooks**

**Why:** These two hooks cover the highest-value security checks (secrets and code vulnerabilities) with sub-second to low-second performance. Both have official pre-commit framework support and can be auto-configured by gdev. The tiered approach lets teams opt into heavier scanning when needed.

---

## 12. Policy-as-Code

OPA/Rego, Cedar, or other policy engines for enforcing organizational security standards.

### 12.1 OPA Conftest

| Field | Value |
|-------|-------|
| **URL** | https://github.com/open-policy-agent/conftest |
| **What it does** | Tests structured configuration files against policies written in OPA Rego. Validates Terraform, Kubernetes YAML, Dockerfiles, JSON, TOML, and 18+ other formats against custom organizational policies. |
| **Integration approach** | CLI tool, pre-commit hook, CI action |
| **Ecosystem coverage** | 18+ input formats: Terraform HCL, Kubernetes YAML, Dockerfile, JSON, TOML, CUE, INI, XML, Jsonnet, HOCON, CycloneDX, SPDX, EDN, TextProto, VCL, environment files |
| **Maturity** | Part of the OPA ecosystem (CNCF Graduated), active development |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate. Rego policies require learning the language, but are powerful and expressive. Policies distribute via OCI registries, Git repos, or S3. gdev can generate a starter policy bundle and conftest config |
| **Cost** | Free/open-source |

**gdev integration notes:** Conftest is the bridge between gdev's generated configs and organizational policy enforcement. gdev can: (1) generate a `policy/` directory with starter Rego policies per ecosystem (e.g., "Dockerfiles must not use `latest` tag", "Terraform must enable encryption at rest"), (2) generate a Conftest CI step that validates all config files against the policy bundle, (3) distribute the consulting firm's standard policies via an OCI registry that teams pull automatically.

### 12.2 OPA (Open Policy Agent)

| Field | Value |
|-------|-------|
| **URL** | https://github.com/open-policy-agent/opa |
| **What it does** | General-purpose policy engine that decouples policy decision-making from services. Evaluates policies written in Rego against JSON data. Used for API authorization, Kubernetes admission control, CI/CD gating, and more. |
| **Integration approach** | CLI tool, Go library, REST API server, Kubernetes admission controller (Gatekeeper) |
| **Ecosystem coverage** | Any system that can produce JSON data for evaluation |
| **Maturity** | CNCF Graduated project, 11.5k+ GitHub stars, Google/Styra-backed |
| **License** | Apache-2.0 |
| **Configuration complexity** | Moderate-high. Rego is powerful but has a learning curve. OPA itself is a general engine — you need Conftest or Gatekeeper for specific use cases |
| **Cost** | Free/open-source. Styra DAS adds enterprise management |

**gdev integration notes:** OPA is the engine; Conftest is the interface gdev should target. Direct OPA integration is for runtime policy enforcement (API gateways, K8s admission) which is beyond gdev's scope. Conftest wraps OPA for config-file testing, which is exactly what gdev needs.

### 12.3 Kyverno

| Field | Value |
|-------|-------|
| **URL** | https://github.com/kyverno/kyverno |
| **What it does** | Kubernetes-native policy engine that validates, mutates, generates, and cleans up resources using admission controls. Policies written in YAML (no Rego needed), making them more accessible than OPA Gatekeeper. |
| **Integration approach** | Kubernetes admission controller, CLI for testing |
| **Ecosystem coverage** | Kubernetes only |
| **Maturity** | 5k+ GitHub stars, CNCF Incubating, Nirmata-backed |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low (YAML-based policies). gdev could generate Kyverno policies for K8s-based deployments |
| **Cost** | Free/open-source |

**gdev integration notes:** Kyverno is relevant only for teams deploying to Kubernetes. gdev could generate Kyverno policy YAML files as part of the Helm/Kubernetes ecosystem modules. However, this is runtime policy enforcement, not dev-time — lower priority for gdev's scope.

### 12.4 Cedar

| Field | Value |
|-------|-------|
| **URL** | https://github.com/cedar-policy/cedar |
| **What it does** | AWS-created policy language designed for fine-grained, context-aware authorization. Formally verified, deterministic, and more readable than Rego. |
| **Integration approach** | Library (Rust/Java), CLI |
| **Ecosystem coverage** | Application authorization (not config validation) |
| **Maturity** | AWS-backed, open-sourced 2023, growing adoption |
| **License** | Apache-2.0 |
| **Configuration complexity** | Low-moderate (readable syntax), but no config-file testing equivalent of Conftest |
| **Cost** | Free/open-source |

**gdev integration notes:** Cedar is designed for application-level authorization, not config-file policy testing. Not relevant for gdev's use case. Noted for completeness.

### Category Recommendation: **OPA Conftest**

**Why:** Apache-2.0 license, part of the CNCF-Graduated OPA ecosystem, supports 18+ configuration file formats that map directly to gdev's generated configs, and policies distribute via OCI registries for team-wide sharing. gdev should generate: (1) a starter Rego policy bundle with per-ecosystem security policies, (2) Conftest CI steps for config validation, (3) a policy distribution pattern using OCI registries that teams can extend. Rego has a learning curve, but Conftest's CLI makes it accessible, and gdev can ship curated policies that teams use without writing Rego themselves.

---

## Cross-Category Integration Architecture

### How These Tools Fit Together in gdev

```
Developer Workstation (pre-commit)        CI/CD Pipeline
================================         ================================
Gitleaks (secrets)                        TruffleHog (secret verification)
Semgrep CE (SAST)                         Semgrep CE (full scan)
Checkov (IaC)                             Checkov (full IaC scan)
TFLint (Terraform)                        Grype + Syft (SCA + SBOM)
Hadolint (Dockerfile) [existing]          ScanCode (license compliance)
                                          Dockle (container config)
                                          Scorecard (repo health)
                                          actions/attest-build-provenance
                                          Conftest (policy enforcement)
                                          Cosign (signing) [existing]

Repository Configuration                  Org-Level Enforcement
================================         ================================
.gitleaks.toml                            Allstar (.allstar/ configs)
.semgrep.yaml                             OCI policy registry (Conftest)
.checkov.yaml                             
.tflint.hcl                               
.grype.yaml                               
policy/ (Rego policies)                    
```

### Per-Ecosystem Tool Mapping

| Ecosystem | Pre-commit Hooks | CI Scanners | Config Files Generated |
|-----------|-----------------|-------------|----------------------|
| **JS/TS** | Gitleaks, Semgrep (p/javascript) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |
| **Python** | Gitleaks, Semgrep (p/python) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |
| **Go** | Gitleaks, Semgrep (p/golang) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |
| **Rust** | Gitleaks, Semgrep (p/rust) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |
| **Java/Kotlin** | Gitleaks, Semgrep (p/java) | Grype, OWASP Dep-Check, Semgrep | `.gitleaks.toml`, `.semgrep.yaml`, Maven/Gradle verification configs |
| **.NET** | Gitleaks, Semgrep (p/csharp) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |
| **Docker** | Hadolint, Gitleaks | Grype, Dockle, Checkov | `.hadolint.yaml`, `.gitleaks.toml`, `.checkov.yaml` |
| **Terraform** | Checkov, TFLint, Gitleaks | Checkov, Conftest | `.checkov.yaml`, `.tflint.hcl`, `policy/` |
| **Helm** | Checkov, Gitleaks | Checkov, Conftest | `.checkov.yaml`, `policy/` |
| **Kubernetes** | Checkov, Gitleaks | Checkov, Conftest | `.checkov.yaml`, `policy/` |
| **All others** | Gitleaks, Semgrep (language-specific) | Grype, Semgrep | `.gitleaks.toml`, `.semgrep.yaml` |

### Priority Implementation Order

**Phase 5 additions (Security & Infrastructure Integration):**
1. Gitleaks — secret scanning pre-commit hook + `.gitleaks.toml` generation
2. Semgrep CE — SAST pre-commit hook + `.semgrep.yaml` generation with per-ecosystem rule sets
3. Grype — vulnerability scanning CI step + `.grype.yaml` generation (pairs with existing Syft)
4. GitHub Artifact Attestations — `actions/attest-build-provenance` in generated CI workflows

**Phase 7 additions (Ecosystem Modules Tiers 2-4):**
5. Checkov — IaC scanning pre-commit + CI + `.checkov.yaml` generation
6. TFLint — Terraform-specific linting + `.tflint.hcl` generation
7. Dockle — container image config compliance in CI

**Phase 8 additions (Migration, Update & Polish):**
8. Conftest — policy-as-code CI step + starter Rego policies
9. Scorecard — scheduled security assessment GitHub Action
10. Allstar — org-level policy enforcement configs
11. ScanCode — license compliance CI step (opt-in)

**Future / Optional:**
12. Bubblewrap sandbox wrappers (high-effort, high-impact)
13. Gradle Dependency Verification / Maven-Lockfile (JVM-specific)
14. TruffleHog CI integration (complements Gitleaks)

---

## Tool Summary Matrix

| Tool | Category | License | Language | Pre-commit | CI | Config Autogen | Priority |
|------|----------|---------|----------|------------|-----|----------------|----------|
| **Grype** | SCA | Apache-2.0 | Go | No | Yes | `.grype.yaml` | High |
| **Semgrep CE** | SAST | LGPL-2.1 | OCaml/Python | Yes | Yes | `.semgrep.yaml` | High |
| **Gitleaks** | Secrets | MIT | Go | Yes | Yes | `.gitleaks.toml` | High |
| **Checkov** | IaC | Apache-2.0 | Python | Yes | Yes | `.checkov.yaml` | High |
| **GitHub Attestations** | Attestation | MIT | N/A | No | Yes | Workflow YAML | High |
| **TFLint** | IaC (TF) | MPL-2.0 | Go | Yes | Yes | `.tflint.hcl` | Medium |
| **Dockle** | Container | Apache-2.0 | Go | No | Yes | `.dockleignore` | Medium |
| **Conftest** | Policy | Apache-2.0 | Go | Yes | Yes | `policy/` dir | Medium |
| **Scorecard** | CI/CD | Apache-2.0 | Go | No | Yes | Workflow YAML | Medium |
| **Allstar** | CI/CD | Apache-2.0 | Go | No | N/A | `.allstar/` | Medium |
| **ScanCode** | License | Apache-2.0 | Python | No | Yes | CI script | Low |
| **TruffleHog** | Secrets | AGPL-3.0 | Go | Optional | Yes | CI config | Low |
| **Bubblewrap** | Runtime | LGPL-2.0 | C | No | No | Wrapper scripts | Future |
| **OWASP Dep-Check** | SCA (JVM) | Apache-2.0 | Java | No | Yes | Maven/Gradle plugin | Low |
| **Maven-Lockfile** | Pinning | MIT | Java | No | No | Maven plugin config | Low |
| **Gradle Verification** | Pinning | Apache-2.0 | Kotlin | No | Yes | CLI command | Low |
| **Kyverno** | Policy (K8s) | Apache-2.0 | Go | No | No | Policy YAML | Future |
| **GUAC** | Visibility | Apache-2.0 | Go | No | No | N/A (platform) | Future |
| **Witness** | Attestation | Apache-2.0 | Go | No | Yes | CLI config | Future |

---

## Critical Security Advisories (2026)

### Trivy Supply Chain Compromise (March 2026)
CVE-2026-33634. TeamPCP threat actor compromised Trivy's GitHub Actions, Docker Hub images, and release automation. Over 1,000 enterprise environments affected. Malware exfiltrated CI/CD secrets, SSH keys, and cloud credentials. Vect ransomware group used stolen data for double-extortion.

**gdev implication:** Do NOT use Trivy as a default tool in generated configs. Recommend Grype + Syft as the primary scanner stack. If teams insist on Trivy, generated configs must pin by hash (not tag) and verify checksums.

### KICS/Checkmarx Supply Chain Compromise (April-May 2026)
Same TeamPCP actor compromised Checkmarx's KICS Docker images, VS Code extensions, and Jenkins AST Plugin.

**gdev implication:** Use KICS with caution. Pin by hash if used. Recommend Checkov as the primary IaC scanner.

### Broader Implications
The Trivy and KICS compromises demonstrate that security tools themselves are high-value supply chain targets. gdev's generated configs should:
1. Pin all tools and actions by SHA hash, never by mutable tag
2. Verify checksums where possible
3. Prefer tools distributed as static binaries (Go) over those requiring runtime installation (npm, pip)
4. Include Harden-Runner (already in plan) to detect exfiltration attempts during CI
