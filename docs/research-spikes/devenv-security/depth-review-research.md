# Depth Checklist Review: Devenv-Security Spike

## Date: 2026-05-12

## Scope
Review of all 12 research reports in the devenv-security spike against the depth checklist, plus cross-report consistency verification.

**Research question**: How can devenv.sh be configured as a security-hardened boilerplate for developers?

---

## Per-Report Depth Checklist Scores

### 1. architecture-research.md (P1-T1)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Deep treatment of evaluation model, C FFI in v2.0, caching architecture, NixOS module system. Data flow diagrams. |
| Tradeoffs & limitations | PASS | Section 13 covers 6 categories of edge cases. Comparison table (Section 12) identifies vendor lock-in, purity issues, lock format changes. |
| Compared to alternatives | PASS | Detailed comparison table against nix develop, lorri, nix-direnv, services-flake with 14 feature dimensions. |
| Failure modes & edge cases | PASS | Evaluation purity, lock file drift, cache coherence, process cleanup, platform gaps, module conflicts all covered. |
| Concrete examples | PASS | Code examples for devenv.nix, devenv.yaml, lock file JSON format, direnv integration, import composition. |
| Standalone readable | PASS | Fully self-contained architectural overview. No need to consult external sources. |

**Overall**: PASS (6/6). Exemplary report.

---

### 2. security-surface-research.md (P1-T2)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Each of 10 attack vectors has mechanism, prerequisites, impact rating, and gap analysis. Binary cache provenance gap (Deriver field unsigned) is especially well-explained. |
| Tradeoffs & limitations | PASS | Cross-cutting concerns section identifies 5 architectural observations including the fundamental "sandbox protects builds, not developers" insight. |
| Compared to alternatives | PARTIAL | Compares Nix's security model to npm/pip implicitly (e.g., "unlike npm, Nix packages do not have install scripts"), but no systematic comparison to container-based or VM-based dev environments for threat modeling. |
| Failure modes & edge cases | PASS | 25+ sub-vectors including re-evaluation fork bomb (3d), TOCTOU attack on .envrc (5b), registry attacks (7c), impureEnvVars leakage (8c). |
| Concrete examples | PASS | Malicious enterShell example, scripts masquerading example, env-vars file vulnerability details with CVE references. |
| Standalone readable | PASS | Complete threat model with summary matrix. Actionable without consulting sources. |

**Overall**: PASS (5/6, 1 PARTIAL). The alternatives comparison is implicit rather than explicit -- the report compares Nix to other ecosystems within each vector but doesn't have a dedicated "how does this threat model compare to Docker/VM-based dev environments" section. Minor gap.

---

### 3. nix-security-mechanisms-research.md (P1-T3)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | 9 mechanisms with deep technical detail. Store path hash computation (4-step process), sandbox namespace isolation, CA derivation trust model, signature verification chain all explained mechanistically. |
| Tradeoffs & limitations | PASS | Each mechanism has explicit limitations section. Key insight: "sandbox is for reproducibility, not security." Historical CVEs cited (3 sandbox escapes). |
| Compared to alternatives | PASS | CA derivations compared to input-addressed model. Pure eval compared to restrict-eval. Trustix mentioned as alternative trust model. NixOS security features compared to non-NixOS. |
| Failure modes & edge cases | PASS | FOD network exception, sandbox-fallback silently degrading, macOS sandbox weakness, trusted-users being root-equivalent, env-vars file vulnerability (patched). |
| Concrete examples | PASS | NixOS configuration examples, nix.conf examples, devenv flake.nix nixConfig showing exact keys/URLs. Interaction matrix table. |
| Standalone readable | PASS | Comprehensive reference. Summary table maps every mechanism to devenv behavior and hardening action. |

**Overall**: PASS (6/6). Excellent reference document.

---

### 4. config-options-research.md (P1-T4)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Every option has type, default, mechanism description, and security implications. Layered environment variable model (7 sources in order) is especially useful. |
| Tradeoffs & limitations | PASS | Each option has hardening guidance with explicit tradeoffs. Section 6 (no process isolation) and Section 8 (no sandbox for any execution context) are unflinching about limitations. |
| Compared to alternatives | PARTIAL | Options are compared to each other (dotenv vs secretspec, clean vs unsetEnvVars), but no comparison to how other dev environment tools handle the same concerns (e.g., how devcontainers handle secrets vs secretspec). |
| Failure modes & edge cases | PASS | devenv.local.nix override problem documented. Socket activation unauthenticated. File regeneration on shell entry. container.isBuilding conditional behavior. |
| Concrete examples | PASS | Complete code examples for every option. Full hardened boilerplate at the end (Section 9). |
| Standalone readable | PASS | Complete reference inventory with actionable configuration. |

