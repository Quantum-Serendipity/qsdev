---
source: https://raw.githubusercontent.com/devcontainers/ci/main/docs/github-action.md
retrieved: 2026-03-20
type: documentation
---

# Dev Containers CI GitHub Action Documentation

## Overview

The devcontainers/ci action enables developers to "re-use Dev Containers in a GitHub workflow" for CI tasks, testing, and pre-building container images with automated feature support.

## Core Inputs

| Input | Required | Purpose |
|-------|----------|---------|
| `imageName` | Yes | Registry-qualified name for the built container image |
| `runCmd` | Yes | Command executed within the container after building |
| `imageTag` | No | One or more comma-separated tags (defaults to `latest`) |
| `subFolder` | No | Path to folder containing `.devcontainer` directory |
| `configFile` | No | Custom path to devcontainer.json file |
| `env` | No | Environment variables passed to container execution |
| `inheritEnv` | No | Inherit all runner machine environment variables |
| `push` | No | Control image push timing: `never`, `filter`, or `always` |
| `cacheFrom` | No | Additional images for build caching |
| `noCache` | No | Force rebuild with `--no-cache` flag |
| `platform` | No | Target platforms (comma-separated for multi-platform builds) |

## Outputs

The action produces a single output: `runCmdOutput` containing the complete result from executing the specified command.

## Essential Usage Patterns

**Basic workflow:**
```yaml
- uses: devcontainers/ci@v0.3
  with:
    runCmd: yarn test
```

**With registry caching:**
The action automatically pushes built images to registries when `imageName` is provided and conditions are met, enabling "future image builds in step 1 to use the image layers as a cache."

**Multi-command execution:**
Use pipe syntax to run sequential commands within a single container instance.

## Environment Variables

Variables can be specified through the action's `env` block. For `remoteEnv` sections using `localEnv` references in devcontainer.json, variables must be set at the action level—not nested under `with`—to ensure proper resolution by the devcontainer CLI.

## Advanced Features

- **Docker BuildKit support:** Pre-installed on hosted runners; use `docker/setup-buildx-action` for custom runners
- **Dev Container Features:** Automatically included during image building
- **Metadata labeling:** Container metadata automatically placed on images per specification
- **UID/GID synchronization:** Automatically matches non-root container users to host (can be disabled)

## Workflow Organization

For efficiency, organizations can separate prebuild workflows from CI workflows. Prebuild jobs push cached images to registries; subsequent CI jobs reference these images via `cacheFrom` to accelerate builds.
