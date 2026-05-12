# Publication Age Quarantine Gates & Package Hold Strategies

## Executive Summary

Age-gating — delaying consumption of newly published package versions by a configurable period — has emerged as one of the highest-impact, lowest-cost defenses against software supply chain attacks. The mechanism works because most malicious packages are detected and removed within hours to days of publication: PyPI handles 66% of malware reports within 4 hours and 92% within 24 hours. A hold period of even 3-7 days eliminates the attacker's time advantage for the vast majority of incidents. As of mid-2026, age-gating has moved from fringe idea to ecosystem default, with native support in npm, pnpm, Yarn, Bun, Deno, pip, uv, and Cargo, plus configurable delays in Renovate, Dependabot, and Snyk.

---

## 1. Why Age-Gating Works: The Theory

### The Attacker's Window

Language package managers distribute new versions instantly — `npm publish` or `pip upload` makes a package installable worldwide in seconds. If a dependency bot (Dependabot, Renovate) or a developer runs `npm install` during the window between publication and detection, malicious code enters projects without human review.

The core insight is that most supply chain attacks are **short-lived**: attackers rely on speed of propagation, not sophistication of concealment. The malicious version needs to exist just long enough for automated tooling to pull it in.

### Detection Time Statistics

| Source | Metric | Value |
|--------|--------|-------|
| PyPI 2025 Year in Review | Malware reports handled within 4 hours | 66% |
| PyPI 2025 Year in Review | Malware reports handled within 24 hours | 92% |
| PyPI 2025 Year in Review | Maximum remediation time for complex cases | 4 days |
| William Woodruff analysis | Attacks with opportunity window under 1 week | 8 of 10 |
| Nx compromise (Aug 2025) | Time to detection and removal | 4-5 hours |
| Axios compromise (Mar 2026) | Time to detection | 39 minutes |
| chalk/debug compromise (Sep 2025) | Time from publish to community alert | ~2 hours |
| TanStack compromise | Time to public detection | 20 minutes |
| LiteLLM/Xinference (2026) | Time to removal | Hours |

### The Math

A 3-day cooldown would have blocked the vast majority of known supply chain attacks. A 7-day cooldown would have blocked 8 out of 10 attacks in Woodruff's analysis. A 14-day cooldown provides extremely high coverage with diminishing marginal returns beyond that point.

The tradeoff is not detection probability (which is very high within days) but rather the operational cost of delayed access to legitimate updates.

### What Age-Gating Protects Against

**Effective for:**
- Compromised maintainer accounts publishing short-lived malicious versions
- Automated malware injection campaigns
- Account takeover attacks (the most common vector)
- Dependency confusion/substitution with newly registered packages

**Ineffective for:**
- Typosquatting attacks (the malicious package may be old)
- Long-term maintainer compromise where malicious code is introduced gradually
- Attacks that rewrite existing version tags (e.g., the Trivy compromise where attackers redirected 76 of 77 version tags to malicious commits)
- Zero-day vulnerabilities that require immediate patching

---

## 2. Package Manager Native Support

### JavaScript Ecosystem (Most Mature)

The JavaScript ecosystem achieved near-complete age-gating coverage within approximately 6 months (September 2025 - February 2026):

**pnpm** (v10.16, Sep 2025; default in v11)
```yaml
# pnpm-workspace.yaml
minimumReleaseAge: 1440          # minutes; 1440 = 1 day (v11 default)
minimumReleaseAgeExclude:
  - '@your-org/*'                # exempt internal packages
```
- Unit: minutes
- Default in v11: 1440 (1 day)
- Exclusion support: yes, via `minimumReleaseAgeExclude`
- Also includes `blockExoticSubdeps` (blocks git/tarball transitive deps) and `trustPolicy: no-downgrade`

**npm** (v11.10.0, Feb 2026)
```ini
# .npmrc
min-release-age=3                # days
```
- Unit: days
- No default yet (proposal to make 7 days the default is open)
- Exclusion support: **not yet** — no way to exempt internal packages (open GitHub issue)
- Limitation: creating operational friction for mixed internal/external dependency portfolios

**Yarn** (v4.10.0, Sep 2025)
```yaml
# .yarnrc.yml
npmMinimalAgeGate: 7d
```

**Bun** (v1.3, Oct 2025)
```toml
# bunfig.toml
minimumReleaseAge = "7d"
```
- Supports per-package exclusions

**Deno**
```
--minimum-dependency-age=7d
```

### Python Ecosystem

