<!-- Source: https://devenv.sh/integrations/direnv/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv's direnv Integration Documentation

## Overview

devenv supports direnv for seamless development environment switching across project directories. However, the documentation notes that "devenv now supports native auto activation without direnv." As of version 2.0, the recommended approach uses `devenv shell` with native reloading and `devenv hook` for automatic directory-based activation.

## Setup Steps

**Installation Requirements:**
1. Install direnv from system packages
2. Add the direnv hook to your shell configuration

**Configuration:**
Create an `.envrc` file in your project with:
```bash
#!/usr/bin/env bash
eval "$(devenv direnvrc)"
use devenv
```

**Approval Process:**
After creating `.envrc`, run `direnv allow` to approve the file. This security measure ensures you've reviewed the content before environment modifications occur.

## How It Works

When properly configured, direnv "automatically load[s] and unload[s] the devenv environment whenever you enter and exit the project directory."

## Key Features

- **Passing flags:** You can add devenv options: `use devenv --option services.postgres.enable:bool true`
- **Prompt customization:** The documentation recommends Starship for PS1 awareness
- **Version management:** v1.4+ automatically uses the latest compatible direnvrc version

## Limitations & Notes

- devenv 2.0 has native auto activation without direnv
- direnv remains useful primarily for "in place environment modification without a subshell"
- No documented caching mechanisms or known multi-project performance issues
