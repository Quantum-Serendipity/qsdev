<!-- Source: https://devenv.sh/getting-started/ -->
<!-- Retrieved: 2026-05-12 -->

# Devenv Installation Guide

## Installation Methods

Devenv requires Nix as a prerequisite. The installation process involves three main steps:

### 1. Nix Installation

The documentation recommends the modern Nix installer across multiple platforms:

- **Linux/macOS**: `sh <(curl -L https://nixos.org/nix/install) --daemon`
- **macOS (alternative)**: Uses `https://artifacts.nixos.org/nix-installer`
- **Windows (WSL2)**: `sh <(curl -L https://nixos.org/nix/install) --no-daemon`
- **Docker**: `docker run -it nixos/nix`

### 2. Devenv Installation

Multiple approaches are supported depending on your system:

- **Newcomers**: `nix-env --install --attr devenv -f https://...` (traditional method)
- **Nix profiles**: `nix profile install nixpkgs#devenv` (requires experimental flags)
- **NixOS/nix-darwin**: System-level configuration via `configuration.nix` or `home.nix`
- **home-manager**: Integration through `home.packages`

### 3. GitHub Token Configuration (Optional but Recommended)

To prevent API rate-limiting, the guide recommends creating a personal access token and adding it to `~/.config/nix/nix.conf`:

> "access-tokens = github.com=<GITHUB_TOKEN>"

## Key Requirements

- **Nix Package Manager**: Required foundation
- **No special trust steps mentioned**: Installation uses standard curl-based scripts
- **GitHub API access**: Recommended for efficient package downloads
