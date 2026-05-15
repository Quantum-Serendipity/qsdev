# npm-scan (lateos-ai/npm-scan) — Deep Dive Research

## Executive Summary

npm-scan is a young, solo-maintained npm supply chain scanner with an ambitious feature surface but shallow detection depth. While its README claims superiority over npm audit, Snyk, and Socket across 16+ capability dimensions, source code analysis reveals that most detectors are single-regex pattern matchers operating on concatenated file contents — a fundamentally different (and weaker) approach than the AST-level and behavioral analysis advertised. The project was built from scratch to v0.9.7 in 4 days (May 9-12, 2026) by a single maintainer, has 4 GitHub stars, 0 forks, and ~4,253 monthly npm downloads. It is not suitable as a gdev default or configuration option, but its ATK taxonomy structure and policy-as-code patterns offer modest concept inspiration.

---

## 1. Architecture & Mechanisms

### 1.1 How It Works

npm-scan operates as a Node.js CLI (Commander.js) with three core commands:

1. **`scan <package>`** — Fetches a tarball from the npm registry (or reads a local `.tgz`), extracts it to a temp directory, reads all `.js` files, then runs 11 detector functions serially against the package.json and file contents.

2. **`scan-lockfile`** — Parses `package-lock.json`, `yarn.lock`, or `pnpm-lock.yaml`, extracts package metadata (name, version, integrity, dependencies), runs typosquatting checks and dependency graph analysis.

3. **`report`** — Generates output in various formats (JSON, HTML, text, SARIF, CSV, NIST, CRA) from stored scan results in a local SQLite database (sql.js/WASM).

### 1.2 Scan Pipeline (Single Package)

```
npm registry → fetch tarball → gunzip + tar extract → read all .js files
    → for each ATK detector (001-011):
        detector.scan(pkgJson, files[]) → findings[]
    → aggregate findings → apply policy → calculate risk score → store in SQLite
```

### 1.3 Detector Architecture

Each detector is a standalone ES module exporting `async function scan(pkgJson, files)`. The `detectors/index.js` orchestrator calls all 11 sequentially via `runAll()`.

**Critical finding: Most detectors operate on concatenated source code via `files.map(f => f.content).join('\n')` — a single string tested against regex patterns.** This is NOT AST-level analysis despite the README's claims.

Only ATK-002 (obfuscation) shows genuine sophistication:
- Multiple detection layers (eval+decode, double-encoding, network+decode, charcode, shell patterns)
- Context awareness (dist/build vs. test vs. lifecycle script)
- Evidence objects with line numbers, decoded previews, encoding types
- Known-safe domain allowlisting

The remaining detectors are shallow:
- **ATK-001** (lifecycle): Single regex on script values (`/curl|wget|sh |bash/`)
- **ATK-003** (credentials): Single regex (`/process.env.(NPM_TOKEN|GIT_TOKEN|AWS_SECRET)/)`)
- **ATK-004** (persistence): Single regex (`/mkdir.*(\.vscode|\.claude|\.cursor)/`)
- **ATK-005** (exfiltration): Single regex on lowercased code (`/curl.*(-d|--data)|pastebin|dns\.resolve/`)
- **ATK-009** (dormant triggers): CI env checks + time-based patterns, with some severity escalation based on co-occurring suspicious patterns
- **ATK-010** (sandbox evasion): Pattern list for debugger/hostname probes + system fingerprinting API counting
- **ATK-007** (typosquatting in lockfile): Hardcoded regex list for 6 popular package name variants — NOT edit-distance matching despite the README claim

### 1.4 Lockfile Analysis

The lockfile scanner parses npm, yarn, and pnpm lockfile formats into a unified package structure. Lockfile-specific checks include:
- **ATK-007**: Hardcoded typosquatting patterns (lodash/axios/react/express/vue/webpack variants)
- **ATK-011**: Flags peer dependencies containing "plugin"/"hook"/"ext" in their name, packages with >5 dependencies where >3 are scoped, and packages with >10 optional dependencies

The ATK-011 worm propagation detection is particularly weak — flagging any peer dependency with "plugin" in its name would produce massive false positives on real projects (eslint-plugin-*, babel-plugin-*, webpack-plugin-*).

