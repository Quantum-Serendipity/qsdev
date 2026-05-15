# Security Architecture

This document describes the threat model, defense layers, permission model, and known limitations of the security hardening provided by qsdev.

## Threat Model

### Adversary Goals

1. **Dependency confusion / typosquatting** -- Trick the developer or AI agent into installing a malicious package with a name similar to a legitimate one.
2. **Supply chain compromise** -- Inject malicious code into a legitimate package via account takeover, build system compromise, or CI action hijacking.
3. **Agent exploitation** -- Use Claude Code's tool-calling capability to run arbitrary package installs, pipe-to-shell commands, or destructive operations.
4. **Credential exfiltration** -- Read secrets from the developer's environment, `.env` files, or cloud provider credentials.
5. **Lockfile manipulation** -- Modify lockfiles to redirect package resolution to attacker-controlled registries.

### Trust Boundaries

- **Developer workstation** -- The devenv shell is the primary trust boundary. Clean mode strips credentials; pre-commit hooks scan for secrets.
- **Claude Code sandbox** -- Deny rules and PreToolUse hooks constrain what the AI agent can execute. The permission model prevents bypassing these controls.
- **CI pipeline** -- harden-runner constrains network egress; vulnerability scanners block known-bad dependencies.
- **Package registry** -- Registry proxies (Nexus, Artifactory, GitHub Packages) provide a controlled ingestion point.

## Defense Layers

### Layer 1: Age-Gating

New package versions are blocked for a configurable period (3 days for `consulting-default`, 7 days for `enterprise`). This provides a window for the community to discover and report compromised releases.

- **Renovate**: Configured with `minimumReleaseAge` in the generated `renovate.json`.
- **Dependabot**: Uses the `open-pull-requests-limit` setting to slow automated updates.

### Layer 2: Install Script Blocking

Per-ecosystem security configuration files disable install-time script execution:

| Ecosystem | Config File | Key Setting |
|-----------|------------|-------------|
| JavaScript (npm) | `.npmrc` | `ignore-scripts=true` |
| JavaScript (yarn) | `.yarnrc.yml` | `enableScripts: false` |
| JavaScript (pnpm) | `.npmrc` | `ignore-scripts=true` |
| Python | `pip.conf` | `--no-deps` enforcement via CLAUDE.md rules |
| Rust | `.cargo/config.toml` | Registry pinning |
| Ruby | `.bundle/config` | `BUNDLE_DISABLE_EXEC_LOAD: true` |
| PHP | `composer.json` config | `process-timeout: 0`, script restrictions |
| .NET | `nuget.config` | Source pinning |

### Layer 3: Lockfile Enforcement

- **Pre-commit hooks**: The `lock-file-audit` custom hook flags changes to `devenv.lock` and `flake.lock` with a warning to verify the diff during code review.
- **CI**: Security scan workflows verify lockfile integrity as part of the build.
- **CLAUDE.md rules**: Generated project documentation instructs Claude Code to never modify lockfiles without explicit approval.

### Layer 4: Vulnerability Scanning

Configured via infrastructure profiles:

| Scanner | Profiles | Integration |
|---------|----------|-------------|
| OSV-Scanner | `consulting-default`, `startup-github` | CI workflow step |
| Snyk | `enterprise` | CI workflow step |
| Socket.dev | All profiles | Behavioral analysis of dependencies |

The generated `.github/workflows/security-scan.yml` runs on every push and pull request.

### Layer 5: PreToolUse Hooks (Claude Code)

The `package-guard.py` hook (installed by the `safety-block` hook preset) runs as a Claude Code PreToolUse interceptor:

1. **Pattern matching** -- Checks every `Bash` tool invocation against install command patterns.
2. **Allowlist check** -- Permits known-safe commands (build, test, lint).
3. **Block or warn** -- Blocks package install commands and provides the developer with a safe alternative.

This is the runtime complement to the static deny rules in `settings.json`. The hook catches commands that might slip through pattern-based deny rules via creative shell escaping.

### Layer 6: Nix Hardening

The generated `devenv.yaml` enforces:

- **`impure: false`** -- Prevents the build from accessing anything outside the Nix store.
- **`allow_unfree: false`** -- Blocks unfree packages unless explicitly listed.
- **`allow_broken: false`** -- Blocks broken packages.
- **`clean.enabled: true`** -- Strips the shell environment on entry, keeping only a minimal allowlist (TERM, HOME, USER, SSH_AUTH_SOCK, etc.).

The generated `devenv.nix` additionally:

- **Unsets 50+ credential-bearing variables** -- AWS, GCP, Azure, GitHub, GitLab, Docker, database, secrets management, and generic API keys.
- **Sets `DEVENV_SECURITY_HARDENED=true`** -- A sentinel flag verified by `devenv test`.
- **Installs security pre-commit hooks** -- ripsecrets, check-added-large-files, no-commit-to-branch, check-merge-conflict, shellcheck, statix.
- **Installs custom hooks** -- lock-file-audit and nix-secrets-check (detects hardcoded credentials in `.nix` files).

