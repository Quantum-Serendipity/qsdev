# Devenv.sh Architecture and Internals

## 1. What Is Devenv.sh?

Devenv is a developer environment management tool that provides a high-level, declarative interface on top of Nix. It is developed and maintained by Cachix (the company behind the Cachix binary cache service) and licensed under Apache 2.0. The CLI is written primarily in Rust (53.9%) with Nix module definitions (44.3%). As of May 2026, it is at version 2.1 with 6.8k GitHub stars.

Devenv's core value proposition is making Nix-based development environments accessible to developers who may not understand the Nix language, while providing reproducible, composable, and fast environment setup. It uses the NixOS module system to expose a configuration interface that looks like NixOS configuration but produces development shells rather than system configurations.

### Relationship to the Nix Ecosystem

| Component | Role in devenv |
|---|---|
| **Nix** | The underlying build/evaluation engine. Devenv 2.0 calls the Nix evaluator directly via C FFI (nix-bindings-rust) rather than spawning CLI processes. |
| **Nix Flakes** | Devenv uses the flake input/lock mechanism for dependency pinning. `devenv.lock` is a flake-compatible lock file. Devenv can run standalone or be embedded within a `flake.nix`. |
| **nixpkgs** | The package set. Devenv defaults to `github:cachix/devenv-nixpkgs/rolling` rather than upstream nixpkgs (see Section 7). |
| **NixOS Module System** | Devenv's entire configuration model is built on the NixOS module system (`lib.mkOption`, `lib.mkIf`, module merging). Each language, service, and integration is a Nix module. |
| **Cachix** | The Cachix binary cache is tightly integrated. `devenv.cachix.org` is a default pull cache. Cachix push/pull is configured directly in `devenv.nix`. |

## 2. Architecture Overview

### High-Level Data Flow

```
devenv.yaml          devenv.nix           devenv.local.nix
(inputs, imports,    (environment         (personal overrides,
 nixpkgs config)      definition)          not committed)
       |                  |                      |
       v                  v                      v
  ┌─────────────────────────────────────────────────┐
  │         NixOS Module System Evaluation          │
  │  (merges all modules, resolves options,         │
  │   applies overlays, evaluates lazy attrs)       │
  └─────────────────────────────────────────────────┘
                          |
              ┌───────────┼───────────┐
              v           v           v
         Shell Profile  Processes   Containers
         (packages,     (managed    (OCI images
          env vars,      by native   built without
          scripts,       Rust PM)    Docker)
          hooks)
```

### Command Lifecycle

**`devenv init`** scaffolds four files:
1. `devenv.nix` -- the primary environment definition (a Nix function returning an attribute set)
2. `devenv.yaml` -- input sources and configuration metadata
3. `.envrc` -- direnv integration hook
4. `.gitignore` -- excludes `.devenv/`, `.direnv/`

**`devenv shell`** performs:
1. Read `devenv.yaml` to resolve inputs and imports
2. Check evaluation cache (SQLite database in `.devenv/`) -- if cache is valid (content hashes of all accessed files match), return cached result in milliseconds
3. If cache miss: evaluate `devenv.nix` through the NixOS module system using the Nix C API (v2.0+) or by spawning `nix` processes (v1.x)
4. Build the profile derivation (`buildEnv` aggregating all packages)
5. Fetch/build any missing store paths (checking binary caches first, then building from source)
6. Enter a new shell with `$PATH` pointing to the profile, environment variables set, and `enterShell` hooks executed
7. In v2.0+, the shell supports background rebuilds with a status line; `Ctrl+Alt+R` applies the new environment

**`devenv up`** performs:
1. Same evaluation as `devenv shell`
2. Starts the native Rust process manager (or process-compose in v1.x) with configured processes
3. Process manager handles dependency ordering, restart policies, readiness probes (exec/HTTP/systemd-notify), watchdog heartbeats, socket activation, and file watching
4. Automatic port allocation finds free ports if configured ports are in use

**`devenv test`** performs:
1. Same evaluation as `devenv shell`
2. Starts processes if any are defined
3. Executes `enterTest` hook
4. Stops processes and reports results

**`devenv build`** evaluates and builds specified outputs, returning JSON mapping attribute names to Nix store paths.

## 3. Configuration Model: devenv.nix vs devenv.yaml

### devenv.yaml -- Declarative Metadata

