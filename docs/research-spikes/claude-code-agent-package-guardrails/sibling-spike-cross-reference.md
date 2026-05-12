# Sibling Spike Cross-Reference: Actionable Findings for Claude Code Agent Package Guardrails

**Source spikes**: `package-supply-chain-security/`, `devenv-security/`
**Target spike**: `claude-code-agent-package-guardrails/`
**Date**: 2026-05-12

## Purpose

This report extracts concrete, actionable findings from two completed sibling research spikes and maps them to the three-layer defense architecture already established in this spike (hooks + permissions + MCP). Each finding includes the source spike, the specific mechanism, and how it maps to a Claude Code implementation layer.

---

## 1. Publication Age Quarantine Gates

**Source**: `package-supply-chain-security/quarantine-gates-research.md`

### Key Findings

- **92% of PyPI malware is caught within 24 hours** of publication. A 3-day hold period blocks the vast majority of supply chain attacks.
- Native age-gating now exists in npm (`min-release-age`), pnpm (`minimumReleaseAge`), Yarn (`npmMinimalAgeGate`), Bun, Deno, pip (`--uploaded-prior-to`), uv (`--exclude-newer`), and Cargo.
- All major dependency update tools (Renovate `minimumReleaseAge`, Dependabot `cooldown`, Snyk 21-day default) auto-exempt security updates from age gates.
- The `set-minimum-package-release-age` bash script configures age-gating across npm, pnpm, Yarn, Bun, pip, and uv in a single invocation.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Parse install commands and query the registry API (npm registry JSON, PyPI JSON API, crates.io API) for publication timestamp. Block packages published < N days ago. The hook can use `updatedInput` to rewrite to the most recent version that passes the age gate. |
| **MCP server** | A custom MCP `check-package-age` tool wrapping registry API calls. Socket.dev MCP's `depscore` already flags "recently published" as a risk signal. |
| **CLAUDE.md** | Instruct agent: "Never install package versions published less than 3 days ago. When adding new dependencies, prefer versions with at least 7 days of publication history." |
| **OS/environment** | Configure `.npmrc` (`min-release-age=3`), `pnpm-workspace.yaml` (`minimumReleaseAge: 4320`), `pip.conf` (`uploaded-prior-to`), `.yarnrc.yml` (`npmMinimalAgeGate: 3d`) as project-level defaults. These enforce age-gating even for commands that bypass hooks. |
| **Permission deny** | Not directly applicable — age is a property of the package, not the command shape. |

### Priority: HIGH — lowest cost, highest impact single defense.

---

## 2. Install Script Sandboxing

**Source**: `package-supply-chain-security/install-sandboxing-research.md`

### Key Findings

- Install-time code execution is the single most exploited attack vector. npm lifecycle scripts, Python `setup.py`, Rust `build.rs`, and Ruby `extconf.rb` all allow arbitrary code execution during install.
- **pnpm v10+** blocks lifecycle scripts by default (`strictDepBuilds: true`) with per-package `allowBuilds`.
- **npm** requires `ignore-scripts=true` in `.npmrc` plus `@lavamoat/allow-scripts` for version-pinned allowlisting.
- **Python**: `--only-binary :all:` refuses all source distributions, eliminating `setup.py` execution.
- **Deno**: scripts blocked by default with `--allow-scripts=npm:pkg` granularity.
- The **two-phase install pattern** (download with network, build offline via bubblewrap/containers) neutralizes exfiltration.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Intercept `npm install`, `pip install`, `cargo build` commands. Rewrite via `updatedInput` to append safety flags: `npm install` → `npm install --ignore-scripts`, `pip install` → `pip install --only-binary :all:`, `cargo build` → `cargo build --locked`. If the agent needs scripts for a specific package, require it to use the project's allowlisted wrapper script. |
| **Permission deny** | `Bash(npm install *)` denied; `Bash(./scripts/safe-install *)` allowed. Forces agent through a wrapper that enforces `--ignore-scripts` + `@lavamoat/allow-scripts`. Similarly for pip: deny raw `pip install *`, allow `./scripts/pip-install *` wrapper. |
| **MCP server** | An `install-package` MCP tool that wraps the install command with safety flags and validates against the allowlist before execution. |
| **CLAUDE.md** | "Always use `--ignore-scripts` with npm/yarn. Always use `--only-binary :all:` with pip. Never run `cargo build` without `--locked`. Use the project's designated install wrapper scripts." |
| **OS/environment** | Set `.npmrc` `ignore-scripts=true` at project level. Set `PIP_ONLY_BINARY=:all:` as environment variable. These survive even if the hook is bypassed. |

