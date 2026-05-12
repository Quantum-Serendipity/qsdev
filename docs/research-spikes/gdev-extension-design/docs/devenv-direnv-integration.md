# devenv.sh direnv Integration
- **Source**: https://devenv.sh/integrations/direnv/
- **Retrieved**: 2026-05-12

## Overview

DevEnv supports integration with direnv, a tool enabling automatic development environment switching between project directories. The documentation notes "devenv now supports native auto activation without direnv," and the native approach using `devenv shell` with `devenv hook` is recommended for most workflows.

## Installation

Setup requires two steps:

1. Install direnv from system packages
2. Add the direnv hook to your shell configuration

## Configuration

Create an `.envrc` file in your project directory. The current recommended approach (v1.4+) uses:

```bash
#!/usr/bin/env bash

eval "$(devenv direnvrc)"

use devenv
```

**Important note:** ".envrc is not created by `devenv init`. You need to create it manually."

## Activation and Approval

After placing the `.envrc` file, direnv displays a security warning. Running `direnv allow` approves the file and enables automatic environment loading/unloading when entering/exiting the project directory.

## Advanced Configuration

### Passing Flags

Add command-line options after `use devenv`:

```bash
use devenv --option services.postgres.enable:bool true
```

### Version Pinning

For manual control over updates, use the older method with `source_url`:

```bash
source_url "https://raw.githubusercontent.com/cachix/devenv/VERSION/devenv/direnvrc" "HASH"

use devenv
```

Compute the hash using: `direnv fetchurl "https://raw.githubusercontent.com/cachix/devenv/VERSION/devenv/direnvrc"`

## Additional Setup

Add `.direnv` to `.gitignore`:
```bash
echo ".direnv" >> .gitignore
```

The documentation notes this occurs automatically when running `devenv init`.
