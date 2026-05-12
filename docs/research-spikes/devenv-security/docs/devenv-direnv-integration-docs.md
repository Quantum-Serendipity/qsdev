<!-- Source: https://devenv.sh/integrations/direnv/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv direnv Integration Documentation

## Overview

devenv integrates with direnv to enable automatic environment switching when navigating between project directories. However, the documentation notes that "devenv now supports native auto activation without direnv," making direnv optional for most workflows.

## Installation

To set up direnv integration:
1. Install direnv from system packages
2. Add the direnv hook to your shell configuration

## Configuration File

Create a `.envrc` file in your project root. For version 1.4 and later, use:

```bash
#!/usr/bin/env bash

eval "$(devenv direnvrc)"

use devenv
```

The documentation specifies that ".envrc is not created by `devenv init`" and requires manual creation.

## Approval Mechanism

When direnv encounters an `.envrc` file, it blocks execution with this message:

> "direnv: error ~/myproject/.envrc is blocked. Run `direnv allow` to approve its content"

This approval process functions as a security measure requiring explicit user review before environment modifications occur.

## Passing Flags

You can extend the `use devenv` command with additional parameters:

```bash
use devenv --option services.postgres.enable:bool true
```

## Key Notes

- The `.direnv` directory should be added to `.gitignore`
- For prompt customization with direnv, Starship is recommended
- Version pinning allows auditing the integration script before updates