**pip** (v26.0, Jan 2026)
```ini
# pip.conf
uploaded-prior-to = <absolute timestamp>
```
- Supports both absolute timestamps and relative durations
- Absolute timestamps enable reproducible builds across time

**uv** (v0.9.17, Dec 2025)
```
--exclude-newer=7d
```
- Native age gate with per-package exceptions
- Supports relative durations
- The `set-minimum-package-release-age` tool includes a daily cron refresh for absolute-timestamp-based uv configurations

### Ruby Ecosystem

**gem.coop** (community registry, beta)
- Registry-level 48-hour delay — newly published gem versions are hidden from Bundler resolution entirely until aged past threshold
- Served from separate endpoint: `beta.gem.coop/cooldown`
- Escape hatch: pull urgent security patches from primary gem.coop source
- Approach: implements cooldowns at the index/registry level rather than the client, sidestepping the need for Bundler changes
- Bundler itself has no native cooldown support yet

### Rust Ecosystem

**Cargo** (v1.94, Mar 2026)
- Registry-side infrastructure stabilized for age-gating support

### Ecosystems Without Native Support (as of May 2026)

Go, Bundler (direct), Composer (PHP), Maven/Gradle (Java), Swift Package Manager, Dart pub, Elixir Hex

### Configuration Fragmentation

A notable pain point: ten different naming conventions across tools for essentially the same feature:

| Tool | Setting Name | Unit |
|------|-------------|------|
| pnpm | `minimumReleaseAge` | minutes |
| npm | `min-release-age` | days |
| Yarn | `npmMinimalAgeGate` | duration string |
| Bun | `minimumReleaseAge` | duration string |
| Deno | `--minimum-dependency-age` | duration string |
| pip | `--uploaded-prior-to` | timestamp/duration |
| uv | `--exclude-newer` | duration string |
| Renovate | `minimumReleaseAge` | duration string |
| Dependabot | `cooldown.default-days` | days |
| Snyk | (non-configurable) | 21 days |

---

## 3. Dependency Update Tool Support

### Renovate (`minimumReleaseAge`)

The most mature and flexible implementation, available long before package managers added native support.

**Configuration:**
```json
{
  "extends": ["security:minimumReleaseAgeNpm"],
  "minimumReleaseAge": "14 days",
  "internalChecksFilter": "strict",
  "packageRules": [
    {
      "matchPackageNames": ["@internal-org/*"],
      "minimumReleaseAge": null
    }
  ]
}
```

**Key behaviors:**
- Waits for each version independently (not "no releases for X days")
- Adds pending/passing status checks to PR branches
- `internalChecksFilter: strict` prevents branch/PR creation until age check passes (recommended)
- **Security updates bypass `minimumReleaseAge` automatically**
- For npm: passes `--before=<date>` to npm commands during lock file generation, calculated as `now - minimumReleaseAge`
- Renovate 42 changed defaults: missing timestamps now treated as "not yet passing" (safer)
- Mend Renovate 42 made 3-day minimum the default for npm via `security:minimumReleaseAgeNpm` preset

**Limitation:** Renovate does not manage transitive dependencies — this is why configuring age-gating in both Renovate and your package manager is recommended.

### Dependabot (`cooldown`)

Shipped July 2025 with per-semver-level granularity.

**Configuration:**
```yaml
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "daily"
    cooldown:
      default-days: 5
      semver-major-days: 30
      semver-minor-days: 10
      semver-patch-days: 2
      include: ["*"]
      exclude: ["express", "lodash*"]
```

**Key behaviors:**
- `default-days`: baseline cooldown for all updates (1-90 days)
- `semver-major-days`, `semver-minor-days`, `semver-patch-days`: override per update type
- `include`/`exclude`: wildcard matching, max 150 items each; `exclude` takes precedence
- **Security updates bypass cooldown entirely** — CVE patches flow immediately
- SemVer-level overrides supported for most package managers (npm, pip, cargo, maven, nuget, etc.)
- Package managers without SemVer support (Docker, GitHub Actions, Terraform, Helm) can only use `default-days`

### Snyk

Non-configurable 21-day cooldown on automatic upgrade PRs. The most aggressive stance — no configuration needed, but no flexibility either.

### npm-check-updates

`--cooldown` parameter with duration suffixes for manual update workflows.

### StepSecurity NPM Package Cooldown Check

A GitHub PR status check that blocks dependencies released within a configurable timeframe (default 48 hours):
- Operates as automated status gate in GitHub workflows
- Shows which dependency is too recent and when the check will auto-pass
- Emergency override available for org admins
- Complements other StepSecurity checks (compromised packages, PWN requests, script injection)

