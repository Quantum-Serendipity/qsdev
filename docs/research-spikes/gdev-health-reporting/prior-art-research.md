# Prior Art: DevSecOps Health & Compliance Reporting Tools

## Overview

Survey of how existing DevSecOps tools present health, compliance, and vulnerability information. These tools establish the patterns that gdev's reporting should build on -- both to meet developer expectations and to interoperate with the broader ecosystem.

## Tool-by-Tool Analysis

### OpenSSF Scorecard

**What it does:** Automated security posture assessment for open-source projects. Runs 23 checks against a repository and produces a weighted aggregate score (0-10).

**Output formats:** Default text, JSON. Scorecard v6 (2026 roadmap) adds OSPS Baseline conformance output, enriched JSON, in-toto attestations, Gemara SDK, and OSCAL Assessment Results.

**Scoring model:**
- 23 individual checks, each scored 0-10
- Risk-weighted aggregation: Critical (10x), High (7.5x), Medium (5x), Low (2.5x)
- Checks cover: binary artifacts, branch protection, CI tests, CII best practices, code review, contributors, dangerous workflows, dependency updates, fuzzing, license, maintained, pinned dependencies, packaging, SAST, security policy, signed releases, token permissions, vulnerabilities, webhooks

**UX patterns:**
- Single aggregate score as headline metric
- Per-check breakdown with detailed failure reasons (`--show-details`)
- Badge generation via `api.scorecard.dev/projects/github.com/{owner}/{repo}/badge`
- Scorecard v6 shifts from pure numeric scoring to dual-track: numeric scores AND conformance labels (PASS/FAIL/UNKNOWN/NOT_OBSERVABLE)

**Multi-repo aggregation:** `scorecard-monitor` GitHub Action tracks scores across organizations. Produces markdown reports + JSON database with historical tracking. Generates GitHub issues when scores drop. Used by Node.js Security WG, NodeSecure.

**Key insight for gdev:** The weighted scoring model is directly applicable -- gdev's defense layers and tools each have different risk profiles. The dual-track model (numeric + conformance) aligns with gdev's need to show both "how much is enabled" and "is it correctly configured."

### npm audit

**What it does:** Scans package-lock.json for known vulnerabilities in npm dependencies.

**Output formats:** Human-readable text (default), JSON (`--json`). npm-audit-report adds SARIF 2.1.0, markdown, GitHub Actions workflow commands. npm-audit-sarif converts to SARIF for SonarQube.

**Severity model:** Five levels: info, low, moderate, high, critical. `--audit-level` flag sets the threshold for non-zero exit codes.

**UX patterns:**
- Summary table at top: total vulnerabilities by severity
- Per-vulnerability detail: advisory ID, module name, affected versions, path, fix available
- `npm audit fix` for auto-remediation
- Bulk advisory endpoint (fast) with fallback to full tree analysis
- Workspace support for monorepos (`--workspaces`)

**Key insight for gdev:** The `--audit-level` flag pattern is essential. Consulting projects might tolerate `low` findings but want CI to fail on `high`. gdev should support a similar threshold. The summary-then-detail pattern (counts at top, details below) is the standard expectation.

### cargo audit

**What it does:** Audits Cargo.lock for crates with known security vulnerabilities via RustSec advisory database.

**Output formats:** Human-readable text (default), JSON (`--json` flag). JSON structure includes database info, dependency counts, vulnerability-found boolean, and vulnerability array with RUSTSEC IDs, crate names, versions, dates, URLs, titles, and solutions.

**UX patterns:**
- Advisory-centric display: each finding shows RUSTSEC ID, title, crate, version, date, URL
- Dependency tree display showing how vulnerable crate enters the project
- Solution field with recommended fix
- Binary auditing via `cargo-auditable` for production binaries

**Notable:** JSON output format has been unstable across versions and caused breakage in external tools. There's an open issue to stabilize it. The RustSec database exports to OSV format.

**Key insight for gdev:** The unstable JSON format is a cautionary tale. gdev should version its JSON output schema from day one. The dependency-tree visualization showing "how did this vulnerable thing get here" is valuable for understanding transitive risk.

### Python Safety CLI

**What it does:** Scans Python dependencies for known security vulnerabilities.

**Output formats:** screen (default), text (no formatting), JSON (requires API key), bare (simplified JSON), HTML5.

**JSON structure:**
- `report_meta`: timestamps, scan targets, package/vulnerability counts
- `scanned_packages`: packages found during scan
- `affected_packages`: packages with vulnerabilities
- `vulnerabilities`: vulnerability details
- `ignored_vulnerabilities`: excluded by policy
- `remediations`: fix recommendations
- `announcements`: Safety team messages

**UX patterns:**
- Screen output is color-coded terminal display
- JSON requires paid API key (barrier to automation)
- Safety CLI 3.x changed JSON format substantially from 2.x (breaking change)

**Key insight for gdev:** The `remediations` field is a strong pattern -- not just reporting problems but suggesting fixes. The `ignored_vulnerabilities` field with policy support is essential for managing acceptable risk. However, paywalling JSON output is an anti-pattern gdev should avoid.

### govulncheck

**What it does:** Reports known Go vulnerabilities using static analysis of source code or binary symbol tables.

**Output formats:** text (default), JSON (streaming), SARIF, OpenVEX.

**JSON streaming format:** Emits a series of Message objects as analysis proceeds. Config message first (protocol version, scanner, database), then findings incrementally. Same vulnerability can appear multiple times as analysis deepens (module required -> imported -> called).

