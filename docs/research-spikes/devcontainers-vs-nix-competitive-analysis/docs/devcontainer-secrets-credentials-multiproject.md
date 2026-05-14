---
source: multiple web searches
retrieved: 2026-03-20
type: web-search-synthesis
sources:
  - https://github.com/devcontainers/cli/issues/772
  - https://github.com/devcontainers/spec/issues/198
  - https://github.com/devcontainers/spec/blob/main/docs/specs/secrets-support.md
  - https://deepwiki.com/devcontainers/spec/6.1-declarative-secrets
  - https://code.visualstudio.com/remote/advancedcontainers/environment-variables
  - https://dev.to/graezykev/dev-containers-part-5-multiple-projects-shared-container-configuration-2hoi
  - https://www.1password.community/discussions/developers/secrets-in-a-devcontainer-setup/85811
---

# Dev Container Secrets, Credentials, and Multi-Project Patterns

## Environment Variable Approaches

### containerEnv
- Set in devcontainer.json, applied to the entire container
- Static for container lifetime — changes require rebuild
- Visible in devcontainer.json (not suitable for actual secret values)

### remoteEnv
- Set in devcontainer.json, applied to VS Code server and subprocesses
- Can be updated without rebuild
- Supports `${localEnv:VAR}` to pull from host environment

### ${localEnv:VAR} Pattern
- Pull host environment variables into the container at runtime
- Key pattern for multi-project consulting: set different env vars on host per client, reference them in devcontainer.json

## Declarative Secrets (Spec Feature)

The `secrets` property in devcontainer.json provides metadata about what secrets a container needs, without storing values:
- Declared secrets are recommendations, not requirements
- Missing secrets should not prevent container creation
- Secrets can be dynamically updated without rebuilding
- CLI supports `--secrets-file` to inject secrets at runtime
- Currently, secrets set via --secrets-file are injected as remoteEnv (only accessible by VS Code server process)
- Issue #772 tracks support for setting secrets as containerEnv (all processes)

## Multi-Project Patterns

### Per-Project .devcontainer
- Each project/client gets its own `.devcontainer/` folder
- Own docker-compose.yml and devcontainer.json
- Different base images, tools, and configuration per client

### Shared Configuration
- Common base configuration can be extracted
- Docker Compose extends pattern for shared services

### Project Switching
- `Dev Containers: Switch Container` command in VS Code
- Each switch reloads the VS Code window
- Cannot use multiple containers in same VS Code window
- Can open multiple VS Code windows, each connected to different containers

## Credential Isolation Patterns

### Host-Side Strategy
- Set all secrets for all projects on host system
- Use `remoteEnv` with `${localEnv:VAR}` to selectively expose per-project
- Risk: all secrets accessible on host, filtering is advisory

### External Secrets Manager
- Store secrets in 1Password, Vault, etc.
- Store references in .env files per repository
- Use CLI (e.g., `op run`) to launch devcontainer with resolved secrets
- Strongest isolation: only the needed secrets enter each container

### Docker Secrets (Compose only)
- Docker Swarm secrets mechanism for Compose-based setups
- Mounted as files, not env vars
- More secure than env vars (not visible in `docker inspect`)
