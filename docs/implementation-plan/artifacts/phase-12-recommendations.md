# Phase 12: Extended Integrations — Recommendations

## Phase Overview

Phase 12 extends gdev with high-leverage integrations across four areas: security hardening (SAST, secret scanning, container security, license compliance), developer experience (secret management, changelog automation, CI workflow generation), AI agent enhancement (context management, code review), and infrastructure (MCP server curation, additional pre-commit hooks). Every tool selected passes a five-point consulting-firm filter: works across all client projects, minimal maintenance burden, commercially licensable (MIT/Apache), mature enough to depend on, and clear value that justifies the complexity. Tools that fail any criterion are explicitly rejected with reasons.

---

## Recommended Tool List

### Security Hardening

#### 1. Semgrep Community Edition (SAST)

- **URL:** https://github.com/semgrep/semgrep
- **License:** LGPL-2.1 (CE CLI); AppSec Platform free tier for <=10 contributors
- **What it does:** Fast, multi-language static analysis with 3,000+ community rules and regex-like pattern matching that developers can actually write custom rules for.
- **Why it matters:** A consulting firm needs one SAST tool that works across all 27 ecosystems without per-client licensing — Semgrep CE is the only tool that achieves this at $0 with meaningful coverage.
- **Integration approach:** gdev generates a `.semgrep.yml` rule config per detected ecosystem (e.g., `p/owasp-top-ten`, `p/golang`, `p/typescript`, `p/python`), adds `semgrep` to devenv.nix packages, registers a pre-commit hook (`semgrep --config auto`), and includes a CI step in generated workflows. The ecosystem module interface gets a `SemgrepRuleSets() []string` method.
- **Maturity:** Stable. 11K+ GitHub stars, backed by Semgrep Inc (formerly r2c), used by Dropbox, Figma, Snowflake. CE has been stable since 2020. v1.160.0 (April 2026) added Scala tree-sitter support.
- **Priority:** Must-have

#### 2. Gitleaks (Secret Scanning)

- **URL:** https://github.com/gitleaks/gitleaks
- **License:** MIT
- **What it does:** Detects 150+ secret patterns (AWS keys, GitHub tokens, database credentials) in staged git changes and commit history using regex and entropy analysis.
- **Why it matters:** 28 million credentials leaked on GitHub in 2025 alone — pre-commit secret blocking is the single highest-ROI security control after age-gating, and gitleaks is the industry standard (GitLab switched from ripsecrets to gitleaks).
- **Integration approach:** gdev adds `gitleaks` to devenv.nix packages, registers it as a baseline pre-commit hook (always-on alongside ripsecrets — belt and suspenders for the highest-risk category), generates a `.gitleaks.toml` with team-specific allowlists (internal registry URLs, test fixtures), and adds a CI workflow step for full-history scanning. Replaces ripsecrets as primary (ripsecrets stays as fallback for speed).
- **Maturity:** Stable. 26K+ GitHub stars, Go single binary, used by CISA, GitLab, and major enterprises. Active development through 2026.
- **Priority:** Must-have

#### 3. Grype (Container Image Scanning)

- **URL:** https://github.com/anchore/grype
- **License:** Apache-2.0
- **What it does:** Scans container images and filesystems for known vulnerabilities using NVD, GitHub Advisories, and distro-specific databases, with composite risk scoring via EPSS and KEV.
- **Why it matters:** The March 2026 Trivy supply chain compromise (malicious releases, poisoned Docker images, suspended vuln DB updates) makes Grype the necessary primary scanner — it provides EPSS-based prioritization that Trivy lacked even before the compromise.
- **Integration approach:** gdev generates a CI workflow step (`grype <image>:<tag> --fail-on high`), configures Grype alongside Syft in the SBOM pipeline (`syft <image> -o cyclonedx-json | grype`), and adds Docker ecosystem module support for local image scanning via `gdev docker scan`. Generated Dockerfiles include a `# Scan: grype` comment with the exact scan command.
- **Maturity:** Stable. 9.3K+ GitHub stars, Apache-2.0, maintained by Anchore alongside Syft. Actively updated vulnerability database (unlike Trivy post-compromise).
- **Priority:** Must-have

#### 4. Syft (SBOM Generation)

- **URL:** https://github.com/anchore/syft
- **License:** Apache-2.0
- **What it does:** Generates Software Bill of Materials (SBOM) from container images and filesystems in CycloneDX, SPDX, and other formats, covering 40+ package ecosystems.
- **Why it matters:** SBOM generation is increasingly mandated by client contracts (especially government and regulated industries) — Syft + Grype is the open-source standard pipeline that satisfies these requirements at $0.
- **Integration approach:** gdev generates CI workflow steps for SBOM creation (`syft <image> -o cyclonedx-json > sbom.json`), adds `syft` to devenv.nix packages when Docker ecosystem is detected, and configures the Syft-to-Grype pipeline. The infrastructure profile's SBOM config (Phase 1, Unit 1.8) gains a concrete implementation. Pairs with cosign for signed SBOM attestations.
- **Maturity:** Stable. 8.4K+ GitHub stars, Apache-2.0, maintained by Anchore. v1.42.0 (Feb 2026). Powers `docker sbom` under the hood.
- **Priority:** Must-have

#### 5. Cosign / Sigstore (Container Signing)

- **URL:** https://github.com/sigstore/cosign
- **License:** Apache-2.0
- **What it does:** Keyless container image signing and verification using OIDC identity (GitHub Actions, Google, Microsoft) with transparency logging via Rekor.
- **Why it matters:** Container signing closes the gap between "we scanned the image" and "we can prove this is the image we scanned" — critical for consulting engagements with supply chain requirements and the SLSA framework.
- **Integration approach:** gdev generates CI workflow steps for keyless signing in GitHub Actions (`cosign sign --yes <image>@<digest>`), adds verification commands to Docker ecosystem module (`cosign verify`), and configures `.cosign/` policy files. Integrates with the existing Helm and Docker modules for OCI artifact verification.
- **Maturity:** Stable. Part of the Linux Foundation's Sigstore project, backed by Google, Red Hat, and Chainguard. SLSA L3 standard tooling. Widely adopted in Kubernetes ecosystem.
- **Priority:** Should-have