### 1.5 Policy Engine

The policy-as-code engine (`backend/policy.js`) is the most well-engineered component:
- YAML/JSON policy loading with input validation
- Package allowlists
- Context-aware suppress rules (file path, dist/build, test, lifecycle hook, domain, reputation tier)
- Severity overrides per ATK ID
- `fail_on` threshold enforcement
- Safety guards: lifecycle hook and multi-layer findings cannot be suppressed
- Known reputable packages list for tier-based filtering

### 1.6 Output & Reporting

Comprehensive output format support including JSON, HTML, text, CycloneDX SBOM (1.5), SPDX (2.3), SARIF v2.1, CSV, NIST 800-161 compliance matrix, and EU CRA mapping. Premium features (license key gated): PDF reports, SIEM exports (CEF, ECS, Sentinel, QRadar).

---

## 2. Problem It Solves

npm-scan targets the gap between traditional CVE-based scanners (npm audit, Snyk) and behavioral analysis tools (Socket) — specifically, the detection of:

1. Malicious install scripts that download/execute payloads
2. Obfuscated code hiding credential theft or exfiltration
3. Packages that probe for CI/production environments before activating
4. Typosquatting and dependency confusion
5. Packages that tamper with editor/IDE configuration directories

This is a real and growing problem space. The 2025-2026 wave of npm supply chain attacks (chalk/debug compromise, Lazarus Group campaigns, Shai-Hulud worm) validated the need for deeper-than-CVE scanning.

---

## 3. Maturity, Maintenance & Community

### 3.1 Development Timeline (RED FLAG)

The entire project from v0.1.0 to v0.9.7 was developed in **4 days** (May 9-12, 2026):
- May 9: Initial foundation (v0.1.0)
- May 10: 8 releases in one day (v0.2.0 through v0.6.0) — added all detectors, SIEM, CRA, license enforcement, PostgreSQL, FastAPI, Helm chart, webhooks
- May 11: 4 releases (v0.9.0-0.9.2) — replaced deps, added tests
- May 12: 3 releases (v0.9.5-0.9.7) — README fixes, Sigstore

This development velocity strongly suggests AI-assisted code generation. No human writes 11 detectors, 4 SIEM exporters, a PostgreSQL schema, a FastAPI REST API, a Helm chart, a policy engine, SBOM generators, a license key system, Docker configs, and 324 tests in 4 days.

### 3.2 Adoption Metrics

| Metric | Value | Assessment |
|--------|-------|------------|
| GitHub stars | 4 | Negligible |
| GitHub forks | 0 | No community |
| npm downloads (30 days) | ~4,253 | Very low |
| Contributors | 1 (solo maintainer) | Bus factor = 1 |
| Issues/PRs | Not observed | No community engagement |
| Age | 6 days old (as of 2026-05-15) | Extremely immature |

### 3.3 Maintainer

Roongrunchai Chongolnee — claims CISSP, CEH, Cisco Security, AWS Cloud Practitioner certifications with infrastructure/security background at Philips. Single maintainer with no apparent team or organizational backing beyond "Lateos AI" branding.

### 3.4 Version Discrepancy

package.json reports version 0.15.1, GitHub release tag says v1.0.0, and CHANGELOG goes up to 0.9.7. This inconsistency suggests rapid, poorly coordinated releases.

---

## 4. Comparison to Alternatives

### 4.1 npm audit

**What it does**: Checks installed packages against npm's advisory database (known CVEs).

**Strengths over npm-scan**: Built into npm (zero install), massive CVE database maintained by npm/GitHub, mature and battle-tested, zero false positives on CVE matches.

**Weaknesses**: Cannot detect novel malware, no behavioral analysis, no typosquatting detection, no obfuscation detection.

**Assessment**: npm audit and npm-scan serve different purposes. npm audit is production-grade for CVEs; npm-scan attempts to cover novel threats but with immature detection.

### 4.2 Socket.dev (@socket/cli)

**What it does**: Pre-install behavioral analysis of npm packages. Analyzes what packages actually do at the code level. 70+ risk types with AI analysis.

