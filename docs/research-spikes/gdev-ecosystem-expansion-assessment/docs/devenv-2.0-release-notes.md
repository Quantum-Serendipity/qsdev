# devenv 2.0: A Fresh Interface to Nix

- **Source URL**: https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/
- **Retrieval Date**: 2026-05-14

## Interactive Experience

**Terminal UI**
The release introduces a live terminal interface replacing cryptic build logs. Instead of scrolling through Nix output, users see "structured progress: what Nix is evaluating, how many derivations need to be built and downloaded."

**Native Shell Reloading**
Files can be saved and rebuilt in the background while maintaining shell interactivity. Users press `Ctrl+Alt+R` to apply new environments. "Your shell stays interactive the entire time" with errors displayed non-disruptively. Currently supports bash, with fish and zsh coming soon.

**Built-in Process Manager**
A Rust-based process manager replaces process-compose as the default. Features include:
- Dependency ordering with `@ready` and `@completed` modes
- Readiness probes (exec, HTTP, systemd notify)
- Systemd socket activation
- Watchdog heartbeats
- File watching capabilities
- Automatic port allocation

## Performance Improvements

The evaluation cache is now incremental, with "each evaluated attribute cached individually along with the files and environment variables it touched." A C FFI backend built on nix-bindings-rust replaces multiple CLI invocations, calling "the Nix evaluator and store directly through the C API."

Performance gains mean running `devenv shell` twice: the second execution takes milliseconds.

## Polyrepo Support

Projects can reference outputs from other devenv repositories via `inputs.<name>.devenv.config`. The `--from` flag enables ad-hoc environments: `devenv shell --from github:myorg/devenv-configs?dir=rust-web`

## Coding Agent Features

**Automatic Port Allocation**
Named ports automatically find free alternatives when primary ports are occupied. `--strict-ports` fails rather than searching.

**SecretSpec Integration**
Version 0.7.2 provides declarative, provider-agnostic secrets management. Secrets come from keyring, dotenv, 1Password, or environment variables—preventing "silently leaked to agents running in the background."

**MCP Server**
Exposes package and option search over stdio and HTTP, with a public instance at `mcp.devenv.sh`.

## Additional Capabilities

- Language server support via `lsp.enable` and `lsp.package` options
- `devenv lsp` for configuration file completion and diagnostics
- `devenv eval` returns attribute values as JSON
- `devenv build` outputs structured JSON mapping attributes to store paths
- Global `NIXPKGS_CONFIG` for consistent nixpkgs configuration

## Breaking Changes from 1.x

- `git-hooks` input no longer included by default (must be added to `devenv.yaml`)
- `devenv container --copy` removed; use `devenv container copy` instead
- `devenv build` outputs JSON instead of plain store paths
- Native process manager is now default (set `process.manager.implementation = "process-compose"` for legacy behavior)

## Deprecation Notice

devenv 0.x is deprecated with support ending in devenv 3.