**Overall**: PASS (5/6, 1 PARTIAL). Minor gap on cross-tool comparison.

---

### 5. prior-art-research.md (P1-T5)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Each tool/project has mechanism description: how nixpkgs review works, how sbomnix generates SBOMs, how vulnix matches CVEs, how SecretSpec separates declaration from provisioning. |
| Tradeoffs & limitations | PASS | Key tradeoff identified: Nix provides reproducibility but weak runtime isolation, containers provide isolation but weak reproducibility. SBOM gaps documented. |
| Compared to alternatives | PASS | Section 5 compares Docker devcontainers, GitHub Codespaces, Gitpod, VM-level isolation (Kata, gVisor), and Devbox. Table of complementary tools (Section 6). |
| Failure modes & edge cases | PASS | Real-world incidents: "Pwning the Nix Ecosystem" GitHub Actions exploit, cache poisoning risks (Garnix), nixConfig injection. |
| Concrete examples | PASS | References to specific tools (sbomnix, vulnix, nix-mineral, nix-security-tracker) with maturity assessments and integration paths. |
| Standalone readable | PASS | Clear assessment of what exists and what the boilerplate should include across 3 actionability tiers. |

**Overall**: PASS (6/6).

---

### 6. supply-chain-cross-ref-research.md (P1-T6)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PARTIAL | This is a cross-reference document, not primary research. It maps concepts rather than explaining mechanisms from scratch. Mechanisms are described at a high level with references to other reports. |
| Tradeoffs & limitations | PASS | "Contradictions and Tensions" table (7 rows) explicitly maps where general supply chain best practices conflict with Nix reality. |
| Compared to alternatives | PASS | The entire document is a comparison -- general supply chain security vs Nix-specific approaches across 6 dimensions. |
| Failure modes & edge cases | PASS | 7 devenv-specific gaps identified (G1-G7) that the general spike won't cover: shell hook security, flake input trust chain, binary cache substitution, module security, FOD gaps, pre-commit hook integrity, nix.conf trust boundaries. |
| Concrete examples | PARTIAL | References specific CVEs (CVE-2024-27297, CVE-2024-38531, CVE-2026-39860) but doesn't provide configuration examples or code. |
| Standalone readable | PASS | Clear mapping document that can be read independently. |

**Overall**: PASS (4/6, 2 PARTIAL). Acceptable for a cross-reference document -- it is not meant to stand alone as primary research. The mechanism depth is intentionally lighter because it references other reports.

---

### 7. boilerplate-research.md (P2-T1)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Every setting has inline comments explaining the mechanism. The unsetEnvVars list is particularly well-motivated with specific credential variable names. Secretspec runtime loading model explained. |
| Tradeoffs & limitations | PASS | Every setting has explicit tradeoff documentation. "What This Boilerplate Does NOT Protect Against" section (6 items) is honest about architectural limitations. MUST-HAVE/RECOMMENDED/OPTIONAL classification. |
| Compared to alternatives | PARTIAL | The boilerplate itself is the alternative (to no hardening), but the report doesn't compare this approach to, e.g., wrapping devenv in a container or using a different tool entirely. The companion nix.conf section is referenced but not compared to alternative system-level hardening approaches. |
| Failure modes & edge cases | PASS | Fork bomb (re-evaluation loop), devenv.local.nix override problem, .envrc TOCTOU, direnv approval bypass for devenv.nix changes all addressed. |
| Concrete examples | PASS | Four complete, copy-paste-ready files (devenv.yaml, devenv.nix, devenv.local.nix.example, .envrc) plus secretspec.toml. Deployment checklist. |
| Standalone readable | PASS | The boilerplate is fully self-contained with inline documentation. A developer can adopt it without reading any other report. |

**Overall**: PASS (5/6, 1 PARTIAL). The alternatives comparison is light because this is a deliverable (the boilerplate itself) rather than analytical research.

