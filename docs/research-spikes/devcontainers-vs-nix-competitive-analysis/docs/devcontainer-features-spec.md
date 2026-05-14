---
source: https://raw.githubusercontent.com/devcontainers/spec/main/docs/specs/devcontainer-features.md
retrieved: 2026-03-20
type: specification
---

# Dev Container Features Specification

## Core Definition

Development Container Features are "self-contained, shareable units of installation code and development container configuration" that enable rapid addition of tooling and runtimes to containers.

## File Structure

Each Feature requires a minimal directory layout:
- `devcontainer-feature.json` (metadata file)
- `install.sh` (execution script)
- Additional optional files

## devcontainer-feature.json Properties

**Required fields:**
- `id`: Unique lowercase identifier matching the directory name
- `version`: Semantic versioning (e.g., 1.0.0)
- `name`: Human-readable display name

**Optional metadata includes:**
- `description`, `documentationURL`, `licenseURL`
- `keywords` (search terms)
- `deprecated` (boolean flag)

**Configuration properties:**
- `options`: Parameter definitions with type, proposals/enum, defaults
- `containerEnv`: Environment variable overrides
- `privileged`: Docker privileged mode flag
- `init`: Tini init process inclusion
- `capAdd`, `securityOpt`: Container capability/security settings
- `entrypoint`: Container startup script
- `mounts`: Volume/bind mount specifications

**Dependency management:**
- `dependsOn`: Hard dependencies (required Features)
- `installsAfter`: Soft dependencies (ordering hints)

## Lifecycle Hooks

Features support command execution at specific build stages:
- `onCreateCommand`
- `updateContentCommand`
- `postCreateCommand`
- `postStartCommand`
- `postAttachCommand`

These mirror `devcontainer.json` behavior and execute in Feature installation order before user commands.

## Options Schema

Features define options with this structure:

```json
"options": {
  "optionId": {
    "type": "string|boolean",
    "description": "...",
    "proposals": ["val1", "val2"],
    "enum": ["strict", "list"],
    "default": "value"
  }
}
```

Options convert to uppercase environment variables accessible during `install.sh` execution.

## Installation Mechanism

**Execution context:** Features run as root during image build, enabling both system-level and user-level modifications via `su` command switching.

**Invocation:** Direct execution via `./install.sh` with proper execute bit permissions ensures correct shell interpretation.

**Environment variables passed to scripts:**
- `_REMOTE_USER` / `_CONTAINER_USER`: Container user identities
- `_REMOTE_USER_HOME` / `_CONTAINER_USER_HOME`: User home directories
- User-defined options as uppercase environment variables

## Dependency Resolution Algorithm

The installation order process follows three steps:

1. **Build dependency graph** from `dependsOn` (hard) and `installsAfter` (soft) properties
2. **Assign round priority** values for override handling
3. **Round-based sorting** to determine execution sequence while respecting all constraints

Features install only after all dependencies complete. Circular dependencies trigger fatal errors.

## Feature Referencing

Three identifier formats are supported:

| Format | Example |
|--------|---------|
| OCI registry | `ghcr.io/user/repo/go:1.18` |
| HTTPS URI | `https://github.com/user/repo/releases/feature.tgz` |
| Local path | `./myFeature` |

## Versioning & Distribution

Features follow semantic versioning. When republishing, major/minor versions are always re-released per semver; exact versions are never republished if already present.

Distribution occurs via OCI registries implementing the OCI Artifact Distribution Specification.

## Implementation Notes

- "The order of execution of Features is determined by the application, based on the `installsAfter` property used by feature authors"
- Features create layers for improved caching and rebuild efficiency
- Container parameters (`privileged`, `init`) apply if any Feature requires them
- Array parameters (`capAdd`, `securityOpt`) concatenate across Features