**Exit codes:** Non-zero only in text mode when vulnerabilities found. JSON/SARIF/OpenVEX always exit 0.

**UX patterns:**
- Progressive disclosure in JSON: findings emitted at increasing specificity
- Call-stack traces (`-show traces`) showing exact path to vulnerable function
- Source mode distinguishes "called" vs "imported" vs "just in dependency tree"
- Binary mode for production artifact scanning

**Key insight for gdev:** The progressive refinement model (required -> imported -> called) is powerful for prioritization. gdev could similarly distinguish "available but disabled" -> "enabled but misconfigured" -> "enabled and working." The SARIF+OpenVEX dual output is the emerging standard.

### OWASP Dependency-Check

**What it does:** Software composition analysis detecting publicly disclosed vulnerabilities in application dependencies.

**Output formats:** HTML, XML, CSV, JSON, JUNIT, SARIF, JENKINS, GITLAB, ALL. Multiple formats can be comma-separated in a single run.

**UX patterns:**
- HTML report as primary human-readable output
- SARIF for GitHub Code Scanning integration
- JUnit format for test framework integration (vulnerabilities as test failures)
- GitLab format for native CI dashboard integration
- Dependency-Track can ingest XML/SARIF for central dashboard

**Key insight for gdev:** The JUnit format trick is clever -- presenting security findings as test failures integrates with existing CI pass/fail infrastructure. The `ALL` format option is convenient for initial setup where you don't know which consumer needs what. The GitLab-specific format shows the value of targeting specific CI platforms.

## Cross-Tool Pattern Analysis

### Universal Patterns (implement these)

| Pattern | Used By | Relevance to gdev |
|---------|---------|-------------------|
| JSON machine-readable output | All 6 tools | CI integration, dashboards, aggregation |
| Severity levels (3-5 tiers) | npm, Safety, OWASP DC | Threshold-based CI pass/fail |
| Exit code based on findings | All 6 tools | CI gate integration |
| Summary-then-detail display | npm, cargo, Scorecard | Terminal UX for quick assessment |
| Per-finding remediation hints | npm, Safety, cargo | Actionable output, not just problem reports |
| SARIF output | govulncheck, OWASP DC, npm (via converter) | GitHub Code Scanning, IDE integration |

### Strong Patterns (consider adopting)

| Pattern | Used By | Notes |
|---------|---------|-------|
| Weighted scoring | Scorecard | Risk-proportional aggregation |
| Badge generation | Scorecard, Snyk | Visual posture summary for READMEs |
| Multi-repo tracking | scorecard-monitor | Organizational overview with trend tracking |
| Streaming JSON | govulncheck | Progress indication for large projects |
| JUnit output | OWASP DC | Leverage existing CI test infrastructure |
| Multiple simultaneous formats | OWASP DC | `--format ALL` convenience |
| Dual-track evaluation | Scorecard v6 | Numeric score AND conformance PASS/FAIL |

### Anti-Patterns (avoid these)

| Anti-Pattern | Tool | Why |
|--------------|------|-----|
| Paywalled JSON output | Safety CLI | Blocks CI automation for free tier |
| Unstable JSON schema | cargo audit | Breaks downstream tooling |
| Machine-readable always exits 0 | govulncheck | Forces wrapper scripts for CI gates |
| No versioned schema | Most tools | No way to detect breaking changes |

## Doctor Command Pattern

Several developer tools use a `doctor` or `check` subcommand for system health validation:

- **`flutter doctor`**: Sections per component (Flutter SDK, Android toolchain, iOS, IDE). Checkmark/X per section with sub-items. Color-coded (green pass, red fail, yellow warning). Suggests specific remediation commands.
- **`brew doctor`**: Warns about potential problems. Lists issues with explanations.
- **`rustup check`**: Shows installed vs available versions per component.
- **React Native CLI `doctor`**: Verifies dependencies, suggests fixes.

**Common UX elements:**
- Hierarchical check structure (categories -> individual checks)
- Three-state indicators: pass/warn/fail (green/yellow/red)
- Inline remediation: "Run X to fix"
- Machine-readable alternative (JSON) alongside human-readable default

## Output Format Ecosystem

### SARIF (Static Analysis Results Interchange Format)
- OASIS standard, JSON-based
- Structure: sarifLog -> runs[] -> results[]
- Result fields: ruleId, level (error/warning/note/none), message, locations, codeFlows
- GitHub Code Scanning natively consumes SARIF
- Supported by: govulncheck, OWASP DC, Semgrep, gosec, and many others
- The standard format for IDE and CI integration

### OpenVEX (Vulnerability EXchange)
- Newer format for vulnerability status communication
- Supported by govulncheck
- Growing adoption for supply chain transparency

### OSCAL (Open Security Controls Assessment Language)
- NIST standard for security assessment documentation
- Scorecard v6 adding support
- Most relevant for compliance-heavy environments

### In-toto
- Supply chain integrity framework
- Scorecard v6 adding attestation support
- Relevant for provable security posture

## Depth Checklist

- [x] Underlying mechanism explained: Scoring algorithms, output formats, CI integration patterns
- [x] Key tradeoffs and limitations identified: Paywalled features, unstable schemas, exit code inconsistencies
- [x] Compared to at least one alternative: 6 tools compared head-to-head
- [x] Failure modes and edge cases: Schema instability, paywall barriers, exit code gotchas
- [x] Concrete examples or reference implementations: Scorecard-monitor, npm-audit-report, shields.io endpoint badges
- [x] Report is standalone-readable: Full pattern analysis with actionable recommendations
