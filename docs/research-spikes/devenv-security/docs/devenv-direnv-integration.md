<!-- Source: https://devenv.sh/integrations/direnv/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv direnv Integration - Complete Documentation

## Overview

The documentation indicates that "devenv now supports native auto activation without direnv." As of version 2.0, the recommended approach combines `devenv shell` with native shell reloading and `devenv hook` for automatic directory-based activation, eliminating the need for external dependencies.

However, direnv remains available for users who prefer "in place environment modification without a subshell."

## Installation Steps

### Installing direnv

1. Install direnv from system packages via direnv's installation guide
2. Add the direnv hook to your shell configuration following direnv's hook documentation

## Configuration

### Creating the .envrc File

Two configuration methods are documented:

**Version 1.4+:**
```bash
#!/usr/bin/env bash

eval "$(devenv direnvrc)"

# You can pass flags to the devenv command
# For example: use devenv --impure --option services.postgres.enable:bool true
use devenv
```

**Version 1.3 and older:**
```bash
#!/usr/bin/env bash

source_url "https://raw.githubusercontent.com/cachix/devenv/..." "sha256-..."

use devenv
```

The documentation notes that ".envrc is not created by `devenv init`. You need to create it manually."

## Activation Workflow

### Approval Process

Upon entering a project directory with an `.envrc` file, direnv displays: "direnv: error ~/myproject/.envrc is blocked. Run `direnv allow` to approve its content"

This security measure requires explicit user approval before environment modifications occur. After running `direnv allow`, the system automatically loads and unloads environments when entering/exiting the project directory.

## Advanced Configuration

### Passing Flags to devenv

Command-line options can be appended after `use devenv` in the .envrc file:
```bash
use devenv --option services.postgres.enable:bool true
```

### Shell Prompt Customization

For prompt awareness, the documentation recommends installing Starship.

### Git Ignore Configuration

The `.direnv` directory should be added to `.gitignore`. This occurs automatically during `devenv init`, or manually via:
```bash
echo ".direnv" >> .gitignore
```

## Version Management

### Updating the direnvrc Script

From v1.4 forward, the latest compatible version is automatically used with the newer configuration method. For older versions, manual updates are required.

### Pinning Specific Versions

Users can audit and control updates by pinning to specific versions:

1. Locate the script at: `https://raw.githubusercontent.com/cachix/devenv/VERSION/devenv/direnvrc`
2. Compute the sha256 hash using: `direnv fetchurl "https://raw.githubusercontent.com/cachix/devenv/VERSION/devenv/direnvrc"`
3. Update the `.envrc` file with the URL and computed hash

## Related Resources

The documentation cross-references the auto activation guide for setup instructions and recommends this as the primary approach for most workflows over the direnv integration.