### Priority: HIGH — eliminates the #1 attack vector.

---

## 3. Lockfile Enforcement

**Source**: `package-supply-chain-security/lockfile-integrity-research.md`

### Key Findings

- Lock files with hash pinning are the highest-leverage "configure-once" defense, eliminating version resolution drift and detecting registry compromise via cryptographic verification.
- **Lockfile poisoning** is a real attack class: attackers modify `resolved` URLs and `integrity` hashes in PR-submitted lockfile changes. npm's `package-lock.json` is most vulnerable; pnpm is structurally immune (no tarball URLs in lockfile).
- CI enforcement flags: `npm ci`, `pnpm install --frozen-lockfile` (default in CI), `pip install --require-hashes`, `cargo build --locked`, `go mod verify`, `uv sync --frozen`.
- **lockfile-lint** validates that all resolved URLs use HTTPS and point to allowed registries.
- **CODEOWNERS on lockfiles** + dedicated review policies catch lockfile manipulation in PRs.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Before any install command, snapshot the lockfile hash. After install (via PostToolUse), compare. If the lockfile changed unexpectedly, alert the user and optionally block the next model turn via PostToolBatch. Also: intercept `npm install` (which updates lockfiles) and rewrite to `npm ci` (which respects them). |
| **PostToolUse hook** | After any install command completes, run `lockfile-lint` or a custom hash-comparison check. Flag unexpected lockfile modifications. |
| **Permission deny** | Deny `Bash(npm install)` (bare, which resolves and may update lockfile). Allow `Bash(npm ci)` (frozen install). Deny `Bash(pip install *)` without `--require-hashes`. |
| **CLAUDE.md** | "Never run `npm install` in a project with an existing lockfile — use `npm ci`. Never modify lockfiles directly. If a lockfile needs updating, explain why and let the user approve." |
| **OS/environment** | `.npmrc` `save-exact=true` prevents range-based version specs. Yarn `.yarnrc.yml` `enableImmutableInstalls: true`. Git pre-commit hook running lockfile-lint. |

### Priority: HIGH — prevents an entire class of drift and manipulation attacks.

---

## 4. Signature Verification & Provenance

**Source**: `package-supply-chain-security/signature-provenance-research.md`

### Key Findings

- **Sigstore** is converging as the universal provenance layer. npm (~7% adoption) and PyPI (~17%) lead with automatic Sigstore attestations via Trusted Publishing.
- **Critical gap: no major ecosystem allows consumers to require provenance at install time**. Publisher infrastructure is solved; consumer enforcement is 1-2 years behind.
- **pnpm** `trustPolicy: no-downgrade` is the sole partial exception — detects when provenance disappears between versions.
- **Go** checksum database is the only default-enforced integrity mechanism (fail-closed design).
- **NuGet** `signatureValidationMode=require` is the only configurable signature enforcement.
- **cargo-vet** provides human-review attestation — organizations can import audits from Mozilla, Google, etc.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | For npm packages: query npm registry API for `attestations` field. Flag packages that lack provenance when alternatives with provenance exist. For PyPI: query PyPI JSON API for `.provenance` objects. Use this as a soft signal (warn) rather than hard block (too few packages have provenance yet). |
| **MCP server** | Socket.dev MCP's `depscore` already factors provenance into its scoring. A custom server could query `npm audit signatures` output or PyPI's Integrity API. |
| **CLAUDE.md** | "When choosing between equivalent packages, prefer packages with Sigstore provenance attestations. Check `npm audit signatures` after adding new npm dependencies. For Rust projects, respect `cargo vet` audit status." |
| **OS/environment** | For NuGet projects: configure `signatureValidationMode=require` in `nuget.config`. For Go: ensure `GONOSUMDB` is not set to `*`. For pnpm: enable `trustPolicy: no-downgrade`. |
| **Permission deny** | Not directly applicable — provenance is a package property, not a command shape. |