**Strengths over npm-scan**:
- Backed by a funded company with a dedicated security research team
- 70+ risk types vs. npm-scan's 11
- Reachability analysis distinguishing "present" from "executes in my project"
- Integrated directly into npm as an alias (`socket npm`)
- Free tier with deep scanning, massive adoption
- Detects maintainer permission changes, artifact/source mismatches
- Built into npm registry pages natively

**Weaknesses**: Cloud-dependent (sends package data to Socket servers), not fully local.

**Assessment**: Socket is the established market leader for pre-install behavioral analysis. npm-scan's claim of superiority ("catches what Socket misses") is unsupported by its actual detection mechanisms. Socket has deeper detection, wider coverage, and vastly larger adoption.

### 4.3 Snyk

**What it does**: CVE-based vulnerability scanning with continuous monitoring, fix PRs, and container/IaC scanning.

**Strengths over npm-scan**: Massive vulnerability database, automated fix PRs, continuous monitoring, exploitability scoring, enterprise-grade with SLA support.

**Weaknesses**: Reactive (post-disclosure only), requires account/API key, commercial pricing for teams.

**Assessment**: Snyk and npm-scan target different threat models. Snyk is for known vulnerabilities; npm-scan attempts novel threat detection. They are complementary, not competing.

### 4.4 Recommended Layered Approach (Industry Consensus)

The industry consensus for npm supply chain security is a three-layer defense:
1. **Pre-install gate**: Socket CLI (behavioral analysis)
2. **Continuous CVE monitoring**: Snyk or npm audit
3. **CI pipeline gate**: npm audit with `--audit-level=high`

npm-scan would slot into layer 1 but cannot compete with Socket's depth, team, and adoption.

---

## 5. Integration Assessment for gdev

### 5.1 As a Default Tool (REJECT)

**Verdict: Not recommended.**

Reasons:
- 6 days old, 4 stars, solo maintainer — extreme immaturity
- Detection depth is primarily regex pattern matching, not the AST/behavioral analysis claimed
- High false positive risk (ATK-011 flags any peer dep with "plugin" in the name)
- Apache-2.0 + Commons Clause license is semi-restrictive (prohibits selling as a service)
- Version inconsistencies suggest unstable release process
- Zero community validation (0 forks, no issues/PRs from external users)
- gdev already has 6-layer defense-in-depth; adding an unproven scanner as default undermines the "security by default" principle

### 5.2 As a Configuration Option (REJECT)

**Verdict: Not recommended at this time.**

Reasons:
- Including it as an opt-in option implies endorsement
- Users who enable it may develop a false sense of security given the gap between claimed and actual detection capability
- The tool could mature to recommendation-worthy status in 6-12 months, but it's too early now
- Socket CLI is a better recommendation for the same slot (pre-install behavioral scanning)

### 5.3 As Concept/Implementation Inspiration (PARTIAL ACCEPT)

**Verdict: Selectively useful patterns worth borrowing.**

**Borrow-worthy patterns:**

1. **ATK taxonomy structure** — The numbered attack classification (ATK-001 through ATK-011) with NIST 800-161 mappings is a clean organizational pattern. gdev could adopt a similar threat taxonomy for its 6-layer defense documentation, mapping each layer's defenses to specific attack classes.

2. **Policy-as-code engine design** — The YAML policy format with allowlists, severity overrides, context-aware suppressions, and fail-on thresholds is well-designed. gdev's per-ecosystem configuration could adopt similar patterns for security policy customization (especially the unsuppressible safety guards for lifecycle hooks).

3. **Lockfile-triggered scanning** — The pre-commit hook pattern (scan lockfile changes on every commit) is a solid workflow integration point. gdev could generate pre-commit hook configurations that trigger lockfile scanning via the user's preferred scanner (Socket, npm audit, etc.) rather than bundling a specific scanner.

4. **SARIF output for GitHub integration** — SARIF v2.1 output enabling findings in GitHub's Security tab is a valuable integration pattern for any security tooling gdev configures.

**Not worth borrowing:**

