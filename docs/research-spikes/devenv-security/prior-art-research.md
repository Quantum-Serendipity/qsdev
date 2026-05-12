# Prior Art & Community Practices for Hardening devenv.sh / Nix Dev Environments

## Executive Summary

There is no widely-adopted, security-hardened devenv.sh boilerplate. The ecosystem is fragmented: individual security capabilities exist (build sandboxing, signature verification, vulnerability scanning, secrets management) but nobody has assembled them into a cohesive "secure by default" devenv template. This represents a clear gap and an opportunity. The devenv.sh project itself is actively developing security features (SecretSpec for secrets, sandbox PRs for runtime isolation) but these remain either nascent or opt-in. The broader Nix ecosystem has maturing infrastructure for supply chain security (security tracker, SBOM tooling, CVE scanning) that could be wired into a devenv boilerplate as pre-commit hooks or CI checks.

---

## 1. Existing Security-Hardened devenv Templates

**Finding: None exist.** A thorough search of GitHub, community forums, and blog posts found zero dedicated "hardened devenv.sh" boilerplate repositories. Available templates focus on language-specific tooling:

- [the-nix-way/dev-templates](https://github.com/the-nix-way/dev-templates) -- language boilerplates, no security focus
- [shahinism/devenv-templates](https://github.com/shahinism/devenv-templates) -- opinionated configs, no security layer
- [nix-dot-dev/getting-started-devenv-template](https://github.com/nix-dot-dev/getting-started-devenv-template) -- tutorial-oriented, minimal

**Maturity**: N/A -- the gap is confirmed.
**Applicability**: High -- this is exactly what we would build.

---

## 2. Nix Security Guides & Hardening Resources

### 2.1 NixOS Wiki Security Page
The [official wiki](https://wiki.nixos.org/wiki/Security) covers system-level concerns: LUKS encryption, firewall defaults, container/VM isolation, systemd service hardening. It notes that SELinux and AppArmor remain poorly integrated as of 2026. Relevant to devenv mainly as background on what NixOS provides at the OS layer.

### 2.2 Hardening NixOS Guide (nix-book)
A [comprehensive guide](https://saylesss88.github.io/nix/hardening_NixOS.html) covering kernel hardening (sysctl, boot params), privilege escalation prevention (run0 over sudo), memory corruption defenses (hardened_malloc), SSH hardening, and impermanence strategies. Primarily system-level, but the secrets management section (sops-nix, agenix) and the software selection strategy (checking maintainer activity, CVE history, channel choice) are directly applicable to devenv boilerplate guidance.

### 2.3 nix-mineral
[nix-mineral](https://github.com/cynicsketch/nix-mineral) is a NixOS module providing drop-in system hardening: kernel module blacklisting, sysctl hardening, filesystem protection, network security. Alpha status, assumes non-state adversaries. Not directly usable in devenv (it's a NixOS module, not a dev shell config) but demonstrates the pattern of "import one module, get defense-in-depth."

### 2.4 NixOS Hardened Profile
The [nixpkgs hardened profile](https://github.com/NixOS/nixpkgs/blob/master/nixos/modules/profiles/hardened.nix) enables hardened kernel, scudo allocator, and various security options. Again system-level, but it establishes the precedent for a "profile" approach -- a single import that activates a bundle of security defaults.

**Maturity**: System-level guides are mature. Dev-environment-specific guidance is absent.
**Applicability**: The "import one module for defense-in-depth" pattern from nix-mineral and the hardened profile is the model our boilerplate should follow.

---

## 3. Supply Chain Security in the Nix Ecosystem

### 3.1 Nixpkgs Review Process
Nixpkgs changes require review before merging. This is a social control -- "do you trust our review process" -- not a technical one. Users can opt out of the binary cache entirely and build from source, which is reportedly more straightforward in nixpkgs than Debian.

### 3.2 Nixpkgs Security Tracker
The [nix-security-tracker](https://tracker.security.nixos.org/) is a web service matching CVEs to nixpkgs derivations. Funded by the Sovereign Tech Fund, it became operationally active in 2025. The security team uses a three-phase workflow: triage, draft, and mitigation (auto-creating GitHub issues for affected maintainers). This is the canonical source for "is my nixpkgs version affected by CVE-X."

### 3.3 Nixpkgs Supply Chain Security Project
A [broader initiative](https://discourse.nixos.org/t/nixpkgs-supply-chain-security-project/34345) encompassing the tracker plus: automated commit-bit lifetime management, security reviews of core packages, and development of a nix-local-security-scanner. As of mid-2025, most components were entering production.

### 3.4 SBOM Generation
Multiple tools exist but the ecosystem is immature:

- **[sbomnix](https://github.com/tiiuae/sbomnix)**: Most complete. Generates CycloneDX and SPDX SBOMs from Nix flake refs or store paths. Includes vulnxscan for vulnerability scanning, nixgraph for dependency visualization, and provenance for SLSA attestation. 21+ releases. Originates from the Ghaf Framework (Tii UAE).
- **Genealogous** and **bombon**: Operate at the Nix-level rather than on derivations. Less mature.
- **Determinate Systems**: Developing internal SBOM tooling, not yet public.

Key gaps: missing package type classification, insufficient provenance information, string context issues causing missed dependencies. The EU Cyber Resilience Act (enforcement starting Sept 2026) is creating regulatory pressure for better SBOM capabilities.

### 3.5 CVE Scanning
- **[vulnix](https://github.com/nix-community/vulnix)**: Nix-specific CVE scanner. Matches derivation names/versions against NIST NVD. Supports whitelisting, JSON output, patch auto-detection. Limitation: name-matching heuristic is acknowledged as "too simplistic."
- **Trivy/Grype**: General-purpose scanners. Can scan Nix-built container images and filesystems. Not Nix-native but broader coverage.
- **nix-local-security-scanner**: Under development as part of the supply chain security project.

**Maturity**: Security tracker is production. SBOM tooling is functional but gaps remain. CVE scanning works but has known false-positive/negative issues.
**Applicability**: High -- vulnix and sbomnix can be integrated into devenv as pre-commit hooks or CI checks. The security tracker provides the data feed.

---

## 4. devenv.sh Security-Related Features & Discussions

### 4.1 Trust Model (devenv allow / devenv revoke)
devenv implements explicit trust for auto-activation. When you `cd` into a project directory, the shell hook checks a trust database before activating. Users must run `devenv allow` to opt in. `devenv revoke` removes trust. This prevents untrusted projects from modifying your shell environment. The hook only detects projects with `devenv.yaml` (not bare `devenv.nix`).

### 4.2 SecretSpec (devenv 1.8+, July 2025)
[SecretSpec](https://devenv.sh/integrations/secretspec/) separates secret declaration from provisioning. Secrets are defined in `secretspec.toml`; each environment supplies them from preferred backends (keyring, dotenv, 1password, lastpass, env). The recommended approach is runtime loading via `secretspec run -- [command]` to keep secrets out of the shell environment entirely. This is devenv's answer to the "secrets in /nix/store" problem.

### 4.3 Sandbox PRs (Draft, Not Merged)
Two competing approaches are under discussion:

**PR #2427 (bubblewrap, full-shell sandbox)**: Wraps the entire devenv shell in bubblewrap, isolating it via Linux namespaces. Users whitelist filesystem paths via mount declarations. Linux-only. Protects against compromised dependencies executing arbitrary commands but restricts normal shell operations.

**PR #1783 (Landlock, per-executable sandbox)**: Wraps individual Nix-provided executables with Landlock LSM restrictions. The user's shell remains unrestricted; only devenv-provided binaries are sandboxed. Less disruptive to workflow but per-executable only.

Neither is merged. The community discussion suggests a hybrid approach may emerge.

### 4.4 Git Hooks Integration
devenv provides [first-class pre-commit integration](https://devenv.sh/git-hooks/) via git-hooks.nix. Available hooks include shellcheck, black, clippy, ormolu, and custom hooks. The .pre-commit-config.yaml is auto-generated in the Nix store. Security-focused hooks (Trivy filesystem scan, secret detection) can be added as custom hooks but are not built in.

### 4.5 nixConfig Trust Risks
Flakes can embed [nixConfig](https://notashelf.dev/posts/reject-flake-content) settings that modify nix.conf behavior (substituters, trusted keys, allowUnfree). This is a significant attack vector: accepting a flake's nixConfig can introduce unsigned binary caches. Best practice: keep `accept-flake-config = false` (the default prompts but users tend to accept without scrutiny). The Lix fork offers a `reject-flake-config` patch for automatic blocking.

### 4.6 Container Generation
devenv can generate `.devcontainer.json` files, enabling use with VS Code Dev Containers, GitHub Codespaces, etc. This provides an optional additional isolation layer via containerization.

**Maturity**: Trust model is production. SecretSpec is production (v0.4.0). Sandboxing is draft/experimental. Git hooks are production but security hooks are DIY.
**Applicability**: All directly relevant. SecretSpec and trust model are ready for boilerplate inclusion. Sandbox features will be when merged.

---

## 5. Comparison with Other Hardened Dev Environment Approaches

### 5.1 Docker/Podman Devcontainers
Devcontainers provide process-level isolation via Linux namespaces/cgroups. Stronger isolation than Nix shells (which run directly on the host) but weaker reproducibility guarantees. Dockerfiles are not deterministic -- same Dockerfile can produce different images over time. Security depends on base image provenance and update discipline.

### 5.2 GitHub Codespaces
Runs on Azure VMs, providing VM-level isolation. Strong security boundaries between users. But opaque infrastructure -- users cannot audit the host environment. Good for multi-tenant scenarios.

### 5.3 Gitpod (Ona)
Container-based isolation. Recent kernel advances in namespace restrictions have improved security to "near VM parity." Opinionated security model.

### 5.4 VM-Level Isolation (Kata, gVisor, Firecracker)
For maximum protection against malicious dependencies, microVM isolation is recommended. Kata Containers + Cloud Hypervisor or gVisor provide true VM-level separation. This is the gold standard for untrusted code execution but significant overhead.

### 5.5 Devbox (Jetify)
Nix-powered like devenv, with a JSON-based interface instead of Nix language. No security-specific features beyond what Nix provides. Does not add sandboxing or vulnerability scanning.

### Key Tradeoff
Nix/devenv provides **reproducibility and auditability** (deterministic builds, locked dependencies, content-addressed store) but **weak runtime isolation** (no process boundary between dev shell and host). Containers/VMs provide **strong runtime isolation** but **weaker reproducibility**. A hardened devenv boilerplate should leverage Nix's reproducibility strengths while adding runtime protections.

**Maturity**: Devcontainer security is mature. VM-level isolation is mature. Nix-specific runtime isolation is immature.
**Applicability**: The devenv devcontainer generator bridges the gap, allowing teams to get Nix reproducibility inside container isolation.

---

## 6. Tools That Complement devenv for Security

| Tool | Purpose | Nix-Native? | Maturity | Integration Path |
|------|---------|-------------|----------|------------------|
| **vulnix** | CVE scanning of Nix derivations | Yes | Production | Pre-commit hook or CI step |
| **sbomnix** | SBOM generation (CycloneDX/SPDX) | Yes | Production | CI step, compliance reporting |
| **vulnxscan** | Vulnerability scanning via SBOMs | Yes | Production | CI step alongside sbomnix |
| **nix-security-tracker** | CVE-to-nixpkgs matching | Yes | Production | Web dashboard, data feed |
| **Trivy** | Container/filesystem vuln scanning | No | Production | Pre-commit hook (community hooks exist) |
| **Grype** | Vulnerability matching | No | Production | CI step, lower false positives than Trivy |
| **syft** | SBOM generation (general) | No | Production | CI step for non-Nix artifacts |
| **detect-secrets** | Pre-commit secret detection | No | Production | devenv git hook |
| **sops-nix** | Encrypted secrets in Nix | Yes | Production | NixOS-level, complements SecretSpec |
| **agenix** | Age-encrypted secrets for Nix | Yes | Production | NixOS-level, complements SecretSpec |

**Applicability**: vulnix, Trivy (pre-commit), detect-secrets, and sbomnix are the most immediately actionable for a devenv boilerplate.

---

## 7. Real-World Incidents

### 7.1 Pwning the Entire Nix Ecosystem (2025)
A [security researcher](https://ptrpa.ws/nixpkgs-actions-abuse) found that 14 files in the nixpkgs repo used GitHub Actions' `pull_request_target` trigger, which grants read/write access and secrets even from fork PRs. A CODEOWNERS validator could be exploited via symlink to leak GitHub credentials. Discovery, report, and fix completed in one day. Lesson: CI/CD pipeline security is as critical as package security.

### 7.2 Cache Poisoning Risks
The [Garnix blog](https://garnix.io/blog/stop-trusting-nix-caches/) documented how external Nix caches create privilege escalation paths. Anyone with cache write access (often all CI contributors) can push malicious binaries that get substituted for legitimate packages. Packages invoked via sudo (nix-daemon, nixos-rebuild) become direct privilege escalation vectors. Recommendation: restrict signing authority to build infrastructure, audit all configured caches.

### 7.3 nixConfig Injection
A documented [attack pattern](https://notashelf.dev/posts/reject-flake-content) where malicious flakes embed nixConfig to add untrusted substituters or disable signature verification. Users habituated to accepting prompts provide the entry point.

**No incidents of actual compromised nixpkgs packages were found** -- the review process has held so far. But the attack surface (GitHub Actions, cache trust, nixConfig) is well-documented.

---

## 8. Nix RFCs and Proposals for Improved Security

### 8.1 RFC 0062: Content-Addressed Paths
[This RFC](https://github.com/NixOS/rfcs/blob/master/rfcs/0062-content-addressed-paths.md) shifts from input-addressed to content-addressed derivations, enabling cryptographic verification of outputs without trusting signatures. The hash of the content IS the verification. Also enables multi-user scenarios with separate trust domains. Currently behind the `ca-derivation` experimental flag. When stable, this significantly improves the trust model for shared caches and multi-developer environments.

### 8.2 RFC 0100: Git Signing for Nix Projects
Proposes allowing Nix projects to automatically verify trust of upstream projects via Git signatures. Would enable verification that nixpkgs updates are trustworthy. Still seeking shepherds as of search date.

### 8.3 RFC 0136: Stabilizing Flakes
Approved plan to incrementally stabilize the CLI and flakes. Security implications: stabilization would lock down the flake trust model and `nixConfig` behavior, reducing the moving-target problem for security tooling.

### 8.4 Nix Build Sandboxing (Existing, Not RFC)
Already implemented: `sandbox = true` in nix.conf isolates builds via Linux namespaces (PID, mount, network, IPC, UTS). Builds only see the Nix store, temp dirs, and explicitly allowed paths. Enabled by default on NixOS. Fixed-output derivations bypass network isolation (a known security trade-off). This is the foundational security mechanism that devenv inherits.

---

## 9. Assessment: What a Hardened devenv Boilerplate Should Include

Based on the prior art survey, a hardened devenv.sh boilerplate should layer defenses across these categories:

### Immediately Actionable (Production-Ready Tools)
1. **Pin all flake inputs** -- commit `flake.lock`, use explicit revisions not branches
2. **SecretSpec integration** -- declare secrets, load at runtime, never in Nix store
3. **Pre-commit security hooks** -- detect-secrets, Trivy filesystem scan, shellcheck
4. **Cache hygiene** -- audit trusted substituters, enforce `require-sigs = true`, reject `accept-flake-config`
5. **vulnix scanning** -- periodic or CI-gated CVE checks against declared dependencies
6. **Trust model enforcement** -- document `devenv allow`/`devenv revoke` in project README

### Near-Term (Maturing Features)
7. **SBOM generation** -- sbomnix in CI for compliance and auditability
8. **Sandbox** -- track PR #2427 / #1783, enable when merged
9. **Devcontainer generation** -- optional containerized isolation layer
10. **Content-addressed derivations** -- enable `ca-derivation` experimental flag for improved integrity

### Systemic (Requires Ecosystem Maturity)
11. **Git-signed nixpkgs verification** -- awaiting RFC 0100 implementation
12. **Nix-local-security-scanner** -- under development
13. **Better SBOM metadata** -- package types, vulnerability annotations

---

## Sources

All raw source material is saved in `docs/`:
- `docs/nix-attacker-vs-defender-battlefield.md`
- `docs/devenv-sandbox-pr-2427.md`
- `docs/nixos-wiki-security.md`
- `docs/sbomnix-readme.md`
- `docs/discourse-nix-supply-chain-attacks.md`
- `docs/garnix-stop-trusting-nix-caches.md`
- `docs/vulnix-readme.md`
- `docs/nix-mineral-readme.md`
- `docs/discourse-nixpkgs-supply-chain-security-project.md`
- `docs/discourse-nix-state-of-sbom.md`
- `docs/nix-security-tracker-readme.md`
- `docs/pwning-nix-ecosystem.md`
- `docs/devenv-auto-activation.md`
- `docs/devenv-secretspec.md`
- `docs/devenv-git-hooks.md`
- `docs/devenv-options-reference-security.md`
- `docs/devenv-2025-blog-archive.md`
- `docs/rfc-0062-content-addressed-paths.md`
- `docs/hardening-nixos-guide.md`
- `docs/cloud-dev-env-security-comparison.md`
- `docs/nixconfig-flake-security-risks.md`
- `docs/nix-build-sandboxing-discourse.md`