For system-level Nix configuration recommendations, see [nix-conf-hardening.md](nix-conf-hardening.md) (generated with `--nix-hardening-guide`).

## Permission Model

### Claude Code Deny Rules

The system generates deny rules in `.claude/settings.json` across 15 categories:

| Category | Examples | Count |
|----------|----------|-------|
| JS Package Managers | `npm install`, `yarn add`, `pnpm install`, `bun add` | ~30 |
| Python | `pip install`, `uv add`, `pipx install` | ~18 |
| Rust | `cargo add`, `cargo install` | 3 |
| Go | `go get`, `go install` | 2 |
| Ruby | `gem install`, `bundle install/add/update` | ~6 |
| PHP | `composer require/install/update` | ~5 |
| Nix | `nix-env -i`, `nix profile install`, `cachix use` | ~8 |
| System | `apt install`, `brew install`, `pacman -S`, etc. | ~12 |
| Pipe-to-Shell | `curl \| bash`, `wget \| sh` | 8 |
| Shell Wrapping | `bash -c *npm install*`, `sh -c *pip install*` | ~14 |
| env/command Prefix | `env npm install`, `command pip install` | ~10 |
| sudo Prefix | `sudo npm install`, `sudo apt install` | ~8 |
| Subprocess Escape | `python -c *subprocess*`, `node -e *child_process*` | ~9 |
| eval/xargs | `eval *npm install*`, `xargs cargo install` | ~7 |
| Destructive Ops | `git push --force`, `rm -rf`, `Read(./.env)` | ~6 |

Each ecosystem module also contributes module-specific deny rules on top of these base rules.

### Permission Presets

| Preset | Philosophy |
|--------|-----------|
| **minimal** | Read-only by default. Only `Read(*)` and basic build/test commands are allowed. Every write or edit requires approval. |
| **standard** | Productive development. `Read`, `Edit`, `Write`, `git`, build/test/lint, and Nix dev commands are allowed. Package installs are denied. Bypass mode is disabled. |
| **permissive** | Standard plus `make` and `docker` commands. For teams that use Makefiles or Docker-based workflows. |
| **custom** | Only explicitly configured allow/deny patterns. Full manual control for advanced use cases. |

### Sandbox Configuration

When sandbox mode is enabled via the Go API, additional filesystem and network restrictions apply:

- **Write deny**: `/etc`, `/usr` (prevents system modification)
- **Network allow**: Configurable domain allowlist

## CI Security Integration

### Generated Workflows

Infrastructure profiles generate `.github/workflows/security-scan.yml` with:

- **harden-runner** (all profiles) -- Restricts network egress from CI runners, preventing exfiltration of secrets.
- **OSV-Scanner / Snyk / Grype** -- Scans dependencies for known vulnerabilities.

### Generated Update Configuration

- **Renovate** (`consulting-default`, `enterprise`) -- `renovate.json` with `minimumReleaseAge`, `automergeType: "pr"` for patches (enterprise), and lockfile maintenance.
- **Dependabot** (`startup-github`) -- `.github/dependabot.yml` with configured update schedules.

### SBOM Generation

- **Syft** (all profiles) -- Generates software bill of materials.
- **Cosign** (`enterprise` only) -- Signs the SBOM for supply chain attestation.

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

## Known Limitations

### Deny Rule Bypass Vectors

- **Aliases and functions** -- Shell aliases (`alias npm='npm'`) or functions that wrap install commands are not caught by pattern-based deny rules.
- **Encoded commands** -- Base64-encoded or hex-encoded commands piped to decoders are not blocked.
- **Indirect execution via scripts** -- Running a script file that internally calls install commands bypasses the PreToolUse hook.
- **New package managers** -- Deny rules must be updated when new package managers emerge (e.g., a new Rust package manager).

### Environment Hardening

- **Clean mode is advisory** -- A determined developer can re-export stripped variables. The `enterTest` validation catches this retroactively but not in real-time.
- **Host Nix configuration** -- `devenv.yaml` settings only apply within the devenv shell. The host system's `nix.conf` may allow impure builds. Generate and apply `nix-conf-hardening.md` for system-level hardening.

### CI Limitations

- **Action pinning** -- Generated workflows pin to major version tags (e.g., `@v4`), not commit SHAs. Consider pinning to SHAs for maximum supply chain security.
- **Self-hosted runners** -- harden-runner's network egress controls are most effective on GitHub-hosted runners. Self-hosted runners may require additional network controls.

### Scope

- **Runtime dependencies** -- The system hardens the development environment and CI pipeline. It does not scan or constrain runtime container images or deployed artifacts.
- **Secret management** -- Credential stripping prevents accidental exposure in the dev shell but does not replace a proper secret management system (Vault, AWS Secrets Manager, etc.).
