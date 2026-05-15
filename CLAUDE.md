# CLAUDE.md

## Project Overview

This project implements three qsdev addons (`devenv`, `claudecode`, `devinit`) that enable developers to run `qsdev init` and get a fully configured, security-hardened development environment. The system covers 27 language/platform ecosystems and provides defense-in-depth against supply chain attacks.

## System Environment

This machine runs NixOS. Use `nix develop` (via direnv) for the development environment. The flake provides Go tooling.

## Architecture

- **Three qsdev addons**: `devenv` (devenv.sh environment management), `claudecode` (Claude Code configuration), `devinit` (orchestration + wizard)
- **Ecosystem module interface**: Each language/platform is a self-contained module implementing `EcosystemModule`
- **Infrastructure profiles**: Organization-wide choices (registry proxy, Nix cache, build cache) encoded in reusable profiles

## Implementation Plan

The full implementation plan is at `docs/implementation-plan/plan.md`. Phase files with detailed implementation units are in `docs/implementation-plan/phases/`.

## Research Foundation

Four completed research spikes are in `docs/research-spikes/`:
- `gdev-extension-design/` — Addon architecture, wizard UX, template engine, migration strategy
- `package-supply-chain-security/` — Per-ecosystem attack surface, age-gating, lockfile enforcement
- `devenv-security/` — Hardened devenv.sh boilerplate, nix.conf hardening, pre-commit hooks, trust model
- `claude-code-agent-package-guardrails/` — 5-layer defense architecture, PreToolUse hooks, deny rules

## Build Commands

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run
```

## Key Dependencies

- [gdev](https://github.com/fastcat/gdev) — The addon framework (Go, `fastcat.org/go/gdev`)
- [charmbracelet/huh](https://github.com/charmbracelet/huh) — TUI forms for the wizard (already used by gdev bootstrap)
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) — YAML marshaling for devenv.yaml

## Commit Conventions

Do not include `Co-Authored-By` lines in commit messages.
