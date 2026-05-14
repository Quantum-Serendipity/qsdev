---
source: https://raw.githubusercontent.com/devcontainers/cli/main/README.md
retrieved: 2026-03-20
type: documentation
---

# Dev Container CLI Documentation

## Overview

The Dev Container CLI transforms a `devcontainer.json` configuration file into a running development environment. It supports local and remote container execution across private and public cloud infrastructure.

## Available Commands

**Implemented Commands:**
- `devcontainer build` - Creates and pre-builds container images
- `devcontainer up` - Launches containers with configuration applied
- `devcontainer run-user-commands` - Executes lifecycle hooks like `postCreateCommand`
- `devcontainer read-configuration` - Displays current workspace settings
- `devcontainer exec <cmd>` - Runs commands within containers with environment variables and user properties applied
- `devcontainer features` - Assists in authoring and testing Dev Container Features
- `devcontainer templates` - Assists in authoring and testing Dev Container Templates

**Planned Commands:**
- `devcontainer stop` - Container halt functionality (not yet implemented)
- `devcontainer down` - Container cleanup operations (not yet implemented)

## Installation Methods

### Standalone Install Script
Download and execute without pre-installed Node.js (Linux/macOS x64 and arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/devcontainers/cli/main/scripts/install.sh | sh
export PATH="$HOME/.devcontainers/bin:$PATH"
```

Options: `--version`, `--prefix`, `--update`, `--uninstall`

### NPM Package
Requires Python and C/C++ build tools:

```bash
npm install -g @devcontainers/cli
```

## Practical Usage Example

Starting a Rust development environment demonstrates typical CLI workflow:

```bash
git clone https://github.com/microsoft/vscode-remote-try-rust
devcontainer up --workspace-folder <path>
devcontainer exec --workspace-folder <path> cargo run
```

This sequence builds the Docker image, initializes the container, and executes commands within it.

## Building from Source

The repository includes development container configuration. Compilation requires:

```bash
yarn
yarn compile
node devcontainer.js --help
```

## Standards & Resources

The CLI implements the Development Containers Specification, providing standardized configuration while maintaining simplified single-container deployments for CI/CD and development environments.