`devenv.yaml` is a YAML file controlling:
- **Inputs**: Nix flake references for package sources (default: `github:cachix/devenv-nixpkgs/rolling`)
- **Imports**: Paths to other `devenv.nix`/`devenv.yaml` files for composition
- **Nixpkgs config**: `allow_unfree`, `allow_broken`, `permitted_insecure_packages`, license allowlists/blocklists, CUDA/ROCm support
- **Clean shell**: Whether to strip the inherited environment (`clean.enabled`, `clean.keep`)
- **Impure mode**: `impure: true` relaxes hermeticity (allows reading host state)
- **SecretSpec**: Enable declarative secrets management with provider selection
- **Version constraints**: `require_version: ">=2.1"` enforces CLI compatibility

Key property: `devenv.yaml` is evaluated **before** Nix -- it determines **what** Nix evaluates.

### devenv.nix -- Nix Environment Definition

`devenv.nix` is a Nix function that receives inputs and returns an attribute set of module options:

```nix
{ pkgs, lib, config, inputs, ... }:
{
  packages = [ pkgs.git pkgs.jq ];
  languages.python.enable = true;
  services.postgres.enable = true;
  enterShell = ''echo "Welcome"'';
  processes.web.exec = "python -m http.server";
}
```

The function's arguments are:
- `pkgs` -- instantiated nixpkgs from the configured input
- `lib` -- Nix standard library functions
- `config` -- the final resolved configuration (lazy, self-referential)
- `inputs` -- all inputs from `devenv.yaml`
- `...` -- catch-all for unused arguments

### devenv.local.nix / devenv.local.yaml

Personal override files with identical structure but excluded from version control. Used for developer-specific customizations or CI-only settings (e.g., `cachix.push`).

### Evaluation Model

The evaluation follows the NixOS module system pattern:
1. All imported modules (from `devenv.nix`, imports, language/service modules) are collected
2. The module system merges all option declarations and definitions
3. Options are lazily evaluated -- only accessed attributes trigger computation
4. Assertions are checked; warnings are emitted
5. The final configuration produces a shell profile derivation

In devenv 2.0+, this evaluation happens through the Nix C API (via nix-bindings-rust FFI), evaluating one attribute at a time with per-attribute caching. In v1.x, five or more separate `nix` CLI processes were spawned per command.

## 4. Package, Service, Process, and Hook Management

### Packages

Packages are added via `packages = [ pkgs.foo ];` or through language modules (`languages.python.enable = true` automatically adds the Python interpreter, pip, etc.). The `top-level.nix` module aggregates all packages into a `buildEnv` derivation that creates a unified profile in the Nix store.

Overlays can be applied to modify packages: `overlays = [ (final: prev: { ... }) ];`

### Services

40+ service modules (PostgreSQL, Redis, Elasticsearch, etc.) are built-in. Each is a NixOS-style module that adds packages, creates configuration files via `files.<name>`, and registers processes. Services are configured declaratively:

```nix
services.postgres = {
  enable = true;
  listen_addresses = "127.0.0.1";
};
```

Enabling a service typically adds its binary cache to `cachix.pull` automatically.

### Processes

Processes are registered via `processes.<name>` with options for:
- `exec` -- the command to run
- `before` / `after` -- lifecycle hooks
- `env` -- process-specific environment variables
- `ports` -- named port allocation (auto-increments if in use)
- `ready` -- health check configuration (exec, HTTP, systemd-notify)
- `watch` -- file patterns that trigger restart
- `restart` -- restart policy

The process manager is pluggable. Devenv 2.0 defaults to a native Rust implementation supporting dependency ordering via DAG, but `process-compose`, `hivemind`, `honcho`, `mprocs`, and `overmind` are also available.

### Shell Hooks

- `enterShell` -- bash code executed when entering the shell. For complex setup, `tasks` are preferred (DAG-ordered, cached, parallelizable).
- `enterTest` -- bash code executed during `devenv test`.
- `scripts.<name>.exec` -- named scripts added to `$PATH` with explicit dependencies.

### Tasks