#### 6. ScanCode Toolkit (License Compliance)

- **URL:** https://github.com/aboutcode-org/scancode-toolkit
- **License:** Apache-2.0 / CC-BY-4.0 (license data)
- **What it does:** Scans codebases and dependencies for license information, copyright notices, and license compatibility issues across all package ecosystems.
- **Why it matters:** A consulting firm working across dozens of client projects must catch GPL-in-proprietary-codebase violations before they become legal problems — ScanCode is the only fully open-source tool that does this at audit-grade accuracy.
- **Integration approach:** gdev generates a CI workflow step for license scanning (`scancode --license --copyright --json-pp results.json .`), a `.scancode.yml` config with the firm's license policy (allowlist: MIT, Apache-2.0, BSD-2/3, ISC; blocklist: GPL-2.0, GPL-3.0, AGPL; review: LGPL, MPL), and a summary report command. Runs as a periodic CI job (weekly, not per-commit — too slow for pre-commit).
- **Maturity:** Stable. 5.8K+ GitHub stars, Apache-2.0, maintained by AboutCode/nexB since 2016. Used by Eclipse Foundation, Debian, and OpenSSF. Slower than commercial alternatives but zero-cost and audit-grade.
- **Priority:** Should-have

### Developer Experience

#### 7. SecretSpec (Development Secret Management)

- **URL:** https://secretspec.dev / https://github.com/cachix/secretspec
- **License:** Apache-2.0
- **What it does:** Declarative secrets management that separates secret declaration (what secrets your app needs) from provisioning (where they come from) — each developer, CI, and production uses their preferred provider.
- **Why it matters:** Every consulting project needs database passwords, API keys, and service credentials during development — SecretSpec is the devenv-native solution that ships with devenv 2.0, avoiding the need for a separate tool and working seamlessly with the devenv environment gdev already generates.
- **Integration approach:** gdev generates a `secretspec.toml` declaring secrets detected from ecosystem and service configuration (e.g., `DATABASE_URL` when PostgreSQL service is enabled, `AWS_ACCESS_KEY_ID` when Terraform detected), with provider configuration defaulting to `keyring` for local dev and `env` for CI. The devenv.nix template's `secretspec` integration block is populated. Auto-generated secrets (local DB passwords, session keys) use SecretSpec 0.7's declarative generation feature.
- **Maturity:** Emerging but stable for its scope. Ships with devenv 2.0 (SecretSpec 0.7.2). Providers: keyring, dotenv, env, 1Password, LastPass. SOPS not yet supported as a provider — document this limitation.
- **Priority:** Should-have

#### 8. git-cliff (Changelog Automation)

- **URL:** https://github.com/orhun/git-cliff
- **License:** MIT / Apache-2.0 (dual-licensed)
- **What it does:** Generates changelogs from conventional commit messages using regex-powered parsers and Tera templates, processing 10,000 commits in 120ms.
- **Why it matters:** Consulting firms need standardized release processes across all client projects — git-cliff is the only changelog tool that works with any language (Rust single binary, no Node/Python dependency), is fast enough for CI, and configurable enough for different client conventions.
- **Integration approach:** gdev generates a `cliff.toml` configuration file with the firm's standard changelog format, adds `git-cliff` to devenv.nix packages, and includes a CI workflow step for release changelog generation. The devinit wizard offers conventional commits enforcement as a pre-commit hook option (commitlint or git-cliff's built-in parser).
- **Maturity:** Stable. 11K+ GitHub stars, dual MIT/Apache-2.0, Rust binary. v2.12.0 (Jan 2026). Active single-maintainer project with broad adoption.
- **Priority:** Nice-to-have

#### 9. CI Workflow Generation (GitHub Actions + GitLab CI)

- **URL:** N/A — this is a gdev-native capability, not a third-party tool
- **License:** N/A
- **What it does:** Generates CI workflow files (`.github/workflows/security.yml` and `.gitlab-ci.yml`) that enforce all security policies configured by gdev, using SHA-pinned action references.
- **Why it matters:** Security controls that only exist locally are worthless — CI is the enforcement backstop, and a consulting firm needs generated CI workflows that match the local security config exactly, preventing drift between "what gdev configured" and "what CI enforces."
- **Integration approach:** gdev generates workflows that compose steps from detected ecosystems: frozen-install commands, vulnerability scanning (OSV Scanner + Grype), secret scanning (gitleaks), SAST (Semgrep), license compliance (ScanCode — weekly schedule), SBOM generation (Syft), container signing (cosign), and the Claude Code Security Review GitHub Action. All actions SHA-pinned per GitHub's 2026 security recommendations. Supports both GitHub Actions and GitLab CI as generation targets (selected via wizard or `--ci-platform` flag). Harden-Runner included in audit mode.
- **Maturity:** The generated workflows use only stable, well-maintained actions. This is a code generation capability, not a dependency.
- **Priority:** Must-have

### AI Agent Enhancement

#### 10. Claude Code Security Review GitHub Action

