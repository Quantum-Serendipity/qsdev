<!-- Source: https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv 2.0: A Fresh Interface to Nix

## Major New Features

**Interactive Experience**
Live terminal UI replacing cryptic build logs with "structured progress: what Nix is evaluating, how many derivations need to be built and downloaded." Native shell reloading lets developers rebuild environments in the background while remaining interactive -- pressing `Ctrl+Alt+R` applies changes without freezing the terminal.

**Built-in Process Manager**
New Rust-based process manager replaces process-compose, offering dependency ordering, restart policies, readiness probes (exec, HTTP, systemd notify), socket activation, and automatic port allocation. Dependencies default to `@ready` (wait for probe) or `@completed` (wait for exit).

**Performance Gains**
Replacing multiple Nix CLI invocations with "a C FFI backend built on nix-bindings-rust." Instead of spawning five separate Nix processes per command, devenv 2.0 calls the evaluator directly. Evaluation cache works incrementally -- only re-evaluating attributes affected by changes -- returning cached results in milliseconds when nothing changes.

## Developer Experience Enhancements

- **Language servers**: Most language modules gain `lsp.enable` and `lsp.package` options
- **Secret management**: Integrated SecretSpec 0.7.2 prevents silent credential leaks to background agents
- **Polyrepo support**: Reference outputs from other devenv projects via `inputs.<name>.devenv.config`
- **Out-of-tree devenvs**: Use `--from` flag to apply external configurations to projects lacking local devenv files
- **New commands**: `devenv eval` (output JSON attributes) and `devenv lsp` (language server for devenv.nix)
- **Native auto activation**: `devenv hook` replaces direnv for cd-based activation. No .envrc, no external dependencies.

## Breaking Changes

- `git-hooks` input no longer default-included; must be explicitly added to devenv.yaml
- `devenv build` outputs JSON instead of plain store paths
- Native process manager is default; revert with `process.manager.implementation = "process-compose"`
- devenv 0.x support ends with version 3.0
