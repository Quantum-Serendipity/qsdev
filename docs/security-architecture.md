# Security Architecture

## Threat Model

### Adversary Goals

1. **Dependency confusion / typosquatting** — Trick the developer or AI agent into installing a malicious package with a name similar to a legitimate one.
2. **Supply chain compromise** — Inject malicious code into a legitimate package via account takeover, build system compromise, or CI action hijacking.
3. **Agent exploitation** — Use Claude Code's tool-calling capability to run arbitrary package installs, pipe-to-shell commands, or destructive operations.
4. **Credential exfiltration** — Read secrets from the developer's environment, `.env` files, or cloud provider credentials.
5. **Lockfile manipulation** — Modify lockfiles to redirect package resolution to attacker-controlled registries.

### Trust Boundaries

- **Developer workstation** — The devenv shell is the primary trust boundary. Clean mode strips credentials; pre-commit hooks scan for secrets.
- **Claude Code sandbox** — Ask rules, deny rules, and PreToolUse hooks constrain what the AI agent can execute. The permission model prevents bypassing these controls.
- **CI pipeline** — harden-runner constrains network egress; vulnerability scanners block known-bad dependencies.
- **Package registry** — Registry proxies (Nexus, Artifactory, GitHub Packages) provide a controlled ingestion point. pnpm workspace configuration enforces `strictDepBuilds` and `trustPolicy: no-downgrade`.

## Defense Layers

qsdev implements 10 layers of supply-chain defense.

| # | Layer | Catches |
|---|-------|---------|
| 1 | Age-gating | Zero-day package takeovers |
| 2 | Install script blocking | Arbitrary code at install time |
| 3 | Lockfile enforcement | Silent dependency redirection |
| 4 | Vulnerability scanning | Known CVEs in the dependency tree |
| 5 | PreToolUse hooks (package-guard) | AI agent adding unvetted packages |
| 6 | Nix hardening | Impure builds, environment leaks |
| 7 | SAST (Semgrep) | Dangerous code patterns in source |
| 8 | Secrets scanning (ripsecrets + gitleaks) | Credentials committed to source |
| 9 | Container security | Vulnerable base images, misconfigurations |
| 10 | License compliance | Non-permissive transitive dependencies |

### Layer 1: Age-Gating

New package versions are blocked for a configurable period after publication. This provides a window for the community to discover and report compromised releases before they enter your project.

| Infrastructure Profile | Minimum Release Age | Update Tool |
|------------------------|--------------------:|-------------|
| `consulting-default` | 3 days (4320 min) | Renovate |
| `startup-github` | None (0 days) | Dependabot |
| `enterprise` | 7 days | Renovate |

For pnpm workspaces, age-gating is additionally enforced at install time via `minimumReleaseAge: 4320` in `pnpm-workspace.yaml`.

### Layer 2: Install Script Blocking

Per-ecosystem configuration files disable install-time script execution — the single most exploited attack vector in package supply chains.

| Ecosystem | Config File | Key Setting |
|-----------|------------|-------------|
| JavaScript (npm) | `.npmrc` | `ignore-scripts=true` |
| JavaScript (yarn) | `.yarnrc.yml` | `enableScripts: false` |
| JavaScript (pnpm) | `.npmrc` + `pnpm-workspace.yaml` | `ignore-scripts=true`, `strictDepBuilds` |
| Python | `pip.conf` | `--no-deps` enforcement |
| Rust | `.cargo/config.toml` | Registry pinning |
| Ruby | `.bundle/config` | `BUNDLE_DISABLE_EXEC_LOAD: true` |
| PHP | `composer.json` config | Script restrictions |
| .NET | `nuget.config` | Source pinning |

pnpm workspaces additionally enforce `blockExoticSubdeps` to prevent subdependencies from pulling in unexpected transitive packages.

### Layer 3: Lockfile Enforcement

- **Pre-commit hooks** — The `lock-file-audit` custom hook flags changes to `devenv.lock`, `flake.lock`, `package-lock.json`, and `pnpm-lock.yaml` with a warning to verify the diff during code review.
- **CI** — Security scan workflows verify lockfile integrity as part of the build.
- **pnpm workspace** — `trustPolicy: no-downgrade` prevents lockfile changes that regress dependency versions.
- **CLAUDE.md rules** — Generated project documentation instructs Claude Code to never modify lockfiles without explicit approval.