- The regex-based detection approach (too shallow for real-world efficacy)
- The ATK-002 obfuscation detector (sophisticated but better served by Socket's deeper analysis)
- The hardcoded typosquatting patterns (should be edit-distance based, not regex lists)
- The enterprise tier architecture (SIEM, SSO, Helm, PostgreSQL) — overengineered for a tool this immature

---

## 6. Tradeoffs, Limitations & Failure Modes

### 6.1 Fundamental Limitations

1. **Regex-based detection on concatenated source** — Most detectors join all file contents into a single string and run one regex. This misses multi-file attack patterns, context-dependent triggers, and any obfuscation beyond the specific patterns hardcoded.

2. **No actual AST analysis** — Despite the README claiming "AST-level heuristic analysis," only ATK-002 does anything beyond regex. The acorn dependency exists in package.json but is not used by any detector in the examined source.

3. **No runtime/behavioral sandbox** — The README claims behavioral detection, but all detectors are static regex matches. The "dynamic sandbox (gVisor-based)" is listed as premium/unreleased.

4. **Hardcoded typosquatting list** — ATK-007 checks 6 package families with hardcoded regex, not edit-distance comparison. Novel typosquats on packages not in the list are missed entirely.

5. **False positive surface** — ATK-009 flags any package checking `process.env.CI` (common in many legitimate packages like jest, semantic-release, etc.). ATK-011 flags any peer dependency with "plugin" in the name (almost every eslint/babel/webpack plugin).

### 6.2 Security Concerns

1. **npm-scan itself fetches and extracts arbitrary tarballs** — The tool downloads packages from npm and extracts them to temp directories. If the extraction or file reading has vulnerabilities, the scanner itself becomes an attack vector.

2. **No sandboxing of scanned code** — Files are read as strings, not executed, but the pattern-matching approach means a sufficiently obfuscated malicious package would pass all checks.

3. **Commons Clause restricts redistribution as a service** — If gdev were ever offered as a hosted service, bundling npm-scan would create licensing complications.

### 6.3 Reliability Concerns

1. **Solo maintainer with no organizational backing** — Bus factor of 1.
2. **4-day development timeline** — Strongly suggests AI-generated code without deep security review of the detectors themselves.
3. **Version numbering chaos** — 0.15.1 in package.json, v1.0.0 on GitHub release, 0.9.7 in CHANGELOG.
4. **Stub scripts** — `lint` is `echo 'Lint stub'`, `build` is `echo 'Build stub'`. No actual linting or build process.

---

## 7. Conclusions

npm-scan is an interesting proof-of-concept that demonstrates how a supply chain scanner could be structured, but it is not production-ready and its claims significantly exceed its actual capabilities. The gap between marketing (README) and implementation (source code) is substantial.

**For gdev specifically:**
- **Do not integrate** as a default or configuration option
- **Do borrow** the ATK taxonomy structure, policy-as-code YAML format, lockfile-triggered scanning pattern, and SARIF integration concept
- **Do recommend Socket CLI** as the pre-install behavioral scanning tool for gdev's npm ecosystem configuration, as it provides genuine depth, established adoption, and active maintenance
- **Re-evaluate in 6-12 months** if npm-scan gains community traction, improves detection depth, and demonstrates sustained maintenance

---

## Sources

All raw source material saved to `docs/`:
- `npm-scan-readme.md` — Full README from GitHub
- `npm-scan-github-repo-page.md` — Repository metadata (stars, forks, languages)
- `npm-scan-package-json.md` — package.json with dependency analysis
- `npm-scan-detectors-source.md` — Core detector source code (index, ATK-001, 002, 003, 005, 009)
- `npm-scan-additional-detectors.md` — ATK-004, ATK-010 source code
- `npm-scan-lockfile-js.md` — Lockfile parser and lockfile-level detectors
- `npm-scan-fetch-js.md` — Package fetching and tarball extraction
- `npm-scan-policy-js.md` — Policy-as-code engine
- `npm-scan-cli-js.md` — CLI command structure
- `npm-scan-backend-index-js.md` — Library entry point (stub)
- `npm-scan-changelog.md` — Full changelog with timeline analysis
- `npm-scan-licensing.md` — License model (Apache-2.0 + Commons Clause)
- `npm-scan-npm-downloads.md` — npm download statistics
- `snyk-vs-socket-npm-comparison-2026.md` — Industry comparison of npm security tools