### Priority: MEDIUM — high value but limited consumer enforcement tools exist today. Soft signals now, hard enforcement as ecosystems mature.

---

## 5. Private Registries & Validated Mirrors

**Source**: `package-supply-chain-security/private-registries-research.md`

### Key Findings

- Private registries are the ideal enforcement point — they catch problems *before* `npm install` completes, not after.
- **JFrog Curation** intercepts package requests pre-download: blocks by CVE severity, malware databases, license, package age, and community trust signals. Seamlessly substitutes safe older versions when blocking.
- **Sonatype Nexus Firewall** quarantines based on policy violations with fail-closed behavior during unavailability.
- **Verdaccio** (npm-only, open source): `minAgeDays` age-gating, scope/package blocklists, date freezing. Best free option for npm teams.
- Cloud-native services (CodeArtifact, Azure Artifacts, Google AR) are caching layers, not security layers.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Verify that registry URLs in install commands point to the approved private registry, not directly to public registries. Reject commands with `--registry https://registry.npmjs.org` when a private registry is configured. |
| **MCP server** | Not the primary enforcement point — the registry itself does the validation. However, an MCP tool could query the private registry's API for package status/policy violations before install. |
| **CLAUDE.md** | "All package installations must go through the project's configured private registry. Never override the registry URL in install commands. Never add `--registry` flags pointing to public registries." |
| **OS/environment** | `.npmrc` `registry=https://myorg.jfrog.io/...` at project and user level. `pip.conf` `index-url` pointing to private mirror. `GOPROXY` pointing to private proxy. These enforce routing at the package manager level regardless of what the agent does. |
| **Permission deny** | `Bash(*--registry https://registry.npmjs.org*)` denied. `Bash(*--index-url https://pypi.org*)` denied. Prevents registry override attempts. |

### Priority: MEDIUM-HIGH for orgs with private registries; LOW for individual developers (requires infrastructure investment).

---

## 6. Organizational Scanning Tools

**Source**: `package-supply-chain-security/org-tooling-research.md`

### Key Findings

- **Socket.dev** (behavioral analysis) and **Snyk** (CVE scanning) are complementary — Socket catches zero-day malicious packages, Snyk catches known CVEs.
- **OSV Scanner** (Google, free) matches dependencies against the OSV database with fewer false positives than raw CVE matching.
- **OpenSSF Scorecard** evaluates project security posture (branch protection, code review, signed releases) — useful for evaluating packages before adoption.
- **StepSecurity Harden-Runner** is the only tool protecting CI pipelines themselves from compromise.
- Recommended free stack: Dependabot + OSV Scanner + Socket free + Harden-Runner + Scorecard.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Before installing a new package, query Socket.dev API for supply chain score and OSV.dev API for known vulnerabilities. Block if Socket score is below threshold or if critical CVEs are present. The `attach-guard` plugin already implements Socket.dev integration in a PreToolUse hook. |
| **PostToolUse hook** | After install, run `npm audit` / `pip-audit` / `cargo audit` and flag any new vulnerabilities introduced. |
| **MCP server** | Socket.dev MCP (`mcp.socket.dev`) — use `depscore` tool for pre-install scoring. Snyk MCP (bundled with CLI) — use `snyk_sca_scan` for post-install CVE scanning. Both are production-ready. |
| **CLAUDE.md** | "Before adding any new dependency, check its Socket.dev score and OpenSSF Scorecard. Never add a dependency with a critical Socket alert or Scorecard below 5. Run `npm audit` / `pip-audit` / `cargo audit` after any dependency change." |
| **OS/environment** | CI pipeline: OSV Scanner + Socket.dev as GitHub Actions. Pre-commit hooks running `npm audit`. Renovate with `minimumReleaseAge` for automated updates. |

### Priority: HIGH — these tools are the "what to check" layer that hooks and MCP servers delegate to.

---

## 7. Nix-Specific Protections

**Source**: `devenv-security/` spike (multiple reports)

### Key Findings

#### 7.1 Nix's Inherent Supply Chain Advantages

