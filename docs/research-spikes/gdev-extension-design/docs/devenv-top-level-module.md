# devenv Top-Level Nix Module (top-level.nix)
- **Source**: https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix
- **Retrieved**: 2026-05-12
- **Note**: WebFetch returned a summary rather than full source code. Key structural details below.

## Structure Overview

**Core Functions:**
- `listEntries`: Maps directory contents into paths
- `drvOrPackageToPaths`: Extracts outputs and Python dependencies from derivations
- `profile`: Builds a unified environment using `pkgs.buildEnv`

**Configuration Options** (defined in `options`):
- Environment variables (`env`)
- Project naming (`name`)
- Shell initialization code (`enterShell`)
- Package management (`packages`, `inputsFrom`)
- Overlays for package customization
- Standard environment configuration
- Apple SDK handling for macOS
- Development runtime paths and state management

**Module Imports:**
The configuration loads 19 additional modules covering profiles, outputs, files, processes, scripts, languages, services, integrations, and more.

**Configuration Logic** (`config` section):
- Validates overlay compatibility with devenv version 1.4.2+
- Sets environment variables for `DEVENV_PROFILE`, `DEVENV_STATE`, etc.
- Establishes shell initialization with prompt customization
- Manages temporary directory handling
- Creates symbolic links for profile and runtime directories
- Partitions packages for macOS SDK compatibility
- Performs assertion validation

The file represents the foundational configuration layer enabling devenv to create reproducible development environments using Nix.