### Layer 4: Vulnerability Scanning

| Scanner | Profiles | Integration |
|---------|----------|-------------|
| OSV-Scanner | `consulting-default`, `startup-github` | CI workflow + PreToolUse hook |
| Snyk | `enterprise` | CI workflow step |
| Socket.dev | All profiles | MCP server for behavioral analysis |

The package-guard hook (Layer 5) queries OSV.dev in real time when the AI agent requests a package install — blocking packages with known vulnerabilities before they enter the dependency tree.

### Layer 5: PreToolUse Hooks (package-guard)

The `package-guard` hook runs as a Claude Code PreToolUse interceptor on every `Bash` tool invocation:

1. **Pattern matching** — Detects install commands across all supported package managers.
2. **OSV.dev vulnerability check** — Queries the OSV API for known vulnerabilities in the requested package.
3. **Age-gate enforcement** — Rejects packages published less than the configured minimum release age.
4. **Allow or block** — Permits the install (with approval) if the package passes both checks; blocks it otherwise with an explanation.

Package install commands live in the `ask` list (not `deny`), meaning the hook gets a chance to validate them before the user sees a prompt. Only bypass vectors that cannot be safely validated remain in `deny`.

### Layer 6: Nix Hardening

The generated `devenv.yaml` enforces:

- **`impure: false`** — Prevents the build from accessing anything outside the Nix store.
- **`allow_unfree: false`** — Blocks unfree packages unless explicitly listed.
- **`allow_broken: false`** — Blocks broken packages.
- **`clean.enabled: true`** — Strips the shell environment on entry, keeping only a minimal allowlist (TERM, HOME, USER, SSH_AUTH_SOCK, etc.).

The generated `devenv.nix` additionally:

- **Unsets 38 credential-bearing variables** — AWS, GCP, Azure, GitHub, GitLab, Docker, database, secrets management, and generic API keys.
- **Sets `DEVENV_SECURITY_HARDENED=true`** — A sentinel flag verified by `devenv test`.
- **Installs security pre-commit hooks** — ripsecrets, check-added-large-files, no-commit-to-branch, check-merge-conflict, shellcheck, statix.
- **Installs custom hooks** — lock-file-audit and nix-secrets-check (detects hardcoded credentials in `.nix` files).

### Layer 7: SAST (Semgrep)

Semgrep runs as an AlwaysOn tool in the Claude Code environment, providing static analysis during development:

- Detects dangerous code patterns (command injection, path traversal, unsafe deserialization).
- Custom rule sets per ecosystem are included in the generated configuration.
- Also runs in CI via the generated security-scan workflow.

**OpenGrep** (opt-in via `qsdev enable opengrep`) adds 96 taint-focused rules targeting injection flaws, deserialization, and authentication bypasses across 7 frameworks: Next.js, FastAPI, Gin, NestJS, SvelteKit, Prisma, and Drizzle.

### Layer 8: Secrets Scanning (ripsecrets + gitleaks)

Two complementary scanners ensure credentials never reach the repository:

| Tool | Stage | Scope |
|------|-------|-------|
| ripsecrets | Pre-commit hook | Fast, low-false-positive scan on staged files |
| gitleaks | AlwaysOn tool + CI | Full-repo scan including git history |

Both are configured automatically during `qsdev init`. No manual setup required.

### Layer 9: Container Security

When a Dockerfile or Containerfile is detected, qsdev generates runtime-aware security configs for both Docker and Podman:

- **Hadolint configuration** — Linting rules for Dockerfile best practices (no `latest` tags, no root user, etc.).
- **Syft SBOM generation + Grype vulnerability scanning** — CI workflow steps that produce a software bill of materials and scan built images for OS and library vulnerabilities. (Trivy was removed after the March 2026 supply chain compromise.)
- **Base image pinning** — Generated Dockerfiles pin base images to digest, not tag.
- **Runtime-aware deny rules** — In Podman mode, Docker socket mount commands are blocked to prevent accidental privilege escalation.

