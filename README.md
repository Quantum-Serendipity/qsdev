# qsdev

Security-hardened dev environments, generated from your existing project.

[![CI](https://github.com/Quantum-Serendipity/qsdev/actions/workflows/ci.yml/badge.svg)](https://github.com/Quantum-Serendipity/qsdev/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Quantum-Serendipity/qsdev)](https://github.com/Quantum-Serendipity/qsdev/releases)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-informational)]()

## The Problem

Setting up a new project means writing environment config, copying security settings from the last project, configuring pre-commit hooks, and setting AI agent permissions. Then you do it all again next time. Most of the security config never gets written at all because there's always real work to do instead.

qsdev detects your stack and generates a complete environment with supply-chain hardening, AI agent guardrails, and per-ecosystem security config. You get a working setup in about two minutes instead of building it by hand every time.

## Why qsdev

- Detects your stack across 27 ecosystems and generates a complete, working [devenv.sh](https://devenv.sh) environment
- Ships with 10 layers of supply-chain defense (age-gating, install-script blocking, lockfile enforcement, vuln scanning, SAST, secrets detection, etc.) configured out of the box
- Sets up Claude Code with deny rules, operation skills, hooks, and MCP servers so your AI agent can't `curl | sh` or install unvetted packages
- `qsdev check --auto-fix` repairs drifted configs and restores deleted files automatically
- Hooks run inside a sandbox (bubblewrap + landlock + seccomp) with graceful degradation on systems that lack kernel support
- `qsdev teardown` removes everything cleanly. The generated configs are standard files you own
- `qsdev status` gives you a real security score and grade, not a checkbox
- Commit `.qsdev.yaml` and teammates run `qsdev init --mode join` for identical environments
- `qsdev update` preserves your modifications via three-way merge
- 10 project profiles (`go-web`, `ts-fullstack`, `python-web`, etc.) and 3 infrastructure tiers (`consulting-default`, `startup-github`, `enterprise`)

## Quick Start

```bash
# Install (macOS / Linux — or download a binary from Releases)
curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh

# Generate a complete secure dev environment
cd your-project
qsdev init --yes
```

## Try It Out

### On an existing project (non-destructive)

Use `qsdev trial` to evaluate in an isolated git worktree — zero risk to your working branch:

```bash
cd your-project
qsdev trial
```

This creates a worktree with the full qsdev configuration applied. Happy with it? Merge the branch. Not for you? Delete the worktree — zero residue.

### On our example project

Follow the [Critter Queue end-to-end runbook](docs/e2e-runbook-critter-queue.md), which walks through the full qsdev lifecycle on a TypeScript + PostgreSQL + Redis project.

After running, your project has a working environment:

```
devenv.nix                  # Deterministic environment (languages, services, packages)
devenv.yaml                 # Environment inputs
.envrc                      # Automatic shell activation
.pre-commit-config.yaml     # Linting, formatting, lockfile enforcement
.claude/settings.json       # AI agent permissions and deny rules
.claude/hooks/package-guard.py  # Package install interception
.claude/skills/             # Operation skills for AI-assisted workflows
.claude/rules/              # Language-specific convention rules
.qsdev/policy.nix           # Hook sandbox policies
.mcp.json                   # MCP server configuration
CLAUDE.md                   # Project context for AI agents
.npmrc / pip.conf / ...     # Per-ecosystem security configs
.syft.yaml / .grype.yaml    # SBOM + vulnerability scanner configs
.gitignore                  # Updated entries
```

<details>
<summary>Other installation methods</summary>

### Homebrew (macOS / Linux)

```bash
brew install Quantum-Serendipity/tap/qsdev
```

### Nix flake

```bash
nix profile install github:Quantum-Serendipity/qsdev
```

Or run without installing:

```bash
nix run github:Quantum-Serendipity/qsdev -- init
```

### NixOS module

```nix
{
  inputs.qsdev.url = "github:Quantum-Serendipity/qsdev";
  # In your configuration:
  imports = [ qsdev.nixosModules.default ];
}
```

### go install

```bash
go install github.com/Quantum-Serendipity/qsdev/cmd/qsdev@latest
```

### Scoop (Windows)

```powershell
scoop bucket add qsdev https://github.com/Quantum-Serendipity/scoop-bucket
scoop install qsdev
```

### Binary downloads

Pre-built binaries for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64) are available on the [Releases page](https://github.com/Quantum-Serendipity/qsdev/releases).

</details>

The installer handles all dependencies automatically — no manual setup required.

## Supported Ecosystems

| Ecosystems | Coverage |
|-----------|----------|
| Go, JavaScript/TypeScript, Python, Rust, Java/Kotlin, .NET, Containers, Terraform | Full supply-chain hardening |
| PHP, Ruby, Scala, C/C++, Shell, Helm, Ansible | Security configs + deny rules |
| Elixir, Dart, Swift, Haskell, Clojure, Bazel, Nix, Perl, R, Lua, Zig, PowerShell | Packages + deny rules |

## Integrations

| Tool | How qsdev uses it |
|------|-------------------|
| [devenv.sh](https://devenv.sh) | qsdev generates configs; devenv runs the environment |
| [Claude Code](https://claude.ai/code) | Permissions, deny rules, skills, hooks, MCP configs |
| [Nix](https://nixos.org) | Reproducible, hermetic package resolution |
| [direnv](https://direnv.net) | Auto-activates the environment on `cd` |
| [pre-commit](https://pre-commit.com) | Lockfile checks, formatting, linting hooks |
| [Socket.dev](https://socket.dev) | Behavioral supply chain analysis via MCP |
| [Podman](https://podman.io) | Rootless container runtime (auto-detected alongside Docker) |

## Commands

```
qsdev init                    # Generate complete secure environment
qsdev init --profile go-web   # Use a project-type preset
qsdev status                  # Security posture assessment (score + grade)
qsdev check                   # CI enforcement (config integrity, hardening)
qsdev check --auto-fix        # Fix drifted configs automatically
qsdev update                  # Update configs + devenv inputs
qsdev repair                  # Fix corrupted or drifted files
qsdev teardown                # Remove all qsdev configuration (clean exit)
qsdev enable <tool>           # Enable a security/AI tool
qsdev disable <tool>          # Disable a tool
qsdev list                    # Show all available tools
qsdev trial                   # Evaluate in an isolated git worktree
```

<details>
<summary>Full command reference</summary>

### Top-level

| Command | Description |
|---------|-------------|
| `init` | Generate complete secure environment (wizard + detection + generation) |
| `status` | Security posture assessment with score and grade |
| `check` | CI enforcement checks (JSON, SARIF, JUnit output). `--auto-fix` repairs issues |
| `info` | Project status at a glance (cached, instant) |
| `repair` | Fix corrupted or drifted config files |
| `update` | Update binary + configs + devenv inputs |
| `outdated` | Check for outdated dependencies across ecosystems |
| `teardown` | Remove all qsdev configuration from project |
| `enable <tool>` | Enable a tool |
| `disable <tool>` | Disable a tool |
| `list` | List all available tools |
| `evidence` | Generate compliance evidence (SOC2, HIPAA, ASVS) |
| `team-report` | Aggregate posture across multiple projects |
| `trial` | Evaluate qsdev in an isolated git worktree |
| `scaffold-instance` | Create a white-label fork of qsdev |
| `self-update` | Update the qsdev binary to the latest release |
| `completion` | Generate shell completions (bash, zsh, fish, powershell) |

### devenv subcommands

| Command | Description |
|---------|-------------|
| `devenv doctor` | Diagnose environment issues |
| `devenv setup` | Install prerequisites (Nix, devenv, direnv) |
| `devenv add-language <name>` | Add a language ecosystem |
| `devenv add-service <name>` | Add a service (postgres, redis, etc.) |
| `devenv add-package <name>` | Add system packages |
| `devenv add-overlay <path>` | Add a Nix overlay |
| `devenv remove-language/service/package/overlay` | Remove components |
| `devenv changelog` | Generate changelog with git-cliff |

### claude subcommands

| Command | Description |
|---------|-------------|
| `claude init` | Initialize Claude Code config independently |
| `claude add-skill <name>` | Add a skill |
| `claude add-hook <name>` | Enable a hook preset |
| `claude list-skills` | List available skills |
| `claude hooks list` | List registered hooks with deployment tier and status |

### sandbox subcommands

| Command | Description |
|---------|-------------|
| `sandbox exec -- CMD` | Run a command inside the hook sandbox |
| `sandbox status` | Display sandbox capabilities and degradation tier |

### container subcommands

| Command | Description |
|---------|-------------|
| `container detect` | Detect active container runtime and capabilities |
| `container migrate` | Analyze compose files for Docker-to-Podman compatibility. `--auto-fix` applies fixes |

</details>

## Project Profiles

Pre-configured bundles for common project types:

| Profile | Languages | Services | Security |
|---------|-----------|----------|----------|
| `go-web` | Go | PostgreSQL, Redis | Standard + safety-block |
| `ts-fullstack` | TypeScript (pnpm) | PostgreSQL, Redis | Standard + auto-format |
| `ts-backend` | TypeScript (pnpm) | PostgreSQL, Redis | Standard |
| `python-data` | Python (uv) | — | Minimal |
| `python-web` | Python (uv) | PostgreSQL, Redis | Standard |
| `rust-cli` | Rust | — | Minimal + pre-commit |
| `rust-web` | Rust | PostgreSQL, Redis | Standard |
| `java-web` | Java (Gradle) | PostgreSQL, Redis | Standard |
| `elixir-web` | Elixir | PostgreSQL, Redis | Standard |
| `dotnet-web` | .NET | PostgreSQL, Redis | Standard |

Create custom profiles via `.qsdev.yaml` — combine any languages, services, and infrastructure tier.

Infrastructure profiles control organization-wide policy:

| Profile | Focus |
|---------|-------|
| `consulting-default` | Nexus proxy, OSV + Socket scanning, Renovate with 3-day age gate, Syft SBOM |
| `startup-github` | GitHub Packages, OSV + Socket scanning, Dependabot, Turborepo |
| `enterprise` | Artifactory, Snyk + Socket scanning, Renovate with 7-day age gate, Cosign SBOM signing |

## What qsdev is NOT

qsdev generates configuration files. It doesn't:

- Run your environment ([devenv.sh](https://devenv.sh) does that)
- Manage runtime versions (Nix handles that declaratively)
- Run tasks (use Make, Just, or devenv tasks)
- Run containers or deploy anything (container commands analyze and migrate configs, not start services)
- Scaffold application code
- Configure your entire IDE (just `.editorconfig` and VS Code extension recs)

## Built On

qsdev is built on [gdev](https://github.com/fastcat/gdev), a developer experience framework created by Matthew Gabeler-Lee. I've used it for years and always miss it when I can't. Thanks for building such a great tool.

## Build Your Own

qsdev is also a white-label framework. You can fork it, rebrand it, and ship your own `acmedev` with your company's security policies baked in:

```bash
qsdev scaffold-instance acmedev --github-owner acme-corp
cd acmedev && go mod tidy && go build ./cmd/acmedev
./acmedev --help
```

From there, add proprietary addons, wire in registry proxies, or enforce custom compliance profiles.

**[Full guide: Build Your Own *dev Tool](docs/build-your-own.md)**

## Documentation

- [Security Architecture](docs/security-architecture.md) — Threat model, defense layers, permission model
- [Configuration Reference](docs/configuration-reference.md) — Generated files and merge strategies
- [Team Onboarding](docs/team-onboarding.md) — Profiles, policies, team rollout
- [Migration Guide](docs/migration-guide.md) — Adding qsdev to existing projects
- [Build Your Own](docs/build-your-own.md) — Fork and rebrand qsdev as your own tool

## License

[Apache-2.0](LICENSE)

Copyright 2024–2026 Quantum Serendipity Software.