- **URL:** https://github.com/anthropics/claude-code-security-review
- **License:** Proprietary (Anthropic) — costs ~$0.90-$1.80 per 500-line PR via API key
- **What it does:** AI-powered PR security review that performs deep semantic analysis for broken access control, business-logic flaws, insecure deserialization, auth bypass, and DNS rebinding — language-agnostic.
- **Why it matters:** This caught RCE vulnerabilities in Claude Code's own codebase — it is the highest-signal automated security review available and uniquely positioned since gdev already generates Claude Code configuration.
- **Integration approach:** gdev includes the action in generated CI workflows when the Claude Code addon is enabled, with the API key sourced from a repository secret (`ANTHROPIC_API_KEY`). Generated workflow includes the action with `model: claude-sonnet-4-20250514` (configurable) and appropriate file filtering.
- **Maturity:** Stable. Official Anthropic product, actively maintained. Cost is per-scan, not per-seat — reasonable for a consulting firm.
- **Priority:** Should-have

#### 11. Context7 MCP Server (Documentation Context)

- **URL:** https://github.com/upstash/context7
- **License:** Apache-2.0
- **What it does:** Provides up-to-date library and framework documentation as context for AI agents via MCP, eliminating hallucinated API calls from outdated training data.
- **Why it matters:** A consulting firm works across dozens of frameworks per quarter — Context7 prevents AI agents from generating code against stale API signatures, which is the single most common failure mode when working on unfamiliar client stacks.
- **Integration approach:** gdev adds Context7 to the generated `.mcp.json` as a default-on MCP server alongside the existing GitHub MCP and Socket.dev MCP. No configuration needed — it auto-resolves library documentation from the project's dependencies.
- **Maturity:** Emerging but well-adopted. Apache-2.0, backed by Upstash. Recommended in multiple "best MCP servers for 2026" lists. Low risk — it's a read-only documentation provider.
- **Priority:** Should-have

### Infrastructure

#### 12. Harden-Runner (CI Runtime Monitoring)