### Layer 10: License Compliance

Generated CI configuration includes license scanning that:

- Detects non-permissive licenses (GPL, AGPL, SSPL) in transitive dependencies.
- Generates a license report as part of SBOM output.
- Blocks merges when policy-violating licenses are introduced (configurable per infrastructure profile).

## Hook Execution Isolation

Hooks run inside a sandboxed environment that restricts filesystem access, network, and syscalls. The sandbox degrades gracefully based on available kernel features:

| Tier | Isolation | Requires |
|------|-----------|----------|
| Full | Bubblewrap + Landlock + seccomp-BPF + cgroups v2 | Linux 5.13+, bwrap |
| BwrapWithoutLandlock | Bubblewrap + seccomp-BPF | Linux, bwrap |
| BwrapWithoutSeccomp | Bubblewrap namespaces only | Linux, bwrap |
| SystemdRun | systemd-run resource limits | systemd |
| Unsandboxed | No isolation (macOS, minimal Linux) | — |

Run `qsdev sandbox status` to see the active tier on your system. Five hook category profiles (linter, formatter, network-linter, generator, test-runner) control which resources each hook type can access.

## Security Spectrum Positioning

Development security exists on a spectrum from zero configuration to full lockdown. Each increment of security adds corresponding friction. qsdev is deliberately positioned at the optimal inflection point — the highest protection achievable before productivity costs become structural.

### The Eight-Tier Framework

| Tier | Name | Example Controls | Ongoing DX Cost |
|------|------|-----------------|----------------|
| 0 | No Security | Trust everything; no lockfiles; credentials in source | None |
| 1 | Basic Hygiene | Lockfiles committed; .gitignore; SSH keys | Negligible |
| 2 | Dependency Awareness | Vulnerability scanning; Dependabot/Renovate; SBOM | Minutes/week |
| 3 | Active Defense | Age-gating; install-script blocking; secrets scanning; SAST | 3–10s/commit |
| 4 | Environment Hardening | Nix hermetics; credential scrubbing; build sandboxing | None after setup |
| 5 | Agent-Aware Security | PreToolUse hooks; deny rules; MCP gating; self-protection | None after setup |
| 6 | Process Isolation | VM per project; ephemeral environments; network partitioning | 10–20% permanent |
| 7 | Full Lockdown | Air-gapped; HSMs; mandatory multi-person approval | 20–40% permanent |

`qsdev init` delivers **Tiers 2–5** in under two minutes.

### Why Tiers 4–5 (Not Higher)

Three constraints converge at qsdev's position:

**1. Threat model alignment.** The realistic threat surface for development teams — supply chain attacks (454K malicious packages/year), credential theft (28.6M secrets leaked in 2025), AI agent exploitation (73% vulnerable to prompt injection) — is fully addressed by Tiers 3–5. Tier 6–7 defenses protect against nation-state EM surveillance, physical infiltration, and classified-data handling — threats outside the model for commercial software teams.

**2. DX cost cliff.** Tiers 0–5 have manageable or zero ongoing costs (qsdev eliminates setup cost through generation). At Tier 6, costs become *structural* — VM boundaries impose 5–15 minute cold starts, 3–10x slower incremental builds, and eliminate GPU passthrough. These costs cannot be removed by better tooling because they are inherent to the isolation model.

**3. Diminishing marginal returns.** Each tier from 0→5 provides substantial, measurable security improvement (92% malware catch from age-gating; complete elimination of install-script attacks; fail-closed agent policy). Tier 5→6 adds negligible protection against realistic threats while imposing catastrophic productivity loss — equivalent to losing 1–4 developers on a 10-person team.

### The Configuration Cost Innovation

Traditional Tier 4–5 setup takes 2–5 days of a security engineer's time: researching per-ecosystem best practices, writing Nix configurations, crafting deny rules, implementing hooks, testing interactions. Most teams never attempt it — not because they disagree with the security value, but because the configuration cost is prohibitive.