---

### 8. trust-model-research.md (P2-T4)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | 8 trust dependencies each explained mechanistically: Nix daemon trust levels, binary cache Ed25519 verification chain, flake lock narHash verification, direnv hash-check scope. |
| Tradeoffs & limitations | PASS | Each trust dependency has "What verification is missing" section. The direnv critical gap (devenv.nix changes bypass approval) is clearly articulated. |
| Compared to alternatives | PARTIAL | Briefly mentions "the safe way" vs devenv's recommendation (trusted-substituters vs trusted-users). Doesn't compare the devenv trust model to Docker/container trust models or other dev environment trust models systematically. |
| Failure modes & edge cases | PASS | Specific attack examples: malicious enterShell with curl, script masquerading (npm shadow), git hook credential exfiltration. Service modules auto-adding caches. |
| Concrete examples | PASS | Concrete malicious devenv.nix examples. NixOS configuration "the safe way" example. Complete verification matrix table. Three-tier red flag table for code review. |
| Standalone readable | PASS | Written for developers who are not security engineers. Fully standalone -- describes the trust chain from first principles. |

**Overall**: PASS (5/6, 1 PARTIAL). Strong report. The alternatives comparison gap is minor -- a systematic comparison to container trust models would strengthen it but isn't critical.

---

### 9. nix-conf-hardening-research.md (P2-T2)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Every setting has mechanism section with detailed technical explanation. The trusted-users root-equivalent problem (5-step escalation chain) is exceptionally well-explained. Per-user vs daemon config scoping is clear (table of what cannot be set per-user). |
| Tradeoffs & limitations | PASS | "What Breaks" section for every setting. Handling breakage guidance for each. The restrict-eval recommendation (CI only, not workstations) shows practical judgment. |
| Compared to alternatives | PASS | trusted-users vs trusted-substituters comparison is the central comparison. Per-user vs system vs NixOS module formats. CI vs workstation recommendations. |
| Failure modes & edge cases | PASS | sandbox-fallback silent degradation, container-in-container sandbox failure, macOS sandbox weakness, flake nixConfig injection, extra-* prefix behavior, post-build-hook root execution. |
| Concrete examples | PASS | Three complete configuration formats: NixOS module, standalone nix.conf, per-user nix.conf. Organization-specific cache extension example. Deployment checklist. |
| Standalone readable | PASS | Complete guide that can be handed to a system administrator with no other context. |

**Overall**: PASS (6/6). Excellent report.

---

### 10. precommit-hooks-research.md (P2-T3)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Hook execution model, git-hooks.nix architecture, custom hook attribute reference (Section 8), valid stages with semantics, hook bypass mechanism (--no-verify), reinstallation behavior on shell entry. |
| Tradeoffs & limitations | PASS | Performance tiers (commit-time <5s, pre-push <30s, CI-only unlimited). Ripsecrets false positive rate and lack of API verification. Vulnix heuristic acknowledged as "too simplistic." --no-verify bypass is unfixable client-side. |
| Compared to alternatives | PASS | Secret scanner comparison matrix (ripsecrets vs gitleaks vs trufflehog vs detect-secrets) across 13 dimensions. SAST tool comparison (semgrep vs bandit vs gosec vs built-ins). Vulnix vs Grype comparison. |
| Failure modes & edge cases | PASS | Hook bypass (--no-verify, SKIP=), hook uninstallation (devenv reinstalls on shell entry), flake-checker not recognizing devenv.lock format, vulnix NVD download on first run, Grype PURL gap with Nix packages. |
| Concrete examples | PASS | Complete devenv.nix configurations for every hook. Custom hook template (Section 8.3). Complex logic wrapping pattern. Complete hardened suite (Section 11). |
| Standalone readable | PASS | Comprehensive catalog that can be used independently. Section 11 is a copy-paste complete configuration. |

**Overall**: PASS (6/6). Excellent report.

---

