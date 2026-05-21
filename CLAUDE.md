# CLAUDE.md

<!-- BEGIN GENERATED SECTION — do not edit between markers -->

## Project Overview

qsdev — a Go project.



## Build & Test

```bash
go build ./...
go test ./...
go vet ./...
golangci-lint run
```



<!-- qsdev:commands -->
## qsdev Commands
- `qsdev init` — Initialize or re-initialize project
- `qsdev devenv doctor` — Check system and project health
- `qsdev devenv setup` — Install missing prerequisites
- `qsdev enable <tool>` — Enable a tool
- `qsdev disable <tool>` — Disable a tool
- `qsdev status` — Show configuration state
- `qsdev list` — Show available tools
- `qsdev check` — Validate configuration for CI
<!-- /qsdev:commands -->

<!-- qsdev:tasks -->
## Development Tasks
- `qsdev-build` — go build ./...
- `qsdev-test` — go test ./...
- `qsdev-lint` — go vet ./..., golangci-lint run
<!-- /qsdev:tasks -->


## Security

- Never run raw install commands (`npm install`, `pip install`, `nix-env -i`, etc.) — the package guard hook blocks unsafe operations and guides you through the safe workflow.
- To add ecosystem tools: `qsdev enable <tool>` (run `qsdev list` to see available tools)
- To add system packages: `qsdev devenv add-package <name>`
- To add languages: `qsdev devenv add-language <name>`
- To add services: `qsdev devenv add-service <name>`
- Never commit secrets, tokens, or credentials. The pre-commit hook `ripsecrets` blocks accidental leaks.
- Lock files must always be committed.
- Package managers: go modules.
<!-- qsdev:attach-guard -->
- Safety-block hooks are enabled.
<!-- /qsdev:attach-guard -->
<!-- qsdev:agent-postmortem -->
- Agent-postmortem skill is active.
<!-- /qsdev:agent-postmortem -->
<!-- qsdev:version-sentinel -->
- **Version-Sentinel** guards dependency changes in: .
- Version-Sentinel does NOT cover: go.mod. Review these manually.
<!-- /qsdev:version-sentinel -->
- Prefer vendored or pinned dependencies.

@.claude/qsdev-reference.md

<!-- END GENERATED SECTION -->

## Project Overview

This project implements three qsdev addons (`devenv`, `claudecode`, `devinit`) that enable developers to run `qsdev init` and get a fully configured, security-hardened development environment. The system covers 27 language/platform ecosystems and provides defense-in-depth against supply chain attacks.

## System Environment

This machine runs NixOS. The development environment is managed by devenv (via `qsdev init`). Run `direnv allow` to activate, or `devenv shell` for manual activation.

## Architecture

- **Three qsdev addons**: `devenv` (devenv.sh environment management), `claudecode` (Claude Code configuration), `devinit` (orchestration + wizard)
- **Ecosystem module interface**: Each language/platform is a self-contained module implementing `EcosystemModule`
- **Infrastructure profiles**: Organization-wide choices (registry proxy, Nix cache, build cache) encoded in reusable profiles

## Implementation Plan

The full implementation plan is at `internal-docs/implementation-plan/plan.md`. Phase files with detailed implementation units are in `internal-docs/implementation-plan/phases/`. These files are local-only (gitignored).

## Research Foundation

Four completed research spikes are in `internal-docs/research-spikes/` (local-only, gitignored):
- `gdev-extension-design/` — Addon architecture, wizard UX, template engine, migration strategy
- `package-supply-chain-security/` — Per-ecosystem attack surface, age-gating, lockfile enforcement
- `devenv-security/` — Hardened devenv.sh boilerplate, nix.conf hardening, pre-commit hooks, trust model
- `claude-code-agent-package-guardrails/` — 5-layer defense architecture, PreToolUse hooks, deny rules

## Key Dependencies

- [gdev](https://github.com/fastcat/gdev) — The addon framework (Go, `fastcat.org/go/gdev`)
- [charmbracelet/huh](https://github.com/charmbracelet/huh) — TUI forms for the wizard (already used by gdev bootstrap)
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) — YAML marshaling for devenv.yaml

## Commit Conventions

Do not include `Co-Authored-By` lines in commit messages.