---

## 4. Enterprise Repository Manager Support

### JFrog Artifactory — Curation Time-Delay Policies

JFrog Curation implements an "immature package policy" that blocks new packages under a configurable age (e.g., 14 days).

**Key differentiator:** When a developer requests a blocked package, the system **seamlessly substitutes a safe, older version** rather than failing the build. This is the most developer-friendly approach among enterprise tools — builds don't break, they just use a slightly older version.

**Architecture:** JFrog Curation sits in front of Artifactory remote repositories and intercepts requests before they reach the developer. JFrog Xray provides complementary continuous scanning as a second layer.

### Sonatype Nexus Repository — Firewall Quarantine

Nexus Repository Firewall takes a policy-based quarantine approach rather than pure time-based gating.

**How it works:**
1. Configure policies with `FAIL` action at the `PROXY` stage
2. When a newly requested component violates policy, it enters quarantine
3. Quarantined components return 403 errors with violation details
4. Security teams review via Firewall Dashboard
5. Components exit quarantine via waivers or violation resolution

**Key characteristics:**
- Only quarantines **newly requested** components (pre-existing components get auditing, not quarantine)
- **Fails closed** during service unavailability — new components quarantined until evaluation completes
- Three action levels: Fail (quarantine), Warn (email notification), No Action (audit only)
- Waiver system with optional time-window expiration
- REST API for programmatic integration
- Requires Repository + Firewall license

**Important distinction:** Nexus Firewall quarantines based on **policy violations** (known vulnerabilities, license issues, suspicious characteristics), not purely on publication age. It is a scanning-first quarantine, not a time-based hold. However, the policy engine can incorporate age-related rules.

---

## 5. Registry-Level Quarantine

### PyPI

PyPI has introduced an internal quarantine system for suspected malware that **freezes packages for investigation** rather than immediately deleting them. This preserves evidence for security research while preventing downloads. However, this is a reactive measure used by PyPI administrators — not a configurable hold period for consumers.

PyPI does not implement a consumer-facing publication delay or hold period. All packages are immediately installable upon publication.

### npm

npm does not implement a registry-level hold period. The `min-release-age` feature is client-side — the registry serves all packages immediately, and it's the client that decides whether to install based on age.

npm has an open proposal to make 7 days the default `min-release-age`, which would effectively make age-gating opt-out rather than opt-in for the largest package ecosystem.

### gem.coop (Ruby — Alternative Registry)

The only example of a **registry-level** age gate in production. gem.coop's cooldown beta serves a delayed index from a separate endpoint (`beta.gem.coop/cooldown`), hiding packages younger than 48 hours from resolution entirely. This is architecturally significant because:
- No client changes needed (Bundler works unchanged)
- The delay is enforced server-side — client-side tooling cannot bypass it
- Projects opt in by changing their gem source URL

### No Public Registry Has Mandatory Quarantine

No major public registry (npm, PyPI, Maven Central, crates.io, RubyGems.org, NuGet) implements a mandatory hold period before packages become available. The industry consensus appears to be that this should be a **client-side choice** or an **alternative registry feature**, not a mainline registry policy — likely because mandatory delays would break legitimate workflows (coordinated releases, CI/CD pipelines, monorepo publishing).

---

## 6. Custom Implementations & Open-Source Tools

### set-minimum-package-release-age (GitHub)

An open-source bash script that configures age-gating across multiple package managers in a single invocation:

```bash
bash set_package_min_age_linux.sh 7          # 7-day minimum for all tools
bash set_package_min_age_linux.sh 14 \
  --exception "pnpm:@myorg/*" \
  --exception "uv:setuptools=false"
```

**Supported managers:** pip, uv, npm, pnpm, bun, yarn classic, yarn berry

**Design qualities:** idempotent, backs up existing configs, validates syntax, version-aware preflight checks. This is the closest thing to a "configure once across all ecosystems" solution.

### Socket Firewall

Not purely age-gating, but a complementary proxy-based defense:
- Intercepts package manager requests as HTTP/HTTPS proxy
- Queries Socket's security intelligence API
- Blocks malicious packages at any dependency depth
- Free tier covers known malware; Enterprise adds custom policies, AI-detected malware, CVE treatment
- Supports npm, yarn, pnpm, pip, uv, cargo (Enterprise adds Maven/Gradle, gem/Bundler, NuGet)

Socket's approach is behavioral analysis (detecting install scripts, network requests, obfuscated code) rather than time-based gating, making it complementary to age-gating.

