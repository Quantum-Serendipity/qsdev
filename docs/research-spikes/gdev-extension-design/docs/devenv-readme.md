# devenv.sh README
- **Source**: https://raw.githubusercontent.com/cachix/devenv/main/README.md
- **Retrieved**: 2026-05-12

## Header and Badges
The document begins with the devenv.sh logo (with light/dark mode variants) linking to https://devenv.sh, followed by the tagline: "Fast, Declarative, Reproducible, and Composable Developer Environments"

Multiple status badges are displayed, indicating:
- Built with devenv and Nix
- Discord community membership
- Apache 2.0 license
- Current version and CI status

## Features Section

**Developer Experience:**
- Terminal UI showcasing live build progress, task hierarchy, and error details
- Native shell reloading that rebuilds in the background while maintaining shell interactivity
- Instant environments using incremental Nix evaluation caching (sub 100ms response when unchanged)
- LSP support for devenv.nix with autocomplete, hover documentation, and definition navigation
- Ad hoc environments from CLI without configuration files
- Out of tree devenvs referencing configs from other repositories

**Languages, Packages, and Services:**
- 50+ languages with built-in tooling (compilers, LSP servers, formatters, linters)
- 100,000+ packages from Nixpkgs supporting Linux, macOS, x64, ARM64, and WSL2
- 40+ services including PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, and Caddy

**Processes and Tasks:**
- Rust-based native process manager with dependency ordering and restart policies
- Automatic port allocation preventing collisions across parallel environments
- DAG-based task execution with caching and parallel runs
- Scripts accessing all environment packages

**Packaging and Deployment:**
- OCI container building without Docker dependency
- Language-specific output packaging tools
- Polyrepo support for cross-repository references

**Composition and Configuration:**
- Environment profiles for variants
- Composable imports for sharing environments
- Pinned and overridable Nix dependencies

**Security and Integrations:**
- SecretSpec for provider-agnostic secrets management
- Git hooks via git-hooks.nix
- Testing with automatic process management
- direnv integration for automatic activation
- MCP server for AI assistant integration
- AI generation scaffolding via devenv.new

## Quick Start

The section demonstrates initializing a project with `devenv init`, generating a `devenv.nix` configuration file. A sample configuration is provided showing:
- Environment variables
- Package management
- Language setup (Rust example)
- Process definitions
- Service configuration (PostgreSQL example)
- Custom scripts
- Shell entry hooks
- Task definitions
- Test specifications
- Output packaging
- Git hooks

The `devenv shell` command activates the environment.

## Commands Reference

The comprehensive command list includes:
- `init` - scaffolding configuration files
- `generate` - AI-powered environment creation
- `shell` - environment activation
- `update` - dependency management
- `search` - package searching
- `info` - environment information
- `up` - process management
- `processes` - process control
- `tasks` - task execution
- `test` - testing framework
- `container` - container operations
- `inputs` - dependency management
- `build`, `eval` - compilation and evaluation
- `lsp`, `mcp` - language server and AI integration
- `direnvrc` - direnv configuration
- Additional utilities

### Input Override Options

The CLI supports:
- `--from` for specifying devenv.nix sources (filesystem or flake references)
- `-o, --override-input` for input substitution
- `-O, --option` for typed configuration overrides

### Nix Options

Configuration includes:
- Job and core allocation control
- System target specification
- Purity/impurity settings
- Offline mode support
- Nix-specific configuration pass-through
- Debugger access

### Shell Options

Features include:
- Environment variable filtering
- Profile activation (supporting multiple profiles)
- Auto-reload configuration
- Caching controls
- SecretSpec provider management
- Tracing and debugging output

### Output Modes

Global options control:
- Verbosity levels
- Terminal UI display
- Help and version information

## Documentation Links

The README concludes with links to:
- Getting Started guide
- Basics documentation
- Project Roadmap
- Official Blog
- Configuration references (YAML and Nix)
- Contributing guidelines
