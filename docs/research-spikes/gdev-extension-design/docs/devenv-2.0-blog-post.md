# devenv 2.0: A Fresh Interface to Nix
- **Source**: https://devenv.sh/blog/2026/03/05/devenv-20-a-fresh-interface-to-nix/
- **Retrieved**: 2026-05-12

## Key Features

### Interactive Components

**Terminal User Interface (TUI)**
Live terminal interface replacing traditional build logs with structured progress, dependency visualization, and task execution.

**Native Shell Reloading**
Background rebuilds with status indicators. Users press `Ctrl+Alt+R` to apply changes when ready. Currently supports bash, with fish and zsh coming soon.

**Native Process Manager**
Built-in Rust-based process manager replacing process-compose:
- Dependency ordering using `@ready` or `@completed`
- Restart policies and readiness probes (exec, HTTP, systemd notify)
- Systemd socket activation
- Watchdog heartbeats and file watching
- Ability to mix processes and tasks in dependency chains

### Performance Improvements

**Instant Shell Entry**
Subsequent `devenv shell` invocations take milliseconds rather than seconds. Uses C FFI backend via `nix-bindings-rust`, calling the Nix evaluator directly.

**Incremental Evaluation Cache**
Each evaluated attribute caches individually along with touched files and environment variables. Only attributes dependent on changes require re-evaluation.

### Multi-Repository Support

**Polyrepo Capabilities**
Projects can reference outputs from other devenv projects through `inputs.<name>.devenv.config`.

**Out-of-Tree Configurations**
The `--from` flag enables using devenv configurations without checking them into repositories:
- `devenv shell --from github:myorg/devenv-configs?dir=rust-web`
- `devenv shell --from path:../shared-config`

### Coding Agent Features

**Automatic Port Allocation**
Named ports automatically find free alternatives. Sequential port tries (8080, 8081, 8082, etc.) with hold during evaluation to prevent race conditions.

**SecretSpec Integration**
SecretSpec 0.7.2 for declarative secrets management. Unlike `.env` files, password managers prompt users before releasing credentials.

**MCP Server**
`devenv mcp` enables package and option search over stdio and HTTP. Public instance at `mcp.devenv.sh`. `devenv.new` is powered by this functionality.

## Additional Improvements

**Language Server Support**
Most language modules now include `lsp.enable` and `lsp.package` options. Separate LS for `devenv.nix` provides completion and diagnostics via `devenv lsp`.

**Evaluation Commands**
- `devenv eval` evaluates attributes and returns JSON
- `devenv build` outputs JSON mapping attribute names to store paths

**Global Configuration**
`NIXPKGS_CONFIG` environment variable for consistent nixpkgs configuration.

## Breaking Changes

- `git-hooks` input no longer included by default (must be added to `devenv.yaml`)
- `devenv container --copy <name>` removed in favor of `devenv container copy <name>`
- `devenv build` outputs JSON instead of plain store paths
- Native process manager is default (legacy: `process.manager.implementation = "process-compose"`)

## Technical Architecture

Performance gains from C FFI through `nix-bindings-rust`. "We currently carry patches against Nix to extend the C FFI interface" with plans to contribute upstream.