### Verdaccio / Devpi as Quarantine Proxies

Organizations can run internal package proxies (Verdaccio for npm, Devpi for PyPI) in allowlist mode and implement custom quarantine logic:
- Proxy upstream registry
- Hold new versions for a configurable period before making them available internally
- Requires custom scripting but provides complete control
- Several blog posts describe this pattern but no turnkey open-source quarantine proxy exists

---

## 7. Tradeoffs & Exception Handling

### What Breaks with Age-Gating

1. **Zero-day security patches**: When a critical CVE drops with a patched version, a 7-day hold means 7 days of known vulnerability exposure. This is the fundamental tension.

2. **Coordinated releases**: Monorepo publishing (e.g., React releasing `react`, `react-dom`, `react-reconciler` simultaneously) requires all packages to be available together. Age-gating can create version skew if packages publish at slightly different times.

3. **CI/CD pipelines**: Fresh package publishes in CI (e.g., publishing a library then immediately consuming it in a downstream build) will fail if age-gating is enabled without exclusions for internal packages.

4. **Rapid iteration during development**: Early-stage projects publishing multiple versions per day will hit friction.

5. **Timezone and timestamp edge cases**: "A few hours of timezone drift can determine whether a package published six days and twenty-two hours ago passes the cooldown check or not."

### Exception Strategies

**Security update bypass (universal pattern):**
All major tools exempt security updates from age-gating:
- Renovate: security updates bypass `minimumReleaseAge` automatically
- Dependabot: cooldown applies only to version updates, not security updates
- Snyk: 21-day cooldown but security advisories trigger immediate PRs

**Internal package exemption:**
- pnpm: `minimumReleaseAgeExclude: ['@your-org/*']`
- Renovate: `packageRules` with `minimumReleaseAge: null` for internal packages
- Dependabot: `exclude` list with wildcard support
- npm: **not yet supported** (open GitHub issue) — a significant gap

**Security SLA approach (recommended):**
- Critical CVEs: triage within 24 hours, patch within 72 hours
- Document all cooldown bypasses for audit
- Review bypass patches for obfuscation, dynamic execution, unexpected network access

### The Overcorrection Risk

A subtle but important failure mode: "Cooldowns reduce exposure to unknown malicious releases... without active vulnerability alerting and triage, cooldowns can actually increase dwell time for exploitable CVEs."

If teams enable cooldowns but relax their monitoring of security advisories (assuming cooldowns protect them), they may end up *slower* to patch known vulnerabilities. Cooldowns must be paired with active vulnerability alerting, not treated as a substitute for it.

---

## 8. Comparison to Alternative Defenses

Age-gating is one layer in a defense-in-depth strategy. Here is how it compares to and complements other approaches:

| Defense | What It Catches | Deployment Model | Maintenance | Coverage Gap |
|---------|----------------|-----------------|-------------|-------------|
| **Age-gating** | Short-lived malicious versions, compromised account publishes | Client-side config or proxy | Near-zero after setup | Typosquatting, long-term compromise, tag rewriting |
| **Signature/provenance (SLSA, Sigstore)** | Tampered artifacts, unauthorized builds | Registry + CI integration | Moderate (CI pipeline changes) | Compromised build systems, legitimate-looking malicious code |
| **Vulnerability scanning (Snyk, OSV, npm audit)** | Known CVEs, known malicious packages | CI/CD integration | Low (SaaS-managed) | Zero-day malware (no CVE exists yet) |
| **Behavioral analysis (Socket)** | Obfuscated code, suspicious install scripts, network access | Proxy/wrapper | Low (SaaS-managed) | Sophisticated attacks that mimic legitimate behavior |
| **Lock file enforcement** | Unintended dependency changes | Package manager config | Near-zero | Doesn't prevent initial introduction of malicious deps |
| **Install script blocking** | Malicious postinstall/preinstall hooks | Package manager config | Near-zero | Malware that operates at import-time, not install-time |

### Complementary Relationships

Age-gating is **uniquely positioned** as a prevention control that works against **unknown threats** (packages with no CVE yet). Most other tools are detection controls that work against **known threats**. The combination is powerful:

1. **Age-gating + vulnerability scanning**: Age-gating catches the "golden hour" zero-day malware. Vulnerability scanning catches known CVEs that age-gating explicitly bypasses.

2. **Age-gating + lock files**: Lock files create de-facto cooldowns by making updates deliberate. Age-gating adds an explicit time floor even when lock files are updated.

3. **Age-gating + provenance/SLSA**: Provenance verifies *who* built the package and *how*. Age-gating adds *when* as an additional signal. Neither alone is sufficient.