### 11. vuln-scanning-research.md (P2-T5)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | Vulnix matching algorithm (5-step process), sbomnix SBOM generation (runtime vs buildtime), vulnxscan multi-scanner consensus, flake-checker CEL policy engine, nix-security-tracker Django architecture. |
| Tradeoffs & limitations | PASS | Fundamental gap: no tool detects language-level packages in Nix closures. Trivy doesn't support NixOS. Vulnix false positive rate is high. Nix-security-tracker has no API. PURL specification gap for Nix. |
| Compared to alternatives | PASS | Complete comparison matrix (Section 8) across 6 tools and 14 dimensions. Per-tool comparison sections. Recommendation of vulnxscan over individual tools. |
| Failure modes & edge cases | PASS | Trivy supply chain compromise (March 2026). NVD initial download latency. Grype + sbomnix PURL gap (cannot match pkg:nix/ scheme). flake-checker vs devenv.lock format mismatch. Vulnix exit code semantics (Nagios-compatible). |
| Concrete examples | PASS | devenv.nix integration examples for every tool. Complete CI pipeline examples for GitHub Actions and GitLab CI. Scanning tier summary with timing budgets. |
| Standalone readable | PASS | Complete scanning strategy guide with CI pipeline templates. |

**Overall**: PASS (6/6). Excellent report with actionable CI pipeline examples.

---

### 12. runtime-isolation-research.md (P2-T6)