- **URL:** https://github.com/step-security/harden-runner
- **License:** Proprietary (community tier free for public repos; paid for private repos)
- **What it does:** EDR for GitHub Actions runners — monitors network egress, file integrity, and process activity, detecting supply chain attacks in real-time with a global block list maintained by StepSecurity's SOC.
- **Why it matters:** Harden-Runner detected the Trivy compromise, the tj-actions/changed-files compromise (CVE-2025-30066), and the axios npm package compromise — it is the only tool that provides runtime defense for CI pipelines, which is the highest-value attack surface after package registries.
- **Integration approach:** gdev includes `step-security/harden-runner@<sha>` as the first step in all generated CI workflows, configured in `audit` mode by default (logs but doesn't block — safe for initial rollout). The infrastructure profile supports switching to `block` mode with an explicit egress allowlist per ecosystem.
- **Maturity:** Stable. Used by CISA, Microsoft, and the OpenSSF. eBPF-based detection on self-hosted runners. Active 2026 development including Kubernetes ARC support.
- **Priority:** Must-have (already referenced in Phase 5; this formalizes the integration)

---

## Proposed Units

### Unit 12.1: SAST Integration — Semgrep Per-Ecosystem Configuration

**Description:** Extend ecosystem modules with Semgrep rule set selection and generate `.semgrep.yml` project configuration, pre-commit hook, and CI workflow step.

**Context:** Semgrep CE provides single-file SAST with 3,000+ community rules. The free AppSec Platform tier extends this to cross-file analysis for teams <=10 contributors. Each ecosystem maps to specific Semgrep rule sets (e.g., Go maps to `p/golang` + `p/owasp-top-ten`, TypeScript to `p/typescript` + `p/react` + `p/owasp-top-ten`). The `.semgrep.yml` config aggregates rule sets from all detected ecosystems.

**Desired Outcome:** `gdev init` generates a project-appropriate Semgrep configuration, a pre-commit hook for fast local SAST, and a CI workflow step for comprehensive scanning.

**Steps:**
1. Add `SemgrepRuleSets() []string` method to `EcosystemModule` interface (or supplementary `SASTModule` interface).
2. Implement rule set mappings for all Tier 1 ecosystems:
   - Go: `p/golang`, `p/owasp-top-ten`
   - JS/TS: `p/typescript`, `p/javascript`, `p/react`, `p/nextjs`, `p/owasp-top-ten`, `p/xss`
   - Python: `p/python`, `p/django`, `p/flask`, `p/owasp-top-ten`
   - Rust: `p/rust`, `p/owasp-top-ten`
   - Java/Kotlin: `p/java`, `p/kotlin`, `p/spring`, `p/owasp-top-ten`
   - .NET: `p/csharp`, `p/owasp-top-ten`
   - Docker: `p/dockerfile`
   - Terraform: `p/terraform`, `p/terraform-aws`
3. Generate `.semgrep.yml` aggregating rule sets from detected ecosystems, with path exclusions for `vendor/`, `node_modules/`, `dist/`, `.devenv/`.
4. Add `semgrep` to devenv.nix packages.
5. Register pre-commit hook: `semgrep --config auto --error` (uses local config, fails on findings).
6. Generate CI workflow step: `semgrep ci` (respects `.semgrep.yml`).
7. Support team custom rules via `.semgrep/` directory (gdev doesn't generate these but documents the pattern).

**Acceptance Criteria:**
- [ ] `.semgrep.yml` generated with ecosystem-appropriate rule sets
- [ ] Pre-commit hook runs Semgrep on staged files
- [ ] CI step runs full project scan
- [ ] Path exclusions prevent false positives from vendored/generated code
- [ ] Custom rule directory documented in generated CLAUDE.md
- [ ] Unit tests verify rule set aggregation for multi-ecosystem projects

**Research Citations:**
- `artifacts/claude-code-ecosystem-research.md § 5.3 Official Marketplace Scanning Plugins` — Semgrep as SAST tool
- `artifacts/language-ecosystem-coverage.md` — per-ecosystem tooling
- Web research: Semgrep v1.160.0 feature set and rule registry

**Status:** Not Started

---

### Unit 12.2: Secret Scanning — Gitleaks Pre-Commit and CI Integration

**Description:** Replace ripsecrets as the primary secret scanner with gitleaks, generating `.gitleaks.toml` configuration with team-specific allowlists, a pre-commit hook, and CI full-history scanning.

**Context:** Phase 5 (Unit 4.2) configured ripsecrets as the baseline pre-commit secret scanner. Gitleaks has become the industry standard — GitLab switched from ripsecrets to gitleaks, and gitleaks detects 150+ secret patterns vs ripsecrets' more conservative set. This unit upgrades the secret scanning layer while preserving ripsecrets as an ultra-fast backup (belt and suspenders for the highest-risk category).

**Desired Outcome:** Gitleaks is the primary secret scanner in pre-commit hooks and CI, with a team-configurable allowlist for false positive management.

**Steps:**
1. Add `gitleaks` to devenv.nix packages (Go single binary, no dependencies).
2. Generate `.gitleaks.toml` with:
   - Standard rule set (built-in 150+ patterns)
   - Allowlist section for the firm's internal registry URLs, test fixtures, and known false positives
   - Path exclusions: `vendor/`, `node_modules/`, `.devenv/`, `docs/`
   - Infrastructure profile integration: allowlist registry proxy URLs when configured
3. Register gitleaks as a baseline pre-commit hook (tier 1, always-on): `gitleaks protect --staged`.
4. Retain ripsecrets as a secondary hook (belt-and-suspenders) — it runs in ~10ms so the overhead is negligible.
5. Generate CI workflow step: `gitleaks detect --source . --report-format sarif --report-path gitleaks-report.sarif` for full-repo scanning.
6. Add to generated CLAUDE.md: instructions for managing `.gitleaks.toml` allowlists when false positives occur.
7. Support `--gitleaks-allowlist <path>` flag for team-specific allowlist file inclusion.

**Acceptance Criteria:**
- [ ] `.gitleaks.toml` generated with appropriate allowlists
- [ ] Pre-commit hook blocks commits with detected secrets
- [ ] CI step scans full repository history
- [ ] Allowlist includes infrastructure profile registry URLs
- [ ] SARIF output for CI integration with GitHub Code Scanning
- [ ] False positive management documented in CLAUDE.md

**Research Citations:**
- `research-spikes/devenv-security/precommit-hooks-research.md` — original hook tier design
- `artifacts/claude-code-ecosystem-research.md` — secret scanning tool landscape
- Web research: Gitleaks 26K+ stars, GitLab migration from ripsecrets, 150+ patterns

**Status:** Not Started

---

### Unit 12.3: Container Security Pipeline — Grype + Syft + Cosign

**Description:** Implement the container security pipeline: Syft generates SBOMs, Grype scans for vulnerabilities, and Cosign signs verified images — all wired into the Docker ecosystem module and CI workflows.

**Context:** Phase 2 (Unit 2.7) configured Hadolint for Dockerfile linting and referenced Trivy for image scanning. The March 2026 Trivy supply chain compromise (malicious releases, poisoned Docker images, suspended vuln DB) necessitates Grype as the primary scanner. This unit builds the full container security pipeline that Phase 5 (Unit 4.4) referenced but didn't implement.

**Desired Outcome:** Docker ecosystem projects get a complete scan-sign-verify pipeline in CI and local development.

**Steps:**
1. Add `grype`, `syft`, and `cosign` to devenv.nix packages when Docker ecosystem detected.
2. Extend Docker ecosystem module's `CICommands()` with the pipeline:
   - `syft <image>:<tag> -o cyclonedx-json > sbom.json` (SBOM generation)
   - `grype sbom:sbom.json --fail-on high` (vulnerability scanning against SBOM)
   - `cosign sign --yes <image>@<digest>` (keyless signing in CI via OIDC)
   - `cosign verify <image>@<digest> --certificate-identity-regexp=...` (verification)
3. Generate CI workflow steps with correct ordering: build -> scan -> sign -> push.
4. Configure Grype's vulnerability database update in CI (auto-updates on each run).
5. Generate `.grype.yaml` for local scanning: failure threshold, ignored CVEs list.
6. Add `gdev docker scan <image>` command that runs the Syft+Grype pipeline locally.
7. Include Trivy compromise warning in generated security documentation as real-world trust model example.
8. Configure cosign policy file (`.cosign/policy.yaml`) with keyless verification settings.

**Acceptance Criteria:**
- [ ] Syft SBOM generation produces valid CycloneDX JSON
- [ ] Grype scanning fails CI on high/critical vulnerabilities
- [ ] Cosign keyless signing works in GitHub Actions via OIDC
- [ ] Local scan command works without CI infrastructure
- [ ] Trivy compromise documented as trust model lesson
- [ ] Pipeline ordering correct in generated CI workflows (build -> SBOM -> scan -> sign -> push)

**Research Citations:**
- `artifacts/artifact-stores-caches-research.md § 5. SBOM and Provenance Tools` — Syft + Grype pipeline
- `artifacts/artifact-stores-caches-research.md § Sigstore/cosign` — container signing
- Validation: Trivy compromise confirmed March 2026
- Web research: Grype EPSS scoring, Syft 40+ ecosystems, cosign keyless OIDC signing

**Status:** Not Started

---

### Unit 12.4: License Compliance Scanning

**Description:** Generate license compliance configuration using ScanCode Toolkit with the firm's standard license policy, a CI workflow for periodic scanning, and a summary report command.

**Context:** A consulting firm working across dozens of client projects faces significant legal risk from accidental GPL inclusion in proprietary codebases. License compliance scanning is the legal equivalent of vulnerability scanning — it catches problems before they become expensive. ScanCode runs as a periodic CI job (not pre-commit — too slow) and produces machine-readable reports.

**Desired Outcome:** Weekly CI license scans enforce the firm's license policy across all projects.

**Steps:**
1. Generate `.scancode.yml` license policy configuration:
   - Allowed: MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, Unlicense, CC0-1.0, 0BSD
   - Blocked: GPL-2.0, GPL-3.0, AGPL-3.0, SSPL-1.0, EUPL-1.1
   - Review required: LGPL-2.1, LGPL-3.0, MPL-2.0, CDDL-1.0, EPL-2.0
   - Policy configurable via infrastructure profile
2. Generate CI workflow for weekly scheduled scanning:
   - `scancode --license --copyright --json-pp license-report.json .`
   - Policy evaluation step that fails if blocked licenses detected
   - Summary output for PR comment or Slack notification
3. Add `scancode-toolkit` to devenv.nix packages (Python-based, installed via pip/uvx in devenv).
4. Generate `gdev license check` command that runs ScanCode locally with the same policy.
5. Support license exception file (`.license-exceptions.yml`) for known acceptable exceptions with justification.
6. Document in CLAUDE.md: how to handle license policy violations, how to request exceptions.

**Acceptance Criteria:**
- [ ] `.scancode.yml` generated with firm's standard license policy
- [ ] CI workflow runs weekly and reports violations
- [ ] Blocked licenses cause CI failure
- [ ] Review-required licenses create warnings (not failures)
- [ ] Exception file supports justified overrides
- [ ] Policy is configurable via infrastructure profile

**Research Citations:**
- `artifacts/artifact-stores-caches-research.md` — SBOM and compliance tooling
- Web research: ScanCode Toolkit 5.8K stars, Apache-2.0, audit-grade accuracy, used by Eclipse Foundation

**Status:** Not Started

---

### Unit 12.5: Development Secret Management via SecretSpec

**Description:** Generate SecretSpec configuration (`secretspec.toml`) declaring development secrets inferred from detected ecosystems and services, with provider defaults for local dev and CI.

**Context:** Phase 3 (Unit 2.3) generates devenv.nix service sub-templates for PostgreSQL, Redis, etc. These services need credentials. Phase 5 configures environment variable management. SecretSpec (shipped with devenv 2.0) provides the native bridge — it declares what secrets exist and lets each environment provide them differently. This unit connects the service detection to secret declaration.

**Desired Outcome:** `gdev init` generates a `secretspec.toml` declaring all development secrets, with auto-generation for local-only secrets and provider configuration for shared secrets.

**Steps:**
1. Add `SecretDeclarations() []SecretDecl` method to ecosystem modules and service templates.
2. Define `SecretDecl` struct: `Name`, `Description`, `Required bool`, `AutoGenerate bool`, `GenerateSpec` (length, charset), `Environments []string`.
3. Implement secret declarations for services:
   - PostgreSQL: `DATABASE_URL`, `POSTGRES_PASSWORD` (auto-generate for local)
   - Redis: `REDIS_URL` (auto-generate for local)
   - RabbitMQ: `RABBITMQ_DEFAULT_PASS` (auto-generate for local)
4. Implement secret declarations for ecosystems:
   - Terraform: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (no auto-generate — must be provided)
   - Docker: `DOCKER_REGISTRY_TOKEN` (when registry proxy configured)
5. Generate `secretspec.toml` composing declarations from all detected modules.
6. Configure default providers: `keyring` for local dev, `env` for CI.
7. Wire into devenv.nix `secretspec` integration block.
8. Document provider options in CLAUDE.md (keyring, dotenv, env, 1Password, LastPass).
9. Note SOPS limitation: SOPS is not yet a supported SecretSpec provider — document this for teams currently using SOPS.

**Acceptance Criteria:**
- [ ] `secretspec.toml` generated with secrets from detected services
- [ ] Auto-generated secrets work for local-only credentials (DB passwords)
- [ ] Provider defaults appropriate for local dev vs CI
- [ ] devenv.nix secretspec integration block populated
- [ ] SOPS limitation documented
- [ ] Secrets never written in plaintext to any generated file

**Research Citations:**
- `research-spikes/devenv-security/boilerplate-research.md` — devenv secret handling
- Validation: SecretSpec 0.7.2 ships with devenv 2.0, declarative generation confirmed
- Web research: SecretSpec providers (keyring, dotenv, env, 1Password, LastPass)

**Status:** Not Started

---

### Unit 12.6: CI Workflow Generation Engine

**Description:** Implement a CI workflow generation engine that produces GitHub Actions and GitLab CI workflow files composing all security scanning steps from detected ecosystems, infrastructure profile, and Phase 12 tools.

**Context:** Phase 5 (Unit 4.4) generated a basic security scanning workflow. Phase 12 adds Semgrep, gitleaks, Grype/Syft/cosign, ScanCode, Claude Code Security Review, and Harden-Runner. These need to compose into a single coherent CI pipeline per project. The generation engine must produce SHA-pinned action references (per GitHub's 2026 security recommendations) and support both GitHub Actions and GitLab CI.

**Desired Outcome:** `gdev init` generates ready-to-use CI workflow files that enforce all configured security policies with zero manual workflow authoring.

**Steps:**
1. Create `internal/cigeneration/` package with `Generator` interface supporting GitHub Actions and GitLab CI.
2. Implement `GitHubActionsGenerator` producing `.github/workflows/security.yml`:
   - Job 1: `harden-runner` (first step, audit mode)
   - Job 2: `lint-and-sast` — Semgrep, ecosystem linters
   - Job 3: `secret-scan` — gitleaks full-repo scan
   - Job 4: `vulnerability-scan` — OSV Scanner, ecosystem audit commands
   - Job 5: `container-security` (conditional on Docker) — Syft SBOM, Grype scan, cosign sign
   - Job 6: `security-review` (conditional on Claude Code) — Claude Code Security Review Action
   - Job 7: `license-compliance` (weekly schedule) — ScanCode scan
   - All action references SHA-pinned
   - OIDC permissions for cosign keyless signing
3. Implement `GitLabCIGenerator` producing `.gitlab-ci.yml` with equivalent stages.
4. Add `--ci-platform github|gitlab|none` flag to wizard and CLI.
5. Each ecosystem module contributes CI commands via existing `CICommands()` method.
6. Infrastructure profile contributes registry proxy configuration for CI.
7. Generate workflow with correct dependency ordering and parallelism.
8. Include `permissions:` block with least-privilege settings per GitHub 2026 security roadmap.

**Acceptance Criteria:**
- [ ] GitHub Actions workflow is valid YAML
- [ ] All action references are SHA-pinned (no mutable tags)
- [ ] Harden-Runner is the first step in every job
- [ ] Workflow permissions use least-privilege
- [ ] Container security jobs conditional on Docker ecosystem detection
- [ ] GitLab CI equivalent produces working pipeline
- [ ] Multi-ecosystem project gets all relevant scanning steps
- [ ] OIDC permissions configured for cosign

**Research Citations:**
- `research-spikes/package-supply-chain-security/org-tooling-research.md` — CI scanning tools
- `artifacts/claude-code-ecosystem-research.md § 1.2` — Claude Code Security Review Action
- Web research: GitHub Actions 2026 security roadmap, SHA-pinned actions, OIDC permissions

**Status:** Not Started

---

### Unit 12.7: MCP Server Curation and Default Configuration

**Description:** Define the default and opt-in MCP server set that gdev configures in `.mcp.json`, with smart defaults based on detected project type and infrastructure profile.

**Context:** Phase 4 (Unit 3.5) generated basic `.mcp.json` with placeholder MCP servers. Phase 11 added semble. This unit curates the complete MCP server set with an opinionated default configuration. The research consensus is 3-6 servers as the sweet spot — more than 10 slows agents without proportional benefit.

**Desired Outcome:** Generated `.mcp.json` includes a curated, project-appropriate set of MCP servers with no manual configuration needed.

**Steps:**
1. Define MCP server tiers:
   - **Default-on (always configured):**
     - `context7` — up-to-date library documentation (Apache-2.0, zero config)
     - `github` — issue/PR workflow integration (already in Phase 4)
   - **Default-on when ecosystem detected:**
     - `socket-dev` — supply chain risk scoring (already in Phase 4, triggered by JS/Python/Rust/Go)
     - `semble` — semantic code search (already in Phase 11, triggered by Python >=3.10)
   - **Opt-in via wizard:**
     - `postgres` — database MCP (triggered by PostgreSQL service)
     - `filesystem` — extended filesystem operations
     - `slack` — team communication (when Slack integration configured)
2. Add Context7 MCP server to default `.mcp.json` generation:
   ```json
   "context7": {
     "command": "npx",
     "args": ["-y", "@upstash/context7-mcp"]
   }
   ```
3. Ensure total configured servers stays <=6 in default path.
4. Add wizard form field for MCP server selection (multi-select with smart defaults).
5. Document each MCP server's purpose in generated CLAUDE.md.
6. Support `--mcp-servers context7,github,socket-dev` flag for non-interactive selection.

**Acceptance Criteria:**
- [ ] Default `.mcp.json` has <=6 servers
- [ ] Context7 configured by default for all projects
- [ ] Socket.dev conditional on supported ecosystem detection
- [ ] Semble conditional on Python >=3.10 (from Phase 11)
- [ ] PostgreSQL MCP conditional on PostgreSQL service
- [ ] Each server documented in CLAUDE.md with purpose
- [ ] Opt-in servers available via wizard customize path

**Research Citations:**
- `research-spikes/claude-code-agent-package-guardrails/mcp-server-research.md` — Socket.dev MCP
- `artifacts/claude-code-ecosystem-research.md § 6` — MCP server landscape
- Web research: Context7 recommended for multi-stack consulting, 3-6 server sweet spot

**Status:** Not Started

---

### Unit 12.8: Enhanced Pre-Commit Hook Suite

**Description:** Extend the pre-commit hook suite from Phase 5 with gitleaks, Semgrep, commitlint, and ecosystem-specific security hooks.

**Context:** Phase 5 (Unit 4.2) defined three hook tiers: baseline (ripsecrets, check-added-large-files, no-commit-to-branch), enhanced (per-language formatters), and specialized (custom lock-file-audit). This unit adds security-focused hooks from Phase 12 tools and a commit message enforcement hook.

**Desired Outcome:** The pre-commit hook suite provides comprehensive local security enforcement without significantly slowing down commits.

**Steps:**
1. Add gitleaks to baseline tier (replaces ripsecrets as primary, ripsecrets retained as backup):
   - `gitleaks protect --staged` — runs on staged changes only
2. Add Semgrep to enhanced tier (per-language):
   - `semgrep --config auto --error` — fast single-file SAST on changed files
3. Add commitlint to baseline tier (opt-in, default-off):
   - Enforces conventional commit format for changelog automation compatibility
   - `commitlint --edit $1` as a `commit-msg` hook
4. Add ecosystem-specific security hooks to specialized tier:
   - Terraform: `tfsec .` on changed `.tf` files
   - Docker: `hadolint` on changed Dockerfiles (already in Phase 2)
   - Nix: `statix check` on changed `.nix` files (already in Phase 7)
5. Update hook tier documentation in generated CLAUDE.md.
6. Ensure total pre-commit time stays <10 seconds for typical commits (gitleaks <1s, Semgrep <3s on changed files, formatters <2s).
7. Support `--hook-tier baseline|enhanced|specialized|full` flag.

**Acceptance Criteria:**
- [ ] Gitleaks pre-commit hook catches staged secrets
- [ ] Semgrep pre-commit hook catches SAST findings in changed files
- [ ] Commitlint is opt-in and off by default
- [ ] Total hook execution time <10 seconds on typical commits
- [ ] Hook tier selection works via flag and wizard
- [ ] All hooks compatible with prek (devenv 1.11+ runner)

**Research Citations:**
- `research-spikes/devenv-security/precommit-hooks-research.md` — original 3-tier design
- Validation: prek replaces pre-commit as default runner
- Web research: gitleaks <1s staged scan, Semgrep fast incremental scanning

**Status:** Not Started

---

### Unit 12.9: Changelog Automation Integration

**Description:** Generate git-cliff configuration for conventional commit changelog generation, with CI workflow integration for automated release notes.

**Context:** This is a developer experience enhancement that standardizes release processes across client projects. git-cliff is a Rust single binary (no Node/Python dependency, consistent with gdev's single-binary philosophy) that generates changelogs from conventional commits with customizable templates.

**Desired Outcome:** Projects using conventional commits get automated changelog generation in CI and a local `gdev changelog` command.

**Steps:**
1. Add `git-cliff` to devenv.nix packages.
2. Generate `cliff.toml` with the firm's standard changelog format:
   - Group by: `feat`, `fix`, `perf`, `refactor`, `docs`, `chore`, `ci`, `test`
   - Include breaking changes section
   - Link commits to GitHub/GitLab issues
   - Configurable via infrastructure profile (different clients may want different formats)
3. Add `gdev changelog` command (thin wrapper around `git-cliff`).
4. Generate CI workflow step for release automation:
   - On tag push: run `git-cliff --latest -o CHANGELOG.md`
   - Optionally create GitHub Release with changelog body
5. Support `--changelog=false` flag to skip.
6. Wire commitlint pre-commit hook (Unit 12.8) as the enforcement layer for conventional commits.

**Acceptance Criteria:**
- [ ] `cliff.toml` generated with firm's standard format
- [ ] `gdev changelog` produces valid CHANGELOG.md
- [ ] CI workflow generates changelog on tag push
- [ ] Configuration is client-customizable via profile
- [ ] Works without conventional commits (falls back to simple commit list)

**Research Citations:**
- Web research: git-cliff 11K stars, 120ms for 10K commits, dual MIT/Apache-2.0

**Status:** Not Started

---

## Dependencies

### From Earlier Phases

| Phase | Dependency | Why Phase 12 Needs It |
|-------|-----------|----------------------|
| **Phase 1** | Ecosystem module interface, generation pipeline, infrastructure profiles | Phase 12 extends the module interface with `SemgrepRuleSets()` and `SecretDeclarations()` methods, and uses the generation pipeline for all file creation |
| **Phase 2** | Tier 1 ecosystem modules | Phase 12 adds Semgrep rule sets, gitleaks allowlists, and Grype/Syft configuration per ecosystem |
| **Phase 3** | devenv.nix generation | Phase 12 adds packages (semgrep, gitleaks, grype, syft, cosign, git-cliff, scancode-toolkit) to devenv.nix and populates SecretSpec integration |
| **Phase 4** | Claude Code addon (.mcp.json, settings.json, CLAUDE.md) | Phase 12 adds MCP servers to .mcp.json, documents tools in CLAUDE.md, and includes Claude Code Security Review in CI |
| **Phase 5** | Pre-commit hook suite, CI scanning configs | Phase 12 replaces/augments hooks (gitleaks over ripsecrets, adds Semgrep) and generates comprehensive CI workflows |
| **Phase 6** | Wizard infrastructure | Phase 12 adds form fields for SAST, secret scanning, container security, license compliance, changelog, MCP servers, and CI platform |
| **Phase 9** | OS detection, package manager abstraction | Phase 12 tools need platform-appropriate installation (gitleaks Go binary, Semgrep Python, ScanCode Python) |
| **Phase 11** | AI agent tooling, semble MCP | Phase 12's MCP curation (Unit 12.7) composes with Phase 11's semble integration |

### Phase 12 Internal Dependencies

Units can proceed in parallel except:
- Unit 12.6 (CI workflow generation) depends on Units 12.1-12.4 (needs all scanning steps defined)
- Unit 12.8 (pre-commit hooks) depends on Units 12.1-12.2 (gitleaks and Semgrep hook definitions)
- Unit 12.7 (MCP curation) is independent of all other units

---

## Tools Explicitly Rejected

### AI Agent Enhancement

| Tool | Reason for Rejection |
|------|---------------------|
| **CodeRabbit** (AI code review) | $19/seat/month commercial SaaS with no self-hosted option. A consulting firm would need per-client licensing. The Claude Code Security Review Action provides AI review at ~$1/PR via API key — better economics and already in the Anthropic ecosystem. |
| **Qodo / PR-Agent** (AI code review) | Open-source version is usable but requires hosting your own LLM API keys and infrastructure. The value proposition doesn't justify the maintenance burden when Claude Code Security Review covers the security review use case natively. |
| **claude-context / Zilliz** (codebase context) | Requires running a Milvus vector database instance — operational overhead too high for a tool that should be zero-config. Semble (Phase 11) and Context7 cover the same need with simpler deployment. |
| **AI test generation tools** | Claude Code already generates tests natively via CLAUDE.md instructions and skills. Adding a separate test generation tool creates tool conflict and confusion about which tool to use. The agent-postmortem-skill (Phase 11) already enforces test verification. Better to invest in CLAUDE.md test conventions than a separate tool. |
| **Aider repo-map** | Aider-specific feature, not extractable as a standalone tool. Claude Code's built-in tool search and file discovery are sufficient. Semble provides the semantic search layer. |

### Developer Experience

| Tool | Reason for Rejection |
|------|---------------------|
| **SOPS + age** (secret management) | Strong tool, but SecretSpec is the devenv-native solution that ships with devenv 2.0 and integrates directly with the devenv environment gdev generates. Adding SOPS would create two competing secret management patterns. SecretSpec doesn't support SOPS as a provider yet, but it will — better to bet on the native integration. |
| **release-please** (Google changelog/release) | Node.js dependency. A Go CLI tool should not require Node.js for changelog generation. git-cliff is a Rust single binary that matches gdev's zero-prerequisite philosophy. |
| **semantic-release** (changelog/release) | Node.js dependency, same reasoning as release-please. Also more opinionated about the release workflow than a consulting firm wants (different clients have different release processes). |
| **Developer onboarding metrics** | No mature open-source tool exists for measuring time-to-first-commit. The concept is valuable but the implementation is organizational process, not tooling — track it via git log analysis and team retrospectives, not a gdev integration. Adding telemetry to a developer tool creates trust issues. |
| **Configuration drift detection** | Nix/devenv already provides reproducible environments by design — if you use `devenv shell`, you get the declared environment. Drift detection is solving a problem that devenv's architecture prevents. The real drift risk is "developer bypasses devenv" which is a process problem, not a tool problem. `gdev doctor` (Phase 9) already validates the local environment state. |
| **Local service orchestration** (Docker Compose alternatives) | devenv 2.0 includes a native Rust process manager replacing process-compose, with dependency ordering, restart policies, and Linux capabilities. Adding a separate orchestration tool would conflict with devenv's built-in service management. |

### Security Hardening

| Tool | Reason for Rejection |
|------|---------------------|
| **CodeQL** (GitHub SAST) | Free only for public repos; requires GitHub Advanced Security ($49/user/month) for private repos. A consulting firm with hundreds of engineers across private client repos would face massive licensing costs. Semgrep CE provides comparable SAST coverage at $0. |
| **Snyk** (vulnerability scanning) | Commercial SaaS with per-developer pricing. OSV Scanner + Socket.dev + Grype cover the same surface at $0 for a consulting firm. Snyk's developer experience is better, but not $49/user/month better when you're generating all the configuration automatically via gdev. |
| **Trivy** (container/IaC scanning) | March 2026 supply chain compromise: malicious releases published, GitHub Actions tags force-pushed to malware, Docker images poisoned, vulnerability database updates suspended. While the scanning engine itself wasn't compromised, the distribution infrastructure was — and for a security tool, that is disqualifying. Grype is the replacement. Reference Trivy only in generated security documentation as a trust model lesson. |
| **FOSSA** (license compliance) | Commercial SaaS ($0 for <=5 contributors, paid beyond). ScanCode Toolkit provides equivalent license detection at $0 for any team size, with Apache-2.0 licensing. FOSSA's dashboard is nicer, but gdev generates reports directly. |
| **SLSA framework compliance tooling** | SLSA L3 (the practical target) is achievable with the tools already selected: cosign for signing, Syft for SBOM, GitHub Actions OIDC for build provenance. A dedicated "SLSA compliance tool" adds complexity without capability — the individual tools already produce SLSA-compliant artifacts when configured correctly. Document SLSA compliance in generated security docs rather than adding another tool. |
| **Trivy for IaC scanning** | Same supply chain compromise concern. For IaC scanning specifically, Semgrep's `p/terraform` and `p/dockerfile` rule sets plus tfsec (still maintained, pre-compromise Trivy fork for IaC) cover the same surface. |

### Infrastructure

| Tool | Reason for Rejection |
|------|---------------------|
| **SonarQube plugin** (Claude Code marketplace) | Requires a SonarQube server instance — operational overhead too high for the consulting firm's $0/mo default stack. Listed in Phase 4 as "configure when available" which is the right posture — reference but don't default-enable. |
| **Aikido plugin** (Claude Code marketplace) | Commercial SaaS ($24/user/month). Would need per-client licensing. The firm's security stack (Semgrep + gitleaks + Grype + OSV Scanner) covers the same surface at $0. |
| **Monitoring/observability for dev environment** | Developer environments don't need monitoring — they need to work. If they don't work, `gdev doctor` diagnoses the issue. Adding Prometheus/Grafana for dev environments is over-engineering a problem that doesn't exist at the consulting firm scale. |
| **Endor Labs plugin** | Commercial SaaS with limited free tier. OSV Scanner + Socket.dev MCP cover supply chain scanning at $0. |

---

## Implementation Notes

### Wizard Integration

Phase 12 adds the following wizard form groups (or extends existing groups):

1. **Security Scanning** (new group, shown in customize path):
   - SAST: Semgrep on/off (default: on)
   - Secret scanning: Gitleaks on/off (default: on)
   - Container security: Grype+Syft+Cosign on/off (default: on when Docker detected)
   - License compliance: ScanCode on/off (default: off — requires explicit opt-in due to scan duration)

2. **CI Platform** (extends existing infrastructure questions):
   - Platform: GitHub Actions / GitLab CI / None (default: GitHub Actions)
   - Claude Code Security Review: on/off (default: on when Claude Code enabled)

3. **Developer Experience** (extends existing environment questions):
   - SecretSpec: on/off (default: on when services detected)
   - Changelog automation: on/off (default: off — opt-in)
   - Conventional commits enforcement: on/off (default: off — opt-in)

### Total Tool Count

Phase 12 adds 9 tools/capabilities to the gdev ecosystem:
- 4 security (Semgrep, Gitleaks, Grype+Syft, ScanCode) + cosign
- 2 developer experience (SecretSpec, git-cliff)
- 1 AI enhancement (Context7 MCP + Claude Code Security Review Action)
- 1 infrastructure (CI workflow generation engine + Harden-Runner formalization)

This is a deliberate constraint. Each tool was selected because it fills a gap no existing Phase 1-11 tool covers, works across all 27 ecosystems (or is appropriately scoped), and operates at $0/month for the consulting firm's default profile.

### Binary Size Impact

Adding tools to devenv.nix has zero impact on gdev binary size — they're Nix packages resolved at `devenv shell` time. The gdev binary grows only from embedded templates and configurations, which are text files measured in kilobytes.