qsdev eliminates the configuration barrier by generating correct, ecosystem-specific security configurations from a single command. The ongoing cost after generation is 3–10 seconds per commit (pre-commit hooks) — indistinguishable from a project without security hardening.

### Quantified Effectiveness

| Defense | Metric | Source |
|---------|--------|--------|
| Age-gating (24–72h) | 92% of PyPI malware caught within 24h | PyPI security reports |
| Install-script blocking | Eliminates #1 exploited npm attack vector | npm security advisories |
| Secrets scanning (ripsecrets) | 0.32s full-repo scan (95x faster than trufflehog) | Benchmark on Sentry repo |
| MCP datamarking + trust scoring | Attack success rate reduced from ~60% to <2% | MCP security research |
| Nix content-addressing | Every artifact verified by SHA-256 hash | Nix store guarantees |
| Policy evaluation | <50 microseconds per rule | Internal benchmarks |

### Comparison to Alternatives

| Tool/Approach | Tier Coverage | Gap vs. qsdev |
|---------------|---------------|---------------|
| npm audit / Snyk | 2–3 (partial) | No environment hardening, no agent security, no multi-ecosystem |
| Socket.dev | 3 (behavioral only) | No age-gating, no isolation, no agent controls |
| Dev Containers | 4 (isolation only) | No supply chain hardening, no agent awareness |
| Raw Nix | 4 (reproducibility only) | No security configuration, no ecosystem modules |
| Manual Claude Code hooks | 5 (partial) | No supply chain integration, no self-protection harness |

qsdev is the only tool spanning Tiers 2–5 across 27 ecosystems with integrated AI agent security.

## Permission Model

qsdev generates Claude Code permissions in `.claude/settings.json` using a two-tier model: **ask** rules (hook-gated, user-prompted) and **deny** rules (unconditionally blocked).

### Ask Rules (~60 rules)

Package install commands are placed in the `ask` list. When Claude Code attempts one, the PreToolUse package-guard hook validates the request (OSV check + age-gate) before the user sees an approval prompt. This allows legitimate installs while blocking dangerous ones.

Ask rules cover:

| Category | Examples |
|----------|----------|
| JS Package Managers | `npm install`, `yarn add`, `pnpm add`, `bun add` |
| Python | `pip install`, `uv add`, `pipx install` |
| Rust | `cargo add`, `cargo install` |
| Go | `go get`, `go install` |
| Ruby | `gem install`, `bundle add` |
| PHP | `composer require` |
| System | `nix profile install`, `apt install`, `brew install` |

### Deny Rules (~90 rules)

Commands that represent bypass vectors — ways to circumvent the hook-gating — remain unconditionally denied. These cannot be validated safely, so they are blocked outright.

| Category | Examples | Count |
|----------|----------|------:|
| Pipe-to-Shell | `curl \| bash`, `wget \| sh` | ~8 |
| Shell Wrapping | `bash -c *npm install*`, `sh -c *pip install*` | ~14 |
| Subprocess Escape | `python -c *subprocess*`, `node -e *child_process*` | ~9 |
| eval/xargs | `eval *npm install*`, `xargs cargo install` | ~7 |
| env/command Prefix | `env npm install`, `command pip install` | ~10 |
| sudo Prefix | `sudo npm install`, `sudo apt install` | ~8 |
| npx/bunx Execution | `npx <package>`, `bunx <package>` | ~6 |
| Destructive Ops | `git push --force`, `rm -rf /`, `Read(./.env)` | ~6 |
| Nix Bypass | `nix-env -i`, `cachix use` | ~8 |
| Uncategorized | Per-ecosystem edge cases | ~14 |

### Permission Presets

| Preset | Philosophy |
|--------|-----------|
| **minimal** | Read-only by default. Only `Read(*)` and basic build/test commands are allowed. Every write or edit requires approval. |
| **standard** | Productive development. `Read`, `Edit`, `Write`, `git`, build/test/lint, and Nix dev commands are allowed. Package installs are hook-gated (ask). Bypass vectors are denied. |
| **permissive** | Standard plus `make` and `docker` commands. For teams using Makefiles or Docker-based workflows. |
| **supply-chain-only** | Minimal permissions focused exclusively on supply chain defense. Deny rules and package-guard hook without broader development tooling permissions. |
| **custom** | Only explicitly configured allow/deny patterns. Full manual control. |