- **Sandboxed builds by default** (Linux): PID/mount/network/IPC namespace isolation. Stronger than anything available in npm/pip/cargo.
- **Content-addressed store**: Every package path encodes its hash — integrity verification is intrinsic.
- **Flake lock pinning**: `devenv.lock` / `flake.lock` pins every input to a specific commit hash and NAR hash. Coarser-grained than per-package lockfiles but covers the entire dependency closure.
- **Natural quarantine via channel promotion**: nixpkgs packages pass through Hydra builds and channel promotion (days to weeks of natural delay) before reaching stable/unstable channels.
- **No install scripts in the npm/pip sense**: Nix builds are sandboxed; post-install hooks don't exist as a concept. Build scripts (`builder.sh`) run inside the sandbox with no network access (except FODs).

#### 7.2 Nix-Specific Weaknesses

- **Fixed-output derivations (FODs)** bypass sandbox network isolation. A malicious FOD could contact a C2 server during build. CVE-2024-27297 and CVE-2026-39860 confirmed real FOD-related vulnerabilities.
- **Shell hooks run unsandboxed**: devenv's `enterShell`, `scripts`, and git hooks execute with full user privileges outside any sandbox. This is the largest gap.
- **Binary cache trust is single-key**: devenv auto-trusts `devenv.cachix.org`. Adding `cachix.pull` in `devenv.nix` trusts that cache completely. If a developer is a `trusted-user`, project configs can silently add attacker-controlled caches.
- **`trusted-users` is root-equivalent**: devenv documentation recommends adding users to `trusted-users`, which the Nix docs explicitly state is root-equivalent.
- **`devenv.local.nix` can override all security controls** without team visibility (not committed to git).
- **Nix evaluation is unsandboxed**: Evaluating `devenv.nix` runs Nix code with the calling user's full privileges *before* any sandbox applies.

#### 7.3 Nix-Native Scanning Tools

- **vulnix**: Scans Nix closures against NVD/CVE databases. Can be run as pre-commit hook.
- **sbomnix**: Generates CycloneDX/SPDX SBOMs from Nix flake refs. Includes `vulnxscan` for vulnerability scanning.
- **nix-security-tracker**: Matches CVEs to nixpkgs derivations (production since 2025).
- **flake-checker** (Determinate Systems): Validates flake.lock freshness and input health.

### Claude Code Implementation Map

| Layer | Implementation |
|---|---|
| **PreToolUse hook** | Intercept `nix flake update`, `nix profile install`, `devenv shell` after `devenv.nix` changes. Before `nix flake update`: warn about trust implications (updating all inputs). Before any Nix package install: verify it exists in the pinned nixpkgs. Block `nix-env -i` (imperative install bypassing flake pins). |
| **PostToolUse hook** | After `nix flake update`: run `flake-checker` to validate input health. After any devenv change: run `vulnix` against the environment closure. |
| **Permission deny** | `Bash(nix-env -i *)` denied (imperative install). `Bash(nix flake update)` in ask mode (requires user approval). `Bash(nix profile install *)` in ask mode. Deny `Bash(cachix use *)` (prevents agent from adding untrusted caches). |
| **CLAUDE.md** | "Never use `nix-env` — it bypasses flake pinning. Never run `nix flake update` without user approval. Never add entries to `cachix.pull` or `extra-substituters` in devenv.nix. Never modify `trusted-users` or `trusted-public-keys` in any Nix configuration. When adding packages to devenv.nix, use only packages from the pinned nixpkgs input." |
| **OS/environment** | System-level `nix.conf`: `sandbox = true`, `sandbox-fallback = false`, `require-sigs = true`. Use `trusted-substituters` instead of `trusted-users` to control which caches are accepted. Pin nixpkgs to a stable channel rather than `nixpkgs-unstable`. Pre-commit hook running `vulnix` or `sbomnix`. |
| **MCP server** | A custom Nix-aware MCP tool that: (1) resolves package names against the pinned nixpkgs to verify they exist, (2) checks the nix-security-tracker for known CVEs against the pinned nixpkgs commit, (3) validates that flake inputs haven't changed unexpectedly. |

### Priority: HIGH for NixOS-based workflows (applies directly to this user's environment).

---

## 8. Defense-in-Depth: Three-Layer Architecture

