# Composing devenv Environments Using Imports
- **Source**: https://devenv.sh/composing-using-imports/
- **Retrieved**: 2026-05-12

## Overview

The devenv system allows you to compose environments by combining multiple configuration files, either locally or through referenced inputs. This approach works well for projects with separate components.

## Example Structure

Consider a web application with distinct frontend and backend components in separate directories. Here's the configuration approach:

**devenv.yaml**
```yaml
inputs:
  nixpkgs:
    url: github:cachix/devenv-nixpkgs/rolling
  devenv:
    url: github:cachix/devenv
    flake: false
imports:
- ./frontend
- ./backend
- devenv/examples/supported-languages
- devenv/examples/scripts
```

## How It Works

**Directory-specific behavior:** When you navigate into the `frontend` directory, the environment activates based on the `frontend/devenv.nix` file contents.

**Combined environments:** At the project root level, the configuration merges settings from `backend/devenv.nix` and `frontend/devenv.nix`. Running `devenv up` at this level will launch processes from both components.

## Limitations and Features

As of version 1.10, local file imports (both relative and absolute paths) are supported for `devenv.yaml` composition. However, remote inputs cannot yet be imported as `devenv.yaml` files.

For comprehensive configuration options, refer to the devenv.yaml reference documentation on imports.
