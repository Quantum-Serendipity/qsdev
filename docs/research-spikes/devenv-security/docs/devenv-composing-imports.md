<!-- Source: https://devenv.sh/composing-using-imports/ -->
<!-- Retrieved: 2026-05-12 -->

# DevEnv Composing Using Imports

## Overview

The devenv documentation explains that environments can be composed in two ways: locally or through referenced inputs. This feature enables developers to structure projects with multiple components that each have their own environment configurations.

## Key Concept

The documentation presents a web application scenario with separate frontend and backend folders as the primary use case. When you navigate to the `frontend` directory, the environment activates based on the `frontend/devenv.nix` file. At the project's top level, environments combine configurations from both `backend/devenv.nix` and `frontend/devenv.nix`.

## Example Configuration

The documentation provides a `devenv.yaml` example showing:

```yaml
imports:
- ./frontend
- ./backend
- devenv/examples/supported-languages
- devenv/examples/scripts
```

This demonstrates importing local relative paths and remote examples.

## Merged Behavior

When multiple configurations are imported, they merge together. The documentation notes that running `devenv up` at the top level will "start both the frontend and backend processes," showing how process definitions combine across imported files.

## Version Note

According to the content: "Added in 1.10 - Composing `devenv.yaml` files is now supported for local files (relative and absolute paths). Remote inputs are not yet supported for `devenv.yaml` imports."

The documentation references a complete `devenv.yaml` reference guide for additional import options.