**Source**: Both sibling spikes, synthesized against `hooks-research.md`, `permissions-research.md`, `mcp-server-research.md`

### Mapping Sibling Findings to the Three Layers

The package-supply-chain-security spike identified **five configure-once defense layers**. Here is how each maps to Claude Code's three enforcement mechanisms plus OS-level and CLAUDE.md:

| Supply Chain Defense Layer | Hooks (PreToolUse) | Permissions (deny/ask) | MCP Servers | CLAUDE.md | OS/Environment |
|---|---|---|---|---|---|
| **Age-gating** | Query registry API for publish date; block if < N days | N/A | Socket `depscore` includes age signal | "Prefer packages > 7 days old" | `.npmrc` `min-release-age=3`, `pip.conf` `uploaded-prior-to` |
| **Install script blocking** | Rewrite commands to add `--ignore-scripts`, `--only-binary` | Deny raw install; allow wrapper scripts | Install wrapper MCP tool | "Always use --ignore-scripts" | `.npmrc` `ignore-scripts=true`, `PIP_ONLY_BINARY=:all:` |
| **Lockfile enforcement** | Snapshot + compare lockfile hash; rewrite `npm install` → `npm ci` | Deny `npm install` (bare); allow `npm ci` | N/A | "Never run npm install with existing lockfile" | `.yarnrc.yml` `enableImmutableInstalls: true` |
| **Scanning & monitoring** | Pre-install: query OSV.dev + Socket.dev APIs. Post-install: run `npm audit` | N/A | Socket MCP `depscore`, Snyk MCP `snyk_sca_scan` | "Run npm audit after changes" | CI: OSV Scanner + Socket + Harden-Runner |
| **Private registry** | Verify registry URL matches approved private registry | Deny `--registry` pointing to public registries | Query private registry API for policy status | "Never override registry URL" | `.npmrc` `registry=...`, `pip.conf` `index-url=...` |

### Cross-Cutting Defenses from devenv-security

