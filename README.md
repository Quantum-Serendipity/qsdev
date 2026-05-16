# qsdev

**One command replaces 30–90 minutes of manual dev environment setup — with security hardening included.**

[![CI](https://github.com/Quantum-Serendipity/qsdev/actions/workflows/ci.yml/badge.svg)](https://github.com/Quantum-Serendipity/qsdev/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Quantum-Serendipity/qsdev)](https://github.com/Quantum-Serendipity/qsdev/releases)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-informational)]()

<!-- TODO: Terminal recording of `qsdev init --profile ts-fullstack --yes` -->

## The Problem

Every new project starts with the same ritual: write environment config from scratch, copy security settings from the last project, remember which pre-commit hooks you need, figure out the right AI agent permissions. It takes 30–90 minutes of yak-shaving before you write a single line of real code — and you do it every time.

Meanwhile, supply chain attacks are accelerating. A new malicious package lands on npm or PyPI every few hours. Install-time code execution is the single most exploited vector — and most ecosystems still allow it by default. AI agents are adding dependencies, running commands, and making unreviewed trust decisions on your behalf. Every `npm install` an agent runs is an attack surface you didn't audit.

The usual answer is "we'll add security later." Later never comes — or it comes after the incident.

qsdev eliminates the ritual and makes security the default, not the afterthought. One command generates a deterministic environment with 10 layers of supply-chain defense, AI agent guardrails, and per-ecosystem hardening — so you ship fast without shipping vulnerabilities.

## Before vs. After

| Manual setup | With qsdev |
|---|---|
| Write 50–200 lines of environment config from scratch | `qsdev init` |
| Research and configure per-ecosystem security for every package manager | generated |
| Set up pre-commit hooks for linting, formatting, and lockfile enforcement | generated |
| Manually write 150+ deny rules to stop AI agents from running dangerous commands | generated |
| Configure MCP servers for AI-assisted workflows | generated |
| Write project documentation that AI agents can actually use | generated |
| Set up package install interception hooks | generated |
| Add vulnerability scanning, secret detection, and SAST | generated |
| Wire up age-gating to block packages published less than 24 hours ago | generated |
| **Time: 30–90 minutes per project** | **Under 2 minutes** |

## Why qsdev

- **Instant setup** — detects your stack across 27 ecosystems and generates a complete, working environment in under two minutes
- **Secure by default** — 10 defense layers (age-gating, install-script blocking, lockfile enforcement, vulnerability scanning, SAST, secrets detection, and more) configured automatically — not bolted on later
- **AI-agent-ready** — Claude Code permissions, 150+ deny rules, 11 operation skills, hooks, and MCP servers from day one
- **Zero lock-in** — don't like it? `qsdev teardown` removes everything cleanly. Generated configs are standard files you own
- **Provable, not promissory** — `qsdev status` shows your security posture with a real score and grade, not a checkbox
- **Team-reproducible** — commit `.qsdev.yaml`, teammates run `qsdev init --mode join` for identical environments
- **Non-destructive updates** — `qsdev update` preserves your modifications via three-way merge
- **Profile-driven** — `go-web`, `ts-fullstack`, `python-data`, `rust-cli` project presets; `consulting-default`, `startup-fast`, `enterprise` infrastructure tiers

## Quick Start

```bash
# Install (macOS / Linux — or download a binary from Releases)
curl -fsSL https://raw.githubusercontent.com/Quantum-Serendipity/qsdev/main/scripts/install.sh | sh

# Generate a complete secure dev environment
cd your-project
qsdev init --yes
```

After running, your project has a complete security-hardened environment:

```
devenv.nix                  # Deterministic environment (languages, services, packages)
devenv.yaml                 # Environment inputs
.envrc                      # Automatic shell activation
.pre-commit-config.yaml     # Linting, formatting, lockfile enforcement
.claude/settings.json       # AI agent permissions + 150+ deny rules
.claude/hooks/package-guard.py  # Real-time package install interception
.claude/skills/             # 11 operation skills for AI-assisted workflows
.claude/rules/              # Language-specific convention rules
.mcp.json                   # MCP server configuration
CLAUDE.md                   # Project context for AI agents
.npmrc / pip.conf / ...     # Per-ecosystem security configs
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
| Go, JavaScript/TypeScript, Python, Rust, Java/Kotlin, .NET, Docker, Terraform | Full supply-chain hardening |
| PHP, Ruby, Scala, C/C++, Helm, Ansible | Security configs + deny rules |
| Shell, Elixir, Dart, Swift, Haskell, Clojure, Bazel, Nix | Packages + deny rules |
| Perl, R, Lua, Zig, PowerShell | Packages only |

## Integrations

| Tool | How qsdev uses it |
|------|-------------------|
| [devenv.sh](https://devenv.sh) | qsdev generates configs; devenv runs the environment |
| [Claude Code](https://claude.ai/code) | Permissions, deny rules, skills, hooks, MCP configs |
| [Nix](https://nixos.org) | Reproducible, hermetic package resolution |
| [direnv](https://direnv.net) | Auto-activates the environment on `cd` |
| [pre-commit](https://pre-commit.com) | Lockfile checks, formatting, linting hooks |

## Commands

```
qsdev init                    # Generate complete secure environment
qsdev init --profile go-web   # Use a project-type preset
qsdev status                  # Security posture assessment (score + grade)
qsdev check                   # CI enforcement (config integrity, hardening)
qsdev update                  # Update configs + devenv inputs
qsdev repair                  # Fix corrupted or drifted files
qsdev teardown                # Remove all qsdev configuration (clean exit)
qsdev enable <tool>           # Enable a security/AI tool
qsdev disable <tool>          # Disable a tool
qsdev list                    # Show all available tools
```

<details>
<summary>Full command reference</summary>

### Top-level

| Command | Description |
|---------|-------------|
| `init` | Generate complete secure environment (wizard + detection + generation) |
| `status` | Security posture assessment with score and grade |
| `check` | CI enforcement checks (JSON, SARIF, JUnit output) |
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

### devenv subcommands

| Command | Description |
|---------|-------------|
| `devenv doctor` | Diagnose environment issues |
| `devenv setup` | Install prerequisites (Nix, devenv, direnv) |
| `devenv add-language <name>` | Add a language ecosystem |
| `devenv add-service <name>` | Add a service (postgres, redis, etc.) |
| `devenv add-package <name>` | Add system packages |
| `devenv remove-language/service/package` | Remove components |

### claude subcommands

| Command | Description |
|---------|-------------|
| `claude init` | Initialize Claude Code config independently |
| `claude add-skill <name>` | Add a skill |
| `claude add-hook <name>` | Enable a hook preset |
| `claude list-skills` | List available skills |

</details>

## Project Profiles

Pre-configured bundles for common project types:

| Profile | Languages | Services | Security |
|---------|-----------|----------|----------|
| `go-web` | Go | PostgreSQL, Redis | Standard + safety-block |
| `ts-fullstack` | TypeScript (pnpm) | PostgreSQL, Redis | Standard + auto-format |
| `python-data` | Python (uv) | — | Minimal |
| `rust-cli` | Rust | — | Minimal + pre-commit |

Infrastructure profiles control organization-wide policy:

| Profile | Focus |
|---------|-------|
| `consulting-default` | Enhanced security (semgrep, gitleaks, secretspec) |
| `startup-fast` | Baseline security, minimal overhead |
| `enterprise` | Strict security, audit logging, SBOM |

## What qsdev is NOT

qsdev generates configuration files. It does not:

- **Run your environment** — [devenv.sh](https://devenv.sh) does that
- **Manage runtime versions** — Nix pins versions declaratively; no need for nvm/pyenv/mise
- **Run tasks** — Use devenv tasks, Make, or Just
- **Manage containers** — Docker Compose or Podman handles orchestration
- **Deploy anything** — Strictly local development; CI/CD is out of scope
- **Scaffold application code** — Generates dev environment config, not boilerplate
- **Configure your entire IDE** — Only `.editorconfig` and VS Code extension recommendations

## Built On

qsdev is built on [gdev](https://github.com/fastcat/gdev), a fantastic developer experience framework I have used for years and deeply miss whenever I can't created by Matthew Gabeler-Lee. As always thank you for building such an awesome tool.

## Documentation

- [Security Architecture](docs/security-architecture.md) — Threat model, 10 defense layers, permission model
- [Configuration Reference](docs/configuration-reference.md) — Every generated file and its merge strategy
- [Team Onboarding](docs/team-onboarding.md) — Profiles, policies, team rollout
- [Migration Guide](docs/migration-guide.md) — Adding qsdev to existing projects

## License

[Apache-2.0](LICENSE)

Copyright 2024–2026 Quantum Serendipity Software.
