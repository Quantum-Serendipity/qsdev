# qsdev

Three qsdev addons (`devenv`, `claudecode`, `devinit`) that generate a fully configured, security-hardened development environment from a single command. Covers 27 language/platform ecosystems with defense-in-depth against supply chain attacks, including age-gating, lockfile enforcement, vulnerability scanning, and Claude Code guardrails.

## Quick Start

### Prerequisites

- [gdev](https://github.com/fastcat/gdev) installed
- [devenv.sh](https://devenv.sh) installed
- Nix with flakes enabled
- (optional) [direnv](https://direnv.net) for automatic shell activation

### 30-Second Path

Auto-detect languages, accept all defaults, generate everything:

```bash
cd my-project && qsdev init --yes
```

### With a Profile

Apply a preconfigured project-type profile:

```bash
qsdev init --profile go-web --yes
```

### Fully Custom

```bash
qsdev init \
  --lang go,javascript \
  --service postgres \
  --claude-permissions standard \
  --mcp github \
  --claude-skills deploy,security-review \
  --claude-hooks safety-block,pre-commit
```

## Command Reference

### `qsdev init`

The unified orchestrator command. Runs detection, wizard, and both generators.

#### Core Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | (detected) | Languages to configure (e.g. `go,javascript,python`) |
| `--service` | (none) | Services to configure (e.g. `postgres,redis`) |
| `-y, --yes` | `false` | Accept all defaults, skip confirmation prompts |
| `--force` | `false` | Overwrite existing configuration files |
| `--dry-run` | `false` | Preview changes without writing files |
| `--update` | `false` | Regenerate files from saved config, preserving user modifications |
| `--devenv-only` | `false` | Only generate devenv configuration (skip Claude Code) |
| `--claude-only` | `false` | Only generate Claude Code configuration (skip devenv) |
| `--profile` | (none) | Project-type profile name (e.g. `go-web`, `ts-fullstack`) |

#### Language-Specific Flags

| Flag | Description |
|------|-------------|
| `--go-version` | Go version (e.g. `1.24`) |
| `--node-version` | Node.js version (e.g. `22`) |
| `--node-pkg-mgr` | Node package manager (`npm`, `pnpm`, `yarn`, `bun`) |
| `--python-version` | Python version (e.g. `3.12`) |
| `--python-pkg-mgr` | Python package manager (`pip`, `uv`, `poetry`) |
| `--rust-channel` | Rust channel (`stable`, `beta`, `nightly`) |
| `--java-version` | Java version (e.g. `21`) |
| `--java-build-tool` | Java build tool (`maven`, `gradle`) |

Language-specific flags implicitly add their language to the `--lang` list if not already present.

#### Dev Environment Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--direnv` | `true` | Enable direnv integration |
| `--git-hooks` | (none) | Git hooks to configure (e.g. `pre-commit,pre-push`) |
| `--packages` | (none) | Extra Nix packages to include (e.g. `jq,ripgrep`) |
| `--env` | (none) | Environment variables as `KEY=VALUE` pairs |
| `--nix-hardening-guide` | `false` | Generate Nix security hardening guide |
| `--infra-profile` | (none) | Infrastructure profile name (e.g. `consulting-default`) |

#### Claude Code Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--claude-code` | `true` | Enable Claude Code configuration |
| `--claude-permissions` | `standard` | Permission preset (`minimal`, `standard`, `permissive`, `custom`) |
| `--claude-skills` | (none) | Skills to install (e.g. `deploy,review-pr`) |
| `--claude-hooks` | (none) | Hook presets to enable (e.g. `safety-block,auto-format`) |
| `--mcp` | (none) | MCP servers to configure (e.g. `github,filesystem`) |
| `--list-profiles` | `false` | List available project-type profiles and exit |

#### Mutual Exclusions

- `--devenv-only` and `--claude-only` cannot be used together
- `--update` cannot be combined with `--lang`, `--service`, or `--profile`

### `qsdev devenv`

Manage devenv.sh configuration independently.

| Subcommand | Description |
|------------|-------------|
| `init` | Initialize a security-hardened devenv environment (`devenv.yaml`, `devenv.nix`, `.envrc`) |
| `update` | Regenerate devenv files from saved answers |
| `add-service <name>` | Add a service (`postgres`, `redis`, `mysql`, `mongodb`, `elasticsearch`, `rabbitmq`) |
| `add-language <name>` | Add a language ecosystem module to the devenv configuration |

### `qsdev claude`

Manage Claude Code configuration independently.

| Subcommand | Description |
|------------|-------------|
| `init` | Initialize Claude Code configuration (`.claude/settings.json`, `CLAUDE.md`, hooks, skills) |
| `update` | Regenerate Claude Code files from saved answers |
| `add-skill <name>` | Add a skill from the built-in library |
| `add-hook <name>` | Enable a hook preset (`auto-format`, `safety-block`, `pre-commit`, `audit-log`) |
| `list-skills` | List available skills and show which are installed |

## Project-Type Profiles

Pre-configured bundles for common project archetypes. Use `qsdev init --list-profiles` to see all available profiles.

| Profile | Languages | Services | Permission Level | Skills |
|---------|-----------|----------|-----------------|--------|
| `go-web` | Go 1.24 | PostgreSQL, Redis | standard | deploy, security-review |
| `ts-fullstack` | JavaScript (pnpm) | PostgreSQL, Redis | standard | deploy |
| `python-data` | Python 3.12 (uv) | (none) | minimal | security-review |
| `rust-cli` | Rust | (none) | minimal | security-review |

All project-type profiles enable direnv and Claude Code. Hook presets vary by profile:

- **go-web**: safety-block, pre-commit
- **ts-fullstack**: auto-format, safety-block, pre-commit
- **python-data**: safety-block
- **rust-cli**: safety-block, pre-commit

## Infrastructure Profiles

Organization-wide infrastructure choices applied via `--infra-profile`.

| Profile | Registry | Nix Cache | Build Cache | Scanner | Update Tool |
|---------|----------|-----------|-------------|---------|-------------|
| `consulting-default` | Nexus | Cachix | sccache/S3 | OSV + Socket | Renovate (3-day age gate) |
| `startup-github` | GitHub Packages | Cachix | Turborepo | OSV + Socket | Dependabot |
| `enterprise` | Artifactory | Cachix | sccache/S3 | Snyk + Socket | Renovate (7-day age gate) |

The `enterprise` profile also generates SBOM with Cosign signing. All profiles include harden-runner CI protection.

## Supported Ecosystems

27 language and platform modules organized into tiers:

| Tier | Ecosystems |
|------|------------|
| **Tier 1** (full supply chain hardening) | Go, JavaScript/TypeScript, Python, Rust, Java/Kotlin, C#/.NET, Docker, Terraform/OpenTofu |
| **Tier 2** (security configs + deny rules) | PHP, Ruby, Scala, C/C++, Helm, Ansible |
| **Tier 3** (packages + deny rules) | Bash/Shell, Elixir, Dart/Flutter, Swift, Haskell, Clojure, Bazel, Nix |
| **Tier 4** (packages only) | Perl, R, Lua, Zig, PowerShell |

Each module contributes Nix packages, pre-commit hooks, per-ecosystem security configuration files (e.g. `.npmrc`, `pip.conf`, `.cargo/config.toml`), and Claude Code deny rules appropriate to its ecosystem.

## Security Overview

The system implements six defense layers:

1. **Age-gating** -- Block packages newer than a configurable threshold (3 or 7 days) to avoid dependency confusion and typosquatting.
2. **Install script blocking** -- Per-ecosystem security configs disable install-time script execution (e.g. `ignore-scripts=true` in `.npmrc`).
3. **Lockfile enforcement** -- Pre-commit hooks flag lockfile changes; CI workflows verify lockfile integrity.
4. **Vulnerability scanning** -- OSV, Snyk, or Grype scans integrated into CI workflows.
5. **PreToolUse hooks** -- Claude Code `package-guard.py` hook intercepts package install commands at runtime.
6. **Nix hardening** -- Clean environment stripping 50+ credential variables, `impure=false`, empty unfree/insecure allowlists.

Additionally, Claude Code deny rules block 150+ patterns across 15 categories including package managers, pipe-to-shell, shell wrapping, subprocess escapes, `eval`/`xargs`, destructive operations, and `sudo` prefixes.

For the full threat model and architectural details, see [docs/security-architecture.md](docs/security-architecture.md).

## Update Workflow

After initial setup, regenerate configuration to pick up template and skill library upgrades:

```bash
qsdev init --update
```

The update process:

1. Loads saved answers from `.devinit/.qsdev-init-answers.yaml`
2. Re-runs project detection to pick up new files
3. Compares each generated file against stored state to detect user modifications
4. Applies the appropriate merge strategy per file (see [docs/configuration-reference.md](docs/configuration-reference.md)):
   - **Unmodified files** are regenerated in place
   - **Modified files** are handled by their merge strategy (three-way merge, section markers, sidecar, or skip)
   - **Deleted files** are not recreated unless `--force` is used
5. Prints a summary of version changes in templates and skill libraries

Use `--dry-run` to preview what would change before applying, or `--force` to overwrite all files regardless of modification status.

## Further Reading

- [Team Onboarding Guide](docs/team-onboarding.md) -- Choosing profiles, configuring policies, rolling out to a team
- [Security Architecture](docs/security-architecture.md) -- Threat model, defense layers, permission model, known limitations
- [Configuration Reference](docs/configuration-reference.md) -- Every generated file, its merge strategy, and its contents
- [Migration Guide](docs/migration-guide.md) -- Adding qsdev to existing projects with pre-existing configuration