Tasks (added in v2.0) provide structured, cacheable setup operations with:
- DAG-based dependency ordering
- Parallel execution
- Caching (skip if outputs haven't changed)
- Explicit dependencies between tasks

## 5. Direnv Integration (.envrc)

Devenv supports two shell activation approaches:

### Native Activation (v2.0+ recommended)
- `devenv shell` starts a subshell with the environment
- `devenv hook` provides automatic directory-based activation
- Native shell reloading rebuilds in the background; `Ctrl+Alt+R` applies changes

### Direnv Integration (legacy, still supported)
- `.envrc` file contains `eval "$(devenv direnvrc)"` and `use devenv`
- Direnv modifies the current shell **in place** (no subshell)
- `direnv allow` must be explicitly run -- security gate against arbitrary `.envrc` execution
- The `.direnv/` directory caches evaluation results
- Environment automatically loads/unloads when entering/leaving the project directory
- Options can be passed: `use devenv --option services.postgres.enable:bool true`

Security note: direnv's `allow` mechanism is a TOCTOU gate -- it checks content hash at allow-time, but `.envrc` could be modified between `direnv allow` and actual execution if the file is on a shared filesystem or the repo is compromised.

## 6. Install Time, Build Time, and Runtime Control

### Install Time (devenv init / devenv update)
- Scaffolds project files
- Resolves input URLs to specific commit revisions
- Writes `devenv.lock` with pinned hashes
- Does NOT execute any package builds or arbitrary code

### Build Time (Nix evaluation + derivation building)
- Evaluates `devenv.nix` through the module system
- Builds derivations in the Nix sandbox (isolated build environment with no network access by default)
- Fetches pre-built binaries from configured binary caches (signature-verified)
- Builds from source if no cache hit
- Build-time environment variables are cleaned up via `unsetEnvVars`

### Runtime (devenv shell / devenv up)
- Sets `$PATH`, environment variables, shell hooks
- Executes `enterShell` bash code (arbitrary code execution)
- Starts processes via the process manager
- Exposes `$DEVENV_ROOT`, `$DEVENV_DOTFILE`, `$DEVENV_STATE`, `$DEVENV_RUNTIME`, `$DEVENV_PROFILE` environment variables
- Runtime is NOT sandboxed by default -- processes run with the user's full permissions

## 7. Binary Caches and Substituters

### Default Configuration
- `devenv.cachix.org` is added as a pull cache by default
- This cache mirrors packages from the official NixOS cache for the `devenv-nixpkgs/rolling` input
- Some language/service modules automatically add their own caches when enabled

### Trust Model
Binary cache security relies on:
1. **Public key verification**: Each cache has a signing key. Nix refuses to use store paths without a matching trusted public key.
2. **narHash verification**: Each store path has a content hash (NAR hash) recorded in the lock file and verified on download.
3. **Trusted users**: To add substituters, users must either be in `trusted-users` in `/etc/nix/nix.conf` or have caches pre-configured system-wide.

### Configuration
```nix
cachix.pull = [ "mycache" ];     # Download from these caches
cachix.push = "mycache";         # Upload built paths to this cache
cachix.enable = false;           # Disable Cachix entirely
```

Push requires `CACHIX_AUTH_TOKEN` environment variable. Write access is controlled through authentication tokens.

### Security Implications
- The `devenv.cachix.org` default cache is controlled by Cachix (the company). Trusting it means trusting Cachix's build infrastructure.
- The `devenv-nixpkgs/rolling` input adds patches on top of upstream nixpkgs -- these patches are maintained by the Cachix team and not reviewed by the broader nixpkgs community.
- Language modules can auto-add caches, expanding the trust surface without explicit user action.
- `cachix.enable = false` does NOT disable all binary substitution -- Nix still uses caches configured in `/etc/nix/nix.conf`.

## 8. The devenv.lock File

`devenv.lock` uses the Nix flake lock file format (currently schema version 7). It is a JSON file containing a dependency graph where each node has:

- **locked**: The resolved reference with exact commit hash (`rev`), content hash (`narHash`), timestamp (`lastModified`), and source metadata (`owner`, `repo`, `type`)
- **original**: The user-specified reference (may include branch names like `nixpkgs-unstable`)
- **inputs**: Dependency edges to other nodes (including `follows` relationships)

Example structure:
```json
{
  "nodes": {
    "nixpkgs": {
      "locked": {
        "lastModified": 1678724065,
        "narHash": "sha256-MjeRjunqfGTBGU401nxIjs7PC9PZZ1FBCZp/bRB3C2M=",
        "owner": "NixOS",
        "repo": "nixpkgs",
        "rev": "b8afc8489dc96f29f69bec50fdc51e27883f89c1",
        "type": "github"
      },
      "original": {
        "owner": "NixOS",
        "ref": "nixpkgs-unstable",
        "repo": "nixpkgs",
        "type": "github"
      }
    }
  },
  "root": "root",
  "version": 7
}
```

`devenv update` re-resolves inputs and updates the lock file. Individual inputs can be updated selectively with `devenv update <input-name>`.

The lock file pins:
- The nixpkgs commit (and therefore all package versions)
- The devenv modules version (from `cachix/devenv` repo)
- Any git-hooks, pre-commit, or custom inputs
- Transitive dependencies of all inputs

## 9. Plugin/Module System

Devenv's extensibility is built entirely on the NixOS module system. There is no separate "plugin" API -- everything is a module.

### How Modules Work
Each module is a Nix file that:
1. Declares options (using `lib.mkOption` with types, defaults, descriptions)
2. Defines option values (setting other module options based on its own configuration)
3. Gets merged with all other modules by the module system

### Built-in Module Categories
- **Languages** (50+): Each language module (`languages.<name>`) adds compiler/runtime packages, LSP servers, formatters, and linters
- **Services** (40+): Database servers, caches, message queues, web servers
- **Integrations**: Cachix, git-hooks, SecretSpec, direnv, devcontainers, Claude Code
- **Process managers**: Native, process-compose, hivemind, honcho, mprocs, overmind

### Composition via Imports
Modules compose through the `imports` field in `devenv.yaml`:
```yaml
imports:
  - ./frontend
  - ./backend
  - myinput/examples/scripts
```

Local imports merge both `devenv.nix` and `devenv.yaml` files. Remote imports (from inputs) only load Nix configurations. When merged, packages combine, processes combine, and environment variables are union-merged (with conflicts requiring explicit resolution).

### Custom Modules
Users create custom modules by writing Nix files that follow the NixOS module pattern and importing them. The `outputs` option allows modules to expose buildable artifacts.

## 10. devenv-nixpkgs and the Cachix Relationship

### What is devenv-nixpkgs?

`devenv-nixpkgs` is a Cachix-maintained fork of nixpkgs that serves as devenv's default package source. The `rolling` branch is based on `nixpkgs-unstable` plus patches in two categories:

1. **Upstream patches**: Cherry-picked from nixpkgs PRs or unreleased fixes
2. **Local patches**: Fixes not yet submitted upstream

Additionally, `overlays/default.nix` provides package-level modifications described as "more resilient to upstream changes than source patches."

### Testing Infrastructure
- 284 test jobs run across 4 platforms (aarch64-linux, x86_64-linux, aarch64-darwin, x86_64-darwin)
- Weekly CI (Mondays 9:00 UTC) performs: nixpkgs update, patch validation, full test suite, automatic release PR from `main` to `rolling`
- Current success rate: ~93%

### Security Implications
- Using `devenv-nixpkgs/rolling` means depending on a smaller, less-reviewed codebase than upstream nixpkgs
- The Cachix team can apply arbitrary patches -- the trust boundary is the Cachix organization
- Users can switch to upstream nixpkgs: `inputs.nixpkgs.url: github:NixOS/nixpkgs/nixpkgs-unstable`
- The `devenv.cachix.org` binary cache only has pre-built binaries for `devenv-nixpkgs/rolling` -- switching to upstream nixpkgs means more local compilation or relying solely on `cache.nixos.org`

## 11. Evaluation Caching Architecture

### v1.3: SQLite-Based Log Parsing
Inspired by lorri, devenv 1.3 introduced precise evaluation caching:
1. During Nix evaluation, devenv parses Nix's internal logs to identify all accessed files and directories
2. For each path: records file path, content hash, and modification timestamp
3. Metadata stored in SQLite database (in `.devenv/`)
4. On subsequent runs: compares current file state against stored metadata
5. Cache hit = return immediately (single-digit milliseconds); cache miss = full re-evaluation

Advantages over lorri: no background daemon, integrated as a built-in feature.

### v2.0: Incremental Attribute-Level Caching via C API
- Uses nix-bindings-rust to call the Nix evaluator directly via C FFI
- Each attribute is evaluated and cached individually
- Cache includes which files and environment variables each attribute touched
- Changing one file only re-evaluates attributes that accessed it
- Cache invalidation triggers: file content changes, environment variable changes, devenv version/system changes
- Sub-100ms activation when nothing changed

## 12. Comparison with Alternatives

| Feature | devenv.sh | `nix develop` (flakes) | lorri | nix-direnv | services-flake |
|---|---|---|---|---|---|
| **Abstraction level** | High (NixOS modules) | Low (raw Nix) | Low (watches Nix files) | Low (direnv integration) | Medium (flake-parts modules) |
| **Configuration** | `devenv.nix` + `devenv.yaml` | `flake.nix` | `shell.nix` | `flake.nix` | `flake.nix` |
| **Evaluation caching** | Built-in (SQLite/C API) | Flake eval cache only | Background daemon | direnv cache | Flake eval cache |
| **Process management** | Built-in (native Rust PM) | None | None | None | Via process-compose |
| **Service modules** | 40+ built-in | None | None | None | 20+ modules |
| **Binary caching** | Integrated Cachix | Manual `nix.conf` | Manual | Manual | Manual |
| **Sandbox support** | Experimental (PR #2427) | None (runtime) | None | None | None |
| **direnv integration** | Built-in + native activation | Via nix-direnv | Built-in | Purpose-built | Via nix-direnv |
| **Learning curve** | Low (if you know YAML/Nix basics) | High (full Nix knowledge) | Medium | Medium | Medium-High |
| **Vendor dependency** | Cachix (default nixpkgs, cache, CLI) | None | None | None | None |
| **Pure evaluation** | No (requires `--no-pure-eval` when embedded in flake) | Yes (default) | N/A | Yes | Yes |
| **Container building** | Built-in (OCI without Docker) | Separate tooling | None | None | Separate |
| **Garbage collection** | Protected roots | Manual | N/A | Protected | Manual |

### Key Tradeoffs

**devenv.sh advantages:**
- Lowest barrier to entry for Nix-based dev environments
- Most integrated solution (packages + services + processes + caching + containers in one tool)
- Fastest iteration cycle (sub-100ms cached activation)
- Rich module ecosystem with language-specific tooling

**devenv.sh disadvantages:**
- Vendor lock-in to Cachix ecosystem (default nixpkgs fork, binary cache, CLI)
- Cannot use pure evaluation (breaks `devenv.nix` which needs to query the host environment)
- `devenv.lock` format changes between major versions (v1.0 locks incompatible with older versions)
- Runtime is unsandboxed -- `enterShell`, scripts, and processes execute with full user permissions
- The module system hides complexity that may need understanding for debugging

**`nix develop` advantages:**
- No vendor dependency
- Pure evaluation by default
- Standard flake ecosystem compatibility
- Full Nix language control

**`nix develop` disadvantages:**
- No built-in process management, service modules, or caching
- Slower evaluation (no incremental caching)
- Steeper learning curve

## 13. Edge Cases, Limitations, and Failure Modes

### Evaluation Purity
- devenv requires impure evaluation (needs to read `$PWD`, system type, etc.)
- When embedded in a flake, requires `--no-pure-eval` flag
- This means evaluation can depend on host state, reducing reproducibility guarantees

### Lock File Drift
- `devenv update` can silently change the entire package set
- Lock file format changes between major versions
- No built-in mechanism to audit what changed between lock updates

### Cache Coherence
- The evaluation cache in `.devenv/` can become stale if files are modified outside of normal workflows (e.g., `git checkout` while direnv is active)
- nix-direnv has known issues with `extra-substituters` causing hangs (issue #1491)

### Process Management
- Process cleanup occasionally requires manual PID cleanup (vs Docker Compose reliability)
- Socket path limits addressed by `$DEVENV_RUNTIME` but can still be an issue

### Platform Gaps
- Native shell reloading (v2.0) only supports bash; fish and zsh are planned
- Sandbox support (bubblewrap) is Linux-only and still in draft PR status
- Some services are Linux-only due to systemd dependencies

### Module Conflicts
- When composing multiple environments via imports, package collisions are handled by `buildEnv` with collision resolution, but environment variable conflicts require manual resolution
- Overlays from different imports can interfere with each other

---

## Sources

All source documents are saved in `docs/` with full content, source URLs, and retrieval dates. Key sources:

- `docs/devenv-2-0-blog-post.md` -- devenv 2.0 architecture, C API, caching, process manager
- `docs/devenv-1-3-caching-architecture.md` -- SQLite-based evaluation caching design
- `docs/devenv-1-0-rust-rewrite.md` -- Rust rewrite rationale and architecture
- `docs/devenv-1-1-module-system.md` -- Module system and nested outputs
- `docs/devenv-github-readme.md` -- Feature overview and technical implementation
- `docs/devenv-basics.md` -- Configuration structure and core concepts
- `docs/devenv-files-and-variables.md` -- File roles and environment variables
- `docs/devenv-yaml-options-reference.md` -- Complete devenv.yaml option reference
- `docs/devenv-nix-options-reference.md` -- Complete devenv.nix option reference
- `docs/devenv-inputs.md` -- Input system, URI formats, locking, following
- `docs/devenv-binary-caching.md` -- Cachix integration and trust model
- `docs/devenv-direnv-integration.md` -- direnv setup and activation workflow
- `docs/devenv-using-with-flakes.md` -- Flake integration and limitations
- `docs/devenv-composing-imports.md` -- Module composition and merging
- `docs/devenv-nixpkgs-repo.md` -- devenv-nixpkgs rolling branch and patch management
- `docs/devenv-lock-example.md` -- Lock file format with annotated example
- `docs/devenv-top-level-module.md` -- Core module implementation details
- `docs/devenv-sandbox-pr-2427.md` -- Bubblewrap sandbox proposal
- `docs/discourse-devenv-vs-services-flake.md` -- Community comparison discussion
