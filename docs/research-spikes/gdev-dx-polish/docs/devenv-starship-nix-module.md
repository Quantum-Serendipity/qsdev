<!-- Source: https://github.com/cachix/devenv/blob/main/src/modules/integrations/starship.nix -->
<!-- Retrieved: 2026-05-12 -->

# Devenv Starship Integration Module

## Options Exposed

The module provides several configuration options under `starship`:

- **`enable`**: Boolean flag to activate the Starship prompt integration
- **`package`**: Specifies which Starship derivation to use (defaults to `pkgs.starship`)
- **`config.enable`**: Toggles whether to override Starship's configuration
- **`config.path`**: Points to a custom TOML configuration file (e.g., `${config.env.DEVENV_ROOT}/starship.toml`)
- **`config.settings`**: Inline TOML configuration as attribute set (mutually exclusive with `path`)

## Configuration Mechanism

The module enforces that exactly one of `path` or `settings` must be configured when `config.enable` is true. It generates the appropriate config format using `toml.generate` when settings are provided inline.

## Environment Setup

During shell entry, the module:

1. Exports `STARSHIP_CONFIG` pointing to the configuration source
2. Initializes Starship with shell-specific hooks via `"$(starship init $(echo $0))"`
3. Unsets `STARSHIP_SHELL` to prevent conflicts when inheriting shell modifications through direnv

The unset operation preserves `STARSHIP_SESSION_KEY` to maintain logging continuity across shell sessions.