4. **Age-gating + behavioral analysis (Socket)**: Socket catches obfuscated/suspicious code regardless of age. Age-gating catches compromised-but-clean-looking code that was published by a hijacked account and removed within hours.

### When Age-Gating is the Wrong Primary Defense

- **Typosquatting**: The malicious package may be weeks or months old. Namespace verification and behavioral analysis are better.
- **Long-term maintainer compromise**: Malicious code introduced gradually over many versions won't be caught by age-gating. Code review and behavioral analysis are needed.
- **Mutable reference attacks**: The Trivy compromise (rewriting existing git tags) bypasses age-gating entirely. Pinning to immutable references (commit SHAs, content hashes) is the correct defense.

---

## 9. Recommended Implementation Strategy

### Minimum Viable Age-Gating (implement this week)

For an organization using multiple ecosystems, the fastest path to protection:

1. **npm**: Add `min-release-age=3` to `.npmrc`
2. **pnpm**: Upgrade to v11 (1-day default) or set `minimumReleaseAge: 4320` (3 days) in `pnpm-workspace.yaml`
3. **pip**: Add `uploaded-prior-to` to `pip.conf` (or use `set-minimum-package-release-age` script)
4. **Renovate**: Extend `security:minimumReleaseAgeNpm` preset + set `minimumReleaseAge: "7 days"` globally
5. **Dependabot**: Add `cooldown: { default-days: 5 }` to `dependabot.yml`

### Layered Approach (recommended)

Configure age-gating at **both** the package manager level (catches transitive dependencies) and the update tool level (catches direct dependency PRs):

- Package manager config: 1-3 day hold (catches the "golden hour")
- Update tool config: 7-14 day hold for routine updates (higher safety margin)
- Security updates: always bypass (Renovate and Dependabot do this automatically)
- Internal packages: exempt via exclusion rules

### Enterprise Grade

- JFrog Curation with immature package policy (14-day hold, seamless version substitution)
- OR Sonatype Nexus Firewall (policy-based quarantine with manual review workflow)
- Combined with Socket Firewall for behavioral analysis
- All wrapped in SLSA/provenance verification in CI

---

## Sources

All source material is saved in `docs/`:

- `docs/nesbitt-package-managers-cool-down.md` — Comprehensive overview of cooldown adoption across ecosystems
- `docs/renovate-minimum-release-age-docs.md` — Official Renovate documentation for minimumReleaseAge
- `docs/socket-firewall-overview.md` — Socket Firewall architecture and capabilities
- `docs/jfrog-curation-time-delay-policies.md` — JFrog Curation immature package policy
- `docs/sonatype-nexus-firewall-quarantine.md` — Sonatype Nexus Firewall quarantine documentation
- `docs/stepsecurity-npm-cooldown-check.md` — StepSecurity PR-level cooldown check
- `docs/dependabot-cooldown-configuration.md` — Dependabot cooldown YAML configuration
- `docs/pnpm-supply-chain-security.md` — pnpm supply chain security features
- `docs/set-minimum-package-release-age-tool.md` — Open-source cross-ecosystem age-gating script
- `docs/gem-coop-dependency-cooldowns.md` — gem.coop registry-level cooldown beta
- `docs/pypi-2025-year-in-review-malware.md` — PyPI malware detection statistics
- `docs/npm-minimumreleaseage-socket-blog.md` — npm minimumReleaseAge feature details
- `docs/schneider-dependency-cooldowns-defense.md` — Tradeoff analysis and defense-in-depth positioning
- `docs/spring-2026-oss-incidents-hardening.md` — Four-layer defense framework from 2026 incidents

---

## Depth Checklist

- [x] **Underlying mechanism explained**: Why age-gating works (attacker time advantage, detection statistics)
- [x] **Key tradeoffs and limitations**: Security patch delays, coordinated releases, CI/CD friction, timezone edge cases, overcorrection risk
- [x] **Compared to alternatives**: Table comparing age-gating vs signature verification, scanning, behavioral analysis, lock files, install script blocking
- [x] **Failure modes and edge cases**: Typosquatting bypass, long-term compromise bypass, mutable tag rewriting (Trivy), internal package friction, timestamp drift
- [x] **Concrete examples**: Configuration for npm, pnpm, Yarn, Bun, pip, uv, Renovate, Dependabot; real-world incident timelines (Nx, Axios, chalk, TanStack, LiteLLM)
- [x] **Standalone-readable**: Sufficient for implementation decisions without consulting original sources