| Nix Defense | Hooks | Permissions | MCP | CLAUDE.md | OS/Environment |
|---|---|---|---|---|---|
| **Flake lock integrity** | Post-update: run `flake-checker` | Ask: `nix flake update` | Custom: validate input hashes | "Never update flake inputs without approval" | `nix.conf` `sandbox-fallback=false` |
| **Binary cache trust** | Block commands adding untrusted caches | Deny: `cachix use *` | N/A | "Never add binary caches" | `trusted-substituters` allowlist in `nix.conf` |
| **Shell hook safety** | N/A (hooks don't intercept Nix evaluation) | N/A | N/A | "Never add shell hooks that fetch remote resources" | `clean.enabled = true` in `devenv.yaml`, code review |
| **Vulnerability scanning** | Post-change: run `vulnix` on environment closure | N/A | Custom: query nix-security-tracker | "Run vulnix after adding packages" | Pre-commit hook: `vulnix` |

---

## 9. Bypass Vectors and Mitigations (Cross-Referenced)

The hooks-research.md identified 8 bypass vectors for Claude Code guardrails. Sibling spike findings strengthen mitigations for several:

| Bypass Vector | Sibling Spike Finding | Strengthened Mitigation |
|---|---|---|
| **Indirect install via `make`/scripts** | OS-level `.npmrc` `ignore-scripts=true` persists regardless of invocation method | Environment-level config catches what hooks miss |
| **Edit `package.json` then `npm install`** | Lockfile enforcement: `npm ci` rejects manifest-lockfile drift | PostToolUse hook detects lockfile change; `enableImmutableInstalls` blocks update |
| **Bare `npm install` (no args)** | lockfile-integrity-research: `npm ci` is always preferred over `npm install` in existing projects | PreToolUse rewrite: `npm install` (no args) → `npm ci` |
| **Direct manifest editing** | `CODEOWNERS` on lockfiles + `lockfile-lint` in CI | PostToolUse hook running lockfile-lint catches manipulation |
| **Package aliasing** | Socket.dev behavioral analysis detects suspicious packages regardless of name | MCP `depscore` check uses actual package identity, not alias |
| **Nix: `nix-env -i`** | devenv-security: imperative installs bypass flake pinning entirely | Permission deny: `Bash(nix-env *)` |
| **Nix: `cachix use`** | devenv-security: untrusted caches are root-equivalent | Permission deny: `Bash(cachix use *)` |
| **Nix: modify `devenv.nix`** | devenv-security: Nix evaluation is unsandboxed | CLAUDE.md: "Never modify binary cache config or trusted-users"; code review as primary control |

---

## 10. Recommended Implementation Priority

Based on impact, cost, and coverage from both sibling spikes:

### Tier 1: Implement Immediately (High Impact, Low Cost)

1. **Environment-level safety defaults** — Configure `.npmrc`, `pip.conf`, `.yarnrc.yml`, `nix.conf` with secure defaults (ignore-scripts, only-binary, min-release-age, save-exact, sandbox-fallback=false, require-sigs=true). These work even if all other layers fail.

2. **PreToolUse command rewriting** — Rewrite unsafe commands to safe equivalents: `npm install` → `npm ci`, append `--ignore-scripts`, append `--only-binary :all:`, append `--locked`. The `updatedInput` hook capability makes this transparent to the agent.

3. **Permission deny rules for dangerous commands** — Deny `nix-env -i *`, `cachix use *`, `npm install --registry https://registry.npmjs.org*`. Force agent through approved wrappers.

4. **CLAUDE.md supply chain instructions** — Comprehensive instructions covering all findings above. These shape agent behavior at the intent level before commands are formed.

### Tier 2: Implement This Week (High Impact, Moderate Cost)

5. **PreToolUse hook with registry API queries** — Check publication age and known vulnerabilities before allowing install. The OSV.dev API is free and unlimited. npm/PyPI registry APIs are public.

6. **Socket.dev MCP integration** — Connect the Socket.dev MCP server (`mcp.socket.dev`) for `depscore` pre-install scoring. Already production-ready and free.

7. **PostToolUse lockfile integrity check** — After any install command, compare lockfile hashes and run lockfile-lint. Alert on unexpected changes.

### Tier 3: Implement When Needed (Medium Impact, Higher Cost)

8. **Custom MCP install wrapper** — Build a `safe-install` MCP tool that wraps all package managers with pre-flight checks and safety flags.

9. **Nix-specific MCP server** — Query nix-security-tracker, validate flake inputs, check vulnix results.

10. **Private registry enforcement** — For organizations with private registries: PreToolUse validation of registry URLs, deny rules for public registry overrides.

11. **Provenance soft-checks** — Query for Sigstore attestations as an informational signal. Harden to blocking when ecosystem adoption crosses ~50%.

---

## Sources

### package-supply-chain-security spike
- `quarantine-gates-research.md` — Age-gating mechanisms, detection statistics, per-ecosystem configuration
- `install-sandboxing-research.md` — Install script attack vectors, blocking mechanisms, sandbox tools
- `lockfile-integrity-research.md` — Lock file mechanics, hash verification, lockfile poisoning, CI enforcement
- `signature-provenance-research.md` — Sigstore, SLSA, per-ecosystem provenance status, consumer enforcement gap
- `private-registries-research.md` — 12 registry tools evaluated, security capabilities, age-gating support
- `org-tooling-research.md` — 10 scanning/monitoring tools, layered defense recommendations
- `attack-surface-landscape-research.md` — Per-ecosystem attack vectors and architectural security ranking

### devenv-security spike
- `architecture-research.md` — Devenv internals, trust model, evaluation architecture
- `security-surface-research.md` — 10 attack vectors, 25+ sub-vectors, threat model
- `nix-security-mechanisms-research.md` — 9 Nix security mechanisms with devenv interaction
- `config-options-research.md` — Complete configuration inventory, hardened boilerplate
- `prior-art-research.md` — Nix scanning tools, community practices, gap analysis
- `supply-chain-cross-ref-research.md` — Cross-reference between general supply chain and Nix-specific concerns

### claude-code-agent-package-guardrails spike (existing)
- `hooks-research.md` — PreToolUse mechanics, bypass vectors, handler types
- `permissions-research.md` — Deny/allow/ask rules, settings hierarchy, bypass mitigations
- `mcp-server-research.md` — Socket.dev + Snyk MCP servers, custom server architecture
