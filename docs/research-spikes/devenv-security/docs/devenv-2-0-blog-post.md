<!-- Source: https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv 2.0: A Fresh Interface to Nix - Complete Overview

## Introduction

The blog post opens with a vivid description of Nix's usability challenges: cryptic terminal output during builds, unclear whether the system is evaluating or downloading, configuration changes requiring full rebuilds, and direnv triggering unexpected recompilations when switching branches.

## Interactive Features

### Terminal UI
devenv 2.0 introduces "a live terminal interface" replacing scrolling build logs with structured progress visualization. The TUI displays what Nix is evaluating, derivation counts needing building or downloading, task execution with dependency hierarchies, and expandable error details.

### Native Shell Reloading
Rather than freezing when direnv triggers rebuilds, the new approach allows "you save a file, devenv rebuilds in the background, a status line at the bottom of your terminal shows progress, and you press `Ctrl+Alt+R` when you're ready to apply the new environment."

The implementation keeps shells interactive throughout. Background rebuilds display progress in a status line. Failed rebuilds show errors without disrupting the active session. Currently bash is supported, with fish and zsh coming later.

### Native Process Manager
A built-in Rust process manager replaces process-compose, offering "dependency ordering, restart policies, readiness probes (exec, HTTP, and systemd notify), systemd socket activation, watchdog heartbeats, file watching, and port allocation."

Dependencies use `@ready` by default (waiting for probes to pass) or `@completed` (waiting for process exit). Processes and tasks can be freely mixed in dependency chains.

## Performance: The "Instant" Improvement

The most significant architectural change involves replacing multiple `nix` CLI invocations with a C FFI backend built on nix-bindings-rust. Rather than spawning five or more separate Nix processes per command, devenv 2.0 "calls the Nix evaluator and store directly through the C API, evaluating one attribute at a time."

Benefits include:
- Better error messages
- Real-time TUI progress
- Millisecond execution on subsequent runs

The evaluation cache is now incremental. "Each evaluated attribute is cached individually along with the files and environment variables it touched. When you change one thing, only the attributes that depend on that change are re-evaluated; everything else is served from cache."

A single evaluation covers `devenv shell`, `devenv test`, `devenv build`, and other commands. When nothing changed (verified by content hash), cached results return immediately without invoking Nix.

Cache invalidation occurs when:
- Source files read during evaluation change
- Environment variables accessed during evaluation change
- devenv version, system, or configuration options change

## Polyrepo and Multi-Repository Support

Projects can now reference outputs from other devenv projects through `inputs.<name>.devenv.config`. The documentation shows a practical example where a service from another repository is imported and used in processes and packages.

### Out-of-Tree Environments
The `--from` flag enables using configurations without checked-in `devenv.nix` files:
- `devenv shell --from github:myorg/devenv-configs?dir=rust-web`
- `devenv shell --from path:../shared-config`

This works with shell, test, and build commands. Limitations currently prevent support for projects relying on `devenv.yaml` for additional inputs.

## Features for Coding Agents

### Automatic Port Allocation
Named ports search for free alternatives automatically. "If port 8080 is taken, devenv tries 8081, 8082, and so on. Ports are held during evaluation to prevent races, then released just before the process starts."

The `--strict-ports` flag causes failures instead of searching.

### SecretSpec Integration
devenv 2.0 ships with SecretSpec 0.7.2 for "declarative, provider-agnostic secrets management." Developers declare needed secrets in `secretspec.toml`, providing them from keyring, dotenv, 1Password, or environment variables. Crucially, password managers "prompt for credentials before giving them out, secrets are never silently leaked to agents running in the background."

### MCP Server
The devenv MCP server exposes package and option search over stdio and HTTP. A public instance at `mcp.devenv.sh` enables queries without local installation. The same search powers devenv.new, which generates `devenv.nix` files for users.

## Additional Improvements

- **Language servers**: Most language modules now have `lsp.enable` and `lsp.package` options
- **devenv.nix language server**: `devenv lsp` provides completion and diagnostics
- **devenv eval**: Evaluate any attribute and return JSON output
- **devenv build JSON output**: Returns structured JSON mapping attribute names to store paths
- **NIXPKGS_CONFIG**: Global environment variable ensures consistent nixpkgs configuration across operations

## Breaking Changes

- The `git-hooks` input is no longer included by default; users must add it to `devenv.yaml` if needed
- `devenv container --copy <name>` replaced by `devenv container copy <name>`
- `devenv build` now outputs JSON instead of plain store paths
- The native process manager is now default; users can revert with `process.manager.implementation = "process-compose"`

## Deprecation Notice

devenv 0.x is deprecated with support dropping entirely in version 3.