Select a preset during `qsdev init` or set it in `.qsdev.yaml`:

```yaml
claude:
  permission_preset: standard
```

### AlwaysOn Tools

The following tools are installed and available to Claude Code without per-invocation approval:

| Tool | Purpose |
|------|---------|
| semgrep | SAST scanning |
| gitleaks | Secrets detection |
| semble | Semantic code search |
| version-sentinel | Dependency version tracking |
| context7 MCP | Documentation context |
| github MCP | GitHub API access |
| socket MCP | Dependency behavioral analysis |
| agent-postmortem | Session analysis and learning |
| package-guard | PreToolUse hook for install validation |

## CI Security Integration

### Generated Workflows

Infrastructure profiles generate `.github/workflows/security-scan.yml` with:

- **harden-runner** (all profiles) — Restricts network egress from CI runners, preventing exfiltration of secrets.
- **OSV-Scanner / Snyk / Grype** — Scans dependencies for known vulnerabilities.
- **Semgrep** — SAST rules for the detected ecosystems.
- **gitleaks** — Full-repo secrets scan.

### Generated Update Configuration

- **Renovate** (`consulting-default`, `enterprise`) — `renovate.json` with `minimumReleaseAge`, `automergeType: "pr"` for patches (enterprise), and lockfile maintenance.
- **Dependabot** (`startup-github`) — `.github/dependabot.yml` with configured update schedules.

### SBOM Generation

- **Syft** (all profiles) — Generates software bill of materials.
- **Cosign** (`enterprise` only) — Signs the SBOM for supply chain attestation.

## Security Validation

The generated `devenv.nix` includes an `enterTest` script that verifies security controls:

```bash
devenv test
```

This validates:

1. Pre-commit hooks are installed in `.git/hooks/`
2. Credential variables (`AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, `VAULT_TOKEN`, `DATABASE_PASSWORD`) are not present in the environment
3. `ripsecrets` finds no secrets in tracked files
4. The `DEVENV_SECURITY_HARDENED` sentinel flag is set

Run `devenv test` in CI to continuously verify that security controls have not been disabled.

For a full security posture assessment including all 10 layers:

```bash
qsdev status
```

This outputs a score (0–100), letter grade, and per-layer breakdown showing which controls are active, degraded, or missing.

## Known Limitations

### Hook Bypass Vectors

- **Aliases and functions** — Shell aliases (`alias npm='npm'`) or functions that wrap install commands are not caught by pattern-based rules.
- **Encoded commands** — Base64-encoded or hex-encoded commands piped to decoders are not blocked.
- **Indirect execution via scripts** — Running a script file that internally calls install commands bypasses the PreToolUse hook.
- **New package managers** — Rules must be updated when new package managers emerge.

### Environment Hardening

- **Clean mode is advisory** — A determined developer can re-export stripped variables. The `devenv test` validation catches this retroactively but not in real-time.
- **Host Nix configuration** — `devenv.yaml` settings only apply within the devenv shell. The host system's `nix.conf` may allow impure builds.

### CI Limitations

- **Action pinning** — Generated workflows pin to major version tags (e.g., `@v4`), not commit SHAs. Consider pinning to SHAs for maximum supply chain security.
- **Self-hosted runners** — harden-runner's network egress controls are most effective on GitHub-hosted runners. Self-hosted runners may require additional network controls.

### Scope

- **Runtime dependencies** — The system hardens the development environment and CI pipeline. It does not scan or constrain runtime container images or deployed artifacts beyond build-time scanning.
- **Secret management** — Credential stripping prevents accidental exposure in the dev shell but does not replace a proper secret management system (Vault, AWS Secrets Manager, etc.).

## Further Reading

- [Configuration Reference](configuration-reference.md) — Every generated file, its purpose, and merge strategy
- [Team Onboarding](team-onboarding.md) — Infrastructure profiles, team policies, rollout playbook
- [Migration Guide](migration-guide.md) — Adding qsdev to existing projects with pre-existing configs