| Checklist Item | Score | Notes |
|---|---|---|
| Mechanism explained | PASS | 8 isolation approaches each with mechanism explanation. Bubblewrap namespace types, Landlock LSM properties (monotonic, unprivileged, stackable), systemd user service limitations (detailed table of what works vs doesn't), devcontainer isolation levels by runtime. |
| Tradeoffs & limitations | PASS | Each approach has assessment with maturity and recommendation. systemd user services: "useful features are unavailable." Firejail: "SUID design has been a target for CVEs." Devcontainer: "devenv's key selling point is not being in a container." |
| Compared to alternatives | PASS | Comparison matrix (Section 8) across 8 approaches and 10 dimensions. Bubblewrap vs Landlock head-to-head table. |
| Failure modes & edge cases | PASS | What breaks with each approach documented: direnv state loss, shell customization failure, SSH/GPG inaccessibility, process-compose TUI issues, inotify limitations, NixOS-specific paths, Landlock monotonic property conflicting with direnv. |
| Concrete examples | PASS | Working bubblewrap wrapper script, landrun example, systemd-run command, devcontainer.enable config, firejail NixOS config. Reference implementations cited (nix-sandbox, bubblewrap-claude). |
| Standalone readable | PASS | Complete evaluation of all runtime isolation options with clear recommendations. |

**Overall**: PASS (6/6). Excellent report.

---

## Summary: Depth Checklist Results

| # | Report | Mechanisms | Tradeoffs | Alternatives | Failure Modes | Examples | Standalone | Overall |
|---|--------|-----------|-----------|-------------|--------------|---------|------------|---------|
| 1 | architecture-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 2 | security-surface-research.md | PASS | PASS | PARTIAL | PASS | PASS | PASS | **PASS (1 partial)** |
| 3 | nix-security-mechanisms-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 4 | config-options-research.md | PASS | PASS | PARTIAL | PASS | PASS | PASS | **PASS (1 partial)** |
| 5 | prior-art-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 6 | supply-chain-cross-ref-research.md | PARTIAL | PASS | PASS | PASS | PARTIAL | PASS | **PASS (2 partial)** |
| 7 | boilerplate-research.md | PASS | PASS | PARTIAL | PASS | PASS | PASS | **PASS (1 partial)** |
| 8 | trust-model-research.md | PASS | PASS | PARTIAL | PASS | PASS | PASS | **PASS (1 partial)** |
| 9 | nix-conf-hardening-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 10 | precommit-hooks-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 11 | vuln-scanning-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |
| 12 | runtime-isolation-research.md | PASS | PASS | PASS | PASS | PASS | PASS | **PASS** |

**Full passes**: 7 of 12 reports pass all 6 checklist items cleanly.
**Passes with partials**: 5 reports have 1-2 partial items. None have any FAIL items.
**No reports fail the depth checklist.**

---

## Specific Gaps Identified

### Gap 1: Cross-tool alternatives comparisons (4 reports)
Reports 2, 4, 7, and 8 each have a PARTIAL on the alternatives checklist item. The common pattern: they compare Nix-specific options thoroughly (e.g., secretspec vs dotenv, trusted-users vs trusted-substituters) but don't systematically compare the devenv approach to container-based or VM-based alternatives for the same security concern.

**Severity**: Low. The prior-art report (5) and runtime-isolation report (12) already cover these comparisons comprehensively. The gap is structural -- the information exists in the spike but isn't inline in every report.

**Recommendation**: No additional research needed. The cross-references between reports cover this gap.

### Gap 2: Cross-reference document is lighter weight (Report 6)
The supply-chain cross-reference is a mapping document, not primary research. Its mechanism descriptions are high-level because they reference other reports.

**Severity**: Low. This is the correct approach for a cross-reference document. The source spike is still in early Phase 1, so the cross-reference appropriately focuses on conceptual mapping.

**Recommendation**: Revisit this document when the package-supply-chain-security spike completes Phase 1. Pull in concrete findings at that point.

### Gap 3: No empirical validation of the boilerplate
The boilerplate (Report 7) is designed from threat model analysis and configuration documentation. It has not been tested by deploying it to a real project and verifying that: (a) all hooks install correctly, (b) secretspec integration works end-to-end, (c) clean environment doesn't break common workflows, (d) the enterTest assertions pass.

**Severity**: Medium. The boilerplate is actionable but untested. Some configurations may have syntax issues or unexpected interactions.

**Recommendation**: Add a Phase 3 task to deploy the boilerplate to a test project and run `devenv test`. This is validation, not research -- it doesn't require additional reading.

### Gap 4: devenv-nixpkgs/rolling version inconsistency in boilerplate
The boilerplate (Report 7) pins to `nixos-25.11` as the nixpkgs input, but includes `devenv.cachix.org` in the nix.conf trusted-substituters. The nix.conf guide (Report 9) correctly notes that `devenv.cachix.org` only has pre-built binaries for `devenv-nixpkgs/rolling` -- switching to upstream nixpkgs means fewer cache hits from devenv's cache.

**Severity**: Low. This is documented as a tradeoff in both reports. The nix.conf guide's inline comment says "Add devenv.cachix.org ONLY if you use devenv-nixpkgs/rolling." This is consistent -- it's a conscious tradeoff, not an error. However, the boilerplate could be clearer about when devenv.cachix.org provides value vs not.

**Recommendation**: No change needed. The tradeoff is documented.

---

## Cross-Report Consistency Checks

### Check 1: Do boilerplate (T1) settings match what config options (P1-T4) says is possible?

**Result**: CONSISTENT.

Every setting in the boilerplate references options documented in the config options inventory:
- `clean.enabled: true` / `clean.keep` -- documented in config options Section 2.3
- `impure: false` -- documented in Section 2.4
- `secretspec.enable: true` -- documented in Section 2.6
- `nixpkgs.allow_unfree: false` -- documented in Section 2.2
- `require_version: ">=2.1"` -- documented in Section 2.7
- `git-hooks.enable: true` with ripsecrets, check-added-large-files, no-commit-to-branch, shellcheck, statix -- all documented in Section 1.7 and Section 4
- `dotenv.enable = false` -- documented in Section 1.6
- `unsetEnvVars` -- documented in Section 1.13 (the boilerplate extends the defaults with credential variables)
- `enterShell` / `enterTest` -- documented in Sections 1.3 and 1.4
- `files.*` -- documented in Section 1.10
- `scripts.*` -- documented in Section 1.5

The boilerplate uses two custom hooks (`lock-file-audit`, `nix-secrets-check`) that are not in the built-in hook list but are correctly defined using the custom hook mechanism documented in config options Section 1.7.

**One minor inconsistency**: The boilerplate's `.gitignore` block is commented out with a note about projects managing their own .gitignore. The config options report (Section 1.10) presents `files.".gitignore"` as a straightforward feature. The boilerplate's conservative approach (commented out) is the correct call for a template.

### Check 2: Do pre-commit hooks (T3) match what boilerplate (T1) includes?

**Result**: CONSISTENT with intentional scoping differences.

The boilerplate includes:
- `ripsecrets.enable = true` -- matches T3 Section 1.1
- `check-added-large-files.enable = true` -- matches T3 Section 1.3
- `no-commit-to-branch.enable = true` -- matches T3 Section 1.4
- `shellcheck.enable = true` -- matches T3 Section 1.5
- `statix.enable = true` -- matches T3 Section 1.5
- `lock-file-audit` (custom) -- matches T3 Section 3.2
- `nix-secrets-check` (custom) -- unique to boilerplate (devenv-specific pattern, not in T3)

The T3 report's complete hardened suite (Section 11) includes additional hooks NOT in the boilerplate:
- `reuse` (license compliance) -- in T3 Section 11 but not in boilerplate
- `flake-checker` -- in T3 Section 11 but not in boilerplate
- `gitleaks` (pre-push) -- in T3 Section 11 but not in boilerplate
- `semgrep` (pre-push) -- in T3 Section 11 but not in boilerplate
- `trufflehog` (CI-only) -- in T3 Section 11 but not in boilerplate
- `vulnix-scan` (CI-only) -- in T3 Section 11 but not in boilerplate

**Assessment**: This is an intentional design decision, not an inconsistency. The boilerplate is meant to be a minimal, copy-paste starting point. The T3 report provides the expanded suite for teams that want more. The boilerplate explicitly tags the additional hooks as "add project-specific packages below this line," and the T3 report's Section 11 is clearly labeled as a "complete" suite vs the boilerplate's "security baseline."

**However**: The boilerplate should reference the T3 report for teams wanting the expanded suite. Currently it doesn't -- this is a minor documentation gap.

### Check 3: Does nix.conf guide (T2) align with Nix security mechanisms report (P1-T3)?

**Result**: CONSISTENT.

Cross-checking key claims:

| Claim in nix.conf guide (T2) | Nix mechanisms report (P1-T3) says | Consistent? |
|---|---|---|
| sandbox uses PID, mount, network, IPC, UTS namespaces | Same -- Section 1 | Yes |
| sandbox-fallback=true silently degrades | Same -- Section 1 "sandbox-fallback = true by default" | Yes |
| FODs bypass network isolation | Same -- Section 1 "Fixed-output derivations bypass network isolation" | Yes |
| trusted-users is root-equivalent | Same -- Section 7 quotes Nix docs: "essentially equivalent to giving that user root access" | Yes |
| require-sigs=true is default | Same -- Section 3 "require-sigs = true default" | Yes |
| filter-syscalls blocks setuid/setgid/ACLs/xattrs | Section 1 mentions seccomp but T2 adds more detail on what it blocks | Yes (T2 is more detailed) |
| restrict-eval constrains filesystem and network at eval time | Same -- Section 4 | Yes |
| trusted-substituters is the scoped alternative to trusted-users | Same -- Section 7 "Hardening opportunity" | Yes |

The nix.conf guide (T2) provides 3 historical CVEs for sandbox escapes. The mechanisms report (P1-T3) provides 7 CVEs covering sandbox escapes plus other categories. These lists overlap correctly -- T2's 3 CVEs are a subset of P1-T3's 7.

T2 adds the `accept-flake-config = false` setting that P1-T3 doesn't explicitly mention. P1-T3 notes the `nixConfig` attack surface via devenv's flake.nix, so this is a complementary addition, not a contradiction.

**One nuance**: T2 recommends `connect-timeout = 10` and `download-attempts = 3`. P1-T3 doesn't mention these settings. This is fine -- T2 is the dedicated nix.conf guide and covers more settings.

### Check 4: Does trust model (T4) accurately reflect the threat model from P1-T2?

**Result**: CONSISTENT.

The trust model document maps 8 trust dependencies. Cross-checking against the threat model's 10 attack vectors:

| Threat Model Vector (P1-T2) | Trust Model Coverage (T4) | Match? |
|---|---|---|
| 1a: Malicious overlay | Section 6 (devenv.nix) + overlay review guidance | Yes |
| 1b: Nixpkgs compromise | Section 3 (nixpkgs) + devenv-nixpkgs wrinkle | Yes |
| 1c: Flake input typosquatting | Section 5 (flake inputs) + URL review guidance | Yes |
| 2a-d: Binary cache attacks | Section 4 (binary caches) -- 3 caches, single-key limitation, all-or-nothing trust | Yes |
| 3a-d: Shell hook injection | Section 6 (devenv.nix) -- enterShell, scripts, git hooks examples | Yes |
| 4: Malicious module | Section 7 (devenv modules) -- unrestricted access, no capability model | Yes |
| 5a-d: Direnv risks | Section 8 (direnv) -- critical gap that devenv.nix changes bypass approval | Yes |
| 6a-d: Build-time code execution | Section 1 (Nix daemon) -- build sandbox, FOD exception referenced | Partial (FOD vector less detailed in T4) |
| 7a-e: Flake input manipulation | Section 5 (flake inputs) -- lock file pinning, narHash verification | Yes |
| 8a-e: Environment variable leakage | Part 5 (hardened boilerplate protections) -- clean.enabled, secretspec | Yes |
| 9a-d: Process/service isolation | Not explicitly covered in T4 (focus is on trust, not runtime isolation) | Partial |
| 10a-d: Devenv supply chain | Section 2 (devenv binary) + Section 3 (nixpkgs) | Yes |

**Two partial matches**:
1. Build-time FOD attacks (6b) are mentioned in T4's daemon section but not as a separate trust dependency. This is minor -- the FOD exception is covered in P1-T3 and the nix.conf guide.
2. Process/service isolation (9a-d) is not covered in the trust model document. This makes sense -- the trust model focuses on the trust chain before code executes, while process isolation is about what happens during execution. The runtime isolation report (12) covers this.

**No contradictions found.**

---

## Contradictions Between Reports

**No substantive contradictions found across all 12 reports.** The reports are internally consistent and reinforce each other's findings. Some minor alignment notes:

### Version pinning recommendation
- The boilerplate (Report 7) pins to `nixos-25.11`
- The trust model (Report 8) uses `nixos-24.11` in its example
- The config options (Report 4) uses `nixos-24.11` in its example

This is not a contradiction -- the boilerplate uses the latest stable release (25.11 at the time of writing), while earlier reports used the then-current stable release. The principle (pin to a stable release branch) is consistent. However, this should be normalized in the final conclusions to recommend "latest stable NixOS release" rather than a specific version number.

### devenv.cachix.org trust
- The nix.conf guide (Report 9) includes `devenv.cachix.org` in trusted-substituters
- The boilerplate (Report 7) recommends switching away from devenv-nixpkgs/rolling to upstream nixpkgs
- The nix.conf guide has an inline comment: "Add devenv.cachix.org ONLY if you use devenv-nixpkgs/rolling"

This is internally consistent but could confuse a reader who takes the NixOS module verbatim without reading the comments. The nix.conf guide should make the conditional inclusion more prominent (e.g., using a separate "Organization-specific" section).

---

## Overall Assessment

**The spike's research quality is high.** All 12 reports pass the depth checklist (7 with full passes, 5 with 1-2 partial items, zero failures). The research thoroughly answers the original question: how to configure devenv.sh as a security-hardened boilerplate.

### Strengths
1. **Consistent threat model anchoring**: Reports reference the P1-T2 threat vectors by number (e.g., "Vector 8a", "Vector 3d"), creating traceable links between the threat model and mitigations.
2. **Actionable deliverables**: The boilerplate (Report 7), nix.conf guide (Report 9), and pre-commit hook suite (Report 10) are immediately deployable.
3. **Honest limitations**: Every report documents what it does NOT protect against. The "What This Boilerplate Does NOT Protect Against" section in Report 7 is particularly valuable.
4. **Source management**: Every report has a Sources section with file references to docs/. Claims are traceable to saved sources.

### Weaknesses
1. **No empirical validation**: The boilerplate has not been deployed and tested on a real project (Gap 3 above).
2. **Cross-tool comparisons are sparse in some reports**: The alternatives checklist item scored PARTIAL in 4 reports. The information exists elsewhere in the spike, but individual reports don't always reference alternative approaches outside the Nix ecosystem.
3. **Version number normalization**: The nixpkgs channel version varies between reports (24.11 vs 25.11). Should be normalized in conclusions.

### Recommendations

1. **No additional research needed before writing conclusions.** The 12 reports collectively cover the research question with sufficient depth. All checklist items are either PASS or PARTIAL (with the partial items being minor).

2. **The boilerplate should be empirically validated** as a Phase 3 task (deploy to test project, run `devenv test`, verify hooks install, verify clean environment works).

3. **The conclusions document should normalize the nixpkgs channel recommendation** to "latest stable NixOS release" with a note to update when new releases ship.

4. **The boilerplate should add a cross-reference** to the pre-commit hooks report (T3) for teams wanting the expanded hook suite beyond the baseline.
