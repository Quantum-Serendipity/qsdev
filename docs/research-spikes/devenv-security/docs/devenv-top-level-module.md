# devenv Top-Level Module (top-level.nix)
- **Source**: https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix
- **Retrieved**: 2026-05-12

## Key Options Defined

**Environment Configuration:**
- `env`: Submodule for exposing environment variables with freeform attribute support
- `name`: Project identifier (defaults to "devenv-shell")
- `enterShell`: Bash code executed when entering the shell environment

**Package Management:**
- `packages`: List of derivations to expose in the developer environment
- `inputsFrom`: Merges build inputs from specified derivations
- `overlays`: Apply Nix overlays to modify packages (requires devenv 1.4.2+)

**Platform-Specific:**
- `stdenv`: Customizable standard environment (removes default Apple SDK on macOS)
- `apple.sdk`: Optional Apple SDK configuration for macOS environments

**Cleanup & Security:**
- `unsetEnvVars`: Removes build-related variables from the environment (includes 25+ defaults like `buildInputs`, `shellHook`, `strictDeps`)
- `hardeningDisable`: Allows disabling hardening modules (currently used for Go)

**Internal Options:**
- `shell`: Generated mkShell derivation
- `ci`: CI-related packages
- `ciDerivation`: Composite CI output
- `assertions`: Configuration validation with error messaging
- `warnings`: User-facing advisory messages
- `devenv`: Runtime paths including `root`, `dotfile`, `state`, `runtime`, `tmpdir`, `profile`

The module imports 15+ submodules covering languages, services, integrations, and process managers.
